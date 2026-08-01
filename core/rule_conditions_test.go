package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func ruleIssuePaths(issues []RuleConditionIssue) []string {
	paths := make([]string, 0, len(issues))
	for _, issue := range issues {
		paths = append(paths, issue.Path)
	}
	return paths
}

func requireRuleIssuePaths(t *testing.T, issues []RuleConditionIssue, want ...string) {
	t.Helper()
	got := ruleIssuePaths(issues)
	if len(got) != len(want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("paths = %v, want %v", got, want)
		}
	}
}

func decodeTestRouteRules(t *testing.T, ruleJSON string) []option.Rule {
	t.Helper()
	var options option.Options
	document := `{"route":{"rules":[` + ruleJSON + `]}}`
	if err := options.UnmarshalJSONContext(registryContext(context.Background()), []byte(document)); err != nil {
		t.Fatalf("decode route rule: %v", err)
	}
	if options.Route == nil {
		return nil
	}
	return options.Route.Rules
}

func TestRuleConditionIssuesReportsExactNestedPaths(t *testing.T) {
	config := `{
		"route":{"rules":[
			{"domain":["keep.example"]},
			{"type":"logical","mode":"and","rules":[
				{"type":"logical","mode":"or","rules":[
					{"domain":["deep.example"]},
					{"type":"logical","mode":"and","rules":[]}
				]}
			]}
		]},
		"dns":{"rules":[
			{"type":"logical","mode":"and","rules":[{"type":"logical","mode":"and","rules":[]}]}
		]}
	}`
	issues, err := RuleConditionIssues([]byte(config))
	if err != nil {
		t.Fatalf("RuleConditionIssues: %v", err)
	}
	requireRuleIssuePaths(t, issues,
		"route.rules[1].rules[0].rules[1].rules",
		"dns.rules[0].rules[0].rules",
	)
	if issues[0].Kind != RuleKindRoute || issues[0].Code != RuleConditionCodeMissingConditions {
		t.Fatalf("unexpected route issue: %+v", issues[0])
	}
	if issues[1].Kind != RuleKindDNS || issues[1].Code != RuleConditionCodeMissingConditions {
		t.Fatalf("unexpected dns issue: %+v", issues[1])
	}
}

func TestRuleConditionIssuesUsesOfficialDecodedSemantics(t *testing.T) {
	for _, shape := range []string{`{"action":"reject"}`, `{"invert":true}`} {
		rules := decodeTestRouteRules(t, shape)
		if len(rules) != 1 || !rules[0].IsValid() {
			t.Fatalf("official decoder rejected valid thin rule %s: %+v", shape, rules)
		}
		issues, err := RuleConditionIssues([]byte(`{"route":{"rules":[{"type":"logical","mode":"and","rules":[` + shape + `]}]}}`))
		if err != nil {
			t.Fatalf("RuleConditionIssues(%s): %v", shape, err)
		}
		if len(issues) != 0 {
			t.Fatalf("thin rule %s produced issues: %+v", shape, issues)
		}
	}
}

func TestRuleConditionIssuesReportsDroppedRulesSeparately(t *testing.T) {
	issues, err := RuleConditionIssues([]byte(`{"dns":{"rules":[{}]}}`))
	if err != nil {
		t.Fatalf("RuleConditionIssues: %v", err)
	}
	requireRuleIssuePaths(t, issues, "dns.rules")
	if issues[0].Code != RuleConditionCodeDroppedRule {
		t.Fatalf("code = %q, want %q", issues[0].Code, RuleConditionCodeDroppedRule)
	}

	issues, err = RuleConditionIssues([]byte(`{"route":{"rules":[{}]}}`))
	if err != nil {
		t.Fatalf("route RuleConditionIssues: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("route empty rule unexpectedly reported: %+v", issues)
	}

	issues, err = RuleConditionIssues([]byte(`{"dns":{"rules":[{}, {"domain":["example.com"]}]}}`))
	if err != nil {
		t.Fatalf("sibling RuleConditionIssues: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("sibling empty rule unexpectedly reported: %+v", issues)
	}
}

func TestRuleConditionIssuesRejectsMalformedRules(t *testing.T) {
	cases := map[string]string{
		"broken JSON":        `{"route":{"rules":[`,
		"unknown rule type":  `{"route":{"rules":[{"type":"nonsense"}]}}`,
		"unknown action":     `{"route":{"rules":[{"action":"nonsense"}]}}`,
		"unknown field":      `{"route":{"rules":[{"not_a_field":true}]}}`,
		"rule is not object": `{"route":{"rules":["nope"]}}`,
		"rules is not array": `{"route":{"rules":{"a":1}}}`,
	}
	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := RuleConditionIssues([]byte(config)); err == nil {
				t.Fatal("expected a decode error")
			}
		})
	}
}

