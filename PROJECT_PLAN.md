# Missing Utils: Project Plan

**Status:** Initial proposal

**Working title:** Missing Utils

**Primary platform:** Linux, with explicit cross-platform support where noted
**Implementation assumption:** Go, producing dependency-light static binaries

## 1. Executive summary

Missing Utils is a collection of small, composable command-line tools for DevOps and security work. The project focuses on two gaps in the Unix/Linux ecosystem:

1. Routine operations still require fragile pipelines or platform-specific commands.
2. Existing tools report system state but rarely explain the chain of decisions that produced it.

The project will therefore ship two related families of utilities:

- **Operational primitives** provide fast, predictable, structured replacements for common shell pipelines: `jwalk`, `envsub`, `hashsum`, `pwatch`, and `ports`.
- **Causal diagnostics** correlate evidence from processes, services, packages, permissions, namespaces, networking, and security controls: `portwhy`, `accesswhy`, `patchwhy`, and later utilities.

Each command must work independently. Higher-level diagnostic commands may reuse the same internal libraries and evidence model, but users will not need to install a daemon or the entire suite to use a single utility.

## 2. Product thesis

Linux already has excellent low-level inspection commands. The missing layer is a coherent set of tools that is:

- safe and read-only by default;
- structured for automation but pleasant for humans;
- explicit about evidence, uncertainty, and platform limitations;
- namespace- and container-aware;
- consistent across commands;
- deployable as small standalone binaries;
- capable of explaining *why*, not merely reporting *what*.

The primary users are SREs, platform engineers, security engineers, incident responders, CI/CD maintainers, and developers debugging production-like systems.

## 3. Goals

- Replace recurring, error-prone shell pipelines with stable command contracts.
- Make JSON Lines the standard streaming format and ordinary JSON the standard aggregate format.
- Correlate system evidence into auditable explanations.
- Operate without a resident agent for normal use.
- Support least-privilege operation and clearly identify unavailable evidence.
- Provide stable exit codes and machine-readable errors.
- Share libraries without forcing all commands into one large binary.
- Package for common Linux distributions and offer signed release artifacts.

## 4. Non-goals

- Building a full monitoring, EDR, SIEM, configuration-management, or vulnerability-scanning platform.
- Replacing `find`, `systemd`, eBPF tooling, osquery, or platform-native debuggers in every use case.
- Silently modifying firewall rules, permissions, services, or configuration.
- Evaluating arbitrary shell expressions in templates.
- Attaching a debugger or sending disruptive signals without explicit authorization.
- Claiming a definitive answer when permissions or platform support prevent collecting required evidence.

## 5. Utility portfolio

### 5.1 Operational primitives

| Command | Purpose | Initial platform |
|---|---|---|
| `jwalk` | High-performance directory traversal with filtering and JSON Lines output | Linux, BSD/macOS |
| `envsub` | Strict, typed, secret-aware environment substitution | Cross-platform |
| `hashsum` | Concurrent multi-algorithm hashing and verifiable stream manifests | Cross-platform |
| `pwatch` | Process and child-tree health/lifecycle observation | Linux first |
| `ports` | Uniform process-to-listener inventory | Linux first; BSD and Solaris adapters follow |

### 5.2 Causal diagnostics

| Command | Question answered | Dependencies |
|---|---|---|
| `portwhy` | Who owns this port, how was it launched, and where is it reachable? | `ports` evidence library |
| `accesswhy` | Why can or cannot an identity access a filesystem object? | Filesystem, ACL, capabilities, mount, and LSM adapters |
| `patchwhy` | Which processes use replaced code, and what must restart? | Process and package adapters |
| `netwhy` | Which DNS, route, namespace, proxy, and firewall decisions affect a connection? | Network evidence adapters |
| `servicewhy` | Why is a service unhealthy, restarting, or blocked? | `pwatch` and service-manager adapters |
| `binarywhy` | Where did an executable come from, and has it changed? | Package and provenance adapters |
| `authwhy` | Which account, group, SSH, PAM, NSS, or policy rule affects login? | Identity adapters |
| `certwhy` | Which certificate and trust path are in use, and why does validation fail? | TLS and trust-store adapters |
| `expose` | Which services are reachable from each interface or namespace? | `ports` and network-policy adapters |
| `sandboxdiff` | How do two workloads differ in isolation and attack surface? | Process, systemd, namespace, capability, seccomp, and LSM adapters |
| `driftwhy` | What security-relevant state changed and what mechanism changed it? | Snapshot and provenance model |
| `incidentsnap` | How can a responder collect a redacted, integrity-verifiable host snapshot? | Shared collectors and redaction library |

