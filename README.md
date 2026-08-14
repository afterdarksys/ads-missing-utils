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
> Seven roadmap commands (`jwalk`, `envsub`, `hashsum`, `tfchanges`, `varmerge`, `jsonprobe`, and `jsondiff`) are implemented in this repository but have not been released yet. The other 17 roadmap commands are clearly marked test scaffolds: they build, support `--help` and `--version`, and intentionally return a nonzero status for operational use until implemented.

Linux has excellent low-level tools for inspecting files, processes, services, networks, and security controls. What is often missing is a safe, structured way to combine that evidence—or to replace a fragile shell pipeline with a predictable command.

Missing Utils is a planned suite of 24 focused utilities built around two ideas:

- make routine operational work fast, safe, and JSON-native;
- explain *why* a system behaves as it does, with evidence and explicit uncertainty.

## Toolkit roadmap

### Operational primitives

| Command | Status | Purpose |
|---|---|---|
| `jwalk` | Implemented, unreleased | Walk large directory trees with regex, age, size, and type filters, emitting JSON Lines. |
| `envsub` | Implemented, unreleased | Render environment-backed templates with strict types, defaults, validation, and secret-aware diagnostics. |
| `hashsum` | Implemented, unreleased | Compute SHA-256 and BLAKE3 concurrently and create verifiable manifests for large file sets. |
| `meow` | Implemented, unreleased | Whimsical cat sound text transformer with translation, prefix, emphasis, and pitch controls. |
| `pwatch` | Implemented, unreleased | Passively sample Linux process state, RSS, and thread count. |
| `ports` | Implemented, unreleased | Produce a Linux TCP/UDP listener map with best-effort process ownership. |

### macOS filesystem and bundle research

| Command | Status | Purpose |
|---|---|---|
| `homestate` | Implemented, unreleased | Index a home or mapped directory, report changes and moves, and safely repair or restore problematic names. |
| `metascore` | Implemented, unreleased MVP | Score file metadata and extended-attribute-name changes against a saved baseline. |
| `pll` | Implemented, unreleased MVP | Lint XML property lists and flag malformed or risky launchd logic. |
| `bundleinfo` | Implemented, unreleased MVP | Inventory `.app` and `.pkg` structure and metadata without executing bundle contents. |
| `clt` | Implemented, unreleased MVP | Correlate hidden executables with identical content and optionally quarantine them. |
| `dtp` | Implemented, unreleased MVP | Compare daemon executable trees with an HMAC-authenticated baseline. |
| `tjt` | Implemented, unreleased MVP | Sample and report transient processes and deleted executable paths. |
| `cded` | Implemented, unreleased MVP | Detect sensitive clipboard patterns and explicitly drain unauthorized content. |
| `amtm` | Implemented, unreleased MVP | Map AppleScript automation text into a static permission graph without execution. |
| `aaas` | Implemented, unreleased MVP | Audit and explicitly revoke macOS Accessibility TCC grants. |
| `webkit-tool` | Implemented, unreleased MVP | Statically flag risky WebKit settings and native script bridges. |

### Causal diagnostics

| Command | Status | Question it answers |
|---|---|---|
| `portwhy` | Implemented, unreleased | Explain visible Linux listener ownership and report partial visibility. |
| `accesswhy` | Implemented, unreleased | Report path mode and local identity evidence, with explicit access-control gaps. |
| `patchwhy` | Implemented, unreleased | Capture a current binary/library identity for restart analysis. |
| `pcapwhy` | Implemented, unreleased | Turn offline PCAP/PCAPNG captures into bounded flow, VLAN, ICMP, and reset evidence. |
| `netwhy` | Implemented, unreleased | Collect DNS, default-route, and redacted proxy evidence for a destination. |
| `servicewhy` | Implemented, unreleased | Collect passive process-health evidence for a service PID. |
| `binarywhy` | Implemented, unreleased | Report executable metadata and a SHA-256 integrity observation. |
| `authwhy` | Implemented, unreleased | Report local account and group evidence, with explicit authentication-policy gaps. |
| `certwhy` | Implemented, unreleased | Inspect a live TLS peer certificate, negotiated connection, and validation result. |
| `expose` | Implemented, unreleased | Report local listener exposure, with explicit firewall/reachability gaps. |
| `sandboxdiff` | Implemented, unreleased | Compare two JSON workload snapshots. |
| `driftwhy` | Implemented, unreleased | Capture a current file-content fingerprint for later drift comparison. |
| `incidentsnap` | Implemented, unreleased | Capture a bounded read-only host and boot-context snapshot. |
| `unitwhy` | Implemented, unreleased | Explain systemd unit state, selected hardening controls, and bounded journal evidence. |
| `restartwhy` | Implemented, unreleased | Find visible processes still mapping deleted files after updates. |
| `selinuxwhy` | Implemented, unreleased | Summarize SELinux enforcement state and bounded AVC denial evidence. |

### Automation and infrastructure as code

| Command | Status | Purpose |
|---|---|---|
| `tfchanges` | Implemented, unreleased | Normalize Terraform/OpenTofu plan JSON into concise resource-change records and evidence-based risk facts. |
| `varmerge` | Implemented, unreleased | Merge environment, JSON, YAML, and defaults under a typed schema with value provenance. |
| `jsonprobe` | Implemented, unreleased | Run declarative TCP, HTTP, process, and filesystem readiness checks. |
| `jsondiff` | Implemented, unreleased | Compare structured desired and observed state with JSON Pointer paths and machine-readable changes. |
| `jsongate` | Implemented, unreleased | Convert JSON findings into consistent pass, deny, or approval-required decisions. |
| `spacelift-helper` | Implemented, unreleased | Normalize allowlisted Spacelift hook context. |
| `regocheck` | Implemented, unreleased | Detect a local OPA engine and report policy-evaluation readiness. |

### Binary and documentation tooling

| Command | Status | Purpose |
|---|---|---|
| `binparse` | Implemented, unreleased | Read ELF, Mach-O, and PE structure safely using Go debug readers. |
| `logic` | Implemented, unreleased | Read Logic Manual v1, roff man-page, and GNU Info source files. |
| `man2logic` | Implemented, unreleased | Convert roff man source into Logic Manual v1. |
| `info2logic` | Implemented, unreleased | Convert GNU Info source into Logic Manual v1. |

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

The file-manifest example is runnable with the v1 foundation below. The Terraform example remains a proposed contract for the planned automation/IaC track.

## Build, test, and command availability

`make build` and `./build.sh build` create all roadmap binaries in `dist/`. This is intentional: it lets package, installation, smoke-test, and shell-completion work exercise the complete command inventory. `jwalk`, `envsub`, `hashsum`, `tfchanges`, `varmerge`, `jsonprobe`, and `jsondiff` are functional unreleased commands; every other roadmap binary is a test scaffold and describes that status through `--help`.

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

./dist/portwhy --help # planned command scaffold
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

The repository contains a Go 1.24+ implementation of the v1 foundation plus four portable automation commands, their schemas, tests, CI, and release configuration. Linux is the primary platform; the file/configuration and automation commands also build on macOS and Windows. The 17 scaffold commands compile on those platforms for packaging and integration testing; their final platform support is described in the project plan.

The next milestone is an unreleased v1 validation period for command contracts and packaging, followed by the automation/IaC track.

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
