# VEIL ↔ CODEX Integration & Implementation Plan

This document collects the repository audit findings, decisions, step-by-step instructions, developer actions, and migration guidance required to merge and unify VEIL and CODEX into a single cohesive ecosystem.

Status: Draft — created from the repository audit and initial planning session.

---

## Summary

Veil is a Go single-binary application that functions as a local-first "second brain" (SQLite-backed), with an extensible plugin system and a web UI (`/web`). Codex is a knowledge-graph, versioned data system with CLI and UI prototype under `cmd/codex` and `codex-universalis/`. The goal is to converge these into a single ecosystem where:

- **Codex** is the canonical, versioned knowledge graph (library: `pkg/codex`).
- **Veil** is the user-facing environment, editor, plugin host, and the interface into Codex (binary: `veil`).

This doc provides precise, actionable instructions and context so contributors can implement the unification reliably and safely.

---

## Repo map & immediate findings (from audit)

- `main.go` — Veil app entry-point and CLI handlers (init, serve, gui, new, publish, export, etc.).
- `cmd/codex/` — Codex CLI commands (add, annotate, commit, entity, gui, init, push, server, status).
- `.codex/objects/` — Existing content-addressed JSON objects (Codex artifacts).
- `web/` and `codex-universalis/` — UI assets and prototypes (Veil and Codex UI views).
- `plugins.go`, `plugins_api.go` — Plugin registry and execution API; many plugins implemented in Go files (git, ipfs, media, svg, shader, terminal, reminder, todo, etc.).
- `veil.db` — SQLite database containing Veil nodes and plugin registry.
- `test.sh` — Basic integration script used in repo.

Key conclusions:
- Veil already implements a lot of UI/plugin infrastructure that Codex can reuse.
- A refactor to a canonical library package (`pkg/codex`) will make Codex functionality reusable by Veil, the CLI, and hosted services.

---

## Design decisions & recommendations (executive)

- Create `pkg/codex` as the canonical library with public API for: data model, storage interface, version/diff logic, commit model, and query engine.
- Implement pluggable storage backends (file object store for `.codex`, sqlite index, and optional IPFS adapter).
- Keep `veil` as the main single-binary distribution and implement `veil codex <command>` subcommands. Keep `cmd/codex` as a shim or alias during migration.
- Merge `codex-universalis` UI into `web/` as `/codex` routes and provide a small JS client that talks to `/api/codex/*` and a WebSocket change-stream endpoint.
- Implement a migration tool `veil migrate` (with `--dry-run` and `--backup`) that safely transforms existing `.codex` objects and Veil DB nodes into the unified Codex store with verification and rollback.
- Extend the plugin model into a two-tier system: in-process trusted Go plugins + out-of-process gRPC/HTTP plugins for sandboxed execution.

---

## Canonical data model (brief)

Primary objects (JSON-LD compatible):

- Entity: { urn, labels, type, properties }
- Text: { urn, source, content, spans }
- Relationship: { from, to, predicate, properties }
- Commit: { hash, parents[], author, timestamp, message, manifest[] }
- Artifact: binary object reference (content-hash and metadata)
- Provenance: plugin actions and signed assertions

Storage: content-addressed objects stored in object store (`.codex/objects`), with a lightweight index table (sqlite) for commits, heads, and search indexes.

---

## Concrete developer actions & code-level guidance

1. **Create `pkg/codex` skeleton**

   - Path: `pkg/codex`
   - Expose types and interfaces (example signatures):

```go
package codex

import "time"

// Commit represents a change set
type Commit struct {
    Hash      string
    Parents   []string
    Author    string
    Timestamp time.Time
    Message   string
    Objects   []string
}

// Storage is the pluggable storage interface
type Storage interface {
    PutObject(hash string, payload []byte) error
    GetObject(hash string) ([]byte, error)
    ListObjects(prefix string) ([]string, error)
    PutCommit(c *Commit) error
    GetCommit(hash string) (*Commit, error)
}

// NewRepository creates a new repository instance backed by a Storage implementation
func NewRepository(storage Storage) *Repository { /* ... */ }
```

