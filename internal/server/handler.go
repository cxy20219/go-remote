package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"goremote/internal/common"
)

// Handler handles WebSocket messages
type Handler struct {
	server *Server
}

// NewHandler creates a new Handler
func NewHandler(server *Server) *Handler {
	return &Handler{server: server}
}

// Handle processes incoming messages
func (h *Handler) Handle(client *Client, msg *common.Message) error {
	switch msg.Type {
	case "register":
		return h.handleRegister(client, msg)
	case "exec":
		return h.handleExec(client, msg)
	case "result":
		return h.handleResult(client, msg)
	case "ping":
		return h.handlePing(client, msg)
	case "upload":
		return h.handleUpload(client, msg)
	case "upload_data":
		return h.handleUploadData(client, msg)
	case "upload_complete":
		return h.handleUploadComplete(client, msg)
	case "download":
		return h.handleDownload(client, msg)
	case "download_complete":
		return h.handleDownloadComplete(client, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleRegister handles client registration
func (h *Handler) handleRegister(client *Client, msg *common.Message) error {
	var payload common.RegisterPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return h.sendRegisterFail(client, "invalid payload")
	}

	// Verify auth key
	if payload.Key != h.server.config.Auth.Key {
		return h.sendRegisterFail(client, "invalid key")
	}

	// Set client info
	client.Hostname = payload.Hostname
	client.OS = payload.OS

	// Generate client ID
	client.ClientID = fmt.Sprintf("client-%s", generateUUID())

	// Add to client manager
	h.server.clients.Add(client)

	slog.Info("client registered",
		"client_id", client.ClientID,
		"hostname", client.Hostname,
		"os", client.OS)

	return h.sendRegisterOK(client)
}

// handleExec handles command execution requests
func (h *Handler) handleExec(client *Client, msg *common.Message) error {
	var payload common.ExecPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	// Create task
	task := &Task{
		TaskID:   payload.TaskID,
		ClientID: client.ClientID,
		Command:  payload.Command,
		Timeout:  payload.Timeout,
	}
	h.server.tasks.Create(task)
	h.server.tasks.Update(payload.TaskID, StatusRunning)

	slog.Info("command queued",
		"task_id", payload.TaskID,
		"client_id", client.ClientID,
		"command", payload.Command)

	// Forward to client
	payloadJSON, _ := json.Marshal(payload)
	return client.Conn.WriteJSON(common.Message{
		Type:    "exec",
		Payload: payloadJSON,
	})
}

// handleResult handles command execution results
func (h *Handler) handleResult(client *Client, msg *common.Message) error {
	var payload common.ResultPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	task := h.server.tasks.Get(payload.TaskID)
	if task == nil {
		slog.Warn("result for unknown task", "task_id", payload.TaskID)
		return nil
	}

	result := &TaskResult{
		ExitCode: payload.ExitCode,
		Stdout:   payload.Stdout,
		Stderr:   payload.Stderr,
	}
	h.server.tasks.SetResult(payload.TaskID, result)

	slog.Info("task completed",
		"task_id", payload.TaskID,
		"exit_code", payload.ExitCode)

	return nil
}

// handlePing handles heartbeat
func (h *Handler) handlePing(client *Client, msg *common.Message) error {
	h.server.clients.UpdatePing(client.ClientID)
	return client.Conn.WriteJSON(common.Message{Type: "pong"})
}

// handleUpload handles file upload initiation (server -> client)
func (h *Handler) handleUpload(client *Client, msg *common.Message) error {
	var payload common.UploadPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	slog.Info("upload initiated",
		"file_id", payload.FileID,
		"filename", payload.Filename,
		"client_id", client.ClientID)

	// Create upload record
	upload := &UploadFile{
		FileID:   payload.FileID,
		Filename: payload.Filename,
		Size:     payload.Size,
		Offset:   payload.Offset,
		TaskID:   payload.TaskID,
		ClientID: client.ClientID,
	}
	h.server.uploadManager.Add(upload)

	// Send file in chunks
	return h.sendFileInChunks(client, payload.FileID, payload.Filename)
}

// sendFileInChunks sends a file to a client in chunks
func (h *Handler) sendFileInChunks(client *Client, fileID, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return h.sendUploadComplete(client, fileID, false, err.Error())
	}
	defer file.Close()

	stat, _ := file.Stat()
	chunkSize := 64 * 1024 // 64KB per chunk
	buffer := make([]byte, chunkSize)
	offset := int64(0)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			data := base64.StdEncoding.EncodeToString(buffer[:n])
			eof := int64(offset)+int64(n) >= stat.Size()

			uploadDataPayload := common.UploadDataPayload{
				FileID: fileID,
				Offset: offset,
				Data:   data,
				EOF:    eof,
			}
			payloadJSON, _ := json.Marshal(uploadDataPayload)
			err := client.Conn.WriteJSON(common.Message{
				Type:    "upload_data",
				Payload: payloadJSON,
			})
			if err != nil {
				return err
			}
			offset += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// handleUploadData handles file data from client
func (h *Handler) handleUploadData(client *Client, msg *common.Message) error {
	var payload common.UploadDataPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	transfer := h.server.uploadManager.Get(payload.FileID)
	if transfer == nil {
		return fmt.Errorf("unknown file_id: %s", payload.FileID)
	}

	// Decode and write data
	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return err
	}

	_, err = transfer.File.Write(data)
	if err != nil {
		transfer.File.Close()
		h.server.uploadManager.Remove(payload.FileID)
		return err
	}

	transfer.Transferred += int64(len(data))

	if payload.EOF {
		transfer.Status = "completed"
		transfer.File.Close()
		h.server.uploadManager.Remove(payload.FileID)
	}

	return nil
}

// handleUploadComplete handles upload completion confirmation
func (h *Handler) handleUploadComplete(client *Client, msg *common.Message) error {
	var payload common.UploadCompletePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	if !payload.Success {
		slog.Warn("upload failed",
			"file_id", payload.FileID,
			"error", payload.Error)
	} else {
		slog.Info("upload completed",
			"file_id", payload.FileID)
	}

	return nil
}

// handleDownload handles client download request
func (h *Handler) handleDownload(client *Client, msg *common.Message) error {
	var payload common.DownloadPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	slog.Info("download requested",
		"file_id", payload.FileID,
		"filename", payload.Filename,
		"client_id", client.ClientID)

	// Send file in chunks to client
	return h.sendFileInChunks(client, payload.FileID, payload.Filename)
}

// handleDownloadComplete handles download completion
func (h *Handler) handleDownloadComplete(client *Client, msg *common.Message) error {
	var payload common.DownloadCompletePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	if !payload.Success {
		slog.Warn("download failed",
			"file_id", payload.FileID,
			"error", payload.Error)
	} else {
		slog.Info("download completed",
			"file_id", payload.FileID)
	}

	return nil
}

// Helper methods

func (h *Handler) sendRegisterOK(client *Client) error {
	payload := common.RegisterOKPayload{ClientID: client.ClientID}
	payloadJSON, _ := json.Marshal(payload)
	return client.Conn.WriteJSON(common.Message{
		Type:    "register_ok",
		Payload: payloadJSON,
	})
}

func (h *Handler) sendRegisterFail(client *Client, reason string) error {
	payload := common.RegisterFailPayload{Reason: reason}
	payloadJSON, _ := json.Marshal(payload)
	return client.Conn.WriteJSON(common.Message{
		Type:    "register_fail",
		Payload: payloadJSON,
	})
}

func (h *Handler) sendUploadComplete(client *Client, fileID string, success bool, errMsg string) error {
	payload := common.UploadCompletePayload{
		FileID:  fileID,
		Success: success,
		Error:   errMsg,
	}
	payloadJSON, _ := json.Marshal(payload)
	return client.Conn.WriteJSON(common.Message{
		Type:    "upload_complete",
		Payload: payloadJSON,
	})
}
