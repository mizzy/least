// Package terraform implements the Provider interface for Terraform/OpenTofu.
package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/mizzy/least/internal/provider"
	"github.com/zclconf/go-cty/cty"
)

// Provider implements the provider.Provider interface for Terraform
type Provider struct {
	parser *hclparse.Parser
}

// New creates a new Terraform provider
func New() *Provider {
	return &Provider{
		parser: hclparse.NewParser(),
	}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "terraform"
}

// FileExtensions returns file extensions this provider handles
func (p *Provider) FileExtensions() []string {
	return []string{".tf", ".tf.json"}
}

// Detect checks if the given path contains Terraform files
func (p *Provider) Detect(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		ext := filepath.Ext(path)
		return ext == ".tf" || strings.HasSuffix(path, ".tf.json"), nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tf") || strings.HasSuffix(name, ".tf.json") {
			return true, nil
		}
	}

	return false, nil
}

// Parse parses Terraform files and returns resources and policies
func (p *Provider) Parse(ctx context.Context, path string) (*provider.ParseResult, error) {
	result := &provider.ParseResult{
		Resources: make([]provider.Resource, 0),
		Policies:  make([]provider.IAMPolicy, 0),
	}

	// Track visited paths to prevent infinite loops
	visited := make(map[string]bool)

	if err := p.parseWithModules(ctx, path, result, visited); err != nil {
		return nil, err
	}

	return result, nil
}

// parseWithModules parses Terraform files and recursively processes module calls
func (p *Provider) parseWithModules(ctx context.Context, path string, result *provider.ParseResult, visited map[string]bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("accessing path: %w", err)
	}

	var dir string
	var files []string

	if info.IsDir() {
		dir = path
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("reading directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".tf") {
				files = append(files, filepath.Join(path, entry.Name()))
			}
		}
	} else {
		dir = filepath.Dir(path)
		files = []string{path}
	}

	// Normalize and check if already visited
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}
	if visited[absDir] {
		return nil // Already processed this directory
	}
	visited[absDir] = true

	// Parse files in current directory
	for _, filePath := range files {
		if err := p.parseFile(ctx, filePath, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("parsing %s: %w", filePath, err))
		}
	}

	// Use terraform-config-inspect to find module calls
	if info.IsDir() {
		module, diags := tfconfig.LoadModule(path)
		if diags.HasErrors() {
			result.Errors = append(result.Errors, fmt.Errorf("loading module info: %s", diags.Error()))
			return nil // Continue without module parsing
		}

		for name, modCall := range module.ModuleCalls {
			modPath, err := p.resolveModuleSource(path, modCall.Source)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("resolving module %q: %w", name, err))
				continue
			}

			if modPath == "" {
				// Remote module not downloaded yet
				result.Errors = append(result.Errors, fmt.Errorf("module %q: remote module not found in .terraform/modules (run 'terraform init' first)", name))
				continue
			}

			// Recursively parse the module
			if err := p.parseWithModules(ctx, modPath, result, visited); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("parsing module %q: %w", name, err))
			}
		}
	}

	return nil
}

// resolveModuleSource resolves a module source to a local path
func (p *Provider) resolveModuleSource(basePath, source string) (string, error) {
	// Local path (starts with ./ or ../ or is absolute)
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		modPath := filepath.Join(basePath, source)
		if _, err := os.Stat(modPath); err == nil {
			return modPath, nil
		}
		return "", fmt.Errorf("local module path not found: %s", modPath)
	}

	// Absolute path
	if filepath.IsAbs(source) {
		if _, err := os.Stat(source); err == nil {
			return source, nil
		}
		return "", fmt.Errorf("absolute module path not found: %s", source)
	}

	// Remote module - check .terraform/modules directory
	// After 'terraform init', modules are downloaded to .terraform/modules/
	modulesDir := filepath.Join(basePath, ".terraform", "modules")
	modulesJSON := filepath.Join(modulesDir, "modules.json")

	if _, err := os.Stat(modulesJSON); os.IsNotExist(err) {
		// .terraform/modules doesn't exist
		return "", nil
	}

	// Parse modules.json to find the correct path
	modPath, err := p.findModuleInManifest(modulesJSON, source)
	if err != nil {
		return "", err
	}

	if modPath != "" {
		// The path in modules.json is relative to .terraform/modules
		fullPath := filepath.Join(basePath, modPath)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}

	return "", nil
}

