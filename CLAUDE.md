# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go TUI music downloader for Debian Linux. Integrates with TuneHub V3 API to search and download music from multiple platforms (Netease, QQ, Kuwo). Built with Bubble Tea framework.

## Build & Run

```bash
# Build
go build -o kotodama-kamataichi ./cmd/kotodama-kamataichi

# Run
./kotodama-kamataichi -output downloads

# Dev run
go run ./cmd/kotodama-kamataichi -output downloads

# Format
go fmt ./...

# Vet
go vet ./...

# Test (compile check — no unit tests yet)
go test ./...

# Single test
go test ./internal/tunehub -run '^TestName$' -count=1
```

## Architecture

```
cmd/kotodama-kamataichi/main.go   Entry point; wires jsbox → tunehub → downloader → TUI
internal/
  tui/          Bubble Tea UI: three screens (search → results → downloading)
  tunehub/      TuneHub API client; search via server-provided method configs
  download/     Parallel audio/cover/metadata/lyrics download with .part temp files
  jsbox/        Subprocess-based JS sandbox (goja); executes untrusted transform functions
  audiotag/     Embeds metadata (title, artist, album, lyrics, cover art) into MP3 (ID3v2) and FLAC (Vorbis Comment) files post-download
  template/     Placeholder rendering ({{keyword}}, {{page}}) with restricted JS eval
```

### Data Flow

1. User enters keyword/platform/quality in TUI search screen
2. `tunehub.Client` fetches method config from `/v1/methods/{platform}/search`, renders URL/params via `template`, executes JS `transform` via `jsbox` subprocess
3. Results displayed in list screen; user selects a track
4. `tunehub.Client` calls `/v1/parse` with API key to get audio URL
5. `download.DownloadSong` fetches audio (`.part` → rename), cover, metadata JSON, lyrics LRC in parallel via `errgroup`

### Key Design Decisions

- **Search is NOT proxied** — the app calls third-party music APIs directly using method configs from TuneHub; only `/v1/parse` goes through TuneHub
- **Netease host rewriting** — `music.163.com/api/...` is rewritten to `interface.music.163.com/api/...` with a dedicated IPv4-only HTTP client (no HTTP/2)
- **QQ Music search** — hardcoded direct API call in `qq_search.go`, bypasses method config system
- **JS sandbox isolation** — transform functions run in a subprocess (re-invokes binary with `js-sandbox` flag), not in-process; strict limits: 64KB code, 256KB output, 250ms timeout, 2s CPU RLIMIT

## Conventions

- **Imports**: stdlib → blank line → third-party → blank line → local module
- **Acronyms**: all caps (`HTTP`, `URL`, `ID`)
- **Error handling**: return early, wrap with `%w`, show user-facing errors in TUI
- **HTTP**: always `defer res.Body.Close()`, validate 2xx status, bound reads with `io.LimitReader`
- **Timeouts**: search 20s, download 2min, parse 25s, JS sandbox 250ms, template eval 50ms
- **TUI**: ASCII-only borders (no Unicode spinners), call `m.onResize()` after layout changes, avoid composing multiple `Style.Render()` with backgrounds
- **Filenames**: sanitized to max 120 chars, illegal characters stripped
- **Comments/docs**: non-required comments forbidden; code self-documents
