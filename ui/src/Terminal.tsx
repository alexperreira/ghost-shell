import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

const WS_URL = "ws://localhost:8080/ws";

type GhostMode = { active: false } | { active: true; buffer: string };
type PendingCmd = { command: string } | null;

export default function TerminalComponent() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const term = new Terminal({ cursorBlink: true });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerRef.current!);
    fitAddon.fit();

    const ws = new WebSocket(WS_URL);
    ws.binaryType = "arraybuffer";

    let atLineStart = true;
    let ghostMode: GhostMode = { active: false };
    let pendingCmd: PendingCmd = null;
    let isLoading = false;

    ws.onopen = () => {
      const { rows, cols } = term;
      ws.send(JSON.stringify({ type: "resize", rows, cols }));
    };

    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data as string);
      switch (msg.type) {
        case "output":
          term.write(msg.data);
          if ((msg.data as string).includes("\r")) {
            atLineStart = true;
          }
          break;
        case "exit":
          term.write("\r\n[Process exited]\r\n");
          ws.close();
          break;
        case "ai_chunk":
          if (isLoading) {
            term.write("\r\x1b[K");
            isLoading = false;
          }
          term.write(`\x1b[36m${msg.data as string}\x1b[0m`);
          break;
        case "ai_done":
          if (isLoading) {
            term.write("\r\x1b[K");
            isLoading = false;
          }
          term.write("\r\n");
          if (msg.command) {
            term.write(
              `\x1b[2m[Enter] Run: ${msg.command as string}   [Esc] Dismiss\x1b[0m\r\n`
            );
            pendingCmd = { command: msg.command as string };
          }
          atLineStart = true;
          break;
        case "ai_error":
          if (isLoading) {
            term.write("\r\x1b[K");
            isLoading = false;
          }
          term.write(`\x1b[31m[ghost error] ${msg.data as string}\x1b[0m\r\n`);
          atLineStart = true;
          break;
      }
    };

    ws.onerror = () => {
      term.write("\r\n\x1b[31m[ghost] WebSocket error — connection failed\x1b[0m\r\n");
    };

    ws.onclose = () => {
      term.write("\r\n[Disconnected]\r\n");
    };

    term.onData((data) => {
      if (ws.readyState !== WebSocket.OPEN) return;

      // Pending command intercept — highest priority.
      if (pendingCmd) {
        if (data === "\r") {
          ws.send(JSON.stringify({ type: "run_command", data: pendingCmd.command }));
          pendingCmd = null;
          atLineStart = true;
        } else if (data === "\x1b") {
          term.write("\x1b[2m[dismissed]\x1b[0m\r\n");
          pendingCmd = null;
          atLineStart = true;
        }
        // Swallow all other keys while a command is pending.
        return;
      }

      // Ghost input mode.
      if (ghostMode.active) {
        if (data === "\r") {
          ws.send(JSON.stringify({ type: "ai_query", data: ghostMode.buffer }));
          ghostMode = { active: false };
          isLoading = true;
          term.write("\r\n\x1b[2m[ghost] thinking...\x1b[0m");
          atLineStart = false;
        } else if (data === "\x1b") {
          // Cancel — erase "ghost> " prefix (7 chars) + buffered text.
          const eraseCount = 7 + ghostMode.buffer.length;
          term.write("\b \b".repeat(eraseCount));
          ghostMode = { active: false };
          atLineStart = true;
        } else if (data === "\x7f") {
          // Backspace.
          if ((ghostMode as { active: true; buffer: string }).buffer.length > 0) {
            ghostMode = {
              active: true,
              buffer: (ghostMode as { active: true; buffer: string }).buffer.slice(0, -1),
            };
            term.write("\b \b");
          }
        } else if (data >= " ") {
          // Printable character.
          ghostMode = {
            active: true,
            buffer: (ghostMode as { active: true; buffer: string }).buffer + data,
          };
          term.write(data);
        }
        return;
      }

      // Normal input — check for ghost mode trigger.
      if (atLineStart && data === "?") {
        ghostMode = { active: true, buffer: "" };
        term.write("\x1b[36mghost> \x1b[0m");
        atLineStart = false;
        return;
      }

      ws.send(JSON.stringify({ type: "input", data }));
      if (data.includes("\r") || data.includes("\n")) {
        atLineStart = true;
      } else {
        atLineStart = false;
      }
    });

    const handleResize = () => {
      fitAddon.fit();
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
      }
    };
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      ws.close();
      term.dispose();
    };
  }, []);

  return <div ref={containerRef} style={{ width: "100%", height: "100%" }} />;
}
