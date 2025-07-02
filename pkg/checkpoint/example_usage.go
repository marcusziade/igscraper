package checkpoint

import (
	"fmt"
	"log"
)

func ExampleManager() {
	// Create checkpoint manager for a username
	mgr, err := NewManager("johndoe")
	if err != nil {
		log.Fatal(err)
	}

	// Check if checkpoint exists
	if mgr.Exists() {
		// Load existing checkpoint
		cp, err := mgr.Load()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Resuming from checkpoint: %d photos downloaded\n", cp.TotalDownloaded)
		
		// Continue from where we left off
		fmt.Printf("Last cursor: %s\n", cp.EndCursor)
	} else {
		// Create new checkpoint
		cp, err := mgr.Create("johndoe", "user123456")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Starting fresh download")
		
		// Record progress as we download
		err = mgr.RecordDownload(cp, "ABC123", "ABC123.jpg")
		if err != nil {
			log.Fatal(err)
		}
		
		// Update pagination progress
		err = mgr.UpdateProgress(cp, "next_cursor_xyz", 1)
		if err != nil {
			log.Fatal(err)
		}
	}
	
	// When download completes successfully, delete checkpoint
	err = mgr.Delete()
	if err != nil {
		log.Printf("Failed to delete checkpoint: %v", err)
	}
}

func ExampleCheckpoint_IsPhotoDownloaded() {
	mgr, _ := NewManager("testuser")
	cp, _ := mgr.Create("testuser", "12345")
	
	// Record some downloads
	mgr.RecordDownload(cp, "photo1", "photo1.jpg")
	mgr.RecordDownload(cp, "photo2", "photo2.jpg")
	
	// Check if photos are downloaded
	if cp.IsPhotoDownloaded("photo1") {
		fmt.Println("photo1 already downloaded, skipping")
	}
	
	if !cp.IsPhotoDownloaded("photo3") {
		fmt.Println("photo3 not downloaded yet, will download")
	}
	
	// Output:
	// photo1 already downloaded, skipping
	// photo3 not downloaded yet, will download
}