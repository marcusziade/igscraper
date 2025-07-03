package downloader

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"igscraper/pkg/instagram"
	"igscraper/pkg/ratelimit"
)

// MockClient is a mock implementation of the Instagram client
type MockClient struct {
	downloadDelay   time.Duration
	downloadError   error
	downloadCounter int32
}

func (m *MockClient) DownloadPhoto(url string) ([]byte, error) {
	atomic.AddInt32(&m.downloadCounter, 1)
	if m.downloadDelay > 0 {
		time.Sleep(m.downloadDelay)
	}
	if m.downloadError != nil {
		return nil, m.downloadError
	}
	return []byte("mock photo data"), nil
}

func (m *MockClient) GetDownloadCount() int {
	return int(atomic.LoadInt32(&m.downloadCounter))
}

// MockStorageManager is a mock implementation of the storage manager
type MockStorageManager struct {
	savedPhotos map[string]bool
	saveError   error
	mu          sync.Mutex
}

func NewMockStorageManager() *MockStorageManager {
	return &MockStorageManager{
		savedPhotos: make(map[string]bool),
	}
}

func (m *MockStorageManager) IsDownloaded(shortcode string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.savedPhotos[shortcode]
}

func (m *MockStorageManager) SavePhoto(r io.Reader, shortcode string) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedPhotos[shortcode] = true
	return nil
}

func (m *MockStorageManager) SavePhotoWithMetadata(r io.Reader, shortcode string, node *instagram.Node) error {
	// For testing, just call SavePhoto since we don't need to test metadata saving
	return m.SavePhoto(r, shortcode)
}

func (m *MockStorageManager) GetSavedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.savedPhotos)
}

func TestWorkerPoolBasicFunctionality(t *testing.T) {
	// Create mocks
	mockClient := &MockClient{downloadDelay: 10 * time.Millisecond}
	mockStorage := NewMockStorageManager()
	rateLimiter := ratelimit.NewTokenBucket(100, time.Second)
	
	// Create worker pool
	pool := NewWorkerPool(3, mockClient, mockStorage, rateLimiter, nil)
	pool.Start()
	
	// Collect results
	var results []DownloadResult
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			results = append(results, result)
		}
	}()
	
	// Submit jobs
	numJobs := 10
	for i := 0; i < numJobs; i++ {
		job := DownloadJob{
			URL:       fmt.Sprintf("https://example.com/photo%d.jpg", i),
			Shortcode: fmt.Sprintf("shortcode%d", i),
			Username:  "testuser",
		}
		err := pool.Submit(job)
		if err != nil {
			t.Errorf("Failed to submit job %d: %v", i, err)
		}
	}
	
	// Stop pool and wait for results
	pool.Stop()
	wg.Wait()
	
	// Verify results
	if len(results) != numJobs {
		t.Errorf("Expected %d results, got %d", numJobs, len(results))
	}
	
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	
	if successCount != numJobs {
		t.Errorf("Expected %d successful downloads, got %d", numJobs, successCount)
	}
	
	if mockClient.GetDownloadCount() != numJobs {
		t.Errorf("Expected %d download calls, got %d", numJobs, mockClient.GetDownloadCount())
	}
	
	if mockStorage.GetSavedCount() != numJobs {
		t.Errorf("Expected %d saved photos, got %d", numJobs, mockStorage.GetSavedCount())
	}
}

