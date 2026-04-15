package registry

import "testing"

func TestGetStaticModelDefinitionsByChannelZhipu(t *testing.T) {
	models := GetStaticModelDefinitionsByChannel("zhipu")
	if len(models) == 0 {
		t.Fatalf("zhipu models len = 0, want > 0")
	}
	if models[0].Type != "zhipu" {
		t.Fatalf("model type = %q, want zhipu", models[0].Type)
	}
	if models[0].OwnedBy != "zhipu" {
		t.Fatalf("owned_by = %q, want zhipu", models[0].OwnedBy)
	}
}
