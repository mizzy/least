package terraform

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		wantResources int
		wantErrors    bool
	}{
		{
			name:          "simple - no modules",
			pattern:       "simple",
			wantResources: 2,
			wantErrors:    false,
		},
		{
			name:          "local-module - with local modules",
			pattern:       "local-module",
			wantResources: 7,
			wantErrors:    false,
		},
		{
			name:          "nested-modules - modules calling modules",
			pattern:       "nested-modules",
			wantResources: 5,
			wantErrors:    false,
		},
		{
			name:          "multi-call - same module multiple times",
			pattern:       "multi-call",
			wantResources: 2, // Same module dir parsed once
			wantErrors:    false,
		},
		{
			name:          "circular-ref - handles circular references",
			pattern:       "circular-ref",
			wantResources: 3,
			wantErrors:    false,
		},
		{
			name:          "multi-file - multiple tf files",
			pattern:       "multi-file",
			wantResources: 8, // includes aws_iam_role_policy
			wantErrors:    false,
		},
		{
			name:          "for-each - resources with for_each/count",
			pattern:       "for-each",
			wantResources: 5,
			wantErrors:    false,
		},
		{
			name:          "data-only - no resources",
			pattern:       "data-only",
			wantResources: 0,
			wantErrors:    false,
		},
		{
			name:          "no-resources - only variables",
			pattern:       "no-resources",
			wantResources: 0,
			wantErrors:    false,
		},
		{
			name:          "empty-module - module with no resources",
			pattern:       "empty-module",
			wantResources: 1,
			wantErrors:    false,
		},
		{
			name:          "remote-module - without terraform init",
			pattern:       "remote-module",
			wantResources: 1,
			wantErrors:    true, // Errors because remote modules not downloaded
		},
		{
			name:          "mixed-resources - various services",
			pattern:       "mixed-resources",
			wantResources: 8,
			wantErrors:    false,
		},
	}

	testdataDir := findTestdataDir(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New()
			tfPath := filepath.Join(testdataDir, tt.pattern)

			result, err := provider.Parse(context.Background(), tfPath)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(result.Resources) != tt.wantResources {
				t.Errorf("got %d resources, want %d", len(result.Resources), tt.wantResources)
				for _, r := range result.Resources {
					t.Logf("  - %s.%s", r.Type, r.Name)
				}
			}

			if tt.wantErrors && len(result.Errors) == 0 {
				t.Errorf("expected errors, but got none")
			}
			if !tt.wantErrors && len(result.Errors) > 0 {
				t.Errorf("unexpected errors: %v", result.Errors)
			}
		})
	}
}

func TestParseRelativePath(t *testing.T) {
	testdataDir := findTestdataDir(t)
	provider := New()

	// Test relative path module (../shared)
	tfPath := filepath.Join(testdataDir, "relative-path", "envs", "prod")

	result, err := provider.Parse(context.Background(), tfPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should find: aws_instance (prod) + aws_vpc, aws_subnet, aws_internet_gateway (shared)
	if len(result.Resources) != 4 {
		t.Errorf("got %d resources, want 4", len(result.Resources))
		for _, r := range result.Resources {
			t.Logf("  - %s.%s", r.Type, r.Name)
		}
	}
}

func TestDetect(t *testing.T) {
	testdataDir := findTestdataDir(t)
	provider := New()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "directory with tf files",
			path: filepath.Join(testdataDir, "simple"),
			want: true,
		},
		{
			name: "empty directory",
			path: t.TempDir(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.Detect(tt.path)
			if err != nil {
				t.Fatalf("Detect failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileExtensions(t *testing.T) {
	provider := New()
	exts := provider.FileExtensions()

	if len(exts) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(exts))
	}

	hasExt := func(ext string) bool {
		for _, e := range exts {
			if e == ext {
				return true
			}
		}
		return false
	}

	if !hasExt(".tf") {
		t.Error("expected .tf extension")
	}
	if !hasExt(".tf.json") {
		t.Error("expected .tf.json extension")
	}
}

// findTestdataDir locates the testdata directory
func findTestdataDir(t *testing.T) string {
	candidates := []string{
		"../../../testdata",
		"../../testdata",
		"testdata",
	}

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			absPath, _ := filepath.Abs(dir)
			return absPath
		}
	}

	t.Fatal("testdata directory not found")
	return ""
}
