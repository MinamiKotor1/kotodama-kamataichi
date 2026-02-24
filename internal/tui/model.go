package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"kotodama-kamataichi/internal/download"
	"kotodama-kamataichi/internal/tunehub"
)

type screen int

const (
	screenSearch screen = iota
	screenResults
	screenDownloading
	screenFilePicker
)

const (
	focusAPIKey = iota
	focusKeyword
	focusPlatform
	focusQuality
	focusOutput
	focusCount
)

type model struct {
	th *tunehub.Client
	dl *download.Downloader

	w int
	h int

	platforms []string
	qualities []string
	platIdx   int
	qualIdx   int

	apiKey     textinput.Model
	keyword    textinput.Model
	outDirInput textinput.Model

	list     list.Model
	delegate *resultDelegate
	picker   filepicker.Model
	spinner  spinner.Model
	progress progress.Model

	screen   screen
	focusIdx int
	loading  bool
	errMsg   string
	status   string

	dlCh    chan tea.Msg
	dlBytes int64
	dlTotal int64

	lastResult download.Result
}

type searchResultMsg struct {
	items []tunehub.SearchItem
	err   error
}

type downloadProgressMsg struct {
	kind  string
	bytes int64
	total int64
}

type downloadDoneMsg struct {
	res download.Result
	err error
}

type listItem tunehub.SearchItem

func (i listItem) Title() string {
	if strings.TrimSpace(i.Name) == "" {
		return i.ID
	}
	return i.Name
}

func (i listItem) Description() string {
	parts := []string{}
	if strings.TrimSpace(i.Artist) != "" {
		parts = append(parts, i.Artist)
	}
	if strings.TrimSpace(i.Album) != "" {
		parts = append(parts, i.Album)
	}
	return strings.Join(parts, " | ")
}

func (i listItem) FilterValue() string {
	return strings.TrimSpace(i.Name + " " + i.Artist + " " + i.Album + " " + i.ID)
}

func New(th *tunehub.Client, dl *download.Downloader, outDir string) tea.Model {
	api := textinput.New()
	api.Placeholder = "TUNEHUB API Key (th_...)"
	api.Prompt = "API Key:  "
	api.EchoMode = textinput.EchoPassword
	api.EchoCharacter = '*'
	api.SetValue(strings.TrimSpace(os.Getenv("TUNEHUB_API_KEY")))

	kw := textinput.New()
	kw.Placeholder = "Keyword"
	kw.Prompt = "Search:   "

	od := textinput.New()
	od.Placeholder = "Download directory"
	od.Prompt = "Output:   "
	od.SetValue(outDir)

	initFocus := focusAPIKey
	if strings.TrimSpace(api.Value()) != "" {
		initFocus = focusKeyword
		kw.Focus()
	} else {
		api.Focus()
	}

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(colorCyan)

	del := newResultDelegate()
	l := list.New(nil, del, 0, 0)
	// We render our own header/footer. Keep list internals lean.
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.FilterInput.Prompt = "Filter: "
	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(colorMagenta).Bold(true)
	l.FilterInput.TextStyle = valueStyle
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	// Avoid conflicting with our back key.
	l.KeyMap.PrevPage = key.NewBinding(
		key.WithKeys("left", "h", "pgup", "u"),
		key.WithHelp("left", "prev page"),
	)

	p := progress.New(
		progress.WithFillCharacters('█', '░'),
		progress.WithScaledGradient("#00FFFF", "#FF00FF"),
		progress.WithoutPercentage(),
	)
	p.EmptyColor = "#1A1A2E"

	fp := filepicker.New()
	if abs, err := filepath.Abs(outDir); err == nil {
		fp.CurrentDirectory = abs
	} else {
		fp.CurrentDirectory = outDir
	}
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.ShowHidden = false
	fp.AutoHeight = false
	fp.Cursor = "▸"
	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(colorMagenta).Bold(true)
	fp.Styles.Directory = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	fp.Styles.File = faintStyle
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Underline(true)
	fp.Styles.Symlink = lipgloss.NewStyle().Foreground(colorMagenta).Italic(true)
	fp.Styles.EmptyDirectory = faintStyle
	fp.Styles.DisabledFile = faintStyle
	fp.Styles.DisabledCursor = faintStyle
	fp.Styles.DisabledSelected = faintStyle
	fp.KeyMap.Back = key.NewBinding(key.WithKeys("h", "backspace", "left"), key.WithHelp("←", "back"))

	m := &model{
		th:          th,
		dl:          dl,
		w:           80,
		h:           24,
		platforms:   []string{"netease", "qq", "kuwo"},
		qualities:   []string{"320k", "128k", "flac", "flac24bit"},
		focusIdx:    initFocus,
		apiKey:      api,
		keyword:     kw,
		outDirInput: od,
		spinner:     sp,
		list:        l,
		delegate:    del,
		picker:      fp,
		progress:    p,
		screen:    screenSearch,
	}
	m.applyInputStyles()
	m.onResize()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
		m.onResize()
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenSearch:
		return m.updateSearch(msg)
	case screenResults:
		return m.updateResults(msg)
	case screenDownloading:
		return m.updateDownloading(msg)
	case screenFilePicker:
		return m.updateFilePicker(msg)
	default:
		return m, nil
	}
}

