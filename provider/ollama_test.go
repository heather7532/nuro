package provider

import (
	"testing"
)

func TestOllamaProviderName(t *testing.T) {
	provider := NewOllamaProvider("")
	if provider.Name() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", provider.Name())
	}
}

func TestOllamaProviderDefaultBaseURL(t *testing.T) {
	provider := NewOllamaProvider("")
	ollamaProvider := provider.(*ollamaProvider)

	expected := "http://localhost:11434"
	if ollamaProvider.baseURL != expected {
		t.Errorf("Expected base URL '%s', got '%s'", expected, ollamaProvider.baseURL)
	}
}

func TestOllamaProviderCustomBaseURL(t *testing.T) {
	customURL := "http://my-ollama-server:8080"
	provider := NewOllamaProvider(customURL)
	ollamaProvider := provider.(*ollamaProvider)

	if ollamaProvider.baseURL != customURL {
		t.Errorf("Expected base URL '%s', got '%s'", customURL, ollamaProvider.baseURL)
	}
}

func TestOllamaProviderTrimsTrailingSlash(t *testing.T) {
	urlWithSlash := "http://localhost:11434/"
	provider := NewOllamaProvider(urlWithSlash)
	ollamaProvider := provider.(*ollamaProvider)

	expected := "http://localhost:11434"
	if ollamaProvider.baseURL != expected {
		t.Errorf("Expected base URL '%s' (trimmed), got '%s'", expected, ollamaProvider.baseURL)
	}
}

func TestBuildOllamaPrompt(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		data           string
		expectedResult string
	}{
		{
			name:           "Prompt and data combined",
			prompt:         "count words",
			data:           "one two three four 5",
			expectedResult: "count words\n\nData:\n```\none two three four 5\n```",
		},
		{
			name:           "Only prompt",
			prompt:         "hello world",
			data:           "",
			expectedResult: "hello world",
		},
		{
			name:           "Only data",
			prompt:         "",
			data:           "test data here",
			expectedResult: "Here is some data to analyze:\n\n```\ntest data here\n```",
		},
		{
			name:           "Empty both",
			prompt:         "",
			data:           "",
			expectedResult: "",
		},
		{
			name:           "Summarize with data",
			prompt:         "summarize the content",
			data:           "This is a long document with many paragraphs...",
			expectedResult: "summarize the content\n\nData:\n```\nThis is a long document with many paragraphs...\n```",
		},
		{
			name:           "Translate with data",
			prompt:         "translate to Spanish",
			data:           "Hello world",
			expectedResult: "translate to Spanish\n\nData:\n```\nHello world\n```",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := buildOllamaPrompt(tt.prompt, tt.data)

				if result != tt.expectedResult {
					t.Logf("Test: %s", tt.name)
					t.Logf("Prompt: '%s'", tt.prompt)
					t.Logf("Data: '%s'", tt.data)
					t.Logf("Expected: '%s'", tt.expectedResult)
					t.Logf("Got: '%s'", result)
					t.Errorf("Expected content '%s', got '%s'", tt.expectedResult, result)
				} else {
					t.Logf("✓ Test '%s' passed", tt.name)
				}
			},
		)
	}
}

func TestOllamaProviderBuild(t *testing.T) {
	res := &ProviderResolution{
		ProviderName: "ollama",
		BaseURL:      "http://localhost:11434",
	}

	provider, err := BuildProvider(res)
	if err != nil {
		t.Fatalf("Unexpected error building Ollama provider: %v", err)
	}

	if provider.Name() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", provider.Name())
	}
}
