package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"igscraper/pkg/ui"
)

var (
	// Version information
	version   = "2.0.0"
	gitCommit = "unknown"
	buildDate = "unknown"

	// Global flags
	configFile    string
	logLevel      string
	noColor       bool
	notifications bool
	quiet         bool
	progressOnly  bool
	verbose       bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "igscraper",
	Short: "A powerful Instagram photo downloader with advanced features",
	Long: `IGScraper - Instagram Photo Downloader v` + version + `

A command-line tool for downloading photos from Instagram profiles with advanced features
for power users and developers.

FEATURES:
  • Concurrent Downloads    - Download multiple photos simultaneously
  • Smart Rate Limiting     - Avoid Instagram API restrictions  
  • Resume Support          - Continue interrupted downloads
  • Secure Authentication   - Keychain, encrypted, or environment storage
  • Multiple UI Modes       - TUI, progress bar, or quiet operation
  • Duplicate Detection     - Skip already downloaded photos
  • Metadata Extraction     - Save captions and photo details

QUICK START:
  1. Authenticate:     igscraper auth login
  2. Download photos:  igscraper username
  3. Use TUI mode:     igscraper --tui username

EXAMPLES:
  # Download all photos from a profile
  igscraper username

  # Download with beautiful terminal UI
  igscraper --tui username

  # Download first 100 photos with 5 workers
  igscraper -l 100 -w 5 username

  # Resume interrupted download
  igscraper --resume username

DOCUMENTATION:
  Full manual: https://github.com/marcusziade/igscraper/blob/master/docs/MANUAL.md
  Report bugs: https://github.com/marcusziade/igscraper/issues

For more information and examples, visit: https://github.com/marcusziade/igscraper`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildDate),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Progress mode is default unless verbose is specified
		if !verbose && !quiet {
			progressOnly = true
		}
		
		// Set quiet mode if requested or log level is error
		if quiet || logLevel == "error" {
			ui.SetQuietMode(true)
		}
		
		// Set progress-only mode
		if progressOnly {
			ui.SetProgressOnlyMode(true)
			// Also set log level to error to suppress logs
			logLevel = "error"
		}
		
		// Don't show logo for certain commands
		if cmd.Name() != "version" && cmd.Name() != "help" && cmd.Name() != "completion" {
			ui.PrintLogo()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is $HOME/.igscraper.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&notifications, "notifications", true, "enable desktop notifications")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress all output except errors")
	rootCmd.PersistentFlags().BoolVarP(&progressOnly, "progress", "p", false, "show only progress bar and essential info")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show all output (logo, logs, progress)")

	// Version template
	rootCmd.SetVersionTemplate(`Instagram Scraper {{.Version}}
Go Version: ` + runtime.Version() + `
OS/Arch: ` + runtime.GOOS + `/` + runtime.GOARCH + `
`)

	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// This will be called before any command execution
	// Config loading logic will be handled in individual commands
}