// findModuleInManifest searches for a module in the Terraform modules manifest
func (p *Provider) findModuleInManifest(manifestPath, source string) (string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}

	// Parse the modules.json file
	// Format: {"Modules":[{"Key":"...","Source":"...","Dir":"..."},...]}
	type moduleEntry struct {
		Key    string `json:"Key"`
		Source string `json:"Source"`
		Dir    string `json:"Dir"`
	}
	type modulesManifest struct {
		Modules []moduleEntry `json:"Modules"`
	}

	var manifest modulesManifest
	if err := parseJSON(data, &manifest); err != nil {
		return "", err
	}

	for _, mod := range manifest.Modules {
		if mod.Source == source {
			return mod.Dir, nil
		}
	}

	return "", nil
}

func (p *Provider) parseFile(ctx context.Context, filename string, result *provider.ParseResult) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	file, diags := p.parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
		},
	})
	if diags.HasErrors() {
		return fmt.Errorf("extracting content: %s", diags.Error())
	}

	for _, block := range content.Blocks {
		if len(block.Labels) < 2 {
			continue
		}

		resourceType := block.Labels[0]
		resourceName := block.Labels[1]

		switch block.Type {
		case "resource":
			// Add to resources list
			res := provider.Resource{
				Provider:      "terraform",
				Type:          resourceType,
				Name:          resourceName,
				CloudProvider: detectCloudProvider(resourceType),
				Attributes:    make(map[string]interface{}),
				Location: provider.SourceLocation{
					File: filename,
					Line: block.DefRange.Start.Line,
				},
			}
			result.Resources = append(result.Resources, res)

			// Check if this is an IAM policy resource
			if isIAMPolicyResource(resourceType) {
				policy, err := p.parseInlinePolicy(block, filename)
				if err == nil && policy != nil {
					result.Policies = append(result.Policies, *policy)
				}
			}

		case "data":
			// Parse aws_iam_policy_document data sources
			if resourceType == "aws_iam_policy_document" {
				policy, err := p.parseIAMPolicyDocument(block, filename)
				if err == nil && policy != nil {
					policy.Name = resourceName
					result.Policies = append(result.Policies, *policy)
				}
			}
		}
	}

	return nil
}

func detectCloudProvider(resourceType string) string {
	prefixes := map[string]string{
		"aws_":        "aws",
		"azurerm_":    "azure",
		"google_":     "gcp",
		"oci_":        "oci",
		"digitalocean_": "digitalocean",
		"linode_":     "linode",
		"alicloud_":   "alicloud",
	}

	for prefix, cloud := range prefixes {
		if strings.HasPrefix(resourceType, prefix) {
			return cloud
		}
	}
	return "unknown"
}

func isIAMPolicyResource(resourceType string) bool {
	policyResources := []string{
		"aws_iam_policy",
		"aws_iam_role_policy",
		"aws_iam_user_policy",
		"aws_iam_group_policy",
	}
	for _, pr := range policyResources {
		if resourceType == pr {
			return true
		}
	}
	return false
}

func (p *Provider) parseIAMPolicyDocument(block *hcl.Block, filename string) (*provider.IAMPolicy, error) {
	policy := &provider.IAMPolicy{
		Location: provider.SourceLocation{
			File: filename,
			Line: block.DefRange.Start.Line,
		},
	}

	content, _, diags := block.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "statement"},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("extracting statements: %s", diags.Error())
	}

	for _, stmtBlock := range content.Blocks {
		if stmtBlock.Type != "statement" {
			continue
		}

		stmt, err := p.parseStatement(stmtBlock)
		if err != nil {
			continue
		}
		policy.Statements = append(policy.Statements, *stmt)
	}

	return policy, nil
}

