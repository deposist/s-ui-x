package capabilities

import (
	"sort"
	"strings"
)

// RenderMatrix renders the operator-facing protocol matrix from protocols.json.
func RenderMatrix() string {
	type row struct {
		typeName                           string
		inbound, outbound, group           bool
		endpoint, service                  bool
		tlsTemplate, users                 bool
		clientDelivery, buildTag, assembly string
		notes                              []string
	}

	rows := make(map[string]*row)
	order := make([]string, 0)
	get := func(capabilityType string) *row {
		if existing, found := rows[capabilityType]; found {
			return existing
		}
		created := &row{typeName: capabilityType}
		rows[capabilityType] = created
		order = append(order, capabilityType)
		return created
	}
	merge := func(target *row, note string) {
		if note == "" {
			return
		}
		for _, existing := range target.notes {
			if existing == note {
				return
			}
		}
		target.notes = append(target.notes, note)
	}

	for _, inbound := range loaded.Inbounds {
		if inbound.Alias {
			continue
		}
		current := get(inbound.Type)
		current.inbound = true
		current.tlsTemplate = inbound.HasTLSTemplate
		current.users = inbound.HasUsers
		current.clientDelivery = inbound.ClientDelivery
		current.buildTag = inbound.BuildTag
		merge(current, inbound.Notes)
	}
	for _, outbound := range loaded.Outbounds {
		current := get(outbound.Type)
		current.outbound = true
		if current.buildTag == "" {
			current.buildTag = outbound.BuildTag
		}
		merge(current, outbound.Notes)
	}
	for _, group := range loaded.Groups {
		current := get(group.Type)
		current.group = true
		current.assembly = group.CoreType
		if group.AssembledAs != "" {
			current.assembly = group.AssembledAs
		}
		merge(current, group.Notes)
	}
	for _, endpoint := range loaded.Endpoints {
		current := get(endpoint.Type)
		current.endpoint = true
		if current.buildTag == "" {
			current.buildTag = endpoint.BuildTag
		}
		merge(current, endpoint.Notes)
	}
	for _, service := range loaded.Services {
		current := get(service.Type)
		current.service = true
		if current.buildTag == "" {
			current.buildTag = service.BuildTag
		}
		merge(current, service.Notes)
	}

	mark := func(value bool) string {
		if value {
			return "yes"
		}
		return ""
	}
	var builder strings.Builder
	builder.WriteString("<!-- GENERATED from core/capabilities/protocols.json by RenderMatrix.\n")
	builder.WriteString("     Do not edit by hand. -->\n\n")
	builder.WriteString("# Protocol capability matrix\n\n")
	builder.WriteString("Single source of truth: `core/capabilities/protocols.json`.\n\n")
	builder.WriteString("| Type | in | out | group | endpoint | service | tls-tmpl | users | clientDelivery | buildTag | assembledAs | notes/gap |\n")
	builder.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, capabilityType := range order {
		current := rows[capabilityType]
		notes := append([]string(nil), current.notes...)
		sort.Strings(notes)
		builder.WriteString("| ")
		builder.WriteString(current.typeName)
		for _, cell := range []string{
			mark(current.inbound), mark(current.outbound), mark(current.group),
			mark(current.endpoint), mark(current.service), mark(current.tlsTemplate),
			mark(current.users), current.clientDelivery, current.buildTag,
			current.assembly, strings.Join(notes, "; "),
		} {
			builder.WriteString(" | ")
			builder.WriteString(strings.ReplaceAll(cell, "|", "\\|"))
		}
		builder.WriteString(" |\n")
	}
	return builder.String()
}
