package scraper

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"igscraper/internal/downloader"
	"igscraper/pkg/checkpoint"
	"igscraper/pkg/config"
	"igscraper/pkg/instagram"
	"igscraper/pkg/logger"
	"igscraper/pkg/ratelimit"
	"igscraper/pkg/storage"
	"igscraper/pkg/ui"
)

const (
	retryDelay = time.Second * 2
)

// Scraper orchestrates the Instagram photo download process
type Scraper struct {
	client         InstagramClient
	storageManager *storage.Manager
	rateLimiter    ratelimit.Limiter
	tracker        *ui.StatusTracker
	progress       *ui.ProgressDisplay
	notifier       *ui.Notifier
	config         *config.Config
	logger         logger.Logger
	checkpointMgr  *checkpoint.Manager
	tui            ui.TUI
}

// New creates a new Scraper instance
func New(cfg *config.Config) (*Scraper, error) {
	// Get logger
	log := logger.GetLogger()
	
	// Create Instagram client with retry configuration
	client := instagram.NewClientWithConfig(cfg.Download.DownloadTimeout, &cfg.Retry, log)
	
	// Build cookie string with all necessary cookies
	var cookies []string
	if cfg.Instagram.SessionID != "" {
		cookies = append(cookies, fmt.Sprintf("sessionid=%s", cfg.Instagram.SessionID))
	}
	if cfg.Instagram.CSRFToken != "" {
		cookies = append(cookies, fmt.Sprintf("csrftoken=%s", cfg.Instagram.CSRFToken))
		client.SetHeader("x-csrftoken", cfg.Instagram.CSRFToken)
	}
	
	// Add other required cookies for Instagram
	cookies = append(cookies, "ig_did=B989A751-1974-4530-B367-030C95169F23")
	cookies = append(cookies, "mid=Z5NxAAAEAAHNiER_fWDXTvFWFM3t")
	cookies = append(cookies, "ds_user_id=192008031")
	
	if len(cookies) > 0 {
		client.SetHeader("Cookie", strings.Join(cookies, "; "))
	}
	
	if cfg.Instagram.UserAgent != "" {
		client.SetHeader("User-Agent", cfg.Instagram.UserAgent)
	}

	// Rate limiter based on config
	var rateLimiter ratelimit.Limiter
	if cfg.RateLimit.RequestsPerMinute > 0 {
		rateLimiter = ratelimit.NewTokenBucket(
			cfg.RateLimit.RequestsPerMinute,
			time.Minute,
		)
	} else {
		rateLimiter = ratelimit.NewTokenBucket(60, time.Minute) // Default 60/min
	}

	return &Scraper{
		client:      client,
		rateLimiter: rateLimiter,
		tracker:     ui.NewStatusTracker(),
		notifier:    ui.NewNotifier(),
		config:      cfg,
		logger:      logger.GetLogger(),
	}, nil
}

// SetTUI sets the terminal UI for the scraper
func (s *Scraper) SetTUI(tui ui.TUI) {
	s.tui = tui
}

// getOutputDir determines the output directory for a username
func (s *Scraper) getOutputDir(username string) string {
	if s.config.Output.CreateUserFolders {
		return filepath.Join(s.config.Output.BaseDirectory, username+"_photos")
	}
	return s.config.Output.BaseDirectory
}

// DownloadUserPhotos downloads all photos from a user's profile
func (s *Scraper) DownloadUserPhotos(username string) error {
	return s.downloadUserPhotosWithOptions(username, false, false)
}

// DownloadUserPhotosWithResume downloads photos with checkpoint support
func (s *Scraper) DownloadUserPhotosWithResume(username string, resume bool, forceRestart bool) error {
	return s.downloadUserPhotosWithOptions(username, resume, forceRestart)
}

