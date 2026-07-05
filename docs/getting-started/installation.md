# Installation

## Requirements

- Go 1.26 or later
- GitHub personal access token with appropriate scopes

## Installing the Package

```bash
go get github.com/plexusone/omni-github
```

## GitHub Token Scopes

### For Storage Provider

The storage provider needs access to repository contents:

| Scope | Required For |
|-------|--------------|
| `repo` | Full repository access (private repos) |
| `public_repo` | Public repository access only |

### For GitHub Skill

The GitHub skill needs access to issues and pull requests:

| Scope | Required For |
|-------|--------------|
| `repo` | Full repository access |
| `read:org` | List organization repositories |

## Setting Up Authentication

### Environment Variable (Recommended)

```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
```

### GitHub CLI

If you have the GitHub CLI installed:

```bash
gh auth token
```

### GitHub Enterprise

For GitHub Enterprise, you'll also need to set the base URL:

```bash
export GITHUB_API_URL="https://github.mycompany.com/api/v3"
```

## Verifying Installation

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/plexusone/omni-github/omniskill/github"
)

func main() {
    skill := github.New(github.Config{
        Token: os.Getenv("GITHUB_TOKEN"),
    })

    if err := skill.Init(context.Background()); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("GitHub skill initialized: %s\n", skill.Name())
}
```

## Next Steps

- [Quick Start](quickstart.md) - Get started with examples
- [Storage Provider](../omnistorage/overview.md) - Use GitHub as storage
- [GitHub Skill](../omniskill/github.md) - Use GitHub tools with AI agents
