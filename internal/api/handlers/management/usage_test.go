package management

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestGetUsageStatistics_ReturnsLegacyUsageAndOverviewFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stats := usage.NewRequestStatistics()
	seedUsageManagementStats(t, stats)

	h := &Handler{usageStats: stats}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage", nil)

	h.GetUsageStatistics(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var overview usage.UsageOverview
	if err := json.Unmarshal(rec.Body.Bytes(), &overview); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if overview.Usage.TotalRequests != 3 {
		t.Fatalf("legacy total_requests = %d, want 3", overview.Usage.TotalRequests)
	}
	if overview.FailedRequests != 1 {
		t.Fatalf("failed_requests = %d, want 1", overview.FailedRequests)
	}
	if got := overview.Providers["zhipu"].Summary.Requests; got != 2 {
		t.Fatalf("zhipu provider requests = %d, want 2", got)
	}
	if got := overview.Models["glm-4.5"].Requests; got != 2 {
		t.Fatalf("glm-4.5 model requests = %d, want 2", got)
	}
	if got := len(overview.RecentDays); got != 3 {
		t.Fatalf("recent_days len = %d, want 3", got)
	}
	if got := overview.Persistence.Path; got == "" {
		t.Fatalf("persistence path is empty")
	}
	if got := overview.Persistence.FileSizeHuman; got == "" {
		t.Fatalf("persistence file_size_human is empty")
	}
	if got := overview.Usage.APIs["zhipu-key"].Models["glm-4.5"].Details; len(got) != 2 {
		t.Fatalf("legacy api details len = %d, want 2", len(got))
	}
}

func TestGetUsageHistory_FiltersByRangeProviderAndModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stats := usage.NewRequestStatistics()
	seedUsageManagementStats(t, stats)

	h := &Handler{usageStats: stats}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage/history?start=2026-04-02&end=2026-04-03&provider=zhipu&model=glm-4.5", nil)

	h.GetUsageHistory(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload struct {
		Days []usage.DayUsageSnapshot `json:"days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got := len(payload.Days); got != 2 {
		t.Fatalf("days len = %d, want 2", got)
	}
	for _, day := range payload.Days {
		if day.Summary.Requests != 1 {
			t.Fatalf("day %s summary requests = %d, want 1", day.Date, day.Summary.Requests)
		}
		if got := day.Providers["zhipu"].Requests; got != 1 {
			t.Fatalf("day %s zhipu provider requests = %d, want 1", day.Date, got)
		}
		if got := day.Models["glm-4.5"].Requests; got != 1 {
			t.Fatalf("day %s glm-4.5 model requests = %d, want 1", day.Date, got)
		}
	}
}

func TestGetUsagePersistence_ReturnsPersistenceMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stats := usage.NewRequestStatistics()
	seedUsageManagementStats(t, stats)

	h := &Handler{usageStats: stats}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage/persistence", nil)

	h.GetUsagePersistence(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var meta usage.PersistenceMetadata
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !meta.Enabled {
		t.Fatalf("enabled = false, want true")
	}
	if meta.Path == "" {
		t.Fatalf("path is empty")
	}
	if meta.FileSizeBytes <= 0 {
		t.Fatalf("file_size_bytes = %d, want > 0", meta.FileSizeBytes)
	}
	if meta.RecordedDays != 3 {
		t.Fatalf("recorded_days = %d, want 3", meta.RecordedDays)
	}
}

func TestExportUsageStatistics_GetKeepsLegacyEnvelopeAndPostReturnsCanonicalSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stats := usage.NewRequestStatistics()
	seedUsageManagementStats(t, stats)

	h := &Handler{usageStats: stats}

	getRec := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRec)
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage/export", nil)

	h.ExportUsageStatistics(getCtx)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}

	var legacy struct {
		Version int                      `json:"version"`
		Usage   usage.StatisticsSnapshot `json:"usage"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &legacy); err != nil {
		t.Fatalf("GET unmarshal error: %v", err)
	}
	if legacy.Version != 1 {
		t.Fatalf("legacy version = %d, want 1", legacy.Version)
	}
	if legacy.Usage.TotalRequests != 3 {
		t.Fatalf("legacy total_requests = %d, want 3", legacy.Usage.TotalRequests)
	}

	postBody, err := json.Marshal(usageFilterPayload{
		StartDate: "2026-04-02",
		EndDate:   "2026-04-03",
		Provider:  "zhipu",
		Model:     "glm-4.5",
	})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	postRec := httptest.NewRecorder()
	postCtx, _ := gin.CreateTestContext(postRec)
	postCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/usage/export", bytes.NewReader(postBody))
	postCtx.Request.Header.Set("Content-Type", "application/json")

	h.ExportUsageStatistics(postCtx)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d; body=%s", postRec.Code, http.StatusOK, postRec.Body.String())
	}

	var snapshot usage.PersistedUsageSnapshot
	if err := json.Unmarshal(postRec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("POST unmarshal error: %v", err)
	}
	if snapshot.Version != usage.PersistedUsageVersion {
		t.Fatalf("snapshot version = %d, want %d", snapshot.Version, usage.PersistedUsageVersion)
	}
	if snapshot.Summary.Requests != 2 {
		t.Fatalf("snapshot summary requests = %d, want 2", snapshot.Summary.Requests)
	}
	if got := len(snapshot.Days); got != 2 {
		t.Fatalf("snapshot days len = %d, want 2", got)
	}
	if snapshot.Meta.Path == "" {
		t.Fatalf("snapshot meta path is empty")
	}
	if snapshot.Meta.RecordedDays != 2 {
		t.Fatalf("snapshot recorded_days = %d, want 2", snapshot.Meta.RecordedDays)
	}
}

