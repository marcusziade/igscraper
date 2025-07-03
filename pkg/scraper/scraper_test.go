package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/errors"
	"igscraper/pkg/instagram"
	"igscraper/pkg/ratelimit"
	"igscraper/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockInstagramServer creates a test server that mimics Instagram API
type mockInstagramServer struct {
	server          *httptest.Server
	profileCalls    int32
	mediaCalls      int32
	downloadCalls   int32
	failProfile     bool
	failMedia       bool
	failDownload    bool
	requiresLogin   bool
	mu              sync.Mutex
}

func newMockInstagramServer() *mockInstagramServer {
	m := &mockInstagramServer{}
	
	mux := http.NewServeMux()
	
	// Profile endpoint
	mux.HandleFunc("/api/v1/users/web_profile_info/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&m.profileCalls, 1)
		
		m.mu.Lock()
		defer m.mu.Unlock()
		
		if m.failProfile {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		_ = r.URL.Query().Get("username") // username would be used in a real implementation
		response := instagram.InstagramResponse{
			Status:          "ok",
			RequiresToLogin: m.requiresLogin,
			Data: instagram.Data{
				User: instagram.User{
					ID: "123456",
					EdgeOwnerToTimelineMedia: instagram.EdgeOwnerToTimelineMedia{
						Edges: []instagram.Edge{
							{
								Node: instagram.Node{
									ID:         "media1",
									Shortcode:  "ABC123",
									DisplayURL: fmt.Sprintf("%s/photos/photo1.jpg", m.server.URL),
									IsVideo:    false,
								},
							},
							{
								Node: instagram.Node{
									ID:         "media2",
									Shortcode:  "DEF456",
									DisplayURL: fmt.Sprintf("%s/photos/photo2.jpg", m.server.URL),
									IsVideo:    true, // This should be skipped
								},
							},
						},
						PageInfo: instagram.PageInfo{
							HasNextPage: true,
							EndCursor:   "cursor1",
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	// Media pagination endpoint
	mux.HandleFunc("/graphql/query/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&m.mediaCalls, 1)
		
		m.mu.Lock()
		defer m.mu.Unlock()
		
		if m.failMedia {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		variables := r.URL.Query().Get("variables")
		
		// Default response for first page
		response := instagram.InstagramResponse{
			Status: "ok",
			Data: instagram.Data{
				User: instagram.User{
					EdgeOwnerToTimelineMedia: instagram.EdgeOwnerToTimelineMedia{
						Edges: []instagram.Edge{
							{
								Node: instagram.Node{
									ID:         "media1",
									Shortcode:  "ABC123",
									DisplayURL: fmt.Sprintf("%s/photos/photo1.jpg", m.server.URL),
									IsVideo:    false,
								},
							},
							{
								Node: instagram.Node{
									ID:         "media2",
									Shortcode:  "DEF456",
									DisplayURL: fmt.Sprintf("%s/photos/photo2.jpg", m.server.URL),
									IsVideo:    true, // This is a video
								},
							},
						},
						PageInfo: instagram.PageInfo{
							HasNextPage: true,
							EndCursor:   "cursor1",
						},
					},
				},
			},
		}
		
		// Check if this is a second page request (has after parameter)
		if variables != "" && strings.Contains(variables, `"after":"cursor1"`) {
			// Return data for second page
			response.Data.User.EdgeOwnerToTimelineMedia = instagram.EdgeOwnerToTimelineMedia{
				Edges: []instagram.Edge{
					{
						Node: instagram.Node{
							ID:         "media3",
							Shortcode:  "GHI789",
							DisplayURL: fmt.Sprintf("%s/photos/photo3.jpg", m.server.URL),
							IsVideo:    false,
						},
					},
				},
				PageInfo: instagram.PageInfo{
					HasNextPage: false,
					EndCursor:   "",
				},
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	// Photo download endpoint
	mux.HandleFunc("/photos/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&m.downloadCalls, 1)
		
		m.mu.Lock()
		defer m.mu.Unlock()
		
		if m.failDownload {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		// Return fake image data
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("fake image data"))
	})
	
	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockInstagramServer) Close() {
	m.server.Close()
}

func (m *mockInstagramServer) URL() string {
	return m.server.URL
}

func (m *mockInstagramServer) GetCallCounts() (profile, media, download int32) {
	return atomic.LoadInt32(&m.profileCalls),
		atomic.LoadInt32(&m.mediaCalls),
		atomic.LoadInt32(&m.downloadCalls)
}

// mockInstagramClient is a mock implementation of InstagramClient interface
type mockInstagramClient struct {
	getJSON       func(url string, target interface{}) error
	downloadPhoto func(photoURL string) ([]byte, error)
}

func (m *mockInstagramClient) GetJSON(url string, target interface{}) error {
	if m.getJSON != nil {
		return m.getJSON(url, target)
	}
	return nil
}

func (m *mockInstagramClient) DownloadPhoto(photoURL string) ([]byte, error) {
	if m.downloadPhoto != nil {
		return m.downloadPhoto(photoURL)
	}
	return nil, nil
}

func (m *mockInstagramClient) FetchUserProfile(username string) (*instagram.InstagramResponse, error) {
	// For simplicity in tests, we can use GetJSON internally
	var response instagram.InstagramResponse
	url := instagram.GetProfileURL(username)
	err := m.GetJSON(url, &response)
	return &response, err
}

func (m *mockInstagramClient) FetchUserMedia(userID string, after string) (*instagram.InstagramResponse, error) {
	// For simplicity in tests, we can use GetJSON internally
	var response instagram.InstagramResponse
	url := instagram.GetMediaURL(userID, after)
	err := m.GetJSON(url, &response)
	return &response, err
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Instagram: config.InstagramConfig{
			SessionID: "test_session",
			CSRFToken: "test_csrf",
			UserAgent: "test_agent",
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 30,
		},
		Retry: config.RetryConfig{
			Enabled:     true,
			MaxAttempts: 3,
		},
		Download: config.DownloadConfig{
			DownloadTimeout: 30 * time.Second,
		},
	}
	
	scraper, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, scraper)
	assert.NotNil(t, scraper.client)
	assert.NotNil(t, scraper.rateLimiter)
	assert.NotNil(t, scraper.tracker)
	assert.NotNil(t, scraper.notifier)
	assert.Equal(t, cfg, scraper.config)
}

func TestGetOutputDir(t *testing.T) {
	tests := []struct {
		name              string
		createUserFolders bool
		baseDir           string
		username          string
		expected          string
	}{
		{
			name:              "with user folders",
			createUserFolders: true,
			baseDir:           "/tmp/downloads",
			username:          "testuser",
			expected:          "/tmp/downloads/testuser_photos",
		},
		{
			name:              "without user folders",
			createUserFolders: false,
			baseDir:           "/tmp/downloads",
			username:          "testuser",
			expected:          "/tmp/downloads",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Output.CreateUserFolders = tt.createUserFolders
			cfg.Output.BaseDirectory = tt.baseDir
			
			scraper, err := New(cfg)
			require.NoError(t, err)
			
			result := scraper.getOutputDir(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		shortcode string
		expected  string
	}{
		{
			name:      "default pattern",
			pattern:   "{shortcode}.jpg",
			shortcode: "ABC123",
			expected:  "ABC123.jpg",
		},
		{
			name:      "with timestamp pattern",
			pattern:   "{shortcode}_{timestamp}.jpg",
			shortcode: "ABC123",
			expected:  "ABC123_", // Timestamp will vary
		},
		{
			name:      "with date pattern",
			pattern:   "{date}_{shortcode}.jpg",
			shortcode: "ABC123",
			expected:  time.Now().Format("2006-01-02") + "_ABC123.jpg",
		},
		{
			name:      "no extension adds jpg",
			pattern:   "{shortcode}",
			shortcode: "ABC123",
			expected:  "ABC123.jpg",
		},
		{
			name:      "empty pattern uses default",
			pattern:   "",
			shortcode: "ABC123",
			expected:  "ABC123.jpg",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Output.FileNamePattern = tt.pattern
			
			scraper, err := New(cfg)
			require.NoError(t, err)
			
			result := scraper.generateFilename(tt.shortcode)
			
			if tt.name == "with timestamp pattern" {
				assert.Contains(t, result, tt.expected)
				assert.Contains(t, result, ".jpg")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// mockTransport wraps http.RoundTripper to redirect requests to test server
type mockTransport struct {
	testServerURL string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Parse the original URL and redirect to test server
	testURL, _ := neturl.Parse(t.testServerURL)
	req.URL.Scheme = testURL.Scheme
	req.URL.Host = testURL.Host
	
	// Use default transport to make the actual request
	return http.DefaultTransport.RoundTrip(req)
}

func TestGetUserID(t *testing.T) {
	server := newMockInstagramServer()
	defer server.Close()
	
	cfg := config.DefaultConfig()
	scraper, err := New(cfg)
	require.NoError(t, err)
	
	// Create a mock client that redirects to test server
	scraper.client = &mockInstagramClient{
		getJSON: func(url string, target interface{}) error {
			// Replace Instagram URL with test server URL
			testURL := url
			if strings.Contains(url, "/api/v1/users/web_profile_info/") {
				testURL = server.URL() + "/api/v1/users/web_profile_info/?username=" + strings.Split(url, "username=")[1]
			} else if strings.Contains(url, "/graphql/query/") {
				testURL = server.URL() + "/graphql/query/"
			}
			
			resp, err := http.Get(testURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				return &errors.Error{
					Type:    errors.ErrorTypeServerError,
					Message: "server error",
					Code:    resp.StatusCode,
				}
			}
			
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			
			return json.Unmarshal(body, target)
		},
	}
	
	t.Run("successful fetch", func(t *testing.T) {
		userID, err := scraper.getUserID("testuser")
		require.NoError(t, err)
		assert.Equal(t, "123456", userID)
		
		profile, _, _ := server.GetCallCounts()
		assert.Equal(t, int32(1), profile)
	})
	
	t.Run("requires login", func(t *testing.T) {
		server.mu.Lock()
		server.requiresLogin = true
		server.mu.Unlock()
		
		userID, err := scraper.getUserID("privateuser")
		assert.Error(t, err)
		assert.Empty(t, userID)
		assert.Contains(t, err.Error(), "authentication")
	})
	
	t.Run("server error", func(t *testing.T) {
		server.mu.Lock()
		server.failProfile = true
		server.requiresLogin = false
		server.mu.Unlock()
		
		userID, err := scraper.getUserID("testuser")
		assert.Error(t, err)
		assert.Empty(t, userID)
	})
}

func TestFetchMediaBatch(t *testing.T) {
	server := newMockInstagramServer()
	defer server.Close()
	
	cfg := config.DefaultConfig()
	scraper, err := New(cfg)
	require.NoError(t, err)
	
	// Create a test-specific client
	scraper.client = &mockInstagramClient{
		getJSON: func(url string, target interface{}) error {
			var testURL string
			if strings.Contains(url, "/api/v1/users/web_profile_info/") {
				testURL = server.URL() + "/api/v1/users/web_profile_info/?username=testuser"
			} else if strings.Contains(url, "/graphql/query/") {
				// Parse the URL to get query parameters
				u, _ := neturl.Parse(url)
				testURL = server.URL() + "/graphql/query/?" + u.RawQuery
			} else {
				testURL = server.URL() + url
			}
			
			resp, err := http.Get(testURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				return &errors.Error{
					Type:    errors.ErrorTypeServerError,
					Message: "server error",
					Code:    resp.StatusCode,
				}
			}
			
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			
			return json.Unmarshal(body, target)
		},
	}
	
	t.Run("first page from profile", func(t *testing.T) {
		media, pageInfo, err := scraper.fetchMediaBatch("testuser", "123456", "")
		require.NoError(t, err)
		assert.Len(t, media, 2) // 2 items (1 photo, 1 video)
		assert.True(t, pageInfo.HasNextPage)
		assert.Equal(t, "cursor1", pageInfo.EndCursor)
	})
	
	t.Run("subsequent page", func(t *testing.T) {
		media, pageInfo, err := scraper.fetchMediaBatch("testuser", "123456", "cursor1")
		require.NoError(t, err)
		assert.Len(t, media, 1) // 1 more photo
		assert.False(t, pageInfo.HasNextPage)
		assert.Empty(t, pageInfo.EndCursor)
	})
	
	t.Run("server error", func(t *testing.T) {
		server.mu.Lock()
		server.failMedia = true
		server.mu.Unlock()
		
		media, pageInfo, err := scraper.fetchMediaBatch("testuser", "123456", "cursor1")
		assert.Error(t, err)
		assert.Nil(t, media)
		assert.Equal(t, instagram.PageInfo{}, pageInfo)
		
		server.mu.Lock()
		server.failMedia = false
		server.mu.Unlock()
	})
}

func TestDownloadPhoto(t *testing.T) {
	server := newMockInstagramServer()
	defer server.Close()
	
	// Create temp directory for tests
	tempDir := t.TempDir()
	
	cfg := config.DefaultConfig()
	cfg.Output.BaseDirectory = tempDir
	
	scraper, err := New(cfg)
	require.NoError(t, err)
	
	// Set up storage manager
	scraper.storageManager, err = storage.NewManager(tempDir)
	require.NoError(t, err)
	
	// Create test client
	scraper.client = &mockInstagramClient{
		downloadPhoto: func(url string) ([]byte, error) {
			resp, err := http.Get(url)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				return nil, &errors.Error{
					Type:    errors.ErrorTypeServerError,
					Message: "download failed",
					Code:    resp.StatusCode,
				}
			}
			
			return io.ReadAll(resp.Body)
		},
	}
	
	t.Run("successful download", func(t *testing.T) {
		photoURL := server.URL() + "/photos/photo1.jpg"
		err := scraper.downloadPhoto(photoURL, "ABC123")
		require.NoError(t, err)
		
		// Check file exists
		expectedPath := filepath.Join(tempDir, "ABC123.jpg")
		_, err = os.Stat(expectedPath)
		assert.NoError(t, err)
		
		// Check content
		data, err := os.ReadFile(expectedPath)
		require.NoError(t, err)
		assert.Equal(t, "fake image data", string(data))
	})
	
	t.Run("download error", func(t *testing.T) {
		server.mu.Lock()
		server.failDownload = true
		server.mu.Unlock()
		
		photoURL := server.URL() + "/photos/photo2.jpg"
		err := scraper.downloadPhoto(photoURL, "DEF456")
		assert.Error(t, err)
		
		// File should not exist
		expectedPath := filepath.Join(tempDir, "DEF456.jpg")
		_, err = os.Stat(expectedPath)
		assert.True(t, os.IsNotExist(err))
		
		server.mu.Lock()
		server.failDownload = false
		server.mu.Unlock()
	})
}

func TestRateLimiting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.RateLimit.RequestsPerMinute = 2 // Very low for testing
	
	scraper, err := New(cfg)
	require.NoError(t, err)
	
	// Replace with a custom rate limiter for testing
	scraper.rateLimiter = ratelimit.NewTokenBucket(2, time.Second)
	
	// First two requests should be allowed immediately
	assert.True(t, scraper.rateLimiter.Allow())
	assert.True(t, scraper.rateLimiter.Allow())
	
	// Third request should be rate limited
	assert.False(t, scraper.rateLimiter.Allow())
	
	// Wait for rate limit to reset
	time.Sleep(time.Second)
	assert.True(t, scraper.rateLimiter.Allow())
}

func TestConcurrentDownloads(t *testing.T) {
	server := newMockInstagramServer()
	defer server.Close()
	
	tempDir := t.TempDir()
	
	cfg := config.DefaultConfig()
	cfg.Output.BaseDirectory = tempDir
	cfg.Download.ConcurrentDownloads = 3
	
	scraper, err := New(cfg)
	require.NoError(t, err)
	
	// Use real storage manager
	scraper.storageManager, err = storage.NewManager(tempDir)
	require.NoError(t, err)
	
	// Download multiple photos concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			photoURL := fmt.Sprintf("%s/photos/photo%d.jpg", server.URL(), i)
			shortcode := fmt.Sprintf("CODE%d", i)
			_ = scraper.downloadPhoto(photoURL, shortcode)
		}(i)
	}
	
	wg.Wait()
	
	// Check that all files were downloaded
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 10, len(files))
}

func TestErrorRecovery(t *testing.T) {
	server := newMockInstagramServer()
	defer server.Close()
	
	tempDir := t.TempDir()
	
	cfg := config.DefaultConfig()
	cfg.Output.BaseDirectory = tempDir
	
	scraper, err := New(cfg)
	require.NoError(t, err)
	
	scraper.storageManager, err = storage.NewManager(tempDir)
	require.NoError(t, err)
	
	t.Run("download failure", func(t *testing.T) {
		// Test that download errors are properly propagated
		scraper.client = &mockInstagramClient{
			downloadPhoto: func(url string) ([]byte, error) {
				return nil, &errors.Error{
					Type:    errors.ErrorTypeNetwork,
					Message: "network error",
					Code:    0,
				}
			},
		}
		
		err := scraper.downloadPhoto("http://example.com/photo.jpg", "FAIL123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network error")
	})
	
	t.Run("successful download after client retry", func(t *testing.T) {
		// Test successful download (retry logic is in the real client)
		scraper.client = &mockInstagramClient{
			downloadPhoto: func(url string) ([]byte, error) {
				return []byte("success data"), nil
			},
		}
		
		err := scraper.downloadPhoto("http://example.com/photo.jpg", "SUCCESS123")
		require.NoError(t, err)
		
		// Verify file was saved
		expectedPath := filepath.Join(tempDir, "SUCCESS123.jpg")
		data, err := os.ReadFile(expectedPath)
		require.NoError(t, err)
		assert.Equal(t, "success data", string(data))
	})
}

// Benchmark tests
func BenchmarkDownloadPhoto(b *testing.B) {
	server := newMockInstagramServer()
	defer server.Close()
	
	tempDir := b.TempDir()
	cfg := config.DefaultConfig()
	cfg.Output.BaseDirectory = tempDir
	
	scraper, _ := New(cfg)
	scraper.storageManager, _ = storage.NewManager(tempDir)
	
	scraper.client = &mockInstagramClient{
		downloadPhoto: func(url string) ([]byte, error) {
			return []byte("benchmark image data"), nil
		},
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		photoURL := fmt.Sprintf("http://example.com/photo%d.jpg", i)
		shortcode := fmt.Sprintf("BENCH%d", i)
		_ = scraper.downloadPhoto(photoURL, shortcode)
	}
}

func BenchmarkConcurrentDownloads(b *testing.B) {
	server := newMockInstagramServer()
	defer server.Close()
	
	tempDir := b.TempDir()
	cfg := config.DefaultConfig()
	cfg.Output.BaseDirectory = tempDir
	cfg.Download.ConcurrentDownloads = 5
	
	scraper, _ := New(cfg)
	scraper.storageManager, _ = storage.NewManager(tempDir)
	
	scraper.client = &mockInstagramClient{
		downloadPhoto: func(url string) ([]byte, error) {
			return []byte("benchmark image data"), nil
		},
	}
	
	b.ResetTimer()
	
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			photoURL := fmt.Sprintf("http://example.com/photo%d.jpg", i)
			shortcode := fmt.Sprintf("BENCH%d", i)
			_ = scraper.downloadPhoto(photoURL, shortcode)
		}(i)
		
		if i%cfg.Download.ConcurrentDownloads == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
}