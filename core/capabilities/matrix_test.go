package capabilities

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateMatrix = flag.Bool("update", false, "rewrite docs/protocol-matrix.md from the manifest")

func TestProtocolMatrixDoc(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "protocol-matrix.md")
	want := RenderMatrix()
	if *updateMatrix {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatalf("write matrix: %v", err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (regenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update): %v", path, err)
	}
	normalize := func(value string) string { return strings.ReplaceAll(value, "\r\n", "\n") }
	if normalize(string(got)) != normalize(want) {
		t.Fatal("docs/protocol-matrix.md is out of date; regenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update")
	}
}

func TestRenderMatrixCoversOfficialCategories(t *testing.T) {
	matrix := RenderMatrix()
	for _, capabilityType := range []string{
		"socks", "vless", "hysteria2", "direct", "selector", "wireguard", "tailscale", "resolved", "ssm-api", "derp",
	} {
		if !strings.Contains(matrix, "| "+capabilityType+" |") {
			t.Fatalf("matrix does not include official type %q", capabilityType)
		}
	}
	for _, extendedType := range []string{"sudoku", "trusttunnel", "mtproxy", "bond", "core-failover", "vpn", "ccm", "ocm", "oom-killer", "profiler"} {
		if strings.Contains(matrix, "| "+extendedType+" |") {
			t.Fatalf("matrix contains extended-only type %q", extendedType)
		}
	}
}

func TestRenderMatrixCarriesOfficialBuildTags(t *testing.T) {
	matrix := RenderMatrix()
	for _, buildTag := range []string{"with_quic", "with_naive_outbound", "with_wireguard", "with_tailscale"} {
		if !strings.Contains(matrix, buildTag) {
			t.Fatalf("matrix does not include official build tag %q", buildTag)
		}
	}
}
