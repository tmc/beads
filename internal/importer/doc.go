// Package importer handles parsing and merging JSONL issue data into the database.
//
// Beads exports issues to JSONL for version control. The importer detects when
// the JSONL has changed (user edits or merges) and applies those changes back
// into the database. It handles collisions, prefix mismatches, missing parents,
// and idempotent updates.
//
// Main use case: auto-import triggered when JSONL file is modified externally.
//
// Key types:
//   - Options: import configuration (dry-run, skip-update, strict, renaming)
//   - Result: statistics about the import (created, updated, skipped, collisions, mappings)
//   - OrphanHandling: policy for issues with missing parents (strict, resurrect, skip, allow)
//
// Key functions:
//   - ImportIssues: main entry point; detects collisions, updates, and applies changes
//   - SortByDepth: order issues for hierarchical import (parents before children)
//   - GroupByDepth: partition issues by parent depth
//   - IssueDataChanged: detect field changes to avoid redundant updates
//   - RenameImportedIssuePrefixes: bulk rename imported issues to match database prefix
//
// Import flow:
//   1. Read external JSONL file
//   2. Detect collisions (same ID created twice)
//   3. Detect prefix mismatches (imported issues don't match database prefix)
//   4. Detect updates (existing issues with changed fields)
//   5. Upsert issues (create or update)
//   6. Import dependencies and labels
//   7. Handle errors: orphans, duplicates, strict validation
//
// Result includes detailed mapping of remapped IDs and collision details for debugging.
package importer
