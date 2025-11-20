// Package main implements the bd command-line tool for distributed issue planning.
//
// bd is a Git-native issue tracker that stores all data in a SQLite database
// (or git notes in minimal implementations). Commands operate in two modes:
//
//   - Direct mode: opens the database directly
//   - Daemon mode: connects to a background daemon via RPC for coordinated access
//
// The daemon is started automatically on first invocation and manages concurrent access,
// background flushes, and incremental exports. Commands try the daemon first and fall
// back to direct mode if the daemon is unavailable.
//
// Core operations include: create, update, list, ready (actionable items), delete, close,
// and dependency management. Additional features include compaction, import/export, and
// hierarchical issue support.
//
// Key types:
//   - DaemonStatus: describes daemon connection state for the current command
//
// Configuration via flags, environment variables (BD_ prefix), or config.yaml.
// Database location determined by: --db flag, BD_DB env, or walking up to .beads directory.
package main
