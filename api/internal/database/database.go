package database

import (
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/libtnb/sqlite"

	"gorm.io/gorm"
)

var databaseConfig = &gorm.Config{}

func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("database.db"), databaseConfig)

	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&models.User{})

	return db, nil
}
