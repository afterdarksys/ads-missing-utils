# Missing Utils

Small, composable command-line tools for DevOps, infrastructure automation, and security diagnostics.

The repository also includes eleven functional macOS-oriented research commands,
binary inspection, and portable manual tooling.
Their contracts and platform boundaries are documented in
[specs/MACOS_SECURITY_SUITE.md](specs/MACOS_SECURITY_SUITE.md).

`osx` is a Go-native macOS subsystem utility inspired by the command-dispatch
idea in `mac-cli`. It provides structured access to `defaults`, `caffeinate`,
power, network, DNS, security, updates, and volume controls; all state-changing
actions require `--apply`. Its contract is in [specs/OSX.md](specs/OSX.md).

> [!IMPORTANT]
> Commands are implemented at the bounded scopes listed below and remain unreleased. Building successfully does not establish production readiness.

Linux has excellent low-level tools for inspecting files, processes, services, networks, and security controls. What is often missing is a safe, structured way to combine that evidence—or to replace a fragile shell pipeline with a predictable command.

Missing Utils is a suite of focused utilities built around two ideas:

- make routine operational work fast, safe, and JSON-native;
- explain *why* a system behaves as it does, with evidence and explicit uncertainty.

## Command capabilities

The authoritative inventory is [docs/commands.json](docs/commands.json).
A [linear, 72-column version](docs/COMMANDS.txt) is available for speech and magnification.
Run `python3 scripts/update-capabilities.py` after changing command scope;
CI checks that the inventory covers every command and this matrix is current.

<!-- BEGIN GENERATED CAPABILITIES -->
This matrix describes implemented scope, not production certification. All commands are unreleased.

