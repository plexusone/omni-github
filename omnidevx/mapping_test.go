package omnidevx

import (
	"context"
	"testing"
	"time"

	"github.com/grokify/gogithub/profile"
	core "github.com/plexusone/omnidevx-core"
)

func fixtureProfile() *profile.UserProfile {
	return &profile.UserProfile{
		Username:           "testuser",
		TotalCommits:       1754,
		TotalPRs:           12,
		TotalIssues:        5,
		TotalReviews:       3,
		TotalReposCreated:  4,
		ReposContributedTo: 2,
		TotalAdditions:     1716920,
		TotalDeletions:     251531,
		RepoStats: []profile.RepoContribution{
			{Owner: "plexusone", Name: "omnidevx-core", FullName: "plexusone/omnidevx-core",
				Commits: 900, Additions: 1000000, Deletions: 100000},
			{Owner: "grokify", Name: "private-repo", FullName: "grokify/private-repo",
				IsPrivate: true, Commits: 854, Additions: 716920, Deletions: 151531},
		},
		Calendar: &profile.ContributionCalendar{
			TotalContributions: 60,
			Weeks: []profile.CalendarWeek{{
				StartDate: time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC),
				Days: [7]profile.CalendarDay{
					{Date: time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC), ContributionCount: 0},
					{Date: time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC), ContributionCount: 25},
					{Date: time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC), ContributionCount: 35},
				},
			}},
		},
	}
}

func request() core.CollectRequest {
	return core.CollectRequest{
		Subject: core.SubjectRef{PersonID: "person:test"},
		Period: core.Period{
			Start: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestProfileEvents(t *testing.T) {
	req := request()
	events := profileEvents(fixtureProfile(), req)

	counts := map[core.EventType]int{}
	for _, e := range events {
		counts[e.Type]++
		if e.Subject.PersonID != "person:test" {
			t.Errorf("subject not stamped on %s", e.ID)
		}
		if e.Provenance.CollectionMode != core.ModeAPI {
			t.Errorf("mode: got %s on %s", e.Provenance.CollectionMode, e.ID)
		}
		if !req.Period.Contains(e.Timestamp) {
			t.Errorf("event %s timestamp %s outside period", e.ID, e.Timestamp)
		}
	}
	if counts[core.EventProfileSnapshot] != 1 {
		t.Errorf("profile snapshots: got %d, want 1", counts[core.EventProfileSnapshot])
	}
	if counts[core.EventContributionSnapshot] != 2 {
		t.Errorf("repo snapshots: got %d, want 2", counts[core.EventContributionSnapshot])
	}
	if counts[core.EventContributionRecorded] != 2 { // zero-count day skipped
		t.Errorf("daily contributions: got %d, want 2", counts[core.EventContributionRecorded])
	}

	for _, e := range events {
		switch e.Type {
		case core.EventProfileSnapshot:
			if e.Attributes[core.AttrCommits] != 1754 || e.Attributes[core.AttrInsertions] != 1716920 {
				t.Errorf("profile snapshot attrs: %+v", e.Attributes)
			}
		case core.EventContributionSnapshot:
			if e.Context.Repository == "github.com/grokify/private-repo" {
				if e.Attributes[core.AttrPrivateRepo] != true {
					t.Errorf("private flag missing: %+v", e.Attributes)
				}
			}
		}
	}
}

func TestProfileEventsDeterministicIDs(t *testing.T) {
	req := request()
	a := profileEvents(fixtureProfile(), req)
	b := profileEvents(fixtureProfile(), req)
	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Errorf("nondeterministic ID at %d: %s vs %s", i, a[i].ID, b[i].ID)
		}
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(Config{Token: "x"}); err == nil {
		t.Error("expected error for missing username")
	}
	if _, err := New(Config{Username: "x"}); err == nil {
		t.Error("expected error for missing token and clients")
	}
	c, err := New(Config{Username: "x", Token: "fake"})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Source(); got.Provider != "github" || got.Product != "github" {
		t.Errorf("source: %+v", got)
	}
}

func TestCollectRequiresBoundedPeriod(t *testing.T) {
	c, err := New(Config{Username: "x", Token: "fake"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Collect(context.Background(), core.CollectRequest{}); err == nil {
		t.Error("expected error for unbounded period")
	}
}
