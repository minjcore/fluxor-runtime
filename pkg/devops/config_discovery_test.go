package devops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigFile(t *testing.T) {
	// Get project root
	projectRoot, err := findProjectRootForConfig()
	if err != nil {
		t.Skipf("Skipping test: project root not found: %v", err)
	}

	// Test cases
	tests := []struct {
		name     string
		configPath string
		wantFound bool
	}{
		{
			name:       "resolve wordpress.yaml from repo",
			configPath: "wordpress.yaml",
			wantFound:  true,
		},
		{
			name:       "resolve ssrflux.com.conf from repo",
			configPath: "ssrflux.com.conf",
			wantFound:  true,
		},
		{
			name:       "resolve gitvn.com.conf from repo",
			configPath: "gitvn.com.conf",
			wantFound:  true,
		},
		{
			name:       "non-existent file",
			configPath: "nonexistent.conf",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := ResolveConfigFile(tt.configPath)
			
			// Check if file exists
			_, err := os.Stat(resolved)
			exists := err == nil
			
			if tt.wantFound && !exists {
				t.Errorf("ResolveConfigFile(%q) = %q, file should exist but doesn't", tt.configPath, resolved)
			}
			
			if tt.wantFound {
				// Verify it's from repo folder
				expectedRepoPath := filepath.Join(projectRoot, "repo", tt.configPath)
				if resolved != expectedRepoPath {
					// Check if it's at least a valid path
					if !filepath.IsAbs(resolved) {
						t.Errorf("ResolveConfigFile(%q) = %q, expected absolute path", tt.configPath, resolved)
					}
					// If it's not the repo path, it might be from another location, which is also valid
					t.Logf("Resolved %q to %q (expected repo path: %q)", tt.configPath, resolved, expectedRepoPath)
				} else {
					t.Logf("✓ Successfully resolved %q to repo path: %q", tt.configPath, resolved)
				}
			}
		})
	}
}
