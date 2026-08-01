package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateEntityIdentityContract(t *testing.T) {
	cases := []struct {
		name string
		obj  string
		act  string
		data string
		want bool
	}{
		{"valid outbound tag", "outbounds", "new", `{"tag":"proxy-a","type":"direct"}`, false},
		{"empty outbound tag", "outbounds", "new", `{"tag":"","type":"direct"}`, true},
		{"whitespace outbound tag", "outbounds", "new", `{"tag":"   ","type":"direct"}`, true},
		{"missing outbound tag", "outbounds", "new", `{"type":"direct"}`, true},
		{"non-string outbound tag", "outbounds", "new", `{"tag":123}`, true},
		{"empty inbound tag", "inbounds", "new", `{"tag":"","type":"vless"}`, true},
		{"empty service tag", "services", "new", `{"tag":"","type":"resolved"}`, true},
		{"empty endpoint tag", "endpoints", "new", `{"tag":"","type":"wireguard"}`, true},
		{"valid TLS name", "tls", "new", `{"name":"cert-a"}`, false},
		{"empty TLS name", "tls", "new", `{"name":""}`, true},
		{"missing TLS name", "tls", "new", `{"server":{}}`, true},
		{"tag does not satisfy TLS name", "tls", "new", `{"tag":"cert-a"}`, true},
		{"clients are not identity checked", "clients", "new", `{"name":""}`, false},
		{"settings are not identity checked", "settings", "new", `{"webPort":"2095"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEntityIdentity(tc.obj, tc.act, json.RawMessage(tc.data))
			if tc.want && err == nil {
				t.Fatalf("expected %s/%s %s to fail", tc.obj, tc.act, tc.data)
			}
			if !tc.want && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateEntityIdentityErrorNamesField(t *testing.T) {
	err := validateEntityIdentity("tls", "new", json.RawMessage(`{"name":""}`))
	if err == nil || !strings.Contains(err.Error(), "tls") || !strings.Contains(err.Error(), "name") {
		t.Fatalf("error should identify tls.name, got %v", err)
	}
}

func TestDispatchSaveRejectsInvalidPanelContractsBeforePersistence(t *testing.T) {
	service := &ConfigService{}
	if _, _, _, err := service.dispatchSave(nil, "outbounds", "new", json.RawMessage(`{"tag":"","type":"direct"}`), "", ""); err == nil {
		t.Fatal("blank outbound tag was accepted")
	}
	if _, _, _, err := service.dispatchSave(nil, "config", "edit", json.RawMessage(`{"route":{"rules":[{"type":"logical","mode":"and","rules":[]}]}}`), "", ""); err == nil || !strings.Contains(err.Error(), "route.rules[0]") {
		t.Fatalf("condition-less logical rule was not rejected with its path: %v", err)
	}
}

func TestValidateConfigRuleConditionsContract(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"no rules", `{"route":{"rules":[]}}`, false},
		{"action-only rule", `{"route":{"rules":[{"action":"sniff"}]}}`, false},
		{"valid logical rule", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["example.com"]}]}]}}`, false},
		{"empty logical rule", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[]}]}}`, true},
		{"empty nested logical rule", `{"dns":{"rules":[{"type":"logical","mode":"and","rules":[{"type":"logical","mode":"or","rules":[]}]}]}}`, true},
		{"undecodable rule", `{"route":{"rules":[{"action":"not-a-real-action"}]}}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfigRuleConditions(json.RawMessage(tc.data))
			if tc.want && err == nil {
				t.Fatal("expected validation failure")
			}
			if !tc.want && err != nil {
				t.Fatalf("unexpected validation failure: %v", err)
			}
		})
	}
}