func (m *model) View() string {
	switch m.screen {
	case screenSearch:
		return m.viewSearch()
	case screenResults:
		return m.viewResults()
	case screenDownloading:
		return m.viewDownloading()
	case screenFilePicker:
		return m.viewFilePicker()
	default:
		return ""
	}
}

func (m *model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.loading {
		m.spinner, _ = m.spinner.Update(msg)
		switch msg := msg.(type) {
		case searchResultMsg:
			m.loading = false
			if msg.err != nil {
				m.errMsg = msg.err.Error()
				return m, nil
			}
			items := make([]list.Item, 0, len(msg.items))
			for _, it := range msg.items {
				items = append(items, listItem(it))
			}
			m.list.SetItems(items)
			m.errMsg = ""
			m.screen = screenResults
			m.onResize()
			return m, nil
		}
		return m, m.spinner.Tick
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		case "tab", "down":
			m.focusIdx = (m.focusIdx + 1) % focusCount
			m.syncFocus()
			return m, nil
		case "shift+tab", "up":
			m.focusIdx = (m.focusIdx + focusCount - 1) % focusCount
			m.syncFocus()
			return m, nil
		case "left":
			switch m.focusIdx {
			case focusPlatform:
				m.platIdx = (m.platIdx + len(m.platforms) - 1) % len(m.platforms)
			case focusQuality:
				m.qualIdx = (m.qualIdx + len(m.qualities) - 1) % len(m.qualities)
			}
		case "right":
			switch m.focusIdx {
			case focusPlatform:
				m.platIdx = (m.platIdx + 1) % len(m.platforms)
			case focusQuality:
				m.qualIdx = (m.qualIdx + 1) % len(m.qualities)
			}
		case "b":
			if m.focusIdx == focusOutput {
				dir := strings.TrimSpace(m.outDirInput.Value())
				if abs, err := filepath.Abs(dir); err == nil {
					m.picker.CurrentDirectory = abs
				} else {
					m.picker.CurrentDirectory = dir
				}
				m.screen = screenFilePicker
				m.onResize()
				return m, m.picker.Init()
			}
		case "enter":
			kw := strings.TrimSpace(m.keyword.Value())
			if kw == "" {
				m.errMsg = "Keyword required"
				return m, nil
			}
			m.errMsg = ""
			m.status = ""
			m.loading = true
			plat := m.platforms[m.platIdx]
			cmds = append(cmds, searchCmd(m.th, plat, kw))
			cmds = append(cmds, m.spinner.Tick)
		}
	}

	var cmd tea.Cmd
	m.apiKey, cmd = m.apiKey.Update(msg)
	cmds = append(cmds, cmd)
	m.keyword, cmd = m.keyword.Update(msg)
	cmds = append(cmds, cmd)
	m.outDirInput, cmd = m.outDirInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) updateResults(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "b":
			m.screen = screenSearch
			return m, nil
		case "esc":
			// Let list handle esc for filtering/clearing; only quit when unfiltered.
			if m.list.FilterState() == list.Unfiltered {
				return m, tea.Quit
			}
		case "enter":
			if m.list.FilterState() == list.Filtering {
				break
			}
			if len(m.list.Items()) == 0 {
				m.errMsg = "No results to download"
				return m, nil
			}
			it, ok := m.list.SelectedItem().(listItem)
			if !ok {
				return m, nil
			}
			apiKey := strings.TrimSpace(m.apiKey.Value())
			if apiKey == "" {
				m.errMsg = "API key required to download (press b to go back)"
				return m, nil
			}
			plat := m.platforms[m.platIdx]
			qual := m.qualities[m.qualIdx]
			m.startDownload(apiKey, plat, qual, tunehub.SearchItem(it))
			m.onResize()
			return m, listenMsg(m.dlCh)
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) updateDownloading(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case downloadProgressMsg:
		if msg.kind == "audio" {
			m.dlBytes = msg.bytes
			m.dlTotal = msg.total
		}
		return m, listenMsg(m.dlCh)
	case downloadDoneMsg:
		m.dlCh = nil
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.screen = screenResults
			m.onResize()
			return m, nil
		}
		m.lastResult = msg.res
		m.status = "Download complete: " + msg.res.Dir
		m.errMsg = ""
		m.screen = screenResults
		m.onResize()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		case "b":
			m.errMsg = "Cancel not implemented (download continues in background)"
			m.screen = screenResults
			m.onResize()
			return m, nil
		}
	}

	return m, tea.Batch(cmd, listenMsg(m.dlCh))
}

