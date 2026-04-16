package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagementHTML_ZhipuNotInOverlay(t *testing.T) {
	htmlPath := filepath.Join("..", "management.html")

	content, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read management.html: %v", err)
	}

	text := string(content)

	// The local management.html overlay must NOT contain the Zhipu section
	// (Zhipu is now injected server-side for both local and Docker).
	forbidden := []string{
		`<h3>Zhipu</h3>`,
		`data-role="zhipu-meta"`,
		`data-role="zhipu-note"`,
		`zhipuMetaEl`,
		`zhipuNoteEl`,
		`zhipu-provider-enhancement`,
		`Zhipu / GLM`,
		`Usage Persistence + Zhipu`,
	}
	for _, needle := range forbidden {
		if strings.Contains(text, needle) {
			t.Fatalf("management.html must NOT contain %q (Zhipu is now injected server-side, not embedded in the HTML file)", needle)
		}
	}

	if !strings.Contains(text, `Usage Persistence`) {
		t.Fatal("expected management.html overlay to contain 'Usage Persistence'")
	}
}
