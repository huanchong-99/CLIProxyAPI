package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	usagepkg "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

// dashboardModel displays server info, stats cards, and config overview.
type dashboardModel struct {
	client   *Client
	viewport viewport.Model
	content  string
	err      error
	width    int
	height   int
	ready    bool

	// Cached data for re-rendering on locale change.
	lastConfig    map[string]any
	lastUsage     usagepkg.UsageOverview
	lastAuthFiles []map[string]any
	lastAPIKeys   []string
}

type dashboardDataMsg struct {
	config    map[string]any
	usage     usagepkg.UsageOverview
	authFiles []map[string]any
	apiKeys   []string
	err       error
}

func newDashboardModel(client *Client) dashboardModel {
	return dashboardModel{client: client}
}

func (m dashboardModel) Init() tea.Cmd {
	return m.fetchData
}

func (m dashboardModel) fetchData() tea.Msg {
	cfg, cfgErr := m.client.GetConfig()
	usage, usageErr := m.client.GetUsage()
	authFiles, authErr := m.client.GetAuthFiles()
	apiKeys, keysErr := m.client.GetAPIKeys()

	var err error
	for _, e := range []error{cfgErr, usageErr, authErr, keysErr} {
		if e != nil {
			err = e
			break
		}
	}
	return dashboardDataMsg{config: cfg, usage: usage, authFiles: authFiles, apiKeys: apiKeys, err: err}
}

func (m dashboardModel) Update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case localeChangedMsg:
		m.content = m.renderDashboard(m.lastConfig, m.lastUsage, m.lastAuthFiles, m.lastAPIKeys)
		m.viewport.SetContent(m.content)
		return m, m.fetchData

	case dashboardDataMsg:
		if msg.err != nil {
			m.err = msg.err
			m.content = errorStyle.Render("鈿?Error: " + msg.err.Error())
		} else {
			m.err = nil
			m.lastConfig = msg.config
			m.lastUsage = msg.usage
			m.lastAuthFiles = msg.authFiles
			m.lastAPIKeys = msg.apiKeys
			m.content = m.renderDashboard(msg.config, msg.usage, msg.authFiles, msg.apiKeys)
		}
		m.viewport.SetContent(m.content)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			return m, m.fetchData
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *dashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	if !m.ready {
		m.viewport = viewport.New(w, h)
		m.viewport.SetContent(m.content)
		m.ready = true
	} else {
		m.viewport.Width = w
		m.viewport.Height = h
	}
}

func (m dashboardModel) View() string {
	if !m.ready {
		return T("loading")
	}
	return m.viewport.View()
}