func (m *model) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "b":
			m.screen = screenSearch
			m.onResize()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)

	if didSelect, path := m.picker.DidSelectFile(msg); didSelect {
		m.outDirInput.SetValue(path)
		m.screen = screenSearch
		m.onResize()
		return m, nil
	}

	return m, cmd
}

func (m *model) viewFilePicker() string {
	padX, padY, w, h := m.layout()
	container := lipgloss.NewStyle().Padding(padY, padX)
	if w < 24 || h < 10 {
		return container.Render("Terminal too small. Press q to quit.")
	}

	left := headerTitleStyle.Render(">> kotodama-kamataichi") + headerSubStyle.Render(" // browse")
	right := headerFillStyle.Render("")

	pathLine := labelStyle.Render("  ") + valueStyle.Render(m.picker.CurrentDirectory)

	lines := []string{
		renderHeader(w, left, right),
		renderDivider(w),
		pathLine,
		renderPanel("", w, m.picker.View()),
		renderFooterKeys(w, "Enter", "select", "b", "back", "↑↓", "navigate", "→/←", "open/parent"),
	}
	return container.Render(strings.Join(filterEmpty(lines), "\n"))
}

func (m *model) startDownload(apiKey, platform, quality string, it tunehub.SearchItem) {
	m.screen = screenDownloading
	m.loading = true
	m.errMsg = ""
	m.status = fmt.Sprintf("Parse and download: %s - %s", it.Name, it.Artist)
	m.dlBytes = 0
	m.dlTotal = 0
	m.onResize()

	ch := make(chan tea.Msg, 128)
	m.dlCh = ch

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		pd, err := m.th.Parse(ctx, apiKey, platform, it.ID, quality)
		if err != nil {
			ch <- downloadDoneMsg{err: err}
			close(ch)
			return
		}
		if len(pd.Data) == 0 {
			ch <- downloadDoneMsg{err: fmt.Errorf("parse returned empty data")}
			close(ch)
			return
		}
		pi := pd.Data[0]
		if !pi.Success {
			errText := strings.TrimSpace(pi.Error)
			if errText == "" {
				errText = "parse failed"
			}
			ch <- downloadDoneMsg{err: fmt.Errorf("%s", errText)}
			close(ch)
			return
		}

		res, err := m.dl.DownloadSong(ctx, m.outDirInput.Value(), pi, func(p download.Progress) {
			select {
			case ch <- downloadProgressMsg{kind: p.Kind, bytes: p.Bytes, total: p.Total}:
			default:
			}
		})
		ch <- downloadDoneMsg{res: res, err: err}
		close(ch)
	}()
}

