package management

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

type usageExportPayload struct {
	Version    int                      `json:"version"`
	ExportedAt time.Time                `json:"exported_at"`
	Usage      usage.StatisticsSnapshot `json:"usage"`
}

type usageFilterPayload struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

type usageImportOptions struct {
	Mode string `json:"mode"`
}

type usagePrunePayload struct {
	BeforeDate string `json:"before_date"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	FailedOnly bool   `json:"failed_only"`
}

// GetUsageStatistics returns the expanded usage overview while preserving the legacy snapshot shape.
func (h *Handler) GetUsageStatistics(c *gin.Context) {
	var overview usage.UsageOverview
	if h != nil && h.usageStats != nil {
		overview = h.usageStats.Overview(30)
	}
	c.JSON(http.StatusOK, overview)
}

// GetUsageHistory returns day-bucket usage history for the requested range.
func (h *Handler) GetUsageHistory(c *gin.Context) {
	if h == nil || h.usageStats == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage statistics unavailable"})
		return
	}
	days, err := h.usageStats.History(usage.HistoryQuery{
		StartDate: c.Query("start"),
		EndDate:   c.Query("end"),
		Provider:  c.Query("provider"),
		Model:     c.Query("model"),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": days})
}

// GetUsagePersistence returns persistence state and current file metadata.
func (h *Handler) GetUsagePersistence(c *gin.Context) {
	meta := usage.PersistenceMetadata{}
	if h != nil && h.usageStats != nil {
		meta = h.usageStats.PersistenceStatus()
	}
	c.JSON(http.StatusOK, meta)
}

// ExportUsageStatistics returns a legacy export on GET and the canonical export on POST.
func (h *Handler) ExportUsageStatistics(c *gin.Context) {
	if h == nil || h.usageStats == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage statistics unavailable"})
		return
	}
	if c.Request.Method == http.MethodGet {
		snapshot := h.usageStats.Snapshot()
		c.JSON(http.StatusOK, usageExportPayload{
			Version:    1,
			ExportedAt: time.Now().UTC(),
			Usage:      snapshot,
		})
		return
	}

	query := usage.HistoryQuery{
		StartDate: c.Query("start"),
		EndDate:   c.Query("end"),
		Provider:  c.Query("provider"),
		Model:     c.Query("model"),
	}
	var body usageFilterPayload
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if strings.TrimSpace(body.StartDate) != "" {
			query.StartDate = body.StartDate
		}
		if strings.TrimSpace(body.EndDate) != "" {
			query.EndDate = body.EndDate
		}
		if strings.TrimSpace(body.Provider) != "" {
			query.Provider = body.Provider
		}
		if strings.TrimSpace(body.Model) != "" {
			query.Model = body.Model
		}
	}

	snapshot, err := h.usageStats.Export(query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// ImportUsageStatistics merges or overwrites persisted usage history.
func (h *Handler) ImportUsageStatistics(c *gin.Context) {
	if h == nil || h.usageStats == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage statistics unavailable"})
		return
	}

	mode := usage.MergeMode(strings.TrimSpace(c.Query("mode")))
	if mode == "" {
		mode = usage.MergeModeMerge
	}

	if contentType := strings.ToLower(strings.TrimSpace(c.ContentType())); strings.Contains(contentType, "json") && c.Request.ContentLength > 0 {
		var body usageImportOptions
		if err := c.ShouldBindQuery(&body); err == nil && strings.TrimSpace(body.Mode) != "" {
			mode = usage.MergeMode(strings.TrimSpace(body.Mode))
		}
	}

	snapshot, err := usage.DecodePersistedSnapshot(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.usageStats.ImportPersisted(snapshot, mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// PruneUsageStatistics removes day-bucket history by date range.
func (h *Handler) PruneUsageStatistics(c *gin.Context) {
	if h == nil || h.usageStats == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage statistics unavailable"})
		return
	}
	payload := usagePrunePayload{
		BeforeDate: c.Query("before_date"),
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
	}
	result, err := h.usageStats.Prune(usage.PruneQuery{
		BeforeDate: payload.BeforeDate,
		StartDate:  payload.StartDate,
		EndDate:    payload.EndDate,
		FailedOnly: payload.FailedOnly,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
