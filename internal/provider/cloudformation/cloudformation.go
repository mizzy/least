// Package cloudformation implements the Provider interface for AWS CloudFormation.
// This is a stub implementation showing how to add new providers.
package cloudformation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mizzy/least/internal/provider"
)

// Provider implements the provider.Provider interface for CloudFormation
type Provider struct{}

// New creates a new CloudFormation provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "cloudformation"
}

// FileExtensions returns file extensions this provider handles
func (p *Provider) FileExtensions() []string {
	return []string{".yaml", ".yml", ".json", ".template"}
}

// Detect checks if the given path contains CloudFormation templates
func (p *Provider) Detect(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return p.isCloudFormationFile(path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(path, entry.Name())
		if ok, _ := p.isCloudFormationFile(filePath); ok {
			return true, nil
		}
	}

	return false, nil
}

// isCloudFormationFile checks if a file is a CloudFormation template
func (p *Provider) isCloudFormationFile(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" && ext != ".json" && ext != ".template" {
		return false, nil
	}

	// Read file and check for CloudFormation markers
	content, err := os.ReadFile(path)
	if err != nil {
		return false, nil
	}

	contentStr := string(content)
	// Check for common CloudFormation markers
	markers := []string{
		"AWSTemplateFormatVersion",
		"AWS::CloudFormation::",
		"AWS::S3::Bucket",
		"AWS::EC2::Instance",
		"AWS::Lambda::Function",
	}

	for _, marker := range markers {
		if strings.Contains(contentStr, marker) {
			return true, nil
		}
	}

	return false, nil
}

// Parse parses CloudFormation templates and returns resources and policies
func (p *Provider) Parse(ctx context.Context, path string) (*provider.ParseResult, error) {
	// TODO: Implement CloudFormation parsing
	// This would involve:
	// 1. Parsing YAML/JSON templates
	// 2. Extracting resources (AWS::S3::Bucket -> aws_s3_bucket equivalent)
	// 3. Mapping CloudFormation resource types to IAM actions

	return nil, fmt.Errorf("CloudFormation provider not yet implemented")
}
