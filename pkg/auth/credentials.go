package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Account represents an Instagram account's credentials
type Account struct {
	Username     string    `json:"username"`
	SessionID    string    `json:"session_id"`
	CSRFToken    string    `json:"csrf_token"`
	UserAgent    string    `json:"user_agent,omitempty"`
	LastModified time.Time `json:"last_modified"`
}

// CredentialStore is the interface for storing and retrieving credentials
type CredentialStore interface {
	// Store saves credentials for a given account
	Store(account *Account) error

	// Retrieve gets credentials for a specific username
	Retrieve(username string) (*Account, error)

	// List returns all stored accounts
	List() ([]*Account, error)

	// Delete removes credentials for a specific username
	Delete(username string) error

	// Exists checks if credentials exist for a username
	Exists(username string) bool
}

// Manager handles credential storage with fallback mechanisms
type Manager struct {
	stores []CredentialStore
}

// NewManager creates a new credential manager with appropriate storage backends
func NewManager() (*Manager, error) {
	var stores []CredentialStore

	// Try keyring first (system keychain)
	keyringStore, err := NewKeyringStore()
	if err == nil {
		stores = append(stores, keyringStore)
	}

	// Always add encrypted file store as fallback
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	encryptedStore, err := NewEncryptedFileStore(filepath.Join(configDir, "credentials.enc"))
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted store: %w", err)
	}
	stores = append(stores, encryptedStore)

	// Add environment store as last resort
	stores = append(stores, NewEnvironmentStore())

	return &Manager{stores: stores}, nil
}

// Store saves credentials using the first available store
func (m *Manager) Store(account *Account) error {
	if account.Username == "" {
		return errors.New("username is required")
	}
	if account.SessionID == "" {
		return errors.New("session ID is required")
	}
	if account.CSRFToken == "" {
		return errors.New("CSRF token is required")
	}

	account.LastModified = time.Now()

	// Try each store in order
	var lastErr error
	for _, store := range m.stores {
		if err := store.Store(account); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to store credentials: %w", lastErr)
	}
	return errors.New("no available credential stores")
}

// Retrieve gets credentials from the first store that has them
func (m *Manager) Retrieve(username string) (*Account, error) {
	for _, store := range m.stores {
		if account, err := store.Retrieve(username); err == nil && account != nil {
			return account, nil
		}
	}
	return nil, fmt.Errorf("credentials not found for user: %s", username)
}

// RetrieveDefault gets credentials for the default account or the first available
func (m *Manager) RetrieveDefault() (*Account, error) {
	// First try to get from environment (for backward compatibility)
	if envStore, ok := m.stores[len(m.stores)-1].(*EnvironmentStore); ok {
		if account, err := envStore.Retrieve(""); err == nil && account != nil {
			return account, nil
		}
	}

	// Then try to get the first available account
	accounts, err := m.List()
	if err == nil && len(accounts) > 0 {
		return accounts[0], nil
	}

	return nil, errors.New("no credentials found")
}

// List returns all stored accounts from all stores
func (m *Manager) List() ([]*Account, error) {
	accountMap := make(map[string]*Account)

	for _, store := range m.stores {
		accounts, err := store.List()
		if err != nil {
			continue
		}
		for _, account := range accounts {
			// Use the most recently modified version
			if existing, ok := accountMap[account.Username]; !ok || account.LastModified.After(existing.LastModified) {
				accountMap[account.Username] = account
			}
		}
	}

	var result []*Account
	for _, account := range accountMap {
		result = append(result, account)
	}

	return result, nil
}

// Delete removes credentials from all stores
func (m *Manager) Delete(username string) error {
	var deleted bool
	var lastErr error

	for _, store := range m.stores {
		if err := store.Delete(username); err == nil {
			deleted = true
		} else {
			lastErr = err
		}
	}

	if !deleted && lastErr != nil {
		return fmt.Errorf("failed to delete credentials: %w", lastErr)
	}
	if !deleted {
		return fmt.Errorf("credentials not found for user: %s", username)
	}

	return nil
}

// DeleteAll removes all stored credentials
func (m *Manager) DeleteAll() error {
	accounts, err := m.List()
	if err != nil {
		return err
	}

	for _, account := range accounts {
		_ = m.Delete(account.Username) // Ignore individual errors
	}

	return nil
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "igscraper")
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "igscraper")
	default: // Linux and others
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			configDir = filepath.Join(xdgConfig, "igscraper")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, ".config", "igscraper")
		}
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// SanitizeAccount creates a copy of the account with sensitive data masked
func SanitizeAccount(account *Account) *Account {
	if account == nil {
		return nil
	}

	return &Account{
		Username:     account.Username,
		SessionID:    maskString(account.SessionID),
		CSRFToken:    maskString(account.CSRFToken),
		UserAgent:    account.UserAgent,
		LastModified: account.LastModified,
	}
}

// maskString masks all but the first 4 and last 4 characters of a string
func maskString(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// Errors
var (
	ErrCredentialsNotFound = errors.New("credentials not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrStoreUnavailable    = errors.New("credential store unavailable")
)
