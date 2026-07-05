// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package github provides a GitHub skill for interacting with GitHub repositories.
//
// This skill provides tools for managing issues, pull requests, and searching code.
// Authentication is via a GitHub personal access token.
//
// Example usage:
//
//	import "github.com/plexusone/omni-github/omniskill/github"
//
//	skill := github.New(github.Config{
//		Token: os.Getenv("GITHUB_TOKEN"),
//	})
//	if err := skill.Init(ctx); err != nil {
//		log.Fatal(err)
//	}
//	defer skill.Close()
//
//	// Use with an agent
//	agent.RegisterSkill(skill)
package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v88/github"
	"github.com/plexusone/omniskill/skill"
)

// Config configures the GitHub skill.
type Config struct {
	// Token is the GitHub personal access token.
	// Required for authenticated API access.
	Token string

	// BaseURL is the GitHub API base URL.
	// Defaults to https://api.github.com for github.com.
	// Set to your GitHub Enterprise URL for GHE (e.g., https://github.mycompany.com/api/v3).
	BaseURL string

	// DefaultOwner is the default repository owner for operations.
	// Can be overridden per-tool call.
	DefaultOwner string

	// DefaultRepo is the default repository name for operations.
	// Can be overridden per-tool call.
	DefaultRepo string
}

// Skill provides GitHub integration tools.
type Skill struct {
	config Config
	client *gh.Client
}

// New creates a new GitHub skill with the given configuration.
func New(config Config) *Skill {
	return &Skill{config: config}
}

// Name returns the skill name.
func (s *Skill) Name() string {
	return "github"
}

// Description returns the skill description.
func (s *Skill) Description() string {
	return "Interact with GitHub repositories: issues, pull requests, code search"
}

// Init initializes the GitHub client.
func (s *Skill) Init(ctx context.Context) error {
	if s.config.Token == "" {
		return fmt.Errorf("github: token is required")
	}

	// Create GitHub client with auth token
	var opts []gh.ClientOptionsFunc
	opts = append(opts, gh.WithAuthToken(s.config.Token))

	// Add enterprise URLs if configured
	if s.config.BaseURL != "" {
		opts = append(opts, gh.WithEnterpriseURLs(s.config.BaseURL, s.config.BaseURL))
	}

	client, err := gh.NewClient(opts...)
	if err != nil {
		return fmt.Errorf("github: create client: %w", err)
	}
	s.client = client

	return nil
}

// Close releases resources.
func (s *Skill) Close() error {
	return nil
}

// Tools returns the GitHub tools.
func (s *Skill) Tools() []skill.Tool {
	return []skill.Tool{
		s.listIssuesTool(),
		s.getIssueTool(),
		s.createIssueTool(),
		s.updateIssueTool(),
		s.addIssueCommentTool(),
		s.listPullRequestsTool(),
		s.getPullRequestTool(),
		s.addPullRequestCommentTool(),
		s.searchCodeTool(),
		s.searchIssuesTool(),
	}
}

// Ensure Skill implements skill.Skill.
var _ skill.Skill = (*Skill)(nil)

// getOwnerRepo extracts owner and repo from params, falling back to defaults.
func (s *Skill) getOwnerRepo(params map[string]any) (string, string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)

	if owner == "" {
		owner = s.config.DefaultOwner
	}
	if repo == "" {
		repo = s.config.DefaultRepo
	}

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("owner and repo are required")
	}

	return owner, repo, nil
}

// listIssuesTool lists issues in a repository.
func (s *Skill) listIssuesTool() skill.Tool {
	return skill.NewTool(
		"list_issues",
		"List issues in a GitHub repository",
		map[string]skill.Parameter{
			"owner":    {Type: "string", Description: "Repository owner (username or org)"},
			"repo":     {Type: "string", Description: "Repository name"},
			"state":    {Type: "string", Description: "Issue state", Enum: []any{"open", "closed", "all"}, Default: "open"},
			"labels":   {Type: "string", Description: "Comma-separated list of label names"},
			"per_page": {Type: "integer", Description: "Results per page (max 100)", Default: 30},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			owner, repo, err := s.getOwnerRepo(params)
			if err != nil {
				return nil, err
			}

			opts := &gh.IssueListByRepoOptions{
				State: getString(params, "state", "open"),
				ListOptions: gh.ListOptions{
					PerPage: getInt(params, "per_page", 30),
				},
			}

			if labels, ok := params["labels"].(string); ok && labels != "" {
				opts.Labels = strings.Split(labels, ",")
			}

			issues, _, err := s.client.Issues.ListByRepo(ctx, owner, repo, opts)
			if err != nil {
				return nil, fmt.Errorf("list issues: %w", err)
			}

			return formatIssues(issues), nil
		},
	)
}

