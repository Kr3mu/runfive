// Package database provides connection and schema migration for the SQLite database.
//
// Opens a SQLite database via GORM and auto-migrates all domain models.
package database

import (
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"
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
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
