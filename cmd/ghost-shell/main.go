package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/alexperreira/ghost-shell/internal/server"
)

// loadDotEnv reads a .env file and sets any unset environment variables found in it.
// Lines starting with # and blank lines are ignored. Already-set vars are not overwritten.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" && os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func main() {
	loadDotEnv(".env")

	addr := flag.String("addr", ":8080", "WebSocket server address")
	flag.Parse()

	if err := server.ListenAndServe(*addr); err != nil {
		log.Fatal(err)
	}
}
