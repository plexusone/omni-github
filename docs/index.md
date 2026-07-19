# Omni-GitHub

GitHub providers and skills for the PlexusOne ecosystem.

## Overview

Omni-GitHub provides three main components:

| Package | Description |
|---------|-------------|
| [`omnistorage/`](omnistorage/overview.md) | GitHub as a storage backend for [omnistorage-core](https://github.com/plexusone/omnistorage-core) |
| [`omniskill/github/`](omniskill/github.md) | GitHub skill for AI agents (issues, PRs, code search) |
| [`omnidevx/`](omnidevx/overview.md) | GitHub DevX contribution collector for [omnidevx-core](https://github.com/plexusone/omnidevx-core) |

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

=== "DevX Collector"

    ```go
    import "github.com/plexusone/omni-github/omnidevx"

    collector, err := omnidevx.New(omnidevx.Config{
        Token:    os.Getenv("GITHUB_TOKEN"),
        Username: "octocat",
    })

    result, err := collector.Collect(ctx, core.CollectRequest{
        Subject: core.SubjectRef{PersonID: "person:octocat"},
        Period:  core.Period{Start: periodStart, End: periodEnd},
    })
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

### DevX Collector

- 📊 Profile, per-repository, and daily contribution snapshots
- 🔄 Canonical `devx.*` events for the OmniDevX domain
- 🧮 REST + GraphQL via `go-github` and `githubv4`

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
- [omnidevx-core](https://github.com/plexusone/omnidevx-core) - DevX event domain
- [omniagent](https://github.com/plexusone/omniagent) - AI agent runtime
