// Package auth provides password hashing and comparison using bcrypt.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword produces a bcrypt hash of the given plaintext password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
// Returns true if the password matches the hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