func (p *Provider) parseStatement(block *hcl.Block) (*provider.IAMStatement, error) {
	stmt := &provider.IAMStatement{
		Effect: "Allow",
	}

	content, _, diags := block.Body.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "sid"},
			{Name: "effect"},
			{Name: "actions"},
			{Name: "resources"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "principals"},
			{Type: "condition"},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing statement: %s", diags.Error())
	}

	for name, attr := range content.Attributes {
		val, valDiags := attr.Expr.Value(nil)
		if valDiags.HasErrors() {
			continue
		}

		switch name {
		case "sid":
			if val.Type() == cty.String {
				stmt.Sid = val.AsString()
			}
		case "effect":
			if val.Type() == cty.String {
				stmt.Effect = val.AsString()
			}
		case "actions":
			stmt.Actions = ctyToStringSlice(val)
		case "resources":
			stmt.Resources = ctyToStringSlice(val)
		}
	}

	return stmt, nil
}

func (p *Provider) parseInlinePolicy(block *hcl.Block, filename string) (*provider.IAMPolicy, error) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		content, _, pDiags := block.Body.PartialContent(&hcl.BodySchema{
			Attributes: []hcl.AttributeSchema{
				{Name: "policy"},
			},
		})
		if pDiags.HasErrors() {
			return nil, fmt.Errorf("extracting policy attribute: %s", pDiags.Error())
		}
		attrs = content.Attributes
	}

	policyAttr, ok := attrs["policy"]
	if !ok {
		return nil, fmt.Errorf("no policy attribute found")
	}

	// Try to evaluate as a static string
	val, valDiags := policyAttr.Expr.Value(nil)
	if !valDiags.HasErrors() && val.Type() == cty.String {
		return parseJSONPolicy(val.AsString(), filename, block.DefRange.Start.Line)
	}

	// Try to extract from jsonencode function call
	if funcExpr, ok := policyAttr.Expr.(*hclsyntax.FunctionCallExpr); ok {
		if funcExpr.Name == "jsonencode" && len(funcExpr.Args) > 0 {
			return p.parseJsonencodeArg(funcExpr.Args[0], filename, block.DefRange.Start.Line)
		}
	}

	return nil, fmt.Errorf("policy uses dynamic reference")
}

func (p *Provider) parseJsonencodeArg(expr hcl.Expression, filename string, line int) (*provider.IAMPolicy, error) {
	policy := &provider.IAMPolicy{
		Location: provider.SourceLocation{
			File: filename,
			Line: line,
		},
	}

	if objExpr, ok := expr.(*hclsyntax.ObjectConsExpr); ok {
		for _, item := range objExpr.Items {
			keyVal, _ := item.KeyExpr.Value(nil)
			if keyVal.Type() != cty.String {
				continue
			}
			key := keyVal.AsString()

			if key == "Statement" {
				if tupleExpr, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr); ok {
					for _, stmtExpr := range tupleExpr.Exprs {
						stmt := parseStatementFromExpr(stmtExpr)
						if stmt != nil {
							policy.Statements = append(policy.Statements, *stmt)
						}
					}
				}
			}
		}
	}

	return policy, nil
}

func parseStatementFromExpr(expr hcl.Expression) *provider.IAMStatement {
	objExpr, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil
	}

	stmt := &provider.IAMStatement{
		Effect: "Allow",
	}

	for _, item := range objExpr.Items {
		keyVal, _ := item.KeyExpr.Value(nil)
		if keyVal.Type() != cty.String {
			continue
		}
		key := keyVal.AsString()

		val, _ := item.ValueExpr.Value(nil)

		switch key {
		case "Sid":
			if val.Type() == cty.String {
				stmt.Sid = val.AsString()
			}
		case "Effect":
			if val.Type() == cty.String {
				stmt.Effect = val.AsString()
			}
		case "Action":
			stmt.Actions = ctyToStringSlice(val)
		case "Resource":
			stmt.Resources = ctyToStringSlice(val)
		}
	}

	return stmt
}

func ctyToStringSlice(val cty.Value) []string {
	var result []string

	if val.Type() == cty.String {
		return []string{val.AsString()}
	}

	if val.Type().IsTupleType() || val.Type().IsListType() || val.Type().IsSetType() {
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			if v.Type() == cty.String {
				result = append(result, v.AsString())
			}
		}
	}

	return result
}

// parseJSON is a helper to unmarshal JSON data
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