2. **Implement `storage/fs` adapter** (wraps existing `.codex/objects`) as the canonical initial storage backend.
3. **Implement `storage/sqlite` index** that keeps heads, commits, entity indexes, and enables fast queries.
4. **Refactor `cmd/codex` to call `pkg/codex` APIs** and then deprecate it in favor of `veil codex`.
5. **Add REST API handlers** in `main.go` or `api/codex/`:
   - `POST /api/codex/query`
   - `POST /api/codex/commit`
   - `GET /api/codex/object/{hash}`
   - `GET /api/codex/repo/status`
   - `GET /api/codex/stream` (WS / SSE real-time changes)

6. **Merge `codex-universalis` into `web/`** under `/codex` route; add a simple JS client to fetch and stream updates.

7. **Migration CLI** `veil migrate --dry-run --backup` should:
   - Backup `veil.db` and `.codex` into a ZIP (timestamped)
   - Scan `.codex/objects/` and Veil SQLite for nodes
   - Convert Veil nodes into Codex Entities/Texts with provenance
   - Import objects into `pkg/codex` storage and create a migration commit
   - Verify counts and checksums; fail safely with actionable error messages

8. **Plugin changes**
   - Add hooks in `pkg/codex` for operations (OnCommit, OnQuery, OnResolveURI)
   - Extend `plugins_api.go` to register Codex-capable plugins (declare capabilities)
   - Implement gRPC plugin adapter for external plugin execution with timeouts and resource limits

---

## Migration & backup commands (how to perform a manual backup prior to running automated migrate)

```bash
# Back up DB and codex objects
cd /path/to/veil
cp veil.db veil.db.bak
zip -r veil-backup-$(date -u +%Y%m%dT%H%M%SZ).zip veil.db .codex/

# Run veil migrate (once implemented)
./veil migrate --dry-run --backup
```

---

## Testing & CI

- Add unit tests for `pkg/codex` (commit/diff/merge semantics, storage adapters) in `_test.go` files.
- Add integration tests for migration and CLI commands.
- Add GitHub Actions workflows with steps: lint -> build -> test -> integration -> e2e. Use `go test ./...` for initial coverage.

---

## Recommended immediate tasks (priority order)

1. Create `pkg/codex` skeleton and `storage/fs` adapter (spike + tests).
2. Add `veil codex status` command that calls into `pkg/codex` to demonstrate wiring.
3. Add `docs/INTEGRATION_PLAN.md` (this document) and add README references.
4. Implement migration `veil migrate --dry-run` spike and add a comprehensive integration test using a copied fixture repo.

---

## Developer workflow (commands)

- Build Veil: `go build .` or `make build`
- Run Veil server: `./veil serve --port 8080`
- Run tests: `go test ./...`
- Run lint: `golangci-lint run`
- Run codex CLI (during migration): `go build ./cmd/codex && ./codex <command>`
- Run the hypothetical migrate (once implemented): `./veil migrate --dry-run --backup`

---

## Risks and mitigations

- Back up before migrating: always ZIP the DB and `.codex`.
- Keep a compatibility shim: maintain `cmd/codex` as an alias until `veil codex` is stable.
- Validate commits with checksums and automated tests to avoid data loss.

---

## Next steps (explicit, actionable)

1. Add `pkg/codex` skeleton and tests (PID: #1 in project Todo list).
2. Implement `storage/fs` adapter and unit tests.
3. Wire `veil codex status` to `pkg/codex`.
4. Add migration CLI skeleton: `veil migrate --dry-run`.

---

## Where this file lives

- Path: `docs/INTEGRATION_PLAN.md` (this file)

---

If you'd like, I can now implement the `pkg/codex` skeleton and a `veil codex status` integration as the first PR. Which task should I start with? 
