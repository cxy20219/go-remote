package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"goremote/internal/server"

	_ "goremote/docs"
)

func init() {
	// Disable Gin colored output
	gin.DisableConsoleColor()
}

func main() {
	configPath := flag.String("config", "config/server.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	config, err := server.LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	server.InitLogger(config.Log.Level, config.Log.File)

	// Create server
	srv := server.NewServer(config)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		srv.Stop()
		os.Exit(0)
	}()

	// Start server
	slog.Info("server starting")
	if err := srv.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