// downloadUserPhotosWithOptions is the internal implementation with checkpoint support
func (s *Scraper) downloadUserPhotosWithOptions(username string, resume bool, forceRestart bool) error {
	if s.tui == nil {
		ui.PrintHighlight("\n[INITIATING EXTRACTION SEQUENCE]\n")
	} else {
		s.tui.LogInfo("Initiating extraction sequence for user: %s", username)
	}
	
	// Initialize checkpoint manager
	checkpointMgr, err := checkpoint.NewManager(username)
	if err != nil {
		s.logger.WithError(err).WithField("username", username).Error("Failed to create checkpoint manager")
		return fmt.Errorf("failed to create checkpoint manager: %w", err)
	}
	s.checkpointMgr = checkpointMgr
	
	// Handle checkpoint logic
	var cp *checkpoint.Checkpoint
	if forceRestart && checkpointMgr.Exists() {
		// Force restart: delete existing checkpoint
		if err := checkpointMgr.Delete(); err != nil {
			s.logger.WithError(err).Warn("Failed to delete existing checkpoint")
		}
		ui.PrintInfo("Force restart", "Ignoring existing checkpoint")
	} else if resume && checkpointMgr.Exists() {
		// Resume from checkpoint
		cp, err = checkpointMgr.Load()
		if err != nil {
			s.logger.WithError(err).Error("Failed to load checkpoint")
			return fmt.Errorf("failed to load checkpoint: %w", err)
		}
		if cp != nil {
			ui.PrintInfo("Resuming from checkpoint", fmt.Sprintf("Downloaded: %d photos", cp.TotalDownloaded))
			s.logger.InfoWithFields("Resuming from checkpoint", map[string]interface{}{
				"username":         username,
				"total_downloaded": cp.TotalDownloaded,
				"last_cursor":      cp.EndCursor,
			})
		}
	} else if checkpointMgr.Exists() && !resume {
		// Checkpoint exists but resume not requested
		info, _ := checkpointMgr.GetCheckpointInfo()
		if info != nil {
			// Only show checkpoint message if not in quiet mode
			if !ui.IsQuietMode() {
				fmt.Printf("\n%s Previous download found (%d photos)\n", ui.Yellow("►"), info["total_downloaded"])
				fmt.Printf("  Use: %s to continue where you left off\n", ui.Green("--resume"))
				fmt.Printf("  Use: %s to start fresh\n\n", ui.Yellow("--force-restart"))
			}
			return fmt.Errorf("checkpoint exists - use --resume to continue or --force-restart to start fresh")
		}
	}
	
	// Log the start of download process
	s.logger.InfoWithFields("Starting photo download for user", map[string]interface{}{
		"username": username,
		"action":   "download_start",
		"resume":   resume && cp != nil,
	})
	
	// Setup output directory
	outputDir := s.getOutputDir(username)
	s.logger.DebugWithFields("Setting up output directory", map[string]interface{}{
		"username":   username,
		"output_dir": outputDir,
	})
	
	storageManager, err := storage.NewManager(outputDir)
	if err != nil {
		s.logger.WithError(err).WithField("username", username).Error("Failed to create storage manager")
		return fmt.Errorf("failed to create storage manager: %w", err)
	}
	s.storageManager = storageManager
	
	// Create worker pool for concurrent downloads
	workerPool := downloader.NewWorkerPool(
		s.config.Download.ConcurrentDownloads,
		s.client,
		s.storageManager,
		s.rateLimiter,
		s.logger,
	)
	workerPool.Start()
	
	// Start result processor goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.processDownloadResults(workerPool.Results(), username)
	}()
	
	// Get initial user data or use from checkpoint
	var userID string
	var totalPhotos int
	if cp != nil && cp.UserID != "" {
		userID = cp.UserID
		s.logger.InfoWithFields("Using user ID from checkpoint", map[string]interface{}{
			"username": username,
			"user_id":  userID,
		})
		// We don't have total photos from checkpoint, will update later
		totalPhotos = -1
	} else {
		s.logger.DebugWithFields("Fetching user info", map[string]interface{}{
			"username": username,
		})
		
		userID, totalPhotos, err = s.getUserInfo(username)
		if err != nil {
			s.logger.WithError(err).WithField("username", username).Error("Failed to get user info")
			return fmt.Errorf("failed to get user info: %w", err)
		}
		
		s.logger.InfoWithFields("Successfully fetched user info", map[string]interface{}{
			"username":     username,
			"user_id":      userID,
			"total_photos": totalPhotos,
		})
		
		// Initialize metadata collection
		s.storageManager.InitializeUserMetadata(username, userID, totalPhotos)
		
		// Create new checkpoint if needed
		if cp == nil {
			cp, err = checkpointMgr.Create(username, userID)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to create checkpoint")
				// Continue without checkpoint
				cp = &checkpoint.Checkpoint{
					Username:         username,
					UserID:           userID,
					DownloadedPhotos: make(map[string]string),
				}
			}
		}
	}
	
	// Initialize progress display if not using TUI
	if s.tui == nil {
		debugMode := strings.ToLower(s.config.Logging.Level) == "debug"
		s.progress = ui.NewProgressDisplay(username, totalPhotos, debugMode)
		if cp != nil && cp.TotalDownloaded > 0 {
			s.progress.SetDownloadedCount(cp.TotalDownloaded)
		}
	}

	hasMore := true
	endCursor := ""
	totalQueued := 0
	pageNum := 0
	
	// Resume from checkpoint if available
	if cp != nil && cp.EndCursor != "" {
		endCursor = cp.EndCursor
		totalQueued = cp.TotalQueued
		pageNum = cp.LastProcessedPage
		s.tracker.SetDownloadedCount(cp.TotalDownloaded)
	}

	for hasMore {
		if s.progress != nil {
			s.progress.ScanningBatch(pageNum + 1)
		} else {
			s.tracker.PrintBatchStatus()
		}

		// Rate limit check for API calls (not downloads)
		if !s.rateLimiter.Allow() {
			logger.LogRateLimit("instagram_api", 3600) // 1 hour in seconds
			s.logger.WarnWithFields("Rate limit reached, cooling down", map[string]interface{}{
				"username":      username,
				"cooldown_time": "1 hour",
			})
			
			if s.tui != nil {
				// Update rate limit in TUI
				resetTime := time.Now().Add(time.Hour)
				s.tui.UpdateRateLimit(s.config.RateLimit.RequestsPerMinute, s.config.RateLimit.RequestsPerMinute, resetTime)
				s.tui.LogWarning("Rate limit reached, cooling down for 1 hour")
			} else if s.progress != nil {
				s.progress.RateLimitWarning(time.Hour)
			} else {
				s.notifier.SendNotification("RATE LIMIT", "Cooling down for 1 hour...")
				ui.PrintWarning("\n[COOLING DOWN FOR 1 HOUR]\n")
			}
			
			s.rateLimiter.Wait()
			
			s.logger.Info("Rate limit cooldown completed, resuming")
			if s.tui != nil {
				s.tui.LogInfo("Rate limit cooldown completed, resuming")
				s.tui.UpdateRateLimit(0, s.config.RateLimit.RequestsPerMinute, time.Now().Add(time.Minute))
			} else if s.progress == nil {
				s.notifier.SendNotification("RESUMING", "Continuing extraction process")
			}
		}

		// Fetch media batch
		s.logger.DebugWithFields("Fetching media batch", map[string]interface{}{
			"username":   username,
			"user_id":    userID,
			"end_cursor": endCursor,
		})
		
		media, pageInfo, err := s.fetchMediaBatch(username, userID, endCursor)
		if err != nil {
			s.logger.WithError(err).WithFields(map[string]interface{}{
				"username":   username,
				"end_cursor": endCursor,
			}).Error("Error fetching media batch")
			
			ui.PrintError("\nError fetching media: %v. Retrying...\n", err)
			time.Sleep(retryDelay)
			continue
		}
		
		s.logger.InfoWithFields("Media batch fetched successfully", map[string]interface{}{
			"username":    username,
			"media_count": len(media),
			"has_next":    pageInfo.HasNextPage,
		})
		
		// Update total photos if we didn't have it before (from checkpoint)
		if s.progress != nil && totalPhotos == -1 {
			// Get total from first API call
			_, newTotal, _ := s.getUserInfo(username)
			if newTotal > 0 {
				totalPhotos = newTotal
				s.progress.UpdateTotal(totalPhotos)
				// Initialize metadata if not already done
				if s.storageManager.GetUserMetadata() == nil {
					s.storageManager.InitializeUserMetadata(username, userID, totalPhotos)
				}
			}
		}

		// Queue media items for download
		for _, edge := range media {
			if edge.Node.IsVideo {
				s.logger.DebugWithFields("Skipping video", map[string]interface{}{
					"username":  username,
					"shortcode": edge.Node.Shortcode,
					"media_type": "video",
				})
				continue
			}
			
			// Skip if already downloaded (from checkpoint)
			if cp != nil && cp.IsPhotoDownloaded(edge.Node.Shortcode) {
				s.logger.DebugWithFields("Skipping already downloaded photo", map[string]interface{}{
					"username":  username,
					"shortcode": edge.Node.Shortcode,
				})
				continue
			}

			// Submit job to worker pool
			job := downloader.DownloadJob{
				URL:       edge.Node.DisplayURL,
				Shortcode: edge.Node.Shortcode,
				Username:  username,
				Node:      &edge.Node,
			}
			
			err := workerPool.Submit(job)
			if err != nil {
				s.logger.WithError(err).WithFields(map[string]interface{}{
					"username":  username,
					"shortcode": edge.Node.Shortcode,
				}).Error("Failed to submit download job")
				continue
			}
			
			// Notify about new download
			if s.tui != nil {
				// Estimate size (we don't have actual size until download starts)
				estimatedSize := int64(500000) // 500KB estimate
				s.tui.StartDownload(edge.Node.Shortcode, username, edge.Node.Shortcode+".jpg", estimatedSize)
			} else if s.progress != nil {
				s.progress.StartDownload(edge.Node.Shortcode)
			}
			
			totalQueued++
			s.logger.DebugWithFields("Download job queued", map[string]interface{}{
				"username":      username,
				"shortcode":     edge.Node.Shortcode,
				"queue_size":    workerPool.GetQueueSize(),
				"total_queued":  totalQueued,
			})
		}

		// Update checkpoint after processing batch
		pageNum++
		if cp != nil {
			cp.TotalQueued = totalQueued
			if err := checkpointMgr.UpdateProgress(cp, endCursor, pageNum); err != nil {
				s.logger.WithError(err).Warn("Failed to update checkpoint progress")
			}
		}
		
		// Handle pagination
		if pageInfo.HasNextPage {
			endCursor = pageInfo.EndCursor
			s.logger.DebugWithFields("Moving to next page", map[string]interface{}{
				"username":    username,
				"end_cursor":  endCursor,
			})
		} else {
			hasMore = false
			s.logger.InfoWithFields("No more pages to fetch", map[string]interface{}{
				"username": username,
			})
		}
	}

	// Wait for downloads to complete
	s.logger.InfoWithFields("All jobs queued, waiting for downloads to complete", map[string]interface{}{
		"username":     username,
		"total_queued": totalQueued,
	})
	
	// Stop the worker pool and wait for result processor
	workerPool.Stop()
	wg.Wait()
	
	// Save all collected metadata to a single JSON file
	if err := s.storageManager.SaveUserMetadata(); err != nil {
		s.logger.WithError(err).Error("Failed to save metadata file")
		// Don't fail the entire operation if metadata save fails
	} else {
		s.logger.Info("Metadata saved to metadata.json")
	}

	s.logger.InfoWithFields("Photo download completed successfully", map[string]interface{}{
		"username":        username,
		"total_downloaded": s.tracker.GetDownloadedCount(),
		"action":          "download_complete",
	})
	
	// Delete checkpoint on successful completion
	if s.checkpointMgr != nil && s.checkpointMgr.Exists() {
		if err := s.checkpointMgr.Delete(); err != nil {
			s.logger.WithError(err).Warn("Failed to delete checkpoint")
		} else {
			s.logger.Info("Checkpoint deleted after successful completion")
		}
	}
	
	if s.tui == nil {
		if s.progress != nil {
			s.progress.Complete()
		} else {
			ui.PrintSuccess("\n[EXTRACTION COMPLETED SUCCESSFULLY]\n")
		}
	} else {
		s.tui.LogSuccess("Extraction completed successfully for user: %s", username)
	}
	return nil
}

