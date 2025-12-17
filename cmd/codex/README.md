# Codex CLI MVP

This is a minimal, Go-based CLI prototype for the Codex Universal project.

Features (MVP):
- codex init — create a local `.codex/` repository
- codex add <path> — add a file (stored as an object and staged)
- codex commit -m "msg" — commit staged objects
- codex status — show branch, HEAD, staged objects
- codex entity add --id <id> --type <type> [--label en=Name] — add an entity
- codex annotate --text <urn> --entity <urn> --start <n> --end <n> — add an annotation
- codex push <remote-url> — push HEAD commit to a remote server (POST /push)
- codex server — start a mock server (endpoints: /datasets, /push)
- codex gui — launch a local PWA-style GUI (http://localhost:3000)

Usage example:

```
codex init
codex add data/iliad.json
codex commit -m "Add Iliad fragment"
codex server  # in another terminal
codex push http://localhost:8080
```

This MVP stores objects in `.codex/objects/` and maintains a simple `index.json` for staging.

Next steps: ontology validation, merge/conflict handling, cryptographic signing, GUI integration, and federated node discovery.
