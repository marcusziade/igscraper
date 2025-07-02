package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 32
	keySize    = 32
	iterations = 100000
)

// EncryptedFileStore implements CredentialStore using an encrypted file
type EncryptedFileStore struct {
	filepath   string
	passphrase string
	mu         sync.RWMutex
}

// encryptedData represents the structure of the encrypted file
type encryptedData struct {
	Salt      string             `json:"salt"`
	Encrypted string             `json:"encrypted"`
	Accounts  map[string]Account `json:"-"` // Not serialized directly
}

// NewEncryptedFileStore creates a new encrypted file-based credential store
func NewEncryptedFileStore(filePath string) (*EncryptedFileStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	store := &EncryptedFileStore{
		filepath: filePath,
	}

	// Get or create passphrase
	passphrase, err := store.getPassphrase()
	if err != nil {
		return nil, fmt.Errorf("failed to get passphrase: %w", err)
	}
	store.passphrase = passphrase

	return store, nil
}

// Store saves credentials to the encrypted file
func (e *EncryptedFileStore) Store(account *Account) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if account == nil || account.Username == "" {
		return ErrInvalidCredentials
	}

	// Load existing data
	data, err := e.loadData()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing data: %w", err)
	}

	if data == nil {
		data = &encryptedData{
			Accounts: make(map[string]Account),
		}
	}

	// Update account
	data.Accounts[account.Username] = *account

	// Save data
	return e.saveData(data)
}

// Retrieve gets credentials from the encrypted file
func (e *EncryptedFileStore) Retrieve(username string) (*Account, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if username == "" {
		return nil, ErrInvalidCredentials
	}

	data, err := e.loadData()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCredentialsNotFound
		}
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	account, exists := data.Accounts[username]
	if !exists {
		return nil, ErrCredentialsNotFound
	}

	return &account, nil
}

// List returns all stored accounts
func (e *EncryptedFileStore) List() ([]*Account, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	data, err := e.loadData()
	if err != nil {
		if os.IsNotExist(err) {
			return []*Account{}, nil
		}
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	var accounts []*Account
	for _, account := range data.Accounts {
		acc := account // Create a copy
		accounts = append(accounts, &acc)
	}

	return accounts, nil
}

// Delete removes credentials from the encrypted file
func (e *EncryptedFileStore) Delete(username string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if username == "" {
		return ErrInvalidCredentials
	}

	data, err := e.loadData()
	if err != nil {
		if os.IsNotExist(err) {
			return ErrCredentialsNotFound
		}
		return fmt.Errorf("failed to load data: %w", err)
	}

	if _, exists := data.Accounts[username]; !exists {
		return ErrCredentialsNotFound
	}

	delete(data.Accounts, username)

	// If no accounts left, remove the file
	if len(data.Accounts) == 0 {
		return os.Remove(e.filepath)
	}

	return e.saveData(data)
}

// Exists checks if credentials exist
func (e *EncryptedFileStore) Exists(username string) bool {
	account, err := e.Retrieve(username)
	return err == nil && account != nil
}

// loadData loads and decrypts the data file
func (e *EncryptedFileStore) loadData() (*encryptedData, error) {
	// Read file
	content, err := os.ReadFile(e.filepath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var fileData struct {
		Salt      string `json:"salt"`
		Encrypted string `json:"encrypted"`
	}
	if err := json.Unmarshal(content, &fileData); err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Decode salt and encrypted data
	salt, err := base64.StdEncoding.DecodeString(fileData.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(fileData.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	// Derive key
	key := pbkdf2.Key([]byte(e.passphrase), salt, iterations, keySize, sha256.New)

	// Decrypt
	decrypted, err := decrypt(encryptedBytes, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Parse accounts
	var accounts map[string]Account
	if err := json.Unmarshal(decrypted, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse accounts: %w", err)
	}

	return &encryptedData{
		Salt:     fileData.Salt,
		Accounts: accounts,
	}, nil
}

// saveData encrypts and saves the data file
func (e *EncryptedFileStore) saveData(data *encryptedData) error {
	// Generate new salt if needed
	var salt []byte
	if data.Salt == "" {
		salt = make([]byte, saltSize)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		data.Salt = base64.StdEncoding.EncodeToString(salt)
	} else {
		var err error
		salt, err = base64.StdEncoding.DecodeString(data.Salt)
		if err != nil {
			return fmt.Errorf("failed to decode salt: %w", err)
		}
	}

	// Derive key
	key := pbkdf2.Key([]byte(e.passphrase), salt, iterations, keySize, sha256.New)

	// Marshal accounts
	accountsJSON, err := json.Marshal(data.Accounts)
	if err != nil {
		return fmt.Errorf("failed to marshal accounts: %w", err)
	}

	// Encrypt
	encrypted, err := encrypt(accountsJSON, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Prepare file data
	fileData := struct {
		Salt      string    `json:"salt"`
		Encrypted string    `json:"encrypted"`
		Version   int       `json:"version"`
		Modified  time.Time `json:"modified"`
	}{
		Salt:      data.Salt,
		Encrypted: base64.StdEncoding.EncodeToString(encrypted),
		Version:   1,
		Modified:  time.Now(),
	}

	// Marshal to JSON
	content, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file data: %w", err)
	}

	// Write to temporary file first
	tempFile := e.filepath + ".tmp"
	if err := os.WriteFile(tempFile, content, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Rename to final location
	return os.Rename(tempFile, e.filepath)
}

// getPassphrase retrieves or generates the passphrase for encryption
func (e *EncryptedFileStore) getPassphrase() (string, error) {
	// First check environment variable
	if pass := os.Getenv("IGSCRAPER_PASSPHRASE"); pass != "" {
		return pass, nil
	}

	// Check for passphrase file
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	passphraseFile := filepath.Join(configDir, ".passphrase")

	// Try to read existing passphrase
	if content, err := os.ReadFile(passphraseFile); err == nil && len(content) > 0 {
		return string(content), nil
	}

	// Generate new passphrase
	passphrase := generatePassphrase()

	// Save it
	if err := os.WriteFile(passphraseFile, []byte(passphrase), 0600); err != nil {
		return "", fmt.Errorf("failed to save passphrase: %w", err)
	}

	return passphrase, nil
}

// generatePassphrase generates a secure random passphrase
func generatePassphrase() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		// Fallback to less secure method
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// encrypt encrypts data using AES-GCM
func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt decrypts data using AES-GCM
func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
