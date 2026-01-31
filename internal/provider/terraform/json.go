package terraform

import (
	"encoding/json"
	"fmt"

	"github.com/mizzy/least/internal/provider"
)

// parseJSONPolicy parses an inline JSON IAM policy
func parseJSONPolicy(jsonStr string, filename string, line int) (*provider.IAMPolicy, error) {
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

	policy := &provider.IAMPolicy{
		Location: provider.SourceLocation{
			File: filename,
			Line: line,
		},
	}

	for _, stmt := range rawPolicy.Statement {
		iamStmt := provider.IAMStatement{
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