### 5.3 Automation and IaC utilities

These commands consume and emit JSON as their primary interface. They are designed for direct CLI use, Ansible collection wrappers, OpenTofu/Terraform plan pipelines, and Spacelift lifecycle hooks.

| Command | Purpose | Primary inputs |
|---|---|---|
| `tfchanges` | Normalize a Terraform/OpenTofu plan or state into concise resource-change records, summaries, and risk-relevant facts | `terraform show -json` |
| `varmerge` | Merge environment, JSON, YAML, and defaults under a typed schema while preserving value provenance | JSON, YAML, environment, schema |
| `jsonprobe` | Execute declarative DNS, TCP, TLS, HTTP, process, file, and command-free health checks | JSON check specification |
| `jsondiff` | Compare structured desired and observed state with typed paths, ignore rules, keyed-array matching, and machine-readable changes | Two JSON documents |
| `jsongate` | Convert findings from any suite command into a consistent pass, warn, deny, or approval-required decision | Findings and threshold policy |

These tools should remain narrow. `jsongate` is not a replacement for OPA/Rego, Sentinel, or Spacelift policy. It supplies a predictable local/CI exit decision for hooks and simple workflows; organizations with a policy engine can consume the underlying findings directly.

## 6. Initial utility specifications

### 6.1 `jwalk`: modern file walker

#### User promise

Traverse large directory trees quickly and emit one self-contained JSON object per result, without requiring a brittle combination of `find`, shell quoting, `stat`, and `-exec`.

#### MVP capabilities

- Select roots and include or exclude entries by path/name regular expression.
- Filter by entry type, modification age, access age, and size ranges.
- Prune matching directories before descending.
- Choose symlink behavior, filesystem-boundary behavior, and worker count.
- Emit NDJSON with path, relative path, type, size, timestamps, mode, UID/GID, inode, device, and symlink target where available.
- Emit structured error records for permission failures and traversal races.
- Support `--fail-on-error`, `--quiet-errors`, and an explicit unordered fast mode.
- Preserve arbitrary path bytes safely in structured output; never depend on newline-delimited raw filenames.
- Stop promptly on cancellation or a closed output pipe.

#### Example contract

```shell
jwalk /var/log --path-regex '\.log$' --older-than 7d --min-size 10MiB
```

```json
{"schema":"missing-utils/jwalk/v1","path":"/var/log/app.log","type":"file","size":19403821,"mtime":"2026-07-20T14:32:10Z"}
```

#### Deferred capabilities

- Content matching.
- Hashing selected files.
- A constrained action runner. The MVP intentionally composes with `jq`, `xargs -0`, or an API consumer rather than embedding a shell.
- Remote/object-storage walkers.

### 6.2 `envsub`: smart environment injector

#### User promise

Render configuration templates deterministically, fail safely on invalid or missing values, and avoid leaking secrets into diagnostics.

#### MVP capabilities

- Recognize `${NAME}` and `${NAME:-default}` without evaluating shell syntax.
- Load values from the process environment, one or more `.env` files, and explicit `--set` flags with documented precedence.
- Validate variables using a schema supporting string, integer, number, boolean, enum, URL, duration, byte size, and JSON values.
- Distinguish unset values from explicitly empty values.
- Require all referenced keys by default unless a default or optional schema entry exists.
- Mask schema-marked secrets in diagnostics, diffs, and audit output.
- Support `--check`, `--list-keys`, and `--explain` without emitting rendered secrets.
- Write to stdout or atomically replace a named output file while preserving requested mode/ownership behavior.
- Reject recursive expansion, command substitution, and unknown template operators.
- Return machine-readable validation failures.

#### Example contract

```yaml
# env.schema.yaml
PORT:
  type: integer
  min: 1
  max: 65535
DATABASE_URL:
  type: url
  secret: true
LOG_LEVEL:
  type: enum
  values: [debug, info, warn, error]
  default: info
```

```shell
envsub --schema env.schema.yaml --input app.yaml.tmpl --output app.yaml
```

#### Security boundaries

- Secret masking applies to logs and diagnostics, not the rendered configuration that legitimately consumes the secret.
- No implicit secret-manager network access in the MVP.
- Environment and `.env` inputs are treated as sensitive and are never echoed wholesale.

