package instagram

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProfileURL(t *testing.T) {
	tests := []struct {
		name     string
		username string
		expected string
	}{
		{
			name:     "simple username",
			username: "testuser",
			expected: fmt.Sprintf("%s%s?username=testuser", BaseURL, ProfileEndpoint),
		},
		{
			name:     "username with underscore",
			username: "test_user",
			expected: fmt.Sprintf("%s%s?username=test_user", BaseURL, ProfileEndpoint),
		},
		{
			name:     "username with dots",
			username: "test.user",
			expected: fmt.Sprintf("%s%s?username=test.user", BaseURL, ProfileEndpoint),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProfileURL(tt.username)
			assert.Equal(t, tt.expected, result)
			
			// Verify URL is properly encoded
			_, err := url.Parse(result)
			assert.NoError(t, err)
		})
	}
}

func TestGetMediaURL(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		after    string
		expected string
	}{
		{
			name:   "without cursor",
			userID: "123456",
			after:  "",
			expected: fmt.Sprintf(`%s%s?query_hash=%s&variables={"id":"123456","first":%d,"after":""}`,
				BaseURL, MediaEndpoint, MediaQueryHash, DefaultMediaLimit),
		},
		{
			name:   "with cursor",
			userID: "123456",
			after:  "cursor123",
			expected: fmt.Sprintf(`%s%s?query_hash=%s&variables={"id":"123456","first":%d,"after":"cursor123"}`,
				BaseURL, MediaEndpoint, MediaQueryHash, DefaultMediaLimit),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMediaURL(tt.userID, tt.after)
			// URL encode the expected value for comparison
			expected, _ := url.Parse(tt.expected)
			actual, _ := url.Parse(result)
			
			assert.Equal(t, expected.Path, actual.Path)
			assert.Equal(t, expected.Query().Get("query_hash"), actual.Query().Get("query_hash"))
			
			// Check variables parameter contains the right values
			vars := actual.Query().Get("variables")
			assert.Contains(t, vars, tt.userID)
			if tt.after != "" {
				assert.Contains(t, vars, tt.after)
			}
		})
	}
}

func TestGetMediaURLWithLimit(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		after    string
		limit    int
		expected int
	}{
		{
			name:     "default limit when zero",
			userID:   "123456",
			after:    "",
			limit:    0,
			expected: DefaultMediaLimit,
		},
		{
			name:     "negative limit uses default",
			userID:   "123456",
			after:    "",
			limit:    -5,
			expected: DefaultMediaLimit,
		},
		{
			name:     "custom limit within bounds",
			userID:   "123456",
			after:    "",
			limit:    25,
			expected: 25,
		},
		{
			name:     "limit exceeds maximum",
			userID:   "123456",
			after:    "",
			limit:    100,
			expected: MaxMediaLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMediaURLWithLimit(tt.userID, tt.after, tt.limit)
			
			// Parse URL and check the limit in variables
			parsed, err := url.Parse(result)
			assert.NoError(t, err)
			
			vars := parsed.Query().Get("variables")
			expectedVars := fmt.Sprintf(`"first":%d`, tt.expected)
			assert.Contains(t, vars, expectedVars)
		})
	}
}

