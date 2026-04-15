package tui

import (
	"strings"
	"testing"

	usagepkg "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

func TestDashboardRenderShowsUsagePersistenceSummary(t *testing.T) {
	m := dashboardModel{client: NewClient(1, "")}
	got := m.renderDashboard(
		map[string]any{},
		usagepkg.UsageOverview{
			Usage: usagepkg.StatisticsSnapshot{
				TotalRequests: 9,
				SuccessCount:  8,
				FailureCount:  1,
				TotalTokens:   1200,
				APIs:          map[string]usagepkg.APISnapshot{},
			},
			Persistence: usagepkg.PersistenceMetadata{
				Enabled:       true,
				Path:          "C:/data/usage.json",
				FileSizeBytes: 2048,
				FileSizeHuman: "2.0 KB",
				RecordedDays:  2,
			},
		},
		nil,
		nil,
	)

	for _, want := range []string{
		"Usage persistence",
		"2.0 KB",
		"C:/data/usage.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderDashboard() missing %q\noutput:\n%s", want, got)
		}
	}
}
