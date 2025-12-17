# Veil — Architecture Overview

This document defines the unified architecture for Veil and describes how `codex` becomes the single source of truth, how plugins connect, and how server/CLI/UI components interact with the core.

## High-level goals
- Make `codex` the authoritative API for all knowledge operations (CRUD, branching, merging, querying).
- Provide a small, well-documented set of interfaces for storage and plugins.
- Keep the system pluggable: storage backends, transport (git/ipfs), and UI can be swapped without changing core logic.

## Core components

**Codex (pkg/codex)**
- Responsibilities:
  - Content-addressed storage of objects and commits
  - Branch and ref management (`refs/heads/*` semantics)
  - Commit creation, listing, diffing, and three-way merge with URN-based conflict resolution
  - High-level APIs for querying nodes and their relationships
- Exposed interfaces:
  - `Storage` — abstract backend used by codex (see `pkg/codex/codex.go`)
  - `Repository` — wrapper providing common repository operations

**Storage Backends (pkg/core, pkg/codex/storage)**
- Responsibilities:
  - Implement `Storage` interface (PutObjectStream/GetObjectStream, PutCommit/GetCommit, refs)
  - Efficient streaming for large blobs
  - Filesystem-backed reference implementation in `pkg/core/fs_storage.go` and `pkg/codex/storage/fs`

**Plugin System (pkg/plugins)**
- Responsibilities:
  - Define a stable plugin API and lifecycle (`Init`, `AttachCodex`, `Start`, `Stop`, hooks)
  - Provide adapters: `git_plugin`, `ipfs_plugin`, `media_plugin`, `reminder_plugin`, etc.
  - Plugins operate through the `codex` API rather than directly manipulating storage where possible.

**CLI (cmd/codex)**
- Responsibilities:
  - User-facing CLI commands (`init`, `add`, `commit`, `push`, `server`, `status`, `annotate`)
  - Use `codex.Repository` and plugin interfaces for operations

**Server & API**
- Responsibilities:
  - Expose REST, gRPC, and WebSocket gateways for codex operations
  - Provide authentication and access-control mechanisms
  - Manage multi-user workspace and repository lifecycle

**Web UI (web/ and cmd/codex/static)**
- Responsibilities:
  - Present codex data and graph visualizations
  - Provide UX for branching, merging, publishing, and media management

## Integration points and flow

1. CLI/HTTP request arrives -> `codex` API call (Repository methods) -> `Storage` backend performs persistence.
2. Plugins register with the core plugin manager at startup and receive a handle to `codex.Repository` (or specific interfaces) via `AttachCodex`.
3. Plugins can subscribe to commit hooks or provide commands that trigger codex actions (e.g., `git_plugin` can create commits from a local git repository and push codex objects to remote storage).

## Refactor checklist (practical steps)

- Ensure `Storage` supports full streaming methods and deprecate legacy non-streaming methods in internal code paths.
- Move all direct storage access in plugins and CLI to use `Repository` methods.
- Implement a plugin manager in `pkg/plugins/plugins.go` that registers plugins, wires lifecycle, and exposes runtime plugin metadata.
- Add REST/gRPC adapters in `cmd/server` (or `cmd/codex/cmd_server.go`) that call `Repository` APIs.
- Add authentication middleware and multi-tenant scoping around repositories in server mode.

## Data model notes

- Objects are content-addressed; use SHA256 of canonical JSON (as implemented by `computeCommitHash`).
- Commits reference object hashes and parents. Objects that represent entities should contain a stable `urn` field used for merging.

## Next steps and priorities

1. Audit code to find all places that bypass `codex.Repository` and update them to use the repository API. (See `docs/COMPONENTS.md` for file map.)
2. Implement a `pkg/plugins/manager.go` to centralize plugin lifecycle and dependency injection.
3. Harden `pkg/codex/Storage` implementations for streaming large objects.
4. Add REST/gRPC endpoints and integrate auth.
# Veil Architecture (Draft)

This document defines the unified core architecture of Veil and the immediate migration/refactor plan so that `Codex` becomes the single source of truth and plugins interact cleanly.

## Goals
- Make `pkg/codex` the canonical knowledge graph implementation exposing a stable interface for CRUD, branching, merging, and object storage.
- Provide pluggable storage backends that support streaming large/binary objects and chunking.
- Standardize plugin API and lifecycle so plugins operate only through safe, documented interfaces into Codex and services.
- Centralize server APIs and route registration with clear handler responsibilities.

## High-level Components
- Codex
  - Repository: high-level API for commits, branches, refs, history, diff, snapshots.
  - Objects: arbitrary blobs with metadata (content-type, size, checksum, encoding). Support streaming and chunk reads.
  - Indexing & Query: a query API for listing, prefix search, and simple graph queries.
- Storage interface
  - Storage.PutObject(reader) -> objectHash
  - Storage.GetObject(hash) -> reader
  - Storage.PutCommit(commit)
  - Storage.GetCommit(hash)
  - Implementations: FSStorage (existing), S3Storage (future), Content-Addressed CAS (future).
- Plugin system
  - Plugin interface (lifecycle): Init(db *sql.DB, cfg map[string]interface{}), Start(), Stop(), HandleOperation(op PluginOp)
  - Plugins interact with Codex through explicit APIs; they should not access internal storage directly.
- Server/API
  - All APIs live under `/api/` and are registered in `server/` package. `main.go` only wires up server config and plugin loading.
  - HTTP handlers return JSON, use consistent error model, and are tested.
- UI/CLI
  - Keep `web/` and `codex-universalis/` as static assets; update to call unified APIs.

## Data models (core)
- Object
  - hash: string
  - type: string (application/json, image/png, etc.)
  - size: int64
  - metadata: map[string]interface{}
- Commit
  - hash: string
  - parents: []string
  - author: string
  - timestamp: time.Time
  - message: string
  - objects: []string
- Node (app-level content)
  - id: string
  - path: string
  - content: reference to Object or blob in codex
  - metadata, tags, references, backlinks

## Migration & Refactor Plan (short-term)
1. Consolidate `pkg/core` concepts into `pkg/codex` and export stable interfaces.
2. Add `Storage` interface and update `pkg/codex/storage/fs` to implement streaming read/write methods.
3. Replace direct file JSON assumptions with streaming object storage; keep backward compatibility adapter for `.codex/objects/*.json`.
4. Add `plugins.Plugin` interface in `pkg/plugins` and refactor existing plugins to implement it.
5. Move route registration to `server/` package and add tests for all handlers.

## Acceptance criteria
- `pkg/codex` exposes the documented interfaces and compiles without duplication.
- Existing unit tests pass and new unit tests added for storage stream semantics.
- No public code paths access `pkg/core` after migration; deprecate and remove `pkg/core` if redundant.
- A developer can run `go test ./...` and the codebase builds and runs `veil serve` successfully.

---

*This document is a starting point and will be iterated during implementation.*
