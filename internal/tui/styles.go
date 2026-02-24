package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorCyan     = lipgloss.Color("#00FFFF")
	colorMagenta  = lipgloss.Color("#FF00FF")
	colorGreen    = lipgloss.Color("#39FF14")
	colorHotPink  = lipgloss.Color("#FF2E97")
	colorAmber    = lipgloss.Color("#FFB000")
	colorBgHeader = lipgloss.Color("#1A0A2E")
	colorFg       = lipgloss.Color("#E0E0E0")
	colorMuted    = lipgloss.Color("#6B7280")
	colorFaint    = lipgloss.Color("#374151")
	colorBorder   = lipgloss.Color("#00BFFF")
)

var (
	headerFillStyle  = lipgloss.NewStyle().Background(colorBgHeader)
	headerTitleStyle = headerFillStyle.Foreground(colorCyan).Bold(true)
	headerSubStyle   = headerFillStyle.Foreground(colorMagenta)
	headerLabelStyle = headerFillStyle.Foreground(colorMuted)
	headerValueStyle = headerFillStyle.Foreground(colorCyan).Bold(true)

	panelStyle      = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(colorBorder).Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorMagenta)

	labelStyle = lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle = lipgloss.NewStyle().Foreground(colorFg)
	faintStyle = lipgloss.NewStyle().Foreground(colorFaint)
	infoStyle  = lipgloss.NewStyle().Foreground(colorAmber)
	errorStyle = lipgloss.NewStyle().Foreground(colorHotPink)
	okStyle    = lipgloss.NewStyle().Foreground(colorGreen)

	footerKeyStyle  = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	footerDescStyle = lipgloss.NewStyle().Foreground(colorFaint)
)

func renderHeader(width int, left, right string) string {
	if width <= 0 {
		return left + " " + right
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + headerFillStyle.Render(strings.Repeat(" ", gap)) + right
}

func renderDivider(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("═", width))
}

func renderFooterKeys(width int, pairs ...string) string {
	parts := make([]string, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, footerKeyStyle.Render(pairs[i])+" "+footerDescStyle.Render(pairs[i+1]))
	}
	line := strings.Join(parts, footerDescStyle.Render("  "))
	if width > 0 {
		return lipgloss.NewStyle().Width(width).Render(line)
	}
	return line
}

func renderPanel(title string, width int, content string) string {
	body := content
	if strings.TrimSpace(title) != "" {
		body = panelTitleStyle.Render(title) + "\n" + content
	}
	s := panelStyle
	if width > 0 {
		s = s.Width(width)
	}
	return s.Render(body)
}

func renderSelector(prompt string, items []string, activeIdx int, focused bool) string {
	parts := make([]string, 0, len(items))
	for i, it := range items {
		if i == activeIdx {
			if focused {
				parts = append(parts, lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(it))
			} else {
				parts = append(parts, valueStyle.Render(it))
			}
		} else {
			parts = append(parts, faintStyle.Render(it))
		}
	}
	ps := lipgloss.NewStyle().Foreground(colorMuted)
	arrow := lipgloss.NewStyle().Foreground(colorMuted)
	if focused {
		ps = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
		return ps.Render(prompt) + arrow.Render("◄ ") + strings.Join(parts, "  ") + arrow.Render(" ►")
	}
	return ps.Render(prompt) + strings.Join(parts, "  ")
}

func renderStatusLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return okStyle.Render(s)
}

func renderErrorLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return errorStyle.Render("Error: " + s)
}

func renderInfoLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return infoStyle.Render(s)
}

func renderCompactSize(bytes, total int64) string {
	if bytes <= 0 && total <= 0 {
		return ""
	}
	if total > 0 {
		return fmt.Sprintf("%s / %s", formatBytes(bytes), formatBytes(total))
	}
	return formatBytes(bytes)
}

func formatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
}