func (m dashboardModel) renderDashboard(cfg map[string]any, usage usagepkg.UsageOverview, authFiles []map[string]any, apiKeys []string) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(T("dashboard_title")))
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render(T("dashboard_help")))
	sb.WriteString("\n\n")

	// Connection status.
	connStyle := lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	baseURL := ""
	if m.client != nil {
		baseURL = m.client.baseURL
	}
	sb.WriteString(connStyle.Render(T("connected")))
	sb.WriteString(fmt.Sprintf("  %s", baseURL))
	sb.WriteString("\n\n")

	// Stats cards.
	cardWidth := 25
	if m.width > 0 {
		cardWidth = (m.width - 6) / 4
		if cardWidth < 18 {
			cardWidth = 18
		}
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(cardWidth).
		Height(2)

	keyCount := len(apiKeys)
	card1 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111")).Render(fmt.Sprintf("🔑 %d", keyCount)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(T("mgmt_keys")),
	))

	authCount := len(authFiles)
	activeAuth := 0
	for _, f := range authFiles {
		if !getBool(f, "disabled") {
			activeAuth++
		}
	}
	card2 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("76")).Render(fmt.Sprintf("📄 %d", authCount)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%s (%d %s)", T("auth_files_label"), activeAuth, T("active_suffix"))),
	))

	totalReqs := usage.Usage.TotalRequests
	successReqs := usage.Usage.SuccessCount
	failedReqs := usage.Usage.FailureCount
	totalTokens := usage.Usage.TotalTokens
	card3 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(fmt.Sprintf("📈 %d", totalReqs)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%s (✓%d ✗%d)", T("total_requests"), successReqs, failedReqs)),
	))

	tokenStr := formatLargeNumber(totalTokens)
	card4 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render(fmt.Sprintf("🔤 %s", tokenStr)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(T("total_tokens")),
	))

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, card1, " ", card2, " ", card3, " ", card4))
	sb.WriteString("\n\n")

	if hasUsagePersistence(usage.Persistence) {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render("Usage persistence"))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")
		sb.WriteString(renderPersistenceSummary(usage.Persistence))
		sb.WriteString("\n")
	}

	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(T("current_config")))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
	sb.WriteString("\n")

	if cfg != nil {
		debug := getBool(cfg, "debug")
		retry := getFloat(cfg, "request-retry")
		proxyURL := getString(cfg, "proxy-url")
		loggingToFile := getBool(cfg, "logging-to-file")
		usageEnabled := true
		if v, ok := cfg["usage-statistics-enabled"]; ok {
			if b, ok2 := v.(bool); ok2 {
				usageEnabled = b
			}
		}

		configItems := []struct {
			label string
			value string
		}{
			{T("debug_mode"), boolEmoji(debug)},
			{T("usage_stats"), boolEmoji(usageEnabled)},
			{T("log_to_file"), boolEmoji(loggingToFile)},
			{T("retry_count"), fmt.Sprintf("%.0f", retry)},
		}
		if proxyURL != "" {
			configItems = append(configItems, struct {
				label string
				value string
			}{T("proxy_url"), proxyURL})
		}

		for _, item := range configItems {
			sb.WriteString(fmt.Sprintf("  %s %s\n",
				labelStyle.Render(item.label+":"), valueStyle.Render(item.value)))
		}

		strategy := "round-robin"
		if routing, ok := cfg["routing"].(map[string]any); ok {
			if s := getString(routing, "strategy"); s != "" {
				strategy = s
			}
		}
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render(T("routing_strategy")+":"), valueStyle.Render(strategy)))
	}

	sb.WriteString("\n")

	if len(usage.RecentDays) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render("Recent Days"))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")
		sb.WriteString(renderRecentDaysSummary(usage.RecentDays))
		sb.WriteString("\n")
	}

	if len(usage.Usage.APIs) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(T("model_stats")))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")

		header := fmt.Sprintf("  %-40s %10s %12s", T("model"), T("requests"), T("tokens"))
		sb.WriteString(tableHeaderStyle.Render(header))
		sb.WriteString("\n")

		for _, apiSnap := range usage.Usage.APIs {
			for model, stats := range apiSnap.Models {
				row := fmt.Sprintf("  %-40s %10d %12s", truncate(model, 40), stats.TotalRequests, formatLargeNumber(stats.TotalTokens))
				sb.WriteString(tableCellStyle.Render(row))
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

func formatKV(key, value string) string {
	return fmt.Sprintf("  %s %s\n", labelStyle.Render(key+":"), valueStyle.Render(value))
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case json.Number:
			f, _ := n.Float64()
			return f
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func boolEmoji(b bool) string {
	if b {
		return T("bool_yes")
	}
	return T("bool_no")
}

func formatLargeNumber(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func hasUsagePersistence(meta usagepkg.PersistenceMetadata) bool {
	return meta.Enabled || meta.Path != "" || meta.FileSizeBytes > 0 || meta.RecordedDays > 0 || meta.LastFlushAt != (time.Time{})
}

func renderPersistenceSummary(meta usagepkg.PersistenceMetadata) string {
	var sb strings.Builder
	lines := []string{}

	if meta.Path != "" {
		lines = append(lines, fmt.Sprintf("  Path: %s", meta.Path))
	}
	if size := formatFileSize(meta); size != "" {
		lines = append(lines, fmt.Sprintf("  File size: %s", size))
	}
	if meta.RecordedDays > 0 {
		lines = append(lines, fmt.Sprintf("  Recorded days: %d", meta.RecordedDays))
	}
	if meta.OldestDate != "" || meta.NewestDate != "" {
		lines = append(lines, fmt.Sprintf("  Date range: %s - %s", emptyFallback(meta.OldestDate), emptyFallback(meta.NewestDate)))
	}
	if !meta.LastFlushAt.IsZero() {
		lines = append(lines, fmt.Sprintf("  Last flush: %s", meta.LastFlushAt.UTC().Format(time.RFC3339)))
	}

	if len(lines) == 0 {
		lines = append(lines, "  Persistence metadata unavailable")
	}

	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

func renderRecentDaysSummary(days []usagepkg.DayUsageSnapshot) string {
	if len(days) == 0 {
		return ""
	}

	var sb strings.Builder
	header := fmt.Sprintf("  %-12s %10s %12s %12s", "Date", "Requests", "Tokens", "Providers")
	sb.WriteString(tableHeaderStyle.Render(header))
	sb.WriteString("\n")
	for _, day := range days {
		row := fmt.Sprintf("  %-12s %10d %12s %12d",
			day.Date,
			day.Summary.Requests,
			formatLargeNumber(day.Summary.TotalTokens),
			len(day.Providers),
		)
		sb.WriteString(tableCellStyle.Render(row))
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatFileSize(meta usagepkg.PersistenceMetadata) string {
	if strings.TrimSpace(meta.FileSizeHuman) != "" {
		return meta.FileSizeHuman
	}
	if meta.FileSizeBytes <= 0 {
		return ""
	}
	return humanizeBytes(meta.FileSizeBytes)
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

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(unknown)"
	}
	return value
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
