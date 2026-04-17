package v1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	ws "github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"

	"github.com/runfivedev/runfive/internal/auth"
	"github.com/runfivedev/runfive/internal/launcher"
	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/permissions"
)

const (
	// wsPongWait is how long we'll wait for a pong from the client before
	// treating the connection as dead. Must be larger than wsPingPeriod.
	wsPongWait = 60 * time.Second
	// wsPingPeriod is how often the server sends a ping frame. Should be ~60%
	// of wsPongWait to tolerate one missed pong.
	wsPingPeriod = 30 * time.Second
	// wsWriteWait caps the time any single websocket write may block.
	wsWriteWait = 10 * time.Second
)

type serverRegistry interface {
	List() ([]models.ManagedServer, error)
	Create(name, artifactVersion, licenseKey string, port, maxPlayers int) (models.ManagedServer, error)
	Reload() error
}

type serverArtifactManager interface {
	Install(context.Context, string) (models.InstalledArtifact, error)
}

type serverLauncher interface {
	Start(string) (models.ServerProcessStatus, error)
	Stop(string) (models.ServerProcessStatus, error)
	Status(string) (models.ServerProcessStatus, error)
	Tail(string, int) ([]models.ServerLogLine, error)
	Subscribe(string) (*launcher.Subscription, error)
	SendCommand(string, string) error
}

// ServerHandler serves filesystem-backed managed server endpoints.
type ServerHandler struct {
	registry  serverRegistry
	artifacts serverArtifactManager
	launcher  serverLauncher
}

// NewServerHandler creates a server handler with its dependencies.
func NewServerHandler(registry serverRegistry, artifacts serverArtifactManager, serverLauncher serverLauncher) *ServerHandler {
	return &ServerHandler{registry: registry, artifacts: artifacts, launcher: serverLauncher}
}

// List returns filesystem-discovered servers, filtered by RBAC visibility.
//
// GET /v1/servers
func (h *ServerHandler) List(c fiber.Ctx) error {
	servers, err := h.registry.List()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	perms := auth.GetPermissions(c)
	if perms != nil && !perms.IsOwner {
		filtered := make([]models.ManagedServer, 0, len(servers))
		for i := range servers {
			if _, ok := perms.Servers[servers[i].ID]; ok {
				filtered = append(filtered, servers[i])
			}
		}
		servers = filtered
	}

	for i := range servers {
		status, statusErr := h.launcher.Status(servers[i].ID)
		if statusErr == nil {
			servers[i].Status = status.Status
		}
	}

	return c.JSON(servers)
}

// Create creates a new server folder and server.toml, auto-installing artifacts as needed.
//
// POST /v1/servers
func (h *ServerHandler) Create(c fiber.Ctx) error {
	var req models.CreateServerRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	req.Name = strings.TrimSpace(req.Name)
	req.ArtifactVersion = strings.TrimSpace(req.ArtifactVersion)
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)

	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "server name is required")
	}
	if req.ArtifactVersion == "" {
		return fiber.NewError(fiber.StatusBadRequest, "artifact version is required")
	}
	if req.LicenseKey != "" && !strings.HasPrefix(req.LicenseKey, "cfxk_") {
		return fiber.NewError(fiber.StatusBadRequest, "license key must start with cfxk_")
	}

	if _, err := h.artifacts.Install(context.Background(), req.ArtifactVersion); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	server, err := h.registry.Create(req.Name, req.ArtifactVersion, req.LicenseKey, req.Port, req.MaxPlayers)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(server)
}

