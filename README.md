# ghost-shell

A terminal emulator with built-in AI assistance. Run shell commands, ask natural language questions, and get executable suggestions — without leaving the terminal.

> "Ask questions, run commands, break things faster than ever before."

<!-- demo GIF goes here -->

## Usage

| Input | Behavior |
|---|---|
| Any normal keystrokes | Forwarded to your shell |
| `?` at the start of a line | Enters ghost mode |
| `? <query>` + Enter | Asks the AI; streams response inline |
| Enter (on a suggestion) | Runs the suggested command |
| Esc (on a suggestion) | Dismisses it |
| Esc (in ghost mode) | Cancels the query |

**Example:**
```
$ ? how do I find files larger than 100MB
[ghost] thinking...
Use find to search by size:

find . -type f -size +100M

[Enter] Run: find . -type f -size +100M   [Esc] Dismiss
```

## Quickstart (WSL2)

**Requirements:** WSL2, Go 1.23+, Node 20+, pnpm, an [Anthropic API key](https://console.anthropic.com/)

```bash
git clone https://github.com/alexperreira/ghost-shell.git
cd ghost-shell

# Set your API key
echo "ANTHROPIC_API_KEY=sk-ant-..." > .env

# Build and run
make run
```

Then open **http://localhost:8080** in your Windows browser.

> The Go backend and PTY run inside WSL. The frontend is served as a static web app — no Electron or Tauri required.

## Development

Run the backend and frontend separately for hot-reload:

```bash
# Terminal 1 — Go backend
go run ./cmd/ghost-shell

# Terminal 2 — Vite dev server
cd ui && pnpm install && pnpm dev
```

Open **http://localhost:5173** (Vite proxies WebSocket to :8080 automatically via the browser's direct connection).

## Build

```bash
make build      # builds ui then Go binary → bin/ghost-shell
make run        # build + run
make clean      # remove bin/ and internal/web/static/
```

Requires Node 20+ in your `PATH` or via nvm. The Makefile defaults to `~/.nvm/versions/node/v22.22.0`.

## Tech Stack

| Layer | Choice |
|---|---|
| Backend | Go — `creack/pty`, `gorilla/websocket` |
| Frontend | React + TypeScript + Vite + `@xterm/xterm` |
| LLM | Claude API via `anthropic-sdk-go` |
| Packaging | `go:embed` — single self-contained binary |

## Project Structure

```
ghost-shell/
├── cmd/ghost-shell/main.go   # Entry point
├── internal/
│   ├── llm/client.go         # Claude streaming client + prompt builder
│   ├── pty/pty.go            # PTY lifecycle and CWD tracking
│   ├── server/ws.go          # WebSocket server, session routing, AI dispatch
│   └── web/embed.go          # Embeds built frontend via go:embed
├── ui/                       # React frontend
│   └── src/
│       ├── Terminal.tsx      # xterm.js, ghost input mode, AI rendering
│       └── App.tsx
├── Makefile
└── SCOPE.md
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `ANTHROPIC_API_KEY` | — | Required. Set in `.env` or environment. |
| `-addr` flag | `:8080` | HTTP/WebSocket listen address. |
