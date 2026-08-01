//go:build with_tailscale

package capabilities

func init() { compiledBuildTags["with_tailscale"] = true }
