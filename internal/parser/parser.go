package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// Resource represents a Terraform resource extracted from HCL
type Resource struct {
	Type       string            // e.g., "aws_s3_bucket"
	Name       string            // e.g., "my_bucket"
	Attributes map[string]string // relevant attributes for IAM
	File       string            // source file
	Line       int               // line number
}

// Parser parses Terraform HCL files
type Parser struct {
	parser *hclparse.Parser
}

// New creates a new Parser
func New() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseDirectory parses all .tf files in a directory
func (p *Parser) ParseDirectory(dir string) ([]Resource, error) {
	var resources []Resource

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".tf" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		fileResources, err := p.ParseFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filePath, err)
		}
		resources = append(resources, fileResources...)
	}

	return resources, nil
}

// ParseFile parses a single .tf file and extracts resources
func (p *Parser) ParseFile(filename string) ([]Resource, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	file, diags := p.parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	return p.extractResources(file, filename)
}

// extractResources extracts resource blocks from parsed HCL
func (p *Parser) extractResources(file *hcl.File, filename string) ([]Resource, error) {
	var resources []Resource

	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("extracting content: %s", diags.Error())
	}

	for _, block := range content.Blocks {
		if block.Type != "resource" || len(block.Labels) < 2 {
			continue
		}

		res := Resource{
			Type:       block.Labels[0],
			Name:       block.Labels[1],
			Attributes: make(map[string]string),
			File:       filename,
			Line:       block.DefRange.Start.Line,
		}

		resources = append(resources, res)
	}

	return resources, nil
}