func TestRuleConditionIssuesIgnoresUnrelatedConfigAndEmptyConfig(t *testing.T) {
	for name, config := range map[string]string{
		"empty object": `{}`,
		"no rules":     `{"route":{}}`,
		"empty arrays": `{"route":{"rules":[]},"dns":{"rules":[]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			issues, err := RuleConditionIssues([]byte(config))
			if err != nil || len(issues) != 0 {
				t.Fatalf("issues = %v, err = %v", issues, err)
			}
		})
	}
	issues, err := RuleConditionIssues([]byte(`{"inbounds":[{"type":"totally-unknown"}],"route":{"rules":[{"domain":["a.example"]}]}}`))
	if err != nil || len(issues) != 0 {
		t.Fatalf("unrelated config changed rule result: issues=%v err=%v", issues, err)
	}
}

func TestRuleConditionIssuesRemainSeparateFromValidateConfig(t *testing.T) {
	duplicate := []byte(`{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"same"},{"type":"direct","tag":"same"}],"route":{"rules":[{"domain":["a.example"]}]}}`)
	issues, err := RuleConditionIssues(duplicate)
	if err != nil || len(issues) != 0 {
		t.Fatalf("condition helper reported construction error: issues=%v err=%v", issues, err)
	}
	if err := ValidateConfig(duplicate); err == nil {
		t.Fatal("ValidateConfig accepted duplicate outbound tags")
	}

	complete := []byte(`{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"rules":[{"type":"logical","mode":"and","rules":[{"invert":true}],"outbound":"direct"}]}}`)
	issues, err = RuleConditionIssues(complete)
	if err != nil || len(issues) != 0 {
		t.Fatalf("valid config produced condition issues=%v err=%v", issues, err)
	}
	if err := ValidateConfig(complete); err != nil {
		t.Fatalf("ValidateConfig rejected valid config: %v", err)
	}

	emptyLogical := []byte(`{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"rules":[{"type":"logical","mode":"and","rules":[],"outbound":"direct"}]}}`)
	issues, err = RuleConditionIssues(emptyLogical)
	if err != nil {
		t.Fatalf("empty logical RuleConditionIssues: %v", err)
	}
	requireRuleIssuePaths(t, issues, "route.rules[0].rules")
	if err := ValidateConfig(emptyLogical); err == nil {
		t.Fatal("ValidateConfig accepted empty logical rule")
	}
}

func TestSingleRuleConditionIssues(t *testing.T) {
	issues, err := SingleRuleConditionIssues(RuleKindRoute, []byte(`{"type":"logical","mode":"and","rules":[{"type":"logical","mode":"and","rules":[]}]}`))
	if err != nil {
		t.Fatalf("route SingleRuleConditionIssues: %v", err)
	}
	requireRuleIssuePaths(t, issues, "route.rules[0].rules[0].rules")

	issues, err = SingleRuleConditionIssues(RuleKindDNS, []byte(`{}`))
	if err != nil {
		t.Fatalf("dns SingleRuleConditionIssues: %v", err)
	}
	if len(issues) != 1 || issues[0].Code != RuleConditionCodeDroppedRule || issues[0].Path != "dns.rules" {
		t.Fatalf("unexpected dropped DNS issue: %+v", issues)
	}

	if issues, err := SingleRuleConditionIssues(RuleKindRoute, []byte(`{"domain":["a.example"]}`)); err != nil || len(issues) != 0 {
		t.Fatalf("valid single rule: issues=%v err=%v", issues, err)
	}
	for _, tc := range []struct {
		kind string
		rule []byte
	}{
		{"outbound", []byte(`{}`)},
		{RuleKindRoute, nil},
		{RuleKindRoute, []byte(`{"type":"nonsense"}`)},
	} {
		if _, err := SingleRuleConditionIssues(tc.kind, tc.rule); err == nil {
			t.Fatalf("expected error for kind=%q rule=%s", tc.kind, tc.rule)
		}
	}
}

func TestRuleConditionIssuesEnforceTreeLimits(t *testing.T) {
	rule := `{"domain":["leaf.example"]}`
	for range maxRuleTreeDepth {
		rule = `{"type":"logical","mode":"and","rules":[` + rule + `]}`
	}
	if _, err := SingleRuleConditionIssues(RuleKindRoute, []byte(rule)); err == nil || !strings.Contains(err.Error(), "maximum depth") {
		t.Fatalf("expected maximum depth error, got %v", err)
	}

	children := make([]string, maxRuleTreeNodes)
	for i := range children {
		children[i] = `{"domain":["a.example"]}`
	}
	wide := `{"type":"logical","mode":"and","rules":[` + strings.Join(children, ",") + `]}`
	if _, err := SingleRuleConditionIssues(RuleKindRoute, []byte(wide)); err == nil || !strings.Contains(err.Error(), "maximum node count") {
		t.Fatalf("expected maximum node count error, got %v", err)
	}
}

func TestRuleConditionIssueFormattingAndJSON(t *testing.T) {
	if FormatRuleConditionIssues(nil) != nil {
		t.Fatal("FormatRuleConditionIssues(nil) must be nil")
	}
	issues := []RuleConditionIssue{{Kind: RuleKindRoute, Path: "route.rules[0].rules", Code: RuleConditionCodeMissingConditions, Message: "logical rule has no conditions"}}
	formatted := FormatRuleConditionIssues(issues)
	if len(formatted) != 1 || formatted[0] != "route.rules[0].rules: logical rule has no conditions" {
		t.Fatalf("formatted = %v", formatted)
	}
	encoded, err := json.Marshal(issues[0])
	if err != nil {
		t.Fatalf("marshal issue: %v", err)
	}
	for _, key := range []string{`"kind"`, `"path"`, `"code"`, `"message"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("encoded issue missing %s: %s", key, encoded)
		}
	}
}
