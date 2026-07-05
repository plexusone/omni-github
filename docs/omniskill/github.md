# GitHub Skill

The GitHub skill provides tools for managing issues, pull requests, and searching code on GitHub.

## Installation

```go
import "github.com/plexusone/omni-github/omniskill/github"
```

## Configuration

```go
type Config struct {
    // Token is the GitHub personal access token (required)
    Token string

    // BaseURL is the GitHub API base URL
    // Default: https://api.github.com
    // Set for GitHub Enterprise
    BaseURL string

    // DefaultOwner is the default repository owner
    DefaultOwner string

    // DefaultRepo is the default repository name
    DefaultRepo string
}
```

## Basic Usage

```go
skill := github.New(github.Config{
    Token: os.Getenv("GITHUB_TOKEN"),
})

if err := skill.Init(ctx); err != nil {
    log.Fatal(err)
}
defer skill.Close()

// Register with agent
agent.RegisterSkill(skill)
```

## Available Tools

### Issue Management

#### `list_issues`

List issues in a repository.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `state` | string | No | `open`, `closed`, or `all` (default: `open`) |
| `labels` | string | No | Comma-separated label names |
| `per_page` | integer | No | Results per page, max 100 (default: 30) |

*Uses default if configured

#### `get_issue`

Get details of a specific issue.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `number` | integer | Yes | Issue number |

#### `create_issue`

Create a new issue.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `title` | string | Yes | Issue title |
| `body` | string | No | Issue body (markdown) |
| `labels` | string | No | Comma-separated label names |
| `assignees` | string | No | Comma-separated usernames |

#### `update_issue`

Update an existing issue.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `number` | integer | Yes | Issue number |
| `title` | string | No | New title |
| `body` | string | No | New body |
| `state` | string | No | `open` or `closed` |
| `labels` | string | No | Comma-separated labels (replaces existing) |

#### `add_issue_comment`

Add a comment to an issue.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `number` | integer | Yes | Issue number |
| `body` | string | Yes | Comment body (markdown) |

### Pull Request Management

#### `list_pull_requests`

List pull requests in a repository.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `state` | string | No | `open`, `closed`, or `all` (default: `open`) |
| `base` | string | No | Filter by base branch |
| `head` | string | No | Filter by head branch (`user:branch`) |
| `per_page` | integer | No | Results per page, max 100 (default: 30) |

#### `get_pull_request`

Get details of a specific pull request.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `number` | integer | Yes | PR number |

#### `add_pull_request_comment`

Add a comment to a pull request.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `owner` | string | No* | Repository owner |
| `repo` | string | No* | Repository name |
| `number` | integer | Yes | PR number |
| `body` | string | Yes | Comment body (markdown) |

### Search

#### `search_code`

Search code across repositories.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query (GitHub syntax) |
| `per_page` | integer | No | Results per page, max 100 (default: 30) |

Example queries:

- `"TODO" repo:plexusone/omni-github` - Search for TODO in a repo
- `"func main" language:go` - Search Go main functions
- `"API_KEY" filename:.env` - Search for API keys in .env files

#### `search_issues`

Search issues and pull requests.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query (GitHub syntax) |
| `per_page` | integer | No | Results per page, max 100 (default: 30) |

Example queries:

- `is:issue is:open label:bug` - Open bug issues
- `is:pr is:merged author:username` - Merged PRs by user
- `"memory leak" is:issue` - Issues mentioning memory leak

## GitHub Enterprise

```go
skill := github.New(github.Config{
    Token:   os.Getenv("GITHUB_TOKEN"),
    BaseURL: "https://github.mycompany.com/api/v3",
})
```

## Default Repository

Set defaults to avoid repeating owner/repo:

```go
skill := github.New(github.Config{
    Token:        os.Getenv("GITHUB_TOKEN"),
    DefaultOwner: "plexusone",
    DefaultRepo:  "omni-github",
})

// Now you can omit owner/repo
result, _ := listTool.Execute(ctx, map[string]any{
    "state": "open",
})
```

## Error Handling

All tools return errors for:

- Missing required parameters
- API errors (rate limits, permissions, not found)
- Network errors

```go
result, err := tool.Execute(ctx, params)
if err != nil {
    // Handle error
    log.Printf("Tool failed: %v", err)
}
```
