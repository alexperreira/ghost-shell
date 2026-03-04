package pty

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// Manager wraps a PTY-attached shell process.
type Manager struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

// Start spawns the given shell in a PTY and returns a Manager.
func Start(shell string) (*Manager, error) {
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &Manager{ptmx: ptmx, cmd: cmd}, nil
}

// CWD returns the current working directory of the shell process.
func (m *Manager) CWD() (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/cwd", m.cmd.Process.Pid))
}

// Write sends raw bytes into the PTY (keystrokes from the client).
func (m *Manager) Write(p []byte) (int, error) {
	return m.ptmx.Write(p)
}

// Read reads raw output from the PTY (shell output to send to the client).
func (m *Manager) Read(p []byte) (int, error) {
	return m.ptmx.Read(p)
}

// Resize updates the PTY window size (sent when the browser terminal resizes).
func (m *Manager) Resize(rows, cols uint16) error {
	return pty.Setsize(m.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

// Close shuts down the PTY.
func (m *Manager) Close() error {
	return m.ptmx.Close()
}
