// Package binparse provides safe, read-only summaries of ELF, Mach-O, and PE
// binaries using Go's standard debug readers.
package binparse

import (
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const Schema = "missing-utils/binparse/v1"

type Report struct {
	Schema       string        `json:"schema"`
	Path         string        `json:"path"`
	Format       string        `json:"format"`
	Architecture string        `json:"architecture,omitempty"`
	ByteOrder    string        `json:"byte_order,omitempty"`
	Entry        uint64        `json:"entry,omitempty"`
	Sections     []Section     `json:"sections"`
	Symbols      int           `json:"symbols"`
	DWARF        *DWARFSummary `json:"dwarf,omitempty"`
}

type Section struct {
	Name    string `json:"name"`
	Address uint64 `json:"address"`
	Size    uint64 `json:"size"`
	Flags   string `json:"flags,omitempty"`
}
type DWARFSummary struct {
	CompilationUnits int `json:"compilation_units"`
}

func Parse(path string) (Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return Report{}, err
	}
	defer f.Close()
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return Report{}, fmt.Errorf("read binary header: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Report{}, err
	}
	switch {
	case magic[0] == 0x7f && string(magic[1:]) == "ELF":
		return parseELF(path)
	case magic[0] == 'M' && magic[1] == 'Z':
		return parsePE(path)
	case isMachMagic(binary.BigEndian.Uint32(magic)), isMachMagic(binary.LittleEndian.Uint32(magic)):
		return parseMach(path)
	default:
		return Report{}, fmt.Errorf("unsupported binary format (expected ELF, Mach-O, or PE)")
	}
}

func parseELF(path string) (Report, error) {
	f, err := elf.Open(path)
	if err != nil {
		return Report{}, err
	}
	defer f.Close()
	r := Report{Schema: Schema, Path: path, Format: "elf", Architecture: f.Machine.String(), Entry: f.Entry, ByteOrder: byteOrder(f.ByteOrder)}
	for _, s := range f.Sections {
		r.Sections = append(r.Sections, Section{Name: s.Name, Address: s.Addr, Size: s.Size, Flags: s.Flags.String()})
	}
	if symbols, err := f.Symbols(); err == nil {
		r.Symbols = len(symbols)
	}
	if d, err := f.DWARF(); err == nil {
		r.DWARF = summarizeDWARF(d)
	}
	return r, nil
}

func parseMach(path string) (Report, error) {
	f, err := macho.Open(path)
	if err == nil {
		defer f.Close()
		return reportMach(path, f)
	}
	fat, fatErr := macho.OpenFat(path)
	if fatErr != nil {
		return Report{}, err
	}
	defer fat.Close()
	if len(fat.Arches) == 0 {
		return Report{}, fmt.Errorf("Mach-O fat binary contains no architectures")
	}
	r, err := reportMach(path, fat.Arches[0].File)
	if err != nil {
		return Report{}, err
	}
	r.Architecture = "fat/" + r.Architecture
	return r, nil
}

func reportMach(path string, f *macho.File) (Report, error) {
	r := Report{Schema: Schema, Path: path, Format: "macho", Architecture: f.Cpu.String(), ByteOrder: byteOrder(f.ByteOrder)}
	for _, s := range f.Sections {
		r.Sections = append(r.Sections, Section{Name: s.Name, Address: s.Addr, Size: s.Size, Flags: fmt.Sprintf("%#x", s.Flags)})
	}
	if f.Symtab != nil {
		r.Symbols = len(f.Symtab.Syms)
	}
	if d, err := f.DWARF(); err == nil {
		r.DWARF = summarizeDWARF(d)
	}
	return r, nil
}

func parsePE(path string) (Report, error) {
	f, err := pe.Open(path)
	if err != nil {
		return Report{}, err
	}
	defer f.Close()
	r := Report{Schema: Schema, Path: path, Format: "pe", Architecture: fmt.Sprintf("%#x", f.FileHeader.Machine), ByteOrder: "little-endian"}
	for _, s := range f.Sections {
		r.Sections = append(r.Sections, Section{Name: s.Name, Address: uint64(s.VirtualAddress), Size: uint64(s.Size), Flags: fmt.Sprintf("%#x", s.Characteristics)})
	}
	if f.OptionalHeader != nil {
		switch h := f.OptionalHeader.(type) {
		case *pe.OptionalHeader32:
			r.Entry = uint64(h.AddressOfEntryPoint)
		case *pe.OptionalHeader64:
			r.Entry = uint64(h.AddressOfEntryPoint)
		}
	}
	r.Symbols = len(f.Symbols)
	return r, nil
}

func summarizeDWARF(d *dwarf.Data) *DWARFSummary {
	result := &DWARFSummary{}
	reader := d.Reader()
	for {
		entry, err := reader.Next()
		if err != nil || entry == nil {
			break
		}
		if entry.Tag == dwarf.TagCompileUnit {
			result.CompilationUnits++
		}
	}
	return result
}
func byteOrder(order binary.ByteOrder) string {
	if order == binary.LittleEndian {
		return "little-endian"
	}
	return "big-endian"
}
func isMachMagic(value uint32) bool {
	switch value {
	case macho.Magic32, macho.Magic64, macho.MagicFat:
		return true
	default:
		return false
	}
}
