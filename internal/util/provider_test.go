package util

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
)

func TestGetProviderNameTrimsModelNameBeforeLookup(t *testing.T) {
	registry.GetGlobalRegistry().RegisterClient("test-zhipu", "zhipu", []*registry.ModelInfo{{
		ID:      "glm-4.6",
		OwnedBy: "zhipu",
		Type:    "zhipu",
	}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient("test-zhipu")
	})

	providers := GetProviderName("  glm-4.6  ")
	if len(providers) == 0 {
		t.Fatalf("providers len = 0, want > 0")
	}
	if providers[0] != "zhipu" {
		t.Fatalf("provider = %q, want zhipu", providers[0])
	}
}
