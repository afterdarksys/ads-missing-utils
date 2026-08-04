package meow

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"unicode"
)

// Pitch definitions
const (
	PitchNormal = "normal"
	PitchMew    = "mew"
	PitchHiss   = "hiss"
	PitchPurr   = "purr"
	PitchRawr   = "rawr"
)

// Volume definitions
const (
	VolumeNormal = "normal"
	VolumeLoud   = "loud"
)

// Options configures the meow text transformation engine.
type Options struct {
	InputReader  io.Reader
	InputText    string
	InputPath    string
	Mode         string  // "translate", "prefix", "emphasis", "default"
	Translate    bool
	Prefix       bool
	PrefixString string
	Suffix       bool
	SuffixString string
	Emphasis     bool
	KeyboardWalk bool
	Pitch        string  // normal, mew, hiss, purr, rawr
	Volume       string  // normal, loud
	Frequency    float64 // 0.0 - 1.0 (default 0.3)
	Format       string  // text, json, ndjson
	Seed         int64
	ShowAscii    bool
}

// Result holds structured output data for JSON/NDJSON formats.
type Result struct {
	Mode           string   `json:"mode"`
	Pitch          string   `json:"pitch"`
	Volume         string   `json:"volume"`
	Frequency      float64  `json:"frequency"`
	LinesProcessed int      `json:"lines_processed"`
	Output         string   `json:"output"`
	OutputLines    []string `json:"output_lines,omitempty"`
	AsciiArt       string   `json:"ascii_art,omitempty"`
}

// PitchVocabulary defines cat sounds for a specific pitch.
type PitchVocabulary struct {
	Short    []string
	Medium   []string
	Long     []string
	Purrs    []string
	Prefix   string
	Suffix   string
	Walks    []string
}

var vocabularies = map[string]PitchVocabulary{
	PitchNormal: {
		Short:  []string{"mew", "mia", "nyan", "meow"},
		Medium: []string{"meow", "mewow", "mrroow", "nyan~"},
		Long:   []string{"meoooow", "mrrroooow", "meow-meow", "nyan-nyan"},
		Purrs:  []string{"*purrrr*", "*prrr-prrr*", "*mrrr*"},
		Prefix: "meow: ",
		Suffix: " ~ meow!",
		Walks:  []string{"asdfghjkl;", "zxcvbnm,", "qwertyuiop", "1234567890-="},
	},
	PitchMew: {
		Short:  []string{"mew", "miu", "mewp", "nyan"},
		Medium: []string{"mewmew", "mew~", "nyan~", "miuuu"},
		Long:   []string{"mewmewmew", "nyanyanya", "mewwwww"},
		Purrs:  []string{"*purrr~*", "*purrrrr~*", "*soft purr*"},
		Prefix: "mew: ",
		Suffix: " ~ mew! 🐾",
		Walks:  []string{"asdfgh", "qwert", "zxcv", "12321"},
	},
	PitchHiss: {
		Short:  []string{"hiss", "shh", "kss", "rawr"},
		Medium: []string{"kssss", "hiss!", "grrr", "SHHH"},
		Long:   []string{"HISS!!!!!", "kssssssss", "GRRRRRR!"},
		Purrs:  []string{"*grrrrr*", "*angry growl*", "*low rumble*"},
		Prefix: "HISS: ",
		Suffix: " !!! KSSSS!",
		Walks:  []string{"!@#$%^&*", "AAAAAAA", "///////", "FJDKSL;"},
	},
	PitchPurr: {
		Short:  []string{"purr", "prrr", "mrrr", "zZZz"},
		Medium: []string{"purrrr", "prrrrr", "zZZzzZ", "mrrr-mrrr"},
		Long:   []string{"purrrrrrrrr", "prrrr-prrr-prrr", "zZZzZZZz"},
		Purrs:  []string{"*purrrrrrrrr*", "*happy purrr*", "*rumbles softly*"},
		Prefix: "purr: ",
		Suffix: " ... purrrr~ 🐾",
		Walks:  []string{"zzzzzz", "mmmmmm", ".....", "---"},
	},
	PitchRawr: {
		Short:  []string{"MEOW", "RAWR", "ROAR", "MRR"},
		Medium: []string{"MEOWWW", "GROAAR", "MRRRR", "RAWWR!"},
		Long:   []string{"MEOOOOOWWW!", "GROAAAAAR!", "RAWRRRRR!"},
		Purrs:  []string{"*PRRRRRRR*", "*HEAVY PURR*", "*LOUD RUMBLE*"},
		Prefix: "RAWR: ",
		Suffix: " !!! MEOW!!",
		Walks:  []string{"QWERTYUIOP", "ASDFGHJKL", "ZXCVBNM", "99999999"},
	},
}

