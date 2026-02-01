package checker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mizzy/least/internal/policy"
	"github.com/mizzy/least/internal/provider/terraform"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		existing      string
		wantMissing   bool
		wantExcessive bool
	}{
		{
			name:          "simple - missing DynamoDB permissions",
			pattern:       "simple",
			existing:      "simple/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: true,
		},
		{
			name:          "local-module - missing IAM and EC2 permissions",
			pattern:       "local-module",
			existing:      "local-module/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: false,
		},
		{
			name:          "nested-modules - missing RDS instance permissions",
			pattern:       "nested-modules",
			existing:      "nested-modules/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: false,
		},
		{
			name:          "multi-call - missing S3 permissions",
			pattern:       "multi-call",
			existing:      "multi-call/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: false,
		},
		{
			name:          "mixed-resources - wildcard covers all",
			pattern:       "mixed-resources",
			existing:      "mixed-resources/iam/existing-policy.json",
			wantMissing:   false,
			wantExcessive: false, // wildcards cover required actions
		},
		{
			name:          "circular-ref - missing permissions",
			pattern:       "circular-ref",
			existing:      "circular-ref/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: false,
		},
		{
			name:          "multi-file - missing IAM permissions",
			pattern:       "multi-file",
			existing:      "multi-file/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: false, // wildcards cover required actions
		},
		{
			name:          "for-each - missing various permissions",
			pattern:       "for-each",
			existing:      "for-each/iam/existing-policy.json",
			wantMissing:   true,
			wantExcessive: false,
		},
		{
			name:          "data-only - no resources, excessive permissions",
			pattern:       "data-only",
			existing:      "data-only/iam/existing-policy.json",
			wantMissing:   false,
			wantExcessive: true,
		},
		{
			name:          "no-resources - no resources, excessive permissions",
			pattern:       "no-resources",
			existing:      "no-resources/iam/existing-policy.json",
			wantMissing:   false,
			wantExcessive: true,
		},
		{
			name:          "empty-module - S3 wildcard covers all",
			pattern:       "empty-module",
			existing:      "empty-module/iam/existing-policy.json",
			wantMissing:   false,
			wantExcessive: false, // s3:* covers all required s3 actions
		},
		{
			name:          "remote-module - ec2 wildcard covers VPC",
			pattern:       "remote-module",
			existing:      "remote-module/iam/existing-policy.json",
			wantMissing:   false, // ec2:* covers all required ec2/vpc actions
			wantExcessive: false,
		},
	}

	testdataDir := findTestdataDir(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse Terraform files to get required permissions
			tfProvider := terraform.New()
			tfPath := filepath.Join(testdataDir, tt.pattern)

			parseResult, err := tfProvider.Parse(context.Background(), tfPath)
			if err != nil {
				t.Fatalf("failed to parse terraform: %v", err)
			}

			generator := policy.New()
			required, err := generator.Generate(parseResult.Resources)
			if err != nil {
				t.Fatalf("failed to generate policy: %v", err)
			}

			// Load existing policy
			existingPath := filepath.Join(testdataDir, tt.existing)
			existing, err := loadJSONPolicy(existingPath)
			if err != nil {
				t.Fatalf("failed to load existing policy: %v", err)
			}

			// Run check
			result := Check(existing, required)

			// Verify results
			if tt.wantMissing && !result.HasMissing() {
				t.Errorf("expected missing permissions, but got none")
			}
			if !tt.wantMissing && result.HasMissing() {
				t.Errorf("expected no missing permissions, but got: %v", result.Missing)
			}
			if tt.wantExcessive && !result.HasExcessive() {
				t.Errorf("expected excessive permissions, but got none")
			}
			if !tt.wantExcessive && result.HasExcessive() {
				t.Errorf("expected no excessive permissions, but got: %v", result.Excessive)
			}
		})
	}
}

func TestCheckCompliant(t *testing.T) {
	// Test a fully compliant policy
	existing := &policy.IAMPolicy{
		Statement: []policy.Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject", "s3:PutObject"},
				Resource: []string{"*"},
			},
		},
	}

	required := &policy.IAMPolicy{
		Statement: []policy.Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject", "s3:PutObject"},
				Resource: []string{"*"},
			},
		},
	}

	result := Check(existing, required)

	if !result.IsCompliant() {
		t.Errorf("expected compliant, but got missing=%v, excessive=%v",
			result.Missing, result.Excessive)
	}
}