### 6.3 `hashsum`: stream manifest generator

#### User promise

Hash very large file sets efficiently, show useful progress without corrupting piped output, and produce a portable manifest that can later prove whether files are missing, added, changed, or unreadable.

#### MVP capabilities

- Compute SHA-256 and BLAKE3 by default, with an explicit algorithm list.
- Feed multiple digest implementations from one read of each file rather than rereading once per algorithm.
- Hash multiple files concurrently through a bounded, disk-aware worker pool.
- Accept paths as command arguments, safe delimited input, or `jwalk` NDJSON records.
- Support `create` and `verify` modes with a versioned manifest format.
- Emit NDJSON for streaming workflows and a deterministic aggregate manifest for signing, storage, and reproducible comparison.
- Store root-relative paths, file type, size, selected metadata, digests, errors, and pre/post-read identity observations.
- Detect files that change while being read and mark them `unstable` rather than publishing an apparently authoritative digest.
- Define explicit policies for symlinks, special files, filesystem boundaries, and unreadable entries.
- Put progress on stderr only when attached to a terminal; `--no-progress` and `CI` behavior remain deterministic.
- Verify missing, unexpected, changed, unstable, and unreadable files as distinct result classes.
- Stop promptly on cancellation or a closed pipe and bound memory independently of file-set size.

#### Example contracts

```shell
jwalk ./release --type file | hashsum create --from-jwalk -o release.hashes.json
hashsum verify release.hashes.json --root ./release
```

```json
{
  "schema": "missing-utils/hashsum/v1",
  "path": "bin/server",
  "size": 18402912,
  "digests": {
    "sha256": "...",
    "blake3": "..."
  },
  "status": "stable"
}
```

#### Integrity boundaries

- A manifest records observed content; it does not establish publisher identity unless the manifest is signed externally or by a future explicit signing feature.
- Verification must reject duplicate normalized paths, path traversal outside `--root`, unsupported algorithms, and ambiguous encodings.
- Metadata checks are separate from content checks so cross-platform restoration differences do not silently invalidate content integrity.
- Progress, warnings, and filenames are escaped before terminal display.
- MD5 and SHA-1 are excluded from default integrity policies; compatibility algorithms, if added, are visibly labeled as weak.

#### Deferred capabilities

- Manifest signing and transparency-log integration.
- Resumable hashing with a trusted cache keyed by stable file identity.
- Chunked/Merkle manifests for partial verification of very large artifacts.
- Object-storage inputs.

### 6.4 `pwatch`: process health and lifecycle

The display name may be written as **pWatch**, but the executable should remain lowercase as `pwatch` for normal Unix conventions.

#### User promise

Track a process and its descendants, explain lifecycle events and resource spikes, and capture useful diagnostics when a health policy fails.

#### MVP capabilities

- Select a target by PID, exact executable/name match, systemd unit, or listening port.
- Track descendants across forks and execs; report the limitations of reparenting and PID reuse.
- Prefer pidfds and stable process identity where the kernel supports them.
- Sample CPU, RSS, virtual memory, I/O, file descriptor count, thread count, context switches, and process state.
- Record exec, fork, exit, signal, and out-of-memory evidence where permissions permit.
- Apply threshold and duration policies such as sustained CPU, memory growth, descriptor growth, or lack of health-probe progress.
- Emit lifecycle and threshold events as NDJSON.
- Capture `/proc` state and language/runtime stack dumps through explicit, configurable adapters.
- Provide bounded retention, output rotation, and redaction.

#### Safety rules

- Passive observation is the default.
- Sending `SIGQUIT`, attaching `gdb`/`pstack`, invoking a runtime profiler, or creating a core dump requires an explicit capture policy.
- Capture adapters must declare whether they pause, signal, or otherwise affect the target.
- `--dry-run` shows every action a policy could take.
- Process matching that produces multiple candidates fails unless the user explicitly requests all matches.

#### Example contract

```shell
pwatch --port 8080 --children \
  --alert 'rss > 1GiB for 30s' \
  --capture proc,stacks --output events.ndjson
```

### 6.5 `ports`: network listener audit

#### User promise

Return one clean, uniform map of listening sockets and their owning processes across supported operating systems.

#### MVP capabilities

