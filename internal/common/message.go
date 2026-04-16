package common

import "encoding/json"

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// RegisterPayload is sent by client to register
type RegisterPayload struct {
	Key      string `json:"key"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
}

// RegisterOKPayload is sent by server on successful registration
type RegisterOKPayload struct {
	ClientID string `json:"client_id"`
}

// RegisterFailPayload is sent by server on failed registration
type RegisterFailPayload struct {
	Reason string `json:"reason"`
}

// ExecPayload is sent by server to execute a command
type ExecPayload struct {
	TaskID  string `json:"task_id"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// ResultPayload is sent by client with command result
type ResultPayload struct {
	TaskID   string `json:"task_id"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// UploadPayload is sent by server to initiate file upload to client
type UploadPayload struct {
	TaskID   string `json:"task_id"`
	FileID   string `json:"file_id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Offset   int64  `json:"offset"`
}

// UploadDataPayload contains chunk of file data
type UploadDataPayload struct {
	FileID string `json:"file_id"`
	Offset int64  `json:"offset"`
	Data   string `json:"data"` // base64 encoded
	EOF    bool   `json:"eof"`
}

// UploadCompletePayload is sent by client after receiving file
type UploadCompletePayload struct {
	FileID  string `json:"file_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// DownloadPayload is sent by client to request file download from server
type DownloadPayload struct {
	TaskID   string `json:"task_id"`
	FileID   string `json:"file_id"`
	Filename string `json:"filename"`
	Offset   int64  `json:"offset"`
}

// DownloadDataPayload contains chunk of file data from server
type DownloadDataPayload struct {
	FileID string `json:"file_id"`
	Offset int64  `json:"offset"`
	Data   string `json:"data"` // base64 encoded
	EOF    bool   `json:"eof"`
}

// DownloadCompletePayload is sent by client after download completes
type DownloadCompletePayload struct {
	FileID  string `json:"file_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
