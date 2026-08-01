package core

import (
	"testing"

	"github.com/deposist/s-ui-x/core/capabilities"
)

func TestServiceRegistryMatchesCapabilityManifest(t *testing.T) {
	registry := ServiceRegistry()
	for _, service := range capabilities.Services() {
		if _, ok := registry.CreateOptions(service.Type); !ok {
			t.Fatalf("manifest service %q is not registered", service.Type)
		}
	}
	for _, excluded := range []string{"ccm", "ocm", "oom-killer"} {
		if _, ok := registry.CreateOptions(excluded); ok {
			t.Fatalf("excluded service %q is registered", excluded)
		}
	}
}
