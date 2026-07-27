// Package omnidevx collects developer-experience events from AWS Kiro's local
// stores for the OmniDevX domain (github.com/plexusone/omnidevx-core).
//
// Kiro persists CLI conversations in a local SQLite database and community
// usage trackers may archive snapshots under ~/.kiro_sessions. Both are local,
// internal formats rather than public APIs. The collector parses defensively,
// emits diagnostics for schema drift, and stores only metadata and estimated
// token counts in canonical OmniDevX events.
//
// This collector lives in omni-aws rather than omnidevx-core/providers because
// reading the live Kiro database requires a SQLite driver.
package omnidevx