func TestImportUsageStatistics_OverwriteLegacyPayloadReplacesExistingUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	existing := usage.NewRequestStatistics()
	existing.Record(context.Background(), coreusage.Record{
		Provider:    "gemini",
		APIKey:      "gemini-key",
		Model:       "gemini-2.5-flash",
		RequestedAt: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		Detail: coreusage.Detail{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	})

	imported := usage.NewRequestStatistics()
	imported.Record(context.Background(), coreusage.Record{
		Provider:    "zhipu",
		APIKey:      "zhipu-key",
		Model:       "glm-4.5",
		RequestedAt: time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC),
		Detail: coreusage.Detail{
			InputTokens:  5,
			OutputTokens: 7,
			TotalTokens:  12,
		},
	})

	legacyPayload, err := json.Marshal(struct {
		Version int                      `json:"version"`
		Usage   usage.StatisticsSnapshot `json:"usage"`
	}{
		Version: 1,
		Usage:   imported.Snapshot(),
	})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	h := &Handler{usageStats: existing}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v0/management/usage/import?mode=overwrite", bytes.NewReader(legacyPayload))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ImportUsageStatistics(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result usage.ImportResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.Mode != usage.MergeModeOverwrite {
		t.Fatalf("mode = %q, want %q", result.Mode, usage.MergeModeOverwrite)
	}
	if result.TotalRequests != 1 {
		t.Fatalf("total_requests = %d, want 1", result.TotalRequests)
	}

	snapshot := existing.PersistentSnapshot()
	if got := len(snapshot.Days); got != 1 {
		t.Fatalf("days len = %d, want 1", got)
	}
	if _, ok := snapshot.Days["2026-04-04"]; !ok {
		t.Fatalf("snapshot missing imported day 2026-04-04")
	}
	if _, ok := snapshot.Providers["gemini"]; ok {
		t.Fatalf("gemini provider should have been replaced")
	}
}

func TestPruneUsageStatistics_RangeBodyRemovesSelectedDays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stats := usage.NewRequestStatistics()
	seedUsageManagementStats(t, stats)

	h := &Handler{usageStats: stats}

	body, err := json.Marshal(usagePrunePayload{
		StartDate: "2026-04-02",
		EndDate:   "2026-04-03",
	})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v0/management/usage/prune", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.PruneUsageStatistics(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result usage.PruneResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.RemovedDays != 2 {
		t.Fatalf("removed_days = %d, want 2", result.RemovedDays)
	}
	if result.TotalRequests != 1 {
		t.Fatalf("total_requests = %d, want 1", result.TotalRequests)
	}

	snapshot := stats.PersistentSnapshot()
	if got := len(snapshot.Days); got != 1 {
		t.Fatalf("days len = %d, want 1", got)
	}
	if _, ok := snapshot.Days["2026-04-01"]; !ok {
		t.Fatalf("remaining day 2026-04-01 missing")
	}
}

func seedUsageManagementStats(t *testing.T, stats *usage.RequestStatistics) {
	t.Helper()

	stats.Record(context.Background(), coreusage.Record{
		Provider:    "gemini",
		APIKey:      "gemini-key",
		Model:       "gemini-2.5-flash",
		RequestedAt: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		Detail: coreusage.Detail{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	})
	stats.Record(context.Background(), coreusage.Record{
		Provider:    "zhipu",
		APIKey:      "zhipu-key",
		Model:       "glm-4.5",
		RequestedAt: time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC),
		Failed:      true,
		Detail: coreusage.Detail{
			InputTokens:  4,
			OutputTokens: 6,
			TotalTokens:  10,
		},
	})
	stats.Record(context.Background(), coreusage.Record{
		Provider:    "zhipu",
		APIKey:      "zhipu-key",
		Model:       "glm-4.5",
		RequestedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		Detail: coreusage.Detail{
			InputTokens:  5,
			OutputTokens: 7,
			TotalTokens:  12,
		},
	})

	stats.SetPersistenceStatus(usage.PersistenceMetadata{
		Enabled:       true,
		Path:          filepath.Join(t.TempDir(), "usage.json"),
		FileSizeBytes: 2048,
		LastFlushAt:   time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC),
	})
}
