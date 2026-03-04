# ghost-shell

A terminal emulator with built-in AI assistance. Run normal shell commands, ask natural language questions, get command suggestions — all without leaving the terminal.

> **Tagline:** "Ask questions, run commands, break things faster than ever before."

## Status

Core terminal emulation and AI integration are both working.

- Type normally to use the shell as usual.
- Prefix any input with `?` to ask the AI a question.
- If the AI suggests a command, press `Enter` to run it or `Esc` to dismiss.

## Dev Quickstart

Requires Go 1.22+ and Node/pnpm.

**1. Set your Anthropic API key:**

Copy `.env.example` to `.env` and fill in your key:
```bash
cp .env.example .env
# then edit .env:
# ANTHROPIC_API_KEY=sk-ant-...
```

Alternatively, pass it inline: `ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/ghost-shell`

**2. Start the Go backend:**
```bash
go run ./cmd/ghost-shell
# Listens on :8080 by default. Override with -addr :9090
```

**3. Start the frontend dev server (separate terminal):**
```bash
cd ui
pnpm install   # first time only
pnpm dev
```

**4. Open your browser:**
```
http://localhost:5173
```

You should see a full-screen terminal connected to your `$SHELL`. Type `? how do I find files larger than 100MB` to try the AI.

## Tech Stack

| Layer | Choice |
|---|---|
| Backend | Go (`creack/pty`, `gorilla/websocket`) |
| Frontend | React + TypeScript + Vite |
| Terminal rendering | `@xterm/xterm` + `@xterm/addon-fit` |
| LLM | Claude API (`anthropic-sdk-go`) |
| Desktop packaging (upcoming) | Tauri |

## Project Structure

```
ghost-shell/
├── cmd/ghost-shell/main.go   # Entry point (.env loader, flags)
├── internal/
│   ├── llm/client.go         # Anthropic streaming client
│   ├── pty/pty.go            # PTY management
│   └── server/ws.go          # WebSocket server + AI routing
├── ui/                       # React frontend
│   └── src/
│       ├── App.tsx
│       └── Terminal.tsx      # Ghost input mode + AI rendering
├── docs/
│   ├── BASIC_TERMINAL_EMULATION.md
│   └── AI_INTEGRATION.md
├── .env.example
└── SCOPE.md
```
