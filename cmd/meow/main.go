package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/afterdarksys/ads-missing-utils/internal/cli"
	"github.com/afterdarksys/ads-missing-utils/internal/meow"
)

func main() { os.Exit(run()) }

func printUsage(w io.Writer) {
	fmt.Fprint(w, `meow - playful cat sound text transformer utility

USAGE:
  meow [FLAGS] [TEXT...]
  cat input.txt | meow [FLAGS]

DESCRIPTION:
  meow is a whimsical easter egg text transformation tool for Missing Utils.
  It reads standard input (or positional text / file inputs) and prefixes,
  suffixes, intersperses, or translates text into delightful cat sounds like
  "meow", "mew", "hiss", and "purr".

MODES:
  -t, --translate      Replaces words with cat noises matching length and punctuation.
  -P, --prefix         Adds a pitch-aware cat prefix (e.g. "meow: ") to every line.
  -e, --emphasis       Intersperses random purr sounds and letter casing jitters.
  -k, --keyboard-walk  Simulates a cat stepping across the keyboard (zoomies mode).
  -s, --suffix         Appends cat tails or purr sounds to the end of every line.

FLAGS:
  -h, --help           Show this manual.
  -v, --volume         Set loudness: "normal" or "loud" (converts output to ALL CAPS!).
  -p, --pitch          Cat voice style: "normal", "mew" (high), "hiss" (angry),
                       "purr" (sleepy), "rawr" (deep). Default: normal.
  -f, --frequency      Probability (0.0 - 1.0) of extra cat noise injection. Default: 0.3.
  -i, --input <file>   Read text from specified file instead of standard input.
      --format <fmt>   Output format: "text", "json", or "ndjson". Default: text.
      --ascii          Prepend a cute ASCII cat drawing to the output.
      --seed <int>     RNG seed for deterministic output (0 = random/timestamp).
      --version        Print version information and exit.

EXAMPLES:
  1. Translate standard input into cat speak:
     echo "Hello world from Missing Utils!" | meow -t

  2. Add angry hiss prefix and loud volume to alert logs:
     meow -P -p hiss -v "System error detected!"

  3. Add purr emphasis and cat keyboard walks:
     echo "Deploying to production" | meow -e -k -f 0.5

  4. Output structured JSON response:
     meow -t --format json "Paws on keyboard"
`)
}

