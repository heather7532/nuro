package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/heather7532/nuro/config"
	"github.com/heather7532/nuro/provider"
	"github.com/heather7532/nuro/resolver"
	"github.com/spf13/pflag"
)

type cliFlags struct {
	promptFlag     string // value when provided as --prompt "..."
	promptUseStdin bool   // true when --prompt-stdin is present
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
	force          bool // -f / --force to override data size warnings
}

func parseFlags() (*cliFlags, error) {
	var f cliFlags

	pflag.StringVarP(
		&f.promptFlag, "prompt", "p", "",
		"Prompt text. Use --prompt-stdin to read prompt from stdin instead.",
	)
	pflag.BoolVar(
		&f.promptUseStdin, "prompt-stdin", false,
		"Read prompt from stdin instead of using --prompt",
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
	pflag.BoolVarP(&f.force, "force", "f", false, "Force sending large data without warnings.")
	pflag.BoolVar(&f.showVersion, "version", false, "Print version and exit.")
	// --help is auto-provided

	pflag.Parse()

	// Check for conflicting prompt flags
	if f.promptFlag != "" && f.promptUseStdin {
		return nil, usageError("cannot use both --prompt and --prompt-stdin")
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

	// Handle version immediately after flag parsing, before any stdin processing
	if flags.showVersion {
		fmt.Println(version)
		return
	}
	// Load .nuro config file if present, applying values as env vars
	cfg, err := config.LoadConfig()
	if err != nil {
		exitWithErr(err, 2) // Exit code 2 for config loading error
	}
	if cfg != nil {
		if err := cfg.Validate(); err != nil {
			exitWithErr(fmt.Errorf("invalid .nuro config: %w", err), 2)
		}
		if err := cfg.Apply(); err != nil {
			exitWithErr(fmt.Errorf("failed to apply .nuro config: %w", err), 2)
		}
	}

	// Resolve prompt & data per rules
	prompt, data, err := resolvePromptAndData(flags)
	if err != nil {
		exitWithErr(err, 2)
	}

	// Validate data size and warn about potential costs
	if err := validateDataSize(data, flags.force, flags.verbose); err != nil {
		exitWithErr(err, 2)
	}

	// Build the combined message content for verbose output
	combinedContent := buildCombinedContent(prompt, data)

	// Discover provider/model from env/args (no MCP in v1)
	res, err := resolver.ResolveProviderAndModel(flags.modelArg)
	if err != nil {
		exitWithErr(err, 3)
	}

	if flags.verbose || (pflag.CommandLine.Changed("model") && !flags.jsonOut) {
		keyDisplay := redactKey(res.APIKey)
		_, _ = fmt.Fprintf(
			os.Stderr, "nuro: provider=%s model=%s key=%s source=%s\n", res.ProviderName, res.Model,
			keyDisplay, res.KeySource,
		)

		if flags.verbose {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"nuro: args max_tokens=%d temp=%.1f top_p=%.1f timeout=%ds stream=%t json=%t\n",
				flags.maxTokens, flags.temperature, flags.topP, flags.timeoutSec, flags.stream,
				flags.jsonOut,
			)
			_, _ = fmt.Fprintf(
				os.Stderr, "nuro: prompt_len=%d data_len=%d\n", len(prompt), len(data),
			)
			_, _ = fmt.Fprintf(
				os.Stderr, "nuro: final_prompt='%s'\n", combinedContent,
			)
		}
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

	prov, err := provider.BuildProvider(res)
	if err != nil {
		exitWithErr(err, 3)
	}

	if flags.stream {
		total, usage, err := prov.Stream(
			ctx, args, func(delta string) {
				// Stream deltas to stdout as they arrive
				_, _ = fmt.Fprint(os.Stdout, delta)
			},
		)
		if err != nil {
			exitWithErr(err, 4)
		}

		if flags.verbose {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"nuro: stream response total_len=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d\n",
				len(total), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
			)
		}
		if flags.jsonOut {
			out := provider.JSONResult{
				Provider: prov.Name(),
				Model:    res.Model,
				Usage:    usage,
				Text:     total,
			}
			_, _ = fmt.Fprintln(os.Stdout) // newline after streaming text block if any
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

	if flags.verbose {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"nuro: response text_len=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d\n",
			len(text), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
		)
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
		_, _ = fmt.Fprintln(os.Stdout, text)
	}
}

func exitWithErr(err error, code int) {
	_, _ = fmt.Fprintf(os.Stderr, "nuro: %v\n", err)
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

func redactKey(key string) string {
	if len(key) <= 14 {
		// Short key, just show first few chars
		if len(key) <= 6 {
			return key[:2] + "***"
		}
		return key[:4] + "***"
	}
	// Standard format: show first 10 and last 4 chars
	return key[:10] + "***" + key[len(key)-4:]
}

func buildCombinedContent(prompt, data string) string {
	// Mirror the same logic as in provider/openai.go buildUserContent
	p := strings.TrimSpace(prompt)
	d := strings.TrimSpace(data)

	if p != "" && d != "" {
		return fmt.Sprintf("%s in the following data: %s", p, d)
	}
	if p != "" {
		return p
	}
	if d != "" {
		return fmt.Sprintf("Data:\n```\n%s\n```", d)
	}
	return ""
}

// Data size thresholds (in bytes)
const (
	dataSizeWarningThreshold = 50 * 1024  // 50KB - warning threshold
	dataSizeErrorThreshold   = 500 * 1024 // 500KB - error threshold (requires --force)
)

// validateDataSize checks if data is too large and provides warnings
func validateDataSize(data string, force, verbose bool) error {
	if data == "" {
		return nil // No data, no issue
	}

	dataSize := len([]byte(data))

	if verbose {
		_, _ = fmt.Fprintf(
			os.Stderr, "nuro: data size=%s (%d bytes)\n", formatBytes(dataSize), dataSize,
		)
	}

	// Large data that requires --force to proceed
	if dataSize > dataSizeErrorThreshold {
		if !force {
			return fmt.Errorf(
				"data size %s (%d bytes) exceeds safe limit (%s). This could be expensive to send to LLM.\n"+
					"Use --force/-f to proceed anyway, or reduce data size.\n"+
					"Consider filtering with: head, tail, grep, jq, or similar tools",
				formatBytes(dataSize), dataSize, formatBytes(dataSizeErrorThreshold),
			)
		}
		if verbose {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"nuro: WARNING: Large data size %s forced with --force flag. This may be expensive.\n",
				formatBytes(dataSize),
			)
		}
		return nil
	}

	// Medium data that gets a warning
	if dataSize > dataSizeWarningThreshold {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"nuro: WARNING: Data size %s (%d bytes) is large and may increase LLM costs.\n",
			formatBytes(dataSize), dataSize,
		)
		if !verbose {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"nuro: Use --verbose to see more details or --force/-f to suppress warnings.\n",
			)
		}
	}

	return nil
}

// formatBytes formats byte count into human-readable format
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
