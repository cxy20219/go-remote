package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"goremote/internal/common"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB
)

// Connector manages WebSocket connection
type Connector struct {
	serverAddr string
	tlsEnabled bool
	tlsCA      string
	authKey    string

	conn    *websocket.Conn
	mu      sync.Mutex
	done    chan struct{}
	closed  bool
}

// NewConnector creates a new Connector
func NewConnector(serverAddr string, tlsEnabled bool, tlsCA, authKey string) *Connector {
	return &Connector{
		serverAddr: serverAddr,
		tlsEnabled: tlsEnabled,
		tlsCA:      tlsCA,
		authKey:    authKey,
		done:       make(chan struct{}),
	}
}

// Connect establishes WebSocket connection
func (c *Connector) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build URL
	scheme := "ws"
	if c.tlsEnabled {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: c.serverAddr, Path: "/ws"}

	slog.Info("connecting to server", "addr", c.serverAddr)

	// Set connection options
	dialer := websocket.Dialer{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	if c.tlsEnabled && c.tlsCA != "" {
		// TODO: Configure TLS with CA certificate
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	c.conn = conn
	c.closed = false

	slog.Info("connected to server", "addr", c.serverAddr)
	return nil
}

// Close closes the connection
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Send sends a message
func (c *Connector) Send(msg *common.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteJSON(msg)
}

// Receive receives a message
func (c *Connector) Receive() (*common.Message, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	msg := &common.Message{}
	err := c.conn.ReadJSON(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// Done returns the done channel
func (c *Connector) Done() <-chan struct{} {
	return c.done
}

// IsConnected returns true if connected
func (c *Connector) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed
}

// Register sends a register message
func (c *Connector) Register(hostname, os string) error {
	payload := common.RegisterPayload{
		Key:      c.authKey,
		Hostname: hostname,
		OS:       os,
	}
	payloadJSON, _ := json.Marshal(payload)

	return c.Send(&common.Message{
		Type:    "register",
		Payload: payloadJSON,
	})
}

// SendPing sends a ping message
func (c *Connector) SendPing() error {
	return c.Send(&common.Message{Type: "ping"})
}

// SendResult sends a command result
func (c *Connector) SendResult(taskID string, exitCode int, stdout, stderr string) error {
	payload := common.ResultPayload{
		TaskID:   taskID,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}
	payloadJSON, _ := json.Marshal(payload)

	return c.Send(&common.Message{
		Type:    "result",
		Payload: payloadJSON,
	})
}