// getUserInfo fetches the user ID and total photo count for the given username
func (s *Scraper) getUserInfo(username string) (string, int, error) {
	endpoint := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	
	s.logger.DebugWithFields("Making API request for user info", map[string]interface{}{
		"username": username,
		"endpoint": endpoint,
	})
	
	var result instagram.InstagramResponse
	err := s.client.GetJSON(endpoint, &result)
	if err != nil {
		s.logger.WithError(err).WithField("username", username).Error("Failed to get user info")
		return "", 0, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	if result.RequiresToLogin {
		s.logger.WarnWithFields("Profile requires authentication", map[string]interface{}{
			"username": username,
		})
		return "", 0, fmt.Errorf("this profile requires authentication")
	}

	photoCount := result.Data.User.EdgeOwnerToTimelineMedia.Count
	
	s.logger.DebugWithFields("Successfully fetched user info", map[string]interface{}{
		"username":    username,
		"user_id":     result.Data.User.ID,
		"photo_count": photoCount,
	})
	
	return result.Data.User.ID, photoCount, nil
}

// getUserID fetches the user ID for the given username (backward compatibility)
func (s *Scraper) getUserID(username string) (string, error) {
	userID, _, err := s.getUserInfo(username)
	return userID, err
}

// fetchMediaBatch fetches a batch of media items
func (s *Scraper) fetchMediaBatch(username, userID, endCursor string) ([]instagram.Edge, instagram.PageInfo, error) {
	// Always use the media endpoint with the user ID
	variables := fmt.Sprintf(`{"id":"%s","first":50,"after":"%s"}`, userID, endCursor)
	endpoint := fmt.Sprintf("https://www.instagram.com/graphql/query/?query_hash=%s&variables=%s", instagram.MediaQueryHash, variables)
	
	s.logger.DebugWithFields("Fetching media batch", map[string]interface{}{
		"username":   username,
		"user_id":    userID,
		"end_cursor": endCursor,
		"endpoint":   endpoint,
	})

	var result instagram.InstagramResponse
	err := s.client.GetJSON(endpoint, &result)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"username":   username,
			"end_cursor": endCursor,
		}).Error("Failed to fetch media batch")
		return nil, instagram.PageInfo{}, fmt.Errorf("failed to fetch media: %w", err)
	}

	media := result.Data.User.EdgeOwnerToTimelineMedia
	
	s.logger.DebugWithFields("Media batch fetched", map[string]interface{}{
		"username":      username,
		"media_count":   len(media.Edges),
		"has_next_page": media.PageInfo.HasNextPage,
	})
	
	return media.Edges, media.PageInfo, nil
}

