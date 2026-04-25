// Package github provides a GitHub repository backend for omnistorage.
//
// This backend provides read and write access to files in GitHub repositories.
// It uses the GitHub Contents API to read and write file contents.
//
// Basic usage:
//
//	cfg := github.Config{
//	    Owner: "grokify",
//	    Repo:  "omnistorage",
//	    Token: os.Getenv("GITHUB_TOKEN"),
//	}
//	backend, err := github.New(cfg)
package github

import (
	"errors"
	"os"
	"strings"
)

// Config errors.
var (
	ErrOwnerRequired = errors.New("github: owner is required")
	ErrRepoRequired  = errors.New("github: repo is required")
	ErrTokenRequired = errors.New("github: token is required")
)

// CommitAuthor represents the author of a commit.
type CommitAuthor struct {
	Name  string
	Email string
}

// Config holds configuration for the GitHub backend.
type Config struct {
	// Owner is the repository owner (user or organization). Required.
	Owner string

	// Repo is the repository name. Required.
	Repo string

	// Branch is the branch to read from. Default: "main".
	Branch string

	// Token is the GitHub personal access token. Required.
	// Needs "repo" scope for private repos, or "public_repo" for public repos.
	Token string

	// BaseURL is the GitHub API base URL. Default: "https://api.github.com/".
	// Set this for GitHub Enterprise (e.g., "https://github.example.com/api/v3/").
	BaseURL string

	// UploadURL is the GitHub upload URL. Default: "https://uploads.github.com/".
	// Set this for GitHub Enterprise.
	UploadURL string

	// CommitMessage is the commit message template for write operations.
	// Use {path} as a placeholder for the file path.
	// Default: "Update {path} via omnistorage"
	CommitMessage string

	// CommitAuthor is the author for commits. If nil, uses the authenticated user.
	CommitAuthor *CommitAuthor
}

// FormatCommitMessage formats the commit message with the given path.
func (c *Config) FormatCommitMessage(filePath string) string {
	msg := c.CommitMessage
	if msg == "" {
		msg = "Update {path} via omnistorage"
	}
	return strings.ReplaceAll(msg, "{path}", filePath)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Branch:        "main",
		BaseURL:       "https://api.github.com/",
		UploadURL:     "https://uploads.github.com/",
		CommitMessage: "Update {path} via omnistorage",
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Owner == "" {
		return ErrOwnerRequired
	}
	if c.Repo == "" {
		return ErrRepoRequired
	}
	if c.Token == "" {
		return ErrTokenRequired
	}
	return nil
}

// ConfigFromMap creates a Config from a string map.
// Supported keys:
//   - owner: repository owner (required)
//   - repo: repository name (required)
//   - branch: branch name (default: "main")
//   - token: GitHub personal access token (required)
//   - base_url: GitHub API base URL (for GitHub Enterprise)
//   - upload_url: GitHub upload URL (for GitHub Enterprise)
//   - commit_message: commit message template (default: "Update {path} via omnistorage")
//   - commit_author_name: commit author name
//   - commit_author_email: commit author email
func ConfigFromMap(m map[string]string) Config {
	cfg := DefaultConfig()

	if v, ok := m["owner"]; ok {
		cfg.Owner = v
	}
	if v, ok := m["repo"]; ok {
		cfg.Repo = v
	}
	if v, ok := m["branch"]; ok && v != "" {
		cfg.Branch = v
	}
	if v, ok := m["token"]; ok {
		cfg.Token = v
	}
	if v, ok := m["base_url"]; ok && v != "" {
		cfg.BaseURL = v
	}
	if v, ok := m["upload_url"]; ok && v != "" {
		cfg.UploadURL = v
	}
	if v, ok := m["commit_message"]; ok && v != "" {
		cfg.CommitMessage = v
	}

	// Commit author
	authorName := m["commit_author_name"]
	authorEmail := m["commit_author_email"]
	if authorName != "" || authorEmail != "" {
		cfg.CommitAuthor = &CommitAuthor{
			Name:  authorName,
			Email: authorEmail,
		}
	}

	return cfg
}

// ConfigFromEnv creates a Config from environment variables.
// Environment variables:
//   - OMNISTORAGE_GITHUB_OWNER or GITHUB_OWNER: repository owner
//   - OMNISTORAGE_GITHUB_REPO or GITHUB_REPO: repository name
//   - OMNISTORAGE_GITHUB_BRANCH: branch name (default: "main")
//   - OMNISTORAGE_GITHUB_TOKEN or GITHUB_TOKEN: personal access token
//   - OMNISTORAGE_GITHUB_BASE_URL or GITHUB_API_URL: API base URL
//   - OMNISTORAGE_GITHUB_UPLOAD_URL: upload URL
//   - OMNISTORAGE_GITHUB_COMMIT_MESSAGE: commit message template
//   - OMNISTORAGE_GITHUB_COMMIT_AUTHOR_NAME: commit author name
//   - OMNISTORAGE_GITHUB_COMMIT_AUTHOR_EMAIL: commit author email
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	// Owner
	if v := os.Getenv("OMNISTORAGE_GITHUB_OWNER"); v != "" {
		cfg.Owner = v
	} else if v := os.Getenv("GITHUB_OWNER"); v != "" {
		cfg.Owner = v
	}

	// Repo
	if v := os.Getenv("OMNISTORAGE_GITHUB_REPO"); v != "" {
		cfg.Repo = v
	} else if v := os.Getenv("GITHUB_REPO"); v != "" {
		cfg.Repo = v
	}

	// Branch
	if v := os.Getenv("OMNISTORAGE_GITHUB_BRANCH"); v != "" {
		cfg.Branch = v
	}

	// Token
	if v := os.Getenv("OMNISTORAGE_GITHUB_TOKEN"); v != "" {
		cfg.Token = v
	} else if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		cfg.Token = v
	}

	// Base URL
	if v := os.Getenv("OMNISTORAGE_GITHUB_BASE_URL"); v != "" {
		cfg.BaseURL = v
	} else if v := os.Getenv("GITHUB_API_URL"); v != "" {
		cfg.BaseURL = v
	}

	// Upload URL
	if v := os.Getenv("OMNISTORAGE_GITHUB_UPLOAD_URL"); v != "" {
		cfg.UploadURL = v
	}

	// Commit message
	if v := os.Getenv("OMNISTORAGE_GITHUB_COMMIT_MESSAGE"); v != "" {
		cfg.CommitMessage = v
	}

	// Commit author
	authorName := os.Getenv("OMNISTORAGE_GITHUB_COMMIT_AUTHOR_NAME")
	authorEmail := os.Getenv("OMNISTORAGE_GITHUB_COMMIT_AUTHOR_EMAIL")
	if authorName != "" || authorEmail != "" {
		cfg.CommitAuthor = &CommitAuthor{
			Name:  authorName,
			Email: authorEmail,
		}
	}

	return cfg
}