func TestWorkerPoolWithErrors(t *testing.T) {
	// Create mocks with error
	mockClient := &MockClient{
		downloadError: fmt.Errorf("download error"),
	}
	mockStorage := NewMockStorageManager()
	rateLimiter := ratelimit.NewTokenBucket(100, time.Second)
	
	// Create worker pool
	pool := NewWorkerPool(2, mockClient, mockStorage, rateLimiter, nil)
	pool.Start()
	
	// Collect results
	var results []DownloadResult
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			results = append(results, result)
		}
	}()
	
	// Submit jobs
	numJobs := 5
	for i := 0; i < numJobs; i++ {
		job := DownloadJob{
			URL:       fmt.Sprintf("https://example.com/photo%d.jpg", i),
			Shortcode: fmt.Sprintf("shortcode%d", i),
			Username:  "testuser",
		}
		err := pool.Submit(job)
		if err != nil {
			t.Errorf("Failed to submit job %d: %v", i, err)
		}
	}
	
	// Stop pool and wait for results
	pool.Stop()
	wg.Wait()
	
	// Verify all jobs failed
	if len(results) != numJobs {
		t.Errorf("Expected %d results, got %d", numJobs, len(results))
	}
	
	for _, result := range results {
		if result.Success {
			t.Error("Expected all downloads to fail")
		}
		if result.Error == nil {
			t.Error("Expected error in result")
		}
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	// Create mocks with delay to test concurrency
	mockClient := &MockClient{downloadDelay: 100 * time.Millisecond}
	mockStorage := NewMockStorageManager()
	rateLimiter := ratelimit.NewTokenBucket(100, time.Second)
	
	// Create worker pool with 5 workers
	pool := NewWorkerPool(5, mockClient, mockStorage, rateLimiter, nil)
	pool.Start()
	
	// Collect results
	var results []DownloadResult
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			results = append(results, result)
		}
	}()
	
	// Submit 10 jobs
	numJobs := 10
	startTime := time.Now()
	
	for i := 0; i < numJobs; i++ {
		job := DownloadJob{
			URL:       fmt.Sprintf("https://example.com/photo%d.jpg", i),
			Shortcode: fmt.Sprintf("shortcode%d", i),
			Username:  "testuser",
		}
		err := pool.Submit(job)
		if err != nil {
			t.Errorf("Failed to submit job %d: %v", i, err)
		}
	}
	
	// Stop pool and wait for results
	pool.Stop()
	wg.Wait()
	
	elapsed := time.Since(startTime)
	
	// With 5 workers and 10 jobs taking 100ms each, it should take ~200ms
	// Allow some buffer for overhead
	expectedTime := 300 * time.Millisecond
	if elapsed > expectedTime {
		t.Errorf("Downloads took too long: %v (expected < %v)", elapsed, expectedTime)
	}
	
	if len(results) != numJobs {
		t.Errorf("Expected %d results, got %d", numJobs, len(results))
	}
}

func TestWorkerPoolDuplicateDetection(t *testing.T) {
	// Create mocks
	mockClient := &MockClient{}
	mockStorage := NewMockStorageManager()
	
	// Pre-populate some "already downloaded" photos
	mockStorage.savedPhotos["existing1"] = true
	mockStorage.savedPhotos["existing2"] = true
	
	rateLimiter := ratelimit.NewTokenBucket(100, time.Second)
	
	// Create worker pool
	pool := NewWorkerPool(2, mockClient, mockStorage, rateLimiter, nil)
	pool.Start()
	
	// Collect results
	var results []DownloadResult
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			results = append(results, result)
		}
	}()
	
	// Submit jobs including duplicates
	jobs := []DownloadJob{
		{URL: "https://example.com/new1.jpg", Shortcode: "new1", Username: "testuser"},
		{URL: "https://example.com/existing1.jpg", Shortcode: "existing1", Username: "testuser"},
		{URL: "https://example.com/new2.jpg", Shortcode: "new2", Username: "testuser"},
		{URL: "https://example.com/existing2.jpg", Shortcode: "existing2", Username: "testuser"},
	}
	
	for _, job := range jobs {
		err := pool.Submit(job)
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}
	}
	
	// Stop pool and wait for results
	pool.Stop()
	wg.Wait()
	
	// Should have results for all jobs
	if len(results) != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), len(results))
	}
	
	// Only new photos should have been downloaded
	expectedDownloads := 2
	if mockClient.GetDownloadCount() != expectedDownloads {
		t.Errorf("Expected %d downloads, got %d", expectedDownloads, mockClient.GetDownloadCount())
	}
	
	// Total saved should be 4 (2 existing + 2 new)
	if mockStorage.GetSavedCount() != 4 {
		t.Errorf("Expected 4 saved photos, got %d", mockStorage.GetSavedCount())
	}
}