// processDownloadResults processes results from the worker pool
func (s *Scraper) processDownloadResults(results <-chan downloader.DownloadResult, username string) {
	for result := range results {
		if result.Success {
			logger.LogDownload(username, result.Job.Shortcode, "photo", true, nil)
			
			// Extract metadata for progress display
			var metadata map[string]interface{}
			if result.Job.Node != nil {
				metadata = make(map[string]interface{})
				if len(result.Job.Node.EdgeMediaToCaption.Edges) > 0 {
					metadata["caption"] = result.Job.Node.EdgeMediaToCaption.Edges[0].Node.Text
				}
				metadata["likes"] = result.Job.Node.EdgeLikedBy.Count
				metadata["comments"] = result.Job.Node.EdgeMediaToComment.Count
			}
			
			if s.tui != nil {
				// Complete the download in TUI
				s.tui.CompleteDownload(result.Job.Shortcode)
			} else if s.progress != nil {
				// Use new progress display
				s.progress.CompleteDownload(result.Job.Shortcode, int64(result.Size), metadata)
			} else {
				// Fallback to old tracker
				s.tracker.IncrementDownloaded()
				s.tracker.PrintProgress()
			}
			
			// Record successful download in checkpoint
			if s.checkpointMgr != nil {
				// Load current checkpoint to get latest state
				cp, err := s.checkpointMgr.Load()
				if err == nil && cp != nil {
					filename := fmt.Sprintf("%s.jpg", result.Job.Shortcode)
					if err := s.checkpointMgr.RecordDownload(cp, result.Job.Shortcode, filename); err != nil {
						s.logger.WithError(err).Warn("Failed to record download in checkpoint")
					}
				}
			}
			
			s.logger.DebugWithFields("Download completed successfully", map[string]interface{}{
				"username":  username,
				"shortcode": result.Job.Shortcode,
				"duration":  result.Duration,
				"size":      result.Size,
			})
		} else {
			logger.LogDownload(username, result.Job.Shortcode, "photo", false, result.Error)
			
			if s.tui != nil {
				// Fail the download in TUI
				s.tui.FailDownload(result.Job.Shortcode, result.Error)
			} else if s.progress != nil {
				// Use new progress display
				s.progress.FailDownload(result.Job.Shortcode, result.Error)
			} else {
				// Use regular error printing
				ui.PrintError("\nError downloading %s: %v\n", result.Job.Shortcode, result.Error)
			}
			
			s.logger.ErrorWithFields("Download failed", map[string]interface{}{
				"username":  username,
				"shortcode": result.Job.Shortcode,
				"error":     result.Error.Error(),
				"duration":  result.Duration,
			})
		}
	}
}

