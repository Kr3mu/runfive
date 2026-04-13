package v1

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/Kr3mu/runfive/internal/permissions"
)

// RoleHandler groups role management HTTP handlers.
type RoleHandler struct {
	db *gorm.DB
}

// NewRoleHandler creates the role handler.
func NewRoleHandler(db *gorm.DB) *RoleHandler {
	return &RoleHandler{db: db}
}

// List returns all roles with assignment counts.
// Requires global "roles.read" permission (enforced by middleware).
//
// GET /v1/roles
func (h *RoleHandler) List(c fiber.Ctx) error {
	var roles []models.Role
	if err := h.db.Order("position ASC, id ASC").Find(&roles).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	response := make([]models.RoleListItem, 0, len(roles))
	for i := range roles {
		r := &roles[i]
		item := models.RoleListItem{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Color:       r.Color,
			IsSystem:    r.IsSystem,
			Position:    r.Position,
		}

		globalPerms, _ := permissions.Parse(r.GlobalPerms)
		item.GlobalPerms = globalPerms
		serverPerms, _ := permissions.Parse(r.ServerPerms)
		item.ServerPerms = serverPerms

		var globalCount int64
		h.db.Model(&models.User{}).Where("global_role_id = ?", r.ID).Count(&globalCount)
		var serverCount int64
		h.db.Model(&models.UserServerRole{}).Where("role_id = ?", r.ID).Count(&serverCount)
		item.AssignedUsers = int(globalCount + serverCount)

		response = append(response, item)
	}

	return c.JSON(response)
}

// Get returns a single role by ID.
// Requires global "roles.read" permission (enforced by middleware).
//
// GET /v1/roles/:id
func (h *RoleHandler) Get(c fiber.Ctx) error {
	roleID := fiber.Params[uint](c, "id")
	if roleID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid role id")
	}

	var role models.Role
	if err := h.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "role not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	globalPerms, _ := permissions.Parse(role.GlobalPerms)
	serverPerms, _ := permissions.Parse(role.ServerPerms)

	var globalCount int64
	h.db.Model(&models.User{}).Where("global_role_id = ?", role.ID).Count(&globalCount)
	var serverCount int64
	h.db.Model(&models.UserServerRole{}).Where("role_id = ?", role.ID).Count(&serverCount)

	return c.JSON(models.RoleListItem{
		ID:            role.ID,
		Name:          role.Name,
		Description:   role.Description,
		Color:         role.Color,
		GlobalPerms:   globalPerms,
		ServerPerms:   serverPerms,
		IsSystem:      role.IsSystem,
		Position:      role.Position,
		AssignedUsers: int(globalCount + serverCount),
	})
}

// Create creates a new role.
// Requires global "roles.create" permission (enforced by middleware).
// Privilege escalation guard: non-owner cannot grant permissions they don't have.
//
// POST /v1/roles
func (h *RoleHandler) Create(c fiber.Ctx) error {
	var req models.CreateRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || len(req.Name) > 64 {
		return fiber.NewError(fiber.StatusBadRequest, "name must be 1-64 characters")
	}
	if len(req.Color) != 7 || req.Color[0] != '#' {
		return fiber.NewError(fiber.StatusBadRequest, "color must be a 7-character hex code")
	}

	if err := validateEscalation(c, req.GlobalPerms, req.ServerPerms); err != nil {
		return err
	}

	globalJSON, err := marshalPerms(req.GlobalPerms)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid global permissions")
	}
	serverJSON, err := marshalPerms(req.ServerPerms)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server permissions")
	}

	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		GlobalPerms: globalJSON,
		ServerPerms: serverJSON,
	}

	if err := h.db.Create(&role).Error; err != nil {
		return fiber.NewError(fiber.StatusConflict, "role name already taken")
	}

	return c.Status(fiber.StatusCreated).JSON(role)
}