func TestGetPhotoURL(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected string
	}{
		{
			name: "valid node",
			node: &Node{
				DisplayURL: "https://example.com/photo.jpg",
			},
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "nil node",
			node:     nil,
			expected: "",
		},
		{
			name: "empty display URL",
			node: &Node{
				DisplayURL: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPhotoURL(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPostURL(t *testing.T) {
	tests := []struct {
		name      string
		shortcode string
		expected  string
	}{
		{
			name:      "valid shortcode",
			shortcode: "ABC123xyz",
			expected:  fmt.Sprintf("%s/p/ABC123xyz/", BaseURL),
		},
		{
			name:      "empty shortcode",
			shortcode: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPostURL(tt.shortcode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserProfileURL(t *testing.T) {
	tests := []struct {
		name     string
		username string
		expected string
	}{
		{
			name:     "valid username",
			username: "testuser",
			expected: fmt.Sprintf("%s/testuser/", BaseURL),
		},
		{
			name:     "empty username",
			username: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserProfileURL(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		expected bool
	}{
		{
			name:     "valid simple username",
			username: "testuser",
			expected: true,
		},
		{
			name:     "valid with underscore",
			username: "test_user",
			expected: true,
		},
		{
			name:     "valid with dot",
			username: "test.user",
			expected: true,
		},
		{
			name:     "valid with numbers",
			username: "user123",
			expected: true,
		},
		{
			name:     "valid uppercase",
			username: "TestUser",
			expected: true,
		},
		{
			name:     "empty username",
			username: "",
			expected: false,
		},
		{
			name:     "too long",
			username: "thisusernameiswaytoolongandexceedsthirtychars",
			expected: false,
		},
		{
			name:     "invalid with space",
			username: "test user",
			expected: false,
		},
		{
			name:     "invalid with hyphen",
			username: "test-user",
			expected: false,
		},
		{
			name:     "invalid with special char",
			username: "test@user",
			expected: false,
		},
		{
			name:     "invalid with emoji",
			username: "test😀user",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidUsername(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		expected string
	}{
		{
			name:     "clean username",
			username: "testuser",
			expected: "testuser",
		},
		{
			name:     "username with @ prefix",
			username: "@testuser",
			expected: "testuser",
		},
		{
			name:     "username with trailing slash",
			username: "testuser/",
			expected: "testuser",
		},
		{
			name:     "username with trailing space",
			username: "testuser ",
			expected: "testuser",
		},
		{
			name:     "username with multiple trailing chars",
			username: "testuser// ",
			expected: "testuser",
		},
		{
			name:     "username with @ and trailing slash",
			username: "@testuser/",
			expected: "testuser",
		},
		{
			name:     "empty username",
			username: "",
			expected: "",
		},
		{
			name:     "just @",
			username: "@",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeUsername(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLConstruction(t *testing.T) {
	t.Run("base URL is HTTPS", func(t *testing.T) {
		assert.True(t, len(BaseURL) > 0)
		assert.Contains(t, BaseURL, "https://")
		assert.Contains(t, BaseURL, "instagram.com")
	})

	t.Run("endpoints start with slash", func(t *testing.T) {
		assert.True(t, len(ProfileEndpoint) > 0)
		assert.Equal(t, "/", string(ProfileEndpoint[0]))
		
		assert.True(t, len(MediaEndpoint) > 0)
		assert.Equal(t, "/", string(MediaEndpoint[0]))
	})

	t.Run("media limits are reasonable", func(t *testing.T) {
		assert.Greater(t, DefaultMediaLimit, 0)
		assert.LessOrEqual(t, DefaultMediaLimit, MaxMediaLimit)
		assert.Greater(t, MaxMediaLimit, 0)
		assert.LessOrEqual(t, MaxMediaLimit, 100) // Instagram typically limits to 50
	})

	t.Run("query hash is valid", func(t *testing.T) {
		assert.True(t, len(MediaQueryHash) > 0)
		// Query hash should be alphanumeric
		for _, char := range MediaQueryHash {
			assert.True(t, (char >= 'a' && char <= 'z') || 
				(char >= 'A' && char <= 'Z') || 
				(char >= '0' && char <= '9'),
				"Query hash contains invalid character: %c", char)
		}
	})
}

func BenchmarkGetProfileURL(b *testing.B) {
	username := "testuser"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = GetProfileURL(username)
	}
}

func BenchmarkGetMediaURL(b *testing.B) {
	userID := "123456789"
	cursor := "QVFCdGVzdGN1cnNvcg=="
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = GetMediaURL(userID, cursor)
	}
}

func BenchmarkIsValidUsername(b *testing.B) {
	username := "test_user.123"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = IsValidUsername(username)
	}
}

func BenchmarkSanitizeUsername(b *testing.B) {
	username := "@testuser/"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = SanitizeUsername(username)
	}
}