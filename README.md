# Omni-GitHub

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omni-github/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-github/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-github/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-github/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omni-github/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omni-github/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-github
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-github
 [docs-mkdoc-svg]: https://img.shields.io/badge/docs-guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.github.io/omni-github
 [viz-svg]: https://img.shields.io/badge/repo-visualization-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomni-github
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-github/blob/main/LICENSE

GitHub providers and skills for the PlexusOne ecosystem.

## Packages

| Package | Description | Documentation |
|---------|-------------|---------------|
| `omnistorage/` | GitHub as storage backend | [Guide](https://plexusone.github.io/omni-github/omnistorage/overview/) |
| `omniskill/github/` | GitHub skill for AI agents | [Guide](https://plexusone.github.io/omni-github/omniskill/github/) |
| `omnidevx/` | GitHub DevX contribution collector | [Guide](https://plexusone.github.io/omni-github/omnidevx/overview/) |

## Installation

```bash
go get github.com/plexusone/omni-github
```

## Quick Start

### Storage Provider

```go
import "github.com/plexusone/omni-github/omnistorage"

backend, err := omnistorage.New(omnistorage.Config{
    Owner:  "myorg",
    Repo:   "myrepo",
    Branch: "main",
    Token:  os.Getenv("GITHUB_TOKEN"),
})

// Read files
r, _ := backend.NewReader(ctx, "README.md")
data, _ := io.ReadAll(r)

// Write files (creates commit)
w, _ := backend.NewWriter(ctx, "docs/example.txt")
w.Write([]byte("Hello!"))
w.Close()

// Batch operations (single commit)
batch, _ := backend.NewBatch(ctx, "Update files")
batch.Write("file1.txt", []byte("content"))
batch.Delete("old.txt")
batch.Commit()
```

### GitHub Skill

```go
import "github.com/plexusone/omni-github/omniskill/github"

skill := github.New(github.Config{
    Token: os.Getenv("GITHUB_TOKEN"),
})
skill.Init(ctx)

// 10 tools: list_issues, create_issue, search_code, etc.
agent.RegisterSkill(skill)
```

### DevX Collector

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

- 📄 Read/write files to any branch
- ⚡ Batch operations for atomic commits
- 📂 List files with prefix filtering
- 🏢 GitHub Enterprise support

### GitHub Skill

- 🎫 Issue management (list, create, update, comment)
- 🔀 Pull request operations
- 🔍 Code and issue search
- 🏢 GitHub Enterprise support

### DevX Collector

- 📊 Profile, per-repository, and daily contribution snapshots
- 🔄 Canonical `devx.*` events for the OmniDevX domain
- 🧮 REST + GraphQL via `go-github` and `githubv4`

## Documentation

Full documentation at [plexusone.github.io/omni-github](https://plexusone.github.io/omni-github)

- [Installation](https://plexusone.github.io/omni-github/getting-started/installation/)
- [Quick Start](https://plexusone.github.io/omni-github/getting-started/quickstart/)
- [Storage Provider](https://plexusone.github.io/omni-github/omnistorage/overview/)
- [GitHub Skill](https://plexusone.github.io/omni-github/omniskill/github/)
- [DevX Collector](https://plexusone.github.io/omni-github/omnidevx/overview/)

## Requirements

- Go 1.26 or later
- GitHub personal access token

## Related Projects

- [omnistorage-core](https://github.com/plexusone/omnistorage-core) - Storage abstraction
- [omniskill](https://github.com/plexusone/omniskill) - Skill interface
- [omnidevx-core](https://github.com/plexusone/omnidevx-core) - DevX event domain
- [omniagent](https://github.com/plexusone/omniagent) - AI agent runtime
- [omniagent-starter](https://github.com/plexusone/omniagent-starter) - Batteries-included bundle

## License

MIT License - see [LICENSE](LICENSE) for details.
