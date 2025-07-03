package auth

import (
	"fmt"
	"sync"
)

// MockStore implements CredentialStore for testing purposes
type MockStore struct {
	accounts map[string]*Account
	mu       sync.RWMutex
	
	// Error injection for testing
	StoreError    error
	RetrieveError error
	ListError     error
	DeleteError   error
}

// NewMockStore creates a new mock credential store
func NewMockStore() *MockStore {
	return &MockStore{
		accounts: make(map[string]*Account),
	}
}

// Store saves credentials to the mock store
func (m *MockStore) Store(account *Account) error {
	if m.StoreError != nil {
		return m.StoreError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if account == nil || account.Username == "" {
		return ErrInvalidCredentials
	}
	
	// Create a copy to avoid external modifications
	accountCopy := *account
	m.accounts[account.Username] = &accountCopy
	
	return nil
}

// Retrieve gets credentials from the mock store
func (m *MockStore) Retrieve(username string) (*Account, error) {
	if m.RetrieveError != nil {
		return nil, m.RetrieveError
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if username == "" {
		return nil, ErrInvalidCredentials
	}
	
	account, exists := m.accounts[username]
	if !exists {
		return nil, ErrCredentialsNotFound
	}
	
	// Return a copy to avoid external modifications
	accountCopy := *account
	return &accountCopy, nil
}

// List returns all stored accounts from the mock store
func (m *MockStore) List() ([]*Account, error) {
	if m.ListError != nil {
		return nil, m.ListError
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var accounts []*Account
	for _, account := range m.accounts {
		// Create a copy for each account
		accountCopy := *account
		accounts = append(accounts, &accountCopy)
	}
	
	return accounts, nil
}

// Delete removes credentials from the mock store
func (m *MockStore) Delete(username string) error {
	if m.DeleteError != nil {
		return m.DeleteError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if username == "" {
		return ErrInvalidCredentials
	}
	
	if _, exists := m.accounts[username]; !exists {
		return ErrCredentialsNotFound
	}
	
	delete(m.accounts, username)
	return nil
}

// Exists checks if credentials exist in the mock store
func (m *MockStore) Exists(username string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.accounts[username]
	return exists
}

// Clear removes all accounts from the mock store (useful for test cleanup)
func (m *MockStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.accounts = make(map[string]*Account)
}

// Count returns the number of accounts in the mock store (useful for testing)
func (m *MockStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.accounts)
}

// NewMockManager creates a Manager with a mock store for testing
func NewMockManager() (*Manager, *MockStore) {
	mockStore := NewMockStore()
	manager := &Manager{
		stores: []CredentialStore{mockStore},
	}
	return manager, mockStore
}

// NewMockManagerWithStores creates a Manager with multiple stores for testing
func NewMockManagerWithStores(stores ...CredentialStore) *Manager {
	return &Manager{
		stores: stores,
	}
}

// GetAccount returns a copy of the account for inspection (useful for testing)
func (m *MockStore) GetAccount(username string) (*Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	account, exists := m.accounts[username]
	if !exists {
		return nil, fmt.Errorf("account not found: %s", username)
	}
	
	accountCopy := *account
	return &accountCopy, nil
}