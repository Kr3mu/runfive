package v1

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/Kr3mu/runfive/internal/preferences"
)

// PreferenceHandler groups per-user preference HTTP handlers.
type PreferenceHandler struct {
	db *gorm.DB
}

// NewPreferenceHandler creates the preference handler with its dependencies.
func NewPreferenceHandler(db *gorm.DB) *PreferenceHandler {
	return &PreferenceHandler{db: db}
}

// Get returns the stored value for the given key, scoped to the current user.
// Responds 404 if the user has no entry for this key.
//
// GET /v1/preferences/:key
func (h *PreferenceHandler) Get(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	key := c.Params("key")
	if !preferences.IsAllowed(key) {
		return fiber.NewError(fiber.StatusBadRequest, "unknown preference key")
	}

	var pref models.UserPreference
	err := h.db.Where("user_id = ? AND key = ?", user.ID, key).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "preference not set")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.JSON(models.PreferenceResponse{Key: pref.Key, Value: pref.Value})
}

// Put upserts the value for the given key, scoped to the current user.
//
// PUT /v1/preferences/:key
func (h *PreferenceHandler) Put(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	key := c.Params("key")
	if !preferences.IsAllowed(key) {
		return fiber.NewError(fiber.StatusBadRequest, "unknown preference key")
	}

	var req models.PreferenceUpdateRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if len(req.Value) > preferences.MaxValueBytes {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "preference value too large")
	}

	pref := models.UserPreference{
		UserID: user.ID,
		Key:    key,
		Value:  req.Value,
	}
	// Upsert on the (user_id, key) unique index.
	err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&pref).Error
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.JSON(models.PreferenceResponse{Key: pref.Key, Value: pref.Value})
}

// Delete removes the stored value for the given key, scoped to the current
// user. Responds 204 whether or not a row existed (idempotent).
//
// DELETE /v1/preferences/:key
func (h *PreferenceHandler) Delete(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	key := c.Params("key")
	if !preferences.IsAllowed(key) {
		return fiber.NewError(fiber.StatusBadRequest, "unknown preference key")
	}

	if err := h.db.Where("user_id = ? AND key = ?", user.ID, key).
		Delete(&models.UserPreference{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
