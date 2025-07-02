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
	client         *instagram.Client
	storageManager *storage.Manager
	rateLimiter    ratelimit.Limiter
	tracker        *ui.StatusTracker
	notifier       *ui.Notifier
	config         *config.Config
	logger         logger.Logger
	checkpointMgr  *checkpoint.Manager
}

// New creates a new Scraper instance
func New(cfg *config.Config) (*Scraper, error) {
	// Get logger
	log := logger.GetLogger()
	
	// Create Instagram client with retry configuration
	client := instagram.NewClientWithConfig(cfg.Download.DownloadTimeout, &cfg.Retry, log)
	if cfg.Instagram.SessionID != "" {
		client.SetHeader("Cookie", fmt.Sprintf("sessionid=%s", cfg.Instagram.SessionID))
	}
	if cfg.Instagram.CSRFToken != "" {
		client.SetHeader("x-csrftoken", cfg.Instagram.CSRFToken)
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
	ui.PrintHighlight("\n[INITIATING EXTRACTION SEQUENCE]\n")
	
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
			ui.PrintWarning("Existing checkpoint found. Use --resume to continue or --force-restart to start over.")
			ui.PrintInfo("Checkpoint info", fmt.Sprintf("Downloaded: %d photos, Last updated: %v ago", 
				info["total_downloaded"], info["age"]))
			return fmt.Errorf("checkpoint exists, use --resume to continue")
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
	defer workerPool.Stop()
	
	// Start result processor goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.processDownloadResults(workerPool.Results(), username)
	}()
	
	// Get initial user data or use from checkpoint
	var userID string
	if cp != nil && cp.UserID != "" {
		userID = cp.UserID
		s.logger.InfoWithFields("Using user ID from checkpoint", map[string]interface{}{
			"username": username,
			"user_id":  userID,
		})
	} else {
		s.logger.DebugWithFields("Fetching user ID", map[string]interface{}{
			"username": username,
		})
		
		userID, err = s.getUserID(username)
		if err != nil {
			s.logger.WithError(err).WithField("username", username).Error("Failed to get user ID")
			return fmt.Errorf("failed to get user ID: %w", err)
		}
		
		s.logger.InfoWithFields("Successfully fetched user ID", map[string]interface{}{
			"username": username,
			"user_id":  userID,
		})
		
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
		s.tracker.PrintBatchStatus()

		// Rate limit check for API calls (not downloads)
		if !s.rateLimiter.Allow() {
			logger.LogRateLimit("instagram_api", 3600) // 1 hour in seconds
			s.logger.WarnWithFields("Rate limit reached, cooling down", map[string]interface{}{
				"username":      username,
				"cooldown_time": "1 hour",
			})
			
			s.notifier.SendNotification("RATE LIMIT", "Cooling down for 1 hour...")
			ui.PrintWarning("\n[COOLING DOWN FOR 1 HOUR]\n")
			s.rateLimiter.Wait()
			
			s.logger.Info("Rate limit cooldown completed, resuming")
			s.notifier.SendNotification("RESUMING", "Continuing extraction process")
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
			}
			
			err := workerPool.Submit(job)
			if err != nil {
				s.logger.WithError(err).WithFields(map[string]interface{}{
					"username":  username,
					"shortcode": edge.Node.Shortcode,
				}).Error("Failed to submit download job")
				continue
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
	
	ui.PrintSuccess("\n[EXTRACTION COMPLETED SUCCESSFULLY]\n")
	return nil
}

// getUserID fetches the user ID for the given username
func (s *Scraper) getUserID(username string) (string, error) {
	endpoint := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	
	s.logger.DebugWithFields("Making API request for user ID", map[string]interface{}{
		"username": username,
		"endpoint": endpoint,
	})
	
	var result instagram.InstagramResponse
	err := s.client.GetJSON(endpoint, &result)
	if err != nil {
		s.logger.WithError(err).WithField("username", username).Error("Failed to get user ID")
		return "", fmt.Errorf("failed to fetch user profile: %w", err)
	}

	if result.RequiresToLogin {
		s.logger.WarnWithFields("Profile requires authentication", map[string]interface{}{
			"username": username,
		})
		return "", fmt.Errorf("this profile requires authentication")
	}

	s.logger.DebugWithFields("Successfully fetched user ID", map[string]interface{}{
		"username": username,
		"user_id":  result.Data.User.ID,
	})
	
	return result.Data.User.ID, nil
}

// fetchMediaBatch fetches a batch of media items
func (s *Scraper) fetchMediaBatch(username, userID, endCursor string) ([]instagram.Edge, instagram.PageInfo, error) {
	var endpoint string
	if endCursor == "" {
		endpoint = fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	} else {
		variables := fmt.Sprintf(`{"id":"%s","first":50,"after":"%s"}`, userID, endCursor)
		endpoint = fmt.Sprintf("https://www.instagram.com/graphql/query/?query_hash=69cba40317214236af40e7efa697781d&variables=%s", variables)
	}
	
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
			s.tracker.IncrementDownloaded()
			s.tracker.PrintProgress()
			
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
			ui.PrintError("\nError downloading %s: %v\n", result.Job.Shortcode, result.Error)
			
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
