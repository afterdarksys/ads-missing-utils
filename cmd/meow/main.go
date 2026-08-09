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

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

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

type commandConfig struct {
	options   meow.Options
	inputPath string
	format    string
	arguments []string
	help      bool
	version   bool
}

type rawCommandOptions struct {
	help, helpLong, version bool
	translate, tShort       bool
	prefix, pfxShort        bool
	prefixText              string
	emphasis, eShort        bool
	keyWalk, kShort         bool
	suffix, sfxShort        bool
	suffixText              string
	ascii, asciiAlt         bool
	pitch, pitchShort       string
	volStr                  string
	vBool, volBool          bool
	freq                    float64
	inputPath, format       string
	seed                    int64
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	config, err := parseCommand(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return cli.ExitUsage
	}
	if config.help {
		printUsage(stdout)
		return cli.ExitOK
	}
	if config.version {
		fmt.Fprintln(stdout, cli.Version)
		return cli.ExitOK
	}

	res, closer, err := transform(config, stdin)
	if closer != nil {
		defer closer.Close()
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return cli.ExitCode(err)
	}
	if err := writeOutput(stdout, config.format, res); err != nil {
		fmt.Fprintf(stderr, "error writing output: %v\n", err)
		return cli.ExitRuntime
	}
	return cli.ExitOK
}

func parseCommand(rawArgs []string, stderr io.Writer) (commandConfig, error) {
	valueFlags := map[string]bool{
		"-i": true, "--input": true,
		"-p": true, "--pitch": true,
		"--volume": true,
		"-f":       true, "--frequency": true,
		"-P": true, "--prefix-text": true,
		"-s": true, "--suffix-text": true,
		"--format": true, "--seed": true,
	}

	args := cli.ReorderInterspersed(rawArgs, valueFlags)

	fs := flag.NewFlagSet("meow", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printUsage(stderr) }

	var raw rawCommandOptions
	bindFlags(fs, &raw)

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return commandConfig{help: true}, nil
		}
		return commandConfig{}, err
	}

	return resolveCommand(raw, fs.Args())
}

func bindFlags(fs *flag.FlagSet, raw *rawCommandOptions) {
	fs.BoolVar(&raw.help, "h", false, "Show help")
	fs.BoolVar(&raw.helpLong, "help", false, "Show help")
	fs.BoolVar(&raw.version, "version", false, "Print version")
	fs.BoolVar(&raw.translate, "translate", false, "Enable translation mode")
	fs.BoolVar(&raw.tShort, "t", false, "Enable translation mode")
	fs.BoolVar(&raw.prefix, "prefix", false, "Enable prefix mode")
	fs.BoolVar(&raw.pfxShort, "P", false, "Enable prefix mode")
	fs.StringVar(&raw.prefixText, "prefix-text", "", "Custom prefix text")
	fs.BoolVar(&raw.emphasis, "emphasis", false, "Enable emphasis mode")
	fs.BoolVar(&raw.eShort, "e", false, "Enable emphasis mode")
	fs.BoolVar(&raw.keyWalk, "keyboard-walk", false, "Enable cat keyboard-walk mode")
	fs.BoolVar(&raw.kShort, "k", false, "Enable cat keyboard-walk mode")
	fs.BoolVar(&raw.keyWalk, "zoomies", false, "Enable cat keyboard-walk mode")
	fs.BoolVar(&raw.suffix, "suffix", false, "Enable suffix mode")
	fs.BoolVar(&raw.sfxShort, "s", false, "Enable suffix mode")
	fs.StringVar(&raw.suffixText, "suffix-text", "", "Custom suffix text")
	fs.StringVar(&raw.pitch, "pitch", "normal", "Cat voice style")
	fs.StringVar(&raw.pitchShort, "p", "", "Cat voice style")
	fs.StringVar(&raw.volStr, "volume", "", "Volume level (normal, loud)")
	fs.BoolVar(&raw.vBool, "v", false, "Loud volume (ALL CAPS)")
	fs.BoolVar(&raw.volBool, "loud", false, "Loud volume (ALL CAPS)")
	fs.Float64Var(&raw.freq, "frequency", 0.3, "Injection frequency (0.0-1.0)")
	fs.Float64Var(&raw.freq, "f", 0.3, "Injection frequency (0.0-1.0)")
	fs.StringVar(&raw.inputPath, "input", "", "Input file path")
	fs.StringVar(&raw.inputPath, "i", "", "Input file path")
	fs.StringVar(&raw.format, "format", "text", "Output format (text, json, ndjson)")
	fs.BoolVar(&raw.ascii, "ascii", false, "Show ASCII cat art")
	fs.BoolVar(&raw.asciiAlt, "cat", false, "Show ASCII cat art")
	fs.Int64Var(&raw.seed, "seed", 0, "RNG seed")
}

