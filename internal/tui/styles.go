package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAccent = lipgloss.Color("39")
	colorBg     = lipgloss.Color("24")
	colorFg     = lipgloss.Color("252")
	colorMuted  = lipgloss.Color("245")
	colorFaint  = lipgloss.Color("240")
	colorError  = lipgloss.Color("9")
	colorOK     = lipgloss.Color("10")
	colorWarn   = lipgloss.Color("208")
)

var asciiBorder = lipgloss.Border{
	Top:          "-",
	Bottom:       "-",
	Left:         "|",
	Right:        "|",
	TopLeft:      "+",
	TopRight:     "+",
	BottomLeft:   "+",
	BottomRight:  "+",
	MiddleLeft:   "+",
	MiddleRight:  "+",
	Middle:       "+",
	MiddleTop:    "+",
	MiddleBottom: "+",
}

var (
	headerFillStyle  = lipgloss.NewStyle().Background(colorBg)
	headerTitleStyle = headerFillStyle.Copy().Foreground(colorAccent).Bold(true)
	headerSubStyle   = headerFillStyle.Copy().Foreground(colorMuted)
	headerLabelStyle = headerFillStyle.Copy().Foreground(colorMuted)
	headerValueStyle = headerFillStyle.Copy().Foreground(lipgloss.Color("231")).Bold(true)

	footerBarStyle = lipgloss.NewStyle().Foreground(colorMuted)

	panelStyle      = lipgloss.NewStyle().Border(asciiBorder).BorderForeground(colorFaint).Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	chipActiveStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(colorAccent).Bold(true).Padding(0, 1)
	chipInactiveStyle = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)

	labelStyle = lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle = lipgloss.NewStyle().Foreground(colorFg)
	faintStyle = lipgloss.NewStyle().Foreground(colorFaint)
	infoStyle  = lipgloss.NewStyle().Foreground(colorWarn)
	errorStyle = lipgloss.NewStyle().Foreground(colorError)
	okStyle    = lipgloss.NewStyle().Foreground(colorOK)
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
	return faintStyle.Render(strings.Repeat("-", width))
}

func renderFooter(width int, s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	if width <= 0 {
		return footerBarStyle.Render(s)
	}
	return footerBarStyle.Width(width).Render(s)
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

func renderChips(label string, items []string, activeIdx int) string {
	parts := make([]string, 0, len(items))
	for i, it := range items {
		if i == activeIdx {
			parts = append(parts, chipActiveStyle.Render(it))
			continue
		}
		parts = append(parts, chipInactiveStyle.Render(it))
	}
	return labelStyle.Render(label+":") + " " + strings.Join(parts, " ")
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
