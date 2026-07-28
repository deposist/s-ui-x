// Package subpage implements the public-facing "cabinet" landing page that
// turns a raw subscription endpoint into a personalised dashboard for the
// end-user. It exposes deep-links into sing-box / Hiddify / Clash / mihomo /
// v2rayN clients and renders traffic / expiry / online status pulled from the
// underlying client record.
//
// The package is deliberately isolated from sub/ to keep upstream
// (alireza0/s-ui) merge surface area minimal: only a single Mount() call
// inside sub/sub.go touches upstream code, and the rest lives in this folder
// which upstream does not know about.
package subpage

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Public surface used by sub/sub.go to mount this package onto the existing
// sub-server engine. All wiring is here; upstream only sees this single call.
func Mount(engine *gin.Engine, subPath string) error {
	enabled, err := (&service.SettingService{}).GetSubPageEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	cabinetPath, err := (&service.SettingService{}).GetSubPagePath()
	if err != nil {
		return err
	}
	if cabinetPath == "" {
		return nil
	}
	if cabinetPath == subPath {
		// Defensive: refuse to clobber the actual subscription endpoints.
		return errors.New("subpage: cabinet path conflicts with sub path")
	}
	group := engine.Group(cabinetPath)
	group.Use(rateLimitFromSub())
	group.GET("/:subid", renderCabinet)
	group.HEAD("/:subid", renderCabinet)
	return nil
}

// rateLimitFromSub wires the same per-IP rate limiter used by /sub endpoints
// so the landing page cannot be used as a side channel for /sub enumeration.
// Implemented in page_middleware.go to keep this file small.
func rateLimitFromSub() gin.HandlerFunc {
	return subRateLimit
}

// pageData is the struct rendered into the HTML template. Fields are kept
// flat and pre-escaped at construction time (url.URL.String() etc.) so the
// template engine only deals with safe strings.
type pageData struct {
	Title            string
	ClientName       string
	Enabled          bool
	VolumeText       string
	UsedTrafficText  string
	RemainingTraffic string
	ExpiryText       string
	OnlineStatus     string
	LastOnlineText   string
	SupportURL       string
	Announcement     string
	DeepLinks        []deepLink
	RawSubURL        string
	RawJSONURL       string
	RawClashURL      string
	WarnInsecure     bool
	ProfileTitle     string
}

type deepLink struct {
	AppName  string
	URL      template.URL // rendered through template.URL so go's html/template
	OSHint   string       // keeps the alt attribute honest
	Note     string       // short subtitle under the button
}

var pageTmpl = template.Must(template.New("cabinet").Funcs(template.FuncMap{
	"formatTraffic":  formatTraffic,
	"formatLastSeen": formatLastSeen,
}).Parse(pageHTML))

func renderCabinet(c *gin.Context) {
	subID := strings.TrimSpace(c.Param("subid"))
	if subID == "" || len(subID) > 128 {
		c.String(http.StatusNotFound, "Not Found")
		return
	}
	client, err := lookupClient(subID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusNotFound, "Not Found")
			return
		}
		logger.Warning("subpage: lookup failed:", err)
		c.String(http.StatusInternalServerError, "Error")
		return
	}

	// Build the public subscription URLs the user actually owns. We can't
	// trust c.Request.Host verbatim behind reverse proxies; the upstream
	// sub-server relies on the configured subDomain or the raw Host header.
	host := resolvePublicHost(c)
	base := buildBaseURL(c, host)

	data, err := buildPageData(client, base)
	if err != nil {
		logger.Warning("subpage: build data failed:", err)
		c.String(http.StatusInternalServerError, "Error")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	// Tell legacy SIP clients this is not a subscription payload even if the
	// path is somehow misconfigured to overlap.
	c.Header("X-Content-Type-Options", "nosniff")
	if err := pageTmpl.Execute(c.Writer, data); err != nil {
		logger.Warning("subpage: render failed:", err)
	}
}