// downloadPhoto downloads a single photo
func (s *Scraper) downloadPhoto(url, shortcode string) error {
	s.logger.DebugWithFields("Starting photo download", map[string]interface{}{
		"shortcode": shortcode,
		"url":       url,
	})
	
	start := time.Now()
	data, err := s.client.DownloadPhoto(url)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"shortcode": shortcode,
			"url":       url,
		}).Error("Failed to download photo")
		return fmt.Errorf("failed to download photo: %w", err)
	}
	
	downloadDuration := time.Since(start)
	s.logger.DebugWithFields("Photo downloaded", map[string]interface{}{
		"shortcode":    shortcode,
		"size_bytes":   len(data),
		"duration_ms":  downloadDuration.Milliseconds(),
	})

	// SavePhoto expects shortcode, not filename
	err = s.storageManager.SavePhoto(bytes.NewReader(data), shortcode)
	if err != nil {
		s.logger.WithError(err).WithField("shortcode", shortcode).Error("Failed to save photo")
		return err
	}
	
	s.logger.DebugWithFields("Photo saved successfully", map[string]interface{}{
		"shortcode": shortcode,
	})
	
	return nil
}

// generateFilename generates a filename based on the configured pattern
func (s *Scraper) generateFilename(shortcode string) string {
	pattern := s.config.Output.FileNamePattern
	if pattern == "" {
		pattern = "{shortcode}.jpg"
	}
	
	// Replace placeholders
	filename := strings.ReplaceAll(pattern, "{shortcode}", shortcode)
	filename = strings.ReplaceAll(filename, "{timestamp}", fmt.Sprintf("%d", time.Now().Unix()))
	filename = strings.ReplaceAll(filename, "{date}", time.Now().Format("2006-01-02"))
	
	// Ensure proper extension
	if !strings.Contains(filename, ".") {
		filename += ".jpg"
	}
	
	return filename
}
