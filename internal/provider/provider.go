// Package provider defines interfaces for IaC tool providers.
// This abstraction allows the tool to support multiple IaC frameworks
// such as Terraform, CloudFormation, Pulumi, CDK, etc.
package provider

import "context"

// Resource represents a cloud resource defined in IaC code.
// This is the common representation across all providers.
type Resource struct {
	// Provider identifies the IaC tool (e.g., "terraform", "cloudformation")
	Provider string

	// Type is the resource type (e.g., "aws_s3_bucket", "AWS::S3::Bucket")
	Type string

	// Name is the logical name of the resource
	Name string

	// CloudProvider is the cloud platform (e.g., "aws", "gcp", "azure")
	CloudProvider string

	// Attributes contains relevant resource attributes
	Attributes map[string]interface{}

	// Location contains source file information
	Location SourceLocation
}

// SourceLocation identifies where a resource is defined
type SourceLocation struct {
	File   string
	Line   int
	Column int
}

// IAMStatement represents a policy statement from IaC code
type IAMStatement struct {
	Sid       string
	Effect    string
	Actions   []string
	Resources []string
}

// IAMPolicy represents an IAM policy defined in IaC code
type IAMPolicy struct {
	Name       string
	Statements []IAMStatement
	Location   SourceLocation
}

// ParseResult contains the results of parsing IaC files
type ParseResult struct {
	// Resources are the cloud resources defined in the IaC code
	Resources []Resource

	// Policies are IAM policies defined in the IaC code
	Policies []IAMPolicy

	// Errors encountered during parsing (non-fatal)
	Errors []error
}

// Provider is the interface that IaC tool parsers must implement
type Provider interface {
	// Name returns the provider identifier (e.g., "terraform", "cloudformation")
	Name() string

	// Detect checks if the given path contains files for this provider
	Detect(path string) (bool, error)

	// Parse parses IaC files at the given path and returns resources and policies
	Parse(ctx context.Context, path string) (*ParseResult, error)

	// FileExtensions returns file extensions this provider handles
	FileExtensions() []string
}

// Registry manages available providers
type Registry struct {
	providers []Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make([]Provider, 0),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(p Provider) {
	r.providers = append(r.providers, p)
}

// Detect finds providers that can handle the given path
func (r *Registry) Detect(path string) ([]Provider, error) {
	var matched []Provider
	for _, p := range r.providers {
		ok, err := p.Detect(path)
		if err != nil {
			return nil, err
		}
		if ok {
			matched = append(matched, p)
		}
	}
	return matched, nil
}

// Get returns a provider by name
func (r *Registry) Get(name string) Provider {
	for _, p := range r.providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// All returns all registered providers
func (r *Registry) All() []Provider {
	return r.providers
}