// Update modifies an existing role.
// Requires global "roles.update" permission (enforced by middleware).
// Privilege escalation guard: non-owner cannot grant permissions they don't have.
//
// PUT /v1/roles/:id
func (h *RoleHandler) Update(c fiber.Ctx) error {
	roleID := fiber.Params[uint](c, "id")
	if roleID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid role id")
	}

	var role models.Role
	if err := h.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "role not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	var req models.UpdateRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updates := map[string]interface{}{}

	if req.Name != nil {
		if *req.Name == "" || len(*req.Name) > 64 {
			return fiber.NewError(fiber.StatusBadRequest, "name must be 1-64 characters")
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Color != nil {
		if len(*req.Color) != 7 || (*req.Color)[0] != '#' {
			return fiber.NewError(fiber.StatusBadRequest, "color must be a 7-character hex code")
		}
		updates["color"] = *req.Color
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}

	if req.GlobalPerms != nil {
		if err := validateEscalation(c, *req.GlobalPerms, nil); err != nil {
			return err
		}
		j, err := marshalPerms(*req.GlobalPerms)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid global permissions")
		}
		updates["global_perms"] = j
	}
	if req.ServerPerms != nil {
		if err := validateEscalation(c, nil, *req.ServerPerms); err != nil {
			return err
		}
		j, err := marshalPerms(*req.ServerPerms)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid server permissions")
		}
		updates["server_perms"] = j
	}

	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}

	if err := h.db.Model(&role).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Delete removes a role. Fails if it is a system role or assigned to any user.
// Requires global "roles.delete" permission (enforced by middleware).
//
// DELETE /v1/roles/:id
func (h *RoleHandler) Delete(c fiber.Ctx) error {
	roleID := fiber.Params[uint](c, "id")
	if roleID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid role id")
	}

	var role models.Role
	if err := h.db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "role not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if role.IsSystem {
		return fiber.NewError(fiber.StatusForbidden, "cannot delete system role")
	}

	var globalCount int64
	h.db.Model(&models.User{}).Where("global_role_id = ?", roleID).Count(&globalCount)
	var serverCount int64
	h.db.Model(&models.UserServerRole{}).Where("role_id = ?", roleID).Count(&serverCount)
	if globalCount+serverCount > 0 {
		return fiber.NewError(fiber.StatusConflict, "cannot delete role that is assigned to users")
	}

	if err := h.db.Unscoped().Delete(&role).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// PermissionSchema returns the canonical list of resources and their actions.
// Used by the frontend role editor to build the permission matrix.
//
// GET /v1/permissions/schema
func PermissionSchema(c fiber.Ctx) error {
	global := make(map[string]models.ResourceSchemaDTO, len(permissions.GlobalResourceSchema))
	for k, v := range permissions.GlobalResourceSchema {
		global[k] = models.ResourceSchemaDTO{CRUD: v.CRUD, Sub: v.Sub}
	}
	server := make(map[string]models.ResourceSchemaDTO, len(permissions.ServerResourceSchema))
	for k, v := range permissions.ServerResourceSchema {
		server[k] = models.ResourceSchemaDTO{CRUD: v.CRUD, Sub: v.Sub}
	}
	return c.JSON(models.PermissionSchemaResponse{Global: global, Server: server})
}

// validateEscalation checks that a non-owner caller is not granting permissions
// they don't have themselves.
func validateEscalation(c fiber.Ctx, globalPerms, serverPerms permissions.PermissionMap) error {
	perms := auth.GetPermissions(c)
	if perms == nil || perms.IsOwner {
		return nil
	}

	if globalPerms != nil && !globalPerms.IsSubsetOf(perms.Global) {
		return fiber.NewError(fiber.StatusForbidden, "cannot grant permissions you don't have")
	}

	if serverPerms != nil {
		callerServerUnion := mergeServerPerms(perms.Servers)
		if !serverPerms.IsSubsetOf(callerServerUnion) {
			return fiber.NewError(fiber.StatusForbidden, "cannot grant permissions you don't have")
		}
	}

	return nil
}

// marshalPerms serializes a permission map to JSON string.
func marshalPerms(pm map[string]map[string]bool) (string, error) {
	if pm == nil {
		return "{}", nil
	}
	b, err := json.Marshal(pm)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
