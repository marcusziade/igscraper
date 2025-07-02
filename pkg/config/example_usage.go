package config

// Example usage of the configuration system:
//
// 1. Load configuration with all sources:
//
//     config, err := config.Load("", nil)
//     if err != nil {
//         log.Fatal(err)
//     }
//
// 2. Load with custom config file:
//
//     config, err := config.Load("/path/to/config.yaml", nil)
//     if err != nil {
//         log.Fatal(err)
//     }
//
// 3. Load with command line flags:
//
//     flags := map[string]interface{}{
//         "session-id": "abc123",
//         "csrf-token": "xyz789",
//         "output": "./my-downloads",
//         "concurrent": 5,
//         "log-level": "debug",
//     }
//     config, err := config.Load("", flags)
//     if err != nil {
//         log.Fatal(err)
//     }
//
// 4. Programmatic configuration:
//
//     config := config.DefaultConfig()
//     config.Instagram.SessionID = "your-session-id"
//     config.Instagram.CSRFToken = "your-csrf-token"
//     config.Download.ConcurrentDownloads = 5
//     
//     if err := config.Validate(); err != nil {
//         log.Fatal(err)
//     }
//
// 5. Save configuration to file:
//
//     if err := config.Save(".igscraper.yaml"); err != nil {
//         log.Fatal(err)
//     }
//
// 6. Environment variables:
//
//     export IGSCRAPER_SESSION_ID="your-session-id"
//     export IGSCRAPER_CSRF_TOKEN="your-csrf-token"
//     export IGSCRAPER_OUTPUT_DIR="./downloads"
//     export IGSCRAPER_CONCURRENT_DOWNLOADS="5"
//     export IGSCRAPER_REQUESTS_PER_MINUTE="30"
//     export IGSCRAPER_NOTIFICATIONS_ENABLED="true"
//     export IGSCRAPER_LOG_LEVEL="debug"
//
// 7. Using configuration in your application:
//
//     // Create Instagram client with config
//     client := instagram.NewClient(
//         config.Instagram.SessionID,
//         config.Instagram.CSRFToken,
//     )
//     
//     // Set up rate limiter
//     limiter := ratelimit.NewLimiter(
//         config.RateLimit.RequestsPerMinute,
//         config.RateLimit.BurstSize,
//     )
//     
//     // Configure downloader
//     downloader := downloader.New(
//         config.Download.ConcurrentDownloads,
//         config.Download.DownloadTimeout,
//     )