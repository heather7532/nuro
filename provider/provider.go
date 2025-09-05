package provider

import (
	"context"
	"fmt"
	"time"
)

type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type ProviderResolution struct {
	ProviderName string
	Model        string
	APIKey       string
	BaseURL      string
	KeySource    string
}

type JSONResult struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Usage    Usage  `json:"usage,omitempty"`
	Text     string `json:"text"`
}

type CompletionArgs struct {
	Model       string
	Prompt      string
	Data        string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Stream      bool
	JSONOut     bool
	Timeout     time.Duration
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, args CompletionArgs) (text string, usage Usage, err error)
	Stream(ctx context.Context, args CompletionArgs, onDelta func(delta string)) (
		total string, usage Usage, err error,
	)
}

func BuildProvider(res *ProviderResolution) (Provider, error) {
	switch res.ProviderName {
	case "openai":
		return NewOpenAIProvider(res.APIKey, res.BaseURL), nil
	default:
		return nil, fmt.Errorf(
			"provider '%s' not implemented yet; set NURO_PROVIDER=openai or provide OPENAI_API_KEY",
			res.ProviderName,
		)
	}
}
