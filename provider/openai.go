package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// Responses API request shape (simplified)
type oaResponsesRequest struct {
	Model           string  `json:"model"`
	Input           string  `json:"input"`
	MaxOutputTokens int     `json:"max_output_tokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"top_p,omitempty"`
	Stream          bool    `json:"stream,omitempty"`
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

// Streamed chat chunk
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

// Simplified Responses API streaming chunk
type oaResponsesStreamChunk struct {
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	} `json:"output"`
}

// Simplified Responses API non-stream response
type oaResponsesResp struct {
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	} `json:"output"`
}

// Decide which models should use the Responses API
func modelUsesResponsesAPI(model string) bool {
	m := strings.TrimSpace(strings.ToLower(model))
	if m == "" {
		return false
	}
	// Models that should use the Responses API
	prefixes := []string{"o1", "gpt-4.1", "gpt-5"}
	for _, p := range prefixes {
		if strings.HasPrefix(m, p) {
			return true
		}
	}
	return false
}

// Some Responses API models (observed: gpt-5 family) don't accept sampling params
// like temperature/top_p. Return true when the model supports sampling.
func responsesSupportsSampling(model string) bool {
	m := strings.TrimSpace(strings.ToLower(model))
	if m == "" {
		return true
	}
	// If model is gpt-5 family, don't send sampling params
	if strings.HasPrefix(m, "gpt-5") {
		return false
	}
	// default: assume sampling params are supported
	return true
}

func (p *openAIProvider) Complete(ctx context.Context, args CompletionArgs) (string, Usage, error) {
	useResponses := modelUsesResponsesAPI(args.Model)

	if useResponses {
		var body oaResponsesRequest
		body.Model = args.Model
		body.Input = buildUserContent(args.Prompt, args.Data)
		body.MaxOutputTokens = args.MaxTokens
		body.Stream = false

		if responsesSupportsSampling(args.Model) {
			body.Temperature = args.Temperature
			body.TopP = args.TopP
		}

		buf, _ := json.Marshal(body)

		if vb, _ := ctx.Value("nuro_verbose").(bool); vb {
			_, _ = fmt.Fprintf(
				os.Stderr, "nuro: openai: using /responses endpoint for model=%s\n", args.Model,
			)
			bstr := string(buf)
			if len(bstr) > 800 {
				bstr = bstr[:800] + "..."
			}
			_, _ = fmt.Fprintf(os.Stderr, "nuro: openai: request body: %s\n", bstr)
		}

		req, err := http.NewRequestWithContext(
			ctx, "POST", p.baseURL+"/responses", bytes.NewReader(buf),
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
			return "", Usage{}, fmt.Errorf(
				"openai responses error: %s - %s", resp.Status, trimBody(b),
			)
		}

		var r oaResponsesResp
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return "", Usage{}, err
		}

		var sb strings.Builder
		for _, out := range r.Output {
			for _, c := range out.Content {
				if c.Text != "" {
					sb.WriteString(c.Text)
				}
			}
		}

		return sb.String(), Usage{}, nil
	}

	// Fallback to chat completions API
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
	useResponses := modelUsesResponsesAPI(args.Model)

	if useResponses {
		var body oaResponsesRequest
		body.Model = args.Model
		body.Input = buildUserContent(args.Prompt, args.Data)
		body.MaxOutputTokens = args.MaxTokens
		body.Stream = true
		if responsesSupportsSampling(args.Model) {
			body.Temperature = args.Temperature
			body.TopP = args.TopP
		}
		buf, _ := json.Marshal(body)

		if vb, _ := ctx.Value("nuro_verbose").(bool); vb {
			_, _ = fmt.Fprintf(
				os.Stderr, "nuro: openai: streaming /responses endpoint for model=%s\n", args.Model,
			)
			bstr := string(buf)
			if len(bstr) > 800 {
				bstr = bstr[:800] + "..."
			}
			_, _ = fmt.Fprintf(os.Stderr, "nuro: openai: request body: %s\n", bstr)
		}

		req, err := http.NewRequestWithContext(
			ctx, "POST", p.baseURL+"/responses", bytes.NewReader(buf),
		)
		if err != nil {
			return "", Usage{}, err
		}
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
		req.Header.Set("Content-Type", "application/json")

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
			return "", Usage{}, fmt.Errorf(
				"openai responses error: %s - %s", resp.Status, trimBody(b),
			)
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
					var chunk oaResponsesStreamChunk
					if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
						for _, out := range chunk.Output {
							for _, c := range out.Content {
								if c.Text != "" {
									onDelta(c.Text)
									total.WriteString(c.Text)
								}
							}
						}
						continue
					}
					var chatChunk oaStreamChunk
					if err := json.Unmarshal([]byte(payload), &chatChunk); err == nil {
						for _, ch := range chatChunk.Choices {
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
				if err == io.ErrUnexpectedEOF {
					continue
				}
				if err != nil && err != io.EOF {
					return total.String(), Usage{}, err
				}
			}
		}

		return total.String(), Usage{}, nil
	}

	// Chat completions streaming path
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
			if err == io.ErrUnexpectedEOF {
				continue
			}
			if err != nil && err != io.EOF {
				return total.String(), Usage{}, err
			}
		}
	}

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
