# Implementation Plan: Core Features 2–4 — AI Integration

Covers CF2 (AI Chat Mode), CF3 (Command Suggestion & Execution), CF4 (Context Awareness).
Combined because CF3 and CF4 are the accept/dismiss UX and prompt enrichment layers on top
of CF2's LLM pipeline — splitting would mean wiring the API twice.

## What this phase ships
- `?`-prefixed input is routed to Claude instead of the shell, streamed token-by-token
- AI responses render inline, visually distinct (cyan, ghost-prefixed)
- If the AI suggests a runnable command, the user can accept (Enter) or dismiss (Esc)
- Context injected into every AI prompt: current working directory + last 10 shell commands

## What this phase does NOT include
- Persistent conversation history across sessions
- "explain this command" Tab flow (polish phase)
- Destructive command safety confirmation (polish phase)
- Any UI packaging (still browser-only)

---

## Design Decision: Ghost Input Mode

**Problem:** The frontend currently forwards each keystroke directly to the PTY, which echoes
them back. If the server intercepted `?`-lines at newline-time, the user would have already
seen the query echoed as apparent shell input — visual noise, and the shell prompt would be
in a dirty state.

**Solution: Frontend-side ghost input mode.**

When the user types `?` as the very first character of a new line:
1. The frontend enters **ghost input mode** — keystrokes stop being forwarded to the PTY.
2. The frontend renders the user's typing itself (writes to xterm directly with a `ghost> ` prefix).
3. Backspace, left/right arrows work locally within the buffer.
4. **Enter** → frontend sends `{type: "ai_query", data: "<query without ?>"}` to backend, clears ghost buffer.
5. **Esc** → cancels ghost mode silently, returns to normal PTY forwarding.

The PTY never sees any ghost input. No echo artifacts. The shell prompt stays clean.

**Tracking "start of line":** The frontend tracks `atLineStart: boolean`, set to `true` after
each `\r` received from the PTY, and `false` on the first keystroke forwarded. Ghost mode is
only entered when `atLineStart` is true and the key pressed is `?`.

---

## Architecture Changes

### New: `internal/llm/client.go`

Claude streaming client. API key read from `ANTHROPIC_API_KEY` env var.

```go
type Client struct{ apiKey string }

func New() (*Client, error)  // errors if ANTHROPIC_API_KEY is unset

// Stream sends the query to Claude and writes tokens to out.
// Returns when streaming is complete or ctx is cancelled.
func (c *Client) Stream(ctx context.Context, req Request, out chan<- string) error

type Request struct {
    Query   string
    CWD     string
    History []string // last N shell commands, oldest first
}
```

**System prompt** (static, baked in):

```
You are ghost, an AI assistant built into a terminal emulator.
The user is working in a shell and asking for help with commands and shell tasks.

Rules:
- Be concise. Prefer showing a command over explaining one.
- If the answer is a single runnable command, put it alone on the last line
  inside a fenced code block with no language tag.
- If multiple commands are needed, use a numbered list, then put the full
  pipeline on the last line in a fenced code block.
- Never suggest destructive commands (rm -rf, mkfs, dd) without a warning.
- Do not repeat the user's question back to them.
```

**User prompt template:**

```
Working directory: {CWD}
Recent commands:
{history[0]}
{history[1]}
...

User: {query}
```

**Command extraction heuristic** (pure function, easy to test):
1. Find the last fenced code block (` ``` ... ``` `) in the full response.
2. If found and the block contains exactly one non-empty line → that is the suggested command.
3. If no fenced block found and the entire response is a single non-empty line → treat as command.
4. Otherwise → no command extracted; `ai_done` sent with empty `command` field.

### Modified: `internal/pty/pty.go`

Store `cmd *exec.Cmd` on `Manager` to access the shell PID.
Add `CWD() (string, error)` reading `/proc/{pid}/cwd` via `os.Readlink`.

```go
type Manager struct {
    ptmx *os.File
    cmd  *exec.Cmd  // added
}

func (m *Manager) CWD() (string, error) {
    return os.Readlink(fmt.Sprintf("/proc/%d/cwd", m.cmd.Process.Pid))
}
```

### Modified: `internal/server/ws.go`

Add a `session` struct per WebSocket connection:

```go
type session struct {
    pty        *pty.Manager
    llm        *llm.Client
    history    []string      // capped at 10 entries
    cancelAI   context.CancelFunc // non-nil while a stream is in flight
}
```

**Input routing** (on `type: "ai_query"`):
1. If a stream is already in flight, call `cancelAI()` before starting a new one.
2. Call `mgr.CWD()` to get current directory.
3. Build `llm.Request{Query, CWD, History}`.
4. Start goroutine: call `client.Stream(ctx, req, tokenCh)`, forward each token as `ai_chunk`.
5. After stream ends: run command extraction on full accumulated response, send `ai_done{command}`.

**History append:** on `type: "input"` (normal PTY input), buffer lines server-side.
When a `\r` or `\n` is seen, flush the buffered line to history (trimmed, skip blanks,
skip lines starting with `?`). Truncate each entry to 200 chars. Cap slice at 10.

**`run_command` handler:**
1. Write `data + "\n"` to PTY.
2. Append `data` to history.

**Concurrent stream safety:** `cancelAI` and the token goroutine are only touched from
the single WS read loop goroutine — no mutex needed. The PTY write goroutine only writes
to the WebSocket; it never touches session state.

### Modified: `ui/src/Terminal.tsx`

**New state:**
```ts
type GhostMode = { active: false } | { active: true; buffer: string };
type PendingCmd = { command: string } | null;
```

