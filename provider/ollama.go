package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ollamaProvider struct {
	baseURL string
	client  *http.Client
}

func NewOllamaProvider(baseURL string) Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &ollamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 0}, // use context timeouts per request
	}
}

func (p *ollamaProvider) Name() string { return "ollama" }

type ollamaGenerateRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Stream   bool   `json:"stream"`
	System   string `json:"system,omitempty"`
	Template string `json:"template,omitempty"`
	Context  []int  `json:"context,omitempty"`
	Options  struct {
		Temperature float64 `json:"temperature,omitempty"`
		TopP        float64 `json:"top_p,omitempty"`
		NumPredict  int     `json:"num_predict,omitempty"`
	} `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

func (p *ollamaProvider) Complete(ctx context.Context, args CompletionArgs) (
	string,
	Usage, error,
) {
	prompt := buildOllamaPrompt(args.Prompt, args.Data)

	body := ollamaGenerateRequest{
		Model:  args.Model,
		Prompt: prompt,
		Stream: false,
	}

	// Set options if provided
	if args.Temperature != 0 {
		body.Options.Temperature = args.Temperature
	}
	if args.TopP != 0 {
		body.Options.TopP = args.TopP
	}
	if args.MaxTokens != 0 {
		body.Options.NumPredict = args.MaxTokens
	}

	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(buf),
	)
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", Usage{}, fmt.Errorf("ollama error: %s - %s", resp.Status, trimBody(b))
	}

	var r ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", Usage{}, err
	}

	// Convert Ollama's token counts to Usage format
	usage := Usage{
		PromptTokens:     r.PromptEvalCount,
		CompletionTokens: r.EvalCount,
		TotalTokens:      r.PromptEvalCount + r.EvalCount,
	}

	return r.Response, usage, nil
}

func (p *ollamaProvider) Stream(
	ctx context.Context, args CompletionArgs, onDelta func(string),
) (string, Usage, error) {
	prompt := buildOllamaPrompt(args.Prompt, args.Data)

	body := ollamaGenerateRequest{
		Model:  args.Model,
		Prompt: prompt,
		Stream: true,
	}

	// Set options if provided
	if args.Temperature != 0 {
		body.Options.Temperature = args.Temperature
	}
	if args.TopP != 0 {
		body.Options.TopP = args.TopP
	}
	if args.MaxTokens != 0 {
		body.Options.NumPredict = args.MaxTokens
	}

	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(buf),
	)
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	// No client timeout here; rely on ctx
	oldTimeout := p.client.Timeout
	p.client.Timeout = 0
	defer func() { p.client.Timeout = oldTimeout }()

	resp, err := p.client.Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", Usage{}, fmt.Errorf("ollama error: %s - %s", resp.Status, trimBody(b))
	}

	reader := bufio.NewReader(resp.Body)
	var total strings.Builder
	var finalUsage Usage
	for {
		// Check context cancellation before reading
		if ctx.Err() != nil {
			return total.String(), finalUsage, ctx.Err()
		}

		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSpace(line)
			if line != "" {
				var chunk ollamaGenerateResponse
				if err := json.Unmarshal([]byte(line), &chunk); err == nil {
					if chunk.Response != "" {
						onDelta(chunk.Response)
						total.WriteString(chunk.Response)
					}

					// If this is the final chunk, capture usage info
					if chunk.Done {
						finalUsage = Usage{
							PromptTokens:     chunk.PromptEvalCount,
							CompletionTokens: chunk.EvalCount,
							TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
						}
						break
					}
				}
			}
		}

		// Handle errors with proper precedence
		if err != nil {
			// Check for context cancellation first
			if ctx.Err() != nil {
				return total.String(), finalUsage, ctx.Err()
			}

			// Handle EOF conditions
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				continue
			}

			// Return other errors
			return total.String(), finalUsage, err
		}
	}

	return total.String(), finalUsage, nil
}

// buildOllamaPrompt creates a prompt for Ollama's native format
func buildOllamaPrompt(prompt, data string) string {
	p := strings.TrimSpace(prompt)
	d := strings.TrimSpace(data)

	// When both prompt and data are present, combine them naturally
	if p != "" && d != "" {
		return fmt.Sprintf("%s\n\nData:\n```\n%s\n```", p, d)
	}

	// If only prompt, use it directly
	if p != "" {
		return p
	}

	// If only data, present it clearly
	if d != "" {
		return fmt.Sprintf("Here is some data to analyze:\n\n```\n%s\n```", d)
	}

	// If both are empty, still send an empty string
	return ""
}
