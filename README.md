# ghost-shell

A terminal emulator with built-in AI assistance. Run normal shell commands, ask natural language questions, get command suggestions — all without leaving the terminal.

> **Tagline:** "Ask questions, run commands, break things faster than ever before."

## Status

Core Feature 1 (basic terminal emulation) is working. AI integration coming next.

## Dev Quickstart

Requires Go 1.21+ and Node/pnpm.

**1. Start the Go backend:**
```bash
go run ./cmd/ghost-shell
# Listens on :8080 by default. Override with -addr :9090
```

**2. Start the frontend dev server (separate terminal):**
```bash
cd ui
pnpm install   # first time only
pnpm dev
```

**3. Open your browser:**
```
http://localhost:5173
```

You should see a full-screen terminal connected to your `$SHELL`.

## Tech Stack

| Layer | Choice |
|---|---|
| Backend | Go (`creack/pty`, `gorilla/websocket`) |
| Frontend | React + TypeScript + Vite |
| Terminal rendering | `@xterm/xterm` + `@xterm/addon-fit` |
| LLM (upcoming) | Claude API |
| Desktop packaging (upcoming) | Tauri |

## Project Structure

```
ghost-shell/
├── cmd/ghost-shell/main.go   # Entry point
├── internal/
│   ├── pty/pty.go            # PTY management
│   └── server/ws.go          # WebSocket server
├── ui/                       # React frontend
│   └── src/
│       ├── App.tsx
│       └── Terminal.tsx
├── docs/
│   └── BASIC_TERMINAL_EMULATION.md
└── SCOPE.md
```
