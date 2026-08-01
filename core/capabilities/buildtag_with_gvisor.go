//go:build with_gvisor

package capabilities

func init() { compiledBuildTags["with_gvisor"] = true }
