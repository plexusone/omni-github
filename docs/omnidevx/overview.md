# DevX Collector Overview

The `omnidevx/` package collects developer-experience events from GitHub for the OmniDevX domain ([`omnidevx-core`](https://github.com/plexusone/omnidevx-core)), following the omni-github per-domain package convention alongside `omnistorage` and `omniskill`.

## Configuration

```go
import "github.com/plexusone/omni-github/omnidevx"

collector, err := omnidevx.New(omnidevx.Config{
    Token:    os.Getenv("GITHUB_TOKEN"),
    Username: "octocat",
})
```

| Field | Description |
|-------|-------------|
| `Token` | GitHub API token. Required unless both clients are set. |
| `Username` | GitHub account whose contributions are collected. Required. |
| `RESTClient` | Overrides the token-derived REST client (for testing or custom transports). |
| `GraphQLClient` | Overrides the token-derived GraphQL client. |

## Collecting Events

```go
result, err := collector.Collect(ctx, core.CollectRequest{
    Subject: core.SubjectRef{PersonID: "person:octocat"},
    Period: core.Period{
        Start: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
        End:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
    },
})
```

`Collect` requires a bounded period (both `Start` and `End` set) — GitHub's contribution query needs an explicit range, and unbounded scans of account history are rarely intended.

## Event Types

Contribution data comes from [`gogithub/profile`](https://github.com/grokify/gogithub) (GraphQL contribution calendar and per-repository commit statistics), normalized into canonical `devx.*` events:

| Event | Description |
|-------|-------------|
| `devx.profile.snapshot` | Whole-profile period totals: commits, PRs, issues, reviews, repos created, repos contributed to, insertions, deletions |
| `devx.contribution.snapshot` | Per-repository period totals: commits, insertions, deletions, releases, private-repo flag |
| `devx.contribution.recorded` | Daily contribution-calendar counts (zero-count days are skipped) |

All events carry API-mode provenance (`core.ModeAPI`) since they're observed from the live GitHub API rather than reconstructed from local git history.

## Overlap with Local Git History

GitHub-reported contributions overlap with locally-collected git commits (e.g. the `providers/git` collector in `omnidevx-core`). The aggregation layer keeps the two sources separate via `bySource` metrics rather than summing them, since GitHub's counting rules (default branches, squash merges, private-repo restrictions) differ from local git history.

## Architecture

```
omni-github/
├── omnistorage/         # Storage provider
├── omniskill/
│   └── github/          # GitHub skill for AI agents
└── omnidevx/             # GitHub DevX collector (current)
```
