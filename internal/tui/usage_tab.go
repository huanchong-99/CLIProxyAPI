package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	usagepkg "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

// usageTabModel displays usage statistics with charts and breakdowns.
type usageTabModel struct {
	client   *Client
	viewport viewport.Model
	usage    usagepkg.UsageOverview
	err      error
	width    int
	height   int
	ready    bool
}

type usageDataMsg struct {
	usage usagepkg.UsageOverview
	err   error
}

func newUsageTabModel(client *Client) usageTabModel {
	return usageTabModel{client: client}
}

func (m usageTabModel) Init() tea.Cmd {
	return m.fetchData
}

func (m usageTabModel) fetchData() tea.Msg {
	usage, err := m.client.GetUsage()
	return usageDataMsg{usage: usage, err: err}
}

func (m usageTabModel) Update(msg tea.Msg) (usageTabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case localeChangedMsg:
		m.viewport.SetContent(m.renderContent())
		return m, nil
	case usageDataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.usage = msg.usage
		}
		m.viewport.SetContent(m.renderContent())
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

func (m *usageTabModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	if !m.ready {
		m.viewport = viewport.New(w, h)
		m.viewport.SetContent(m.renderContent())
		m.ready = true
		return
	}
	m.viewport.Width = w
	m.viewport.Height = h
}

func (m usageTabModel) View() string {
	if !m.ready {
		return T("loading")
	}
	return m.viewport.View()
}

