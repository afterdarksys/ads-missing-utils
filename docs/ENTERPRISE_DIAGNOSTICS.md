# Enterprise diagnostics

The explanation tools collect bounded evidence for common operator questions;
they do not apply remediation or claim visibility they do not have.

| Command | Evidence collected | Deliberate boundary |
| --- | --- | --- |
| `unitwhy` | systemd state, unit/drop-in paths, selected hardening controls, bounded journal tail | no service-manager mutation or journal interpretation beyond state evidence |
| `restartwhy` | visible Linux process mappings marked `(deleted)` | no process restart, package-manager interrogation, or invisible-process claim |
| `selinuxwhy` | enforcement state and bounded AVC audit records | no relabeling, policy changes, or `audit2allow` recommendation |
| `netwhy` | DNS answers, Linux default route, redacted proxy configuration | no remote connection, firewall, namespace, or reachability assertion |
| `certwhy` | verified TLS result and peer certificate metadata | unverified fallback is diagnostic-only and remains a failed validation result |
| `incidentsnap` | host, OS release, kernel, boot ID, and uptime when readable | not a forensic collection, archive, or integrity-verifiable evidence bundle |

All commands emit versioned JSON, retain standard exit status conventions, and
return partial results rather than hiding permission or platform gaps. This
makes them safe for incident triage, support bundles, CI evidence capture, and
configuration-management wrappers.
