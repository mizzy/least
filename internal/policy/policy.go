package policy

import (
	"encoding/json"
	"sort"

	"github.com/mizzy/least/internal/mapping"
	"github.com/mizzy/least/internal/parser"
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

// Generator generates IAM policies from parsed Terraform resources
type Generator struct{}

// New creates a new Generator
func New() *Generator {
	return &Generator{}
}

// Generate creates a minimal IAM policy for the given resources
func (g *Generator) Generate(resources []parser.Resource) (*IAMPolicy, error) {
	actionSet := make(map[string]bool)

	for _, res := range resources {
		actions := mapping.GetActionsForResource(res.Type)
		for _, action := range actions {
			actionSet[action] = true
		}
	}

	// Convert to sorted slice
	actions := make([]string, 0, len(actionSet))
	for action := range actionSet {
		actions = append(actions, action)
	}
	sort.Strings(actions)

	policy := &IAMPolicy{
		Version: "2012-10-17",
		Statement: []Statement{
			{
				Sid:      "TerraformLeastPrivilege",
				Effect:   "Allow",
				Action:   actions,
				Resource: []string{"*"},
			},
		},
	}

	return policy, nil
}

// ToJSON converts the policy to JSON string
func (p *IAMPolicy) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
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

// FromParsedPolicies creates an IAMPolicy from parsed Terraform IAM policies
func FromParsedPolicies(docs []parser.IAMPolicyDoc) *IAMPolicy {
	actionSet := make(map[string]bool)

	for _, doc := range docs {
		for _, action := range doc.GetAllActions() {
			actionSet[action] = true
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
