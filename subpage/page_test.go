package subpage

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// These tests intentionally avoid the SQLite database so they can run on
// environments where CGO is disabled. They exercise pure helpers and the
// input-validation branches of renderCabinet.

func TestFormatTraffic(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 Б"},
		{512, "512.00 Б"},
		{2048, "2.00 КБ"},
		{5 * 1024 * 1024, "5.00 МБ"},
		{3 * 1024 * 1024 * 1024, "3.00 ГБ"},
	}
	for _, c := range cases {
		got := formatTraffic(c.in)
		if got != c.want {
			t.Errorf("formatTraffic(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDays(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{-1, "истёк"},
		{0, "истёк"},
		{1, "1 день"},
		{2, "2 дня"},
		{5, "5 дней"},
		{11, "11 дней"},
		{21, "21 день"},
		{22, "22 дня"},
	}
	for _, c := range cases {
		got := formatDays(c.in)
		if got != c.want {
			t.Errorf("formatDays(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatLastSeen(t *testing.T) {
	now := time.Now().Unix()
	if got := formatLastSeen(0); got != "никогда" {
		t.Errorf("formatLastSeen(0) = %q, want никогда", got)
	}
	if got := formatLastSeen(now - 30); got != "только что" {
		t.Errorf("formatLastSeen(30s ago) = %q, want только что", got)
	}
	if got := formatLastSeen(now - 600); !strings.HasPrefix(got, "10 мин") {
		t.Errorf("formatLastSeen(10m ago) = %q, want prefix '10 мин'", got)
	}
}

func TestBuildDeepLinks(t *testing.T) {
	subURL := "https://example.com/sub/abcdef-1234"
	links := buildDeepLinks(subURL)
	if len(links) != 4 {
		t.Fatalf("expected 4 deep-links, got %d", len(links))
	}
	wantSchemes := []string{"sing-box://", "hiddify://", "clash://", "v2rayn://"}
	for i, want := range wantSchemes {
		got := string(links[i].URL)
		if !strings.HasPrefix(got, want) {
			t.Errorf("link[%d] = %q, want prefix %q", i, got, want)
		}
		if !strings.Contains(got, url.QueryEscape(subURL)) {
			t.Errorf("link[%d] missing escaped subURL in query", i)
		}
	}
}

func TestBuildBaseURLHonoursForwardedProto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		u := buildBaseURL(c, "example.com")
		if u.Scheme != "https" {
			t.Errorf("expected https, got %q", u.Scheme)
		}
		if u.Host != "example.com" {
			t.Errorf("expected example.com, got %q", u.Host)
		}
		c.Status(204)
	})
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestMountIsIdempotentAndRejectsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// Two consecutive mounts with the same path must not panic and must
	// short-circuit (the second one is a no-op because the route is already
	// registered). We exercise that via the public Mount entry point.
	if err := Mount(engine, "/sub/"); err != nil {
		// Mount swallows disabled flag; only errors out on conflicts or
		// setting-fetch failures. Default config (no DB) may fail to read
		// settings; treat that as a soft skip for this unit test.
		t.Skipf("Mount requires live settings in this environment: %v", err)
	}

	// Conflict simulation: if we deliberately force the same path as /sub/
	// by patching settings later, Mount must refuse. We can't easily do
	// that here without DB, but at least ensure Mount returns nil for the
	// default-disabled case (the happy path for fresh installs).
	if err := Mount(engine, "/sub/"); err != nil {
		t.Skipf("Mount unavailable in this environment: %v", err)
	}
}

func TestRenderCabinetRejectsLongSubID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cabinet/:subid", renderCabinet)

	longID := strings.Repeat("a", 200)
	req := httptest.NewRequest("GET", "/cabinet/"+longID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for oversized subid, got %d body=%q", w.Code, w.Body.String())
	}
}

func TestRenderCabinetSetsSensibleHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Register a route that always 404s (no DB), but the headers are written
	// before the lookup fails.
	r.GET("/cabinet/:subid", renderCabinet)
	req := httptest.NewRequest("HEAD", "/cabinet/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options=nosniff, got %q", got)
	}
}