**Ghost input mode logic** (in `term.onData`):
```
if ghostMode.active:
    handle backspace, printable chars locally (write to xterm, update buffer)
    on \r → send {type:"ai_query", data: buffer}, exit ghost mode
    on \x1b (Esc) → clear ghost display, exit ghost mode silently
else:
    if atLineStart && data === '?':
        enter ghost mode, write "ghost> " to xterm in cyan
    else:
        forward to WS as normal {type:"input"}
        update atLineStart (true if data contains \r)
```

**WS message handling:**
- `ai_chunk`: write `data` to xterm in cyan (`\x1b[36m...\x1b[0m`)
- `ai_done`:
  - Write `\r\n` to end the streamed response
  - If `command` non-empty: write hint line in dim white, set `pendingCmd`
    ```
    \x1b[2m[Enter] Run: command   [Esc] Dismiss\x1b[0m
    ```
  - If `command` empty: nothing extra
- `ai_error`: write `\x1b[31m[ghost error] data\x1b[0m\r\n`

**Pending command intercept** (in `term.onData`, checked before ghost mode):
```
if pendingCmd:
    on \r → send {type:"run_command", data: pendingCmd.command}, clear pendingCmd
    on \x1b → write "[dismissed]\r\n" in dim, clear pendingCmd
    swallow all other keys
```

**ANSI color reference used:**
- `\x1b[36m` cyan (AI output)
- `\x1b[2m` dim (hint line)
- `\x1b[31m` red (errors)
- `\x1b[0m` reset

---

## Steps

- [x] **Step 1 — LLM client (`internal/llm/client.go`)**
  - `go get github.com/anthropics/anthropic-sdk-go`
  - `New()`, `Stream()`, `Request` struct
  - System prompt and user prompt template as specified above
  - `extractCommand(response string) string` as a pure function (testable in isolation)

- [x] **Step 2 — PTY CWD (`internal/pty/pty.go`)**
  - Store `cmd` on Manager
  - Add `CWD()` via `os.Readlink("/proc/{pid}/cwd")`

- [x] **Step 3 — Session + routing (`internal/server/ws.go`)**
  - `session` struct with history buffer, `cancelAI`
  - Server-side line buffer for history tracking on normal input
  - Route `ai_query` → `llm.Stream` → stream `ai_chunk` → send `ai_done`
  - Handle `run_command` → PTY write + history append
  - Concurrent stream cancellation via `cancelAI`

- [x] **Step 4 — Frontend ghost mode + AI rendering (`ui/src/Terminal.tsx`)**
  - `atLineStart` tracking
  - Ghost input mode state machine
  - `pendingCmd` intercept
  - `ai_chunk` / `ai_done` / `ai_error` handlers with ANSI styling

- [x] **Step 5 — Smoke test**
  - `? how do I find files larger than 100MB` → streams → command suggested → Enter runs it
  - `? what is my current directory` → answer uses injected CWD
  - Type `? ...` then Esc → cancels cleanly, shell prompt unaffected
  - Fire two `?` queries rapidly → second cancels first, no duplicate output
  - Esc on pending command → dismissed, cursor returns to shell

- [x] **Step 6 — Cleanup**
  - Missing `ANTHROPIC_API_KEY`: `New()` returns error → `handleWS` writes
    `\x1b[31m[ghost] ANTHROPIC_API_KEY not set\x1b[0m\r\n` and disables AI routing
  - History truncation: 200 char/line cap, 10-line max enforced in `appendHistory()`
  - Token budget note: 10 lines × 200 chars ≈ 500 tokens of context, well within limits
  - Update README: add `ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/ghost-shell` example
  - `.env` file support via loader in `main.go`; `.env` gitignored; `.env.example` added

---

## Complete WS Protocol (after this phase)

### Client → Server
| `type` | Fields | When sent |
|---|---|---|
| `input` | `data: string` | Normal keystroke forwarded to PTY |
| `resize` | `rows, cols: number` | Terminal window resized |
| `ai_query` | `data: string` | User submitted a `?`-prefixed line |
| `run_command` | `data: string` | User accepted a suggested command |

### Server → Client
| `type` | Fields | When sent |
|---|---|---|
| `output` | `data: string` | PTY output chunk |
| `exit` | — | Shell process exited |
| `ai_chunk` | `data: string` | Streaming LLM token |
| `ai_done` | `command: string` | Stream complete; `command` may be empty |
| `ai_error` | `data: string` | LLM call failed |

---

## File Structure After This Phase
```
ghost-shell/
├── cmd/ghost-shell/main.go
├── internal/
│   ├── llm/
│   │   └── client.go          # NEW
│   ├── pty/
│   │   └── pty.go             # +cmd field, +CWD()
│   └── server/
│       └── ws.go              # +session, +ai_query routing, +run_command
├── ui/src/
│   ├── App.tsx
│   └── Terminal.tsx           # +ghost mode, +pending cmd, +ai message handlers
└── ...
```

## Key Decisions (all resolved)

| Decision | Choice | Rationale |
|---|---|---|
| Streaming | Yes, token-by-token | Better UX; Claude SDK supports it natively |
| `?` interception | Frontend ghost input mode | Avoids PTY echo artifacts; shell prompt stays clean |
| CWD source | `/proc/{pid}/cwd` symlink | No subprocess needed, always accurate |
| History tracking | Server-side, 10 lines × 200 chars | ~500 tokens; client-agnostic |
| Command extraction | Last fenced block, single-line fallback | Simple, testable, good enough for MVP |
| Concurrent streams | Cancel via `context.CancelFunc` | Clean; no mutex needed on single read loop |
| AI visual style | Inline ANSI cyan | Minimal, works in any terminal; box in polish phase |
| API key missing | Degrade gracefully, write error to terminal | Don't crash the shell session |