- Enumerate TCP and UDP listeners, IPv4/IPv6 addresses, Unix sockets where supported, and network namespace identity.
- Correlate socket identity to PID, executable, UID, command line, and service/container metadata when permitted.
- Filter by port, protocol, address, PID, process name, UID, or namespace.
- Clearly separate listeners, established connections, and unowned/kernel-owned sockets.
- Emit a versioned JSON schema with consistent nullability across platforms.
- Report partial visibility caused by privileges instead of silently omitting results.
- Avoid parsing human-oriented output from `netstat`, `ss`, or `lsof` as the primary backend.

#### Platform strategy

- **Linux:** native socket diagnostics and `/proc` correlation, with namespace-aware collection.
- **BSD family:** a dedicated adapter for the native kernel/process interfaces available on each supported BSD; BSD variants are tested separately rather than assumed identical.
- **Solaris/illumos:** a dedicated native adapter with an explicit capability matrix and fixture-based contract tests.
- **Fallback:** platform commands may be used only as a clearly identified compatibility backend, with provenance included in output.

#### Example output

```json
{
  "schema": "missing-utils/ports/v1",
  "transport": "tcp",
  "family": "ipv6",
  "local_address": "::",
  "local_port": 443,
  "state": "listen",
  "pid": 1842,
  "process": "nginx",
  "uid": 33,
  "namespace": "4026531840",
  "visibility": "complete"
}
```

#### Relationship to `portwhy`

`ports` reports the listener map. `portwhy` consumes the same evidence and adds service launch provenance, package ownership, firewall/routing analysis, container identity, and a reachability explanation. The two commands must not duplicate user-facing scope.

### 6.6 `tfchanges`: Terraform/OpenTofu change normalizer

#### User promise

Turn the large, versioned Terraform plan representation into stable, streamable facts that shell scripts, Ansible, CI hooks, and policy tools can consume without reimplementing the plan schema.

#### MVP capabilities

- Read a JSON plan or state produced by `terraform show -json` or the OpenTofu equivalent.
- Emit one NDJSON record per resource change plus an aggregate summary.
- Preserve resource address, module path, provider, resource type, actions, replacement reason, before/after sensitivity, and unknown-value state.
- Classify create, read, update, replace, delete, no-op, drift, and deferred/unknown cases without flattening meaningful distinctions.
- Filter by address, module, provider, type, action, or selected attribute path.
- Produce facts such as destructive change count, public-network exposure candidates, IAM/policy changes, database replacement, and sensitive-output changes.
- Separate evidence-based facts from configurable severity or organizational policy.
- Reject unsupported major plan-format versions and tolerate unknown fields in compatible minor versions.

#### Example

```shell
terraform show -json tfplan | tfchanges --format ndjson
terraform show -json tfplan | tfchanges summary --format json
```

### 6.7 `varmerge`: typed variable bridge

#### User promise

Combine variables from multiple automation systems deterministically and emit the exact structured format required by the next tool.

#### MVP capabilities

- Merge JSON, YAML, environment prefixes, `.env` files, and explicit values with a visible precedence chain.
- Reuse the `envsub` type schema and secret metadata.
- Emit ordinary JSON, `terraform.tfvars.json`, an Ansible extra-vars JSON object, or an `envsub` value document.
- Explain the source selected for every non-secret leaf value and identify shadowed inputs.
- Mask secret values in provenance and diagnostics.
- Detect type changes, unknown keys, ambiguous key normalization, and accidental stringification.
- Support validation-only mode and atomic output.

`varmerge` handles structured value composition. `envsub` renders text templates. Keeping these operations separate prevents templating rules from becoming a general-purpose configuration language.

### 6.8 `jsonprobe`: declarative readiness checks

#### User promise

Run the same readiness or post-deployment checks from a terminal, an Ansible task, a Terraform-compatible read-only adapter, or a Spacelift hook and receive the same JSON result.

#### MVP capabilities

- Accept a JSON document describing DNS, TCP, TLS, HTTP, process, port, and filesystem assertions.
- Support bounded retries, deadlines, backoff, and required consecutive successes.
- Emit timing, resolved endpoints, observations, redacted evidence, and a stable outcome for each assertion.
- Keep arbitrary shell execution out of the check language.
- Distinguish failed assertions, transport errors, invalid checks, timeouts, and partial visibility.
- Offer a one-shot mode suitable for Terraform's side-effect-free external data-source expectations.

### 6.9 `jsondiff`: structured drift comparison

#### User promise

Compare desired and observed JSON without treating array order, volatile fields, or representation differences as infrastructure drift.

