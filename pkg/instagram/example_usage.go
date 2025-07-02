package instagram

// Example of how to update code to use the new instagram package:
//
// Old code:
//   import "igscraper/pkg/models"
//   var result models.InstagramResponse
//
// New code:
//   import "igscraper/pkg/instagram"
//   var result instagram.InstagramResponse
//
// Or using the client:
//   import "igscraper/pkg/logger"
//   log := logger.GetLogger()
//   client := instagram.NewClient(30 * time.Second, log)
//   result, err := client.FetchUserProfile(username)
//
// The main benefits of using the client:
// 1. Centralized header management
// 2. Better error handling with typed errors
// 3. Automatic retry logic (can be added)
// 4. Cleaner API with dedicated methods