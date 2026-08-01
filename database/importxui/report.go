package importxui

import "fmt"

type Report struct {
	Summary         Summary             `json:"summary"`
	Warnings        []string            `json:"warnings"`
	ByInbound       []InboundStat       `json:"by_inbound"`
	BackupPath      string              `json:"backup_path,omitempty"`
	GeneratedAdmins []GeneratedAdmin    `json:"generated_admins,omitempty"`
	Unsupported     []UnsupportedEntity `json:"unsupported,omitempty"`
}

type Summary struct {
	Inbounds   CountSummary    `json:"inbounds"`
	Endpoints  EndpointSummary `json:"endpoints"`
	Outbounds  EndpointSummary `json:"outbounds"`
	TLS        TLSSummary      `json:"tls"`
	Clients    ClientSummary   `json:"clients"`
	Historical CountSummary    `json:"historical,omitempty"`
	Routing    CountSummary    `json:"routing,omitempty"`
}

type CountSummary struct {
	Total     int `json:"total"`
	Imported  int `json:"imported"`
	Skipped   int `json:"skipped"`
	Conflicts int `json:"conflicts"`
}

type EndpointSummary struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type TLSSummary struct {
	Created int `json:"created"`
	Reused  int `json:"reused"`
}

type ClientSummary struct {
	UniqueEmails int `json:"unique_emails"`
	Merged       int `json:"merged"`
	Created      int `json:"created"`
}

type InboundStat struct {
	SrcTag  string `json:"src_tag"`
	DstTag  string `json:"dst_tag"`
	Clients int    `json:"clients"`
}

type GeneratedAdmin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UnsupportedEntity struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func (r *Report) markUnsupported(kind, source, entityType, reason string) {
	r.Unsupported = append(r.Unsupported, UnsupportedEntity{Kind: kind, Source: source, Type: entityType, Reason: reason})
	r.warn(fmt.Sprintf("%s %s: %s", kind, source, reason))
}

func (r *Report) warn(message string) {
	if message == "" {
		return
	}
	r.Warnings = append(r.Warnings, message)
}

func (r *Report) warnAll(messages []string) {
	for _, message := range messages {
		r.warn(message)
	}
}
