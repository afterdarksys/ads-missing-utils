# Handoff: Missing Utils release-readiness pass

## Current state

The working tree contains one coherent, uncommitted implementation/documentation
pass. Do not discard or reset its changes. The active branch is `main`; its
starting commit was `df56719` (`fix reviewed security and build issues`). No
commit or push has been made for this pass because `TODO` is not complete.

The remaining active TODO items are intentionally still unchecked:

1. Produce signed release artifacts, checksums, SBOMs, and provenance.
2. Complete package/name collision review before publishing binaries.

The candidate-tool and AI-tool research items are product discovery, not
release blockers for the current codebase.

## What changed in this pass

- Deleted the completed `TOFIX.md` ledger and moved its completion record into
  `TODO`.
- Added `binparse`, `logic`, `man2logic`, and `info2logic`, plus their internal
  packages, documentation, and source man pages.
- Added a section-1 roff page for every command under `cmd/` (44 pages).
- Added `make man-build`, `make man-install`, `./build.sh man-build`, and
  `./build.sh man-install`. Built pages are gzip files under `dist/man`;
  installation targets `PREFIX/share/man/man1` or `prefix/share/man/man1`.
- Added the Logic Manual v1 format specification in `docs/LOGIC_FORMAT.md`.
- Replaced disabled `tests/v1_spec_test.go` stubs with executable black-box
  acceptance tests in `tests/v1_acceptance_test.go`.
- Fixed `hashsum --from-jwalk` to prefer `jwalk`'s `relative_path`; this makes
  the documented `jwalk ROOT | hashsum create --from-jwalk --root ROOT`
  workflow work when `jwalk` emits absolute paths.
- Added JSON Schema validation to the v1 acceptance suite using pinned
  `github.com/santhosh-tekuri/jsonschema/v6 v6.0.2`.
- Fixed the Windows compilation issue in `internal/ports` by separating
  Linux-specific UID extraction into build-tagged files. `ports` remains
  Linux-only at runtime and reports that boundary explicitly.
- Added enterprise Linux diagnostic MVPs: `unitwhy` (bounded systemd and
  journal evidence), `restartwhy` (visible deleted process mappings), and
  `selinuxwhy` (enforcement and bounded AVC evidence). Each has a v1 schema
  and a section-1 manual page.
- Strengthened `netwhy` with DNS, Linux default-route, and credential-redacted
  proxy evidence; strengthened `certwhy` with TLS metadata and a clearly
  failed unverified diagnostic fallback; and expanded `incidentsnap` with OS,
  kernel, boot ID, and uptime evidence. The manual pages state collection
  boundaries.
- Added `pcapwhy`, an offline-only PCAP/PCAPNG analyzer for bounded IPv4/IPv6
  flow, MAC/VLAN, TCP reset, ICMP, payload-metadata, and multi-capture evidence.
  It has a generated-capture test, v1 schema, docs, release entry, and man page.
- Updated `README.md`, `PROJECT_PLAN.md`, `TODO`, `specs/V1.md`, and release
  metadata to reflect this state.

## Verification already completed

All of the following passed after the changes:

```sh
make check
make man-build
GOOS=linux GOARCH=amd64 go build ./cmd/...
GOOS=darwin GOARCH=arm64 go build ./cmd/...
GOOS=windows GOARCH=amd64 go build ./cmd/...
git diff --check
```

The v1 acceptance suite builds the three v1 binaries and validates real output
against the `jwalk`, `envsub`, and `hashsum` Draft 2020-12 schemas with format
assertions enabled.

The final enterprise-diagnostics tranche passed its targeted package tests,
`make man-build`, and a 44-command/44-man-page coverage check. A direct
`go test ./...` is not the project verification command: it includes the
intentionally non-compiling demo source in `contrib/vibedetector` and this
sandbox cannot read the user-level Go build cache. Use `make check` with a
workspace-local Go cache for full project verification.

## Recommended next steps

### 1. Decide release signing/provenance authority

Do not mark the signing TODO complete without a real release identity. The
repository needs the owner to choose either keyless OIDC signing (for example,
GitHub Actions with Sigstore/Cosign) or a managed signing key. Confirm the
target tag/version and whether artifacts are released through GitHub Releases.
Then add the appropriate protected CI/release workflow and test it on a
prerelease tag or dry run. The existing `.goreleaser.yaml` already creates
checksums and SBOMs, but it does not configure signing or provenance.

### 2. Perform name-collision review with an owner decision

The plan already flags generic or potentially colliding names including
`jwalk`, `envsub`, `hashsum`, `ports`, and `spacelift-helper`. Review package
registries, Homebrew, Linux distributions, GitHub, shell PATH collisions, and
relevant marks. Record results in a new decision document. Do not rename
binaries without the owner deciding whether to retain standalone names or adopt
a suite prefix.

### 3. Optional: prepare a commit after review

Before committing, inspect the full diff and confirm all 44 man pages are
intended. If approved, use one descriptive commit covering the implementation,
documentation, tests, and release-build changes. Do not push without explicit
owner authorization.

## Important implementation details

- `binparse` is read-only metadata inspection only; it does not execute or
  trust the inspected binary.
- Logic Manual v1 is a bounded text format with YAML front matter and named
  sections. The parser deliberately supports a small, deterministic grammar.
- `man2logic` handles common roff `.TH`, `.SH`, `.SS`, and inline font macros;
  it is not a full troff renderer. `info2logic` imports GNU Info nodes.
- The project must retain the `jsonschema/v6` dependency because acceptance
  tests now compile and validate the repository's JSON schemas.
- The last sandboxed verification used a workspace-local Go module cache.
  If creating one again, Go may make it read-only; remove it with
  `chmod -R u+w .cache && rm -rf .cache` when it is no longer needed.
