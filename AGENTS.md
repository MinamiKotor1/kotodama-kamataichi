# kotodama-kamataichi (Go TUI) - Agent Guide

This repository is a small Go CLI/TUI music downloader for Debian users.
It integrates with TuneHub V3 for parsing/downloading, and uses a "method config"
system for search that can call third-party upstream endpoints.

## Quick Start

- Entry point: `cmd/kotodama-kamataichi/main.go`
- Binary name (recommended): `kotodama-kamataichi`
- TUI: Bubble Tea (bubbles/list, progress, spinner, textinput)

## Build / Run

- Build:
  - `go build -o kotodama-kamataichi ./cmd/kotodama-kamataichi`
- Run:
  - `./kotodama-kamataichi -output downloads`
- Run without building (dev):
  - `go run ./cmd/kotodama-kamataichi -output downloads`

## Test / Lint

There are currently no unit tests; `go test` mainly compiles packages.

- All packages:
  - `go test ./...`
- Lint (standard):
  - `go vet ./...`

### Run a single test (when tests exist)

- One package:
  - `go test ./internal/tunehub -run '^TestName$' -count=1`
- Subtests:
  - `go test ./internal/tunehub -run '^TestName$/^subcase$' -count=1`
- Verbose:
  - `go test ./internal/tunehub -run '^TestName$' -count=1 -v`

### Helpful compile-only checks

- Compile everything quickly:
  - `go test ./... -run '^$'`
- Race (if relevant; slower):
  - `go test ./... -race`

## Formatting

- Always run `gofmt` on changed Go files.
- Prefer standard formatting; avoid stylistic churn.

Common commands:
- `gofmt -w ./cmd ./internal`
- `go fmt ./...` (package-based; still prefer gofmt for file-based edits)

## Repository Layout

- `cmd/kotodama-kamataichi/` - application entry point
- `internal/tui/` - Bubble Tea model, styles, list delegate
- `internal/tunehub/` - TuneHub client + provider-specific search
- `internal/download/` - downloader, file naming, disk layout
- `internal/jsbox/` - JS transform sandbox (subprocess + goja)
- `internal/template/` - safe-ish expression rendering for method configs

Local artifacts to ignore:
- `downloads/` (runtime output; do not treat as source)
- `kotodama-kamataichi` (built binary)

## Key Runtime Configuration

- `TUNEHUB_API_KEY` - API key used for `/v1/parse` (downloading)
- Proxy support is via Go's standard `http.ProxyFromEnvironment`:
  - `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, `ALL_PROXY`

## Networking Notes (Important)

- Search is NOT proxied by TuneHub by default.
  - The app calls `GET https://tunehub.sayqz.com/api/v1/methods/:platform/search`
  - Then it executes the returned upstream request locally (third-party domains).
- Netease host routing:
  - Some networks cannot reach `music.163.com:443`.
  - We rewrite `https://music.163.com/api/...` to `https://interface.music.163.com/api/...`.
  - See: `internal/tunehub/client.go` (`rewriteNeteaseAPIHost`).
- Keep timeouts and `io.LimitReader` usage in HTTP code.

## Code Style and Conventions

### Imports

- Use Go standard grouping (stdlib, blank line, third-party, blank line, local module).
- Avoid alias imports unless needed to disambiguate.

### Naming

- Follow standard Go naming:
  - exported: `CamelCase`
  - unexported: `camelCase`
  - acronyms: `HTTP`, `URL`, `ID`.
- Keep identifiers short but specific (avoid 1-letter names except loop indexes).

### Types and APIs

- Prefer explicit structs for JSON wire formats.
- Avoid `any` unless decoding unknown JSON payloads.
- Do not use unsafe casts or suppression hacks.

### Error handling

- Return errors early; no empty `catch`-style handling.
- Wrap errors with context using `%w` when it helps debugging.
- Keep user-facing errors concise (TUI shows errors directly).

### Context and timeouts

- All network operations must be cancellable via `context.Context`.
- Prefer timeouts at call sites (UI commands) plus sane client-level defaults.

### HTTP patterns

- Always:
  - set required headers from method config
  - validate HTTP status codes
  - `defer res.Body.Close()`
  - bound reads with `io.LimitReader`
- Keep transports explicit where needed (IPv4 forcing, HTTP/2 control).

### Security / Safety

- Treat any `transform` / method config from TuneHub as untrusted.
- Do not execute server-supplied JS in-process.
  - Use `internal/jsbox` sandbox (subprocess + goja).
- Do not log API keys or write them into repository files.

## TUI Guidelines

- Keep UI chrome ASCII-only (no Unicode box drawing/spinners).
  - Some terminals/font setups cannot render glyph spinners reliably.
- Use shared helpers in `internal/tui/styles.go`:
  - avoid composing multiple `Style.Render(...)` segments with backgrounds
    unless you explicitly fill gaps (ANSI resets can break continuous color).
- After state changes that affect layout, call `m.onResize()`.
- Prefer deterministic keybinds; avoid conflicts with list filtering (`/`).

## Downloader Guidelines

- Downloads use a `.part` file and rename on success.
- Write `meta.json` for each song; write `lyrics.lrc` and `cover.*` if present.
- Keep filenames safe via `internal/download/sanitize.go`.

## JS Sandbox Notes

- Sandbox runs as a subprocess invoked via `ExecPath ... js-sandbox`.
- Keep strict limits:
  - input/output size limits
  - short execution timeout
  - CPU limit (best-effort)
- Do not relax sandbox limits without a strong reason.

## Cursor / Copilot Rules

- No `.cursor/rules/`, `.cursorrules`, or `.github/copilot-instructions.md` found.