| Command | Scope | Boundary |
|---|---|---|
| `aaas` | Audit and explicitly revoke macOS Accessibility TCC grants. | macOS TCC audit; revocation requires explicit apply. |
| `accesswhy` | Report path mode and local identity evidence, with explicit access-control gaps. | Mode and local identity evidence; ACL and policy evaluation are incomplete. |
| `amtm` | Map AppleScript automation text into a static permission graph without execution. | Static AppleScript text analysis; no script execution. |
| `authwhy` | Report local account and group evidence, with explicit authentication-policy gaps. | Local account evidence; authentication policy is not evaluated. |
| `binarywhy` | Report executable metadata and a SHA-256 integrity observation. | Metadata and hash observation; no trust or malware verdict. |
| `binparse` | Read ELF, Mach-O, and PE structure safely using Go debug readers. | ELF, Mach-O and PE structure; does not execute input. |
| `bundleinfo` | Inventory `.app` and `.pkg` structure and metadata without executing bundle contents. | See command help for supported inputs and platform constraints. |
| `cded` | Detect sensitive clipboard patterns and explicitly drain unauthorized content. | macOS clipboard inspection; draining requires explicit apply. |
| `certwhy` | Inspect a live TLS peer certificate, negotiated connection, and validation result. | Live TLS connection; failed verification stays failed. |
| `clt` | Correlate hidden executables with identical content and optionally quarantine them. | See command help for supported inputs and platform constraints. |
| `driftwhy` | Capture a current file-content fingerprint for later drift comparison. | Current fingerprint only; needs prior evidence to establish drift. |
| `dtp` | Compare daemon executable trees with an HMAC-authenticated baseline. | See command help for supported inputs and platform constraints. |
| `envsub` | Render environment-backed templates with strict types, defaults, validation, and secret-aware diagnostics. | See command help for supported inputs and platform constraints. |
| `expose` | Report local listener exposure, with explicit firewall/reachability gaps. | Linux listeners; firewall and remote reachability are not evaluated. |
| `hashsum` | Compute SHA-256 and BLAKE3 concurrently and create verifiable manifests for large file sets. | See command help for supported inputs and platform constraints. |
| `homestate` | Index a home or mapped directory, report changes and moves, and safely repair or restore problematic names. | See command help for supported inputs and platform constraints. |
| `incidentsnap` | Capture a bounded read-only host and boot-context snapshot. | Host and boot context only; not a forensic acquisition. |
| `info2logic` | Convert GNU Info source into Logic Manual v1. | See command help for supported inputs and platform constraints. |
| `jsondiff` | Compare structured desired and observed state with JSON Pointer paths and machine-readable changes. | See command help for supported inputs and platform constraints. |
| `jsongate` | Convert JSON findings into consistent pass, deny, or approval-required decisions. | See command help for supported inputs and platform constraints. |
| `jsonprobe` | Run declarative TCP, HTTP, process, and filesystem readiness checks. | TCP, HTTP and file checks; process checks require Linux. No root-cause verdict. |
| `jwalk` | Walk large directory trees with regex, age, size, and type filters, emitting JSON Lines. | See command help for supported inputs and platform constraints. |
| `logic` | Read Logic Manual v1, roff man-page, and GNU Info source files. | See command help for supported inputs and platform constraints. |
| `man2logic` | Convert roff man source into Logic Manual v1. | See command help for supported inputs and platform constraints. |
| `meow` | Whimsical cat sound text transformer with translation, prefix, emphasis, and pitch controls. | See command help for supported inputs and platform constraints. |
| `metascore` | Score file metadata and extended-attribute-name changes against a saved baseline. | See command help for supported inputs and platform constraints. |
| `netwhy` | Collect DNS, default-route, and redacted proxy evidence for a destination. | DNS, route and proxy evidence; no reachability or firewall conclusion. |
| `osx` | Inspect and manage macOS subsystems through structured commands. | macOS; state changes require --apply. |
| `patchwhy` | Capture a current binary/library identity for restart analysis. | Current fingerprint only; no process mapping or restart recommendation. |
| `pcapwhy` | Turn offline PCAP/PCAPNG captures into bounded flow, VLAN, ICMP, and reset evidence. | Offline, bounded capture analysis; no live capture. |
| `pll` | Lint XML property lists and flag malformed or risky launchd logic. | XML property lists only; not a launchd runtime validator. |
| `ports` | Produce a Linux TCP/UDP listener map with best-effort process ownership. | Linux listener inventory; process ownership is best effort. |
| `portwhy` | Explain visible Linux listener ownership and report partial visibility. | Linux; reports partial process visibility. |
| `pwatch` | Passively sample Linux process state, RSS, and thread count. | Linux process sampling only. |
| `regocheck` | Detect a local OPA engine and report policy-evaluation readiness. | Detects OPA readiness only; does not evaluate policies. |
| `restartwhy` | Find visible processes still mapping deleted files after updates. | Linux visible deleted mappings only; no automatic restart. |
| `sandboxdiff` | Compare two JSON workload snapshots. | JSON comparison only; does not launch or observe a sandbox. |
| `selinuxwhy` | Summarize SELinux enforcement state and bounded AVC denial evidence. | Linux enforcement and AVC evidence; no policy changes. |
| `servicewhy` | Collect passive process-health evidence for a service PID. | Linux process evidence; no service dependency or restart analysis. |
| `spacelift-helper` | Normalize allowlisted Spacelift hook context. | Allowlisted environment context only; no hook execution or retention. |
| `tfchanges` | Normalize Terraform/OpenTofu plan JSON into concise resource-change records and evidence-based risk facts. | See command help for supported inputs and platform constraints. |
| `tjt` | Sample and report transient processes and deleted executable paths. | See command help for supported inputs and platform constraints. |
| `unitwhy` | Explain systemd unit state, selected hardening controls, and bounded journal evidence. | Linux systemd and bounded journal evidence; no mutation. |
| `varmerge` | Merge environment, JSON, YAML, and defaults under a typed schema with value provenance. | See command help for supported inputs and platform constraints. |
| `webkit-tool` | Statically flag risky WebKit settings and native script bridges. | Static heuristics; no exploitability verdict. |
<!-- END GENERATED CAPABILITIES -->

