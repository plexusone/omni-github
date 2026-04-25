# Release Notes - v0.2.0

**Release Date:** 2026-04-25

## Overview

This release restructures omni-github as a root-level Go module for simpler imports. The nested `omnistorage/` subdirectory has been removed.

## Breaking Changes

### Module Path Changed

**Before:**
```go
import "github.com/plexusone/omni-github/omnistorage/backend/github"
```

**After:**
```go
import "github.com/plexusone/omni-github/backend/github"
```

### Installation Changed

**Before:**
```bash
go get github.com/plexusone/omni-github/omnistorage@v0.1.1
```

**After:**
```bash
go get github.com/plexusone/omni-github@v0.2.0
```

## Migration Guide

1. Update your `go.mod`:
   ```bash
   go get github.com/plexusone/omni-github@v0.2.0
   ```

2. Update imports in your Go files:
   ```bash
   # Find and replace (example using sed)
   find . -name "*.go" -exec sed -i '' \
     's|github.com/plexusone/omni-github/omnistorage/backend/github|github.com/plexusone/omni-github/backend/github|g' {} +
   ```

3. Run `go mod tidy` to clean up dependencies.

## Dependencies

- `github.com/google/go-github/v84` v84.0.0 (upgraded from v82)

## Why This Change?

Unlike [omni-aws](https://github.com/plexusone/omni-aws) which has multiple modules (omnillm for Bedrock, omnistorage for S3), omni-github has only one module. The nested structure added unnecessary complexity:

| Repository | Modules | Structure |
|------------|---------|-----------|
| omni-aws | 2 (omnillm, omnistorage) | Nested go.mod per module |
| omni-github | 1 | Root-level go.mod |

## Requirements

- Go 1.25.0 or later
- GitHub personal access token with `repo` scope

## Documentation

- [README](README.md) - Full documentation with examples
- [CHANGELOG](CHANGELOG.md) - Version history
- [pkg.go.dev](https://pkg.go.dev/github.com/plexusone/omni-github) - API reference
