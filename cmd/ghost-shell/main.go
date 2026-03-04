package main

import (
	"flag"
	"log"

	"github.com/alexperreira/ghost-shell/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "WebSocket server address")
	flag.Parse()

	if err := server.ListenAndServe(*addr); err != nil {
		log.Fatal(err)
	}
}