## JSON-first automation

Commands intended for automation will follow a shared contract:

- JSON or NDJSON on stdin and stdout;
- logs, progress, and human diagnostics on stderr;
- versioned schemas and stable exit codes;
- explicit `true`, `false`, and `unknown` conclusions;
- first-class completeness, provenance, findings, and diagnostics;
- secret masking on diagnostic surfaces;
- bounded memory and prompt cancellation.

Planned integrations include:

- thin Ansible collection modules with native `changed`, `failed`, and check-mode behavior;
- `terraform show -json`, `terraform.tfvars.json`, and a read-only external-data adapter that accepts and returns Terraform's required string maps, carrying richer typed output in a `result_json` string;
- `spacelift-helper` for lifecycle-hook context, reports, and gate results, while leaving native policy enforcement in OPA/Rego;
- `regocheck` for local Rego evaluation, fixture tests, schema checks, and coverage using a pinned OPA-compatible engine;
- ordinary shell pipelines, CI systems, and JSON-processing tools.

The Terraform external-data adapter will expose observations only. It will not modify infrastructure or local state because Terraform may invoke external data sources repeatedly during refresh. Native Ansible modules and Spacelift examples are planned deliverables, not currently published integrations.

For example, file discovery and manifest generation are designed to compose without unsafe filename parsing:

```sh
jwalk ./release --type file \
  | hashsum create --from-jwalk --root ./release --output release.hashes.json

hashsum verify --root ./release release.hashes.json
```

A Terraform plan pipeline could emit normalized changes and make an explicit gate decision:

```sh
terraform show -json tfplan \
  | tfchanges --format ndjson \
  | jsongate --policy deployment-gate.yaml
```

The file-manifest example is runnable with the v1 foundation below. The Terraform pipeline uses implemented commands; validate its gate policy against your organization’s requirements.

## Build, test, and command availability

`make build` and `./build.sh build` create every command in `cmd/` under `dist/`. The capability matrix above describes each command’s actual scope and limitations. Platform-specific collection can return partial evidence even when the binary builds successfully.

```sh
asdf install # if you use asdf; the project pins Go 1.24.6
make build
make man-build

# Or use the standalone build helper.
./build.sh build
./build.sh man-build
./build.sh man-install prefix="$HOME/.local"
./build.sh help
./build.sh install prefix="$HOME/.local"

./dist/portwhy --help # Linux listener evidence
./dist/jwalk ./release --type file --format ndjson
./dist/envsub --input app.yaml.tmpl --schema env.schema.yaml --output app.yaml
./dist/jwalk ./release --type file | ./dist/hashsum create --from-jwalk --root ./release --output release.hashes.json
./dist/hashsum verify --root ./release release.hashes.json
./dist/binparse ./dist/logic
./dist/man2logic man/logic.1 --output logic.logic
./dist/logic logic.logic
```

Run `make test` for unit tests, `make check` for static analysis plus tests, or `make help` for Make target help. `make man-build` (or `./build.sh man-build`) creates compressed pages in `dist/man`; `man-install` installs them below `PREFIX/share/man/man1`. The v1 specification and its acceptance criteria live in [specs/V1.md](specs/V1.md).

## Contributed companion tools

The `contrib/` directory links companion security utilities as Git submodules:

