package auth

import (
	"os"
	"time"
)

// EnvironmentStore implements CredentialStore using environment variables
// This is primarily for backward compatibility
type EnvironmentStore struct{}

// NewEnvironmentStore creates a new environment-based credential store
func NewEnvironmentStore() *EnvironmentStore {
	return &EnvironmentStore{}
}

// Store is not supported for environment variables
func (e *EnvironmentStore) Store(account *Account) error {
	return ErrStoreUnavailable
}

// Retrieve gets credentials from environment variables
func (e *EnvironmentStore) Retrieve(username string) (*Account, error) {
	sessionID := os.Getenv("IGSCRAPER_SESSION_ID")
	csrfToken := os.Getenv("IGSCRAPER_CSRF_TOKEN")
	userAgent := os.Getenv("IGSCRAPER_USER_AGENT")

	if sessionID == "" || csrfToken == "" {
		return nil, ErrCredentialsNotFound
	}

	// Environment variables don't store username, so we use "default" or the provided one
	if username == "" {
		username = "default"
	}

	return &Account{
		Username:     username,
		SessionID:    sessionID,
		CSRFToken:    csrfToken,
		UserAgent:    userAgent,
		LastModified: time.Now(),
	}, nil
}

// List returns a single account if environment variables are set
func (e *EnvironmentStore) List() ([]*Account, error) {
	account, err := e.Retrieve("")
	if err != nil {
		return []*Account{}, nil
	}
	return []*Account{account}, nil
}

// Delete is not supported for environment variables
func (e *EnvironmentStore) Delete(username string) error {
	return ErrStoreUnavailable
}

// Exists checks if environment credentials exist
func (e *EnvironmentStore) Exists(username string) bool {
	sessionID := os.Getenv("IGSCRAPER_SESSION_ID")
	csrfToken := os.Getenv("IGSCRAPER_CSRF_TOKEN")
	return sessionID != "" && csrfToken != ""
}