// getIssueTool gets a specific issue.
func (s *Skill) getIssueTool() skill.Tool {
	return skill.NewTool(
		"get_issue",
		"Get details of a specific GitHub issue",
		map[string]skill.Parameter{
			"owner":  {Type: "string", Description: "Repository owner"},
			"repo":   {Type: "string", Description: "Repository name"},
			"number": {Type: "integer", Description: "Issue number", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			owner, repo, err := s.getOwnerRepo(params)
			if err != nil {
				return nil, err
			}

			number := getInt(params, "number", 0)
			if number == 0 {
				return nil, fmt.Errorf("issue number is required")
			}

			issue, _, err := s.client.Issues.Get(ctx, owner, repo, number)
			if err != nil {
				return nil, fmt.Errorf("get issue: %w", err)
			}

			return formatIssue(issue), nil
		},
	)
}

// createIssueTool creates a new issue.
func (s *Skill) createIssueTool() skill.Tool {
	return skill.NewTool(
		"create_issue",
		"Create a new GitHub issue",
		map[string]skill.Parameter{
			"owner":     {Type: "string", Description: "Repository owner"},
			"repo":      {Type: "string", Description: "Repository name"},
			"title":     {Type: "string", Description: "Issue title", Required: true},
			"body":      {Type: "string", Description: "Issue body (markdown)"},
			"labels":    {Type: "string", Description: "Comma-separated list of label names"},
			"assignees": {Type: "string", Description: "Comma-separated list of assignee usernames"},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			owner, repo, err := s.getOwnerRepo(params)
			if err != nil {
				return nil, err
			}

			title := getString(params, "title", "")
			if title == "" {
				return nil, fmt.Errorf("title is required")
			}

			req := &gh.IssueRequest{
				Title: &title,
			}

			if body := getString(params, "body", ""); body != "" {
				req.Body = &body
			}

			if labels := getString(params, "labels", ""); labels != "" {
				labelList := strings.Split(labels, ",")
				req.Labels = &labelList
			}

			if assignees := getString(params, "assignees", ""); assignees != "" {
				assigneeList := strings.Split(assignees, ",")
				req.Assignees = &assigneeList
			}

			issue, _, err := s.client.Issues.Create(ctx, owner, repo, req)
			if err != nil {
				return nil, fmt.Errorf("create issue: %w", err)
			}

			return formatIssue(issue), nil
		},
	)
}

// updateIssueTool updates an existing issue.
func (s *Skill) updateIssueTool() skill.Tool {
	return skill.NewTool(
		"update_issue",
		"Update an existing GitHub issue",
		map[string]skill.Parameter{
			"owner":  {Type: "string", Description: "Repository owner"},
			"repo":   {Type: "string", Description: "Repository name"},
			"number": {Type: "integer", Description: "Issue number", Required: true},
			"title":  {Type: "string", Description: "New issue title"},
			"body":   {Type: "string", Description: "New issue body"},
			"state":  {Type: "string", Description: "Issue state", Enum: []any{"open", "closed"}},
			"labels": {Type: "string", Description: "Comma-separated list of label names (replaces existing)"},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			owner, repo, err := s.getOwnerRepo(params)
			if err != nil {
				return nil, err
			}

			number := getInt(params, "number", 0)
			if number == 0 {
				return nil, fmt.Errorf("issue number is required")
			}

			req := &gh.IssueRequest{}

			if title := getString(params, "title", ""); title != "" {
				req.Title = &title
			}
			if body := getString(params, "body", ""); body != "" {
				req.Body = &body
			}
			if state := getString(params, "state", ""); state != "" {
				req.State = &state
			}
			if labels := getString(params, "labels", ""); labels != "" {
				labelList := strings.Split(labels, ",")
				req.Labels = &labelList
			}

			issue, _, err := s.client.Issues.Edit(ctx, owner, repo, number, req)
			if err != nil {
				return nil, fmt.Errorf("update issue: %w", err)
			}

			return formatIssue(issue), nil
		},
	)
}

// addComment is a shared helper for adding comments to issues and PRs.
func (s *Skill) addComment(ctx context.Context, params map[string]any, errPrefix string) (any, error) {
	owner, repo, err := s.getOwnerRepo(params)
	if err != nil {
		return nil, err
	}

	number := getInt(params, "number", 0)
	body := getString(params, "body", "")

	if number == 0 || body == "" {
		return nil, fmt.Errorf("number and body are required")
	}

	comment, _, err := s.client.Issues.CreateComment(ctx, owner, repo, number, &gh.IssueComment{
		Body: &body,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}

	return map[string]any{
		"id":         comment.GetID(),
		"body":       comment.GetBody(),
		"user":       comment.GetUser().GetLogin(),
		"created_at": comment.GetCreatedAt().String(),
		"html_url":   comment.GetHTMLURL(),
	}, nil
}

// addIssueCommentTool adds a comment to an issue.
func (s *Skill) addIssueCommentTool() skill.Tool {
	return skill.NewTool(
		"add_issue_comment",
		"Add a comment to a GitHub issue",
		map[string]skill.Parameter{
			"owner":  {Type: "string", Description: "Repository owner"},
			"repo":   {Type: "string", Description: "Repository name"},
			"number": {Type: "integer", Description: "Issue number", Required: true},
			"body":   {Type: "string", Description: "Comment body (markdown)", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			return s.addComment(ctx, params, "add comment")
		},
	)
}

// listPullRequestsTool lists pull requests.
func (s *Skill) listPullRequestsTool() skill.Tool {
	return skill.NewTool(
		"list_pull_requests",
		"List pull requests in a GitHub repository",
		map[string]skill.Parameter{
			"owner":    {Type: "string", Description: "Repository owner"},
			"repo":     {Type: "string", Description: "Repository name"},
			"state":    {Type: "string", Description: "PR state", Enum: []any{"open", "closed", "all"}, Default: "open"},
			"base":     {Type: "string", Description: "Filter by base branch"},
			"head":     {Type: "string", Description: "Filter by head branch (user:branch)"},
			"per_page": {Type: "integer", Description: "Results per page (max 100)", Default: 30},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			owner, repo, err := s.getOwnerRepo(params)
			if err != nil {
				return nil, err
			}

			opts := &gh.PullRequestListOptions{
				State: getString(params, "state", "open"),
				Base:  getString(params, "base", ""),
				Head:  getString(params, "head", ""),
				ListOptions: gh.ListOptions{
					PerPage: getInt(params, "per_page", 30),
				},
			}

			prs, _, err := s.client.PullRequests.List(ctx, owner, repo, opts)
			if err != nil {
				return nil, fmt.Errorf("list pull requests: %w", err)
			}

			return formatPullRequests(prs), nil
		},
	)
}

// getPullRequestTool gets a specific pull request.
func (s *Skill) getPullRequestTool() skill.Tool {
	return skill.NewTool(
		"get_pull_request",
		"Get details of a specific GitHub pull request",
		map[string]skill.Parameter{
			"owner":  {Type: "string", Description: "Repository owner"},
			"repo":   {Type: "string", Description: "Repository name"},
			"number": {Type: "integer", Description: "PR number", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			owner, repo, err := s.getOwnerRepo(params)
			if err != nil {
				return nil, err
			}

			number := getInt(params, "number", 0)
			if number == 0 {
				return nil, fmt.Errorf("PR number is required")
			}

			pr, _, err := s.client.PullRequests.Get(ctx, owner, repo, number)
			if err != nil {
				return nil, fmt.Errorf("get pull request: %w", err)
			}

			return formatPullRequest(pr), nil
		},
	)
}

// addPullRequestCommentTool adds a comment to a pull request.
// Note: Uses Issues.CreateComment since PR comments use the same API.
func (s *Skill) addPullRequestCommentTool() skill.Tool {
	return skill.NewTool(
		"add_pull_request_comment",
		"Add a comment to a GitHub pull request",
		map[string]skill.Parameter{
			"owner":  {Type: "string", Description: "Repository owner"},
			"repo":   {Type: "string", Description: "Repository name"},
			"number": {Type: "integer", Description: "PR number", Required: true},
			"body":   {Type: "string", Description: "Comment body (markdown)", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			return s.addComment(ctx, params, "add PR comment")
		},
	)
}

// searchCodeTool searches code in repositories.
func (s *Skill) searchCodeTool() skill.Tool {
	return skill.NewTool(
		"search_code",
		"Search code across GitHub repositories",
		map[string]skill.Parameter{
			"query":    {Type: "string", Description: "Search query (supports GitHub code search syntax)", Required: true},
			"per_page": {Type: "integer", Description: "Results per page (max 100)", Default: 30},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			query := getString(params, "query", "")
			if query == "" {
				return nil, fmt.Errorf("query is required")
			}

			opts := &gh.SearchOptions{
				ListOptions: gh.ListOptions{
					PerPage: getInt(params, "per_page", 30),
				},
			}

			results, _, err := s.client.Search.Code(ctx, query, opts)
			if err != nil {
				return nil, fmt.Errorf("search code: %w", err)
			}

			return formatCodeResults(results), nil
		},
	)
}

// searchIssuesTool searches issues and pull requests.
func (s *Skill) searchIssuesTool() skill.Tool {
	return skill.NewTool(
		"search_issues",
		"Search issues and pull requests across GitHub",
		map[string]skill.Parameter{
			"query":    {Type: "string", Description: "Search query (supports GitHub issue search syntax)", Required: true},
			"per_page": {Type: "integer", Description: "Results per page (max 100)", Default: 30},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			query := getString(params, "query", "")
			if query == "" {
				return nil, fmt.Errorf("query is required")
			}

			opts := &gh.SearchOptions{
				ListOptions: gh.ListOptions{
					PerPage: getInt(params, "per_page", 30),
				},
			}

			results, _, err := s.client.Search.Issues(ctx, query, opts)
			if err != nil {
				return nil, fmt.Errorf("search issues: %w", err)
			}

			return formatIssueSearchResults(results), nil
		},
	)
}

// Helper functions

func getString(params map[string]any, key, defaultVal string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getInt(params map[string]any, key string, defaultVal int) int {
	switch v := params[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultVal
}

func formatIssues(issues []*gh.Issue) []map[string]any {
	result := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		// Skip pull requests (they appear in issue lists)
		if issue.PullRequestLinks != nil {
			continue
		}
		result = append(result, formatIssue(issue))
	}
	return result
}

func formatIssue(issue *gh.Issue) map[string]any {
	labels := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labels = append(labels, label.GetName())
	}

	assignees := make([]string, 0, len(issue.Assignees))
	for _, assignee := range issue.Assignees {
		assignees = append(assignees, assignee.GetLogin())
	}

	return map[string]any{
		"number":     issue.GetNumber(),
		"title":      issue.GetTitle(),
		"state":      issue.GetState(),
		"body":       issue.GetBody(),
		"user":       issue.GetUser().GetLogin(),
		"labels":     labels,
		"assignees":  assignees,
		"comments":   issue.GetComments(),
		"created_at": issue.GetCreatedAt().String(),
		"updated_at": issue.GetUpdatedAt().String(),
		"html_url":   issue.GetHTMLURL(),
	}
}

func formatPullRequests(prs []*gh.PullRequest) []map[string]any {
	result := make([]map[string]any, 0, len(prs))
	for _, pr := range prs {
		result = append(result, formatPullRequest(pr))
	}
	return result
}

func formatPullRequest(pr *gh.PullRequest) map[string]any {
	labels := make([]string, 0, len(pr.Labels))
	for _, label := range pr.Labels {
		labels = append(labels, label.GetName())
	}

	return map[string]any{
		"number":     pr.GetNumber(),
		"title":      pr.GetTitle(),
		"state":      pr.GetState(),
		"body":       pr.GetBody(),
		"user":       pr.GetUser().GetLogin(),
		"head":       pr.GetHead().GetRef(),
		"base":       pr.GetBase().GetRef(),
		"labels":     labels,
		"draft":      pr.GetDraft(),
		"mergeable":  pr.GetMergeable(),
		"merged":     pr.GetMerged(),
		"additions":  pr.GetAdditions(),
		"deletions":  pr.GetDeletions(),
		"commits":    pr.GetCommits(),
		"created_at": pr.GetCreatedAt().String(),
		"updated_at": pr.GetUpdatedAt().String(),
		"html_url":   pr.GetHTMLURL(),
	}
}

func formatCodeResults(results *gh.CodeSearchResult) map[string]any {
	items := make([]map[string]any, 0, len(results.CodeResults))
	for _, item := range results.CodeResults {
		items = append(items, map[string]any{
			"name":       item.GetName(),
			"path":       item.GetPath(),
			"sha":        item.GetSHA(),
			"html_url":   item.GetHTMLURL(),
			"repository": item.GetRepository().GetFullName(),
		})
	}

	return map[string]any{
		"total_count": results.GetTotal(),
		"items":       items,
	}
}

func formatIssueSearchResults(results *gh.IssuesSearchResult) map[string]any {
	items := make([]map[string]any, 0, len(results.Issues))
	for _, issue := range results.Issues {
		items = append(items, map[string]any{
			"number":     issue.GetNumber(),
			"title":      issue.GetTitle(),
			"state":      issue.GetState(),
			"user":       issue.GetUser().GetLogin(),
			"repository": issue.GetRepositoryURL(),
			"html_url":   issue.GetHTMLURL(),
			"is_pr":      issue.IsPullRequest(),
			"created_at": issue.GetCreatedAt().String(),
		})
	}

	return map[string]any{
		"total_count": results.GetTotal(),
		"items":       items,
	}
}
