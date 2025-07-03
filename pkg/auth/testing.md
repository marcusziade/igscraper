# Testing Credential Management

## Problem
The original `TestCredentialManager` test was failing because it relied on the real credential stores which have environment-specific behaviors:

1. **KeyringStore**: May not be available in CI environments and doesn't support listing accounts due to library limitations
2. **EnvironmentStore**: Doesn't support storing credentials (read-only)
3. **EncryptedFileStore**: Works but requires proper file system setup

## Solution
Created a `MockStore` implementation that provides:

1. **In-memory storage**: No external dependencies
2. **Full API support**: Implements all CredentialStore methods
3. **Test helpers**: Additional methods like `Count()` and `Clear()`
4. **Error injection**: Ability to inject errors for negative testing

## Usage

### Basic Mock Testing
```go
// Create a manager with mock store
manager, mockStore := NewMockManager()

// Use manager normally
err := manager.Store(account)
accounts, err := manager.List()

// Verify mock state
count := mockStore.Count()
```

### Testing with Real Stores
For integration testing with real stores, use controlled environments:

```go
// Create manager with specific stores
encryptedStore, _ := NewEncryptedFileStore(tempPath)
manager := NewMockManagerWithStores(encryptedStore)
```

### Error Testing
```go
mockStore := NewMockStore()
mockStore.StoreError = errors.New("storage failed")
err := mockStore.Store(account) // Returns injected error
```

## Benefits

1. **Reliable CI/CD**: Tests pass consistently across all environments
2. **Fast execution**: No file I/O or system keychain access
3. **Isolation**: Tests don't interfere with system credentials
4. **Flexibility**: Easy to test edge cases and error conditions