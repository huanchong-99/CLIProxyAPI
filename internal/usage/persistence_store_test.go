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

func TestRequestStatisticsSaveLoadRoundTrip(t *testing.T) {
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
			InputTokens:  5,
			OutputTokens: 7,
			TotalTokens:  12,
		},
	})

	tempPath := filepath.Join(t.TempDir(), "usage-statistics.json")
	meta, err := stats.SaveToFile(tempPath)
	if err != nil {
		t.Fatalf("SaveToFile error: %v", err)
	}
	if meta.FileSizeBytes <= 0 {
		t.Fatalf("FileSizeBytes = %d, want > 0", meta.FileSizeBytes)
	}
	if meta.RecordedDays != 2 {
		t.Fatalf("RecordedDays = %d, want 2", meta.RecordedDays)
	}

	loaded := NewRequestStatistics()
	loadedMeta, err := loaded.LoadFromFile(tempPath)
	if err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}
	if loadedMeta.RecordedDays != 2 {
		t.Fatalf("loaded RecordedDays = %d, want 2", loadedMeta.RecordedDays)
	}

	snapshot := loaded.PersistentSnapshot()
	if snapshot.Summary.Requests != 2 {
		t.Fatalf("summary requests = %d, want 2", snapshot.Summary.Requests)
	}
	if snapshot.Summary.FailedRequests != 1 {
		t.Fatalf("summary failed = %d, want 1", snapshot.Summary.FailedRequests)
	}
	if len(snapshot.Days) != 2 {
		t.Fatalf("days len = %d, want 2", len(snapshot.Days))
	}
	if _, ok := snapshot.Providers["zhipu"]; !ok {
		t.Fatalf("providers missing zhipu bucket")
	}
}

func TestRequestStatisticsPruneBeforeDateRebuildsAggregates(t *testing.T) {
	stats := NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		Provider:    "gemini",
		Model:       "gemini-2.5-flash",
		RequestedAt: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		Detail:      coreusage.Detail{TotalTokens: 10},
	})
	stats.Record(context.Background(), coreusage.Record{
		Provider:    "zhipu",
		Model:       "glm-4.5",
		RequestedAt: time.Date(2026, 4, 2, 8, 0, 0, 0, time.UTC),
		Detail:      coreusage.Detail{TotalTokens: 20},
	})

	result, err := stats.Prune(PruneQuery{BeforeDate: "2026-04-02"})
	if err != nil {
		t.Fatalf("Prune error: %v", err)
	}
	if result.RemovedDays != 1 {
		t.Fatalf("RemovedDays = %d, want 1", result.RemovedDays)
	}
	if result.TotalRequests != 1 {
		t.Fatalf("TotalRequests = %d, want 1", result.TotalRequests)
	}

	snapshot := stats.PersistentSnapshot()
	if snapshot.Summary.Requests != 1 {
		t.Fatalf("summary requests = %d, want 1", snapshot.Summary.Requests)
	}
	if len(snapshot.Days) != 1 {
		t.Fatalf("days len = %d, want 1", len(snapshot.Days))
	}
	if _, ok := snapshot.Days["2026-04-02"]; !ok {
		t.Fatalf("remaining days missing 2026-04-02")
	}
}

func TestRequestStatisticsImportPersistedOverwriteReplacesExistingData(t *testing.T) {
	existing := NewRequestStatistics()
	existing.Record(context.Background(), coreusage.Record{
		Provider:    "gemini",
		Model:       "gemini-2.5-flash",
		RequestedAt: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		Detail:      coreusage.Detail{TotalTokens: 10},
	})

	imported := NewRequestStatistics()
	imported.Record(context.Background(), coreusage.Record{
		Provider:    "zhipu",
		Model:       "glm-4.5",
		RequestedAt: time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC),
		Detail:      coreusage.Detail{TotalTokens: 15},
	})

	result, err := existing.ImportPersisted(imported.PersistentSnapshot(), MergeModeOverwrite)
	if err != nil {
		t.Fatalf("ImportPersisted error: %v", err)
	}
	if result.Mode != MergeModeOverwrite {
		t.Fatalf("mode = %s, want overwrite", result.Mode)
	}
	if result.TotalRequests != 1 {
		t.Fatalf("TotalRequests = %d, want 1", result.TotalRequests)
	}

	snapshot := existing.PersistentSnapshot()
	if len(snapshot.Days) != 1 {
		t.Fatalf("days len = %d, want 1", len(snapshot.Days))
	}
	if _, ok := snapshot.Days["2026-04-03"]; !ok {
		t.Fatalf("days missing 2026-04-03")
	}
	if _, ok := snapshot.Providers["gemini"]; ok {
		t.Fatalf("providers still contains overwritten gemini bucket")
	}
}

func TestDecodePersistedSnapshotSupportsLegacyPayload(t *testing.T) {
	stats := NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		Provider:    "zhipu",
		Model:       "glm-4.5",
		RequestedAt: time.Date(2026, 4, 5, 8, 0, 0, 0, time.UTC),
		Detail:      coreusage.Detail{TotalTokens: 18},
	})

	legacyPayload, err := json.Marshal(struct {
		Version int                `json:"version"`
		Usage   StatisticsSnapshot `json:"usage"`
	}{
		Version: 1,
		Usage:   stats.Snapshot(),
	})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	snapshot, err := DecodePersistedSnapshot(bytes.NewReader(legacyPayload))
	if err != nil {
		t.Fatalf("DecodePersistedSnapshot error: %v", err)
	}
	if snapshot.Summary.Requests != 1 {
		t.Fatalf("summary requests = %d, want 1", snapshot.Summary.Requests)
	}
	if len(snapshot.Days) != 1 {
		t.Fatalf("days len = %d, want 1", len(snapshot.Days))
	}
}