func run() int {
	valueFlags := map[string]bool{
		"-i": true, "--input": true,
		"-p": true, "--pitch": true,
		"--volume": true,
		"-f": true, "--frequency": true,
		"-P": true, "--prefix-text": true,
		"-s": true, "--suffix-text": true,
		"--format": true, "--seed": true,
	}

	rawArgs := os.Args[1:]
	args := cli.ReorderInterspersed(rawArgs, valueFlags)

	fs := flag.NewFlagSet("meow", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { printUsage(os.Stderr) }

	var (
		help, helpLong    bool
		version           bool
		translate, tShort bool
		prefix, pfxShort  bool
		prefixText        string
		emphasis, eShort  bool
		keyWalk, kShort   bool
		suffix, sfxShort  bool
		suffixText        string
		ascii, asciiAlt   bool
		pitch, pitchShort string
		volStr          string
		vBool, volBool   bool
		freq              float64
		inputPath         string
		format            string
		seed              int64
	)

	fs.BoolVar(&help, "h", false, "Show help")
	fs.BoolVar(&helpLong, "help", false, "Show help")
	fs.BoolVar(&version, "version", false, "Print version")

	fs.BoolVar(&translate, "translate", false, "Enable translation mode")
	fs.BoolVar(&tShort, "t", false, "Enable translation mode")

	fs.BoolVar(&prefix, "prefix", false, "Enable prefix mode")
	fs.BoolVar(&pfxShort, "P", false, "Enable prefix mode")
	fs.StringVar(&prefixText, "prefix-text", "", "Custom prefix text")

	fs.BoolVar(&emphasis, "emphasis", false, "Enable emphasis mode")
	fs.BoolVar(&eShort, "e", false, "Enable emphasis mode")

	fs.BoolVar(&keyWalk, "keyboard-walk", false, "Enable cat keyboard-walk mode")
	fs.BoolVar(&kShort, "k", false, "Enable cat keyboard-walk mode")
	fs.BoolVar(&keyWalk, "zoomies", false, "Enable cat keyboard-walk mode")

	fs.BoolVar(&suffix, "suffix", false, "Enable suffix mode")
	fs.BoolVar(&sfxShort, "s", false, "Enable suffix mode")
	fs.StringVar(&suffixText, "suffix-text", "", "Custom suffix text")

	fs.StringVar(&pitch, "pitch", "normal", "Cat voice style")
	fs.StringVar(&pitchShort, "p", "", "Cat voice style")

	fs.StringVar(&volStr, "volume", "", "Volume level (normal, loud)")
	fs.BoolVar(&vBool, "v", false, "Loud volume (ALL CAPS)")
	fs.BoolVar(&volBool, "loud", false, "Loud volume (ALL CAPS)")

	fs.Float64Var(&freq, "frequency", 0.3, "Injection frequency (0.0-1.0)")
	fs.Float64Var(&freq, "f", 0.3, "Injection frequency (0.0-1.0)")

	fs.StringVar(&inputPath, "input", "", "Input file path")
	fs.StringVar(&inputPath, "i", "", "Input file path")

	fs.StringVar(&format, "format", "text", "Output format (text, json, ndjson)")
	fs.BoolVar(&ascii, "ascii", false, "Show ASCII cat art")
	fs.BoolVar(&asciiAlt, "cat", false, "Show ASCII cat art")
	fs.Int64Var(&seed, "seed", 0, "RNG seed")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			printUsage(os.Stdout)
			return cli.ExitOK
		}
		return cli.ExitUsage
	}

	if help || helpLong {
		printUsage(os.Stdout)
		return cli.ExitOK
	}

	if version {
		fmt.Println(cli.Version)
		return cli.ExitOK
	}

	// Resolve aliased flags
	isTranslate := translate || tShort
	isPrefix := prefix || pfxShort || prefixText != ""
	isEmphasis := emphasis || eShort
	isKeyWalk := keyWalk || kShort
	isSuffix := suffix || sfxShort || suffixText != ""
	showAscii := ascii || asciiAlt

	resolvedPitch := pitch
	if pitchShort != "" {
		resolvedPitch = pitchShort
	}

	resolvedVolume := meow.VolumeNormal
	if vBool || volBool || strings.EqualFold(volStr, "loud") || strings.EqualFold(volStr, "caps") || volStr == "true" {
		resolvedVolume = meow.VolumeLoud
	}

	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	format = strings.ToLower(format)
	if format != "text" && format != "json" && format != "ndjson" {
		fmt.Fprintln(os.Stderr, "error: --format must be 'text', 'json', or 'ndjson'")
		return cli.ExitUsage
	}

	var reader io.Reader
	var inputText string

	remainingArgs := fs.Args()
	if len(remainingArgs) > 0 {
		inputText = strings.Join(remainingArgs, " ")
	} else if inputPath != "" {
		f, err := os.Open(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening input file: %v\n", err)
			return cli.ExitFailure
		}
		defer f.Close()
		reader = f
	} else {
		// Check stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			reader = os.Stdin
		} else {
			// No stdin stream provided, use empty input text or cat sample if default
			inputText = "meow"
		}
	}

	opts := meow.Options{
		InputReader:  reader,
		InputText:    inputText,
		InputPath:    inputPath,
		Translate:    isTranslate,
		Prefix:       isPrefix,
		PrefixString: prefixText,
		Emphasis:     isEmphasis,
		KeyboardWalk: isKeyWalk,
		Suffix:       isSuffix,
		SuffixString: suffixText,
		Pitch:        resolvedPitch,
		Volume:       resolvedVolume,
		Frequency:    freq,
		Format:       format,
		Seed:         seed,
		ShowAscii:    showAscii,
	}

	res, err := meow.Transform(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error transforming text: %v\n", err)
		return cli.ExitRuntime
	}

	switch format {
	case "json":
		resp := cli.Response[*meow.Result]{
			Schema:  "missing-utils/meow/v1",
			Command: "meow",
			Outcome: "pass",
			Data:    res,
		}
		if err := cli.WriteJSON(os.Stdout, resp); err != nil {
			return cli.ExitRuntime
		}
	case "ndjson":
		for _, line := range res.OutputLines {
			lineRes := struct {
				Schema  string `json:"schema"`
				Command string `json:"command"`
				Line    string `json:"line"`
			}{
				Schema:  "missing-utils/meow/v1",
				Command: "meow",
				Line:    line,
			}
			if err := cli.WriteJSON(os.Stdout, lineRes); err != nil {
				return cli.ExitRuntime
			}
		}
	default:
		if res.AsciiArt != "" {
			fmt.Print(res.AsciiArt)
		}
		fmt.Println(res.Output)
	}

	return cli.ExitOK
}