#### MVP capabilities

- Use JSON Pointer paths in every reported change.
- Support ignore paths, numeric tolerance, set semantics, and keyed-array matching.
- Normalize timestamps, durations, byte sizes, CIDRs, and case only when explicitly configured.
- Emit added, removed, changed, type-changed, and indeterminate records as NDJSON.
- Generate a deterministic aggregate report suitable for `jsongate` or artifact retention.

### 6.10 `jsongate`: findings-to-decision adapter

#### User promise

Turn findings from `tfchanges`, `jsondiff`, `hashsum`, `ports`, or causal diagnostic commands into a consistent automation outcome without parsing human output.

#### MVP capabilities

- Consume the suite's common finding schema as JSON or NDJSON.
- Apply severity thresholds, finding-code allow/deny lists, and bounded exception files with expiry metadata.
- Emit `pass`, `warn`, `deny`, or `approval_required` plus the findings responsible for the decision.
- Use stable exit codes while retaining the complete decision in stdout JSON.
- Provide output adapters for generic CI annotations and concise Spacelift hook logs.
- Never suppress the original evidence or silently downgrade unknown/partial results.

## 7. Shared architecture

### 7.1 Repository layout

```text
cmd/
  jwalk/
  envsub/
  pwatch/
  ports/
  portwhy/
internal/
  cli/          shared flags, exit codes, and terminal behavior
  evidence/     typed facts, provenance, confidence, and unknowns
  output/       human, JSON, and NDJSON encoders
  redact/       secret and sensitive-data handling
  platform/     build-tagged operating-system adapters
  process/      stable process identity and process-tree collection
  network/      socket, route, namespace, and policy collectors
  filesystem/   traversal and filesystem evidence
  manifest/     digest algorithms, file identity, and verification
schemas/        published JSON Schemas
docs/           command references, threat model, and support matrix
testdata/       fixtures and golden outputs
```

Public Go packages should not be promised during the MVP. Shared implementation remains under `internal/` until the APIs have proven stable through multiple commands.

### 7.2 Evidence model

Every diagnostic fact should carry:

- the observed value;
- the source or collection method;
- collection time;
- subject identity, including namespace where relevant;
- confidence or completeness;
- required privilege;
- any collection error.

Conclusions must distinguish `true`, `false`, and `unknown`. Human output should cite the evidence behind important conclusions; structured output should retain it for auditing.

### 7.3 CLI contract

All commands will support, where applicable:

- `--format human|json|ndjson`;
- `--no-color` and `NO_COLOR`;
- `--quiet` and `--verbose`;
- `--timeout`;
- `--schema-version` or a schema identifier in every structured record;
- stable exit codes;
- completion generation;
- `--help` examples that are executable in tests.

Proposed shared exit codes:

| Code | Meaning |
|---:|---|
| 0 | Completed and policy/query condition was satisfied |
| 1 | Completed and no match, failed check, or triggered policy condition |
| 2 | Invalid invocation or input |
| 3 | Partial result due to permission or unsupported evidence |
| 4 | Operational failure |
| 124 | Timeout |

Commands may refine the meaning of code 1, but must document it and keep JSON output authoritative.

### 7.4 Automation JSON contract

Commands that participate in automation will support a strict stdout discipline:

- stdin accepts one versioned JSON request or an explicitly documented NDJSON stream;
- stdout contains JSON/NDJSON only;
- logs, progress, and human diagnostics go to stderr;
- secrets are represented by redaction metadata and never copied into diagnostics;
- every response includes schema, operation, outcome, data/findings, completeness, and diagnostics fields;
- unknown fields are ignored for compatible minor schema versions;
- schema validation can be run without executing the operation.

Canonical response envelope:

```json
{
  "schema": "missing-utils/response/v1",
  "operation": "check",
  "outcome": "pass",
  "changed": false,
  "complete": true,
  "data": {},
  "findings": [],
  "diagnostics": []
}
```

The `changed` field exists for Ansible integration but is meaningful only for commands that can render or update an explicitly named artifact. Inspection commands return `false`.

### 7.5 Ansible integration

- Publish a `missing_utils.core` collection with thin modules wrapping stable binary contracts.
- Modules accept native Ansible arguments, serialize a JSON request, invoke the binary without a shell, and return parsed data plus `changed`, `failed`, and diagnostics.
- Read-only modules support check mode fully. `envsub` and `varmerge` determine `changed` by comparing planned output without writing in check mode.
- Collection modules pin a minimum compatible binary/schema version and fail clearly on mismatches.
- Remote execution supports either a preinstalled binary or a controlled temporary transfer with checksum verification.
- Raw `ansible.builtin.command` remains supported for users who prefer direct invocation and `from_json` parsing.

