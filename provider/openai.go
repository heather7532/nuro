package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"io"
	"net/http"
	"strings"
)

type openAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(apiKey, baseURL string) Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &openAIProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 0}, // use context timeouts per request
	}
}

func (p *openAIProvider) Name() string { return "openai" }

type oaChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaChatRequest struct {
	Model       string      `json:"model"`
	Messages    []oaChatMsg `json:"messages"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
	TopP        float64     `json:"top_p,omitempty"`
	Stream      bool        `json:"stream,omitempty"`
}

type oaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type oaChoice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

type oaResp struct {
	Choices []oaChoice `json:"choices"`
	Usage   *oaUsage   `json:"usage,omitempty"`
}

type oaStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type oaStreamChoice struct {
	Delta        oaStreamDelta `json:"delta"`
	FinishReason *string       `json:"finish_reason"`
}

type oaStreamChunk struct {
	Choices []oaStreamChoice `json:"choices"`
}

func (p *openAIProvider) Complete(ctx context.Context, args CompletionArgs) (
	string,
	Usage, error,
) {
	body := oaChatRequest{
		Model:       args.Model,
		Messages:    assembleMessages(args.Prompt, args.Data),
		MaxTokens:   args.MaxTokens,
		Temperature: args.Temperature,
		TopP:        args.TopP,
		Stream:      false,
	}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(buf),
	)
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", Usage{}, fmt.Errorf("openai error: %s - %s", resp.Status, trimBody(b))
	}

	var r oaResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", Usage{}, err
	}
	if len(r.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("openai: no choices returned")
	}
	text := r.Choices[0].Message.Content
	usage := Usage{}
	if r.Usage != nil {
		usage = Usage{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		}
	}
	return text, usage, nil
}

func (p *openAIProvider) Stream(
	ctx context.Context, args CompletionArgs, onDelta func(string),
) (string, Usage, error) {
	body := oaChatRequest{
		Model:       args.Model,
		Messages:    assembleMessages(args.Prompt, args.Data),
		MaxTokens:   args.MaxTokens,
		Temperature: args.Temperature,
		TopP:        args.TopP,
		Stream:      true,
	}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(buf),
	)
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
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
		return "", Usage{}, fmt.Errorf("openai error: %s - %s", resp.Status, trimBody(b))
	}

	reader := bufio.NewReader(resp.Body)
	var total strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			l := strings.TrimSpace(line)
			if strings.HasPrefix(l, "data: ") {
				payload := strings.TrimPrefix(l, "data: ")
				if payload == "[DONE]" {
					break
				}
				var chunk oaStreamChunk
				if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
					for _, ch := range chunk.Choices {
						d := ch.Delta.Content
						if d != "" {
							onDelta(d)
							total.WriteString(d)
						}
					}
				}
			}
		}
		if err != nil {
			if errorsIsEOF(err) {
				break
			}
			if ctx.Err() != nil {
				return total.String(), Usage{}, ctx.Err()
			}
			// continue on short reads
			if err == io.ErrUnexpectedEOF {
				continue
			}
			if err != nil && err != io.EOF {
				return total.String(), Usage{}, err
			}
		}
	}

	// Usage is not present in streamed chunks here
	return total.String(), Usage{}, nil
}
func assembleMessages(prompt, data string) []oaChatMsg {
	content := buildUserContent(prompt, data)
	return []oaChatMsg{{Role: "user", Content: content}}
}

// helper builds a single user message with clear labels & fencing
func buildUserContent(prompt, data string) string {
	p := strings.TrimSpace(prompt)
	d := strings.TrimSpace(data)

	// When both prompt and data are present, combine them naturally
	if p != "" && d != "" {
		return fmt.Sprintf("%s in the following data: %s", p, d)
	}

	// If only prompt, use it directly
	if p != "" {
		return p
	}

	// If only data, present it clearly
	if d != "" {
		return fmt.Sprintf("Data:\n```\n%s\n```", d)
	}

	// If both are empty, still send an empty string to satisfy API
	return ""
}

func trimBody(b []byte) string {
	s := string(b)
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}

func errorsIsEOF(err error) bool {
	if err == nil {
		return false
	}
	return errorsIs(err, io.EOF)
}

// Small polyfill to avoid importing errors for Is on older Go (<1.20) if needed.
func errorsIs(err, target error) bool {
	return err == target
}
