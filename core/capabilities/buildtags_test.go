package capabilities

import "testing"

func TestBuildTagsReportsOfficialContract(t *testing.T) {
	tags := BuildTags()
	for _, officialTag := range []string{
		"with_quic", "with_grpc", "with_utls", "with_acme", "with_gvisor",
		"with_naive_outbound", "with_musl", "badlinkname", "tfogo_checklinkname0",
		"with_tailscale", "with_wireguard",
	} {
		if _, found := tags[officialTag]; !found {
			t.Errorf("BuildTags missing official tag %q", officialTag)
		}
	}
	for _, extendedTag := range []string{
		"with_sudoku", "with_trusttunnel", "with_mtproxy", "with_masque",
		"with_openvpn", "with_ccm", "with_ocm", "with_oomkiller", "with_profiler",
	} {
		if _, found := tags[extendedTag]; found {
			t.Errorf("BuildTags exposes forbidden extended tag %q", extendedTag)
		}
	}
}

func TestBuildAPIViewAvailabilityMatchesCompiledTags(t *testing.T) {
	view := BuildAPIView()
	for _, inbound := range view.Inbounds {
		if inbound.Available != tagCompiled(inbound.BuildTag) {
			t.Errorf("inbound %s availability does not match tag %q", inbound.Type, inbound.BuildTag)
		}
		if inbound.Type == "shadowsocks16" {
			t.Fatal("alias type must not be exposed as an independent inbound")
		}
	}
	for _, endpoint := range view.Endpoints {
		if endpoint.Available != tagCompiled(endpoint.BuildTag) {
			t.Errorf("endpoint %s availability does not match tag %q", endpoint.Type, endpoint.BuildTag)
		}
	}
}
