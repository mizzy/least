# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`least` is a Go CLI tool that analyzes Terraform configurations to generate minimal least-privilege IAM policies. It uses CloudFormation Resource Schemas as the authoritative source for IAM permissions.

## Build Commands

```bash
# Build
go build -o least ./cmd/least

# Install globally
go install github.com/mizzy/least/cmd/least@latest

# Update permission mappings from CloudFormation schemas
./scripts/fetch-schemas.sh       # Fetch schemas (requires AWS CLI)
go generate ./internal/mapping   # Generate Go code
```

## Architecture

The codebase uses a **Provider Registry Pattern** for extensibility:

- **cmd/least/**: CLI entry point (Cobra-based)
- **internal/provider/**: Provider abstraction with auto-detection
  - `terraform/`: Parses `.tf` files using HashiCorp HCL/v2
  - `cloudformation/`: Stub for future support
- **internal/mapping/**: Two-tier IAM permission mappings
  - `mapping.go`: Fallback mappings for common resources
  - `generated.go`: Auto-generated from CloudFormation schemas (overrides fallback)
  - `gen/main.go`: Code generator
- **internal/policy/**: Generates IAM policies in JSON or Terraform HCL format
- **internal/checker/**: Wildcard-aware policy compliance checking

**Key Pattern**: Generated mappings from CloudFormation schemas override hardcoded fallback mappings at runtime.

## CLI Commands

```bash
least generate [path]              # Generate IAM policy (default: Terraform HCL format)
least generate [path] -f json      # Output as JSON
least check [path] -p policy.json  # Check policy compliance
```

Exit codes for `check`: 0=compliant, 1=missing permissions, 2=excessive permissions

## Adding a New Provider

1. Create `internal/provider/<name>/<name>.go`
2. Implement the `provider.Provider` interface (Name, Detect, Parse, FileExtensions)
3. Register in `cmd/least/main.go`
