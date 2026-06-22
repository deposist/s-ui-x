package service

import (
	"encoding/json"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"
	"github.com/deposist/s-ui-x/util/ssrf"

	"gorm.io/gorm"
)

// Failover group shared constants - the single source of truth referenced by
// the assembler, validation, the failover manager and the frontend contract.
const (
	// FailoverType is the panel/DB outbound type for an auto-failover group. It
	// is transformed to a sing-box "selector" at the core-assembly seam.
	FailoverType = "failover"
	// DirectTag is the conventional tag of the seeded direct outbound used as
	// the all-down fallback.
	DirectTag = "direct"
	// DefaultProbeTarget is the out-of-the-box HTTP liveness-probe target.
	DefaultProbeTarget = "https://www.gstatic.com/generate_204"
	// DefaultInterval is the default per-group probe interval.
	DefaultInterval = 30 * time.Second
	// MinInterval is the lowest accepted probe interval.
	MinInterval = 5 * time.Second
	// DefaultHysteresis is the default consecutive-healthy-sample count required
	// before failing back to a higher-priority member.
	DefaultHysteresis = 2
)

// failoverProbe is the per-group liveness-probe config stored under the
// "failover" key of a Type:"failover" outbound's Options blob. It drives only
// the failover manager and never reaches the core (stripped at assembly).
type failoverProbe struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	ProbeTarget string `json:"probe_target,omitempty"`
	Interval    string `json:"interval,omitempty"`
	Hysteresis  int    `json:"hysteresis,omitempty"`
}

// failoverOptions is the Options blob of a Type:"failover" outbound. Outbounds
// is the ordered priority list (index 0 = primary).
type failoverOptions struct {
	Outbounds                 []string      `json:"outbounds"`
	Default                   string        `json:"default,omitempty"`
	InterruptExistConnections *bool         `json:"interrupt_exist_connections,omitempty"`
	Failover                  failoverProbe `json:"failover"`
}

func parseFailoverOptions(options json.RawMessage) (failoverOptions, error) {
	var opts failoverOptions
	if len(options) == 0 {
		return opts, common.NewError("failover group has no options")
	}
	if err := json.Unmarshal(options, &opts); err != nil {
		return opts, err
	}
	return opts, nil
}

// ProbeEnabled reports whether the manager should probe and switch this group.
func (p failoverProbe) ProbeEnabled() bool {
	return p.Enabled == nil || *p.Enabled
}

// resolvedInterval returns the configured probe interval, defaulting/clamping.
func (p failoverProbe) resolvedInterval() time.Duration {
	if p.Interval == "" {
		return DefaultInterval
	}
	d, err := time.ParseDuration(p.Interval)
	if err != nil || d < MinInterval {
		return DefaultInterval
	}
	return d
}

func (p failoverProbe) resolvedHysteresis() int {
	if p.Hysteresis < 1 {
		return DefaultHysteresis
	}
	return p.Hysteresis
}

func (p failoverProbe) resolvedTarget() string {
	if p.ProbeTarget == "" {
		return DefaultProbeTarget
	}
	return p.ProbeTarget
}

// assembleFailoverForCore turns a persisted Type:"failover" row into the clean
// sing-box "selector" JSON the core accepts: the failover metadata is stripped
// (sing-box rejects unknown option keys via badjson DisallowUnknownFields), the
// direct fallback member is appended when available, and default is pinned to
// the primary so a cold start before the manager's first tick is deterministic.
func assembleFailoverForCore(o model.Outbound, directTag string) (json.RawMessage, error) {
	opts, err := parseFailoverOptions(o.Options)
	if err != nil {
		return nil, err
	}
	if len(opts.Outbounds) == 0 {
		return nil, common.NewErrorf("failover group %q has no members", o.Tag)
	}
	members := append([]string(nil), opts.Outbounds...)
	if directTag != "" && !stringSliceContains(members, directTag) {
		members = append(members, directTag)
	}
	selector := map[string]any{
		"type":      "selector",
		"tag":       o.Tag,
		"outbounds": members,
		"default":   opts.Outbounds[0],
	}
	if opts.InterruptExistConnections != nil {
		selector["interrupt_exist_connections"] = *opts.InterruptExistConnections
	}
	return json.Marshal(selector)
}

func stringSliceContains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// DirectFallbackTag returns the tag of a direct-type outbound to use as the
// all-down fallback, preferring the conventional "direct" tag; "" when none.
func DirectFallbackTag(tx *gorm.DB) string {
	var tags []string
	if err := tx.Model(model.Outbound{}).Where("type = ?", "direct").Order("tag").Pluck("tag", &tags).Error; err != nil {
		return ""
	}
	for _, t := range tags {
		if t == DirectTag {
			return DirectTag
		}
	}
	if len(tags) > 0 {
		return tags[0]
	}
	return ""
}

