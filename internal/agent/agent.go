package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"goremote/internal/common"
)

// Agent is the main agent controller
type Agent struct {
	config     *Config
	connector  *Connector
	handler    *Handler
	executor   *Executor
	reconnect  *Reconnect
	hostname   string
	os         string
	clientID   string
	mu         sync.RWMutex
	stopped    bool
}

// NewAgent creates a new Agent
func NewAgent(config *Config) *Agent {
	hostname := config.Agent.Hostname
	if hostname == "" {
		hostname = common.GetHostname()
	}

	os := config.Agent.OS
	if os == "" {
		os = common.GetOS()
	}

	connector := NewConnector(
		config.Server.Address,
		config.Server.TLS.Enabled,
		config.Server.TLS.CA,
		config.Auth.Key,
	)

	agent := &Agent{
		config:    config,
		connector: connector,
		handler:   nil,
		executor:  NewExecutor(),
		reconnect: NewReconnect(
			config.Reconnect.Interval,
			config.Reconnect.MaxAttempts,
			config.Reconnect.MaxInterval,
		),
		hostname: hostname,
		os:       os,
	}

	agent.handler = NewHandler(agent)

	return agent
}

// Run starts the agent
func (a *Agent) Run() error {
	slog.Info("agent starting",
		"server", a.config.Server.Address,
		"hostname", a.hostname,
		"os", a.os)

	for {
		if a.isStopped() {
			return nil
		}

		// Connect
		if err := a.connect(); err != nil {
			slog.Warn("connection failed", "error", err)

			if !a.reconnect.shouldRetry() {
				slog.Error("max reconnect attempts reached")
				return fmt.Errorf("max reconnect attempts reached")
			}

			a.reconnect.beforeRetry()
			continue
		}

		// Handle connection
		if err := a.handleConnection(); err != nil {
			slog.Warn("connection error", "error", err)
		}

		// Close connection
		a.connector.Close()
		a.reconnect.beforeRetry()

		if a.isStopped() {
			return nil
		}
	}
}

// Stop stops the agent
func (a *Agent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopped = true
	a.connector.Close()
}

func (a *Agent) isStopped() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stopped
}

func (a *Agent) connect() error {
	if err := a.connector.Connect(); err != nil {
		return err
	}

	// Register
	if err := a.connector.Register(a.hostname, a.os); err != nil {
		a.connector.Close()
		return err
	}

	// Wait for register response
	for {
		msg, err := a.connector.Receive()
		if err != nil {
			a.connector.Close()
			return err
		}

		if msg.Type == "register_ok" {
			var payload common.RegisterOKPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				a.connector.Close()
				return err
			}
			a.SetClientID(payload.ClientID)
			slog.Info("registered successfully", "client_id", payload.ClientID)
			return nil
		}

		if msg.Type == "register_fail" {
			var payload common.RegisterFailPayload
			json.Unmarshal(msg.Payload, &payload)
			a.connector.Close()
			return fmt.Errorf("registration failed: %s", payload.Reason)
		}
	}
}

func (a *Agent) handleConnection() error {
	// Start heartbeat
	hbDone := make(chan struct{})
	go a.startHeartbeat(hbDone)

	// Read messages
	for {
		msg, err := a.connector.Receive()
		if err != nil {
			close(hbDone)
			return err
		}

		if err := a.handler.Handle(msg); err != nil {
			slog.Error("failed to handle message", "type", msg.Type, "error", err)
		}
	}
}

func (a *Agent) startHeartbeat(done chan struct{}) {
	interval := time.Duration(a.config.Heartbeat.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.connector.SendPing(); err != nil {
				slog.Warn("heartbeat failed", "error", err)
				return
			}
		case <-done:
			return
		}
	}
}

// SetClientID sets the client ID
func (a *Agent) SetClientID(clientID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.clientID = clientID
}

// GetClientID returns the client ID
func (a *Agent) GetClientID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.clientID
}

// SendResult sends a command result to server
func (a *Agent) SendResult(taskID string, exitCode int, stdout, stderr string) error {
	return a.connector.SendResult(taskID, exitCode, stdout, stderr)
}

// SendUploadComplete sends upload complete message
func (a *Agent) SendUploadComplete(fileID string, success bool, errMsg string) error {
	payload := common.UploadCompletePayload{
		FileID:  fileID,
		Success: success,
		Error:   errMsg,
	}
	payloadJSON, _ := json.Marshal(payload)
	return a.connector.Send(&common.Message{
		Type:    "upload_complete",
		Payload: payloadJSON,
	})
}

// SendDownloadComplete sends download complete message
func (a *Agent) SendDownloadComplete(fileID string, success bool, errMsg string) error {
	payload := common.DownloadCompletePayload{
		FileID:  fileID,
		Success: success,
		Error:   errMsg,
	}
	payloadJSON, _ := json.Marshal(payload)
	return a.connector.Send(&common.Message{
		Type:    "download_complete",
		Payload: payloadJSON,
	})
}

// SendDownloadRequest sends a download request to server
func (a *Agent) SendDownloadRequest(taskID, fileID, filename string, offset int64) error {
	payload := common.DownloadPayload{
		TaskID:   taskID,
		FileID:   fileID,
		Filename: filename,
		Offset:   offset,
	}
	payloadJSON, _ := json.Marshal(payload)
	return a.connector.Send(&common.Message{
		Type:    "download",
		Payload: payloadJSON,
	})
}
