// Package merge performs 3-way merges of JSONL issue files.
//
// When concurrent JSONL edits occur (e.g., git merge), this package resolves
// conflicts by comparing base, left, and right versions of each issue.
// Conflicts are marked with conflict markers in the output for manual resolution.
//
// This is the default merge driver for .jsonl files (configured in .gitattributes).
//
// Key types:
//   - Issue: a beads issue with all fields (id, title, description, status, etc.)
//   - Dependency: a directed dependency between two issues
//   - IssueKey: uniquely identifies an issue by id, creation time, and creator
//
// Key function:
//   - Merge3Way: performs 3-way merge of JSONL files
//
// Merge algorithm:
//   1. Parse base, left, right JSONL files as Issue arrays
//   2. For each issue, match by IssueKey (id + created_at + created_by)
//   3. Compare base vs left vs right field-by-field
//   4. Apply 3-way merge logic:
//      - If both sides changed the same field identically: use the value
//      - If only one side changed: use the changed value
//      - If both sides changed differently: mark conflict and use both values
//   5. Handle dependencies: merge lists of dependencies from both sides
//   6. Write resolved issues to output, conflicts marked with conflict markers
//
// Output format:
//   - Clean merge: JSONL output ready to commit
//   - With conflicts: JSONL with conflict markers (<<<<<<<, =======, >>>>>>>)
//     indicating base, left, and right versions of disputed fields
//
// Vendored from https://github.com/neongreen/mono (beads-merge) with permission.
package merge
