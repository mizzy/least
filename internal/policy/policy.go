package policy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/mizzy/least/internal/mapping"
	"github.com/mizzy/least/internal/provider"
)

// IAMPolicy represents an AWS IAM policy document
type IAMPolicy struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a single IAM policy statement
type Statement struct {
	Sid      string     `json:"Sid,omitempty"`
	Effect   string     `json:"Effect"`
	Action   StringList `json:"Action"`
	Resource StringList `json:"Resource"`
}

// StringList handles both single string and array of strings in JSON
type StringList []string

// UnmarshalJSON handles both "action" and ["action1", "action2"] formats
func (s *StringList) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a single string first
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}

	// Try to unmarshal as an array
	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return err
	}
	*s = multiple
	return nil
}

// GeneratorOptions configures how policies are generated
type GeneratorOptions struct {
	// OutputFormat is "terraform" or "json"
	OutputFormat string
	// AccountRef is the reference for AWS account ID
	// e.g., "${data.aws_caller_identity.current.account_id}" or "${var.account_id}"
	AccountRef string
	// RegionRef is the reference for AWS region
	// e.g., "${data.aws_region.current.name}" or "${var.aws_region}"
	RegionRef string
	// NeedCallerIdentity indicates if data source needs to be added to output
	NeedCallerIdentity bool
	// NeedRegion indicates if data source needs to be added to output
	NeedRegion bool
}

// Generator generates IAM policies from parsed resources
type Generator struct {
	options GeneratorOptions
}

// New creates a new Generator with default options
func New() *Generator {
	return &Generator{
		options: GeneratorOptions{
			OutputFormat: "terraform",
		},
	}
}

// NewWithOptions creates a new Generator with specified options
func NewWithOptions(opts GeneratorOptions) *Generator {
	return &Generator{options: opts}
}

// Generate creates a minimal IAM policy for the given resources
func (g *Generator) Generate(resources []provider.Resource) (*IAMPolicy, error) {
	statements := make([]Statement, 0)

	for _, res := range resources {
		actions := mapping.GetActionsForResource(res.Type)
		if len(actions) == 0 {
			continue
		}

		// Sort actions for consistent output
		sort.Strings(actions)

		// Build ARNs for this resource
		arns := g.buildARNsForResource(res)

		// Generate Sid from resource type and name
		sid := g.generateSid(res.Type, res.Name)

		statements = append(statements, Statement{
			Sid:      sid,
			Effect:   "Allow",
			Action:   actions,
			Resource: arns,
		})
	}

	// If no statements were generated, return empty policy
	if len(statements) == 0 {
		return &IAMPolicy{
			Version:   "2012-10-17",
			Statement: []Statement{},
		}, nil
	}

	policy := &IAMPolicy{
		Version:   "2012-10-17",
		Statement: statements,
	}

	return policy, nil
}

// generateSid creates a statement ID from resource type and name
func (g *Generator) generateSid(resourceType, resourceName string) string {
	// Convert aws_s3_bucket to AwsS3Bucket
	parts := strings.Split(resourceType, "_")
	for i, part := range parts {
		parts[i] = capitalize(part)
	}

	// Add resource name (capitalize first letter)
	nameParts := strings.Split(resourceName, "_")
	for i, part := range nameParts {
		nameParts[i] = capitalize(part)
	}

	return strings.Join(parts, "") + strings.Join(nameParts, "")
}

// capitalize returns the string with the first letter capitalized
func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// buildARNsForResource constructs the ARN(s) for a resource
func (g *Generator) buildARNsForResource(res provider.Resource) []string {
	pattern, ok := mapping.GetARNPattern(res.Type)
	if !ok {
		// No ARN pattern defined, fall back to *
		return []string{"*"}
	}

	arns := []string{}

	// Build main ARN
	mainARN := g.buildARN(pattern.Pattern, pattern.ResourceAttribute, res)
	arns = append(arns, mainARN)

	// Add child patterns (e.g., S3 objects)
	for _, childPattern := range pattern.ChildPatterns {
		childARN := g.buildARN(childPattern, pattern.ResourceAttribute, res)
		arns = append(arns, childARN)
	}

	return arns
}

// buildARN constructs a single ARN from a pattern
func (g *Generator) buildARN(pattern, attrName string, res provider.Resource) string {
	arn := pattern

	// Replace {account} placeholder
	if strings.Contains(arn, "{account}") {
		if g.options.OutputFormat == "terraform" && g.options.AccountRef != "" {
			arn = strings.ReplaceAll(arn, "{account}", g.options.AccountRef)
		} else {
			// JSON format or no AccountRef: use *
			arn = strings.ReplaceAll(arn, "{account}", "*")
		}
	}

	// Replace {region} placeholder
	if strings.Contains(arn, "{region}") {
		if g.options.OutputFormat == "terraform" && g.options.RegionRef != "" {
			arn = strings.ReplaceAll(arn, "{region}", g.options.RegionRef)
		} else {
			// JSON format or no RegionRef: use *
			arn = strings.ReplaceAll(arn, "{region}", "*")
		}
	}

	// Replace resource-specific attribute placeholder
	if attrName != "" && strings.Contains(arn, "{"+attrName+"}") {
		attrVal, ok := res.Attributes[attrName]
		if ok {
			// Type assertion for AttributeValue (defined in terraform package)
			// We need to check both interface types
			switch v := attrVal.(type) {
			case interface{ GetLiteral() string }:
				lit := v.GetLiteral()
				if lit != "" {
					arn = strings.ReplaceAll(arn, "{"+attrName+"}", lit)
				}
			default:
				// Try to access as a map or struct with Literal/Reference fields
				if m, ok := attrVal.(map[string]interface{}); ok {
					if lit, ok := m["Literal"].(string); ok && lit != "" {
						arn = strings.ReplaceAll(arn, "{"+attrName+"}", lit)
					} else if ref, ok := m["Reference"].(string); ok && ref != "" {
						arn = strings.ReplaceAll(arn, "{"+attrName+"}", "${"+ref+"}")
					}
				} else {
					// Check if it's the AttributeValue struct directly
					// Using reflection-free approach
					arn = g.replaceAttributeValue(arn, attrName, attrVal)
				}
			}
		}

		// If still contains placeholder, fall back to *
		if strings.Contains(arn, "{"+attrName+"}") {
			arn = strings.ReplaceAll(arn, "{"+attrName+"}", "*")
		}
	}

	return arn
}

