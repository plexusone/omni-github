# Omni-GitHub

GitHub providers and skills for the PlexusOne ecosystem.

## Overview

Omni-GitHub provides two main components:

| Package | Description |
|---------|-------------|
| [`omnistorage/`](omnistorage/overview.md) | GitHub as a storage backend for [omnistorage-core](https://github.com/plexusone/omnistorage-core) |
| [`omniskill/github/`](omniskill/github.md) | GitHub skill for AI agents (issues, PRs, code search) |

## Quick Start

=== "Storage Provider"

    ```go
    import "github.com/plexusone/omni-github/omnistorage"

    backend, err := omnistorage.New(omnistorage.Config{
        Owner:  "myorg",
        Repo:   "myrepo",
        Branch: "main",
        Token:  os.Getenv("GITHUB_TOKEN"),
    })
    ```

=== "GitHub Skill"

    ```go
    import "github.com/plexusone/omni-github/omniskill/github"

    skill := github.New(github.Config{
        Token: os.Getenv("GITHUB_TOKEN"),
    })
    if err := skill.Init(ctx); err != nil {
        log.Fatal(err)
    }
    agent.RegisterSkill(skill)
    ```

## Features

### Storage Provider

- 📄 Read and write files to any branch
- ⚡ Batch multiple file operations into a single atomic commit
- 📂 List files with prefix filtering
- 🏢 GitHub Enterprise support

### GitHub Skill

- 🎫 Issue management (list, create, update, comment)
- 🔀 Pull request operations (list, get, comment)
- 🔍 Code and issue search
- 🏢 GitHub Enterprise support

## Installation

```bash
go get github.com/plexusone/omni-github
```

## Requirements

- Go 1.26 or later
- GitHub personal access token

## Related Projects

- [omnistorage-core](https://github.com/plexusone/omnistorage-core) - Storage abstraction library
- [omniskill](https://github.com/plexusone/omniskill) - Skill interface library
- [omniagent](https://github.com/plexusone/omniagent) - AI agent runtime
