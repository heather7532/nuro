package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"nuro/provider"
	"os"
	"time"

	"github.com/spf13/pflag"
)

type cliFlags struct {
	promptFlag     string // value when provided as --prompt "..."
	promptUseStdin bool   // true when -p is present with *no value*
	dataInline     string // --data "..."
	dataFile       string // --data-file path
	modelArg       string // -m / --model
	maxTokens      int
	temperature    float64
	topP           float64
	timeoutSec     int
	stream         bool
	jsonOut        bool
	verbose        bool
	showVersion    bool
}

func parseFlags() (*cliFlags, error) {
	var f cliFlags

	pflag.StringVarP(
		&f.promptFlag, "prompt", "p", "",
		"Prompt text. Use '-p' with no value to read prompt from stdin.",
	)
	pflag.StringVar(&f.dataInline, "data", "", "Inline data/payload string.")
	pflag.StringVar(&f.dataFile, "data-file", "", "Path to file containing data/payload.")
	pflag.StringVarP(
		&f.modelArg, "model", "m", "", "Model id (or $ENV to read model id from env var).",
	)

	pflag.IntVar(&f.maxTokens, "max-tokens", 1024, "Max tokens for completion.")
	pflag.Float64Var(&f.temperature, "temperature", 0.7, "Sampling temperature.")
	pflag.Float64Var(&f.topP, "top-p", 1.0, "Top-p (nucleus sampling).")
	pflag.IntVar(&f.timeoutSec, "timeout", 60, "Request timeout in seconds.")
	pflag.BoolVar(&f.stream, "stream", false, "Stream tokens to stdout.")
	pflag.BoolVar(&f.jsonOut, "json", false, "Emit structured JSON result.")
	pflag.BoolVar(&f.verbose, "verbose", false, "Verbose diagnostics to stderr.")
	pflag.BoolVar(&f.showVersion, "version", false, "Print version and exit.")
	// --help is auto-provided

	pflag.Parse()

	// Enable optional "no value" form for -p / --prompt
	if fl := pflag.Lookup("prompt"); fl != nil {
		// If user wrote `-p` with no following value, pflag sets promptFlag to this sentinel.
		// We configure this sentinel via NoOptDefVal below:
		if fl.NoOptDefVal == "" {
			fl.NoOptDefVal = "__NURO_PROMPT_STDIN__"
		}
		if f.promptFlag == "__NURO_PROMPT_STDIN__" {
			f.promptUseStdin = true
			f.promptFlag = "" // clear to avoid confusion; stdin will be used
		}
	}

	// Disallow --data with no value (must be explicitly provided)
	// pflag already errors when a string flag is used without a value,
	// but in case a shell passes an empty string, we enforce here:
	if pflag.CommandLine.Changed("data") && f.dataInline == "" {
		return nil, usageError("--data requires a value; use --data-file or pipe stdin per rules")
	}

	return &f, nil
}

func main() {
	flags, err := parseFlags()
	if err != nil {
		exitWithErr(err, 2)
	}
	if flags.showVersion {
		fmt.Println(version)
		return
	}

	// Resolve prompt & data per rules
	prompt, data, err := resolvePromptAndData(flags)
	if err != nil {
		exitWithErr(err, 2)
	}

	// Discover provider/model from env/args (no MCP in v1)
	res, err := resolveProviderAndModel(flags.modelArg)
	if err != nil {
		exitWithErr(err, 3)
	}
	if flags.verbose || (pflag.CommandLine.Changed("model") && !flags.jsonOut) {
		fmt.Fprintf(os.Stderr, "nuro: provider=%s model=%s\n", res.ProviderName, res.Model)
	}

	// Build request
	args := provider.CompletionArgs{
		Model:       res.Model,
		Prompt:      prompt,
		Data:        data,
		MaxTokens:   flags.maxTokens,
		Temperature: flags.temperature,
		TopP:        flags.topP,
		JSONOut:     flags.jsonOut,
		Stream:      flags.stream,
		Timeout:     time.Duration(flags.timeoutSec) * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
	defer cancel()

	prov, err := provider.buildProvider(res)
	if err != nil {
		exitWithErr(err, 3)
	}

	if flags.stream {
		total, usage, err := prov.Stream(
			ctx, args, func(delta string) {
				// Stream deltas to stdout as they arrive
				fmt.Fprint(os.Stdout, delta)
			},
		)
		if err != nil {
			exitWithErr(err, 4)
		}
		if flags.jsonOut {
			out := provider.JSONResult{
				Provider: prov.Name(),
				Model:    res.Model,
				Usage:    usage,
				Text:     total,
			}
			fmt.Fprintln(os.Stdout) // newline after streaming text block if any
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out)
		}
		return
	}

	// Non-streaming
	text, usage, err := prov.Complete(ctx, args)
	if err != nil {
		exitWithErr(err, 4)
	}
	if flags.jsonOut {
		out := provider.JSONResult{
			Provider: prov.Name(),
			Model:    res.Model,
			Usage:    usage,
			Text:     text,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Fprintln(os.Stdout, text)
	}
}

func exitWithErr(err error, code int) {
	fmt.Fprintf(os.Stderr, "nuro: %v\n", err)
	os.Exit(code)
}

func usageError(msg string) error { return fmt.Errorf("usage error: %s", msg) }

func resolvePromptAndData(f *cliFlags) (prompt string, data string, err error) {
	stdinData, stdinPresent, err := readMaybeStdin()
	if err != nil {
		return "", "", err
	}

	// Determine prompt
	switch {
	case f.promptUseStdin:
		if !stdinPresent || len(stdinData) == 0 {
			return "", "", usageError("'-p' used with no prompt on stdin")
		}
		prompt = string(stdinData)
	case f.promptFlag != "":
		prompt = f.promptFlag
	default:
		// No explicit prompt; allowed (depends on use-case)
		// It's fine to send only data with an instruction-like prompt in data, but users generally pass prompt.
	}

	// Determine data
	if f.dataFile != "" && f.dataInline != "" {
		return "", "", usageError("cannot use both --data and --data-file")
	}

	if f.dataInline != "" {
		data = f.dataInline
	} else if f.dataFile != "" {
		b, e := os.ReadFile(f.dataFile)
		if e != nil {
			return "", "", fmt.Errorf("failed to read --data-file: %w", e)
		}
		data = string(b)
	} else {
		// Default stdin->data if stdin present AND prompt didn't consume stdin
		if stdinPresent && !f.promptUseStdin {
			data = string(stdinData)
		}
	}

	// Conflict: both prompt and data attempt stdin? Covered above because promptUseStdin "consumed" stdin already.

	// Special rule you specified:
	// If -p (no value) AND --data "some value" => prompt from stdin; data = inline string (already handled)
	return prompt, data, nil
}

func readMaybeStdin() ([]byte, bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("cannot stat stdin: %w", err)
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		// TTY -> no stdin content
		return nil, false, nil
	}
	// Non-tty: read all
	b, err := io.ReadAll(bufio.NewReader(os.Stdin))
	if err != nil {
		return nil, false, fmt.Errorf("failed reading stdin: %w", err)
	}
	return b, true, nil
}