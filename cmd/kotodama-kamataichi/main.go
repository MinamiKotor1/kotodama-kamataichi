package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kotodama-kamataichi/internal/download"
	"kotodama-kamataichi/internal/jsbox"
	"kotodama-kamataichi/internal/tui"
	"kotodama-kamataichi/internal/tunehub"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "js-sandbox" {
		os.Exit(jsbox.RunSandbox())
	}

	outDir := flag.String("output", "downloads", "download output directory")
	flag.Parse()

	jsr, err := jsbox.NewRunner()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	th := tunehub.New(jsr)
	dl := download.NewDownloader()

	m := tui.New(th, dl, *outDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
