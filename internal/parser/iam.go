package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// IAMStatement represents a statement in an IAM policy
type IAMStatement struct {
	Sid       string
	Effect    string
	Actions   []string
	Resources []string
}

// IAMPolicyDoc represents an IAM policy extracted from Terraform
type IAMPolicyDoc struct {
	Name       string         // resource name
	Statements []IAMStatement // policy statements
	File       string         // source file
	Line       int            // line number
}

// ParseIAMPolicies extracts IAM policies from Terraform files in a directory
func (p *Parser) ParseIAMPolicies(dir string) ([]IAMPolicyDoc, error) {
	var policies []IAMPolicyDoc

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
		filePolicies, err := p.parseIAMPoliciesFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filePath, err)
		}
		policies = append(policies, filePolicies...)
	}

	return policies, nil
}

func (p *Parser) parseIAMPoliciesFromFile(filename string) ([]IAMPolicyDoc, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	file, diags := p.parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	return p.extractIAMPolicies(file, filename)
}

func (p *Parser) extractIAMPolicies(file *hcl.File, filename string) ([]IAMPolicyDoc, error) {
	var policies []IAMPolicyDoc

	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("extracting content: %s", diags.Error())
	}

	for _, block := range content.Blocks {
		if len(block.Labels) < 2 {
			continue
		}

		resourceType := block.Labels[0]
		resourceName := block.Labels[1]

		switch {
		case block.Type == "data" && resourceType == "aws_iam_policy_document":
			// Parse aws_iam_policy_document data source
			policy, err := p.parseIAMPolicyDocument(block, filename)
			if err != nil {
				continue // Skip malformed policies
			}
			policy.Name = resourceName
			policies = append(policies, *policy)

		case block.Type == "resource" && isIAMPolicyResource(resourceType):
			// Parse inline policy from aws_iam_policy, aws_iam_role_policy, etc.
			policy, err := p.parseInlinePolicy(block, filename)
			if err != nil {
				continue
			}
			policy.Name = resourceName
			policies = append(policies, *policy)
		}
	}

	return policies, nil
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

// parseIAMPolicyDocument parses aws_iam_policy_document data source
func (p *Parser) parseIAMPolicyDocument(block *hcl.Block, filename string) (*IAMPolicyDoc, error) {
	policy := &IAMPolicyDoc{
		File: filename,
		Line: block.DefRange.Start.Line,
	}

	// Get statement blocks
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

func (p *Parser) parseStatement(block *hcl.Block) (*IAMStatement, error) {
	stmt := &IAMStatement{
		Effect: "Allow", // default
	}

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		// Try partial content for nested blocks
		return p.parseStatementWithBlocks(block)
	}

	for name, attr := range attrs {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
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

func (p *Parser) parseStatementWithBlocks(block *hcl.Block) (*IAMStatement, error) {
	stmt := &IAMStatement{
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

// parseInlinePolicy parses policy attribute from aws_iam_policy resources
func (p *Parser) parseInlinePolicy(block *hcl.Block, filename string) (*IAMPolicyDoc, error) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		// Try with partial content
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

	// Try to evaluate as a static string (jsonencode or heredoc)
	val, valDiags := policyAttr.Expr.Value(nil)
	if !valDiags.HasErrors() && val.Type() == cty.String {
		jsonStr := val.AsString()
		return parseJSONPolicy(jsonStr, filename, block.DefRange.Start.Line)
	}

	// Try to extract from jsonencode function call
	if funcExpr, ok := policyAttr.Expr.(*hclsyntax.FunctionCallExpr); ok {
		if funcExpr.Name == "jsonencode" && len(funcExpr.Args) > 0 {
			return p.parseJsonencodeArg(funcExpr.Args[0], filename, block.DefRange.Start.Line)
		}
	}

	// Check for reference to aws_iam_policy_document (we'll handle this separately)
	return nil, fmt.Errorf("policy uses dynamic reference")
}

func (p *Parser) parseJsonencodeArg(expr hcl.Expression, filename string, line int) (*IAMPolicyDoc, error) {
	// For complex HCL expressions in jsonencode, we need to parse the structure
	policy := &IAMPolicyDoc{
		File: filename,
		Line: line,
	}

	// This is a simplified parser - real implementation would need full HCL evaluation
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

func parseStatementFromExpr(expr hcl.Expression) *IAMStatement {
	objExpr, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil
	}

	stmt := &IAMStatement{
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

func parseJSONPolicy(jsonStr string, filename string, line int) (*IAMPolicyDoc, error) {
	var rawPolicy struct {
		Statement []struct {
			Sid      string      `json:"Sid"`
			Effect   string      `json:"Effect"`
			Action   interface{} `json:"Action"`
			Resource interface{} `json:"Resource"`
		} `json:"Statement"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawPolicy); err != nil {
		return nil, fmt.Errorf("parsing JSON policy: %w", err)
	}

	policy := &IAMPolicyDoc{
		File: filename,
		Line: line,
	}

	for _, stmt := range rawPolicy.Statement {
		iamStmt := IAMStatement{
			Sid:    stmt.Sid,
			Effect: stmt.Effect,
		}

		// Handle Action as string or []string
		switch v := stmt.Action.(type) {
		case string:
			iamStmt.Actions = []string{v}
		case []interface{}:
			for _, a := range v {
				if s, ok := a.(string); ok {
					iamStmt.Actions = append(iamStmt.Actions, s)
				}
			}
		}

		// Handle Resource as string or []string
		switch v := stmt.Resource.(type) {
		case string:
			iamStmt.Resources = []string{v}
		case []interface{}:
			for _, r := range v {
				if s, ok := r.(string); ok {
					iamStmt.Resources = append(iamStmt.Resources, s)
				}
			}
		}

		policy.Statements = append(policy.Statements, iamStmt)
	}

	return policy, nil
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

// GetAllActions returns all actions from the policy
func (p *IAMPolicyDoc) GetAllActions() []string {
	seen := make(map[string]bool)
	var actions []string

	for _, stmt := range p.Statements {
		if strings.EqualFold(stmt.Effect, "Allow") {
			for _, action := range stmt.Actions {
				if !seen[action] {
					seen[action] = true
					actions = append(actions, action)
				}
			}
		}
	}

	return actions
}
