// Package database provides connection and schema migration for the SQLite database.
//
// Opens a SQLite database via GORM, auto-migrates all domain models,
// and seeds default RBAC roles on first run.
package database

import (
	"log"

	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"github.com/Kr3mu/runfive/internal/models"
)

var databaseConfig = &gorm.Config{}

// Connect opens the SQLite database and runs auto-migrations
// for all registered models.
func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("database.db?_journal_mode=WAL&_busy_timeout=5000"), databaseConfig)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.Invite{},
		&models.Role{},
		&models.UserServerRole{},
	)
	if err != nil {
		return nil, err
	}

	if err := seedDefaultRoles(db); err != nil {
		log.Printf("warning: failed to seed default roles: %v", err)
	}

	return db, nil
}

// seedDefaultRoles inserts the three default roles (Admin, Moderator, Viewer)
// if no roles exist yet. Idempotent: skips if any roles are present.
func seedDefaultRoles(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Role{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	roles := []models.Role{
		{
			Name:        "Admin",
			Description: "Full access to all panel and server features",
			Color:       "#ef4444",
			IsSystem:    true,
			Position:    0,
			GlobalPerms: `{"users":{"create":true,"read":true,"update":true,"delete":true},"roles":{"create":true,"read":true,"update":true,"delete":true},"servers":{"create":true,"delete":true},"settings":{"read":true,"update":true}}`,
			ServerPerms: `{"dashboard":{"read":true},"players":{"create":true,"read":true,"update":true,"delete":true,"kick":true,"warn":true},"console":{"read":true,"execute":true},"bans":{"create":true,"read":true,"update":true,"delete":true}}`,
		},
		{
			Name:        "Moderator",
			Description: "Manage players and bans, view console and settings",
			Color:       "#f59e0b",
			IsSystem:    true,
			Position:    1,
			GlobalPerms: `{"users":{"read":true},"roles":{"read":true}}`,
			ServerPerms: `{"dashboard":{"read":true},"players":{"read":true,"update":true,"kick":true,"warn":true},"console":{"read":true},"bans":{"create":true,"read":true,"update":true}}`,
		},
		{
			Name:        "Viewer",
			Description: "Read-only access to server resources",
			Color:       "#6b7280",
			IsSystem:    true,
			Position:    2,
			GlobalPerms: `{}`,
			ServerPerms: `{"dashboard":{"read":true},"players":{"read":true},"console":{"read":true},"bans":{"read":true}}`,
		},
	}

	for i := range roles {
		if err := db.Create(&roles[i]).Error; err != nil {
			return err
		}
	}

	log.Println("seeded default RBAC roles: Admin, Moderator, Viewer")
	return nil
}
