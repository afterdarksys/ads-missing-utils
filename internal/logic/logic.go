// Package logic implements Logic Manual v1, a portable, Git-friendly manual
// format. A Logic document consists of a small YAML front matter block followed
// by Markdown-like named sections.
package logic

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const Schema = "missing-utils/logic/v1"

type Document struct {
	Schema   string    `json:"schema"`
	Title    string    `json:"title"`
	Section  string    `json:"section,omitempty"`
	Summary  string    `json:"summary,omitempty"`
	Source   string    `json:"source,omitempty"`
	Sections []Section `json:"sections"`
}

type Section struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

// Encode writes the canonical Logic Manual v1 representation.
func Encode(w io.Writer, d Document) error {
	if d.Schema == "" {
		d.Schema = Schema
	}
	if d.Title == "" {
		return fmt.Errorf("Logic document requires a title")
	}
	if _, err := fmt.Fprintln(w, "---"); err != nil {
		return err
	}
	for _, pair := range [][2]string{{"logic", d.Schema}, {"title", d.Title}, {"section", d.Section}, {"summary", d.Summary}, {"source", d.Source}} {
		if pair[1] != "" {
			if _, err := fmt.Fprintf(w, "%s: %q\n", pair[0], pair[1]); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(w, "---"); err != nil {
		return err
	}
	for _, s := range d.Sections {
		if _, err := fmt.Fprintf(w, "\n# %s\n", strings.ToUpper(s.Name)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, strings.TrimSpace(s.Body)); err != nil {
			return err
		}
	}
	return nil
}

// Decode reads the deliberately small, portable Logic text grammar.
func Decode(r io.Reader) (Document, error) {
	b, err := io.ReadAll(io.LimitReader(r, 16<<20+1))
	if err != nil {
		return Document{}, err
	}
	if len(b) > 16<<20 {
		return Document{}, fmt.Errorf("Logic document exceeds 16 MiB limit")
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return Document{}, fmt.Errorf("Logic document must begin with YAML front matter")
	}
	d := Document{Schema: Schema}
	i := 1
	for ; i < len(lines) && lines[i] != "---"; i++ {
		parts := strings.SplitN(lines[i], ":", 2)
		if len(parts) != 2 {
			return Document{}, fmt.Errorf("invalid metadata line %q", lines[i])
		}
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch strings.TrimSpace(parts[0]) {
		case "logic":
			d.Schema = value
		case "title":
			d.Title = value
		case "section":
			d.Section = value
		case "summary":
			d.Summary = value
		case "source":
			d.Source = value
		}
	}
	if i == len(lines) || d.Schema != Schema || d.Title == "" {
		return Document{}, fmt.Errorf("invalid Logic document metadata")
	}
	var current *Section
	for _, line := range lines[i+1:] {
		if strings.HasPrefix(line, "# ") {
			d.Sections = append(d.Sections, Section{Name: strings.TrimSpace(line[2:])})
			current = &d.Sections[len(d.Sections)-1]
			continue
		}
		if current != nil {
			if current.Body != "" {
				current.Body += "\n"
			}
			current.Body += line
		}
	}
	if len(d.Sections) == 0 {
		return Document{}, fmt.Errorf("Logic document has no sections")
	}
	for i := range d.Sections {
		d.Sections[i].Body = strings.TrimSpace(d.Sections[i].Body)
	}
	return d, nil
}

var roffEscape = regexp.MustCompile(`\\(?:f[BRIP]|-|&|e|\(em|\(en)`)

// FromMan converts common roff man-page constructs without invoking a shell.
func FromMan(r io.Reader, source string) (Document, error) {
	d := Document{Schema: Schema, Source: source}
	var current *Section
	scanner := bufio.NewScanner(io.LimitReader(r, 16<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, ".TH ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				d.Title, d.Section = trimRoff(fields[1]), trimRoff(fields[2])
			}
			continue
		}
		if strings.HasPrefix(line, ".SH ") || strings.HasPrefix(line, ".SS ") {
			name := trimRoff(strings.TrimSpace(line[3:]))
			d.Sections = append(d.Sections, Section{Name: name})
			current = &d.Sections[len(d.Sections)-1]
			continue
		}
		if isInlineRoffMacro(line) {
			line = trimRoff(strings.TrimSpace(inlineRoffContent(line)))
			if line != "" && current != nil {
				if current.Body != "" {
					current.Body += "\n"
				}
				current.Body += line
			}
			continue
		}
		if strings.HasPrefix(line, ".") {
			continue
		}
		line = trimRoff(line)
		if line == "" || current == nil {
			continue
		}
		if current.Body != "" {
			current.Body += "\n"
		}
		current.Body += line
	}
	if err := scanner.Err(); err != nil {
		return Document{}, err
	}
	if d.Title == "" {
		return Document{}, fmt.Errorf("man source does not contain a .TH title")
	}
	if len(d.Sections) == 0 {
		d.Sections = []Section{{Name: "DESCRIPTION", Body: "No roff sections found."}}
	}
	for _, s := range d.Sections {
		if s.Name == "NAME" {
			d.Summary = strings.TrimSpace(strings.TrimPrefix(s.Body, d.Title+" -"))
			break
		}
	}
	return d, nil
}

// FromInfo converts a GNU Info source into a document while retaining its
// readable prose and headings. Node delimiters become sections.
func FromInfo(r io.Reader, source string) (Document, error) {
	b, err := io.ReadAll(io.LimitReader(r, 16<<20+1))
	if err != nil {
		return Document{}, err
	}
	if len(b) > 16<<20 {
		return Document{}, fmt.Errorf("Info source exceeds 16 MiB limit")
	}
	d := Document{Schema: Schema, Source: source}
	parts := bytes.Split(b, []byte{0x1f})
	for _, raw := range parts {
		text := strings.TrimSpace(string(raw))
		if text == "" {
			continue
		}
		lines := strings.Split(text, "\n")
		name := "DESCRIPTION"
		if strings.HasPrefix(lines[0], "File:") {
			for _, item := range strings.Split(lines[0], ",") {
				item = strings.TrimSpace(item)
				if strings.HasPrefix(item, "Node:") {
					name = strings.TrimSpace(strings.TrimPrefix(item, "Node:"))
				}
			}
			lines = lines[1:]
		}
		body := strings.TrimSpace(strings.Join(lines, "\n"))
		if body == "" {
			continue
		}
		if d.Title == "" {
			d.Title = name
			if d.Title == "Top" {
				d.Title = strings.TrimSuffix(source, ".info")
			}
		}
		d.Sections = append(d.Sections, Section{Name: name, Body: body})
	}
	if d.Title == "" {
		return Document{}, fmt.Errorf("Info source contains no readable nodes")
	}
	return d, nil
}

func trimRoff(s string) string {
	s = strings.Trim(s, ` \"`)
	s = strings.ReplaceAll(s, `"`, "")
	s = roffEscape.ReplaceAllStringFunc(s, func(v string) string {
		if v == `\-` || v == `\(en` {
			return "-"
		}
		return ""
	})
	return strings.Join(strings.Fields(s), " ")
}

func isInlineRoffMacro(line string) bool {
	for _, macro := range []string{".B ", ".I ", ".BR ", ".RB ", ".BI ", ".IB ", ".IR ", ".RI "} {
		if strings.HasPrefix(line, macro) {
			return true
		}
	}
	return false
}

func inlineRoffContent(line string) string {
	for _, macro := range []string{".B ", ".I ", ".BR ", ".RB ", ".BI ", ".IB ", ".IR ", ".RI "} {
		if strings.HasPrefix(line, macro) {
			return strings.TrimPrefix(line, macro)
		}
	}
	return line
}
