# Authentication Package

The auth package provides secure credential management for Instagram accounts with multiple storage backends.

## Features

- **Multiple Storage Backends**: System keychain, encrypted file, environment variables
- **Secure Storage**: Uses AES-256 encryption with PBKDF2 key derivation
- **Multi-Account Support**: Store and manage multiple Instagram accounts
- **Backward Compatibility**: Falls back to environment variables for existing setups

## Storage Backends

### 1. System Keychain (Primary)
- macOS: Keychain Access
- Windows: Credential Manager
- Linux: Secret Service (GNOME Keyring, KWallet)

### 2. Encrypted File (Fallback)
- Location: `~/.config/igscraper/credentials.enc` (Linux/macOS)
- Location: `%APPDATA%\igscraper\credentials.enc` (Windows)
- Encryption: AES-256-GCM with PBKDF2 key derivation
- Passphrase: Auto-generated and stored in `~/.config/igscraper/.passphrase`

### 3. Environment Variables (Legacy)
- `IGSCRAPER_SESSION_ID`: Instagram session ID
- `IGSCRAPER_CSRF_TOKEN`: Instagram CSRF token
- `IGSCRAPER_USER_AGENT`: Optional user agent

## Security Considerations

1. **Never commit credentials**: The encrypted file and passphrase should never be committed to version control
2. **File permissions**: All credential files are created with 0600 permissions (owner read/write only)
3. **Memory safety**: Credentials are cleared from memory after use where possible
4. **Logging**: Credentials are masked in logs and error messages

## Usage

### Command Line

```bash
# Store credentials
igscraper auth login myusername

# List stored accounts
igscraper auth list

# Remove credentials
igscraper auth logout myusername

# Use specific account for scraping
igscraper -account myusername targetuser
```

### Programmatic

```go
// Create credential manager
manager, err := auth.NewManager()
if err != nil {
    log.Fatal(err)
}

// Store credentials
account := &auth.Account{
    Username:  "myusername",
    SessionID: "session_id_here",
    CSRFToken: "csrf_token_here",
}
err = manager.Store(account)

// Retrieve credentials
account, err = manager.Retrieve("myusername")

// List all accounts
accounts, err := manager.List()

// Delete credentials
err = manager.Delete("myusername")
```

## Getting Instagram Credentials

1. Log in to Instagram in your web browser
2. Open Developer Tools (F12)
3. Go to Application/Storage → Cookies
4. Find and copy:
   - `sessionid` cookie value
   - `csrftoken` cookie value

## Troubleshooting

### Keyring Not Available
- Linux: Install `gnome-keyring` or `kwallet`
- SSH sessions: Keyring may not be available, will use encrypted file

### Permission Denied
- Ensure config directory has proper permissions: `chmod 700 ~/.config/igscraper`
- Check file permissions: `chmod 600 ~/.config/igscraper/credentials.enc`

### Lost Passphrase
- Delete `~/.config/igscraper/.passphrase` and `credentials.enc`
- Re-add accounts using `igscraper auth login`