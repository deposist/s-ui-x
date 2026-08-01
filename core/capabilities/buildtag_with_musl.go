//go:build with_musl

package capabilities

func init() { compiledBuildTags["with_musl"] = true }