// AsciiArt returns a cute ASCII cat.
func AsciiArt(pitch string) string {
	switch pitch {
	case PitchHiss:
		return ` (/\_/\)
 ( >_< )
 (  "  )  HISS!
`
	case PitchPurr:
		return ` ( - . - )
 (  "  )  ... purrrr~
`
	case PitchMew:
		return ` (,,>  <,,)
  (  "  )  mew!
`
	case PitchRawr:
		return ` /\_/\
 ( o.o )  RAWR!
  > ^ <
`
	default:
		return ` /\_/\
( o.o )  meow!
 > ^ <
`
	}
}

// Transform accepts an Options struct and transforms input text/stream according to configured modes.
func Transform(opts Options) (*Result, error) {
	pitch := strings.ToLower(opts.Pitch)
	if pitch == "" {
		pitch = PitchNormal
	}
	vocab, ok := vocabularies[pitch]
	if !ok {
		vocab = vocabularies[PitchNormal]
		pitch = PitchNormal
	}

	volume := strings.ToLower(opts.Volume)
	if volume == "" {
		volume = VolumeNormal
	}

	freq := opts.Frequency
	if freq <= 0 {
		freq = 0.3
	} else if freq > 1.0 {
		freq = 1.0
	}

	var rng *rand.Rand
	if opts.Seed != 0 {
		rng = rand.New(rand.NewSource(opts.Seed))
	} else {
		rng = rand.New(rand.NewSource(42)) // default fixed seed for reproducible behavior if unspecified
	}

	var lines []string

	if opts.InputText != "" {
		lines = strings.Split(strings.ReplaceAll(opts.InputText, "\r\n", "\n"), "\n")
	} else if opts.InputReader != nil {
		scanner := bufio.NewScanner(opts.InputReader)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error reading input stream: %w", err)
		}
	} else {
		lines = []string{""}
	}

	outLines := make([]string, 0, len(lines))

	for _, line := range lines {
		transformed := transformLine(line, opts, vocab, volume, freq, rng)
		outLines = append(outLines, transformed)
	}

	ascii := ""
	if opts.ShowAscii {
		ascii = AsciiArt(pitch)
	}

	modeStr := deriveModeString(opts)
	outputStr := strings.Join(outLines, "\n")

	return &Result{
		Mode:           modeStr,
		Pitch:          pitch,
		Volume:         volume,
		Frequency:      freq,
		LinesProcessed: len(lines),
		Output:         outputStr,
		OutputLines:    outLines,
		AsciiArt:       ascii,
	}, nil
}

func deriveModeString(opts Options) string {
	modes := make([]string, 0, 4)
	if opts.Translate {
		modes = append(modes, "translate")
	}
	if opts.Prefix {
		modes = append(modes, "prefix")
	}
	if opts.Emphasis {
		modes = append(modes, "emphasis")
	}
	if opts.KeyboardWalk {
		modes = append(modes, "keyboard_walk")
	}
	if opts.Suffix {
		modes = append(modes, "suffix")
	}
	if len(modes) == 0 {
		return "default"
	}
	return strings.Join(modes, "+")
}