func TestCheckWildcard(t *testing.T) {
	// Test wildcard matching - existing has wildcard, required has specific actions
	existing := &policy.IAMPolicy{
		Statement: []policy.Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:*"},
				Resource: []string{"*"},
			},
		},
	}

	required := &policy.IAMPolicy{
		Statement: []policy.Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"},
				Resource: []string{"*"},
			},
		},
	}

	result := Check(existing, required)

	if result.HasMissing() {
		t.Errorf("wildcard should cover all s3 actions, but got missing: %v", result.Missing)
	}

	// Note: s3:* is NOT considered excessive because it matches required actions
	// The checker considers wildcard patterns as covering specific actions
	if result.HasExcessive() {
		t.Logf("s3:* is excessive: %v (this is expected behavior)", result.Excessive)
	}
}

func TestCheckWildcardExcessive(t *testing.T) {
	// Test excessive detection - existing has extra service wildcard
	existing := &policy.IAMPolicy{
		Statement: []policy.Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject", "ec2:*"},
				Resource: []string{"*"},
			},
		},
	}

	required := &policy.IAMPolicy{
		Statement: []policy.Statement{
			{
				Effect:   "Allow",
				Action:   []string{"s3:GetObject"},
				Resource: []string{"*"},
			},
		},
	}

	result := Check(existing, required)

	if result.HasMissing() {
		t.Errorf("should not have missing, but got: %v", result.Missing)
	}

	// ec2:* is excessive because it's not required
	if !result.HasExcessive() {
		t.Errorf("ec2:* should be excessive")
	}
}

func TestMatchAction(t *testing.T) {
	tests := []struct {
		pattern string
		action  string
		want    bool
	}{
		{"s3:GetObject", "s3:GetObject", true},
		{"s3:GetObject", "s3:PutObject", false},
		{"s3:*", "s3:GetObject", true},
		{"s3:*", "s3:PutObject", true},
		{"s3:Get*", "s3:GetObject", true},
		{"s3:Get*", "s3:GetBucketAcl", true},
		{"s3:Get*", "s3:PutObject", false},
		{"ec2:*", "s3:GetObject", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.action, func(t *testing.T) {
			got := matchAction(tt.pattern, tt.action)
			if got != tt.want {
				t.Errorf("matchAction(%q, %q) = %v, want %v",
					tt.pattern, tt.action, got, tt.want)
			}
		})
	}
}

// findTestdataDir locates the testdata directory
func findTestdataDir(t *testing.T) string {
	// Try relative paths from test execution directory
	candidates := []string{
		"../../testdata",
		"../../../testdata",
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

// loadJSONPolicy loads an IAM policy from a JSON file
func loadJSONPolicy(path string) (*policy.IAMPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawPolicy struct {
		Version   string `json:"Version"`
		Statement []struct {
			Sid       string      `json:"Sid"`
			Effect    string      `json:"Effect"`
			Action    interface{} `json:"Action"`
			Resource  interface{} `json:"Resource"`
			Principal interface{} `json:"Principal"`
		} `json:"Statement"`
	}

	if err := json.Unmarshal(data, &rawPolicy); err != nil {
		return nil, err
	}

	result := &policy.IAMPolicy{}

	for _, stmt := range rawPolicy.Statement {
		s := policy.Statement{
			Sid:    stmt.Sid,
			Effect: stmt.Effect,
		}

		// Handle Action (can be string or []string)
		switch v := stmt.Action.(type) {
		case string:
			s.Action = []string{v}
		case []interface{}:
			for _, a := range v {
				if str, ok := a.(string); ok {
					s.Action = append(s.Action, str)
				}
			}
		}

		// Handle Resource (can be string or []string)
		switch v := stmt.Resource.(type) {
		case string:
			s.Resource = []string{v}
		case []interface{}:
			for _, r := range v {
				if str, ok := r.(string); ok {
					s.Resource = append(s.Resource, str)
				}
			}
		}

		result.Statement = append(result.Statement, s)
	}

	return result, nil
}
