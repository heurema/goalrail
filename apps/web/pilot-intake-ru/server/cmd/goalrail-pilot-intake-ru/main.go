package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/heurema/goalrail/apps/web/pilot-intake-ru/server/internal/pilotlead"
)

func main() {
	config := pilotlead.DefaultConfig()
	if listenAddr := os.Getenv("GOALRAIL_PILOT_LISTEN_ADDR"); listenAddr != "" {
		config.ListenAddr = listenAddr
	}

	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	store := pilotlead.NewStore(config.LeadLogPath, config.Now)
	mailer := pilotlead.NewTransportMailer(config)

	switch command {
	case "serve":
		server := &http.Server{
			Addr:    config.ListenAddr,
			Handler: pilotlead.NewServer(config, store, mailer),
		}
		log.Printf("goalrail pilot intake sidecar listening on %s", config.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	case "digest":
		if err := pilotlead.RunDigest(context.Background(), config, store, mailer, os.Stdout); err != nil {
			os.Exit(1)
		}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s [serve|digest]\n", os.Args[0])
		os.Exit(2)
	}
}
