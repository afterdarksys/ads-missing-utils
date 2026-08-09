# TOFIX

Findings from a code-review-graph pass over `e81652d` (clean tree, `main`).
Graph: 481 nodes / 4,091 edges / 83 files, 49 communities, 61 flows.
Risk score for the `HEAD~1` diff: **0.85 (high)**, 17 changed files, 70 test gaps.

Ordered by severity. Check items off as they land.

---

## 1. Verify path skips the key-length floor the create path enforces

- [x] **`internal/macossec/macossec.go:155` (`CheckDaemonBaseline`)** — high, confirmed

`CreateDaemonBaseline` rejects keys under 16 bytes (`macossec.go:144-146`).
`CheckDaemonBaseline` does not — it goes straight to `hmac.Equal`. A baseline
signed with an empty key therefore verifies successfully.

Confirmed with a throwaway probe test in package `macossec`:

```
create correctly rejects empty key
VERIFY ACCEPTED A BASELINE SIGNED WITH AN EMPTY KEY
```

Reachable in production through `dtp --check --key <empty-or-short-file>`:
`cmd/dtp/main.go:41` reads the key file with `os.ReadFile` and applies no length
validation, so a forged baseline plus a zero-byte key file passes daemon
tampering detection.

**Fix:** apply the same `len(key) < 16` guard in `CheckDaemonBaseline`, or hoist
it into a shared helper both entry points call. Validating in `cmd/dtp/main.go`
too would fail earlier with a clearer message, but the library must not depend
on its callers for this.

## 2. Swallowed marshal error in `signBaseline`

- [x] **`internal/macossec/macossec.go:434`** — medium

`data, _ := json.Marshal(clone)` discards the error. On failure `data` is nil and
the HMAC is computed over an empty message, so every baseline would share one
signature under a given key. This is the exact fail-open shape rule 8 of
`~/development/ads-fable-utils/SECURITY-RULES.md` bans ("no swallowed errors").

Not currently exploitable — `SignedBaseline` holds only strings and
`map[string]string`, which cannot fail to marshal — but `signBaseline` returns a
bare `string` and so *cannot* signal failure if the struct later gains a field
that can (e.g. anything with a custom marshaller, a channel, or a func).

**Fix:** change the signature to `(string, error)` and propagate.

## 3. `make test` and `make check` fail from the repo root

- [x] **`Makefile:19-24`** — medium

`contrib/vibedetector` is a git submodule with no `go.mod` of its own, so
`go test ./...` pulls its demo file into this module's package graph:

```
contrib/vibedetector/demo_go_code.go:15:18: undefined: db
FAIL github.com/afterdarksys/ads-missing-utils/contrib/vibedetector [build failed]
```

`go vet ./...` fails identically. `build.sh` passes because `test_project` and
`check_project` scope to `./cmd/... ./internal/... ./tests`. The two entry points
therefore disagree about whether the repo is green. Every first-party package
passes on its own.

**Fix:** scope the `Makefile` targets the same way `build.sh` does, or add a
`go.mod` inside the submodule so it stops being part of this module.

## 4. Missing wrong-key negative test for the baseline signer

- [x] **`internal/macossec/macossec_test.go:28`** — medium

`TestDTPRejectsTamperAndReportsChange` covers a tampered signature but never
verification under a *different* key, nor a short/empty key. `SECURITY-RULES.md`
requires security-domain assets to prove rejection of
"tampered/expired/oversized/**wrong-key**" input — the missing wrong-key leg is
precisely what finding #1 slipped through.

**Fix:** add cases for (a) verify with a different 16-byte key, (b) verify with
an empty key, (c) verify with a truncated/garbage `signature` field. Item #1 is
not done until (b) is a passing regression test.

## 5. A single unreadable file aborts the whole executable scan

- [x] **`internal/macossec/macossec.go:421` (`executableHashes`)** — low

`hashFile`'s error propagates out of `filepath.WalkDir`, so one EPERM under
`/Library/LaunchDaemons` fails the entire index or check with an opaque message.
Failing closed is the right call for `--check` and should stay — the problem is
legibility, not the policy.

**Fix:** record the unreadable path as an explicit error entry in the baseline
(or a separate `errors` list) so the failure names the file and still cannot be
silently skipped.

## 6. `cmd/meow/main.go::run` is 209 lines

- [x] **`cmd/meow/main.go:64-272`** — low, maintainability

The only function in the repo over 120 lines, and the graph's #1 hub node
(out-degree 56). `cmd/meow` has no test files, so all 209 lines are uncovered.

**Fix:** split flag handling, transformation, and output into separate functions
so the transform path becomes testable without a process.

---

## Test coverage gaps (graph-reported, 70 total)

`internal/macossec` is the densest cluster. Uncovered exported surface includes
`SaveBaseline`, `LoadBaseline`, `safeJoin`, `ScanWebKit`, `SnapshotProcesses`,
`QuarantineCandidates`, and `AuditAccessibility`.

These `cmd/` packages have no test files at all:

```
jsonprobe  jwalk   meow    netwhy  patchwhy  ports   portwhy
pwatch     regocheck  sandboxdiff  servicewhy  spacelift-helper
tfchanges  tjt     varmerge  webkit-tool
```

---

## Verified correct — do not "fix" these

- `CheckDaemonBaseline` verifies the signature (`macossec.go:156`) *before*
  walking `b.Root` (`macossec.go:159`), so the attacker-controllable `Root` field
  in a loaded baseline cannot redirect the scan pre-authentication.
- Signature comparison uses `hmac.Equal`, not `==` (rule 2, constant-time).
- `safeJoin` (`macossec.go:475`) correctly rejects absolute paths and `..`
  escapes; same helper in `internal/homestate/homestate.go:534`.
- `atomicJSON` (`macossec.go:449`) creates state with `0o700`/`0o600`, syncs
  before rename, and cleans up its temp file.
