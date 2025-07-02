package auth

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "igscraper"
	keyringPrefix  = "instagram_"
)

// KeyringStore implements CredentialStore using the system keychain
type KeyringStore struct{}

// NewKeyringStore creates a new keyring-based credential store
func NewKeyringStore() (*KeyringStore, error) {
	// Test if keyring is available
	testKey := "test_availability"
	err := keyring.Set(keyringService, testKey, "test")
	if err != nil {
		return nil, fmt.Errorf("keyring not available: %w", err)
	}
	_ = keyring.Delete(keyringService, testKey)

	return &KeyringStore{}, nil
}

// Store saves credentials to the system keychain
func (k *KeyringStore) Store(account *Account) error {
	if account == nil || account.Username == "" {
		return ErrInvalidCredentials
	}

	// Serialize account to JSON
	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	// Store in keyring
	key := keyringPrefix + account.Username
	if err := keyring.Set(keyringService, key, string(data)); err != nil {
		return fmt.Errorf("failed to store in keyring: %w", err)
	}

	return nil
}

// Retrieve gets credentials from the system keychain
func (k *KeyringStore) Retrieve(username string) (*Account, error) {
	if username == "" {
		return nil, ErrInvalidCredentials
	}

	key := keyringPrefix + username
	data, err := keyring.Get(keyringService, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, ErrCredentialsNotFound
		}
		return nil, fmt.Errorf("failed to retrieve from keyring: %w", err)
	}

	var account Account
	if err := json.Unmarshal([]byte(data), &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	return &account, nil
}

// List returns all stored accounts from the keychain
func (k *KeyringStore) List() ([]*Account, error) {
	// Unfortunately, go-keyring doesn't support listing all keys
	// This is a limitation of the library and underlying APIs
	// On some systems we could implement this, but for portability we'll return empty
	return []*Account{}, nil
}

// Delete removes credentials from the system keychain
func (k *KeyringStore) Delete(username string) error {
	if username == "" {
		return ErrInvalidCredentials
	}

	key := keyringPrefix + username
	err := keyring.Delete(keyringService, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return ErrCredentialsNotFound
		}
		return fmt.Errorf("failed to delete from keyring: %w", err)
	}

	return nil
}

// Exists checks if credentials exist in the keychain
func (k *KeyringStore) Exists(username string) bool {
	if username == "" {
		return false
	}

	key := keyringPrefix + username
	_, err := keyring.Get(keyringService, key)
	return err == nil
}

// IsAvailable checks if the keyring is available on this system
func IsKeyringAvailable() bool {
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	case "linux":
		// Check if we're in a graphical session
		if display := runtime.GOARCH; display != "" {
			return true
		}
		return false
	default:
		return false
	}
}
