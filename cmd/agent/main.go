package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"goremote/internal/agent"
)

func main() {
	configPath := flag.String("config", "config/agent.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	config, err := agent.LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	agent.InitLogger(config.Log.Level, config.Log.File)

	// Create agent
	ag := agent.NewAgent(config)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		ag.Stop()
		os.Exit(0)
	}()

	// Run agent
	slog.Info("agent starting", "server", config.Server.Address)
	if err := ag.Run(); err != nil {
		slog.Error("agent error", "error", err)
		os.Exit(1)
	}
}
