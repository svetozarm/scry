package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		{"github.com", "https://github.com/owner/repo/releases/download/v1/file.tar.gz", true},
		{"objects.githubusercontent.com", "https://objects.githubusercontent.com/path/file", true},
		{"http rejected", "http://github.com/file", false},
		{"attacker domain", "https://evil.com/scry.tar.gz", false},
		{"subdomain spoof", "https://github.com.evil.com/file", false},
		{"empty", "", false},
		{"no scheme", "github.com/file", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.allowed, isAllowedURL(tt.url))
		})
	}
}
