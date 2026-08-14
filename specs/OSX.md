# osx: macOS subsystem utility

`osx` is a Go replacement for the useful dispatcher idea in `mac-cli`, rather
than a port of its unsafe interactive shell snippets. It uses the form
`osx [options] <subsystem> <action> [arguments]` and produces versioned JSON.

## Included MVP

- Read-only: system information, battery and power settings, network quality
  and interfaces, DNS configuration, FileVault, Gatekeeper, firewall state, and
  available system updates.
- `defaults read DOMAIN [KEY]`, plus typed `write` and `delete` guarded by
  `--apply`.
- `caffeinate start [--duration DURATION] --apply`, using the native utility
  directly and keeping it in the foreground so its lifecycle remains visible.
- Spotlight search, LaunchServices opening, speech, pasteboard reads and writes, `ditto`
  copies, file-backed AppleScript execution, and SIP inspection.
- Explicitly guarded DNS flushing, software-update installation, and volume
  modification.

## Safety contract

The command never passes input through a shell. Read-only results include the
native invocation used. Mutating actions are rejected without `--apply`; no
command escalates privileges or makes an implicit remote request. It is macOS
only and reports a runtime error elsewhere.