func transformLine(line string, opts Options, vocab PitchVocabulary, volume string, freq float64, rng *rand.Rand) string {
	res := line

	// 1. Translation Mode
	if opts.Translate {
		res = translateWords(res, vocab, rng)
	}

	// 2. Keyboard Walk Mode
	if opts.KeyboardWalk {
		res = injectKeyboardWalk(res, vocab, freq, rng)
	}

	// 3. Emphasis Mode
	if opts.Emphasis {
		res = injectEmphasis(res, vocab, freq, rng)
	}

	// 4. Prefix Mode
	if opts.Prefix {
		pfx := vocab.Prefix
		if opts.PrefixString != "" {
			pfx = opts.PrefixString
		}
		res = pfx + res
	}

	// 5. Suffix Mode
	if opts.Suffix {
		sfx := vocab.Suffix
		if opts.SuffixString != "" {
			sfx = opts.SuffixString
		}
		res = res + sfx
	}

	// 6. Volume Control (Loud / Caps)
	if volume == VolumeLoud {
		res = strings.ToUpper(res)
	}

	return res
}

func translateWords(text string, vocab PitchVocabulary, rng *rand.Rand) string {
	if text == "" {
		return vocab.Short[rng.Intn(len(vocab.Short))]
	}

	var sb strings.Builder
	var word strings.Builder

	flushWord := func() {
		if word.Len() == 0 {
			return
		}
		w := word.String()
		word.Reset()

		var sound string
		runeCount := len([]rune(w))
		if runeCount <= 3 {
			sound = vocab.Short[rng.Intn(len(vocab.Short))]
		} else if runeCount <= 6 {
			sound = vocab.Medium[rng.Intn(len(vocab.Medium))]
		} else {
			sound = vocab.Long[rng.Intn(len(vocab.Long))]
		}

		// Match original casing style
		if isUpper(w) {
			sound = strings.ToUpper(sound)
		} else if isTitle(w) {
			sound = titleCase(sound)
		}

		sb.WriteString(sound)
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
		} else {
			flushWord()
			sb.WriteRune(r)
		}
	}
	flushWord()

	return sb.String()
}

func injectKeyboardWalk(text string, vocab PitchVocabulary, freq float64, rng *rand.Rand) string {
	if rng.Float64() > freq {
		return text
	}
	walk := vocab.Walks[rng.Intn(len(vocab.Walks))]
	if text == "" {
		return walk
	}
	// Insert walk at random position or end
	pos := rng.Intn(len(text) + 1)
	return text[:pos] + " " + walk + " " + text[pos:]
}

func injectEmphasis(text string, vocab PitchVocabulary, freq float64, rng *rand.Rand) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		if rng.Float64() <= freq {
			return vocab.Purrs[rng.Intn(len(vocab.Purrs))]
		}
		return text
	}

	var result []string
	for _, w := range words {
		if rng.Float64() <= freq/2 {
			// Apply case jitter to word
			w = caseJitter(w, rng)
		}
		result = append(result, w)
		if rng.Float64() <= freq {
			purr := vocab.Purrs[rng.Intn(len(vocab.Purrs))]
			result = append(result, purr)
		}
	}
	return strings.Join(result, " ")
}

func caseJitter(s string, rng *rand.Rand) string {
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsLetter(r) {
			if rng.Float64() > 0.5 {
				runes[i] = unicode.ToUpper(r)
			} else {
				runes[i] = unicode.ToLower(r)
			}
		}
	}
	return string(runes)
}

func isUpper(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}

func isTitle(s string) bool {
	runes := []rune(s)
	if len(runes) == 0 {
		return false
	}
	return unicode.IsUpper(runes[0])
}

func titleCase(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	return string(unicode.ToUpper(runes[0])) + string(runes[1:])
}
