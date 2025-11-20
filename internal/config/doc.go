// Package config provides centralized configuration management for bd.
//
// Configuration is loaded from (in order of precedence):
//   - Environment variables (BD_ prefix, e.g., BD_JSON, BD_NO_DAEMON, BD_ACTOR)
//   - Project config: .beads/config.yaml (discovered by walking up directory tree)
//   - User config: ~/.config/bd/config.yaml or ~/.beads/config.yaml
//
// The package uses viper as its backend and exposes a singleton accessed via
// module-level functions. Initialize() must be called once at application startup.
//
// Key functions:
//   - Initialize: set up the configuration singleton
//   - GetString, GetBool, GetInt, GetDuration: retrieve typed config values
//   - Set: update a config value in memory
//   - GetMultiRepoConfig: parse multi-repo routing configuration
//
// Common configuration keys:
//   - json: output as JSON
//   - no-daemon: disable daemon mode
//   - no-auto-flush: disable incremental JSONL export
//   - no-auto-import: disable auto-import from JSONL
//   - db: explicit database path
//   - actor: author name for mutations
//   - flush-debounce: delay for batching changes (default 30s)
//   - auto-start-daemon: auto-start daemon on connect failure (default true)
//   - routing.mode: multi-repo routing strategy (auto, explicit)
package config
