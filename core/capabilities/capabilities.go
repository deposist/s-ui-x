// Package capabilities reads the embedded official-core capability manifest.
// The manifest is the only source of protocol types exposed by backend and frontend.
package capabilities

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
)

//go:embed protocols.json
var manifestJSON []byte

// InboundCapability describes one supported inbound and its panel projections.
type InboundCapability struct {
	Type           string            `json:"type"`
	HasUsers       bool              `json:"hasUsers"`
	UserField      string            `json:"userField"`
	Alias          bool              `json:"alias"`
	ClientDelivery string            `json:"clientDelivery"`
	LinkScheme     string            `json:"linkScheme"`
	OutJSONBuilder string            `json:"outJsonBuilder"`
	SkipOutJSON    bool              `json:"skipOutJson"`
	HasInData      bool              `json:"hasInData"`
	HasTLSTemplate bool              `json:"hasTlsTemplate"`
	MuxAvailable   bool              `json:"muxAvailable"`
	OnlyTLS        bool              `json:"onlyTls"`
	CredentialMap  map[string]string `json:"credentialMap"`
	UIEditor       string            `json:"uiEditor"`
	BuildTag       string            `json:"buildTag"`
	Notes          string            `json:"notes"`
}

// SimpleCapability describes an outbound, endpoint, or service.
type SimpleCapability struct {
	Type     string `json:"type"`
	BuildTag string `json:"buildTag"`
	Notes    string `json:"notes"`
}

// GroupCapability describes a core-backed or panel-assembled outbound group.
type GroupCapability struct {
	Type            string `json:"type"`
	CoreType        string `json:"coreType,omitempty"`
	AssembledAs     string `json:"assembledAs,omitempty"`
	PanelManaged    bool   `json:"panelManaged,omitempty"`
	SessionRecovery bool   `json:"sessionRecovery"`
	Notes           string `json:"notes"`
}

type manifest struct {
	Version   int                 `json:"version"`
	Inbounds  []InboundCapability `json:"inbounds"`
	Outbounds []SimpleCapability  `json:"outbounds"`
	Groups    []GroupCapability   `json:"groups"`
	Endpoints []SimpleCapability  `json:"endpoints"`
	Services  []SimpleCapability  `json:"services"`
}

var loaded manifest

var fieldIdent = regexp.MustCompile(`^[a-z0-9_]+$`)

var validClientDelivery = map[string]struct{}{
	"none":   {},
	"json":   {},
	"uri":    {},
	"broken": {},
}

func init() {
	if err := json.Unmarshal(manifestJSON, &loaded); err != nil {
		panic(fmt.Sprintf("capabilities: cannot parse protocols.json: %v", err))
	}
	if err := validateManifest(loaded); err != nil {
		panic("capabilities: " + err.Error())
	}
}

func validateManifest(candidate manifest) error {
	if candidate.Version != 1 {
		return fmt.Errorf("unsupported manifest version %d", candidate.Version)
	}
	seenInbounds := make(map[string]struct{}, len(candidate.Inbounds))
	for _, inbound := range candidate.Inbounds {
		if err := validateInbound(inbound, seenInbounds); err != nil {
			return err
		}
	}
	if err := validateSimpleCategory("outbound", candidate.Outbounds); err != nil {
		return err
	}
	if err := validateSimpleCategory("endpoint", candidate.Endpoints); err != nil {
		return err
	}
	if err := validateSimpleCategory("service", candidate.Services); err != nil {
		return err
	}
	return validateGroups(candidate.Groups)
}

func validateInbound(inbound InboundCapability, seen map[string]struct{}) error {
	if inbound.Type == "" {
		return fmt.Errorf("inbound entry with empty type")
	}
	if _, duplicate := seen[inbound.Type]; duplicate {
		return fmt.Errorf("duplicate inbound type %q", inbound.Type)
	}
	seen[inbound.Type] = struct{}{}
	if _, valid := validClientDelivery[inbound.ClientDelivery]; !valid {
		return fmt.Errorf("inbound %q has invalid clientDelivery %q", inbound.Type, inbound.ClientDelivery)
	}
	if inbound.HasUsers {
		if !fieldIdent.MatchString(inbound.UserField) {
			return fmt.Errorf("inbound %q has unsafe or empty userField %q", inbound.Type, inbound.UserField)
		}
	} else if inbound.UserField != "" {
		return fmt.Errorf("inbound %q has userField but hasUsers is false", inbound.Type)
	}
	if inbound.ClientDelivery == "uri" && inbound.LinkScheme == "" {
		return fmt.Errorf("inbound %q has URI delivery but no linkScheme", inbound.Type)
	}
	if inbound.ClientDelivery == "none" && inbound.OutJSONBuilder != "" {
		return fmt.Errorf("inbound %q has no client delivery but declares outJsonBuilder %q", inbound.Type, inbound.OutJSONBuilder)
	}
	return nil
}

func validateSimpleCategory(category string, capabilities []SimpleCapability) error {
	seen := make(map[string]struct{}, len(capabilities))
	for _, capability := range capabilities {
		if capability.Type == "" {
			return fmt.Errorf("%s entry with empty type", category)
		}
		if _, duplicate := seen[capability.Type]; duplicate {
			return fmt.Errorf("duplicate %s type %q", category, capability.Type)
		}
		seen[capability.Type] = struct{}{}
	}
	return nil
}

