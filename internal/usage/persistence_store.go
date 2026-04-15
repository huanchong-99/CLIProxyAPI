package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type providerTotalsState struct {
	Summary UsageTotals
	Models  map[string]UsageTotals
}

type dayBucket struct {
	Summary   UsageTotals
	Providers map[string]UsageTotals
	Models    map[string]UsageTotals
	Requests  []PersistedRequestRecord
}

func newDayBucket() *dayBucket {
	return &dayBucket{
		Providers: make(map[string]UsageTotals),
		Models:    make(map[string]UsageTotals),
	}
}

func (s *RequestStatistics) PersistentSnapshot() PersistedUsageSnapshot {
	result := PersistedUsageSnapshot{
		Version: PersistedUsageVersion,
	}
	if s == nil {
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.persistedSnapshotLocked()
}

// PersistenceStatus returns the latest persistence metadata snapshot.
func (s *RequestStatistics) PersistenceStatus() PersistenceMetadata {
	if s == nil {
		return PersistenceMetadata{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistence
}

// SetPersistenceStatus updates persistence metadata without mutating stored history.
func (s *RequestStatistics) SetPersistenceStatus(meta PersistenceMetadata) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistence = meta
	s.refreshPersistenceMetadataLocked()
}

func (s *RequestStatistics) persistedSnapshotLocked() PersistedUsageSnapshot {
	result := PersistedUsageSnapshot{
		Version:   PersistedUsageVersion,
		UpdatedAt: time.Now().UTC(),
		Summary:   s.summary,
		Providers: make(map[string]ProviderUsageSnapshot, len(s.providers)),
		Models:    make(map[string]UsageTotals, len(s.models)),
		Days:      make(map[string]DayUsageSnapshot, len(s.days)),
		Meta:      s.persistence,
	}
	for provider, stats := range s.providers {
		if stats == nil {
			continue
		}
		providerSnapshot := ProviderUsageSnapshot{
			Summary: stats.Summary,
			Models:  make(map[string]UsageTotals, len(stats.Models)),
		}
		for model, totals := range stats.Models {
			providerSnapshot.Models[model] = totals
		}
		result.Providers[provider] = providerSnapshot
	}
	for model, totals := range s.models {
		result.Models[model] = totals
	}
	for dayKey, bucket := range s.days {
		if bucket == nil {
			continue
		}
		daySnapshot := DayUsageSnapshot{
			Date:      dayKey,
			Summary:   bucket.Summary,
			Providers: make(map[string]UsageTotals, len(bucket.Providers)),
			Models:    make(map[string]UsageTotals, len(bucket.Models)),
			Requests:  append([]PersistedRequestRecord(nil), bucket.Requests...),
		}
		for provider, totals := range bucket.Providers {
			daySnapshot.Providers[provider] = totals
		}
		for model, totals := range bucket.Models {
			daySnapshot.Models[model] = totals
		}
		result.Days[dayKey] = daySnapshot
	}
	return result
}

// Overview returns the expanded management view while preserving the legacy snapshot.
func (s *RequestStatistics) Overview(recentDays int) UsageOverview {
	result := UsageOverview{
		Providers: make(map[string]ProviderUsageSnapshot),
		Models:    make(map[string]UsageTotals),
	}
	if s == nil {
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result.Usage = s.snapshotLocked()
	result.FailedRequests = result.Usage.FailureCount
	for provider, stats := range s.providers {
		if stats == nil {
			continue
		}
		providerSnapshot := ProviderUsageSnapshot{
			Summary: stats.Summary,
			Models:  make(map[string]UsageTotals, len(stats.Models)),
		}
		for model, totals := range stats.Models {
			providerSnapshot.Models[model] = totals
		}
		result.Providers[provider] = providerSnapshot
	}
	for model, totals := range s.models {
		result.Models[model] = totals
	}
	result.Persistence = s.persistence
	result.RecentDays = s.historyLocked(HistoryQuery{}, recentDays)
	return result
}

// History returns day-bucket history filtered by date, provider, and model.
func (s *RequestStatistics) History(query HistoryQuery) ([]DayUsageSnapshot, error) {
	if s == nil {
		return nil, nil
	}

	start, err := parseUsageDate(query.StartDate)
	if err != nil {
		return nil, err
	}
	end, err := parseUsageDate(query.EndDate)
	if err != nil {
		return nil, err
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return nil, fmt.Errorf("end_date must be on or after start_date")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.historyLocked(HistoryQuery{
		StartDate: formatUsageDate(start),
		EndDate:   formatUsageDate(end),
		Provider:  strings.TrimSpace(query.Provider),
		Model:     strings.TrimSpace(query.Model),
	}, 0), nil
}

// Export returns a canonical persisted snapshot filtered by the provided history query.
func (s *RequestStatistics) Export(query HistoryQuery) (PersistedUsageSnapshot, error) {
	result := PersistedUsageSnapshot{Version: PersistedUsageVersion}
	history, err := s.History(query)
	if err != nil {
		return result, err
	}

	filtered := NewRequestStatistics()
	filtered.SetPersistenceStatus(s.PersistenceStatus())
	dayMap := make(map[string]DayUsageSnapshot, len(history))
	for _, day := range history {
		dayMap[day.Date] = day
	}
	if _, err := filtered.ImportPersisted(PersistedUsageSnapshot{
		Version: PersistedUsageVersion,
		Days:    dayMap,
	}, MergeModeOverwrite); err != nil {
		return result, err
	}
	snapshot := filtered.PersistentSnapshot()
	snapshot.Meta.Path = s.PersistenceStatus().Path
	return snapshot, nil
}

func (s *RequestStatistics) historyLocked(query HistoryQuery, limit int) []DayUsageSnapshot {
	keys := sortedDayKeys(s.days)
	if limit > 0 && len(keys) > limit {
		keys = keys[len(keys)-limit:]
	}
	filterProvider := strings.TrimSpace(query.Provider)
	filterModel := strings.TrimSpace(query.Model)
	start := strings.TrimSpace(query.StartDate)
	end := strings.TrimSpace(query.EndDate)
	result := make([]DayUsageSnapshot, 0, len(keys))
	for _, dayKey := range keys {
		if start != "" && dayKey < start {
			continue
		}
		if end != "" && dayKey > end {
			continue
		}
		bucket := s.days[dayKey]
		if bucket == nil {
			continue
		}
		daySnapshot := buildFilteredDaySnapshot(dayKey, bucket, filterProvider, filterModel)
		if daySnapshot == nil || daySnapshot.Summary.Requests == 0 {
			continue
		}
		result = append(result, *daySnapshot)
	}
	return result
}

func buildFilteredDaySnapshot(dayKey string, bucket *dayBucket, provider, model string) *DayUsageSnapshot {
	if bucket == nil {
		return nil
	}
	if provider == "" && model == "" {
		snapshot := &DayUsageSnapshot{
			Date:      dayKey,
			Summary:   bucket.Summary,
			Providers: make(map[string]UsageTotals, len(bucket.Providers)),
			Models:    make(map[string]UsageTotals, len(bucket.Models)),
			Requests:  append([]PersistedRequestRecord(nil), bucket.Requests...),
		}
		for key, totals := range bucket.Providers {
			snapshot.Providers[key] = totals
		}
		for key, totals := range bucket.Models {
			snapshot.Models[key] = totals
		}
		return snapshot
	}

	filtered := &DayUsageSnapshot{
		Date:      dayKey,
		Providers: make(map[string]UsageTotals),
		Models:    make(map[string]UsageTotals),
	}
	for _, record := range bucket.Requests {
		if provider != "" && record.Provider != provider {
			continue
		}
		if model != "" && record.Model != model {
			continue
		}
		filtered.Requests = append(filtered.Requests, record)
		filtered.Summary.addDetail(record.Tokens, record.Failed)
		if record.Provider != "" {
			totals := filtered.Providers[record.Provider]
			totals.addDetail(record.Tokens, record.Failed)
			filtered.Providers[record.Provider] = totals
		}
		if record.Model != "" {
			totals := filtered.Models[record.Model]
			totals.addDetail(record.Tokens, record.Failed)
			filtered.Models[record.Model] = totals
		}
	}
	return filtered
}

// ImportPersisted imports canonical persisted usage history.
func (s *RequestStatistics) ImportPersisted(snapshot PersistedUsageSnapshot, mode MergeMode) (ImportResult, error) {
	result := ImportResult{Mode: mode}
	if s == nil {
		return result, nil
	}
	if mode == "" {
		mode = MergeModeMerge
		result.Mode = mode
	}
	if mode != MergeModeMerge && mode != MergeModeOverwrite {
		return result, fmt.Errorf("unsupported import mode %q", mode)
	}
	if snapshot.Version != 0 && snapshot.Version != PersistedUsageVersion {
		return result, fmt.Errorf("unsupported persisted usage version %d", snapshot.Version)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if mode == MergeModeOverwrite {
		s.resetLocked()
	}

	seen := s.dedupSetLocked()
	keys := make([]string, 0, len(snapshot.Days))
	for key := range snapshot.Days {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, dayKey := range keys {
		daySnapshot := snapshot.Days[dayKey]
		records := daySnapshot.Requests
		if len(records) == 0 {
			continue
		}
		for _, record := range records {
			record = normalisePersistedRecord(record, dayKey)
			dedup := record.dedupKey()
			if _, exists := seen[dedup]; exists {
				result.Skipped++
				continue
			}
			seen[dedup] = struct{}{}
			s.applyPersistedRecordLocked(record)
			result.Added++
		}
	}
	s.refreshPersistenceMetadataLocked()
	result.TotalRequests = s.summary.Requests
	result.Persistence = s.persistence
	return result, nil
}

// LoadPersisted replaces the in-memory statistics with a canonical persisted snapshot.
func (s *RequestStatistics) LoadPersisted(snapshot PersistedUsageSnapshot) error {
	_, err := s.ImportPersisted(snapshot, MergeModeOverwrite)
	return err
}

// Prune removes persisted history by date.
func (s *RequestStatistics) Prune(query PruneQuery) (PruneResult, error) {
	result := PruneResult{}
	if s == nil {
		return result, nil
	}

	before, err := parseUsageDate(query.BeforeDate)
	if err != nil {
		return result, err
	}
	start, err := parseUsageDate(query.StartDate)
	if err != nil {
		return result, err
	}
	end, err := parseUsageDate(query.EndDate)
	if err != nil {
		return result, err
	}
	if before.IsZero() && start.IsZero() && end.IsZero() {
		return result, fmt.Errorf("no prune range specified")
	}
	if !before.IsZero() && (!start.IsZero() || !end.IsZero()) {
		return result, fmt.Errorf("before_date cannot be combined with start_date or end_date")
	}
	if !start.IsZero() && end.IsZero() {
		end = start
	}
	if start.IsZero() && !end.IsZero() {
		start = end
	}
	if !start.IsZero() && end.Before(start) {
		return result, fmt.Errorf("end_date must be on or after start_date")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	keys := sortedDayKeys(s.days)
	removedRequests := int64(0)
	removedDays := 0
	for _, dayKey := range keys {
		remove := false
		dayTime, err := parseUsageDate(dayKey)
		if err != nil {
			log.WithError(err).Warnf("usage: skipping invalid day bucket %q during prune", dayKey)
			continue
		}
		if !before.IsZero() {
			remove = dayTime.Before(before)
		} else {
			remove = (dayTime.Equal(start) || dayTime.After(start)) && (dayTime.Equal(end) || dayTime.Before(end))
		}
		if !remove {
			continue
		}
		if bucket := s.days[dayKey]; bucket != nil {
			removedRequests += int64(len(bucket.Requests))
		}
		delete(s.days, dayKey)
		removedDays++
	}
	if removedDays > 0 {
		s.rebuildLocked()
	}
	result.RemovedRequests = removedRequests
	result.RemovedDays = removedDays
	result.TotalRequests = s.summary.Requests
	result.Persistence = s.persistence
	return result, nil
}

// SaveToFile persists the canonical usage snapshot to disk safely.
func (s *RequestStatistics) SaveToFile(path string) (PersistenceMetadata, error) {
	meta := PersistenceMetadata{}
	if s == nil {
		return meta, nil
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return meta, fmt.Errorf("usage persistence path is empty")
	}

	s.mu.RLock()
	snapshot := s.persistedSnapshotLocked()
	s.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return meta, fmt.Errorf("marshal usage snapshot: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return meta, fmt.Errorf("create usage persistence directory: %w", err)
	}

	tempPath := path + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return meta, fmt.Errorf("open temp usage snapshot: %w", err)
	}
	closeWithErr := func() error {
		if errClose := file.Close(); errClose != nil {
			return fmt.Errorf("close temp usage snapshot: %w", errClose)
		}
		return nil
	}
	if _, err = file.Write(data); err != nil {
		_ = closeWithErr()
		return meta, fmt.Errorf("write temp usage snapshot: %w", err)
	}
	if err = file.Sync(); err != nil {
		_ = closeWithErr()
		return meta, fmt.Errorf("sync temp usage snapshot: %w", err)
	}
	if err = closeWithErr(); err != nil {
		return meta, err
	}
	if _, err = os.Stat(path); err == nil {
		if errRemove := os.Remove(path); errRemove != nil {
			return meta, fmt.Errorf("remove previous usage snapshot: %w", errRemove)
		}
	}
	if err = os.Rename(tempPath, path); err != nil {
		return meta, fmt.Errorf("replace usage snapshot: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return meta, fmt.Errorf("stat usage snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistence.Enabled = true
	s.persistence.Path = path
	s.persistence.FileSizeBytes = info.Size()
	s.persistence.FileSizeHuman = humanizeBytes(info.Size())
	s.persistence.LastFlushAt = time.Now().UTC()
	s.refreshPersistenceMetadataLocked()
	meta = s.persistence
	return meta, nil
}

// LoadFromFile restores canonical usage history from disk.
func (s *RequestStatistics) LoadFromFile(path string) (PersistenceMetadata, error) {
	meta := PersistenceMetadata{}
	if s == nil {
		return meta, nil
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return meta, fmt.Errorf("usage persistence path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return meta, err
	}
	var snapshot PersistedUsageSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return meta, fmt.Errorf("decode usage snapshot: %w", err)
	}
	if err := s.LoadPersisted(snapshot); err != nil {
		return meta, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return meta, fmt.Errorf("stat usage snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistence.Enabled = true
	s.persistence.Path = path
	s.persistence.FileSizeBytes = info.Size()
	s.persistence.FileSizeHuman = humanizeBytes(info.Size())
	if !snapshot.Meta.LastFlushAt.IsZero() {
		s.persistence.LastFlushAt = snapshot.Meta.LastFlushAt
	}
	s.refreshPersistenceMetadataLocked()
	meta = s.persistence
	return meta, nil
}

func (s *RequestStatistics) refreshPersistenceMetadataLocked() {
	keys := sortedDayKeys(s.days)
	s.persistence.RecordedDays = len(keys)
	s.persistence.FileSizeHuman = humanizeBytes(s.persistence.FileSizeBytes)
	if len(keys) == 0 {
		s.persistence.OldestDate = ""
		s.persistence.NewestDate = ""
		return
	}
	s.persistence.OldestDate = keys[0]
	s.persistence.NewestDate = keys[len(keys)-1]
}

func (s *RequestStatistics) dedupSetLocked() map[string]struct{} {
	seen := make(map[string]struct{})
	for _, bucket := range s.days {
		if bucket == nil {
			continue
		}
		for _, record := range bucket.Requests {
			seen[record.dedupKey()] = struct{}{}
		}
	}
	return seen
}

func (s *RequestStatistics) applyPersistedRecordLocked(record PersistedRequestRecord) {
	record = normalisePersistedRecord(record, formatUsageDate(record.Timestamp))
	statsKey := strings.TrimSpace(record.APIKey)
	if statsKey == "" {
		statsKey = strings.TrimSpace(record.APIName)
	}
	if statsKey == "" {
		statsKey = strings.TrimSpace(record.Provider)
	}
	if statsKey == "" {
		statsKey = "unknown"
	}

	s.totalRequests++
	if record.Failed {
		s.failureCount++
	} else {
		s.successCount++
	}
	s.totalTokens += record.Tokens.TotalTokens
	s.summary.addDetail(record.Tokens, record.Failed)

	stats, ok := s.apis[statsKey]
	if !ok {
		stats = &apiStats{Models: make(map[string]*modelStats)}
		s.apis[statsKey] = stats
	}
	s.updateAPIStats(stats, record.Model, RequestDetail{
		Timestamp: record.Timestamp,
		LatencyMs: record.LatencyMs,
		Source:    record.Source,
		AuthIndex: record.AuthIndex,
		Provider:  record.Provider,
		APIKey:    record.APIKey,
		APIName:   record.APIName,
		AuthID:    record.AuthID,
		Tokens:    record.Tokens,
		Failed:    record.Failed,
	})

	provider := strings.TrimSpace(record.Provider)
	if provider == "" {
		provider = "unknown"
	}
	providerStats, ok := s.providers[provider]
	if !ok {
		providerStats = &providerTotalsState{Models: make(map[string]UsageTotals)}
		s.providers[provider] = providerStats
	}
	providerStats.Summary.addDetail(record.Tokens, record.Failed)
	modelTotals := providerStats.Models[record.Model]
	modelTotals.addDetail(record.Tokens, record.Failed)
	providerStats.Models[record.Model] = modelTotals

	overallModel := s.models[record.Model]
	overallModel.addDetail(record.Tokens, record.Failed)
	s.models[record.Model] = overallModel

	dayKey := formatUsageDate(record.Timestamp)
	bucket := s.days[dayKey]
	if bucket == nil {
		bucket = newDayBucket()
		s.days[dayKey] = bucket
	}
	bucket.Summary.addDetail(record.Tokens, record.Failed)
	dayProviderTotals := bucket.Providers[provider]
	dayProviderTotals.addDetail(record.Tokens, record.Failed)
	bucket.Providers[provider] = dayProviderTotals
	dayModelTotals := bucket.Models[record.Model]
	dayModelTotals.addDetail(record.Tokens, record.Failed)
	bucket.Models[record.Model] = dayModelTotals
	bucket.Requests = append(bucket.Requests, record)

	s.requestsByDay[dayKey]++
	s.requestsByHour[record.Timestamp.UTC().Hour()]++
	s.tokensByDay[dayKey] += record.Tokens.TotalTokens
	s.tokensByHour[record.Timestamp.UTC().Hour()] += record.Tokens.TotalTokens
}

func (s *RequestStatistics) resetLocked() {
	s.totalRequests = 0
	s.successCount = 0
	s.failureCount = 0
	s.totalTokens = 0
	s.summary = UsageTotals{}
	s.apis = make(map[string]*apiStats)
	s.providers = make(map[string]*providerTotalsState)
	s.models = make(map[string]UsageTotals)
	s.days = make(map[string]*dayBucket)
	s.requestsByDay = make(map[string]int64)
	s.requestsByHour = make(map[int]int64)
	s.tokensByDay = make(map[string]int64)
	s.tokensByHour = make(map[int]int64)
	s.refreshPersistenceMetadataLocked()
}

func (s *RequestStatistics) rebuildLocked() {
	existingDays := s.days
	s.resetLocked()
	keys := sortedDayKeys(existingDays)
	for _, dayKey := range keys {
		bucket := existingDays[dayKey]
		if bucket == nil {
			continue
		}
		for _, record := range bucket.Requests {
			s.applyPersistedRecordLocked(record)
		}
	}
}

func normalisePersistedRecord(record PersistedRequestRecord, fallbackDay string) PersistedRequestRecord {
	if record.Timestamp.IsZero() {
		if dayTime, err := parseUsageDate(fallbackDay); err == nil && !dayTime.IsZero() {
			record.Timestamp = dayTime
		} else {
			record.Timestamp = time.Now().UTC()
		}
	}
	record.Timestamp = record.Timestamp.UTC()
	record.Provider = strings.TrimSpace(record.Provider)
	record.APIKey = strings.TrimSpace(record.APIKey)
	record.APIName = strings.TrimSpace(record.APIName)
	record.AuthID = strings.TrimSpace(record.AuthID)
	record.AuthIndex = strings.TrimSpace(record.AuthIndex)
	record.Model = strings.TrimSpace(record.Model)
	if record.Model == "" {
		record.Model = "unknown"
	}
	record.Tokens = normaliseTokenStats(record.Tokens)
	if record.LatencyMs < 0 {
		record.LatencyMs = 0
	}
	return record
}

// DecodePersistedSnapshot decodes either the canonical or legacy usage export payload.
func DecodePersistedSnapshot(r io.Reader) (PersistedUsageSnapshot, error) {
	var snapshot PersistedUsageSnapshot
	if r == nil {
		return snapshot, fmt.Errorf("usage import body is empty")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return snapshot, fmt.Errorf("read usage import body: %w", err)
	}
	if err := json.Unmarshal(data, &snapshot); err == nil && (snapshot.Version == 0 || snapshot.Version == PersistedUsageVersion) {
		return snapshot, nil
	}

	var legacy struct {
		Version int                `json:"version"`
		Usage   StatisticsSnapshot `json:"usage"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return snapshot, fmt.Errorf("invalid usage import json: %w", err)
	}
	if legacy.Version != 0 && legacy.Version != 1 {
		return snapshot, fmt.Errorf("unsupported usage import version %d", legacy.Version)
	}
	converted := NewRequestStatistics()
	converted.MergeSnapshot(legacy.Usage)
	return converted.PersistentSnapshot(), nil
}
