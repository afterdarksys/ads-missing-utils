# macOS Security Tool Ideas

This file is the status ledger for the macOS-oriented research tools. A tool is
marked complete only when it has a functional command, tests, documented safety
boundaries, and versioned JSON output.

## Completed

| Tool | Status | What is implemented |
|---|---|---|
| `homestate` | Complete | Index `$HOME` or `--map-dir`, compare state, list new or moved files, detect problematic names, and perform collision-safe reversible name repair. |
| `metascore` | Complete (MVP) | Snapshot file metadata and extended-attribute names where available, then score additions, removals, and metadata mutations. |
| `pll` | Complete (MVP) | Parse and lint XML plists, detect duplicate keys, missing declared executables, and risky `RunAtLoad` + `KeepAlive` launchd persistence. |
| `bundleinfo` | Complete (MVP; former idea 10) | Inventory `.app` bundles and `.pkg` paths without executing contents; report bundle identifiers, executables, signatures, counts, and optional macOS package metadata. |

The implementation contract and acceptance criteria for these four commands are
recorded in [`specs/HOMESTATE_AND_MACOS_TOOLS.md`](specs/HOMESTATE_AND_MACOS_TOOLS.md).

## Remaining ideas

1. `clt` — Clone-Exec Trapping: Detect and quarantine hidden executable payloads spawned via APFS `clonefile()` cloning.
2. `dtp` — Daemon Twin Profiling: Compare running system daemons with cryptographically signed baseline manifests, with a separately designed Secure Enclave trust model.
3. `tjt` — Transient Job Tracking: Audit ephemeral background tasks that execute and self-delete within a sub-second timeframe.
4. `cded` — Clipboard Data Exfiltration Draining: Sanitize sensitive clipboard payloads when an active background process lacks appropriate UI focus.
5. `amtm` — AppleScript Macro Intent Mapping: Translate incoming automation commands into a visual permission graph before execution.
6. `aaas` — Accessibility API Abuse Shield: Protect password-field coordinates from rogue third-party accessibility clients across native Cocoa applications.
7. `webkit-tool` — Research the macOS WebKit API surface and produce structured capability and security evidence.

The remaining active-defense ideas require explicit threat models, macOS entitlement
and API research, and non-destructive acceptance criteria before implementation.
