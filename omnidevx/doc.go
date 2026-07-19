// Package omnidevx collects developer-experience events from GitHub for the
// OmniDevX domain (github.com/plexusone/omnidevx-core), following the
// omni-github per-domain package convention alongside omnistorage and
// omniskill.
//
// Contribution data comes from github.com/grokify/gogithub/profile (GraphQL
// contribution calendar and per-repository commit statistics), normalized
// into canonical devx.* events:
//
//   - devx.profile.snapshot      — whole-profile period totals
//   - devx.contribution.snapshot — per-repository period totals
//   - devx.contribution.recorded — daily contribution-calendar counts
//
// GitHub-reported contributions overlap with locally-collected git commits
// (providers/git in omnidevx-core); the aggregation layer keeps the two
// sources separate via bySource metrics rather than summing them.
//
// Events carry api-mode provenance: they are observed from the live GitHub
// API rather than reconstructed from local history.
package omnidevx
