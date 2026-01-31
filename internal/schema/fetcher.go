package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Fetcher retrieves CloudFormation resource schemas from AWS
type Fetcher struct {
	store *Store
}

// NewFetcher creates a new schema fetcher
func NewFetcher(store *Store) *Fetcher {
	return &Fetcher{store: store}
}

// FetchSchema retrieves a schema from AWS CloudFormation Registry
// Requires AWS CLI to be installed and configured
func (f *Fetcher) FetchSchema(ctx context.Context, cfnType string) (*ResourceSchema, error) {
	// Use AWS CLI to get the schema
	cmd := exec.CommandContext(ctx, "aws", "cloudformation", "describe-type",
		"--type", "RESOURCE",
		"--type-name", cfnType,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("aws cli error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("executing aws cli: %w", err)
	}

	// Parse the response
	var response struct {
		Schema string `json:"Schema"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("parsing aws response: %w", err)
	}

	// Parse the schema JSON
	var schema ResourceSchema
	if err := json.Unmarshal([]byte(response.Schema), &schema); err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	// Cache the schema
	if err := f.store.LoadSchema([]byte(response.Schema)); err != nil {
		// Log but don't fail
	}

	return &schema, nil
}

// FetchForTerraformType fetches schema for a Terraform resource type
func (f *Fetcher) FetchForTerraformType(ctx context.Context, tfType string) (*ResourceSchema, error) {
	cfnType := TerraformToCfnType(tfType)
	if cfnType == "" {
		return nil, fmt.Errorf("no CloudFormation mapping for %s", tfType)
	}
	return f.FetchSchema(ctx, cfnType)
}

// FetchMultiple fetches schemas for multiple types in parallel
func (f *Fetcher) FetchMultiple(ctx context.Context, cfnTypes []string) map[string]*ResourceSchema {
	results := make(map[string]*ResourceSchema)

	// Simple sequential fetching for now
	// Could be parallelized with goroutines
	for _, cfnType := range cfnTypes {
		schema, err := f.FetchSchema(ctx, cfnType)
		if err != nil {
			continue
		}
		results[cfnType] = schema
	}

	return results
}

// IsAWSCLIAvailable checks if AWS CLI is installed and accessible
func IsAWSCLIAvailable() bool {
	cmd := exec.Command("aws", "--version")
	return cmd.Run() == nil
}

// GetAWSRegion returns the current AWS region from environment/config
func GetAWSRegion() string {
	cmd := exec.Command("aws", "configure", "get", "region")
	output, err := cmd.Output()
	if err != nil {
		return "us-east-1" // default
	}
	return strings.TrimSpace(string(output))
}