### 7.6 Terraform/OpenTofu integration

- Prefer consuming `terraform show -json` after plan creation rather than scraping terminal output.
- Use `varmerge` or `envsub` during initialization hooks to create `terraform.tfvars.json` or other mounted configuration artifacts.
- Provide `--terraform-external` only for read-only operations. Terraform's external data-source protocol is string-map-in/string-map-out, so the adapter returns the full typed response as a JSON string under `result_json` plus small string summary fields.
- Never use the external data-source adapter for mutation: Terraform may invoke a data source repeatedly during refresh and expects no observable side effects.
- Recommend a first-class provider only if future use cases require typed Terraform lifecycle semantics that the external protocol cannot provide.

Example external response:

```json
{
  "outcome": "pass",
  "result_json": "{\"schema\":\"missing-utils/response/v1\",\"outcome\":\"pass\",\"data\":{}}"
}
```

### 7.7 Spacelift integration

- Ship a small runner-image layer and a checksum-verified installation recipe.
- Run `envsub`/`varmerge` in `before_init`, `tfchanges` after plan generation, and `jsonprobe`/`jsongate` in the appropriate before/after lifecycle hooks.
- Write full JSON reports to run artifacts where available and print only concise, redacted summaries to logs.
- Treat a nonzero hook exit as an execution gate only when the stack explicitly opts into that behavior.
- Keep Spacelift OPA/Rego policies native. Spacelift policies receive platform-defined JSON and are intentionally self-contained; suite tools should not claim they can inject arbitrary runtime files into policy evaluation.
- Generate documented finding codes that can be mirrored in Rego policies when an organization wants both local and platform-native enforcement.

## 8. Delivery roadmap

The estimates assume one primary engineer. Parallel work can shorten elapsed time but should not weaken shared-contract review.

### Milestone 0: Foundation — 1 week

- Initialize the Go module, command layout, linting, tests, release build, and CI.
- Define CLI conventions, exit codes, logging, cancellation, and structured errors.
- Publish v1 draft schemas for errors and evidence provenance.
- Add security policy, contribution guide, and supported-platform matrix.

**Exit criteria:** A skeleton command builds reproducibly on the initial OS matrix; release artifacts include checksums and provenance.

### Milestone 1: File and pipeline primitives — 4 to 6 weeks

- Implement `jwalk` MVP with traversal-race, permission, symlink-loop, and pathological-tree tests.
- Implement `envsub` MVP with schema validation, secret masking, and atomic output.
- Implement `hashsum` MVP with SHA-256/BLAKE3, concurrent streaming, create/verify modes, and change-during-read detection.
- Define and test the direct `jwalk` NDJSON to `hashsum` input contract.
- Benchmark `jwalk` against representative `find` workflows.
- Benchmark `hashsum` on SSD, rotational-disk, cached, and many-small-file workloads without assuming that more workers are always faster.
- Fuzz template parsing, `.env` parsing, manifest parsing, and path/output encoding.

**Exit criteria:** All three tools have stable v1 CLI/schema candidates, documented threat boundaries, package artifacts, and end-to-end examples. A `jwalk | hashsum` pipeline can create and verify a large manifest without unsafe filename handling or unbounded memory.

### Milestone 1A: Automation contracts and adapters — 3 to 4 weeks

- Freeze the v1 request, response, finding, diagnostic, and gate schemas.
- Implement `tfchanges` and `varmerge` MVPs.
- Implement `jsonprobe` one-shot checks and a minimal `jsongate`; defer the more general `jsondiff` until real desired/observed schemas are available.
- Publish the first Ansible collection modules for `envsub`, `varmerge`, `hashsum`, and `jsonprobe`.
- Publish tested examples for Terraform external read-only use and Spacelift lifecycle hooks.
- Add contract fixtures proving that logs never contaminate stdout JSON.

**Exit criteria:** One example workflow passes typed values through Ansible, creates a Terraform plan, evaluates plan facts in a Spacelift-compatible hook, and retains machine-readable results without shell parsing.

### Milestone 2: Runtime inventory on Linux — 4 to 5 weeks

