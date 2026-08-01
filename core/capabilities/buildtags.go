package capabilities

import "sort"

var compiledBuildTags = map[string]bool{}

// knownBuildTags returns the fixed official release contract. Capability-specific
// tags are read after protocols.json is initialized so the manifest remains authoritative.
func knownBuildTags() []string {
	tags := map[string]struct{}{
		"with_quic":            {},
		"with_grpc":            {},
		"with_utls":            {},
		"with_acme":            {},
		"with_gvisor":          {},
		"with_naive_outbound":  {},
		"with_musl":            {},
		"badlinkname":          {},
		"tfogo_checklinkname0": {},
		"with_tailscale":       {},
	}
	for _, inbound := range loaded.Inbounds {
		if inbound.BuildTag != "" {
			tags[inbound.BuildTag] = struct{}{}
		}
	}
	for _, capabilities := range [][]SimpleCapability{loaded.Outbounds, loaded.Endpoints, loaded.Services} {
		for _, capability := range capabilities {
			if capability.BuildTag != "" {
				tags[capability.BuildTag] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

func BuildTags() map[string]bool {
	tags := knownBuildTags()
	result := make(map[string]bool, len(tags))
	for _, tag := range tags {
		result[tag] = compiledBuildTags[tag]
	}
	return result
}

func tagCompiled(tag string) bool {
	return tag == "" || compiledBuildTags[tag]
}

// APIView is the admin-safe projection of the official manifest and this build.
type APIView struct {
	BuildTags map[string]bool `json:"buildTags"`
	Inbounds  []APIInbound    `json:"inbounds"`
	Outbounds []APISimple     `json:"outbounds"`
	Groups    []APIGroup      `json:"groups"`
	Endpoints []APISimple     `json:"endpoints"`
	Services  []APISimple     `json:"services"`
}

type APIInbound struct {
	Type           string `json:"type"`
	ClientDelivery string `json:"clientDelivery"`
	HasUsers       bool   `json:"hasUsers"`
	HasInData      bool   `json:"hasInData"`
	HasTLSTemplate bool   `json:"hasTlsTemplate"`
	MuxAvailable   bool   `json:"muxAvailable"`
	OnlyTLS        bool   `json:"onlyTls"`
	UIEditor       string `json:"uiEditor"`
	BuildTag       string `json:"buildTag,omitempty"`
	Available      bool   `json:"available"`
}

type APISimple struct {
	Type      string `json:"type"`
	BuildTag  string `json:"buildTag,omitempty"`
	Available bool   `json:"available"`
}

type APIGroup struct {
	Type         string `json:"type"`
	CoreType     string `json:"coreType,omitempty"`
	AssembledAs  string `json:"assembledAs,omitempty"`
	PanelManaged bool   `json:"panelManaged,omitempty"`
	Available    bool   `json:"available"`
}

func BuildAPIView() APIView {
	view := APIView{BuildTags: BuildTags()}
	for _, inbound := range loaded.Inbounds {
		if inbound.Alias {
			continue
		}
		view.Inbounds = append(view.Inbounds, APIInbound{
			Type:           inbound.Type,
			ClientDelivery: inbound.ClientDelivery,
			HasUsers:       inbound.HasUsers,
			HasInData:      inbound.HasInData,
			HasTLSTemplate: inbound.HasTLSTemplate,
			MuxAvailable:   inbound.MuxAvailable,
			OnlyTLS:        inbound.OnlyTLS,
			UIEditor:       inbound.UIEditor,
			BuildTag:       inbound.BuildTag,
			Available:      tagCompiled(inbound.BuildTag),
		})
	}
	view.Outbounds = buildSimpleAPIView(loaded.Outbounds)
	view.Endpoints = buildSimpleAPIView(loaded.Endpoints)
	view.Services = buildSimpleAPIView(loaded.Services)
	for _, group := range loaded.Groups {
		view.Groups = append(view.Groups, APIGroup{
			Type:         group.Type,
			CoreType:     group.CoreType,
			AssembledAs:  group.AssembledAs,
			PanelManaged: group.PanelManaged,
			Available:    true,
		})
	}
	sort.Slice(view.Inbounds, func(i, j int) bool { return view.Inbounds[i].Type < view.Inbounds[j].Type })
	sort.Slice(view.Groups, func(i, j int) bool { return view.Groups[i].Type < view.Groups[j].Type })
	return view
}

func buildSimpleAPIView(capabilities []SimpleCapability) []APISimple {
	result := make([]APISimple, 0, len(capabilities))
	for _, capability := range capabilities {
		result = append(result, APISimple{
			Type:      capability.Type,
			BuildTag:  capability.BuildTag,
			Available: tagCompiled(capability.BuildTag),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Type < result[j].Type })
	return result
}
