# Git Notes Storage Backend Tests

Scripttest tests for the git notes storage backend using rsc.io/script pattern.

## Test Files

### Core Operations
- **gitnotes_init.txt** - Initialize beads with git notes backend
  - Verifies git notes refs creation (refs/notes/beads/issues)
  - Checks metadata files (.beads/metadata.json, .beads/config.json)
  - Validates git repository structure

- **gitnotes_create.txt** - Create issues with git notes storage
  - Tests single and multiple issue creation
  - Verifies issues stored in git notes namespace
  - Validates JSON encoding in notes

- **gitnotes_list.txt** - List issues from git notes
  - Tests listing all issues
  - Verifies filtering by status, priority, type
  - Validates open vs closed issue visibility

- **gitnotes_show.txt** - Show issue details from git notes
  - Tests retrieving full issue metadata
  - Verifies JSON decoding from git notes
  - Validates description and custom fields

- **gitnotes_update.txt** - Update issues in git notes
  - Tests updating status, priority, type, description
  - Verifies atomic updates
  - Validates git notes modification

- **gitnotes_close.txt** - Close issues in git notes
  - Tests closing with/without reason
  - Verifies closed_at timestamp
  - Validates status changes in storage

### Advanced Features
- **gitnotes_dependencies.txt** - Dependency management
  - Tests adding dependencies between issues
  - Verifies dependency storage (refs/notes/beads/graph)
  - Tests removing dependencies

- **gitnotes_sync.txt** - Git notes synchronization
  - Tests push/pull of notes to remote
  - Verifies multi-repository collaboration
  - Validates note merging

- **gitnotes_transactions.txt** - Transaction semantics
  - Tests atomic multi-operation commits
  - Verifies git update-ref usage
  - Validates reflog for audit trail

## Running Tests

### Run all scripttest tests (including git notes):
```bash
cd cmd/bd
go test -tags scripttests -v
```

### Run only git notes tests:
```bash
cd cmd/bd
go test -tags scripttests -v -run 'TestScripts/gitnotes_'
```

### Run specific test:
```bash
cd cmd/bd
go test -tags scripttests -v -run 'TestScripts/gitnotes_init'
```

## Test Pattern

Each test follows the rsc.io/script pattern:

```
# Comment describing test
exec <command>      # Execute shell command
bd <args>           # Run bd command
stdout 'text'       # Assert stdout contains text
! stdout 'text'     # Assert stdout does NOT contain text
exists <file>       # Assert file exists
grep 'pattern' file # Assert file contains pattern
cp src dst          # Copy file
```

## Test Requirements

1. **Git repository**: All tests initialize a git repo with initial commit
2. **Git config**: Sets user.name and user.email for commits
3. **--backend gitnotes**: Uses git notes backend flag (when implemented)
4. **Unix shell**: Tests use `sh -c` for complex operations (Unix only)

## Implementation Status

These tests are written for the **future** git notes backend. Current status:
- Storage interface: ✅ Defined (internal/storage/storage.go)
- Git notes backend: ⏳ Not yet implemented (beads-72 epic)
- Tests: ✅ Ready (will SKIP until backend implemented)

## Design References

See beads-72 epic for architecture:
- Namespace: refs/notes/beads/issues, refs/notes/beads/graph
- Format: JSON per note
- IDs: ULID-based with optional human-readable aliases
- Transactions: git update-ref --stdin for atomicity
- Performance: Target ~1-1.5ms per issue

## Test Coverage

| Feature | Test File | Coverage |
|---------|-----------|----------|
| Init | gitnotes_init.txt | Backend initialization |
| Create | gitnotes_create.txt | Issue creation, JSON encoding |
| Read | gitnotes_show.txt | Issue retrieval, decoding |
| Update | gitnotes_update.txt | Field updates, atomicity |
| Delete | gitnotes_close.txt | Status changes, timestamps |
| List | gitnotes_list.txt | Filtering, sorting |
| Dependencies | gitnotes_dependencies.txt | Graph storage |
| Sync | gitnotes_sync.txt | Remote push/pull |
| Transactions | gitnotes_transactions.txt | Atomicity, reflog |

Total: 9 test files covering core CRUD + advanced features
