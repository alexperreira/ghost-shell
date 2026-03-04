package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/alexperreira/ghost-shell/internal/pty"
	"github.com/gorilla/websocket"
)

// ClientMessage is a message sent from the browser to the backend.
type ClientMessage struct {
	Type string `json:"type"` // "input" | "resize"
	Data string `json:"data"` // raw input bytes (base64 not needed; UTF-8 safe for most input)
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// ServerMessage is a message sent from the backend to the browser.
type ServerMessage struct {
	Type string `json:"type"` // "output" | "exit"
	Data string `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

	// done is closed by whichever side exits first, triggering cleanup of the other.
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
				conn.WriteJSON(ServerMessage{Type: "exit"}) //nolint:errcheck
				conn.Close()
				return
			}
			msg := ServerMessage{Type: "output", Data: string(buf[:n])}
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}()

	// WebSocket → PTY
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			// Browser disconnected — kill the shell.
			mgr.Close()
			<-done // wait for the reader goroutine to finish
			return
		}
		var msg ClientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "input":
			mgr.Write([]byte(msg.Data)) //nolint:errcheck
		case "resize":
			if msg.Rows > 0 && msg.Cols > 0 {
				mgr.Resize(msg.Rows, msg.Cols) //nolint:errcheck
			}
		}
	}
}

// ListenAndServe starts the HTTP server on addr (e.g. ":8080").
func ListenAndServe(addr string) error {
	http.HandleFunc("/ws", handleWS)
	log.Printf("ghost-shell backend listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