// lookupClient resolves a subId either by SubSecret (UUID, default mode) or
// by Name (legacy fallback when subSecretRequired=false). It duplicates the
// logic in sub/subService.go deliberately to avoid widening the upstream
// sub/ package's public API.
func lookupClient(subID string) (*model.Client, error) {
	db := database.GetDB()
	client := &model.Client{}
	err := db.Model(model.Client{}).Where("enable = ? and sub_secret = ?", true, subID).First(client).Error
	if err == nil {
		return client, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	required, _ := (&service.SettingService{}).GetSubSecretRequired()
	if required {
		return nil, gorm.ErrRecordNotFound
	}
	err = db.Model(model.Client{}).Where("enable = ? and name = ?", true, subID).First(client).Error
	if err != nil {
		return nil, err
	}
	return client, nil
}

func buildPageData(client *model.Client, base *url.URL) (pageData, error) {
	cfg := cachedDisplaySettings()

	now := time.Now().Unix()
	used := client.Up + client.Down
	remaining := client.Volume - used
	if remaining < 0 {
		remaining = 0
	}

	expiryText := "∞"
	if client.Expiry > 0 {
		days := (client.Expiry - now) / 86400
		if days < 0 {
			days = 0
		}
		expiryText = formatDays(days)
		if client.Expiry <= now {
			expiryText = "истёк"
		}
	}

	online := ""
	if client.LastOnline > 0 {
		online = formatLastSeen(client.LastOnline)
	} else {
		online = "никогда"
	}

	subURL := base.ResolveReference(mustURL(base.Path + "/sub/" + client.SubSecret)).String()
	jsonURL := base.ResolveReference(mustURL(base.Path + "/sub/json/" + client.SubSecret)).String()
	clashURL := base.ResolveReference(mustURL(base.Path + "/sub/clash/" + client.SubSecret)).String()

	data := pageData{
		Title:            "Личный кабинет",
		ClientName:       client.Name,
		Enabled:          client.Enable,
		VolumeText:       formatTraffic(client.Volume),
		UsedTrafficText:  formatTraffic(used),
		RemainingTraffic: formatTraffic(remaining),
		ExpiryText:       expiryText,
		OnlineStatus:     online,
		SupportURL:       cfg.supportURL,
		Announcement:     cfg.announce,
		ProfileTitle:     cfg.title,
		RawSubURL:        subURL,
		RawJSONURL:       jsonURL,
		RawClashURL:      clashURL,
		WarnInsecure:     !cfg.subSecretRequired,
		DeepLinks:        buildDeepLinks(subURL),
	}
	return data, nil
}

func buildDeepLinks(subURL string) []deepLink {
	encoded := url.QueryEscape(subURL)
	return []deepLink{
		{
			AppName: "sing-box",
			URL:     template.URL("sing-box://import-remote-profile?url=" + encoded),
			OSHint:  "iOS, Android, macOS, Windows, Linux",
			Note:    "Рекомендуется. Поддерживает rule-sets.",
		},
		{
			AppName: "Hiddify",
			URL:     template.URL("hiddify://import-sub?url=" + encoded),
			OSHint:  "iOS, Android, Windows, macOS, Linux",
			Note:    "Удобный кроссплатформенный клиент.",
		},
		{
			AppName: "Clash Verge / mihomo",
			URL:     template.URL("clash://install-config?url=" + encoded),
			OSHint:  "Windows, macOS, Linux, Android (Clash Meta)",
			Note:    "Clash Meta и форки (mihomo) используют тот же диплинк.",
		},
		{
			AppName: "v2rayN / v2rayNG / Nekoray",
			URL:     template.URL("v2rayn://install-config?url=" + encoded),
			OSHint:  "Windows, Android, iOS, macOS, Linux",
			Note:    "Получит raw base64-links без rule-sets.",
		},
	}
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		return &url.URL{Path: "/"}
	}
	return u
}

// resolvePublicHost returns the host the client should use to reach back. If
// subDomain is configured in settings we honour it; otherwise we fall back to
// the inbound Host header, which is fine for direct connections and behind
// reverse proxies that preserve Host.
func resolvePublicHost(c *gin.Context) string {
	if d, _ := (&service.SettingService{}).GetSubDomain(); d != "" {
		return d
	}
	return c.Request.Host
}

// buildBaseURL assembles a *url.URL suitable for ResolveReference(). Scheme
// honours X-Forwarded-Proto when present; otherwise inferred from the
// sub-server's TLS cert configuration.
func buildBaseURL(c *gin.Context, host string) *url.URL {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = strings.Split(proto, ",")[0]
	}
	return &url.URL{Scheme: scheme, Host: host, Path: ""}
}

// --- Display settings cache (re-uses upstream's pattern) ---

type displaySettings struct {
	title             string
	supportURL        string
	announce          string
	subSecretRequired bool
}

func cachedDisplaySettings() displaySettings {
	s := &service.SettingService{}
	title, _ := s.GetSubTitle()
	support, _ := s.GetSubSupportUrl()
	announce, _ := s.GetSubAnnounce()
	required, _ := s.GetSubSecretRequired()
	return displaySettings{
		title:             title,
		supportURL:        support,
		announce:          announce,
		subSecretRequired: required,
	}
}

// --- Formatting helpers (template funcs + Go callers) ---

func formatTraffic(b int64) string {
	if b <= 0 {
		return "0 Б"
	}
	const k = 1024
	units := []string{"Б", "КБ", "МБ", "ГБ", "ТБ"}
	v := float64(b)
	i := 0
	for v >= k && i < len(units)-1 {
		v /= k
		i++
	}
	return fmt.Sprintf("%.2f %s", v, units[i])
}

func formatDays(d int64) string {
	if d <= 0 {
		return "истёк"
	}
	word := "дней"
	switch {
	case d%10 == 1 && d%100 != 11:
		word = "день"
	case d%10 >= 2 && d%10 <= 4 && (d%100 < 10 || d%100 >= 20):
		word = "дня"
	}
	return fmt.Sprintf("%d %s", d, word)
}

func formatLastSeen(unixSec int64) string {
	if unixSec <= 0 {
		return "никогда"
	}
	d := time.Since(time.Unix(unixSec, 0))
	switch {
	case d < time.Minute:
		return "только что"
	case d < time.Hour:
		return fmt.Sprintf("%d мин назад", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d ч назад", int(d.Hours()))
	default:
		return fmt.Sprintf("%d дн назад", int(d.Hours()/24))
	}
}