func resolveCommand(raw rawCommandOptions, arguments []string) (commandConfig, error) {
	format := strings.ToLower(raw.format)
	if format != "text" && format != "json" && format != "ndjson" {
		return commandConfig{}, fmt.Errorf("--format must be 'text', 'json', or 'ndjson'")
	}
	pitch := raw.pitch
	if raw.pitchShort != "" {
		pitch = raw.pitchShort
	}
	volume := meow.VolumeNormal
	if raw.vBool || raw.volBool || strings.EqualFold(raw.volStr, "loud") || strings.EqualFold(raw.volStr, "caps") || raw.volStr == "true" {
		volume = meow.VolumeLoud
	}

	return commandConfig{
		options: meow.Options{
			InputPath:    raw.inputPath,
			Translate:    raw.translate || raw.tShort,
			Prefix:       raw.prefix || raw.pfxShort || raw.prefixText != "",
			PrefixString: raw.prefixText,
			Emphasis:     raw.emphasis || raw.eShort,
			KeyboardWalk: raw.keyWalk || raw.kShort,
			Suffix:       raw.suffix || raw.sfxShort || raw.suffixText != "",
			SuffixString: raw.suffixText,
			Pitch:        pitch,
			Volume:       volume,
			Frequency:    raw.freq,
			Format:       format,
			Seed:         raw.seed,
			ShowAscii:    raw.ascii || raw.asciiAlt,
		},
		inputPath: raw.inputPath,
		format:    format,
		arguments: arguments,
		help:      raw.help || raw.helpLong,
		version:   raw.version,
	}, nil
}

func transform(config commandConfig, stdin io.Reader) (*meow.Result, io.Closer, error) {
	opts := config.options
	if opts.Seed == 0 {
		opts.Seed = time.Now().UnixNano()
	}
	var closer io.Closer
	switch {
	case len(config.arguments) > 0:
		opts.InputText = strings.Join(config.arguments, " ")
	case config.inputPath != "":
		file, err := os.Open(config.inputPath)
		if err != nil {
			return nil, nil, cli.NewError(cli.ExitFailure, "open input file: %v", err)
		}
		opts.InputReader = file
		closer = file
	case isTerminal(stdin):
		opts.InputText = "meow"
	default:
		opts.InputReader = stdin
	}
	res, err := meow.Transform(opts)
	return res, closer, err
}

func isTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	return err == nil && stat.Mode()&os.ModeCharDevice != 0
}

func writeOutput(stdout io.Writer, format string, res *meow.Result) error {
	switch format {
	case "json":
		resp := cli.Response[*meow.Result]{
			Schema:  "missing-utils/meow/v1",
			Command: "meow",
			Outcome: "pass",
			Data:    res,
		}
		return cli.WriteJSON(stdout, resp)
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
			if err := cli.WriteJSON(stdout, lineRes); err != nil {
				return err
			}
		}
	default:
		if res.AsciiArt != "" {
			if _, err := fmt.Fprint(stdout, res.AsciiArt); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(stdout, res.Output)
		return err
	}
	return nil
}