func listenMsg(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func searchCmd(th *tunehub.Client, platform, keyword string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		items, err := th.Search(ctx, platform, keyword, 1, 20)
		return searchResultMsg{items: items, err: err}
	}
}

func (m *model) viewSearch() string {
	padX, padY, w, h := m.layout()
	container := lipgloss.NewStyle().Padding(padY, padX)
	if w < 24 || h < 10 {
		return container.Render("Terminal too small. Press q to quit.")
	}

	left := headerTitleStyle.Render(">> kotodama-kamataichi") + headerSubStyle.Render(" // search")
	right := headerLabelStyle.Render("P:") + headerFillStyle.Render(" ") + headerValueStyle.Render(m.platforms[m.platIdx]) + headerFillStyle.Render("  ") + headerLabelStyle.Render("Q:") + headerFillStyle.Render(" ") + headerValueStyle.Render(m.qualities[m.qualIdx])

	form := strings.Join([]string{
		m.apiKey.View(),
		m.keyword.View(),
		renderSelector("Platform: ", m.platforms, m.platIdx, m.focusIdx == focusPlatform),
		renderSelector("Quality:  ", m.qualities, m.qualIdx, m.focusIdx == focusQuality),
		m.outDirInput.View(),
	}, "\n\n")

	lines := []string{
		renderHeader(w, left, right),
		renderDivider(w),
		renderPanel("Search", w, form),
	}
	if m.loading {
		lines = append(lines, renderInfoLine(m.spinner.View()+" Searching..."))
	}
	if m.errMsg != "" {
		lines = append(lines, renderErrorLine(m.errMsg))
	}
	if m.focusIdx == focusOutput {
		lines = append(lines, renderFooterKeys(w, "b", "browse", "↑↓", "cycle", "Esc", "quit"))
	} else {
		lines = append(lines, renderFooterKeys(w, "Enter", "search", "↑↓", "cycle", "←→", "switch", "Esc", "quit"))
	}

	return container.Render(strings.Join(filterEmpty(lines), "\n"))
}

func (m *model) viewResults() string {
	padX, padY, w, h := m.layout()
	container := lipgloss.NewStyle().Padding(padY, padX)
	if w < 24 || h < 10 {
		return container.Render("Terminal too small. Press q to quit.")
	}

	left := headerTitleStyle.Render(">> kotodama-kamataichi") + headerSubStyle.Render(" // results")
	right := headerLabelStyle.Render("P:") + headerFillStyle.Render(" ") + headerValueStyle.Render(m.platforms[m.platIdx]) + headerFillStyle.Render("  ") + headerLabelStyle.Render("Q:") + headerFillStyle.Render(" ") + headerValueStyle.Render(m.qualities[m.qualIdx])

	listView := m.list.View()
	if len(m.list.Items()) == 0 {
		listView = faintStyle.Render("No results found.")
	}

	lines := []string{
		renderHeader(w, left, right),
		renderDivider(w),
		renderPanel("", w, listView),
	}
	if m.status != "" {
		lines = append(lines, renderStatusLine(m.status))
	}
	if m.errMsg != "" {
		lines = append(lines, renderErrorLine(m.errMsg))
	}
	lines = append(lines, renderFooterKeys(w, "Enter", "download", "/", "filter", "b", "back", "Esc", "quit"))

	return container.Render(strings.Join(filterEmpty(lines), "\n"))
}