- Implement Linux `ports`, including permission-degraded output and namespace fixtures.
- Implement Linux `pwatch`, process identity, child-tree tracking, metrics, and passive captures.
- Add explicit stack-capture adapters for one or two runtimes after safety review.
- Stress test PID reuse, rapid fork/exit storms, high listener counts, and backpressure.

**Exit criteria:** A responder can reliably identify the owner of a listener and observe its process tree without a daemon.

### Milestone 3: First causal diagnostics — 5 to 6 weeks

- Build `portwhy` on the `ports` evidence library.
- Build `accesswhy` for Unix mode bits, path traversal, ACLs, IDs/groups, mount flags, and capabilities; add LSM evidence incrementally.
- Build `patchwhy` for deleted/replaced executables and libraries plus package/service restart recommendations.
- Add `incidentsnap` preview using existing collectors.

**Exit criteria:** Each command produces a useful conclusion, cites evidence, and reports unknowns rather than guessing.

### Milestone 4: Cross-platform `ports` — 4 to 6 weeks

- Implement and test selected BSD adapters.
- Implement and test Solaris/illumos support against real systems or maintained CI runners.
- Publish a field-by-field platform capability matrix.
- Add compatibility fixtures preventing schema drift between backends.

**Exit criteria:** Supported systems pass the same black-box listener ownership contract, with documented differences.

### Milestone 5: Expanded explanation suite — incremental

Prioritize from field feedback:

1. `netwhy` and `expose`;
2. `servicewhy` using `pwatch` evidence;
3. `binarywhy` and `certwhy`;
4. `authwhy` and `sandboxdiff`;
5. `driftwhy` and a production-ready `incidentsnap`.

Each new command requires a problem statement, evidence matrix, privilege analysis, threat model update, and stable structured contract before implementation.

## 9. Testing and quality strategy

### 9.1 Test layers

- Unit tests for parsers, filters, thresholds, encoders, and redaction.
- Property and fuzz tests for untrusted templates, paths, environment files, and kernel-derived data.
- Golden tests for human output and versioned JSON/NDJSON schemas.
- Consumer-driven contract tests for Ansible module wrappers, Terraform's external string-map adapter, and Spacelift hook exit behavior.
- Contract tests run against every platform adapter.
- Namespace/container integration tests on Linux.
- Privilege matrix tests as root and unprivileged users.
- Race-heavy tests for changing filesystem trees and short-lived processes.
- Performance regression benchmarks for traversal, listener enumeration, and event throughput.

### 9.2 Required adversarial cases

- Filenames containing newlines, invalid UTF-8, control characters, and extremely long components.
- Symlink cycles, mount cycles, disappearing files, unreadable directories, and huge fan-out.
- Templates attempting command substitution, recursive expansion, malicious defaults, or secret disclosure through errors.
- Manifests containing duplicate paths, traversal paths, mixed encodings, unknown algorithms, altered digests, or files changing during hashing.
- PID reuse, process exit during collection, namespace changes, and fork bombs in bounded test environments.
- Sockets without visible owners and deliberately restricted `/proc` access.
- Output consumers that close the pipe early or cannot keep up.

## 10. Security model

- Read-only collection is the default across the suite.
- Privilege escalation is never automatic.
- Every active operation is opt-in and included in `--dry-run` output.
- Structured output is considered potentially sensitive because it can expose paths, commands, users, topology, and process metadata.
- Secret-bearing inputs are zeroed or allowed to become unreachable as soon as practical; they are never written to debug logs.
- Incident bundles use an allowlisted collector set, redaction policy, size limits, and a signed/hash manifest.
- Kernel-facing parsers and platform command fallbacks are treated as untrusted-input boundaries.
- Release artifacts use reproducible builds where possible, signed checksums, SBOMs, and build provenance.

## 11. Packaging and release strategy

- Individual binaries plus an optional `missing-utils` bundle package.
- Archives for common architectures; `.deb`, `.rpm`, and selected BSD/Solaris packages as support matures.
- Container image only for CI/demo use, since host-inspection tools need deliberate namespace and privilege configuration.
- Semantic versioning per command contract, with the suite release tracking the included command versions.
- JSON schema changes follow explicit compatibility rules; breaking schema changes require a new schema identifier.
- Experimental commands are labeled and excluded from long-term compatibility promises until promoted.

## 12. Success metrics

