package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	tea "github.com/charmbracelet/bubbletea"
)

type resultDelegate struct {
	compact bool
}

func newResultDelegate() *resultDelegate {
	return &resultDelegate{}
}

func (d *resultDelegate) Height() int {
	if d.compact {
		return 1
	}
	return 2
}

func (d *resultDelegate) Spacing() int {
	if d.compact {
		return 0
	}
	return 1
}

func (d *resultDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d *resultDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(listItem)
	if !ok {
		return
	}
	if m.Width() <= 0 {
		return
	}

	title := strings.TrimSpace(it.Title())
	desc := strings.TrimSpace(it.Description())

	// Conditions
	isSelected := index == m.Index() && m.FilterState() != list.Filtering
	filteringEmpty := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied

	prefix := "  "
	prefixStyle := faintStyle
	titleStyle := valueStyle
	descStyle := faintStyle
	if filteringEmpty {
		titleStyle = faintStyle
		descStyle = faintStyle
	} else if isSelected {
		prefix = "▸ "
		prefixStyle = lipgloss.NewStyle().Foreground(colorMagenta).Bold(true)
		titleStyle = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(colorMuted)
	}

	textW := m.Width() - lipgloss.Width(prefix)
	if textW < 0 {
		textW = 0
	}

	if d.compact {
		line := title
		if strings.TrimSpace(it.Artist) != "" {
			line = fmt.Sprintf("%s - %s", title, strings.TrimSpace(it.Artist))
		}
		line = ansi.Truncate(line, textW, "...")
		if isFiltered {
			matched := m.MatchesForItem(index)
			unmatched := titleStyle.Inline(true)
			matchedStyle := unmatched.Inherit(lipgloss.NewStyle().Underline(true))
			line = lipgloss.StyleRunes(line, matched, matchedStyle, unmatched)
		}
		_, _ = fmt.Fprint(w, prefixStyle.Render(prefix)+titleStyle.Render(line))
		return
	}

	title = ansi.Truncate(title, textW, "...")
	desc = ansi.Truncate(desc, textW, "...")
	if isFiltered {
		matched := m.MatchesForItem(index)
		unmatched := titleStyle.Inline(true)
		matchedStyle := unmatched.Inherit(lipgloss.NewStyle().Underline(true))
		title = lipgloss.StyleRunes(title, matched, matchedStyle, unmatched)
	}

	line1 := prefixStyle.Render(prefix) + titleStyle.Render(title)
	line2 := prefixStyle.Render("  ") + descStyle.Render(desc)
	_, _ = fmt.Fprintf(w, "%s\n%s", line1, line2)
}
