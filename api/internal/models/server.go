package models

// ServerStatus is the lifecycle state surfaced to the dashboard.
type ServerStatus string

const (
	ServerStatusOnline   ServerStatus = "online"
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

// CreateServerRequest is the body for POST /v1/servers.
type CreateServerRequest struct {
	Name            string `json:"name"`
	ArtifactVersion string `json:"artifactVersion"`
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
