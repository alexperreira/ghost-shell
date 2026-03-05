# ghost-shell

## Project Overview
A terminal emulator with built-in AI assistance. Users can run normal shell commands, but also ask natural language questions, get command suggestions, and receive explanations—all without leaving the terminal.

**Tagline:** "Ask questions, run commands, break things faster than ever before."

## Tech Stack
- **Backend/Core:** Go (handles terminal emulation, process management, LLM API calls)
- **Frontend:** React (for a modern, cross-platform terminal UI via Electron or Tauri)
- **LLM Integration:** Claude API (or OpenAI-compatible endpoint)

## MVP Scope (4-6 weeks)

### Core Features (Must Have)
1. **Basic Terminal Emulation**
   - PTY (pseudo-terminal) spawning and management
   - Standard input/output handling
   - Support for common shells (bash, zsh, fish)
   - ANSI color and cursor control support

2. **AI Chat Mode**
   - Trigger with a prefix (e.g., `?` or `ai:`) to send input to LLM instead of shell
   - Example: `? how do I find large files in this directory` → returns a command suggestion
   - Display AI responses inline, visually distinct from shell output

3. **Command Suggestion & Execution**
   - AI suggests commands based on natural language
   - User can accept (press Enter/Tab) to execute, or dismiss
   - Optional: "explain this command" mode before running

4. **Context Awareness (Basic)**
   - Send current working directory to LLM for context
   - Optionally include last N commands or recent output (truncated) for smarter suggestions

### Non-Goals for MVP
- No multi-pane/tmux-style splitting
- No plugin system
- No themes or heavy customization
- No local/on-device LLM (API-only for now)
- No Windows-native binary (targets WSL2/Linux; frontend opens in Windows browser via localhost)

## Architecture Sketch
```
┌─────────────────────────────────────────┐
│              React Frontend              │
│  (Terminal UI, input handling, display)  │
└─────────────────┬───────────────────────┘
                  │ WebSocket / IPC
┌─────────────────▼───────────────────────┐
│               Go Backend                 │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │ PTY Manager │  │  LLM Client      │  │
│  │ (shell I/O) │  │  (Claude API)    │  │
│  └─────────────┘  └──────────────────┘  │
└─────────────────────────────────────────┘
```

## Key Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Terminal rendering | xterm.js in React | Battle-tested, handles ANSI codes well |
| Go PTY library | `creack/pty` | Standard choice for Go terminal apps |
| Distribution | Single Go binary (embed frontend via `go:embed`) | No Tauri/Electron needed for WSL2; user runs binary in WSL, opens browser |
| LLM API | Anthropic Claude | Strong coding assistance, good context handling |
| IPC | WebSocket | Simple bidirectional communication |

## File Structure (Suggested)
```
ghost-shell/
├── cmd/
│   └── ghost-shell/
│       └── main.go          # Entry point
├── internal/
│   ├── pty/
│   │   └── pty.go           # PTY management
│   ├── llm/
│   │   └── client.go        # LLM API client
│   └── server/
│       └── ws.go            # WebSocket server
├── ui/
│   ├── src/
│   │   ├── App.tsx
│   │   ├── Terminal.tsx     # xterm.js wrapper
│   │   └── AIResponse.tsx   # AI output display
│   ├── package.json
│   └── index.html
├── go.mod
├── go.sum
└── README.md
```

## Milestones

### Week 1-2: Foundation
- [x] Go backend with PTY spawning
- [x] WebSocket server for frontend communication
- [x] Basic React app with xterm.js rendering shell output
- [x] Bidirectional I/O working (type command → see output)

### Week 3: AI Integration
- [x] LLM client in Go (Claude API)
- [x] Detect AI trigger prefix in input
- [x] Route AI queries to LLM, display response in terminal
- [x] Basic prompt engineering for command suggestions

### Week 4: Polish & UX
- [x] Visual distinction for AI responses (color, prefix icon)
- [x] Accept/dismiss flow for suggested commands
- [x] Context injection (CWD, recent commands)
- [x] Error handling and loading states

### Week 5-6: Packaging & Testing
- [x] Embed frontend with `go:embed` into a single WSL binary
- [x] Basic install flow (`make build` → `bin/ghost-shell`)
- [ ] README, demo GIF, release

## Open Questions (Decide During Build)
1. Should AI responses stream token-by-token or arrive all at once?
2. How much shell history context is too much? (token limits)
3. Should there be a "safety" confirmation for destructive commands (`rm -rf`, etc.)?

## Example Interactions
```bash
# Normal command
$ ls -la
(normal output)

# AI query
$ ? how do I find files larger than 100MB
┌─ ghost ─────────────────────────────────────────┐
│ Try: find . -type f -size +100M                 │
│ [Enter] Run  [Tab] Explain  [Esc] Dismiss       │
└─────────────────────────────────────────────────┘

# User presses Tab
┌─ ghost ─────────────────────────────────────────┐
│ This finds all files (-type f) in the current   │
│ directory and subdirectories that are larger    │
│ than 100 megabytes (-size +100M).               │
│ [Enter] Run  [Esc] Dismiss                      │
└─────────────────────────────────────────────────┘
```

## Dependencies

### Go
- `github.com/creack/pty` — PTY management
- `github.com/gorilla/websocket` — WebSocket server
- `github.com/anthropics/anthropic-sdk-go` — Claude API (or raw HTTP)

### Frontend
- `xterm` + `xterm-addon-fit` — Terminal emulation
- `react`, `typescript`
- `tauri` or `electron` — Desktop packaging

## Getting Started (For Claude Code)

1. Initialize Go module: `go mod init github.com/yourname/ghost-shell`
2. Scaffold the directory structure above
3. Start with PTY + WebSocket (get shell I/O working first)
4. Add React frontend with xterm.js
5. Layer in LLM integration last

Focus on working software over perfect architecture. Ship the ugly version first.

## Platform Strategy

- Run ghost-shell entirely inside WSL2 (Go backend + PTY run as Linux processes)
- `creack/pty` works normally — no Windows-native PTY workarounds needed
- Frontend served at `http://localhost:8080`; open in any Windows browser
- Distribution: `go build` produces a single binary; `go:embed` bundles the built React app
- Windows-native packaging (Tauri/Electron wrapping WSL) is out of scope for MVP
