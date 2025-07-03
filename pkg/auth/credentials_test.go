package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCredentialManager(t *testing.T) {
	// Use mock manager for reliable testing
	manager, mockStore := NewMockManager()

	// Test storing credentials
	account := &Account{
		Username:     "testuser",
		SessionID:    "test_session_id_12345",
		CSRFToken:    "test_csrf_token_67890",
		UserAgent:    "TestAgent/1.0",
		LastModified: time.Now(),
	}

	err := manager.Store(account)
	if err != nil {
		t.Errorf("Failed to store account: %v", err)
	}

	// Test retrieving credentials
	retrieved, err := manager.Retrieve("testuser")
	if err != nil {
		t.Errorf("Failed to retrieve account: %v", err)
	}

	if retrieved.Username != account.Username {
		t.Errorf("Username mismatch: got %s, want %s", retrieved.Username, account.Username)
	}
	if retrieved.SessionID != account.SessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", retrieved.SessionID, account.SessionID)
	}
	if retrieved.CSRFToken != account.CSRFToken {
		t.Errorf("CSRFToken mismatch: got %s, want %s", retrieved.CSRFToken, account.CSRFToken)
	}

	// Test listing accounts
	accounts, err := manager.List()
	if err != nil {
		t.Errorf("Failed to list accounts: %v", err)
	}
	if len(accounts) == 0 {
		t.Error("Expected at least one account in list")
	}

	// Test sanitization
	sanitized := SanitizeAccount(account)
	if sanitized.SessionID == account.SessionID {
		t.Error("SessionID should be masked")
	}
	if sanitized.CSRFToken == account.CSRFToken {
		t.Error("CSRFToken should be masked")
	}
	if sanitized.Username != account.Username {
		t.Error("Username should not be masked")
	}

	// Test deletion
	err = manager.Delete("testuser")
	if err != nil {
		t.Errorf("Failed to delete account: %v", err)
	}

	// Verify deletion
	_, err = manager.Retrieve("testuser")
	if err == nil {
		t.Error("Expected error retrieving deleted account")
	}
	
	// Verify mock store state
	if mockStore.Count() != 0 {
		t.Errorf("Expected 0 accounts after deletion, got %d", mockStore.Count())
	}
}

func TestEncryptedFileStore(t *testing.T) {
	// Create a temporary file
	tempFile := filepath.Join(os.TempDir(), "test_creds.enc")
	defer os.Remove(tempFile)

	// Set test passphrase
	os.Setenv("IGSCRAPER_PASSPHRASE", "test_passphrase_123")
	defer os.Unsetenv("IGSCRAPER_PASSPHRASE")

	// Create store
	store, err := NewEncryptedFileStore(tempFile)
	if err != nil {
		t.Fatalf("Failed to create encrypted store: %v", err)
	}

	// Test operations
	account := &Account{
		Username:  "encrypted_user",
		SessionID: "encrypted_session",
		CSRFToken: "encrypted_csrf",
	}

	// Store
	err = store.Store(account)
	if err != nil {
		t.Errorf("Failed to store in encrypted file: %v", err)
	}

	// Retrieve
	retrieved, err := store.Retrieve("encrypted_user")
	if err != nil {
		t.Errorf("Failed to retrieve from encrypted file: %v", err)
	}

	if retrieved.SessionID != account.SessionID {
		t.Errorf("SessionID mismatch after encryption/decryption")
	}

	// Verify file is actually encrypted
	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	// File should not contain plaintext credentials
	if contains(fileContent, []byte("encrypted_session")) {
		t.Error("File contains plaintext session ID")
	}
	if contains(fileContent, []byte("encrypted_csrf")) {
		t.Error("File contains plaintext CSRF token")
	}
}

func TestEnvironmentStore(t *testing.T) {
	// Set environment variables
	os.Setenv("IGSCRAPER_SESSION_ID", "env_session")
	os.Setenv("IGSCRAPER_CSRF_TOKEN", "env_csrf")
	defer os.Unsetenv("IGSCRAPER_SESSION_ID")
	defer os.Unsetenv("IGSCRAPER_CSRF_TOKEN")

	store := NewEnvironmentStore()

	// Test retrieve
	account, err := store.Retrieve("")
	if err != nil {
		t.Errorf("Failed to retrieve from environment: %v", err)
	}

	if account.SessionID != "env_session" {
		t.Errorf("SessionID mismatch: got %s, want env_session", account.SessionID)
	}
	if account.CSRFToken != "env_csrf" {
		t.Errorf("CSRFToken mismatch: got %s, want env_csrf", account.CSRFToken)
	}

	// Test that store is not supported
	err = store.Store(&Account{})
	if err != ErrStoreUnavailable {
		t.Error("Expected ErrStoreUnavailable for environment store")
	}
}

func TestRealManagerWithEncryptedStore(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "igscraper-test-real")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Set passphrase for testing
	os.Setenv("IGSCRAPER_PASSPHRASE", "test_passphrase_real_manager")
	defer os.Unsetenv("IGSCRAPER_PASSPHRASE")

	// Create manager with only encrypted file store (most reliable for testing)
	encryptedStore, err := NewEncryptedFileStore(filepath.Join(tempDir, "credentials.enc"))
	if err != nil {
		t.Fatalf("Failed to create encrypted store: %v", err)
	}
	
	manager := NewMockManagerWithStores(encryptedStore)

	// Test storing credentials
	account := &Account{
		Username:     "realuser",
		SessionID:    "real_session_id",
		CSRFToken:    "real_csrf_token",
		UserAgent:    "RealAgent/1.0",
		LastModified: time.Now(),
	}

	err = manager.Store(account)
	if err != nil {
		t.Fatalf("Failed to store account: %v", err)
	}

	// Test listing accounts
	accounts, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account in list, got %d", len(accounts))
	}

	// Test retrieving credentials
	retrieved, err := manager.Retrieve("realuser")
	if err != nil {
		t.Fatalf("Failed to retrieve account: %v", err)
	}

	if retrieved.Username != account.Username {
		t.Errorf("Username mismatch: got %s, want %s", retrieved.Username, account.Username)
	}
	if retrieved.SessionID != account.SessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", retrieved.SessionID, account.SessionID)
	}
}

func TestMockStore(t *testing.T) {
	store := NewMockStore()

	// Test empty store
	accounts, err := store.List()
	if err != nil {
		t.Errorf("Failed to list empty store: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(accounts))
	}

	// Test storing and retrieving
	account := &Account{
		Username:  "mockuser",
		SessionID: "mock_session",
		CSRFToken: "mock_csrf",
	}

	err = store.Store(account)
	if err != nil {
		t.Errorf("Failed to store account: %v", err)
	}

	// Verify count
	if store.Count() != 1 {
		t.Errorf("Expected 1 account, got %d", store.Count())
	}

	// Test exists
	if !store.Exists("mockuser") {
		t.Error("Account should exist")
	}

	// Test error injection
	store.ListError = fmt.Errorf("injected error")
	_, err = store.List()
	if err == nil || err.Error() != "injected error" {
		t.Error("Expected injected error")
	}
}

func contains(data []byte, substr []byte) bool {
	for i := 0; i <= len(data)-len(substr); i++ {
		if string(data[i:i+len(substr)]) == string(substr) {
			return true
		}
	}
	return false
}
