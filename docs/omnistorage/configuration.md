# Storage Configuration

## Config Struct

```go
type Config struct {
    // Owner is the repository owner (required)
    Owner string

    // Repo is the repository name (required)
    Repo string

    // Branch is the branch name (default: "main")
    Branch string

    // Token is the GitHub personal access token (required)
    Token string

    // BaseURL is the API base URL (default: "https://api.github.com/")
    BaseURL string

    // UploadURL is the upload URL (default: "https://uploads.github.com/")
    UploadURL string

    // CommitMessage is the commit message template.
    // Use {path} as a placeholder for the file path.
    // Default: "Update {path} via omnistorage"
    CommitMessage string

    // CommitAuthor is the author for commits.
    // If nil, uses the authenticated user.
    CommitAuthor *CommitAuthor
}

type CommitAuthor struct {
    Name  string
    Email string
}
```

## Basic Configuration

```go
backend, err := omnistorage.New(omnistorage.Config{
    Owner:  "myorg",
    Repo:   "myrepo",
    Branch: "main",
    Token:  os.Getenv("GITHUB_TOKEN"),
})
```

## Custom Commit Messages

```go
backend, err := omnistorage.New(omnistorage.Config{
    Owner:         "myorg",
    Repo:          "myrepo",
    Token:         os.Getenv("GITHUB_TOKEN"),
    CommitMessage: "[bot] Update {path}",
    CommitAuthor: &omnistorage.CommitAuthor{
        Name:  "My Bot",
        Email: "bot@example.com",
    },
})
```

## GitHub Enterprise

```go
backend, err := omnistorage.New(omnistorage.Config{
    Owner:     "myorg",
    Repo:      "myrepo",
    Branch:    "main",
    Token:     os.Getenv("GITHUB_TOKEN"),
    BaseURL:   "https://github.example.com/api/v3/",
    UploadURL: "https://github.example.com/uploads/",
})
```

## Environment Variables

Configuration can be loaded from environment variables:

```go
cfg := omnistorage.ConfigFromEnv()
backend, err := omnistorage.New(cfg)
```

| Variable | Fallback | Description |
|----------|----------|-------------|
| `OMNISTORAGE_GITHUB_OWNER` | `GITHUB_OWNER` | Repository owner |
| `OMNISTORAGE_GITHUB_REPO` | `GITHUB_REPO` | Repository name |
| `OMNISTORAGE_GITHUB_BRANCH` | - | Branch name (default: "main") |
| `OMNISTORAGE_GITHUB_TOKEN` | `GITHUB_TOKEN` | Personal access token |
| `OMNISTORAGE_GITHUB_BASE_URL` | `GITHUB_API_URL` | API base URL |
| `OMNISTORAGE_GITHUB_UPLOAD_URL` | - | Upload URL |
| `OMNISTORAGE_GITHUB_COMMIT_MESSAGE` | - | Commit message template |
| `OMNISTORAGE_GITHUB_COMMIT_AUTHOR_NAME` | - | Commit author name |
| `OMNISTORAGE_GITHUB_COMMIT_AUTHOR_EMAIL` | - | Commit author email |

## Using the Registry

The backend automatically registers with the omnistorage registry:

```go
import (
    omnistorage "github.com/plexusone/omnistorage-core/object"
    _ "github.com/plexusone/omni-github/omnistorage" // Register backend
)

backend, err := omnistorage.Open("github", map[string]string{
    "owner":  "myorg",
    "repo":   "myrepo",
    "branch": "main",
    "token":  os.Getenv("GITHUB_TOKEN"),
})
```
