package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/alexperreira/ghost-shell/internal/llm"
	"github.com/alexperreira/ghost-shell/internal/pty"
	"github.com/gorilla/websocket"
)

// ClientMessage is a message sent from the browser to the backend.
type ClientMessage struct {
	Type string `json:"type"` // "input" | "resize" | "ai_query" | "run_command"
	Data string `json:"data"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// ServerMessage is a message sent from the backend to the browser.
type ServerMessage struct {
	Type    string `json:"type"`              // "output" | "exit" | "ai_chunk" | "ai_done" | "ai_error"
	Data    string `json:"data,omitempty"`    // output text or error message
	Command string `json:"command,omitempty"` // suggested command (ai_done only)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type session struct {
	mgr      *pty.Manager
	llmCli   *llm.Client // nil if API key missing
	mu       sync.Mutex  // guards WebSocket writes
	history  []string    // capped at 10 entries, 200 chars each
	lineBuf  string      // accumulates keystrokes until \r/\n
	cancelAI context.CancelFunc
}

func (s *session) writeWS(conn *websocket.Conn, msg ServerMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn.WriteJSON(msg) //nolint:errcheck
}

func (s *session) appendHistory(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "?") {
		return
	}
	if len(line) > 200 {
		line = line[:200]
	}
	s.history = append(s.history, line)
	if len(s.history) > 10 {
		s.history = s.history[len(s.history)-10:]
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	mgr, err := pty.Start(shell)
	if err != nil {
		log.Println("pty start:", err)
		return
	}
	defer mgr.Close()

	llmCli, llmErr := llm.New()
	sess := &session{mgr: mgr, llmCli: llmCli}

	if llmErr != nil {
		sess.writeWS(conn, ServerMessage{
			Type: "output",
			Data: "\x1b[31m[ghost] ANTHROPIC_API_KEY not set — AI features disabled\x1b[0m\r\n",
		})
	}

	done := make(chan struct{})

	// PTY → WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := mgr.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Println("pty read:", err)
				}
				sess.writeWS(conn, ServerMessage{Type: "exit"})
				conn.Close()
				return
			}
			sess.writeWS(conn, ServerMessage{Type: "output", Data: string(buf[:n])})
		}
	}()

	// WebSocket → PTY / session
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			mgr.Close()
			<-done
			return
		}
		var msg ClientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "input":
			mgr.Write([]byte(msg.Data)) //nolint:errcheck
			// Accumulate into lineBuf; flush to history on newline.
			for _, ch := range msg.Data {
				if ch == '\r' || ch == '\n' {
					sess.appendHistory(sess.lineBuf)
					sess.lineBuf = ""
				} else {
					sess.lineBuf += string(ch)
				}
			}

		case "resize":
			if msg.Rows > 0 && msg.Cols > 0 {
				mgr.Resize(msg.Rows, msg.Cols) //nolint:errcheck
			}

		case "ai_query":
			if sess.llmCli == nil {
				sess.writeWS(conn, ServerMessage{Type: "ai_error", Data: "ANTHROPIC_API_KEY not set"})
				continue
			}
			// Cancel any in-flight stream.
			if sess.cancelAI != nil {
				sess.cancelAI()
			}
			ctx, cancel := context.WithCancel(context.Background())
			sess.cancelAI = cancel

			cwd, _ := mgr.CWD()
			req := llm.Request{
				Query:   msg.Data,
				CWD:     cwd,
				History: append([]string(nil), sess.history...),
			}

			go func() {
				tokenCh := make(chan string, 64)
				errCh := make(chan error, 1)

				go func() {
					errCh <- sess.llmCli.Stream(ctx, req, tokenCh)
					close(tokenCh)
				}()

				var full strings.Builder
				for token := range tokenCh {
					full.WriteString(token)
					sess.writeWS(conn, ServerMessage{Type: "ai_chunk", Data: token})
				}

				streamErr := <-errCh
				if streamErr != nil && ctx.Err() == nil {
					sess.writeWS(conn, ServerMessage{Type: "ai_error", Data: streamErr.Error()})
					return
				}
				if ctx.Err() != nil {
					return // cancelled — do not send ai_done
				}
				cmd := llm.ExtractCommand(full.String())
				sess.writeWS(conn, ServerMessage{Type: "ai_done", Command: cmd})
			}()

		case "run_command":
			mgr.Write([]byte(msg.Data + "\n")) //nolint:errcheck
			sess.appendHistory(msg.Data)
		}
	}
}

// ListenAndServe starts the HTTP server on addr (e.g. ":8080").
func ListenAndServe(addr string) error {
	http.HandleFunc("/ws", handleWS)
	log.Printf("ghost-shell backend listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
