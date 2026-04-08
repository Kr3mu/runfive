// Package auth provides a GORM-backed session store implementing the same schema as SCS's gormstore.
//
// Sessions are stored in a "sessions" table with token (PK), data (BLOB),
// and expiry (TIMESTAMP). A background goroutine cleans up expired sessions.
package auth

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// cleanupInterval is how often expired sessions are purged from the database.
const cleanupInterval = 5 * time.Minute

// session is the GORM model matching SCS's gormstore schema.
type session struct {
	// Session token (SHA-256 hash of the raw cookie token)
	Token string `gorm:"column:token;primaryKey;type:varchar(64)"`
	// Encrypted session payload
	Data []byte `gorm:"column:data;not null"`
	// Absolute expiry timestamp
	Expiry time.Time `gorm:"column:expiry;index;not null"`
}

func (session) TableName() string {
	return "sessions"
}

// gormStore implements scs.Store using GORM for persistence.
type gormStore struct {
	db     *gorm.DB
	stopCh chan struct{}
}

// newGormStore creates the sessions table and starts background cleanup.
func newGormStore(db *gorm.DB) (*gormStore, error) {
	if err := db.AutoMigrate(&session{}); err != nil {
		return nil, fmt.Errorf("auto-migrate sessions: %w", err)
	}

	s := &gormStore{
		db:     db,
		stopCh: make(chan struct{}),
	}
	go s.cleanupLoop()
	return s, nil
}

// Find retrieves session data by token.
func (s *gormStore) Find(token string) ([]byte, bool, error) {
	var sess session
	result := s.db.Where("token = ? AND expiry > ?", token, time.Now()).First(&sess)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, result.Error
	}
	return sess.Data, true, nil
}

// Commit creates or updates a session in the store.
func (s *gormStore) Commit(token string, b []byte, expiry time.Time) error {
	sess := session{
		Token:  token,
		Data:   b,
		Expiry: expiry,
	}
	result := s.db.Save(&sess)
	return result.Error
}

// Delete removes a session from the store.
func (s *gormStore) Delete(token string) error {
	return s.db.Where("token = ?", token).Delete(&session{}).Error
}

// All returns all non-expired sessions. Implements scs.IterableStore.
func (s *gormStore) All() (map[string][]byte, error) {
	var sessions []session
	if err := s.db.Where("expiry > ?", time.Now()).Find(&sessions).Error; err != nil {
		return nil, err
	}
	result := make(map[string][]byte, len(sessions))
	for _, sess := range sessions {
		result[sess.Token] = sess.Data
	}
	return result, nil
}

// StopCleanup stops the background expired-session cleanup goroutine.
func (s *gormStore) StopCleanup() {
	close(s.stopCh)
}

func (s *gormStore) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.db.Where("expiry <= ?", time.Now()).Delete(&session{})
		case <-s.stopCh:
			return
		}
	}
}
