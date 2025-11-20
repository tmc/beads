// Package rpc implements JSON-RPC over Unix sockets for bd daemon communication.
//
// A bd daemon listens on a socket per workspace. The cli connects to the daemon,
// sends JSON-RPC requests, and receives responses. This serializes access to the
// database and enables background operations (export, compaction) without blocking
// the cli.
//
// Protocol:
//   - Request: operation name, typed arguments (args struct), request ID, working directory
//   - Response: success boolean, typed data (JSON-marshaled), or error message
//   - Operations: 30+ covering all issue mutations, queries, and admin tasks
//
// Client usage:
//   1. TryConnect or TryConnectWithTimeout to the daemon socket
//   2. Call Execute methods (Create, Update, List, etc.) or ExecuteWithCwd for custom ops
//   3. Close the connection when done
//
// Key types:
//   - Client: manages a connection to the daemon
//   - Request/Response: wire protocol
//   - 40+ arg/response types for specific operations (CreateArgs, ListResponse, etc.)
//   - MetricsSnapshot: daemon performance metrics
//   - HealthResponse: daemon health status
//
// Key client methods:
//   - Execute/ExecuteWithCwd: send operation to daemon
//   - Ping: verify daemon liveness
//   - Status/Health: daemon state
//   - Create, Update, CloseIssue, List, Show, Ready, Stale, Stats: core operations
//   - Shutdown: graceful daemon termination
//
// Connection management:
//   - Timeouts: configurable via SetTimeout
//   - Database path: set via SetDatabasePath for validation
//   - Automatic retries on transient failures (self-healing)
package rpc
