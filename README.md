# Missing Utils

Small, composable command-line tools for DevOps, infrastructure automation, and security diagnostics.

> [!IMPORTANT]
> Missing Utils is currently in the planning stage. The interfaces and command names described below are proposals; binaries have not been released yet.

Linux has excellent low-level tools for inspecting files, processes, services, networks, and security controls. What is often missing is a safe, structured way to combine that evidence—or to replace a fragile shell pipeline with a predictable command.

Missing Utils is a planned suite of 24 focused utilities built around two ideas:

- make routine operational work fast, safe, and JSON-native;
- explain *why* a system behaves as it does, with evidence and explicit uncertainty.

## What is planned

### Operational primitives

| Command | Purpose |
|---|---|
| `jwalk` | Walk large directory trees with regex, age, size, and type filters, emitting JSON Lines. |
| `envsub` | Render environment-backed templates with strict types, defaults, validation, and secret-aware diagnostics. |
| `hashsum` | Compute SHA-256 and BLAKE3 concurrently and create verifiable manifests for large file sets. |
| `pwatch` | Track a process and its descendants, detect resource spikes, and capture opt-in diagnostics. |
| `ports` | Produce a uniform process-to-listener map across Linux, BSD, and Solaris-family systems. |

### Causal diagnostics

| Command | Question it answers |
|---|---|
| `portwhy` | Who owns this port, how was it launched, and where is it reachable? |
| `accesswhy` | Why can or cannot an identity access this filesystem object? |
| `patchwhy` | Which processes still use replaced code, and what must restart? |
| `netwhy` | Which DNS, route, namespace, proxy, and firewall decisions affect this connection? |
| `servicewhy` | Why is this service unhealthy, restarting, or blocked? |
| `binarywhy` | Where did this executable come from, and has it changed? |
| `authwhy` | Which account, group, SSH, PAM, NSS, or policy rule affects login? |
| `certwhy` | Which certificate and trust path are in use, and why does validation fail? |
| `expose` | Which services are reachable from each interface or namespace? |
| `sandboxdiff` | How do two workloads differ in isolation and attack surface? |
| `driftwhy` | What security-relevant state changed, and what mechanism changed it? |
| `incidentsnap` | How can a responder collect a redacted, integrity-verifiable host snapshot? |

### Automation and infrastructure as code

| Command | Purpose |
|---|---|
| `tfchanges` | Normalize Terraform/OpenTofu plan or state JSON into concise resource-change records and risk facts. |
| `varmerge` | Merge environment, JSON, YAML, and defaults under a typed schema with value provenance. |
| `jsonprobe` | Run declarative DNS, TCP, TLS, HTTP, process, port, and filesystem readiness checks. |
| `jsondiff` | Compare structured desired and observed state with typed paths and machine-readable changes. |
| `jsongate` | Convert findings into consistent pass, warn, deny, or approval-required decisions. |
| `spacelift-helper` | Normalize Spacelift hook context, run suite checks, and preserve redacted JSON reports and gate results. |
| `regocheck` | Evaluate and test OPA/Rego policies locally against versioned Terraform and Spacelift JSON fixtures. |

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
  | hashsum create --from-jwalk --output release.hashes.json

hashsum verify release.hashes.json --root ./release
```

A Terraform plan pipeline could emit normalized changes and make an explicit gate decision:

```sh
terraform show -json tfplan \
  | tfchanges --format ndjson \
  | jsongate --policy deployment-gate.yaml
```

These examples describe the proposed command contracts and are not yet runnable.

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

## Current status

The repository currently contains planning documentation only. The proposed implementation is Go, producing dependency-light standalone binaries. Linux is the primary platform; portable utilities will target additional operating systems, and `ports` has explicit BSD and Solaris/illumos ambitions. Exact minimum versions will be published in a tested support matrix.

The next milestone is to establish the Go module, shared JSON contracts, CI, release process, supported-platform matrix, and initial command skeletons.

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

Before implementation begins, contribution guidelines, a code of conduct, and a security policy will be added.

## License

No open-source license has been selected yet. Until a license is published, the repository remains unlicensed and no reuse rights are granted beyond those provided by applicable law.
