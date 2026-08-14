# binparse

`binparse` is a read-only executable metadata inspector for ELF, Mach-O, and
Portable Executable files. It uses Go standard-library readers (`debug/elf`,
`debug/macho`, `debug/pe`, `debug/dwarf`, and `encoding/binary`) and does not
load, map for execution, or run the input binary.

```sh
binparse /usr/bin/ssh
binparse --format text ./dist/logic
```

Its versioned JSON report contains the detected format, architecture, byte
order, entry point where applicable, section table, symbol count, and a DWARF
compilation-unit count when readable debug information is present. Parsing is
metadata inspection, not a malware verdict, signature verification, or a
security sandbox.