- [`contrib/codedetective`](contrib/codedetective) links [`afterdarksys/ads-codedetective`](https://github.com/afterdarksys/ads-codedetective).
- [`contrib/vibedetector`](contrib/vibedetector) links [`afterdarksys/vibedetector`](https://github.com/afterdarksys/vibedetector).

These tools cover adjacent AI-era application hygiene checks that are useful to run alongside Missing Utils without folding every scanner into this repository. Modern AI-assisted and rapidly generated applications often leak sensitive implementation details directly into browser-visible surfaces: bundled environment values, API keys, sourcemaps, backend route names, verbose build metadata, embedded prompts, model/provider configuration, and assumptions that should only exist server-side. The contributed tools are linked here so those checks can be pulled into a working tree when needed while still retaining their own release cycles, issue trackers, and implementation boundaries.

Clone this repository with submodules to fetch those tools at the recorded commits:

```sh
git clone --recurse-submodules https://github.com/afterdarksys/ads-missing-utils.git
```

For an existing checkout, initialize the contributed tools with:

```sh
make contrib
```

To refresh the contributed tools from their `main` branches and record the new commits in this repository, run:

```sh
make contrib-update
```

## Design principles

- **One narrow promise per command.** Each binary remains useful independently.
- **Read-only by default.** Active capture or mutation requires explicit authorization.
- **Evidence over guesses.** Important conclusions cite their source and report missing visibility.
- **Automation without shell parsing.** Structured output is a primary interface, not an afterthought.
- **Namespace awareness.** Linux process and network evidence includes container and namespace identity where possible.
- **Least privilege.** Partial visibility is reported clearly rather than silently omitted.
- **Portable core, honest adapters.** Platform-specific capabilities are tested and documented instead of assuming parity.
- **No mandatory daemon.** Normal operation uses standalone binaries.

## Roadmap

Development is planned in stages:

1. Shared CLI, schema, error, release, and security foundations.
2. File and pipeline primitives: `jwalk`, `envsub`, and `hashsum`.
3. Automation contracts and adapters: `tfchanges`, `varmerge`, `jsonprobe`, `jsongate`, `spacelift-helper`, and `regocheck`.
4. Linux runtime inventory: `ports` and `pwatch`.
5. First causal diagnostics: `portwhy`, `accesswhy`, and `patchwhy`.
6. BSD and Solaris-family `ports` backends.
7. `jsondiff` after concrete desired/observed schemas have proven its comparison model.
8. The remaining explanation and incident-response utilities, prioritized from field feedback.

See [PROJECT_PLAN.md](PROJECT_PLAN.md) for utility specifications, architecture, delivery milestones, testing strategy, security boundaries, risks, and open decisions.
The active completion tracker is [TODO](TODO). Source man pages live in
[man/](man/); [Logic Manual v1](docs/LOGIC_FORMAT.md) and
[binparse](docs/BINPARSE.md) have dedicated references. The evidence boundaries
for the enterprise diagnostic tools are documented in
[Enterprise diagnostics](docs/ENTERPRISE_DIAGNOSTICS.md); offline packet-capture
analysis is documented in [pcapwhy](docs/PCAPWHY.md).

## Current status

The repository contains implemented, unreleased commands with schemas, tests,
CI, and release configuration. The command matrix is the maintained inventory;
older planning documents describe historical milestones and broader intended scope.
Linux is the primary runtime diagnostic platform. Build portability does not imply
that all collectors work on every operating system.

The next milestone is validation of command contracts, packaging, and concrete
workflows. The [RCDO readiness workflow](docs/RCDO_WORKFLOW.md) connects jsonprobe
observations to accessible reviews with explicit coverage and freshness checks.

Names are provisional. Several proposed command names overlap with existing packages or use generic terms, so naming and package-registry collision checks will happen before the first release.

## Contributing

Early feedback is welcome through [GitHub Issues](https://github.com/afterdarksys/ads-missing-utils/issues), particularly from SREs, platform engineers, security engineers, incident responders, and infrastructure automation maintainers.

Useful contributions at this stage include:

- real operational workflows that currently require fragile pipelines;
- representative Terraform/OpenTofu plan cases;
- Ansible and Spacelift integration requirements;
- redacted Spacelift policy-input fixtures and Rego policy test cases;
- platform-specific behavior for Linux, BSD, Solaris, and illumos;
- security boundaries and failure modes the plan should address;
- benchmark corpora and reproducible test scenarios.

Before broader implementation begins, contribution guidelines, a code of conduct, and a security policy will be added.

## License

Missing Utils is available under the [MIT License](LICENSE).
