// Package daemon provides discovery and lifecycle management for bd background processes.
//
// bd runs a background daemon per workspace to serialize concurrent access to the database
// and batch incremental operations (export, compaction). The daemon is managed transparently:
// commands try to connect and auto-start if configured.
//
// This package handles:
//   - Daemon discovery: finding existing daemons by socket path or workspace path
//   - Daemon termination: graceful shutdown and forced kill
//   - Platform-specific process control: Unix signals, Windows service APIs, WASM fallback
//
// Key types:
//   - DaemonInfo: metadata about a discovered daemon (PID, socket, workspace, version, health)
//   - KillAllFailure: individual failure record in batch kill operation
//   - KillAllResults: summary of multi-daemon termination
//
// Key functions:
//   - DiscoverDaemons: find all running bd daemons
//   - FindDaemonByWorkspace: locate daemon for a specific workspace
//   - StopDaemon: gracefully stop a daemon
//   - KillAllDaemons: terminate multiple daemons with fallback to force kill
//   - CleanupStaleSockets: remove stale socket files for dead daemons
//
// Usage example:
//
//	info, err := daemon.FindDaemonByWorkspace("/path/to/workspace")
//	if err == nil {
//		daemon.StopDaemon(info)
//	}
package daemon
