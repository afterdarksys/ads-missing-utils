# macOS Security Research Suite

**Status:** Implemented and unreleased
**Date:** 2026-08-09

The macOS research suite consists of eleven versioned, JSON-producing commands:
`homestate`, `metascore`, `pll`, `bundleinfo`, `clt`, `dtp`, `tjt`, `cded`,
`amtm`, `aaas`, and `webkit-tool`.

## Command contracts

- `clt` hashes regular files and reports hidden executable files that have an
  identical size and SHA-256 twin. This is clone-payload triage evidence, not a
  claim that APFS exposes clone provenance. Quarantine requires both `--apply`
  and `--quarantine-dir`.
- `dtp` snapshots executable files beneath a daemon root and authenticates the
  baseline with HMAC-SHA-256. The key is supplied by the operator and may be
  provisioned by a Secure Enclave-backed external workflow; this command does
  not claim to store arbitrary manifest data in the Secure Enclave.
- `tjt` samples the native process table at a configurable sub-second interval
  and reports observed PIDs that disappear, including whether their executable
  path was deleted. Processes entirely between samples are inherently unseen.
- `cded` fingerprints private-key, access-token, and seed-phrase patterns without
  emitting clipboard contents. Clipboard clearing requires `--drain`; file input
  is always read-only.
- `amtm` statically maps AppleScript shell, Accessibility, clipboard, network,
  and filesystem intent into a permission graph. It never executes the script.
- `aaas` reads macOS TCC Accessibility grants through `sqlite3`. Revocation uses
  `tccutil reset Accessibility` only with `--revoke BUNDLE_ID --apply`. Full Disk
  Access may be required to audit the user TCC database.
- `webkit-tool` statically scans Swift, Objective-C, JavaScript, and HTML sources
  for risky WebKit file access, private KVC, popup, and script-bridge settings.

## Safety and platform boundaries

Inspection is read-only. The only mutations are explicit CLT quarantine, CDED
clipboard draining, AAAS grant revocation, and Homestate name repair/restoration.
The suite does not install kernel extensions, request Endpoint Security
entitlements, intercept Accessibility calls, execute AppleScript, or represent
static evidence as proof of malicious behavior.

Core parsing, correlation, signing, tracking, and detection behavior is covered
by portable Go tests. Native commands return clear runtime errors when a required
macOS facility (`pbpaste`, `pbcopy`, `osascript`, `tccutil`, or the TCC database)
is unavailable.
