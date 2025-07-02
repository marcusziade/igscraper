// Package instagram provides a client for interacting with Instagram's web API.
//
// This package includes:
//   - A configurable HTTP client with proper headers and error handling
//   - Type-safe models for Instagram API responses
//   - Helper functions for constructing API endpoints
//   - Built-in error types for better error handling
//
// Example usage:
//
//	client := instagram.NewClient(30 * time.Second)
//	
//	// Fetch user profile
//	profile, err := client.FetchUserProfile("username")
//	if err != nil {
//	    if igErr, ok := err.(*instagram.Error); ok {
//	        switch igErr.Type {
//	        case instagram.ErrorTypeAuth:
//	            // Handle authentication error
//	        case instagram.ErrorTypeRateLimit:
//	            // Handle rate limit
//	        }
//	    }
//	}
//	
//	// Download photos
//	for _, edge := range profile.Data.User.EdgeOwnerToTimelineMedia.Edges {
//	    if !edge.Node.IsVideo {
//	        photoData, err := client.DownloadPhoto(edge.Node.DisplayURL)
//	        // Handle photo data
//	    }
//	}
package instagram