// Reload forces a full rescan of the servers directory, rebuilding the
// in-memory registry from disk. Intended as a manual fallback when the
// filesystem watcher is unavailable.
//
// POST /v1/admin/reload-servers
func (h *ServerHandler) Reload(c fiber.Ctx) error {
	if err := h.registry.Reload(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Start launches the managed server process.
//
// POST /v1/servers/:serverId/start
func (h *ServerHandler) Start(c fiber.Ctx) error {
	status, err := h.launcher.Start(c.Params("serverId"))
	if err != nil {
		return launcherHTTPError(err)
	}
	return c.JSON(status)
}

// Stop terminates the managed server process.
//
// POST /v1/servers/:serverId/stop
func (h *ServerHandler) Stop(c fiber.Ctx) error {
	status, err := h.launcher.Stop(c.Params("serverId"))
	if err != nil {
		return launcherHTTPError(err)
	}
	return c.JSON(status)
}

// Status returns the live runtime state of one managed server.
//
// GET /v1/servers/:serverId/status
func (h *ServerHandler) Status(c fiber.Ctx) error {
	status, err := h.launcher.Status(c.Params("serverId"))
	if err != nil {
		return launcherHTTPError(err)
	}
	return c.JSON(status)
}

// Logs returns the recent bounded log tail for one managed server.
//
// GET /v1/servers/:serverId/logs
func (h *ServerHandler) Logs(c fiber.Ctx) error {
	n := parseTailCount(c.Query("n"))
	lines, err := h.launcher.Tail(c.Params("serverId"), n)
	if err != nil {
		return launcherHTTPError(err)
	}
	return c.JSON(models.ServerLogsResponse{Lines: lines})
}

// StreamLogs upgrades to a websocket that streams console lines and accepts
// console commands when the caller also has console.execute.
func (h *ServerHandler) StreamLogs(conn *ws.Conn) {
	serverID := conn.Params("serverId")

	var writeMu sync.Mutex
	writeJSON := func(payload any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
		return conn.WriteJSON(payload)
	}
	writePing := func() error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteControl(ws.PingMessage, nil, time.Now().Add(wsWriteWait))
	}

	status, err := h.launcher.Status(serverID)
	if err != nil {
		_ = writeJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		return
	}

	lines, err := h.launcher.Tail(serverID, 200)
	if err != nil {
		_ = writeJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		return
	}

	subscription, err := h.launcher.Subscribe(serverID)
	if err != nil {
		_ = writeJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		return
	}
	defer subscription.Close()

	// Drop half-open connections: the read loop below resets the deadline on
	// every pong, so a client that stops ponging will trip ReadJSON with an
	// i/o timeout and the handler unwinds cleanly.
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	canExecute, _ := conn.Locals("consoleCanExecute").(bool)

	if err := writeJSON(models.ServerConsoleEvent{
		Type:   "snapshot",
		Status: &status,
		Lines:  lines,
	}); err != nil {
		return
	}

	done := make(chan struct{})
	defer close(done)

	// Broadcast goroutine: forwards subscription events until the handler
	// returns (close(done)) or the subscription channel is closed.
	go func() {
		for {
			select {
			case <-done:
				return
			case event, ok := <-subscription.C:
				if !ok {
					return
				}
				if err := writeJSON(event); err != nil {
					return
				}
			}
		}
	}()

	// Ping goroutine: keeps NAT/proxy state warm and detects dead peers by
	// letting the pong handler reset the read deadline.
	go func() {
		ticker := time.NewTicker(wsPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := writePing(); err != nil {
					return
				}
			}
		}
	}()

	for {
		var inbound struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		}
		if err := conn.ReadJSON(&inbound); err != nil {
			return
		}

		if inbound.Type != "command" {
			continue
		}
		if !canExecute {
			_ = writeJSON(models.ServerConsoleEvent{Type: "error", Error: "missing console.execute permission"})
			continue
		}

		command := strings.TrimSpace(inbound.Command)
		if command == "" {
			continue
		}

		if err := h.launcher.SendCommand(serverID, command); err != nil {
			_ = writeJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		}
	}
}

func launcherHTTPError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, launcher.ErrServerNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, launcher.ErrAlreadyRunning):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, launcher.ErrNotRunning):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, launcher.ErrStopFailed):
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	default:
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
}

func parseTailCount(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 200
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 200
	}
	if n > 2000 {
		return 2000
	}
	return n
}

func canExecuteConsole(perms *permissions.ResolvedPermissions, serverID string) bool {
	if perms == nil {
		return false
	}
	if perms.IsOwner {
		return true
	}
	serverPerms, ok := perms.Servers[serverID]
	if !ok {
		return false
	}
	return serverPerms.Has(permissions.ServerConsole, permissions.ActionExecute)
}
