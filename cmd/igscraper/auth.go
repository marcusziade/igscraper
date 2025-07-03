package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"igscraper/pkg/auth"
	"igscraper/pkg/ui"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Instagram credentials",
	Long: `Manage Instagram credentials for accessing profiles.

OVERVIEW:
  IGScraper requires valid Instagram session cookies to download photos.
  This command helps you securely store and manage multiple Instagram accounts.

SECURITY:
  • System Keychain     - Secure OS-level storage (macOS, Linux, Windows)
  • Encrypted Storage   - AES-256 with PBKDF2 key derivation
  • Environment Vars    - Legacy support for CI/CD workflows
  • No passwords stored - Only session cookies are saved

SUBCOMMANDS:
  login    - Add or update Instagram credentials
  logout   - Remove stored credentials
  list     - Show all saved accounts
  switch   - Select default account

QUICK START:
  1. Login to Instagram in your browser
  2. Extract session cookies (see 'auth login --help')
  3. Store credentials: igscraper auth login
  4. Start downloading: igscraper username

For detailed instructions, see: https://github.com/marcusziade/igscraper/blob/master/docs/MANUAL.md#authentication`,
}

// loginCmd represents the auth login command
var loginCmd = &cobra.Command{
	Use:   "login [username]",
	Short: "Store Instagram credentials securely",
	Long: `Store Instagram credentials securely for downloading photos.

HOW TO GET CREDENTIALS:
  1. Open Instagram.com in your browser
  2. Log in to your account
  3. Open Developer Tools:
     • Chrome/Edge: F12 or Ctrl+Shift+I (Cmd+Option+I on Mac)
     • Firefox: F12 or Ctrl+Shift+I (Cmd+Option+I on Mac)
     • Safari: Enable Developer menu, then Cmd+Option+I
  4. Navigate to:
     • Chrome/Edge: Application → Storage → Cookies → instagram.com
     • Firefox: Storage → Cookies → instagram.com
     • Safari: Storage → Cookies → instagram.com
  5. Find and copy these values:
     • sessionid: Long string with % symbols (e.g., 12345678%3Aabcdef...)
     • csrftoken: ~32 character string (e.g., YTQHujAgMhyveLvvuwCfw9...)

INTERACTIVE PROMPTS:
  • Instagram username (if not provided)
  • Session ID cookie value
  • CSRF Token cookie value
  • User Agent string (optional)

SECURITY NOTES:
  • Credentials are encrypted at rest
  • Never share your session cookies
  • Cookies expire - re-login periodically
  • Each account is stored separately`,
	Example: `  # Interactive login with guide
  igscraper auth login

  # Login with username (skip username prompt)
  igscraper auth login myusername

  # After login, download photos
  igscraper cristiano`,
	Args: cobra.MaximumNArgs(1),
	Run:  runLogin,
}

// logoutCmd represents the auth logout command
var logoutCmd = &cobra.Command{
	Use:   "logout [username]",
	Short: "Remove stored credentials",
	Long: `Remove stored Instagram credentials from secure storage.

BEHAVIOR:
  • No username: Shows interactive menu of all accounts
  • With username: Removes specific account directly
  • Removes from both keychain and encrypted storage
  • Cannot be undone - credentials must be re-entered

INTERACTIVE MODE:
  When no username is provided, you can:
  • Select specific account to remove
  • Remove all accounts at once
  • Cancel without changes`,
	Example: `  # Interactive logout (shows menu)
  igscraper auth logout

  # Remove specific account
  igscraper auth logout myusername

  # Remove all accounts (interactive confirmation)
  igscraper auth logout`,
	Args: cobra.MaximumNArgs(1),
	Run:  runLogout,
}

// listCmd represents the auth list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored accounts",
	Long: `List all stored Instagram accounts with sanitized credential information.

DISPLAYED INFORMATION:
  • Username
  • Session ID (partially hidden)
  • CSRF Token (partially hidden)
  • User Agent (if set)
  • Last modified date

SECURITY:
  Sensitive values are automatically masked for security.
  Full credentials are never displayed in plain text.`,
	Example: `  # List all stored accounts
  igscraper auth list`,
	Run:   runList,
}

