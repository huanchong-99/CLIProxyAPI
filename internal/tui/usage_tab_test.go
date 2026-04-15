package tui

import (
	"strings"
	"testing"

	usagepkg "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

func TestUsageTabRenderContentIncludesPersistenceAndRecentDays(t *testing.T) {
	m := usageTabModel{
		usage: usagepkg.UsageOverview{
			Usage: usagepkg.StatisticsSnapshot{
				TotalRequests: 9,
				SuccessCount:  8,
				FailureCount:  1,
				TotalTokens:   1200,
				RequestsByDay: map[string]int64{
					"2026-04-14": 3,
					"2026-04-15": 6,
				},
				RequestsByHour: map[string]int64{
					"08": 4,
					"09": 5,
				},
				TokensByHour: map[string]int64{
					"08": 400,
					"09": 800,
				},
				APIs: map[string]usagepkg.APISnapshot{
					"zhipu-key": {
						TotalRequests: 9,
						TotalTokens:   1200,
						Models: map[string]usagepkg.ModelSnapshot{
							"glm-4.5": {
								TotalRequests: 9,
								TotalTokens:   1200,
								Details: []usagepkg.RequestDetail{
									{
										LatencyMs: 150,
										Tokens: usagepkg.TokenStats{
											InputTokens:     10,
											OutputTokens:    20,
											ReasoningTokens: 30,
											CachedTokens:    40,
											TotalTokens:     100,
										},
									},
								},
							},
						},
					},
				},
			},
			Persistence: usagepkg.PersistenceMetadata{
				Enabled:       true,
				Path:          "C:/data/usage.json",
				FileSizeBytes: 2048,
				FileSizeHuman: "2.0 KB",
				OldestDate:    "2026-04-14",
				NewestDate:    "2026-04-15",
				RecordedDays:  2,
			},
			RecentDays: []usagepkg.DayUsageSnapshot{
				{
					Date: "2026-04-14",
					Summary: usagepkg.UsageTotals{
						Requests:    3,
						TotalTokens: 400,
					},
				},
				{
					Date: "2026-04-15",
					Summary: usagepkg.UsageTotals{
						Requests:    6,
						TotalTokens: 800,
					},
				},
			},
		},
	}

	got := m.renderContent()
	for _, want := range []string{
		"Usage Persistence",
		"2.0 KB",
		"2026-04-14",
		"2026-04-15",
		"Recent Days",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderContent() missing %q\noutput:\n%s", want, got)
		}
	}
}

func TestRenderLatencyBreakdown(t *testing.T) {
	tests := []struct {
		name         string
		modelStats   usagepkg.ModelSnapshot
		wantEmpty    bool
		wantContains string
	}{
		{
			name:       "no details",
			modelStats: usagepkg.ModelSnapshot{},
			wantEmpty:  true,
		},
		{
			name:       "empty details",
			modelStats: usagepkg.ModelSnapshot{Details: nil},
			wantEmpty:  true,
		},
		{
			name: "details with zero latency",
			modelStats: usagepkg.ModelSnapshot{
				Details: []usagepkg.RequestDetail{{LatencyMs: 0}},
			},
			wantEmpty: true,
		},
		{
			name: "single request with latency",
			modelStats: usagepkg.ModelSnapshot{
				Details: []usagepkg.RequestDetail{{LatencyMs: 1500}},
			},
			wantEmpty:    false,
			wantContains: "avg 1500ms  min 1500ms  max 1500ms",
		},
		{
			name: "multiple requests with varying latency",
			modelStats: usagepkg.ModelSnapshot{
				Details: []usagepkg.RequestDetail{
					{LatencyMs: 100},
					{LatencyMs: 200},
					{LatencyMs: 300},
				},
			},
			wantEmpty:    false,
			wantContains: "avg 200ms  min 100ms  max 300ms",
		},
		{
			name: "mixed valid and invalid latency values",
			modelStats: usagepkg.ModelSnapshot{
				Details: []usagepkg.RequestDetail{
					{LatencyMs: 500},
					{LatencyMs: 0},
					{LatencyMs: 1500},
				},
			},
			wantEmpty:    false,
			wantContains: "avg 1000ms  min 500ms  max 1500ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderModelLatencyBreakdown(tt.modelStats.Details)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("renderLatencyBreakdown() = %q, want empty string", result)
				}
				return
			}

			if result == "" {
				t.Errorf("renderLatencyBreakdown() = empty, want non-empty string")
				return
			}

			if tt.wantContains != "" && !strings.Contains(result, tt.wantContains) {
				t.Errorf("renderLatencyBreakdown() = %q, want to contain %q", result, tt.wantContains)
			}
		})
	}
}

func TestUsageTimeTranslations(t *testing.T) {
	prevLocale := CurrentLocale()
	t.Cleanup(func() {
		SetLocale(prevLocale)
	})

	tests := []struct {
		locale string
		want   string
	}{
		{locale: "en", want: "Time"},
		{locale: "zh", want: "时间"},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			SetLocale(tt.locale)
			if got := T("usage_time"); got != tt.want {
				t.Fatalf("T(usage_time) = %q, want %q", got, tt.want)
			}
		})
	}
}
