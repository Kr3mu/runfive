package models

import "time"

// ServerStatus is the lifecycle state surfaced to the dashboard.
type ServerStatus string

const (
	ServerStatusRunning  ServerStatus = "running"
	ServerStatusStarting ServerStatus = "starting"
	ServerStatusStopped  ServerStatus = "stopped"
	ServerStatusCrashed  ServerStatus = "crashed"
)

// ManagedServer represents a server discovered from the filesystem.
type ManagedServer struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Status          ServerStatus `json:"status"`
	Address         string       `json:"address"`
	PlayerCount     int          `json:"playerCount"`
	MaxPlayers      int          `json:"maxPlayers"`
	CPU             int          `json:"cpu"`
	RamMB           int          `json:"ramMB"`
	TickMs          float64      `json:"tickMs"`
	ArtifactVersion string       `json:"artifactVersion"`
}

// ServerProcessStatus is the live runtime state for one launched server.
type ServerProcessStatus struct {
	ID         string       `json:"id"`
	Status     ServerStatus `json:"status"`
	PID        int          `json:"pid,omitempty"`
	ExitCode   *int         `json:"exitCode,omitempty"`
	ExitReason string       `json:"exitReason,omitempty"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}

// ServerLogLine is one console line captured from a managed server.
type ServerLogLine struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Message   string    `json:"message"`
}

// ServerLogsResponse is returned by GET /v1/servers/:id/logs.
type ServerLogsResponse struct {
	Lines []ServerLogLine `json:"lines"`
}

// ServerConsoleEvent is sent over the live console websocket.
type ServerConsoleEvent struct {
	Type   string               `json:"type"`
	Status *ServerProcessStatus `json:"status,omitempty"`
	Lines  []ServerLogLine      `json:"lines,omitempty"`
	Line   *ServerLogLine       `json:"line,omitempty"`
	Error  string               `json:"error,omitempty"`
}

// CreateServerRequest is the body for POST /v1/servers.
type CreateServerRequest struct {
	Name            string `json:"name"`
	ArtifactVersion string `json:"artifactVersion"`
	// LicenseKey is the optional Cfx.re license key (cfxk_...) the panel
	// encrypts and writes to the new server's TOML. Empty means "I'll set
	// this later" — the server will refuse to boot until one is provided.
	LicenseKey string `json:"licenseKey,omitempty"`
}

// InstalledArtifact represents an extracted artifact on disk.
type InstalledArtifact struct {
	OS      string `json:"os"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// AvailableArtifactVersion represents one upstream artifact version.
type AvailableArtifactVersion struct {
	Version      string `json:"version"`
	Installed    bool   `json:"installed"`
	BrokenReason string `json:"brokenReason,omitempty"`
}

// ArtifactListResponse is returned by GET /v1/artifacts.
type ArtifactListResponse struct {
	OS          string                     `json:"os"`
	Recommended string                     `json:"recommended,omitempty"`
	Installed   []InstalledArtifact        `json:"installed"`
	Available   []AvailableArtifactVersion `json:"available"`
}

// DownloadArtifactRequest is the body for POST /v1/artifacts/download.
type DownloadArtifactRequest struct {
	Version string `json:"version"`
}
