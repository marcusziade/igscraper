package tui

import (
	"testing"
	"time"
)

func TestModel(t *testing.T) {
	model := NewModel(3)

	// Test adding downloads
	model.AddDownload("id1", "user1", "photo1.jpg", 1024*1024)
	model.AddDownload("id2", "user1", "photo2.jpg", 2*1024*1024)

	if len(model.downloads) != 2 {
		t.Errorf("Expected 2 downloads, got %d", len(model.downloads))
	}

	// Test starting download
	model.StartDownload("id1")
	if model.activeDownloads != 1 {
		t.Errorf("Expected 1 active download, got %d", model.activeDownloads)
	}

	// Test updating progress
	model.UpdateDownloadProgress("id1", 512*1024, 1024*1024)
	download := model.downloads["id1"]
	if download.Downloaded != 512*1024 {
		t.Errorf("Expected downloaded to be %d, got %d", 512*1024, download.Downloaded)
	}

	// Test completing download
	model.CompleteDownload("id1")
	if model.activeDownloads != 0 {
		t.Errorf("Expected 0 active downloads, got %d", model.activeDownloads)
	}
	if model.totalDownloaded != 1 {
		t.Errorf("Expected 1 total downloaded, got %d", model.totalDownloaded)
	}

	// Test rate limit update
	resetTime := time.Now().Add(time.Hour)
	model.UpdateRateLimit(50, 100, resetTime)
	if model.rateLimitUsed != 50 {
		t.Errorf("Expected rate limit used to be 50, got %d", model.rateLimitUsed)
	}

	// Test log messages
	model.AddLogMessage("INFO", "Test message")
	if len(model.logMessages) != 1 {
		t.Errorf("Expected 1 log message, got %d", len(model.logMessages))
	}

	// Test GetActiveDownloads
	model.StartDownload("id2")
	active := model.GetActiveDownloads()
	if len(active) != 1 {
		t.Errorf("Expected 1 active download, got %d", len(active))
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024 * 1024, "5.0 GB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		speed    float64
		expected string
	}{
		{1024, "1.0 KB/s"},
		{1024 * 1024, "1.0 MB/s"},
		{512 * 1024, "512.0 KB/s"},
	}

	for _, test := range tests {
		result := FormatSpeed(test.speed)
		if result != test.expected {
			t.Errorf("FormatSpeed(%f) = %s, expected %s", test.speed, result, test.expected)
		}
	}
}