package v1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	ws "github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"

	"github.com/runfivedev/runfive/internal/auth"
	"github.com/runfivedev/runfive/internal/launcher"
	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/permissions"
)

type serverRegistry interface {
	List() ([]models.ManagedServer, error)
	Create(name, artifactVersion string) (models.ManagedServer, error)
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
func NewServerHandler(registry serverRegistry, artifacts serverArtifactManager, launcher serverLauncher) *ServerHandler {
	return &ServerHandler{registry: registry, artifacts: artifacts, launcher: launcher}
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
		for _, server := range servers {
			if _, ok := perms.Servers[server.ID]; ok {
				filtered = append(filtered, server)
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

	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "server name is required")
	}
	if req.ArtifactVersion == "" {
		return fiber.NewError(fiber.StatusBadRequest, "artifact version is required")
	}

	if _, err := h.artifacts.Install(context.Background(), req.ArtifactVersion); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	server, err := h.registry.Create(req.Name, req.ArtifactVersion)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
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

	status, err := h.launcher.Status(serverID)
	if err != nil {
		_ = conn.WriteJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		return
	}

	lines, err := h.launcher.Tail(serverID, 200)
	if err != nil {
		_ = conn.WriteJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		return
	}

	subscription, err := h.launcher.Subscribe(serverID)
	if err != nil {
		_ = conn.WriteJSON(models.ServerConsoleEvent{Type: "error", Error: err.Error()})
		return
	}
	defer subscription.Close()

	canExecute, _ := conn.Locals("consoleCanExecute").(bool)

	var writeMu sync.Mutex
	writeJSON := func(payload any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(payload)
	}

	if err := writeJSON(models.ServerConsoleEvent{
		Type:   "snapshot",
		Status: &status,
		Lines:  lines,
	}); err != nil {
		return
	}

	done := make(chan struct{})
	defer close(done)

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
