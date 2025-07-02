package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"igscraper/pkg/logger"
	"igscraper/pkg/ratelimit"
)

// DownloadJob represents a single download task
type DownloadJob struct {
	URL       string
	Shortcode string
	Username  string
}

// DownloadResult represents the result of a download job
type DownloadResult struct {
	Job      DownloadJob
	Success  bool
	Error    error
	Duration time.Duration
	Size     int
}

// PhotoDownloader interface for downloading photos
type PhotoDownloader interface {
	DownloadPhoto(url string) ([]byte, error)
}

// PhotoStorage interface for storing photos
type PhotoStorage interface {
	IsDownloaded(shortcode string) bool
	SavePhoto(r io.Reader, shortcode string) error
}

// WorkerPool manages concurrent download workers
type WorkerPool struct {
	numWorkers     int
	jobQueue       chan DownloadJob
	resultQueue    chan DownloadResult
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	client         PhotoDownloader
	storageManager PhotoStorage
	rateLimiter    ratelimit.Limiter
	logger         logger.Logger
}

// NewWorkerPool creates a new download worker pool
func NewWorkerPool(
	numWorkers int,
	client PhotoDownloader,
	storageManager PhotoStorage,
	rateLimiter ratelimit.Limiter,
	log logger.Logger,
) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	if log == nil {
		log = logger.GetLogger()
	}
	
	return &WorkerPool{
		numWorkers:     numWorkers,
		jobQueue:       make(chan DownloadJob, numWorkers*2), // Buffer size = 2x workers
		resultQueue:    make(chan DownloadResult, numWorkers),
		ctx:            ctx,
		cancel:         cancel,
		client:         client,
		storageManager: storageManager,
		rateLimiter:    rateLimiter,
		logger:         log,
	}
}

// Start initializes and starts all workers
func (wp *WorkerPool) Start() {
	wp.logger.InfoWithFields("Starting worker pool", map[string]interface{}{
		"num_workers": wp.numWorkers,
	})
	
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() {
	wp.logger.Info("Stopping worker pool...")
	
	// Close job queue to signal no more jobs will be added
	close(wp.jobQueue)
	
	// Wait for all workers to finish processing remaining jobs
	wp.wg.Wait()
	
	// Close result queue
	close(wp.resultQueue)
	
	// Cancel context
	wp.cancel()
	
	wp.logger.Info("Worker pool stopped")
}

// Submit adds a new download job to the queue
func (wp *WorkerPool) Submit(job DownloadJob) error {
	select {
	case wp.jobQueue <- job:
		wp.logger.DebugWithFields("Job submitted to queue", map[string]interface{}{
			"shortcode": job.Shortcode,
			"username":  job.Username,
		})
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	}
}

// Results returns the result channel for consuming download results
func (wp *WorkerPool) Results() <-chan DownloadResult {
	return wp.resultQueue
}

// worker is the main worker routine
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	wp.logger.DebugWithFields("Worker started", map[string]interface{}{
		"worker_id": id,
	})
	
	for job := range wp.jobQueue {
		// Check if context is cancelled
		select {
		case <-wp.ctx.Done():
			wp.logger.DebugWithFields("Worker stopping - context cancelled", map[string]interface{}{
				"worker_id": id,
			})
			return
		default:
		}
		
		// Process the job
		result := wp.processJob(job, id)
		
		// Send result
		select {
		case wp.resultQueue <- result:
		case <-wp.ctx.Done():
			wp.logger.DebugWithFields("Worker stopping - context cancelled while sending result", map[string]interface{}{
				"worker_id": id,
			})
			return
		}
	}
	
	wp.logger.DebugWithFields("Worker stopping - job queue closed", map[string]interface{}{
		"worker_id": id,
	})
}

// processJob handles a single download job
func (wp *WorkerPool) processJob(job DownloadJob, workerID int) DownloadResult {
	start := time.Now()
	result := DownloadResult{
		Job:     job,
		Success: false,
	}
	
	wp.logger.DebugWithFields("Worker processing job", map[string]interface{}{
		"worker_id": workerID,
		"shortcode": job.Shortcode,
		"username":  job.Username,
	})
	
	// Check if already downloaded
	if wp.storageManager.IsDownloaded(job.Shortcode) {
		wp.logger.DebugWithFields("Photo already downloaded", map[string]interface{}{
			"worker_id": workerID,
			"shortcode": job.Shortcode,
		})
		result.Success = true
		result.Duration = time.Since(start)
		return result
	}
	
	// Wait for rate limit
	if !wp.rateLimiter.Allow() {
		wp.logger.DebugWithFields("Worker waiting for rate limit", map[string]interface{}{
			"worker_id": workerID,
			"shortcode": job.Shortcode,
		})
		wp.rateLimiter.Wait()
	}
	
	// Download the photo
	data, err := wp.client.DownloadPhoto(job.URL)
	if err != nil {
		result.Error = fmt.Errorf("download failed: %w", err)
		result.Duration = time.Since(start)
		
		wp.logger.ErrorWithFields("Worker failed to download photo", map[string]interface{}{
			"worker_id": workerID,
			"shortcode": job.Shortcode,
			"error":     err.Error(),
			"duration":  result.Duration,
		})
		
		return result
	}
	
	result.Size = len(data)
	
	// Save the photo
	err = wp.storageManager.SavePhoto(bytes.NewReader(data), job.Shortcode)
	if err != nil {
		result.Error = fmt.Errorf("save failed: %w", err)
		result.Duration = time.Since(start)
		
		wp.logger.ErrorWithFields("Worker failed to save photo", map[string]interface{}{
			"worker_id": workerID,
			"shortcode": job.Shortcode,
			"error":     err.Error(),
			"size":      result.Size,
		})
		
		return result
	}
	
	result.Success = true
	result.Duration = time.Since(start)
	
	wp.logger.DebugWithFields("Worker completed job successfully", map[string]interface{}{
		"worker_id": workerID,
		"shortcode": job.Shortcode,
		"size":      result.Size,
		"duration":  result.Duration,
	})
	
	return result
}

// GetQueueSize returns the current number of jobs in the queue
func (wp *WorkerPool) GetQueueSize() int {
	return len(wp.jobQueue)
}

// GetActiveWorkers returns the number of active workers
func (wp *WorkerPool) GetActiveWorkers() int {
	return wp.numWorkers
}