func (m usageTabModel) renderContent() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(T("usage_title")))
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render(T("usage_help")))
	sb.WriteString("\n\n")

	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		sb.WriteString("\n")
		return sb.String()
	}

	snapshot := m.usage.Usage
	if snapshot.TotalRequests == 0 && len(snapshot.APIs) == 0 && len(m.usage.RecentDays) == 0 {
		sb.WriteString(subtitleStyle.Render(T("usage_no_data")))
		sb.WriteString("\n")
		return sb.String()
	}

	totalReqs := snapshot.TotalRequests
	successCnt := snapshot.SuccessCount
	failureCnt := snapshot.FailureCount
	totalTokens := snapshot.TotalTokens

	cardWidth := 20
	if m.width > 0 {
		cardWidth = (m.width - 6) / 4
		if cardWidth < 16 {
			cardWidth = 16
		}
	}
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(cardWidth).
		Height(3)

	card1 := cardStyle.Copy().BorderForeground(lipgloss.Color("111")).Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(colorMuted).Render(T("usage_total_reqs")),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111")).Render(fmt.Sprintf("%d", totalReqs)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%s: %d  %s: %d", T("usage_success"), successCnt, T("usage_failure"), failureCnt)),
	))
	card2 := cardStyle.Copy().BorderForeground(lipgloss.Color("214")).Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(colorMuted).Render(T("usage_total_tokens")),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(formatLargeNumber(totalTokens)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%s: %s", T("usage_total_token_l"), formatLargeNumber(totalTokens))),
	))
	card3 := cardStyle.Copy().BorderForeground(lipgloss.Color("76")).Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(colorMuted).Render("Providers"),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("76")).Render(fmt.Sprintf("%d", len(m.usage.Providers))),
		lipgloss.NewStyle().Foreground(colorMuted).Render("Breakdown by provider"),
	))
	card4 := cardStyle.Copy().BorderForeground(lipgloss.Color("170")).Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(colorMuted).Render("Days"),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render(fmt.Sprintf("%d", len(m.usage.RecentDays))),
		lipgloss.NewStyle().Foreground(colorMuted).Render("Recent history"),
	))

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, card1, " ", card2, " ", card3, " ", card4))
	sb.WriteString("\n\n")

	if hasUsagePersistence(m.usage.Persistence) {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render("Usage Persistence"))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")
		sb.WriteString(renderPersistenceSummary(m.usage.Persistence))
		sb.WriteString("\n")
	}

	if len(snapshot.RequestsByHour) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(T("usage_req_by_hour")))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")
		sb.WriteString(renderIntBarChart(snapshot.RequestsByHour, m.width-6, lipgloss.Color("111")))
		sb.WriteString("\n")
	}

	if len(snapshot.TokensByHour) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(T("usage_tok_by_hour")))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")
		sb.WriteString(renderIntBarChart(snapshot.TokensByHour, m.width-6, lipgloss.Color("214")))
		sb.WriteString("\n")
	}

	if len(m.usage.RecentDays) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render("Recent Days"))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 60)))
		sb.WriteString("\n")
		sb.WriteString(renderRecentDaysSummary(m.usage.RecentDays))
		sb.WriteString("\n")
	}

	if len(snapshot.APIs) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(T("usage_api_detail")))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", minInt(m.width, 80)))
		sb.WriteString("\n")

		header := fmt.Sprintf("  %-30s %10s %12s", "API", T("requests"), T("tokens"))
		sb.WriteString(tableHeaderStyle.Render(header))
		sb.WriteString("\n")

		apiNames := make([]string, 0, len(snapshot.APIs))
		for apiName := range snapshot.APIs {
			apiNames = append(apiNames, apiName)
		}
		sort.Strings(apiNames)
		for _, apiName := range apiNames {
			apiSnap := snapshot.APIs[apiName]
			row := fmt.Sprintf("  %-30s %10d %12s",
				truncate(maskKey(apiName), 30), apiSnap.TotalRequests, formatLargeNumber(apiSnap.TotalTokens))
			sb.WriteString(lipgloss.NewStyle().Bold(true).Render(row))
			sb.WriteString("\n")

			modelNames := make([]string, 0, len(apiSnap.Models))
			for model := range apiSnap.Models {
				modelNames = append(modelNames, model)
			}
			sort.Strings(modelNames)
			for _, model := range modelNames {
				stats := apiSnap.Models[model]
				modelRow := fmt.Sprintf("    ├─ %-28s %10d %12s",
					truncate(model, 28), stats.TotalRequests, formatLargeNumber(stats.TotalTokens))
				sb.WriteString(tableCellStyle.Render(modelRow))
				sb.WriteString("\n")
				sb.WriteString(renderModelTokenBreakdown(stats.Details))
				sb.WriteString(renderModelLatencyBreakdown(stats.Details))
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func renderModelTokenBreakdown(details []usagepkg.RequestDetail) string {
	if len(details) == 0 {
		return ""
	}
	var inputTotal, outputTotal, cachedTotal, reasoningTotal int64
	for _, detail := range details {
		inputTotal += detail.Tokens.InputTokens
		outputTotal += detail.Tokens.OutputTokens
		cachedTotal += detail.Tokens.CachedTokens
		reasoningTotal += detail.Tokens.ReasoningTokens
	}
	parts := []string{}
	if inputTotal > 0 {
		parts = append(parts, fmt.Sprintf("%s:%s", T("usage_input"), formatLargeNumber(inputTotal)))
	}
	if outputTotal > 0 {
		parts = append(parts, fmt.Sprintf("%s:%s", T("usage_output"), formatLargeNumber(outputTotal)))
	}
	if cachedTotal > 0 {
		parts = append(parts, fmt.Sprintf("%s:%s", T("usage_cached"), formatLargeNumber(cachedTotal)))
	}
	if reasoningTotal > 0 {
		parts = append(parts, fmt.Sprintf("%s:%s", T("usage_reasoning"), formatLargeNumber(reasoningTotal)))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("    │  %s\n", lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Join(parts, "  ")))
}

func renderModelLatencyBreakdown(details []usagepkg.RequestDetail) string {
	if len(details) == 0 {
		return ""
	}
	var totalLatency int64
	var count int64
	var minLatency, maxLatency int64
	for i, detail := range details {
		latencyMs := detail.LatencyMs
		if latencyMs <= 0 {
			continue
		}
		totalLatency += latencyMs
		count++
		if i == 0 || minLatency == 0 || latencyMs < minLatency {
			minLatency = latencyMs
		}
		if latencyMs > maxLatency {
			maxLatency = latencyMs
		}
	}
	if count == 0 {
		return ""
	}
	avgLatency := totalLatency / count
	return fmt.Sprintf("    │  %s: avg %dms  min %dms  max %dms\n",
		lipgloss.NewStyle().Foreground(colorMuted).Render(T("usage_time")),
		avgLatency, minLatency, maxLatency)
}

// renderLatencyBreakdown keeps the historical test helper signature for map-based stats.
func (m usageTabModel) renderLatencyBreakdown(modelStats map[string]any) string {
	details, ok := modelStats["details"]
	if !ok {
		return ""
	}
	detailList, ok := details.([]any)
	if !ok || len(detailList) == 0 {
		return ""
	}
	typed := make([]usagepkg.RequestDetail, 0, len(detailList))
	for _, detail := range detailList {
		detailMap, ok := detail.(map[string]any)
		if !ok {
			continue
		}
		typed = append(typed, usagepkg.RequestDetail{
			LatencyMs: int64(getFloat(detailMap, "latency_ms")),
		})
	}
	return renderModelLatencyBreakdown(typed)
}

func renderIntBarChart(data map[string]int64, maxBarWidth int, barColor lipgloss.Color) string {
	if maxBarWidth < 10 {
		maxBarWidth = 10
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	maxVal := int64(0)
	for _, key := range keys {
		if data[key] > maxVal {
			maxVal = data[key]
		}
	}
	if maxVal == 0 {
		return ""
	}

	barStyle := lipgloss.NewStyle().Foreground(barColor)
	labelWidth := 12
	barAvail := maxBarWidth - labelWidth - 12
	if barAvail < 5 {
		barAvail = 5
	}

	var sb strings.Builder
	for _, key := range keys {
		value := data[key]
		barLen := int(float64(value) / float64(maxVal) * float64(barAvail))
		if barLen < 1 && value > 0 {
			barLen = 1
		}
		sb.WriteString(fmt.Sprintf("  %-*s %s %s\n",
			labelWidth,
			truncate(key, labelWidth),
			barStyle.Render(strings.Repeat("█", barLen)),
			lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d", value)),
		))
	}
	return sb.String()
}
