package managementasset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestPinnedManagementStaticPathResolvesExactFileAndDirectory(t *testing.T) {
	tempDir := t.TempDir()
	pinnedPath := filepath.Join(tempDir, "custom", "management-panel.html")
	if err := os.MkdirAll(filepath.Dir(pinnedPath), 0o755); err != nil {
		t.Fatalf("mkdir pinned parent: %v", err)
	}

	t.Setenv("MANAGEMENT_STATIC_PATH", pinnedPath)

	if got := FilePath("ignored.yaml"); got != filepath.Clean(pinnedPath) {
		t.Fatalf("FilePath() = %q, want %q", got, filepath.Clean(pinnedPath))
	}

	wantDir := filepath.Dir(filepath.Clean(pinnedPath))
	if got := StaticDir("ignored.yaml"); got != wantDir {
		t.Fatalf("StaticDir() = %q, want %q", got, wantDir)
	}
}

func TestEnsureLatestManagementHTMLSkipsPinnedPath(t *testing.T) {
	tempDir := t.TempDir()
	pinnedPath := filepath.Join(tempDir, "panel", "management-panel.html")
	if err := os.MkdirAll(filepath.Dir(pinnedPath), 0o755); err != nil {
		t.Fatalf("mkdir pinned parent: %v", err)
	}

	t.Setenv("MANAGEMENT_STATIC_PATH", pinnedPath)

	staticDir := filepath.Join(tempDir, "static")
	if got := EnsureLatestManagementHTML(context.Background(), staticDir, "", ""); got {
		t.Fatalf("EnsureLatestManagementHTML() = true, want false for pinned path")
	}

	if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
		t.Fatalf("static dir should not be created when pinned path is set, stat err = %v", err)
	}
	if _, err := os.Stat(pinnedPath); !os.IsNotExist(err) {
		t.Fatalf("pinned file should not be touched, stat err = %v", err)
	}
}

func TestEnsureLatestManagementHTMLSkipsWhenUpdateDisabled(t *testing.T) {
	t.Setenv("MANAGEMENT_STATIC_PATH", "")
	t.Cleanup(func() {
		SetCurrentConfig(nil)
	})

	SetCurrentConfig(&config.Config{
		RemoteManagement: config.RemoteManagement{
			DisableAutoUpdatePanel: true,
		},
	})

	staticDir := filepath.Join(t.TempDir(), "static")
	if got := EnsureLatestManagementHTML(context.Background(), staticDir, "", ""); got {
		t.Fatalf("EnsureLatestManagementHTML() = true, want false when disable-auto-update-panel is enabled")
	}

	if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
		t.Fatalf("static dir should not be created when auto-update is disabled, stat err = %v", err)
	}
}

func TestEnsureLatestManagementHTMLSkipsWhenControlPanelDisabled(t *testing.T) {
	t.Setenv("MANAGEMENT_STATIC_PATH", "")
	t.Cleanup(func() {
		SetCurrentConfig(nil)
	})

	SetCurrentConfig(&config.Config{
		RemoteManagement: config.RemoteManagement{
			DisableControlPanel: true,
		},
	})

	staticDir := filepath.Join(t.TempDir(), "static")
	if got := EnsureLatestManagementHTML(context.Background(), staticDir, "", ""); got {
		t.Fatalf("EnsureLatestManagementHTML() = true, want false when control panel is disabled")
	}

	if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
		t.Fatalf("static dir should not be created when control panel is disabled, stat err = %v", err)
	}
}
