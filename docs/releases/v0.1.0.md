# Release Notes - v0.1.0

**Release Date:** 2026-01-10

## Overview

Initial release of omnistorage-github, providing a GitHub repository backend for [OmniStorage](https://github.com/grokify/omnistorage).

## Highlights

- Full `ExtendedBackend` implementation for GitHub repositories
- Read, write, and delete files with automatic commit creation
- Batch commits for atomic multi-file operations via Git Trees API
- GitHub Enterprise support

## Features

### Core Operations

- **Read files** via GitHub Contents API
- **Write files** with automatic commit creation (each write = 1 commit)
- **Delete files** with automatic commit creation (idempotent)
- **List files** via GitHub Trees API with prefix filtering
- **File metadata** (size, SHA1 hash) via Stat
- **Range read** support (offset/limit)

### Batch Commits

Combine multiple file operations into a single atomic commit:

```go
batch, err := backend.NewBatch(ctx, "Update multiple files")
batch.Write("file1.txt", []byte("content1"))
batch.Write("file2.txt", []byte("content2"))
batch.Delete("old.txt")
err = batch.Commit() // Single commit via Git Trees API
```

### Authentication & Configuration

- Personal access token authentication
- GitHub Enterprise support via custom `BaseURL`/`UploadURL`
- Configurable commit messages with `{path}` placeholder
- Configurable commit author (name and email)
- Configuration from environment variables or string maps
- Backend registry integration (`github` name)

## Installation

```bash
go get github.com/grokify/omnistorage-github@v0.1.0
```

## Quick Start

```go
import "github.com/grokify/omnistorage-github/backend/github"

backend, err := github.New(github.Config{
    Owner:  "myorg",
    Repo:   "myrepo",
    Branch: "main",
    Token:  os.Getenv("GITHUB_TOKEN"),
})
defer backend.Close()

// Read a file
r, _ := backend.NewReader(ctx, "README.md")
data, _ := io.ReadAll(r)
r.Close()

// Write a file (creates a commit)
w, _ := backend.NewWriter(ctx, "docs/example.txt")
w.Write([]byte("Hello, GitHub!"))
w.Close()
```

## Limitations

- **File size**: GitHub Contents API supports files up to 1MB (larger files require Git Blobs API, not yet implemented)
- **Rate limits**: GitHub API has rate limits (5,000 requests/hour for authenticated users)
- **Not supported**: `Copy`, `Move`, `Mkdir`, `Rmdir` return `ErrNotSupported`

## Requirements

- Go 1.24.0 or later
- GitHub personal access token with `repo` scope

## Documentation

- [README](README.md) - Full documentation with examples
- [CHANGELOG](CHANGELOG.md) - Version history
- [ROADMAP](ROADMAP.md) - Planned features
- [pkg.go.dev](https://pkg.go.dev/github.com/grokify/omnistorage-github) - API reference
