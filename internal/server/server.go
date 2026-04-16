package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"goremote/internal/common"
)

// Server is the WebSocket server
type Server struct {
	Addr       string
	tlsEnabled bool
	tlsCert    string
	tlsKey     string

	config *Config

	clients       *ClientManager
	tasks         *TaskStore
	handler       *Handler
	uploadManager *UploadManager

	upgrader websocket.Upgrader
	engine   *gin.Engine
}

// NewServer creates a new Server
func NewServer(config *Config) *Server {
	s := &Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		tlsEnabled:   config.Server.TLS.Enabled,
		tlsCert:      config.Server.TLS.Cert,
		tlsKey:       config.Server.TLS.Key,
		config:       config,
		clients:      NewClientManager(),
		tasks:        NewTaskStore(),
		handler:      nil,
		uploadManager: NewUploadManager(),
	}

	s.handler = NewHandler(s)

	s.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return s
}

// Start starts the server
func (s *Server) Start() error {
	slog.Info("server starting", "addr", s.Addr)

	// Setup Gin engine
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.engine.Use(gin.Recovery())

	// WebSocket route - keep original upgrade logic
	s.engine.GET("/ws", s.handleConn)

	// API routes
	s.engine.GET("/api/clients", s.handleAPIClients)
	s.engine.GET("/api/clients/:id", s.handleAPIClientDetail)
	s.engine.POST("/api/exec", s.handleAPIExec)
	s.engine.POST("/api/openclaw", s.handleOpenClaw)
	s.engine.GET("/api/tasks", s.handleAPITasks)
	s.engine.GET("/api/tasks/:id", s.handleAPITaskDetail)
	s.engine.GET("/health", s.handleHealth)

	// Swagger UI
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start heartbeat checker
	go s.startHeartbeatChecker()

	if s.tlsEnabled {
		slog.Info("server starting with TLS", "cert", s.tlsCert)
		return s.engine.RunTLS(s.Addr, s.tlsCert, s.tlsKey)
	}

	return s.engine.Run(s.Addr)
}

// Stop stops the server
func (s *Server) Stop() error {
	slog.Info("server stopping")
	// Gin doesn't have a direct shutdown, but we can use context
	return nil
}

// handleConn handles WebSocket connections
func (s *Server) handleConn(c *gin.Context) {
	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("failed to upgrade connection", "error", err)
		return
	}

	remoteAddr := c.Request.RemoteAddr
	slog.Info("connection opened", "remote", remoteAddr)

	// Create client with temporary ID until registered
	client := &Client{
		Conn: conn,
	}

	// Read messages
	for {
		var msg common.Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("connection error", "error", err)
			}
			break
		}

		// Handle the message
		if err := s.handler.Handle(client, &msg); err != nil {
			slog.Error("failed to handle message",
				"type", msg.Type,
				"error", err)
		}

		// For register messages, client ID is set after registration
		if msg.Type == "register" && client.ClientID != "" {
			// Client registered successfully, start reading loop for this client
			go s.readLoop(client)
			return
		}
	}

	conn.Close()
	slog.Info("connection closed", "remote", remoteAddr)
}

// readLoop reads messages from a registered client
func (s *Server) readLoop(client *Client) {
	defer func() {
		if client.ClientID != "" {
			s.clients.Remove(client.ClientID)
			slog.Info("client disconnected", "client_id", client.ClientID)
		}
		client.Conn.Close()
	}()

	for {
		var msg common.Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("connection error", "client_id", client.ClientID, "error", err)
			}
			return
		}

		if err := s.handler.Handle(client, &msg); err != nil {
			slog.Error("failed to handle message",
				"client_id", client.ClientID,
				"type", msg.Type,
				"error", err)
		}
	}
}

// handleHealth 健康检查
// @Summary 健康检查
// @Description 服务健康状态检查
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// startHeartbeatChecker starts the heartbeat timeout checker
func (s *Server) startHeartbeatChecker() {
	interval := time.Duration(s.config.Heartbeat.Interval) * time.Second
	timeout := time.Duration(s.config.Heartbeat.Timeout) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.checkClientsTimeout(timeout)
	}
}

// checkClientsTimeout checks for timed out clients
func (s *Server) checkClientsTimeout(timeout time.Duration) {
	now := time.Now()
	s.clients.mu.Lock()
	defer s.clients.mu.Unlock()

	for _, client := range s.clients.clients {
		if now.Sub(client.LastPing) > timeout {
			slog.Warn("client heartbeat timeout", "client_id", client.ClientID)
			client.Conn.Close()
		}
	}
}

// GetClients returns all connected clients
func (s *Server) GetClients() []*Client {
	return s.clients.List()
}

// GetTasks returns all tasks
func (s *Server) GetTasks() []*Task {
	return s.tasks.List()
}

// GetTask returns a task by ID
func (s *Server) GetTask(taskID string) *Task {
	return s.tasks.Get(taskID)
}

// SendCommand sends a command to a client
func (s *Server) SendCommand(clientID, taskID, command string, timeout int) error {
	client, ok := s.clients.Get(clientID)
	if !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	payload := common.ExecPayload{
		TaskID:  taskID,
		Command: command,
		Timeout: timeout,
	}
	payloadJSON, _ := json.Marshal(payload)

	return client.Conn.WriteJSON(common.Message{
		Type:    "exec",
		Payload: payloadJSON,
	})
}

// generateUUID generates a simple UUID (for demo purposes)
// In production, use github.com/google/uuid
func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", time.Now().UnixNano(),
		time.Now().Unix()&0xffff, 0x4000, 0x8000, time.Now().UnixNano())
}