func (m *model) viewDownloading() string {
	padX, padY, w, h := m.layout()
	container := lipgloss.NewStyle().Padding(padY, padX)
	if w < 24 || h < 10 {
		return container.Render("Terminal too small. Press q to quit.")
	}

	left := headerTitleStyle.Render(">> kotodama-kamataichi") + headerSubStyle.Render(" // downloading")
	right := headerLabelStyle.Render("P:") + headerFillStyle.Render(" ") + headerValueStyle.Render(m.platforms[m.platIdx]) + headerFillStyle.Render("  ") + headerLabelStyle.Render("Q:") + headerFillStyle.Render(" ") + headerValueStyle.Render(m.qualities[m.qualIdx])

	pct := percent(m.dlBytes, m.dlTotal)
	bar := m.progress.ViewAs(pct)
	pctStyle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	progLine := bar + " " + pctStyle.Render(fmt.Sprintf("%3.0f%%", pct*100))

	bytesLine := renderCompactSize(m.dlBytes, m.dlTotal)
	if bytesLine != "" {
		bytesLine = faintStyle.Render(bytesLine)
	}

	panel := renderPanel("", w, strings.Join(filterEmpty([]string{
		renderInfoLine(m.spinner.View() + " " + m.status),
		progLine,
		bytesLine,
	}), "\n"))

	lines := []string{
		renderHeader(w, left, right),
		renderDivider(w),
		panel,
		renderFooterKeys(w, "b", "back", "Esc", "quit"),
	}
	return container.Render(strings.Join(filterEmpty(lines), "\n"))
}

func (m *model) syncFocus() {
	m.apiKey.Blur()
	m.keyword.Blur()
	m.outDirInput.Blur()
	switch m.focusIdx {
	case focusAPIKey:
		m.apiKey.Focus()
	case focusKeyword:
		m.keyword.Focus()
	case focusOutput:
		m.outDirInput.Focus()
	}
	m.applyInputStyles()
}

func (m *model) applyInputStyles() {
	focused := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	blurred := lipgloss.NewStyle().Foreground(colorMuted)

	if m.focusIdx == focusAPIKey {
		m.apiKey.PromptStyle = focused
	} else {
		m.apiKey.PromptStyle = blurred
	}
	m.apiKey.TextStyle = valueStyle
	m.apiKey.PlaceholderStyle = faintStyle

	if m.focusIdx == focusKeyword {
		m.keyword.PromptStyle = focused
	} else {
		m.keyword.PromptStyle = blurred
	}
	m.keyword.TextStyle = valueStyle
	m.keyword.PlaceholderStyle = faintStyle

	if m.focusIdx == focusOutput {
		m.outDirInput.PromptStyle = focused
	} else {
		m.outDirInput.PromptStyle = blurred
	}
	m.outDirInput.TextStyle = valueStyle
	m.outDirInput.PlaceholderStyle = faintStyle
}

func (m *model) layout() (padX, padY, contentW, contentH int) {
	w := m.w
	h := m.h
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	padX = 2
	padY = 1
	if w < 70 {
		padX = 1
	}
	if h < 18 {
		padY = 0
	}

	contentW = w - padX*2
	contentH = h - padY*2
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		contentH = 0
	}
	return padX, padY, contentW, contentH
}

func (m *model) onResize() {
	_, _, contentW, contentH := m.layout()

	if contentW > 0 {
		inputW := contentW - 10
		if inputW < 20 {
			inputW = 20
		}
		m.apiKey.Width = inputW
		m.keyword.Width = inputW
		m.outDirInput.Width = inputW
		// Reserve space for "[" "]" + percent.
		m.progress.Width = max(10, contentW-12)
	}

	if m.delegate != nil {
		m.delegate.compact = contentH < 18
	}

	listW := contentW - 4
	if listW < 0 {
		listW = 0
	}
	listH := 0
	if contentH > 0 {
		fixed := 1 + 1 + 1 // header + divider + footer
		if m.status != "" {
			fixed++
		}
		if m.errMsg != "" {
			fixed++
		}
		if m.loading && m.screen == screenSearch {
			fixed++
		}
		listH = contentH - fixed - 2 // panel borders
		if listH < 3 {
			listH = 3
		}
	}

	m.list.SetSize(listW, listH)

	// filepicker height: header + divider + pathLine + panel borders + footer = 6 lines
	pickerH := contentH - 6
	if pickerH < 3 {
		pickerH = 3
	}
	m.picker.SetHeight(pickerH)
}

func filterEmpty(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, s := range lines {
		if strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func percent(bytes, total int64) float64 {
	if total <= 0 {
		return 0
	}
	if bytes <= 0 {
		return 0
	}
	p := float64(bytes) / float64(total)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}
