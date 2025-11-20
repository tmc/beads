# bd Scripttest Suite

This directory contains integration tests for bd using the `rsc.io/script` and `rsc.io/script/scripttest` packages. Tests are written as simple text files with shell-like syntax and embedded fixtures.

## Test Framework

Each `.txt` file is a self-contained test scenario following the txtar script pattern:

```
# Comment describing the test
command args
stdout 'expected output'
stderr 'error pattern'
! command args  # command should fail
```

Key directives:
- `exec sh -c 'shell command'` - Execute shell command
- `stdout 'pattern'` - Assert output contains regex
- `stderr 'pattern'` - Assert error contains regex
- `! command` - Assert command fails (non-zero exit)
- `exists file` - Assert file exists
- `cp source dest` - Copy file
- `-- filename --` - Embedded file fixture

Tests run with the `bd` binary built from the current source.

## Test Categories

### Dependency Management (4 tests)

**dep_tree_complex.txt** (51 lines)
- Creates multi-level dependency graph (3 levels deep)
- Tests `bd dep tree` for root, intermediate, and leaf nodes
- Verifies traversal of complex dependency chains

**dep_cycles.txt** (32 lines)
- Creates circular dependencies (A→B→C→A)
- Tests cycle detection/handling
- Verifies graceful error handling or recovery

**dep_remove_advanced.txt** (48 lines)
- Creates issue with 3 dependencies
- Tests removing individual dependency from set
- Verifies remaining dependencies are intact

**dep_add.txt** (basic, existing)
- Creates two issues
- Adds dependency between them
- Verifies dependency appears in show output

### Ready Filtering (2 tests)

**ready_filtering.txt** (34 lines)
- Creates independent, blocked, and completed issues
- Tests `bd ready` command filters out blocked and closed items
- Verifies actionable items are correctly identified

**ready_priority_filter.txt** (31 lines)
- Creates issues with different priorities (1=high, 2=medium, 3=low)
- Tests listing ready items with varying priorities
- Verifies priority information is preserved

### Compaction (2 tests)

**compact_basic.txt** (46 lines)
- Creates 3 issues with multiple updates and closes
- Runs `bd compact` to reduce database size
- Verifies data integrity after compaction
- Checks all issues are still accessible

**compact_with_deps.txt** (44 lines)
- Creates 3-level dependency chain
- Runs `bd compact`
- Verifies dependency relationships survive compaction
- Tests `bd dep tree` still works correctly

### Import/Export (4 tests)

**import_export_roundtrip.txt** (46 lines)
- Creates issues with descriptions, labels, and dependencies
- Exports to JSONL
- Imports into new database with different prefix
- Verifies data integrity across round-trip

**import_collision_handling.txt** (34 lines)
- Exports current state
- Creates modified export with duplicate IDs
- Tests collision detection in `bd import`
- Verifies dry-run mode works
- Tests actual import with collision

**export_with_labels_deps.txt** (44 lines)
- Creates issues with labels and dependencies
- Exports to JSONL
- Verifies labels and dependencies are in exported file
- Imports into new database to verify preservation

**import.txt** (basic, existing)
- Imports issue from embedded JSONL file
- Verifies imported issue appears in database

### Error Handling (2 tests)

**error_invalid_id.txt** (33 lines)
- Attempts operations on non-existent issues
- Tests: show, update, close, dep add with invalid IDs
- Verifies appropriate error messages
- Confirms operations with one invalid ID fail

**error_missing_required.txt** (36 lines)
- Tests missing required fields (empty title, invalid priority)
- Tests invalid issue type
- Tests missing command arguments
- Verifies graceful error messages

### Edge Cases (3 tests)

**edge_case_empty_db.txt** (32 lines)
- Tests operations on empty database
- Tests: list, ready, stats, compact on empty DB
- Verifies graceful handling of no-data scenarios
- Confirms first issue creation still works

**edge_case_special_chars.txt** (35 lines)
- Creates issues with quotes, backslashes, newlines
- Tests very long titles (150+ characters)
- Tests Unicode characters (émojis, non-ASCII)
- Verifies special characters are preserved round-trip

**edge_case_concurrent_updates.txt** (57 lines)
- Performs 3 rapid updates to same issue
- Verifies last-write-wins semantics
- Tests rapid label additions (with duplicates)
- Tests rapid dependency additions
- Verifies idempotency of operations

## Running Tests

To run the scripttest suite:

```bash
# Run with scripttest build tag enabled
go test -tags scripttests ./cmd/bd

# Run specific test file
go test -tags scripttests ./cmd/bd -run TestScripts -v

# From cmd/bd directory
go test -tags scripttests -v
```

Tests build the `bd` binary in a temp directory and set up an isolated environment for each test file. Each test gets a fresh temporary directory to avoid conflicts.

## Test Patterns

### Capturing Output
Tests often capture output to files for later use:
```
bd create 'Issue title'
cp stdout issue_output.txt
exec sh -c 'grep -oE "task-[a-z0-9]+" issue_output.txt > issue_id.txt'
```

Then reference the captured ID:
```
exec sh -c 'bd show $(cat issue_id.txt)'
```

### Assertions
- **stdout pattern** - Check output matches regex
- **stderr pattern** - Check error matches regex
- **! command** - Assert command fails
- **exists file** - Assert file was created

### Embedded Files
Files can be embedded in test scripts:
```
-- import.jsonl --
{"id":"task-99","title":"Imported issue",...}
```

## Adding New Tests

1. Create new `.txt` file in `testdata/`
2. Start with `bd init --prefix <prefix>`
3. Write test steps using `bd` commands
4. Add assertions with `stdout`, `stderr`, or `!`
5. Test with: `go test -tags scripttests -v`

Keep tests focused and self-contained. Each test file should be runnable independently in a fresh directory.

## Maintenance

When bd behavior changes:
1. Update affected test expectations
2. Add new tests for new features
3. Keep error message patterns flexible (use regex alternation)
4. Document edge cases and gotchas

## Known Limitations

- Tests use `sh -c` which requires Unix shell (skip on Windows)
- Timeout for each command is 2 seconds (can be slow on CI)
- File operations use shell; paths must be quoted if they contain spaces
- stderr assertions match patterns (not exact text)
