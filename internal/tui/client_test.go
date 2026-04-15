package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGetUsageDecodesExpandedOverview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/management/usage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"usage": {
				"total_requests": 9,
				"success_count": 8,
				"failure_count": 1,
				"total_tokens": 1200,
				"apis": {
					"zhipu-key": {
						"total_requests": 9,
						"total_tokens": 1200,
						"models": {
							"glm-4.5": {
								"total_requests": 9,
								"total_tokens": 1200,
								"details": [
									{
										"latency_ms": 150,
										"tokens": {
											"input_tokens": 10,
											"output_tokens": 20,
											"reasoning_tokens": 30,
											"cached_tokens": 40,
											"total_tokens": 100
										}
									}
								]
							}
						}
					}
				},
				"requests_by_day": {"2026-04-15": 9},
				"requests_by_hour": {"08": 9},
				"tokens_by_day": {"2026-04-15": 1200},
				"tokens_by_hour": {"08": 1200}
			},
			"recent_days": [
				{
					"date": "2026-04-15",
					"summary": {
						"requests": 9,
						"total_tokens": 1200
					}
				}
			],
			"persistence": {
				"enabled": true,
				"path": "C:/data/usage.json",
				"file_size_bytes": 2048,
				"file_size_human": "2.0 KB",
				"recorded_days": 1,
				"oldest_date": "2026-04-15",
				"newest_date": "2026-04-15"
			}
		}`))
	}))
	t.Cleanup(server.Close)

	client := &Client{baseURL: server.URL, http: server.Client()}
	usage, err := client.GetUsage()
	if err != nil {
		t.Fatalf("GetUsage error: %v", err)
	}

	if usage.Usage.TotalRequests != 9 {
		t.Fatalf("usage total requests = %d, want 9", usage.Usage.TotalRequests)
	}
	if len(usage.RecentDays) != 1 {
		t.Fatalf("recent days len = %d, want 1", len(usage.RecentDays))
	}
	if usage.Persistence.FileSizeHuman != "2.0 KB" {
		t.Fatalf("file size human = %q, want 2.0 KB", usage.Persistence.FileSizeHuman)
	}
	if got := usage.Usage.APIs["zhipu-key"].Models["glm-4.5"].Details[0].Tokens.TotalTokens; got != 100 {
		t.Fatalf("detail tokens = %d, want 100", got)
	}
}
