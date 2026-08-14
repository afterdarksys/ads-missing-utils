package logic

import (
	"strings"
	"testing"
)

func TestManRoundTrip(t *testing.T) {
	d, err := FromMan(strings.NewReader(".TH DEMO 1\n.SH NAME\ndemo - useful test\n.SH DESCRIPTION\nA test utility.\n"), "demo.1")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := Encode(&out, d); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(strings.NewReader(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Title != "DEMO" || len(decoded.Sections) != 2 {
		t.Fatalf("unexpected decoded document: %#v", decoded)
	}
}

func TestFromInfo(t *testing.T) {
	d, err := FromInfo(strings.NewReader("\x1f\nFile: demo.info,  Node: Top\nDemo manual.\n"), "demo.info")
	if err != nil || d.Title != "demo" {
		t.Fatalf("document=%#v err=%v", d, err)
	}
}

func TestFromManKeepsInlineFormattingContent(t *testing.T) {
	d, err := FromMan(strings.NewReader(".TH DEMO 1\n.SH SYNOPSIS\n.B demo\n.RB [ \\-\\-flag ]\n"), "demo.1")
	if err != nil || d.Sections[0].Body != "demo\n[ --flag ]" {
		t.Fatalf("document=%#v err=%v", d, err)
	}
}
