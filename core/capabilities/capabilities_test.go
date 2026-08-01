package capabilities

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOfficialManifestParsesAndValidates(t *testing.T) {
	if err := validateManifest(loaded); err != nil {
		t.Fatalf("manifest failed validation: %v", err)
	}
	if len(Inbounds()) == 0 || len(Outbounds()) == 0 || len(Endpoints()) == 0 || len(Services()) == 0 {
		t.Fatal("official manifest must expose every core capability category")
	}
}

func TestOfficialManifestExcludesExtendedOnlyTypes(t *testing.T) {
	blocked := map[string]struct{}{
		"mieru": {}, "sudoku": {}, "trusttunnel": {}, "mtproxy": {},
		"bond": {}, "core-failover": {}, "masque": {}, "openvpn": {},
		"vpn": {}, "ccm": {}, "ocm": {}, "oom-killer": {}, "profiler": {},
	}
	for category, types := range map[string][]string{
		"inbound":  simpleInboundTypes(Inbounds()),
		"outbound": simpleTypes(Outbounds()),
		"endpoint": simpleTypes(Endpoints()),
		"service":  simpleTypes(Services()),
	} {
		for _, capabilityType := range types {
			if _, found := blocked[capabilityType]; found {
				t.Fatalf("%s manifest contains extended-only type %q", category, capabilityType)
			}
		}
	}
}

func TestValidateManifestRejectsUnsafeUserField(t *testing.T) {
	candidate := manifest{
		Version: 1,
		Inbounds: []InboundCapability{{
			Type:           "vmess",
			HasUsers:       true,
			UserField:      "vmess') FROM clients; --",
			ClientDelivery: "uri",
			LinkScheme:     "vmess",
		}},
	}
	if err := validateManifest(candidate); err == nil {
		t.Fatal("unsafe user field was accepted")
	}
}

func TestValidateManifestRejectsDuplicateCategoryType(t *testing.T) {
	candidate := manifest{
		Version:   1,
		Outbounds: []SimpleCapability{{Type: "direct"}, {Type: "direct"}},
	}
	if err := validateManifest(candidate); err == nil {
		t.Fatal("duplicate outbound type was accepted")
	}
}

func TestCapabilityAccessorsReturnIndependentCopies(t *testing.T) {
	first := Inbounds()
	if len(first) == 0 {
		t.Fatal("missing inbound capabilities")
	}
	original := Inbounds()
	first[0].Type = "mutated"
	if first[0].CredentialMap == nil {
		first[0].CredentialMap = map[string]string{}
	}
	first[0].CredentialMap["name"] = "mutated"
	if reflect.DeepEqual(first, Inbounds()) || !reflect.DeepEqual(original, Inbounds()) {
		t.Fatal("caller mutation changed embedded manifest state")
	}
}

func TestEmbeddedManifestJSONRemainsValid(t *testing.T) {
	var decoded map[string]any
	if err := json.Unmarshal(manifestJSON, &decoded); err != nil {
		t.Fatalf("embedded protocols.json is invalid JSON: %v", err)
	}
	if decoded["version"] != float64(1) {
		t.Fatalf("manifest version = %v, want 1", decoded["version"])
	}
}

func simpleInboundTypes(capabilities []InboundCapability) []string {
	types := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		types = append(types, capability.Type)
	}
	return types
}

func simpleTypes(capabilities []SimpleCapability) []string {
	types := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		types = append(types, capability.Type)
	}
	return types
}
