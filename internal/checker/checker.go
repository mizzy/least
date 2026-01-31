package checker

import (
	"sort"
	"strings"

	"github.com/mizzy/least/internal/policy"
)

// Result represents the result of a policy check
type Result struct {
	// Actions in required but not in existing (missing permissions)
	Missing []string
	// Actions in existing but not in required (excessive permissions)
	Excessive []string
	// Actions that match
	Matched []string
}

// IsCompliant returns true if there are no missing or excessive permissions
func (r *Result) IsCompliant() bool {
	return len(r.Missing) == 0 && len(r.Excessive) == 0
}

// HasMissing returns true if there are missing permissions
func (r *Result) HasMissing() bool {
	return len(r.Missing) > 0
}

// HasExcessive returns true if there are excessive permissions
func (r *Result) HasExcessive() bool {
	return len(r.Excessive) > 0
}

// Check compares an existing policy against a required policy
func Check(existing, required *policy.IAMPolicy) *Result {
	existingActions := existing.GetAllActions()
	requiredActions := required.GetAllActions()

	existingSet := make(map[string]bool)
	for _, a := range existingActions {
		existingSet[a] = true
	}

	requiredSet := make(map[string]bool)
	for _, a := range requiredActions {
		requiredSet[a] = true
	}

	result := &Result{}

	// Find missing actions (required but not existing)
	for _, action := range requiredActions {
		if matchesAny(action, existingActions) {
			result.Matched = append(result.Matched, action)
		} else {
			result.Missing = append(result.Missing, action)
		}
	}

	// Find excessive actions (existing but not required)
	for _, action := range existingActions {
		if !matchesAny(action, requiredActions) && !isMatchedByRequired(action, requiredActions) {
			result.Excessive = append(result.Excessive, action)
		}
	}

	sort.Strings(result.Missing)
	sort.Strings(result.Excessive)
	sort.Strings(result.Matched)

	return result
}

// matchesAny checks if an action matches any action in the list (including wildcards)
func matchesAny(action string, actions []string) bool {
	for _, a := range actions {
		if matchAction(a, action) {
			return true
		}
	}
	return false
}

// isMatchedByRequired checks if an existing action is covered by any required action
func isMatchedByRequired(existing string, required []string) bool {
	for _, req := range required {
		if matchAction(req, existing) {
			return true
		}
	}
	return false
}

// matchAction checks if pattern matches action (supports wildcards)
func matchAction(pattern, action string) bool {
	if pattern == action {
		return true
	}

	// Handle wildcard patterns like "s3:GetBucket*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(action, prefix)
	}

	// Handle pattern matching against wildcards in action
	if strings.HasSuffix(action, "*") {
		prefix := strings.TrimSuffix(action, "*")
		return strings.HasPrefix(pattern, prefix)
	}

	return false
}
