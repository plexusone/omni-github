# Skills Overview

The `omniskill/` directory contains GitHub skills for AI agents. Skills implement the [omniskill](https://github.com/plexusone/omniskill) `Skill` interface, providing tools that agents can use.

## Available Skills

| Skill | Package | Description |
|-------|---------|-------------|
| [GitHub](github.md) | `omniskill/github` | Issues, PRs, and code search |

## Architecture

```
omni-github/
├── omnistorage/         # Storage provider
└── omniskill/
    └── github/          # GitHub skill (current)
    └── actions/         # GitHub Actions skill (planned)
    └── releases/        # Releases skill (planned)
```

## Using Skills with OmniAgent

```go
import (
    "github.com/plexusone/omniagent/agent"
    "github.com/plexusone/omni-github/omniskill/github"
)

// Create skill
githubSkill := github.New(github.Config{
    Token: os.Getenv("GITHUB_TOKEN"),
})

// Initialize
if err := githubSkill.Init(ctx); err != nil {
    log.Fatal(err)
}

// Register with agent
agent, err := agent.New(config,
    agent.WithSkills(githubSkill),
)
```

## Using with OmniAgent Starter

For a batteries-included setup:

```go
import starter "github.com/plexusone/omniagent-starter"

bundle := starter.Default(starter.Config{
    GitHubToken: os.Getenv("GITHUB_TOKEN"),
})

agent, err := agent.New(config,
    agent.WithSkills(bundle.Skills()...),
)
```

## Skill Interface

All skills implement the `skill.Skill` interface:

```go
type Skill interface {
    Name() string
    Description() string
    Init(ctx context.Context) error
    Close() error
    Tools() []Tool
}
```

## Planned Skills

### GitHub Actions (`omniskill/actions/`)

- List workflow runs
- Trigger workflows
- Get workflow logs
- Cancel runs

### Releases (`omniskill/releases/`)

- List releases
- Create releases
- Upload assets
- Manage release notes