- `jwalk` materially reduces wall time or pipeline complexity on the published benchmark corpus.
- `envsub` catches missing/type-invalid deployment inputs before writing output and never reveals schema-marked secrets in diagnostics tests.
- `hashsum` produces reproducible manifests, detects in-flight file changes, and verifies multi-terabyte sets with bounded memory.
- Automation commands can round-trip their documented JSON contracts without stdout contamination, secret leakage, or type loss.
- `tfchanges` preserves destructive, replacement, sensitive, and unknown plan semantics across supported Terraform/OpenTofu plan-format versions.
- `ports` correctly maps listeners to owners across the supported privilege/platform matrix.
- `pwatch` retains correct process identity through PID reuse tests and bounds its own resource consumption.
- At least 90% of structured-output examples remain backward compatible across minor releases.
- Diagnostic commands identify and cite all evidence required for their conclusion or explicitly return `unknown`.
- Installation remains optional per utility, with no mandatory service or external network dependency.

## 13. Key risks and mitigations

| Risk | Mitigation |
|---|---|
| Scope expands into a monitoring or security platform | Require every command to have one narrow user promise and a daemon-free mode |
| Cross-platform parity is overstated | Publish field-level capability matrices and test each backend independently |
| Privilege gaps create misleading results | Make completeness first-class and use an explicit partial-result exit code |
| Process stack capture disrupts production | Passive default, declared adapter effects, opt-in policies, and dry runs |
| JSON contracts freeze too early | Incubate schemas, version every record, and use golden consumer tests |
| High-performance traversal becomes nondeterministic | Make ordering semantics explicit and provide deterministic mode when needed |
| Hash concurrency reduces throughput or overwhelms storage | Use bounded adaptive workers, expose tuning, and benchmark by storage workload |
| A checksum manifest is mistaken for authenticated provenance | Clearly separate hashing from signing and state the integrity boundary in output/docs |
| A generic JSON interface hides Terraform/Ansible semantic differences | Keep a canonical envelope plus thin, tested adapters that preserve each platform's native expectations |
| Terraform external integration is used for side effects | Restrict the adapter to read-only operations and document repeated refresh execution |
| Spacelift hook checks are confused with native policies | Keep hook gates and OPA/Rego policy integration explicitly separate |
| Secret masking creates false confidence | Document that output content may intentionally contain secrets; mask only diagnostics/audit surfaces |
| Utility names conflict with existing projects | Treat names as working names and perform registry/package/PATH collision review before Milestone 1 release |

## 14. Naming note

`envsub` is already used by at least one existing package, and `jwalk` is used by existing software and a Rust traversal library. `ports` and `hashsum` are highly generic. Before publishing binaries, the project should perform a naming review covering distribution repositories, Homebrew, language registries, GitHub, and common shell commands. The architecture and plan do not depend on retaining these names.

## 15. Immediate backlog

1. Confirm project name, license, target Go version, and initial Linux distributions.
2. Decide whether command names remain standalone or gain a collision-resistant prefix.
3. Write the shared CLI and structured-output contract as an RFC.
4. Define `jwalk/v1`, `envsub/v1`, and `hashsum/v1` JSON Schemas.
5. Create benchmark corpora and performance baselines before optimization.
6. Threat-model `envsub` secret handling, `hashsum` manifest verification, and `pwatch` active capture adapters.
7. Prototype Linux socket enumeration and process correlation before freezing the `ports` schema.
8. Establish access to representative BSD and Solaris/illumos test systems before promising release dates.
9. Implement Milestone 0 and open separate MVP issues for `jwalk`, `envsub`, and `hashsum`.
10. Build a reference pipeline showing Ansible JSON input, `terraform show -json`, `tfchanges`, and a Spacelift hook gate.

## 16. Open decisions

- Final project and binary names.
- License and governance model.
- Minimum supported kernel and distribution versions.
- Whether deterministic `jwalk` ordering is global or directory-local.
- The `envsub` schema file format and whether JSON Schema is accepted directly.
- Whether `hashsum` aggregate manifests use canonical JSON, sorted NDJSON, or both, and how optional signing composes with them.
- Which runtime stack adapters `pwatch` supports initially.
- Which BSD variants and Solaris/illumos versions qualify for first-class `ports` support.
- Whether a later optional privileged helper is justified for evidence collection, and how its protocol would be secured.
- Whether the Ansible collection installs binaries, requires them preinstalled, or supports both modes.
- Whether the canonical policy finding schema should align with SARIF in addition to the simpler streaming schema.
