# Quick Start

This guide shows how to use both the storage provider and GitHub skill.

## Storage Provider

### Reading Files

```go
package main

import (
    "context"
    "io"
    "log"
    "os"

    "github.com/plexusone/omni-github/omnistorage"
)

func main() {
    backend, err := omnistorage.New(omnistorage.Config{
        Owner:  "plexusone",
        Repo:   "omnistorage-core",
        Branch: "main",
        Token:  os.Getenv("GITHUB_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer backend.Close()

    ctx := context.Background()

    // Read a file
    r, err := backend.NewReader(ctx, "README.md")
    if err != nil {
        log.Fatal(err)
    }
    defer r.Close()

    data, _ := io.ReadAll(r)
    log.Println(string(data))
}
```

### Writing Files

```go
// Write a new file (creates a commit)
w, err := backend.NewWriter(ctx, "docs/example.txt")
if err != nil {
    log.Fatal(err)
}
w.Write([]byte("Hello, GitHub!"))
w.Close() // Commits the file
```

### Batch Operations

```go
// Create a batch for atomic multi-file commits
batch, err := backend.NewBatch(ctx, "Update multiple files")
if err != nil {
    log.Fatal(err)
}

batch.Write("file1.txt", []byte("content 1"))
batch.Write("file2.txt", []byte("content 2"))
batch.Delete("old-file.txt")

if err := batch.Commit(); err != nil {
    log.Fatal(err)
}
```

## GitHub Skill

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/plexusone/omni-github/omniskill/github"
)

func main() {
    ctx := context.Background()

    skill := github.New(github.Config{
        Token:        os.Getenv("GITHUB_TOKEN"),
        DefaultOwner: "plexusone",
        DefaultRepo:  "omni-github",
    })

    if err := skill.Init(ctx); err != nil {
        log.Fatal(err)
    }
    defer skill.Close()

    // Get available tools
    for _, tool := range skill.Tools() {
        log.Printf("Tool: %s - %s", tool.Name(), tool.Description())
    }
}
```

### With OmniAgent

```go
import (
    "github.com/plexusone/omniagent/agent"
    "github.com/plexusone/omni-github/omniskill/github"
)

// Create GitHub skill
githubSkill := github.New(github.Config{
    Token: os.Getenv("GITHUB_TOKEN"),
})

// Register with agent
agent, err := agent.New(config,
    agent.WithSkills(githubSkill),
)
```

### Calling Tools Directly

```go
// List open issues
listTool := skill.Tools()[0] // list_issues
result, err := listTool.Execute(ctx, map[string]any{
    "owner": "plexusone",
    "repo":  "omni-github",
    "state": "open",
})

// Create an issue
createTool := skill.Tools()[2] // create_issue
result, err := createTool.Execute(ctx, map[string]any{
    "owner":  "plexusone",
    "repo":   "omni-github",
    "title":  "Bug report",
    "body":   "Description of the bug",
    "labels": "bug,priority:high",
})
```

## Next Steps

- [Storage Configuration](../omnistorage/configuration.md) - Detailed configuration options
- [GitHub Skill Tools](../omniskill/github.md) - Complete tool reference
