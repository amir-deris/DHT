package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirderis/DHT/internal/config"
	"github.com/amirderis/DHT/internal/server"
)

func main() {
	cfg := config.Flags()

	flag.StringVar(&cfg.NodeID, "node-id", "", "Unique node identifier")
	flag.StringVar(&cfg.BindAddr, "bind", ":8080", "Bind address, e.g. 0.0.0.0:8080")
	flag.StringVar(&cfg.SeedsCSV, "seeds", "", "Comma-separated seed addresses for gossip (host:port)")
	flag.IntVar(&cfg.ReplicationFactor, "replication-factor", 3, "Replication factor N")
	flag.IntVar(&cfg.ReadQuorum, "r", 2, "Read quorum R")
	flag.IntVar(&cfg.WriteQuorum, "w", 2, "Write quorum W")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	srv := server.NewHTTPServer(cfg)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Printf("node %s listening on %s", cfg.NodeID, cfg.BindAddr)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}
