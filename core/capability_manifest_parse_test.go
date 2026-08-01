package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x/core/capabilities"
	sb "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
)

func TestAvailableManifestTypesParseWithOfficialOptions(t *testing.T) {
	ctx := sb.Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), DNSTransportRegistry(), ServiceRegistry())
	parse := func(t *testing.T, category, capabilityType string, document map[string]any) {
		t.Helper()
		content, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		var options option.Options
		if err := options.UnmarshalJSONContext(ctx, content); err != nil {
			t.Fatalf("official %s type %q does not parse: %v\n%s", category, capabilityType, err, content)
		}
	}

	for _, capability := range capabilities.Inbounds() {
		if capability.Alias || !capabilities.IsTypeAvailable("inbounds", capability.Type) {
			continue
		}
		t.Run("inbound/"+capability.Type, func(t *testing.T) {
			parse(t, "inbound", capability.Type, map[string]any{"inbounds": []any{map[string]any{"type": capability.Type, "tag": "test"}}})
		})
	}
	for _, capability := range capabilities.Outbounds() {
		if !capabilities.IsTypeAvailable("outbounds", capability.Type) {
			continue
		}
		t.Run("outbound/"+capability.Type, func(t *testing.T) {
			parse(t, "outbound", capability.Type, map[string]any{"outbounds": []any{map[string]any{"type": capability.Type, "tag": "test"}}})
		})
	}
	for _, capability := range capabilities.Groups() {
		coreType := capability.CoreType
		if capability.AssembledAs != "" {
			coreType = capability.AssembledAs
		}
		t.Run("group/"+capability.Type, func(t *testing.T) {
			parse(t, "group", capability.Type, map[string]any{"outbounds": []any{map[string]any{"type": coreType, "tag": "test"}}})
		})
	}
	for _, capability := range capabilities.Endpoints() {
		if !capabilities.IsTypeAvailable("endpoints", capability.Type) {
			continue
		}
		t.Run("endpoint/"+capability.Type, func(t *testing.T) {
			parse(t, "endpoint", capability.Type, map[string]any{"endpoints": []any{map[string]any{"type": capability.Type, "tag": "test"}}})
		})
	}
	for _, capability := range capabilities.Services() {
		if !capabilities.IsTypeAvailable("services", capability.Type) {
			continue
		}
		t.Run("service/"+capability.Type, func(t *testing.T) {
			parse(t, "service", capability.Type, map[string]any{"services": []any{map[string]any{"type": capability.Type, "tag": "test"}}})
		})
	}
}

func TestExtendedOnlyTypesAreRejectedByOfficialOptions(t *testing.T) {
	ctx := sb.Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), DNSTransportRegistry(), ServiceRegistry())
	fixtures := []struct {
		name     string
		document map[string]any
	}{
		{name: "inbound/mieru", document: map[string]any{"inbounds": []any{map[string]any{"type": "mieru"}}}},
		{name: "inbound/sudoku", document: map[string]any{"inbounds": []any{map[string]any{"type": "sudoku"}}}},
		{name: "inbound/trusttunnel", document: map[string]any{"inbounds": []any{map[string]any{"type": "trusttunnel"}}}},
		{name: "inbound/mtproxy", document: map[string]any{"inbounds": []any{map[string]any{"type": "mtproxy"}}}},
		{name: "outbound/core-failover", document: map[string]any{"outbounds": []any{map[string]any{"type": "core-failover"}}}},
		{name: "outbound/masque", document: map[string]any{"outbounds": []any{map[string]any{"type": "masque"}}}},
		{name: "outbound/openvpn", document: map[string]any{"outbounds": []any{map[string]any{"type": "openvpn"}}}},
		{name: "endpoint/vpn", document: map[string]any{"endpoints": []any{map[string]any{"type": "vpn"}}}},
		{name: "service/ccm", document: map[string]any{"services": []any{map[string]any{"type": "ccm"}}}},
		{name: "service/ocm", document: map[string]any{"services": []any{map[string]any{"type": "ocm"}}}},
		{name: "service/oom-killer", document: map[string]any{"services": []any{map[string]any{"type": "oom-killer"}}}},
		{name: "service/profiler", document: map[string]any{"services": []any{map[string]any{"type": "profiler"}}}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			content, err := json.Marshal(fixture.document)
			if err != nil {
				t.Fatal(err)
			}
			var options option.Options
			if err := options.UnmarshalJSONContext(ctx, content); err == nil {
				t.Fatalf("extended-only fixture parsed through official registries: %s", content)
			}
		})
	}
}
