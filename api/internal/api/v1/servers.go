package v1

import (
	"context"
	"strings"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
)

type serverRegistry interface {
	List() ([]models.ManagedServer, error)
	Create(name, artifactVersion string) (models.ManagedServer, error)
}

type serverArtifactManager interface {
	Install(context.Context, string) (models.InstalledArtifact, error)
}

// ServerHandler serves filesystem-backed managed server endpoints.
type ServerHandler struct {
	registry  serverRegistry
	artifacts serverArtifactManager
}

// NewServerHandler creates a server handler with its dependencies.
func NewServerHandler(registry serverRegistry, artifacts serverArtifactManager) *ServerHandler {
	return &ServerHandler{registry: registry, artifacts: artifacts}
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
	if perms == nil || perms.IsOwner {
		return c.JSON(servers)
	}

	filtered := make([]models.ManagedServer, 0, len(servers))
	for _, server := range servers {
		if _, ok := perms.Servers[server.ID]; ok {
			filtered = append(filtered, server)
		}
	}

	return c.JSON(filtered)
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
