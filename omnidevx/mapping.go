package omnidevx

import (
	"fmt"
	"time"

	"github.com/grokify/gogithub/profile"
	core "github.com/plexusone/omnidevx-core"
)

// snapshotConfidence reflects that platform-reported aggregates are
// authoritative for what GitHub counts, though GitHub's counting rules
// (default branches, squash merges, private restrictions) differ from local
// git history.
const snapshotConfidence = 1.0

// profileEvents maps a fetched UserProfile to canonical events. Snapshot
// timestamps sit one second inside the period end so half-open period reads
// include them.
func profileEvents(up *profile.UserProfile, req core.CollectRequest) []core.Event {
	periodKey := req.Period.Start.UTC().Format("20060102") + "-" + req.Period.End.UTC().Format("20060102")
	snapshotTime := req.Period.End.Add(-time.Second) // just inside the half-open period

	events := []core.Event{{
		ID:        "github:profile:" + up.Username + ":" + periodKey,
		Type:      core.EventProfileSnapshot,
		Timestamp: snapshotTime,
		Subject:   req.Subject,
		Source:    source,
		Attributes: map[string]any{
			core.AttrCommits:       up.TotalCommits,
			core.AttrPullRequests:  up.TotalPRs,
			core.AttrIssues:        up.TotalIssues,
			core.AttrReviews:       up.TotalReviews,
			core.AttrReposCreated:  up.TotalReposCreated,
			core.AttrRepositories:  up.ReposContributedTo,
			core.AttrInsertions:    up.TotalAdditions,
			core.AttrDeletions:     up.TotalDeletions,
			core.AttrContributions: up.RestrictedContributions,
		},
		Provenance: apiProvenance(),
	}}

	for _, repo := range up.RepoStats {
		events = append(events, core.Event{
			ID:        "github:reposnap:" + up.Username + ":" + repo.FullName + ":" + periodKey,
			Type:      core.EventContributionSnapshot,
			Timestamp: snapshotTime,
			Subject:   req.Subject,
			Source:    source,
			Context: core.EventContext{
				Repository: "github.com/" + repo.FullName,
			},
			Attributes: map[string]any{
				core.AttrCommits:     repo.Commits,
				core.AttrInsertions:  repo.Additions,
				core.AttrDeletions:   repo.Deletions,
				core.AttrReleases:    repo.Releases,
				core.AttrPrivateRepo: repo.IsPrivate,
			},
			Provenance: apiProvenance(),
		})
	}

	if up.Calendar != nil {
		for _, week := range up.Calendar.Weeks {
			for _, day := range week.Days {
				if day.ContributionCount == 0 || day.Date.IsZero() || !req.Period.Contains(day.Date) {
					continue
				}
				events = append(events, core.Event{
					ID: fmt.Sprintf("github:contrib:%s:%s",
						up.Username, day.Date.UTC().Format("2006-01-02")),
					Type:      core.EventContributionRecorded,
					Timestamp: day.Date,
					Subject:   req.Subject,
					Source:    source,
					Attributes: map[string]any{
						core.AttrContributions: day.ContributionCount,
					},
					Provenance: apiProvenance(),
				})
			}
		}
	}
	return events
}

func apiProvenance() core.Provenance {
	return core.Provenance{
		CollectionMode: core.ModeAPI,
		Confidence:     snapshotConfidence,
	}
}