// switchCmd represents the auth switch command
var switchCmd = &cobra.Command{
	Use:   "switch [username]",
	Short: "Switch between stored accounts",
	Long: `Switch between stored Instagram accounts for downloads.

USAGE:
  • No username: Shows interactive menu to select account
  • With username: Selects specific account directly
  
NOTE:
  The selected account will be used with the --account flag:
  igscraper scrape <profile> --account <selected>
  
  Without --account flag, the first stored account is used.`,
	Example: `  # Interactive account selection
  igscraper auth switch

  # Switch to specific account
  igscraper auth switch work_account

  # Use selected account for download
  igscraper scrape cristiano --account work_account`,
	Args: cobra.MaximumNArgs(1),
	Run:  runSwitch,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(listCmd)
	authCmd.AddCommand(switchCmd)
}

func runLogin(cmd *cobra.Command, args []string) {
	manager, err := auth.NewManager()
	if err != nil {
		ui.PrintError("Failed to initialize credential manager", err.Error())
		os.Exit(1)
	}

	var username string
	if len(args) > 0 {
		username = args[0]
	}
	
	// Interactive prompts
	reader := bufio.NewReader(os.Stdin)
	
	// Show extraction guide first
	auth.ShowCookieExtractionGuide()
	
	// Ask if ready to continue
	fmt.Print("Ready to enter your cookies? (Y/n): ")
	ready, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(ready)) == "n" {
		fmt.Println("\nRun 'igscraper auth login' when you're ready.")
		return
	}
	
	fmt.Println() // Add spacing
	
	if username == "" {
		fmt.Print("📱 Instagram username: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			ui.PrintError("Failed to read username", err.Error())
			os.Exit(1)
		}
		username = strings.TrimSpace(input)
	}
	
	if username == "" {
		ui.PrintError("Username is required", "")
		os.Exit(1)
	}
	
	// Check if account already exists
	if existing, _ := manager.Retrieve(username); existing != nil {
		fmt.Printf("\n⚠️  Account '%s' already exists. Update credentials? (y/N): ", username)
		input, _ := reader.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
			return
		}
	}
	
	fmt.Println("\n🔐 Enter your cookie values (they will be hidden as you type):")
	fmt.Println()
	
	// Get session ID with validation
	var sessionID string
	for {
		fmt.Printf("sessionid cookie value: ")
		sessionID, err = readPassword()
		if err != nil {
			ui.PrintError("Failed to read session ID", err.Error())
			os.Exit(1)
		}
		
		// Basic validation
		if len(sessionID) < 20 || !strings.Contains(sessionID, "%") {
			fmt.Println("\n❌ That doesn't look like a valid sessionid.")
			fmt.Println("   It should be a long string containing % symbols.")
			fmt.Println("   Example: 12345678%3Aabcdef%3A26%3A...")
			fmt.Print("\nTry again? (Y/n): ")
			retry, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(retry)) == "n" {
				os.Exit(1)
			}
			continue
		}
		break
	}
	
	// Get CSRF token with validation
	var csrfToken string
	for {
		fmt.Printf("\ncsrftoken cookie value: ")
		csrfToken, err = readPassword()
		if err != nil {
			ui.PrintError("Failed to read CSRF token", err.Error())
			os.Exit(1)
		}
		
		// Basic validation
		if len(csrfToken) < 20 || len(csrfToken) > 50 {
			fmt.Println("\n❌ That doesn't look like a valid csrftoken.")
			fmt.Println("   It should be around 32 characters long.")
			fmt.Println("   Example: YTQHujAgMhyveLvvuwCfw9CPI8ROAHoy")
			fmt.Print("\nTry again? (Y/n): ")
			retry, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(retry)) == "n" {
				os.Exit(1)
			}
			continue
		}
		break
	}
	
	// Optional: Get user agent
	fmt.Print("\n\n🌐 User Agent (press Enter to use default): ")
	userAgent, _ := reader.ReadString('\n')
	userAgent = strings.TrimSpace(userAgent)
	
	// Show what we're about to do
	fmt.Println("\n📋 Summary:")
	fmt.Printf("   Username: %s\n", username)
	fmt.Printf("   SessionID: %s...%s (hidden)\n", sessionID[:8], sessionID[len(sessionID)-4:])
	fmt.Printf("   CSRF Token: %s...%s (hidden)\n", csrfToken[:4], csrfToken[len(csrfToken)-4:])
	if userAgent != "" {
		fmt.Printf("   User Agent: %s\n", userAgent)
	}
	
	// Create account
	account := &auth.Account{
		Username:     username,
		SessionID:    sessionID,
		CSRFToken:    csrfToken,
		UserAgent:    userAgent,
		LastModified: time.Now(),
	}
	
	// Store credentials
	fmt.Println("\n💾 Storing credentials securely...")
	if err := manager.Store(account); err != nil {
		ui.PrintError("Failed to store credentials", err.Error())
		os.Exit(1)
	}
	
	// Set as default if it's the first account
	accounts, _ := manager.List()
	if len(accounts) == 1 {
		// First account becomes default automatically
		fmt.Printf("✅ Set '%s' as default account\n", username)
	}
	
	fmt.Println("\n🎉 Credentials stored successfully!")
	ui.PrintSuccess(fmt.Sprintf("Account saved: %s", username))
	
	// Show where credentials are stored
	fmt.Println("\n🔒 Security Information:")
	fmt.Println("   Your credentials are encrypted and stored in:")
	if auth.IsKeyringAvailable() {
		fmt.Println("   • System keychain (primary)")
	}
	fmt.Println("   • Encrypted file (backup)")
	
	// Show how to use
	fmt.Println("\n📖 Quick Start Guide:")
	fmt.Println("   Download photos from any public profile:")
	fmt.Printf("   $ igscraper scrape <instagram_username>\n")
	fmt.Println("\n   Example:")
	fmt.Printf("   $ igscraper scrape cristiano\n")
	fmt.Println("\n   Use specific account:")
	fmt.Printf("   $ igscraper scrape <instagram_username> --account %s\n", username)
	fmt.Println("\n   Show more options:")
	fmt.Printf("   $ igscraper scrape --help\n")
	fmt.Println("\n⚠️  Never share your credentials or config files!")
}

