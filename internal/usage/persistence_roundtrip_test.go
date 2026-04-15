package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestRequestStatisticsSaveLoadRoundTripRestoresFlushMetadata(t *testing.T) {
	stats := newPersistenceRoundTripFixture(t)
	flushAt := time.Date(2026, 4, 3, 15, 4, 5, 0, time.UTC)
	tempPath := filepath.Join(t.TempDir(), "usage.json")

	stats.SetPersistenceStatus(PersistenceMetadata{
		Enabled:     true,
		Path:        tempPath,
		LastFlushAt: flushAt,
	})

	meta, err := stats.SaveToFile(tempPath)
	if err != nil {
		t.Fatalf("SaveToFile error: %v", err)
	}
	if meta.FileSizeBytes <= 0 {
		t.Fatalf("FileSizeBytes = %d, want > 0", meta.FileSizeBytes)
	}
	if meta.RecordedDays != 3 {
		t.Fatalf("RecordedDays = %d, want 3", meta.RecordedDays)
	}

	loaded := NewRequestStatistics()
	loadedMeta, err := loaded.LoadFromFile(tempPath)
	if err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}
	if loadedMeta.Path != tempPath {
		t.Fatalf("loaded path = %q, want %q", loadedMeta.Path, tempPath)
	}
	if !loadedMeta.Enabled {
		t.Fatalf("loaded enabled = false, want true")
	}
	if !loadedMeta.LastFlushAt.Equal(flushAt) {
		t.Fatalf("loaded last_flush_at = %s, want %s", loadedMeta.LastFlushAt, flushAt)
	}
	if loadedMeta.RecordedDays != 3 {
		t.Fatalf("loaded recorded_days = %d, want 3", loadedMeta.RecordedDays)
	}

	snapshot := loaded.PersistentSnapshot()
	if snapshot.Summary.Requests != 3 {
		t.Fatalf("summary requests = %d, want 3", snapshot.Summary.Requests)
	}
	if got := len(snapshot.Days); got != 3 {
		t.Fatalf("days len = %d, want 3", got)
	}
	if got := snapshot.Providers["zhipu"].Summary.Requests; got != 2 {
		t.Fatalf("zhipu provider requests = %d, want 2", got)
	}
}

func TestRequestStatisticsHistoryFiltersByProviderModelAndDateRange(t *testing.T) {
	stats := newPersistenceRoundTripFixture(t)

	days, err := stats.History(HistoryQuery{
		StartDate: "2026-04-02",
		EndDate:   "2026-04-03",
		Provider:  "zhipu",
		Model:     "glm-4.5",
	})
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if got := len(days); got != 2 {
		t.Fatalf("days len = %d, want 2", got)
	}
	for _, day := range days {
		if day.Summary.Requests != 1 {
			t.Fatalf("day %s summary requests = %d, want 1", day.Date, day.Summary.Requests)
		}
		if got := day.Providers["zhipu"].Requests; got != 1 {
			t.Fatalf("day %s provider requests = %d, want 1", day.Date, got)
		}
		if got := day.Models["glm-4.5"].Requests; got != 1 {
			t.Fatalf("day %s model requests = %d, want 1", day.Date, got)
		}
		if len(day.Requests) != 1 {
			t.Fatalf("day %s requests len = %d, want 1", day.Date, len(day.Requests))
		}
	}
}

func TestRequestStatisticsExportKeepsPersistenceMetadata(t *testing.T) {
	stats := newPersistenceRoundTripFixture(t)
	tempPath := filepath.Join(t.TempDir(), "usage.json")

	stats.SetPersistenceStatus(PersistenceMetadata{
		Enabled:       true,
		Path:          tempPath,
		FileSizeBytes: 8192,
		LastFlushAt:   time.Date(2026, 4, 3, 15, 4, 5, 0, time.UTC),
	})

	snapshot, err := stats.Export(HistoryQuery{
		StartDate: "2026-04-02",
		EndDate:   "2026-04-03",
		Provider:  "zhipu",
	})
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}

	if snapshot.Version != PersistedUsageVersion {
		t.Fatalf("version = %d, want %d", snapshot.Version, PersistedUsageVersion)
	}
	if snapshot.Meta.Path != tempPath {
		t.Fatalf("meta path = %q, want %q", snapshot.Meta.Path, tempPath)
	}
	if snapshot.Meta.FileSizeBytes != 8192 {
		t.Fatalf("meta file_size_bytes = %d, want 8192", snapshot.Meta.FileSizeBytes)
	}
	if snapshot.Meta.RecordedDays != 2 {
		t.Fatalf("meta recorded_days = %d, want 2", snapshot.Meta.RecordedDays)
	}
	if snapshot.Summary.Requests != 2 {
		t.Fatalf("summary requests = %d, want 2", snapshot.Summary.Requests)
	}
	if got := len(snapshot.Days); got != 2 {
		t.Fatalf("days len = %d, want 2", got)
	}
	if _, ok := snapshot.Days["2026-04-01"]; ok {
		t.Fatalf("unexpected filtered day 2026-04-01 present")
	}
}

func TestRequestStatisticsPruneRangeRebuildsAggregates(t *testing.T) {
	stats := newPersistenceRoundTripFixture(t)

	result, err := stats.Prune(PruneQuery{
		StartDate: "2026-04-02",
		EndDate:   "2026-04-03",
	})
	if err != nil {
		t.Fatalf("Prune error: %v", err)
	}
	if result.RemovedDays != 2 {
		t.Fatalf("RemovedDays = %d, want 2", result.RemovedDays)
	}
	if result.RemovedRequests != 2 {
		t.Fatalf("RemovedRequests = %d, want 2", result.RemovedRequests)
	}
	if result.TotalRequests != 1 {
		t.Fatalf("TotalRequests = %d, want 1", result.TotalRequests)
	}

	snapshot := stats.PersistentSnapshot()
	if snapshot.Summary.Requests != 1 {
		t.Fatalf("summary requests = %d, want 1", snapshot.Summary.Requests)
	}
	if got := len(snapshot.Days); got != 1 {
		t.Fatalf("days len = %d, want 1", got)
	}
	if _, ok := snapshot.Days["2026-04-01"]; !ok {
		t.Fatalf("remaining day 2026-04-01 missing")
	}
	if _, ok := snapshot.Providers["zhipu"]; ok {
		t.Fatalf("zhipu provider bucket should have been pruned")
	}
}

func TestDecodePersistedSnapshotAcceptsCanonicalPayload(t *testing.T) {
	stats := newPersistenceRoundTripFixture(t)
	snapshot := stats.PersistentSnapshot()

	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	decoded, err := DecodePersistedSnapshot(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("DecodePersistedSnapshot error: %v", err)
	}
	if decoded.Version != PersistedUsageVersion {
		t.Fatalf("decoded version = %d, want %d", decoded.Version, PersistedUsageVersion)
	}
	if decoded.Summary.Requests != snapshot.Summary.Requests {
		t.Fatalf("decoded summary requests = %d, want %d", decoded.Summary.Requests, snapshot.Summary.Requests)
	}
	if got := len(decoded.Days); got != len(snapshot.Days) {
		t.Fatalf("decoded days len = %d, want %d", got, len(snapshot.Days))
	}
}

func newPersistenceRoundTripFixture(t *testing.T) *RequestStatistics {
	t.Helper()

	stats := NewRequestStatistics()
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

	return stats
}