// replaceAttributeValue handles the AttributeValue struct from terraform package
func (g *Generator) replaceAttributeValue(arn, attrName string, attrVal interface{}) string {
	// Use type assertion for the struct fields via fmt
	// This is a workaround since we can't import terraform package here (circular dep)
	valStr := fmt.Sprintf("%+v", attrVal)

	// Parse the string representation: {Literal:value Reference:}
	if strings.Contains(valStr, "Literal:") {
		// Extract Literal value
		start := strings.Index(valStr, "Literal:") + 8
		end := strings.Index(valStr[start:], " ")
		if end == -1 {
			end = strings.Index(valStr[start:], "}")
		}
		if end > 0 {
			literal := valStr[start : start+end]
			if literal != "" {
				return strings.ReplaceAll(arn, "{"+attrName+"}", literal)
			}
		}
	}

	if strings.Contains(valStr, "Reference:") {
		// Extract Reference value
		start := strings.Index(valStr, "Reference:") + 10
		end := strings.Index(valStr[start:], "}")
		if end > 0 {
			ref := valStr[start : start+end]
			if ref != "" {
				return strings.ReplaceAll(arn, "{"+attrName+"}", "${"+ref+"}")
			}
		}
	}

	return arn
}

// ToJSON converts the policy to JSON string
func (p *IAMPolicy) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TerraformOutputOptions configures the Terraform output
type TerraformOutputOptions struct {
	NeedCallerIdentity bool
	NeedRegion         bool
}

// ToTerraform converts the policy to Terraform HCL (aws_iam_policy_document)
func (p *IAMPolicy) ToTerraform() string {
	return p.ToTerraformWithOptions(TerraformOutputOptions{})
}

// ToTerraformWithOptions converts the policy to Terraform HCL with data sources as needed
func (p *IAMPolicy) ToTerraformWithOptions(opts TerraformOutputOptions) string {
	var b strings.Builder

	// Add data sources if needed
	if opts.NeedCallerIdentity {
		b.WriteString(`data "aws_caller_identity" "current" {}`)
		b.WriteString("\n\n")
	}
	if opts.NeedRegion {
		b.WriteString(`data "aws_region" "current" {}`)
		b.WriteString("\n\n")
	}

	b.WriteString(`data "aws_iam_policy_document" "least_privilege" {`)
	b.WriteString("\n")

	for _, stmt := range p.Statement {
		b.WriteString("  statement {\n")

		if stmt.Sid != "" {
			b.WriteString(`    sid    = "`)
			b.WriteString(stmt.Sid)
			b.WriteString("\"\n")
		}

		b.WriteString(`    effect = "`)
		b.WriteString(stmt.Effect)
		b.WriteString("\"\n")

		b.WriteString("\n    actions = [\n")
		for _, action := range stmt.Action {
			b.WriteString(`      "`)
			b.WriteString(action)
			b.WriteString("\",\n")
		}
		b.WriteString("    ]\n")

		b.WriteString("\n    resources = [\n")
		for _, resource := range stmt.Resource {
			b.WriteString(`      "`)
			b.WriteString(resource)
			b.WriteString("\",\n")
		}
		b.WriteString("    ]\n")

		b.WriteString("  }\n")
	}

	b.WriteString("}\n")

	return b.String()
}

// ParsePolicy parses a JSON IAM policy
func ParsePolicy(data []byte) (*IAMPolicy, error) {
	var policy IAMPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// GetAllActions extracts all actions from a policy
func (p *IAMPolicy) GetAllActions() []string {
	actionSet := make(map[string]bool)
	for _, stmt := range p.Statement {
		if stmt.Effect == "Allow" {
			for _, action := range stmt.Action {
				actionSet[action] = true
			}
		}
	}

	actions := make([]string, 0, len(actionSet))
	for action := range actionSet {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

// FromProviderPolicies creates an IAMPolicy from provider-parsed IAM policies
func FromProviderPolicies(policies []provider.IAMPolicy) *IAMPolicy {
	actionSet := make(map[string]bool)

	for _, pol := range policies {
		for _, stmt := range pol.Statements {
			if strings.EqualFold(stmt.Effect, "Allow") {
				for _, action := range stmt.Actions {
					actionSet[action] = true
				}
			}
		}
	}

	actions := make([]string, 0, len(actionSet))
	for action := range actionSet {
		actions = append(actions, action)
	}
	sort.Strings(actions)

	return &IAMPolicy{
		Version: "2012-10-17",
		Statement: []Statement{
			{
				Sid:      "CombinedPolicy",
				Effect:   "Allow",
				Action:   actions,
				Resource: []string{"*"},
			},
		},
	}
}