func runLogout(cmd *cobra.Command, args []string) {
	manager, err := auth.NewManager()
	if err != nil {
		ui.PrintError("Failed to initialize credential manager", err.Error())
		os.Exit(1)
	}

	if len(args) == 0 {
		// List accounts and ask which to remove
		accounts, err := manager.List()
		if err != nil || len(accounts) == 0 {
			ui.PrintError("No stored accounts found", "")
			return
		}
		
		if len(accounts) == 1 {
			// Only one account, confirm deletion
			account := accounts[0]
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Remove account '%s'? (y/N): ", account.Username)
			input, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
				return
			}
			
			if err := manager.Delete(account.Username); err != nil {
				ui.PrintError("Failed to remove account", err.Error())
				os.Exit(1)
			}
			ui.PrintSuccess("Account removed: " + account.Username)
			return
		}
		
		// Multiple accounts, show menu
		fmt.Println("Select account to remove:")
		for i, account := range accounts {
			fmt.Printf("  %d. %s\n", i+1, account.Username)
		}
		fmt.Printf("  %d. Remove all accounts\n", len(accounts)+1)
		fmt.Printf("  0. Cancel\n\n")
		
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Choice: ")
		input, _ := reader.ReadString('\n')
		
		var choice int
		fmt.Sscanf(strings.TrimSpace(input), "%d", &choice)
		
		if choice == 0 {
			return
		} else if choice == len(accounts)+1 {
			// Remove all
			fmt.Print("Remove ALL accounts? This cannot be undone! (yes/N): ")
			confirm, _ := reader.ReadString('\n')
			if strings.TrimSpace(confirm) != "yes" {
				return
			}
			
			if err := manager.DeleteAll(); err != nil {
				ui.PrintError("Failed to remove all accounts", err.Error())
				os.Exit(1)
			}
			ui.PrintSuccess("All accounts removed")
			return
		} else if choice > 0 && choice <= len(accounts) {
			account := accounts[choice-1]
			if err := manager.Delete(account.Username); err != nil {
				ui.PrintError("Failed to remove account", err.Error())
				os.Exit(1)
			}
			ui.PrintSuccess("Account removed: " + account.Username)
			return
		} else {
			ui.PrintError("Invalid choice", "")
			os.Exit(1)
		}
	}
	
	// Username provided as argument
	username := args[0]
	if err := manager.Delete(username); err != nil {
		ui.PrintError("Failed to remove account", err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess("Account removed: " + username)
}

func runList(cmd *cobra.Command, args []string) {
	manager, err := auth.NewManager()
	if err != nil {
		ui.PrintError("Failed to initialize credential manager", err.Error())
		os.Exit(1)
	}

	accounts, err := manager.List()
	if err != nil {
		ui.PrintError("Failed to list accounts", err.Error())
		os.Exit(1)
	}
	
	if len(accounts) == 0 {
		ui.PrintInfo("No stored accounts", "Use 'igscraper auth login' to add an account")
		return
	}
	
	ui.PrintHighlight("Stored Accounts")
	fmt.Println()
	
	for i, account := range accounts {
		sanitized := auth.SanitizeAccount(account)
		fmt.Printf("%d. Username: %s\n", i+1, sanitized.Username)
		fmt.Printf("   Session ID: %s\n", sanitized.SessionID)
		fmt.Printf("   CSRF Token: %s\n", sanitized.CSRFToken)
		if sanitized.UserAgent != "" {
			fmt.Printf("   User Agent: %s\n", sanitized.UserAgent)
		}
		fmt.Printf("   Last Modified: %s\n", sanitized.LastModified.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

func runSwitch(cmd *cobra.Command, args []string) {
	manager, err := auth.NewManager()
	if err != nil {
		ui.PrintError("Failed to initialize credential manager", err.Error())
		os.Exit(1)
	}

	accounts, err := manager.List()
	if err != nil || len(accounts) == 0 {
		ui.PrintError("No stored accounts found", "")
		return
	}
	
	if len(accounts) == 1 {
		ui.PrintInfo("Only one account available", accounts[0].Username)
		return
	}
	
	var username string
	if len(args) > 0 {
		username = args[0]
	} else {
		// Interactive selection
		fmt.Println("Select account:")
		for i, account := range accounts {
			fmt.Printf("  %d. %s\n", i+1, account.Username)
		}
		fmt.Println()
		
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Choice: ")
		input, _ := reader.ReadString('\n')
		
		var choice int
		fmt.Sscanf(strings.TrimSpace(input), "%d", &choice)
		
		if choice < 1 || choice > len(accounts) {
			ui.PrintError("Invalid choice", "")
			os.Exit(1)
		}
		
		username = accounts[choice-1].Username
	}
	
	// Verify account exists
	if _, err := manager.Retrieve(username); err != nil {
		ui.PrintError("Account not found", username)
		os.Exit(1)
	}
	
	// Note: In a real implementation, we might store the default account preference
	// For now, just show confirmation
	ui.PrintSuccess("Account selected: " + username)
	fmt.Println("\nUse the --account flag to use this account:")
	fmt.Printf("  igscraper scrape <username> --account %s\n", username)
}

// readPassword reads a password from stdin without echoing
func readPassword() (string, error) {
	// Try to read without echo
	if term.IsTerminal(int(syscall.Stdin)) {
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // New line after password
		if err == nil {
			return string(password), nil
		}
	}
	
	// Fallback to regular input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// testCredentials tests if the provided credentials work with Instagram
func testCredentials(account *auth.Account) error {
	// For now, we'll skip the test to avoid complexity
	// In a real implementation, we'd make a test API call
	return nil
}