package v1

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
)

type artifactManager interface {
	HostOS() string
	ListInstalled() ([]models.InstalledArtifact, error)
	ListAvailable(context.Context) ([]models.AvailableArtifactVersion, error)
	Install(context.Context, string) (models.InstalledArtifact, error)
	Delete(version string) error
}

type artifactReferenceRegistry interface {
	ArtifactReferences(version string) ([]string, error)
}

// ArtifactHandler serves shared artifact management endpoints.
type ArtifactHandler struct {
	manager  artifactManager
	registry artifactReferenceRegistry
}

// NewArtifactHandler creates an artifact handler with its dependencies.
func NewArtifactHandler(manager artifactManager, registry artifactReferenceRegistry) *ArtifactHandler {
	return &ArtifactHandler{manager: manager, registry: registry}
}

// List returns installed and upstream artifact versions for the host OS.
//
// GET /v1/artifacts
func (h *ArtifactHandler) List(c fiber.Ctx) error {
	installed, err := h.manager.ListInstalled()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	available, err := h.manager.ListAvailable(context.Background())
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}

	return c.JSON(models.ArtifactListResponse{
		OS:        h.manager.HostOS(),
		Installed: installed,
		Available: available,
	})
}

// Download installs a specific artifact version if needed.
//
// POST /v1/artifacts/download
func (h *ArtifactHandler) Download(c fiber.Ctx) error {
	var req models.DownloadArtifactRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	req.Version = strings.TrimSpace(req.Version)
	if req.Version == "" {
		return fiber.NewError(fiber.StatusBadRequest, "artifact version is required")
	}

	artifact, err := h.manager.Install(context.Background(), req.Version)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(artifact)
}

// Delete removes an unreferenced artifact version.
//
// DELETE /v1/artifacts/:version
func (h *ArtifactHandler) Delete(c fiber.Ctx) error {
	version := strings.TrimSpace(c.Params("version"))
	if version == "" {
		return fiber.NewError(fiber.StatusBadRequest, "artifact version is required")
	}

	refs, err := h.registry.ArtifactReferences(version)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if len(refs) > 0 {
		return fiber.NewError(fiber.StatusConflict, "artifact is still referenced by one or more servers")
	}

	if err := h.manager.Delete(version); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fiber.NewError(fiber.StatusNotFound, "artifact not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}
