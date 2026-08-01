package core

import (
	"context"
	"encoding/json"
	"strconv"

	E "github.com/sagernet/sing/common/exceptions"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

const (
	RuleKindRoute = "route"
	RuleKindDNS   = "dns"
)

const (
	RuleConditionCodeMissingConditions = "missing-conditions"
	RuleConditionCodeInvalidRule       = "invalid-rule"
	RuleConditionCodeDroppedRule       = "dropped-rule"
)

const (
	ruleConditionMessageMissing = "logical rule has no conditions: its sub-rule list is empty"
	ruleConditionMessageInvalid = "rule has no conditions"
	ruleConditionMessageDropped = "empty rules were silently discarded while decoding"
)

const (
	maxRuleTreeDepth = 64
	maxRuleTreeNodes = 4096
)

type RuleConditionIssue struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ruleArrays struct {
	Route json.RawMessage
	DNS   json.RawMessage
}

// RuleConditionIssues delegates rule decoding to the official sing-box option
// decoder, then reports logical rules that have no usable conditions. This
// keeps panel validation aligned with the exact registry and option semantics
// of the current build rather than duplicating those rules in the panel.
func RuleConditionIssues(configJSON []byte) ([]RuleConditionIssue, error) {
	arrays, err := extractRuleArrays(configJSON)
	if err != nil {
		return nil, err
	}
	return ruleConditionIssues(arrays)
}

func SingleRuleConditionIssues(kind string, ruleJSON []byte) ([]RuleConditionIssue, error) {
	if len(ruleJSON) == 0 {
		return nil, E.New("empty rule")
	}
	wrapped, err := json.Marshal([]json.RawMessage{json.RawMessage(ruleJSON)})
	if err != nil {
		return nil, err
	}
	switch kind {
	case RuleKindRoute:
		return ruleConditionIssues(ruleArrays{Route: wrapped})
	case RuleKindDNS:
		return ruleConditionIssues(ruleArrays{DNS: wrapped})
	default:
		return nil, E.New("unknown rule kind: " + kind)
	}
}

func extractRuleArrays(configJSON []byte) (ruleArrays, error) {
	if len(configJSON) == 0 {
		return ruleArrays{}, nil
	}
	var top struct {
		Route struct {
			Rules json.RawMessage `json:"rules"`
		} `json:"route"`
		DNS struct {
			Rules json.RawMessage `json:"rules"`
		} `json:"dns"`
	}
	if err := json.Unmarshal(configJSON, &top); err != nil {
		return ruleArrays{}, E.Cause(err, "decode config rules")
	}
	return ruleArrays{Route: top.Route.Rules, DNS: top.DNS.Rules}, nil
}

func ruleConditionIssues(arrays ruleArrays) ([]RuleConditionIssue, error) {
	if len(arrays.Route) == 0 && len(arrays.DNS) == 0 {
		return nil, nil
	}
	if err := validateRawRuleTreeLimits(arrays); err != nil {
		return nil, err
	}
	document, err := buildRuleDocument(arrays)
	if err != nil {
		return nil, err
	}
	var opt option.Options
	if err := opt.UnmarshalJSONContext(registryContext(context.Background()), document); err != nil {
		return nil, E.Cause(err, "decode rules")
	}

	var issues []RuleConditionIssue
	if opt.Route != nil {
		issues = appendRouteRuleIssues(issues, "route.rules", opt.Route.Rules)
	}
	if opt.DNS != nil {
		issues = appendDNSRuleIssues(issues, "dns.rules", opt.DNS.Rules)
	}
	decodedRoute := 0
	if opt.Route != nil {
		decodedRoute = len(opt.Route.Rules)
	}
	decodedDNS := 0
	if opt.DNS != nil {
		decodedDNS = len(opt.DNS.Rules)
	}
	issues, err = appendDroppedRuleIssues(issues, RuleKindRoute, "route.rules", arrays.Route, decodedRoute)
	if err != nil {
		return nil, err
	}
	issues, err = appendDroppedRuleIssues(issues, RuleKindDNS, "dns.rules", arrays.DNS, decodedDNS)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func appendDroppedRuleIssues(issues []RuleConditionIssue, kind, path string, rawRules []byte, decodedCount int) ([]RuleConditionIssue, error) {
	if len(rawRules) == 0 {
		return issues, nil
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(rawRules, &elements); err != nil {
		return nil, E.Cause(err, "decode "+kind+" rules")
	}
	if len(elements) <= decodedCount {
		return issues, nil
	}
	return append(issues, RuleConditionIssue{
		Kind: kind,
		Path: path,
		Code: RuleConditionCodeDroppedRule,
		Message: ruleConditionMessageDropped + ": " +
			strconv.Itoa(len(elements)) + " submitted, " + strconv.Itoa(decodedCount) + " kept",
	}), nil
}

func validateRawRuleTreeLimits(arrays ruleArrays) error {
	for _, rules := range []struct {
		kind string
		raw  json.RawMessage
	}{{RuleKindRoute, arrays.Route}, {RuleKindDNS, arrays.DNS}} {
		if len(rules.raw) == 0 {
			continue
		}
		var roots []json.RawMessage
		if err := json.Unmarshal(rules.raw, &roots); err != nil {
			return E.Cause(err, "decode "+rules.kind+" rules")
		}
		nodes := 0
		for _, root := range roots {
			if err := validateRawRuleNode(root, 1, &nodes); err != nil {
				return E.Cause(err, rules.kind+" rule tree")
			}
		}
	}
	return nil
}

func validateRawRuleNode(raw json.RawMessage, depth int, nodes *int) error {
	if depth > maxRuleTreeDepth {
		return E.New("exceeds maximum depth of ", maxRuleTreeDepth)
	}
	*nodes++
	if *nodes > maxRuleTreeNodes {
		return E.New("exceeds maximum node count of ", maxRuleTreeNodes)
	}
	var object struct {
		Rules []json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	for _, child := range object.Rules {
		if err := validateRawRuleNode(child, depth+1, nodes); err != nil {
			return err
		}
	}
	return nil
}

func buildRuleDocument(arrays ruleArrays) ([]byte, error) {
	document := make(map[string]any, 2)
	if len(arrays.Route) > 0 {
		document["route"] = map[string]any{"rules": arrays.Route}
	}
	if len(arrays.DNS) > 0 {
		document["dns"] = map[string]any{"rules": arrays.DNS}
	}
	return json.Marshal(document)
}

func appendRouteRuleIssues(issues []RuleConditionIssue, prefix string, rules []option.Rule) []RuleConditionIssue {
	for i, rule := range rules {
		path := indexedPath(prefix, i)
		if rule.Type != C.RuleTypeLogical {
			if !rule.DefaultOptions.IsValid() {
				issues = append(issues, invalidRuleIssue(RuleKindRoute, path))
			}
			continue
		}
		logical := rule.LogicalOptions
		if len(logical.Rules) == 0 {
			issues = append(issues, missingConditionsIssue(RuleKindRoute, path+".rules"))
			continue
		}
		issues = appendRouteRuleIssues(issues, path+".rules", logical.Rules)
	}
	return issues
}

func appendDNSRuleIssues(issues []RuleConditionIssue, prefix string, rules []option.DNSRule) []RuleConditionIssue {
	for i, rule := range rules {
		path := indexedPath(prefix, i)
		if rule.Type != C.RuleTypeLogical {
			if !rule.DefaultOptions.IsValid() {
				issues = append(issues, invalidRuleIssue(RuleKindDNS, path))
			}
			continue
		}
		logical := rule.LogicalOptions
		if len(logical.Rules) == 0 {
			issues = append(issues, missingConditionsIssue(RuleKindDNS, path+".rules"))
			continue
		}
		issues = appendDNSRuleIssues(issues, path+".rules", logical.Rules)
	}
	return issues
}

func indexedPath(prefix string, index int) string {
	return prefix + "[" + strconv.Itoa(index) + "]"
}

func missingConditionsIssue(kind, path string) RuleConditionIssue {
	return RuleConditionIssue{Kind: kind, Path: path, Code: RuleConditionCodeMissingConditions, Message: ruleConditionMessageMissing}
}

func invalidRuleIssue(kind, path string) RuleConditionIssue {
	return RuleConditionIssue{Kind: kind, Path: path, Code: RuleConditionCodeInvalidRule, Message: ruleConditionMessageInvalid}
}

func FormatRuleConditionIssues(issues []RuleConditionIssue) []string {
	if len(issues) == 0 {
		return nil
	}
	formatted := make([]string, 0, len(issues))
	for _, issue := range issues {
		formatted = append(formatted, issue.Path+": "+issue.Message)
	}
	return formatted
}
