package meow

import (
	"bytes"
	"strings"
	"testing"
)

func TestTransformDefaults(t *testing.T) {
	opts := Options{
		InputText: "Hello world!",
		Seed:      12345,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Pitch != PitchNormal {
		t.Errorf("got pitch %s, want %s", res.Pitch, PitchNormal)
	}
	if res.Volume != VolumeNormal {
		t.Errorf("got volume %s, want %s", res.Volume, VolumeNormal)
	}
	if res.Output != "Hello world!" {
		t.Errorf("got output %q, want %q", res.Output, "Hello world!")
	}
}

func TestTransformTranslation(t *testing.T) {
	opts := Options{
		InputText: "Hello beautiful world!",
		Translate: true,
		Seed:      999,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "Hello" (5 chars -> titlecase medium meow), "beautiful" (9 chars -> titlecase long meow), "world!" (5 chars -> lowercase medium meow)
	if res.Output == "" || res.Output == "Hello beautiful world!" {
		t.Errorf("expected translated text, got %q", res.Output)
	}
	if !strings.Contains(strings.ToLower(res.Output), "meow") && !strings.Contains(strings.ToLower(res.Output), "nyan") && !strings.Contains(strings.ToLower(res.Output), "mew") {
		t.Errorf("expected cat sounds in translation output, got %q", res.Output)
	}
}

func TestTransformPrefixSuffix(t *testing.T) {
	opts := Options{
		InputText: "Line one\nLine two",
		Prefix:    true,
		Suffix:    true,
		Pitch:     PitchMew,
		Seed:      100,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedLines := 2
	if len(res.OutputLines) != expectedLines {
		t.Fatalf("got %d lines, want %d", len(res.OutputLines), expectedLines)
	}
	for _, l := range res.OutputLines {
		if !strings.HasPrefix(l, "mew: ") {
			t.Errorf("line %q missing prefix 'mew: '", l)
		}
		if !strings.HasSuffix(l, " ~ mew! 🐾") {
			t.Errorf("line %q missing suffix ' ~ mew! 🐾'", l)
		}
	}
}

func TestTransformVolumeLoud(t *testing.T) {
	opts := Options{
		InputText: "small mouse meow",
		Volume:    VolumeLoud,
		Seed:      42,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Output != "SMALL MOUSE MEOW" {
		t.Errorf("got %q, want %q", res.Output, "SMALL MOUSE MEOW")
	}
}

func TestTransformPitchHiss(t *testing.T) {
	opts := Options{
		InputText: "danger ahead",
		Translate: true,
		Pitch:     PitchHiss,
		Volume:    VolumeLoud,
		Seed:      123,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Output, "HISS") && !strings.Contains(res.Output, "KSSSS") && !strings.Contains(res.Output, "GRRR") && !strings.Contains(res.Output, "RAWR") && !strings.Contains(res.Output, "SHHH") {
		t.Errorf("expected angry hiss noises, got %q", res.Output)
	}
}

func TestTransformEmphasisAndKeyboardWalk(t *testing.T) {
	opts := Options{
		InputText:    "Keyboard testing",
		Emphasis:     true,
		KeyboardWalk: true,
		Frequency:    1.0,
		Seed:         777,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Output == "Keyboard testing" {
		t.Errorf("expected emphasis and keyboard walk changes, got %q", res.Output)
	}
}

func TestTransformInputStream(t *testing.T) {
	inputData := "First stream line\nSecond stream line\n"
	opts := Options{
		InputReader: bytes.NewBufferString(inputData),
		Prefix:      true,
		Pitch:       PitchPurr,
		Seed:        55,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.LinesProcessed != 2 {
		t.Errorf("got lines processed %d, want 2", res.LinesProcessed)
	}
	if !strings.HasPrefix(res.OutputLines[0], "purr: ") {
		t.Errorf("expected purr prefix, got %q", res.OutputLines[0])
	}
}

func TestAsciiArtOutput(t *testing.T) {
	opts := Options{
		InputText: "test ascii",
		ShowAscii: true,
		Pitch:     PitchMew,
	}
	res, err := Transform(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AsciiArt == "" || !strings.Contains(res.AsciiArt, "mew!") {
		t.Errorf("expected ASCII art with 'mew!', got %q", res.AsciiArt)
	}
}