// FailoverGroupConfig is the manager-facing view of a failover group: its
// ordered members plus the resolved probe settings.
type FailoverGroupConfig struct {
	Tag         string
	Members     []string
	ProbeTarget string
	Interval    time.Duration
	Hysteresis  int
	Enabled     bool
}

// LoadFailoverGroups returns every Type:"failover" group with its resolved probe
// settings. Malformed rows are skipped so one broken group never stalls the rest.
func LoadFailoverGroups(db *gorm.DB) ([]FailoverGroupConfig, error) {
	var rows []model.Outbound
	if err := db.Model(model.Outbound{}).Where("type = ?", FailoverType).Find(&rows).Error; err != nil {
		return nil, err
	}
	groups := make([]FailoverGroupConfig, 0, len(rows))
	for _, row := range rows {
		opts, err := parseFailoverOptions(row.Options)
		if err != nil || len(opts.Outbounds) == 0 {
			continue
		}
		groups = append(groups, FailoverGroupConfig{
			Tag:         row.Tag,
			Members:     opts.Outbounds,
			ProbeTarget: opts.Failover.resolvedTarget(),
			Interval:    opts.Failover.resolvedInterval(),
			Hysteresis:  opts.Failover.resolvedHysteresis(),
			Enabled:     opts.Failover.ProbeEnabled(),
		})
	}
	return groups, nil
}

// validateFailoverGroup enforces the save-time rules for a Type:"failover" row:
// non-empty members, every member exists and is a plain outbound (not a group -
// which also rules out cycles), no self-reference, no duplicates, and a valid
// probe target / interval / hysteresis.
func validateFailoverGroup(tx *gorm.DB, o model.Outbound) error {
	opts, err := parseFailoverOptions(o.Options)
	if err != nil {
		return err
	}
	if len(opts.Outbounds) == 0 {
		return common.NewError("failover group needs at least one member")
	}

	var rows []model.Outbound
	if err := tx.Model(model.Outbound{}).Select("tag", "type").Find(&rows).Error; err != nil {
		return err
	}
	typeByTag := make(map[string]string, len(rows))
	for _, row := range rows {
		typeByTag[row.Tag] = row.Type
	}

	seen := make(map[string]struct{}, len(opts.Outbounds))
	for _, member := range opts.Outbounds {
		if member == o.Tag {
			return common.NewError("a failover group cannot reference itself")
		}
		if _, dup := seen[member]; dup {
			return common.NewErrorf("member %q is listed more than once", member)
		}
		seen[member] = struct{}{}
		memberType, exists := typeByTag[member]
		if !exists {
			return common.NewErrorf("failover member %q does not exist", member)
		}
		if memberType == "selector" || memberType == "urltest" || memberType == FailoverType {
			return common.NewErrorf("member %q is a group; failover members must be plain outbounds", member)
		}
	}

	if err := validateProbeTarget(opts.Failover.resolvedTarget()); err != nil {
		return err
	}
	if opts.Failover.Interval != "" {
		d, err := time.ParseDuration(opts.Failover.Interval)
		if err != nil {
			return common.NewErrorf("invalid probe interval %q", opts.Failover.Interval)
		}
		if d < MinInterval {
			return common.NewErrorf("probe interval must be >= %s", MinInterval)
		}
	}
	if opts.Failover.Hysteresis != 0 && opts.Failover.Hysteresis < 1 {
		return common.NewError("hysteresis must be >= 1")
	}
	return nil
}

// validateProbeTarget accepts an absolute http(s) URL whose host may be a
// domain or IP. Unlike the panel's own outbound-check guard it does NOT reject
// private IPs - the probe is dialed THROUGH the member outbound, so a private
// target reachable via the tunnel is legitimate - but it still blocks
// infrastructure/cloud-metadata addresses (e.g. 169.254.169.254).
func validateProbeTarget(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return common.NewErrorf("invalid probe target: %v", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return common.NewError("probe target must be an http(s) URL")
	}
	if parsed.Hostname() == "" {
		return common.NewError("probe target must include a host")
	}
	if parsed.User != nil {
		return common.NewError("probe target must not include userinfo")
	}
	if addr, err := netip.ParseAddr(parsed.Hostname()); err == nil && ssrf.IsInfrastructureAddr(addr) {
		return common.NewError("probe target host is not allowed")
	}
	return nil
}
