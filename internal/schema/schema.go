// Package schema provides CloudFormation Resource Schema parsing
// to extract IAM permissions required for each resource type.
//
// CloudFormation Resource Schemas contain "handlers" sections that
// explicitly list IAM permissions needed for create/read/update/delete/list
// operations. This is the authoritative source for AWS resource permissions.
package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ResourceSchema represents a CloudFormation Resource Schema
type ResourceSchema struct {
	TypeName    string   `json:"typeName"`
	Description string   `json:"description"`
	Handlers    Handlers `json:"handlers"`
}

// Handlers contains the CRUD handlers for a resource
type Handlers struct {
	Create *Handler `json:"create"`
	Read   *Handler `json:"read"`
	Update *Handler `json:"update"`
	Delete *Handler `json:"delete"`
	List   *Handler `json:"list"`
}

// Handler represents a single operation handler
type Handler struct {
	Permissions []string `json:"permissions"`
}

// Store manages CloudFormation resource schemas
type Store struct {
	schemas   map[string]*ResourceSchema
	cacheDir  string
	mu        sync.RWMutex
}

// NewStore creates a new schema store
func NewStore(cacheDir string) *Store {
	return &Store{
		schemas:  make(map[string]*ResourceSchema),
		cacheDir: cacheDir,
	}
}

// GetPermissions returns IAM permissions for a CloudFormation resource type
func (s *Store) GetPermissions(cfnType string) (*Permissions, error) {
	s.mu.RLock()
	schema, ok := s.schemas[cfnType]
	s.mu.RUnlock()

	if !ok {
		// Try to load from cache
		var err error
		schema, err = s.loadFromCache(cfnType)
		if err != nil {
			return nil, fmt.Errorf("schema not found for %s: %w", cfnType, err)
		}
		s.mu.Lock()
		s.schemas[cfnType] = schema
		s.mu.Unlock()
	}

	return extractPermissions(schema), nil
}

// Permissions contains IAM permissions for each operation
type Permissions struct {
	Create []string
	Read   []string
	Update []string
	Delete []string
	List   []string
	All    []string // Deduplicated combination of all
}

func extractPermissions(schema *ResourceSchema) *Permissions {
	p := &Permissions{}
	seen := make(map[string]bool)

	addActions := func(handler *Handler, target *[]string) {
		if handler == nil {
			return
		}
		for _, action := range handler.Permissions {
			*target = append(*target, action)
			if !seen[action] {
				seen[action] = true
				p.All = append(p.All, action)
			}
		}
	}

	addActions(schema.Handlers.Create, &p.Create)
	addActions(schema.Handlers.Read, &p.Read)
	addActions(schema.Handlers.Update, &p.Update)
	addActions(schema.Handlers.Delete, &p.Delete)
	addActions(schema.Handlers.List, &p.List)

	return p
}

// LoadSchema loads a schema from JSON data
func (s *Store) LoadSchema(data []byte) error {
	var schema ResourceSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parsing schema: %w", err)
	}

	s.mu.Lock()
	s.schemas[schema.TypeName] = &schema
	s.mu.Unlock()

	return nil
}

// LoadSchemaFile loads a schema from a JSON file
func (s *Store) LoadSchemaFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading schema file: %w", err)
	}
	return s.LoadSchema(data)
}

// LoadSchemaDir loads all schema files from a directory
func (s *Store) LoadSchemaDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading schema directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := s.LoadSchemaFile(path); err != nil {
			// Log but continue with other files
			continue
		}
	}

	return nil
}

func (s *Store) loadFromCache(cfnType string) (*ResourceSchema, error) {
	if s.cacheDir == "" {
		return nil, fmt.Errorf("no cache directory configured")
	}

	// Convert AWS::S3::Bucket to aws-s3-bucket.json
	filename := strings.ToLower(strings.ReplaceAll(cfnType, "::", "-")) + ".json"
	path := filepath.Join(s.cacheDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var schema ResourceSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// SaveToCache saves a schema to the cache directory
func (s *Store) SaveToCache(schema *ResourceSchema) error {
	if s.cacheDir == "" {
		return fmt.Errorf("no cache directory configured")
	}

	if err := os.MkdirAll(s.cacheDir, 0755); err != nil {
		return err
	}

	filename := strings.ToLower(strings.ReplaceAll(schema.TypeName, "::", "-")) + ".json"
	path := filepath.Join(s.cacheDir, filename)

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ListLoadedTypes returns all loaded resource types
func (s *Store) ListLoadedTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]string, 0, len(s.schemas))
	for t := range s.schemas {
		types = append(types, t)
	}
	return types
}
