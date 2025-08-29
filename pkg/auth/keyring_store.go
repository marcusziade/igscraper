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
	keyringIndex   = "accounts_index"
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

	// Update the accounts index
	if err := k.updateIndex(account.Username, true); err != nil {
		// Log the error but don't fail the operation
		// The account is stored, just the index update failed
		fmt.Printf("Warning: failed to update accounts index: %v\n", err)
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
	// Get the list of usernames from the index
	usernames, err := k.getIndex()
	if err != nil {
		// If we can't read the index, fall back to empty list
		// This maintains backward compatibility
		return []*Account{}, nil
	}

	var accounts []*Account
	for _, username := range usernames {
		account, err := k.Retrieve(username)
		if err != nil {
			// Skip accounts that can't be retrieved
			continue
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
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

	// Update the accounts index
	if err := k.updateIndex(username, false); err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("Warning: failed to update accounts index: %v\n", err)
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

// updateIndex maintains an index of stored account usernames
func (k *KeyringStore) updateIndex(username string, add bool) error {
	usernames, err := k.getIndex()
	if err != nil {
		// If we can't read the index, start with empty list
		usernames = []string{}
	}

	// Find and remove existing username
	for i, u := range usernames {
		if u == username {
			usernames = append(usernames[:i], usernames[i+1:]...)
			break
		}
	}

	// Add username if requested
	if add {
		usernames = append(usernames, username)
	}

	// Store updated index
	indexData, err := json.Marshal(usernames)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := keyring.Set(keyringService, keyringIndex, string(indexData)); err != nil {
		return fmt.Errorf("failed to store index: %w", err)
	}

	return nil
}

// getIndex retrieves the list of stored account usernames
func (k *KeyringStore) getIndex() ([]string, error) {
	indexData, err := keyring.Get(keyringService, keyringIndex)
	if err != nil {
		if err == keyring.ErrNotFound {
			// No index exists yet
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to retrieve index: %w", err)
	}

	var usernames []string
	if err := json.Unmarshal([]byte(indexData), &usernames); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return usernames, nil
}
