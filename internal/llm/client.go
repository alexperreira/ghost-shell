package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const systemPrompt = `You are ghost, an AI assistant built into a terminal emulator.
The user is working in a shell and asking for help with commands and shell tasks.

Rules:
- Be concise. Prefer showing a command over explaining one.
- If the answer is a single runnable command, put it alone on the last line
  inside a fenced code block with no language tag.
- If multiple commands are needed, use a numbered list, then put the full
  pipeline on the last line in a fenced code block.
- Never suggest destructive commands (rm -rf, mkfs, dd) without a warning.
- Do not repeat the user's question back to them.`

// Client wraps the Anthropic streaming API.
type Client struct {
	inner anthropic.Client
}

// Request holds the context for an AI query.
type Request struct {
	Query   string
	CWD     string
	History []string // last N commands, oldest first
}

// New creates a Client. Returns an error if ANTHROPIC_API_KEY is unset.
func New() (*Client, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}
	c := anthropic.NewClient(option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	return &Client{inner: c}, nil
}

// Stream sends req to Claude and writes each token to out.
// Returns when streaming completes or ctx is cancelled.
func (c *Client) Stream(ctx context.Context, req Request, out chan<- string) error {
	stream := c.inner.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeOpus4_6,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildUserPrompt(req))),
		},
	})

	for stream.Next() {
		event := stream.Current()
		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch delta := ev.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				select {
				case out <- delta.Text:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return stream.Err()
}

func buildUserPrompt(req Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Working directory: %s\n", req.CWD)
	if len(req.History) > 0 {
		b.WriteString("Recent commands:\n")
		for _, h := range req.History {
			b.WriteString(h)
			b.WriteByte('\n')
		}
	}
	fmt.Fprintf(&b, "\nUser: %s", req.Query)
	return b.String()
}

// ExtractCommand returns the suggested command from an AI response, or "".
// Finds the last fenced code block with exactly one non-empty line.
// Falls back to the full response if it is itself a single non-empty line.
func ExtractCommand(response string) string {
	// Collect positions of all ``` markers.
	var fences []int
	remaining := response
	offset := 0
	for {
		idx := strings.Index(remaining, "```")
		if idx < 0 {
			break
		}
		fences = append(fences, offset+idx)
		skip := idx + 3
		offset += skip
		remaining = remaining[skip:]
	}

	if len(fences) >= 2 {
		// Take the last complete pair.
		openPos := fences[len(fences)-2]
		closePos := fences[len(fences)-1]

		// Skip optional language tag (up to first newline after opening fence).
		afterOpen := response[openPos+3:]
		nl := strings.IndexByte(afterOpen, '\n')
		if nl < 0 {
			return ""
		}
		bodyStart := openPos + 3 + nl + 1
		if bodyStart > closePos {
			return ""
		}
		blockBody := response[bodyStart:closePos]
		lines := nonEmptyLines(blockBody)
		if len(lines) == 1 {
			return lines[0]
		}
		return ""
	}

	// Fallback: treat whole response as a command if it's a single line.
	lines := nonEmptyLines(response)
	if len(lines) == 1 {
		return lines[0]
	}
	return ""
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}
