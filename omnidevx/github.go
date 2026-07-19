package omnidevx

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v88/github"
	"github.com/grokify/gogithub/profile"
	core "github.com/plexusone/omnidevx-core"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var source = core.Source{
	Provider: "github",
	Product:  "github",
}

// Config holds configuration for the GitHub DevX collector.
type Config struct {
	// Token is a GitHub API token. Required unless both clients are set.
	Token string

	// Username is the GitHub account whose contributions are collected.
	// Required.
	Username string

	// RESTClient overrides the token-derived REST client (for testing or
	// custom transports).
	RESTClient *github.Client

	// GraphQLClient overrides the token-derived GraphQL client.
	GraphQLClient *githubv4.Client
}

// Collector reads GitHub contribution data via gogithub/profile.
type Collector struct {
	cfg  Config
	rest *github.Client
	gql  *githubv4.Client
}

var _ core.Collector = (*Collector)(nil)

// New returns a Collector for the given config.
func New(cfg Config) (*Collector, error) {
	if cfg.Username == "" {
		return nil, fmt.Errorf("github: username is required")
	}
	rest, gql := cfg.RESTClient, cfg.GraphQLClient
	if rest == nil || gql == nil {
		if cfg.Token == "" {
			return nil, fmt.Errorf("github: token is required when clients are not provided")
		}
		if rest == nil {
			var err error
			rest, err = github.NewClient(github.WithAuthToken(cfg.Token))
			if err != nil {
				return nil, fmt.Errorf("github: build REST client: %w", err)
			}
		}
		if gql == nil {
			gql = githubv4.NewClient(oauth2.NewClient(context.Background(),
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})))
		}
	}
	return &Collector{cfg: cfg, rest: rest, gql: gql}, nil
}

// Source implements omnidevx.Collector.
func (c *Collector) Source() core.Source { return source }

// Collect implements omnidevx.Collector. The request period must be bounded:
// GitHub's contribution query needs an explicit range, and unbounded scans
// of account history are rarely intended.
func (c *Collector) Collect(ctx context.Context, req core.CollectRequest) (*core.CollectionResult, error) {
	if req.Period.Start.IsZero() || req.Period.End.IsZero() {
		return nil, fmt.Errorf("github: a bounded period (start and end) is required")
	}

	up, err := profile.GetUserProfile(ctx, c.rest, c.gql, c.cfg.Username,
		req.Period.Start, req.Period.End, profile.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("github: fetch profile for %s: %w", c.cfg.Username, err)
	}

	return &core.CollectionResult{
		Source:      source,
		Subject:     req.Subject,
		Period:      req.Period,
		Events:      profileEvents(up, req),
		CollectedAt: time.Now().UTC(),
	}, nil
}
