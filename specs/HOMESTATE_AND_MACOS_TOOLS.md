# Spec: Homestate and macOS Research Utilities

**Author:** Codex, from the user's requirements
**Date:** 2026-08-09
**Status:** Approved (implementation explicitly requested by the project owner)
**Reviewers:** Project owner
**Related specs:** `specs/V1.md`

## Context

The project needs a local, read-only-by-default way to inventory a home directory,
compare it with a durable baseline, identify additions and likely moves, and find
problematic path names. Name repair must be explicit and reversible. The project's
idea list also contains macOS security research utilities with no implementation or
completion criteria. This increment implements three bounded, evidence-producing
ideas: metadata anomaly comparison, XML property-list linting, and application/package
bundle inventory.

## Functional Requirements

- FR-1: `homestate --index` MUST recursively inventory a root, defaulting to `$HOME`, into a versioned JSON state file.
- FR-2: `homestate --check` MUST compare the live root with the saved state and report added, removed, modified, and likely moved regular files.
- FR-3: `homestate --new-files` and `homestate --moved-files` MUST filter comparison output to their named category.
- FR-4: Move detection MUST correlate removed and added regular files by SHA-256 and size, and MUST label ambiguous hash matches as ambiguous rather than inventing a move.
- FR-5: `homestate --broken-file-names` MUST report path components containing control characters or `~`, `:`, `;`, single quote, or double quote.
- FR-6: `homestate --fix-names` MUST replace invalid runs with `_`, avoid collisions deterministically, and record every successful rename in the state file.
- FR-7: `homestate --restore-names` MUST restore recorded renames only when the original path is free and the current path still exists.
- FR-8: `homestate --map-dir DIR` MUST use DIR instead of `$HOME` for indexing, checking, or name operations.
- FR-9: The state file MUST be excluded from its own inventory.
- FR-10: `metascore index|check` MUST snapshot file metadata and score changed mode, size, modification time, and extended-attribute names when the platform exposes them.
- FR-11: `pll` MUST parse XML property lists, reject malformed or duplicate dictionary keys, and report risky launchd logic such as an absent executable or `RunAtLoad` combined with `KeepAlive`.
- FR-12: `bundleinfo` MUST inventory `.app` directories and `.pkg` paths, report bundle structure and metadata without executing bundle contents, and use `pkgutil` only as an optional read-only enrichment on macOS.
- FR-13: Every command MUST support `--help` and `--version` and emit versioned JSON for operational results.

## Non-Functional Requirements

- NFR-1: All inspection modes MUST be read-only.
- NFR-2: Mutation MUST occur only for explicit `--fix-names` or `--restore-names` operations.
- NFR-3: State writes MUST use a same-directory temporary file followed by rename.
- NFR-4: File content MUST be streamed while hashing; the implementation MUST NOT load whole files into memory.
- NFR-5: Permission and race failures MUST be represented as diagnostics and MUST NOT panic.
- NFR-6: Output ordering MUST be deterministic for identical filesystem state.

## Acceptance Criteria

### AC-1: Durable index (FR-1, FR-9, NFR-3)
Given a temporary tree, when indexed, then the JSON database contains its files but not the database itself and can be decoded after the command exits.

### AC-2: Complete and filtered comparison (FR-2, FR-3)
Given an indexed tree with one added, one removed, and one modified file, when checked, then each change has the correct status and filtered modes contain only their category.

### AC-3: Evidence-based movement (FR-4)
Given a file renamed without content changes, when checked, then one move links the old and new paths; duplicate-content candidates are reported ambiguous.

### AC-4: Broken name detection (FR-5)
Given names containing every configured invalid character, when scanned, then each is reported with its reason.

### AC-5: Reversible safe repair (FR-6, FR-7, NFR-2)
Given a problematic name, when fixed and then restored, then the original path returns and both operations are recorded without overwriting another file.

### AC-6: External map root (FR-8)
Given an external mapping root, when `--map-dir` is supplied, then all stored paths are relative to that root.

### AC-7: Metadata scoring (FR-10)
Given a metadata baseline and a mode or content change, when `metascore check` runs, then a nonzero anomaly score and evidence are returned.

### AC-8: Property-list linting (FR-11)
Given valid and invalid launchd XML plists, when linted, then syntax, duplicate-key, missing-program, and risky persistence findings are deterministic.

### AC-9: Bundle inventory (FR-12)
Given a synthetic `.app`, when inventoried, then its `Info.plist`, executable declaration, code-signature presence, and file counts are reported without execution.

### AC-10: CLI lifecycle (FR-13)
Given each command, when `--help` or `--version` is invoked, then it exits successfully.

## Edge Cases and Error Scenarios

- EC-1: Missing `$HOME` and no `--map-dir` produces usage error.
- EC-2: Missing or malformed state file produces a diagnostic and no filesystem mutation.
- EC-3: Symlinks are inventoried by link target and are never followed during traversal.
- EC-4: A file changing while hashed is marked unstable.
- EC-5: A fix or restore destination collision is reported and left untouched.
- EC-6: Binary plists receive an explicit unsupported-format finding; they are not misparsed as XML.
- EC-7: Non-macOS `.pkg` inspection reports limited evidence rather than failing or executing helpers.

## API Contracts

N/A — these are local CLI contracts and expose no HTTP methods or endpoints.

```typescript
interface HomestateDB { schema: "missing-utils/homestate-state/v1"; root: string; indexed_at: string; entries: Record<string, StateEntry>; renames: RenameRecord[] }
interface StateEntry { path: string; type: "file"|"dir"|"symlink"|"other"; size: number; mode: number; mtime_ns: number; sha256?: string; link_target?: string; unstable?: boolean }
interface Change { status: "added"|"removed"|"modified"|"moved"|"ambiguous_move"; path: string; old_path?: string; evidence?: string[] }
interface RenameRecord { from: string; to: string; applied_at: string; restored_at?: string }
interface CommandResponse<T> { schema: string; command: string; outcome: "pass"|"changes"|"partial"|"error"; data: T; diagnostics?: Diagnostic[] }
interface Diagnostic { code: string; message: string; path?: string }
```

## Data Models

| Entity | Field | Type | Constraints |
|---|---|---|---|
| State database | schema | string | Exact v1 identifier |
| State database | root | absolute path | Existing directory at index time |
| State database | entries | path map | Root-relative slash-separated keys |
| State entry | sha256 | hex string | Regular files only |
| Rename | from/to | relative paths | Must remain beneath root |
| Change | status | enum | Defined by API contract |
| Metadata baseline | entries | path map | Versioned JSON and atomic writes |
| Plist report | findings | array | Stable code, severity, message |
| Bundle report | kind | enum | `app` or `pkg` |

## Out of Scope

- OS-1: Continuous monitoring, filesystem event subscriptions, cloud synchronization, and content restoration.
- OS-2: Treating ordinary spaces or Unicode as broken names.
- OS-3: Following symlinks or indexing device contents.
- OS-4: Automatic quarantine, Secure Enclave storage, kernel extensions, or endpoint-security authorization.
- OS-5: Binary-plist decoding and package extraction in this increment; platform tools may add read-only package metadata.
