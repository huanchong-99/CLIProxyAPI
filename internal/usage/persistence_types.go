package usage

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// PersistedUsageVersion identifies the canonical persisted usage format.
	PersistedUsageVersion = 2
	usageDayLayout        = "2006-01-02"
)

// UsageTotals captures aggregate request and token counters.
type UsageTotals struct {
	Requests        int64 `json:"requests"`
	SuccessRequests int64 `json:"success_requests"`
	FailedRequests  int64 `json:"failed_requests"`
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

func (t *UsageTotals) addDetail(tokens TokenStats, failed bool) {
	if t == nil {
		return
	}
	tokens = normaliseTokenStats(tokens)
	t.Requests++
	if failed {
		t.FailedRequests++
	} else {
		t.SuccessRequests++
	}
	t.InputTokens += tokens.InputTokens
	t.OutputTokens += tokens.OutputTokens
	t.ReasoningTokens += tokens.ReasoningTokens
	t.CachedTokens += tokens.CachedTokens
	t.TotalTokens += tokens.TotalTokens
}

// ProviderUsageSnapshot summarises usage for a provider and its models.
type ProviderUsageSnapshot struct {
	Summary UsageTotals            `json:"summary"`
	Models  map[string]UsageTotals `json:"models,omitempty"`
}

// PersistedRequestRecord is the canonical per-request record stored in usage history.
type PersistedRequestRecord struct {
	Timestamp time.Time  `json:"timestamp"`
	Provider  string     `json:"provider"`
	APIKey    string     `json:"api_key,omitempty"`
	APIName   string     `json:"api_name,omitempty"`
	AuthID    string     `json:"auth_id,omitempty"`
	AuthIndex string     `json:"auth_index,omitempty"`
	Model     string     `json:"model"`
	Source    string     `json:"source,omitempty"`
	LatencyMs int64      `json:"latency_ms,omitempty"`
	Failed    bool       `json:"failed"`
	Tokens    TokenStats `json:"tokens"`
}

func (r PersistedRequestRecord) dedupKey() string {
	tokens := normaliseTokenStats(r.Tokens)
	return fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|%s|%t|%d|%d|%d|%d|%d",
		r.Timestamp.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(r.Provider),
		strings.TrimSpace(r.APIName),
		strings.TrimSpace(r.APIKey),
		strings.TrimSpace(r.AuthID),
		strings.TrimSpace(r.AuthIndex),
		strings.TrimSpace(r.Model),
		r.Failed,
		tokens.InputTokens,
		tokens.OutputTokens,
		tokens.ReasoningTokens,
		tokens.CachedTokens,
		tokens.TotalTokens,
	)
}

// DayUsageSnapshot stores canonical history for one day bucket.
type DayUsageSnapshot struct {
	Date      string                   `json:"date"`
	Summary   UsageTotals              `json:"summary"`
	Providers map[string]UsageTotals   `json:"providers,omitempty"`
	Models    map[string]UsageTotals   `json:"models,omitempty"`
	Requests  []PersistedRequestRecord `json:"requests,omitempty"`
}

// PersistenceMetadata reports backend usage persistence state.
type PersistenceMetadata struct {
	Enabled       bool      `json:"enabled"`
	Path          string    `json:"path,omitempty"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	FileSizeHuman string    `json:"file_size_human,omitempty"`
	LastFlushAt   time.Time `json:"last_flush_at,omitempty"`
	OldestDate    string    `json:"oldest_date,omitempty"`
	NewestDate    string    `json:"newest_date,omitempty"`
	RecordedDays  int       `json:"recorded_days"`
}

// PersistedUsageSnapshot is the canonical persisted usage document.
type PersistedUsageSnapshot struct {
	Version   int                              `json:"version"`
	UpdatedAt time.Time                        `json:"updated_at"`
	Summary   UsageTotals                      `json:"summary"`
	Providers map[string]ProviderUsageSnapshot `json:"providers,omitempty"`
	Models    map[string]UsageTotals           `json:"models,omitempty"`
	Days      map[string]DayUsageSnapshot      `json:"days,omitempty"`
	Meta      PersistenceMetadata              `json:"meta"`
}

// UsageOverview is the expanded management response while preserving the legacy snapshot.
type UsageOverview struct {
	Usage          StatisticsSnapshot               `json:"usage"`
	FailedRequests int64                            `json:"failed_requests"`
	Providers      map[string]ProviderUsageSnapshot `json:"providers"`
	Models         map[string]UsageTotals           `json:"models"`
	RecentDays     []DayUsageSnapshot               `json:"recent_days"`
	Persistence    PersistenceMetadata              `json:"persistence"`
}

// HistoryQuery filters day-bucket history responses.
type HistoryQuery struct {
	StartDate string
	EndDate   string
	Provider  string
	Model     string
}

// PruneQuery selects persisted history to remove.
type PruneQuery struct {
	BeforeDate string
	StartDate  string
	EndDate    string
}

// MergeMode controls persisted import behavior.
type MergeMode string

const (
	// MergeModeMerge appends non-duplicate historical requests.
	MergeModeMerge MergeMode = "merge"
	// MergeModeOverwrite replaces the current history with the imported snapshot.
	MergeModeOverwrite MergeMode = "overwrite"
)

// ImportResult summarises a usage import operation.
type ImportResult struct {
	Added         int64               `json:"added"`
	Skipped       int64               `json:"skipped"`
	Mode          MergeMode           `json:"mode"`
	TotalRequests int64               `json:"total_requests"`
	Persistence   PersistenceMetadata `json:"persistence"`
}

// PruneResult summarises a prune operation.
type PruneResult struct {
	RemovedRequests int64               `json:"removed_requests"`
	RemovedDays     int                 `json:"removed_days"`
	TotalRequests   int64               `json:"total_requests"`
	Persistence     PersistenceMetadata `json:"persistence"`
}

func sortedDayKeys(days map[string]*dayBucket) []string {
	keys := make([]string, 0, len(days))
	for key := range days {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func parseUsageDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(usageDayLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

func formatUsageDate(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(usageDayLayout)
}

func humanizeBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := -1
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