func validateGroups(groups []GroupCapability) error {
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if group.Type == "" {
			return fmt.Errorf("group entry with empty type")
		}
		if _, duplicate := seen[group.Type]; duplicate {
			return fmt.Errorf("duplicate group type %q", group.Type)
		}
		seen[group.Type] = struct{}{}
		if group.CoreType == "" && group.AssembledAs == "" {
			return fmt.Errorf("group %q must declare coreType or assembledAs", group.Type)
		}
		if group.PanelManaged && group.AssembledAs == "" {
			return fmt.Errorf("panel-managed group %q must declare assembledAs", group.Type)
		}
	}
	return nil
}

// Inbounds returns a deep copy in manifest order.
func Inbounds() []InboundCapability {
	result := make([]InboundCapability, len(loaded.Inbounds))
	for index, inbound := range loaded.Inbounds {
		result[index] = inbound
		result[index].CredentialMap = cloneMap(inbound.CredentialMap)
	}
	return result
}

func Outbounds() []SimpleCapability { return cloneSimple(loaded.Outbounds) }
func Endpoints() []SimpleCapability { return cloneSimple(loaded.Endpoints) }
func Services() []SimpleCapability  { return cloneSimple(loaded.Services) }

func Groups() []GroupCapability {
	result := make([]GroupCapability, len(loaded.Groups))
	copy(result, loaded.Groups)
	return result
}

func cloneSimple(source []SimpleCapability) []SimpleCapability {
	result := make([]SimpleCapability, len(source))
	copy(result, source)
	return result
}
func AllowedInboundTypes() map[string]struct{} {
	result := make(map[string]struct{}, len(loaded.Inbounds))
	for _, inbound := range loaded.Inbounds {
		if !inbound.Alias {
			result[inbound.Type] = struct{}{}
		}
	}
	return result
}

func AllowedOutboundTypes() map[string]struct{} {
	result := simpleTypeSet(loaded.Outbounds)
	for _, group := range loaded.Groups {
		result[group.Type] = struct{}{}
	}
	return result
}

func AllowedEndpointTypes() map[string]struct{} { return simpleTypeSet(loaded.Endpoints) }
func AllowedServiceTypes() map[string]struct{}  { return simpleTypeSet(loaded.Services) }

func simpleTypeSet(source []SimpleCapability) map[string]struct{} {
	result := make(map[string]struct{}, len(source))
	for _, capability := range source {
		result[capability.Type] = struct{}{}
	}
	return result
}

func IsTypeAllowed(category, capabilityType string) bool {
	var allowed map[string]struct{}
	switch category {
	case "inbounds":
		allowed = AllowedInboundTypes()
	case "outbounds":
		allowed = AllowedOutboundTypes()
	case "endpoints":
		allowed = AllowedEndpointTypes()
	case "services":
		allowed = AllowedServiceTypes()
	default:
		return false
	}
	_, found := allowed[capabilityType]
	return found
}

func IsTypeAvailable(category, capabilityType string) bool {
	if !IsTypeAllowed(category, capabilityType) {
		return false
	}
	if category == "inbounds" {
		for _, inbound := range loaded.Inbounds {
			if !inbound.Alias && inbound.Type == capabilityType {
				return tagCompiled(inbound.BuildTag)
			}
		}
	}
	if category == "outbounds" {
		for _, group := range loaded.Groups {
			if group.Type == capabilityType {
				return true
			}
		}
	}
	for _, capability := range simpleCapabilities(category) {
		if capability.Type == capabilityType {
			return tagCompiled(capability.BuildTag)
		}
	}
	return false
}

func simpleCapabilities(category string) []SimpleCapability {
	switch category {
	case "outbounds":
		return loaded.Outbounds
	case "endpoints":
		return loaded.Endpoints
	case "services":
		return loaded.Services
	default:
		return nil
	}
}

// UserJSONFields maps inbound type to the validated clients.config field.
func UserJSONFields() map[string]string {
	result := make(map[string]string)
	for _, inbound := range loaded.Inbounds {
		if inbound.HasUsers {
			result[inbound.Type] = inbound.UserField
		}
	}
	return result
}

// AllowedUserJSONFields is the exact safe set interpolated into SQLite JSON paths.
func AllowedUserJSONFields() map[string]struct{} {
	result := make(map[string]struct{})
	for _, field := range UserJSONFields() {
		result[field] = struct{}{}
	}
	return result
}

// InboundTypesWithLink lists official inbound types with URI delivery.
func InboundTypesWithLink() []string {
	result := make([]string, 0)
	for _, inbound := range loaded.Inbounds {
		if !inbound.Alias && inbound.ClientDelivery == "uri" {
			result = append(result, inbound.Type)
		}
	}
	return result
}

func OutJSONBuilders() map[string]string {
	result := make(map[string]string)
	for _, inbound := range loaded.Inbounds {
		if !inbound.Alias {
			result[inbound.Type] = inbound.OutJSONBuilder
		}
	}
	return result
}

func SkipOutJSONTypes() map[string]struct{} {
	result := make(map[string]struct{})
	for _, inbound := range loaded.Inbounds {
		if inbound.SkipOutJSON {
			result[inbound.Type] = struct{}{}
		}
	}
	return result
}

func CredentialMap(inboundType string) map[string]string {
	for _, inbound := range loaded.Inbounds {
		if inbound.Type == inboundType {
			return cloneMap(inbound.CredentialMap)
		}
	}
	return nil
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
