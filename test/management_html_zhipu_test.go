package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagementHTML_ContainsZhipuProviderEnhancement(t *testing.T) {
	htmlPath := filepath.Join("..", "management.html")

	content, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read management.html: %v", err)
	}

	text := string(content)
	required := []string{
		`<script id="zhipu-provider-enhancement">`,
		`provider-zhipu`,
		`/zhipu-api-key`,
		`/model-definitions/zhipu`,
		`Zhipu / GLM`,
	}

	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected management.html to contain %q", needle)
		}
	}
}
