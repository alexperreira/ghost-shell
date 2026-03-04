# Implementation Plan: Core Feature 1 — Basic Terminal Emulation

## Goal
Get a working terminal: spawn a shell in a PTY, pipe I/O bidirectionally over WebSocket, and render it in a React frontend with full ANSI support.

## What this phase ships
- A Go backend that spawns a PTY shell and exposes it over WebSocket
- A React + xterm.js frontend that connects to the backend and renders the terminal
- Bidirectional I/O: keystrokes in → shell output out
- Terminal resize support (SIGWINCH)

## What this phase does NOT include
- LLM/AI anything
- Tauri/Electron packaging (browser-only for now, package later)
- Auth, multi-session, or persistence

---

## Steps

- [x] **Step 1 — Go module + directory scaffold**
  - `go mod init github.com/alexperreira/ghost-shell`
  - Create `cmd/ghost-shell/main.go`, `internal/pty/pty.go`, `internal/server/ws.go`

- [x] **Step 2 — PTY manager (`internal/pty/`)**
  - Use `creack/pty` to spawn a configurable shell (`$SHELL` or flag)
  - Expose `Start()`, `Write([]byte)`, `Read() chan []byte`, `Resize(rows, cols)`
  - Handle shell exit cleanly

- [x] **Step 3 — WebSocket server (`internal/server/`)**
  - Use `gorilla/websocket`
  - Single `/ws` endpoint
  - Protocol: JSON envelope distinguishing message types (`input`, `resize`, `output`)
  - Goroutines: one reading PTY→WS, one reading WS→PTY

- [x] **Step 4 — React + xterm.js frontend (`ui/`)**
  - `pnpm create vite ui -- --template react-ts`
  - Install `@xterm/xterm`, `@xterm/addon-fit` (using modern non-deprecated packages)
  - `Terminal.tsx`: mounts xterm, connects to `ws://localhost:PORT/ws`, wires up `onData` → WS send, WS message → `terminal.write()`
  - `FitAddon` for resize: send `{type:"resize", cols, rows}` on window resize

- [x] **Step 5 — Wire together + smoke test**
  - `main.go` starts PTY manager + WS server
  - Open browser, confirm bidirectional I/O with bash/zsh
  - Test: colors (`ls --color`), cursor control (`vim`, `htop`), shell switching

- [x] **Step 6 — Cleanup**
  - WS disconnect → `mgr.Close()` kills the shell; `done` channel drains the reader goroutine
  - `README` quickstart (`go run ./cmd/ghost-shell` + `pnpm dev` in `ui/`)

---

## File Structure After This Phase
```
ghost-shell/
├── cmd/ghost-shell/main.go
├── internal/
│   ├── pty/pty.go
│   └── server/ws.go
├── ui/
│   ├── src/
│   │   ├── App.tsx
│   │   └── Terminal.tsx
│   ├── package.json
│   └── index.html
├── go.mod
└── go.sum
```

## Key Decisions
| Decision | Choice | Why |
|---|---|---|
| WS protocol | JSON envelopes | Simple to extend for AI messages later |
| Frontend tooling | Vite + React TS | Fast dev loop, no Electron overhead yet |
| PTY library | `creack/pty` | Standard for Go |
| Resize | `FitAddon` → resize message | Prevents garbled output in vim/htop |
