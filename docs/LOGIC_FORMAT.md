# Logic Manual v1

Logic is a portable, Git-friendly manual format. It carries the small amount of
metadata that manuals need while using ordinary text for the material people
read and review. Its filename extension is `.logic`.

## Format

A document begins with YAML front matter, followed by named Markdown-like
sections. The required fields are `logic: "missing-utils/logic/v1"` and
`title`. `section`, `summary`, and `source` are optional.

```text
---
logic: "missing-utils/logic/v1"
title: "binparse"
section: "1"
summary: "inspect executable-file structure safely"
source: "man/binparse.1"
---

# SYNOPSIS
binparse [--format json|text] BINARY

# DESCRIPTION
Reads executable metadata without executing the target.
```

The supported parser intentionally accepts only scalar quoted metadata and
level-one section headings. This makes the format deterministic, bounded, and
easy to implement outside Go. A Logic document is capped at 16 MiB.

## Commands

- `logic SOURCE` reads `.logic`, roff man, or GNU Info source and renders it as
  readable text; `--format json` emits the shared document model.
- `man2logic MAN_SOURCE` converts a roff source using common `.TH`, `.SH`, and
  `.SS` macros. It never executes the source or invokes a shell.
- `info2logic INFO_SOURCE` converts GNU Info nodes into sections.

The converters preserve prose and document the original path in `source`; they
are intentionally conservative and do not claim exact typographic preservation
of every roff macro or Info cross-reference.
