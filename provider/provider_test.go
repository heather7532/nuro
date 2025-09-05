package provider

import (
	"strings"
	"testing"
)

func TestProviderName(t *testing.T) {
	res := &ProviderResolution{
		ProviderName: "openai",
		APIKey:       "test-key",
	}

	provider, err := BuildProvider(res)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if provider.Name() != "openai" {
		t.Errorf("Expected openai, got %s", provider.Name())
	}
}

func TestBuildUserContentCombination(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		data           string
		expectedResult string
		verbose        bool
	}{
		{
			name:           "Prompt and data combined",
			prompt:         "count words",
			data:           "one two three four 5",
			expectedResult: "count words in the following data: one two three four 5",
			verbose:        true,
		},
		{
			name:           "Only prompt",
			prompt:         "hello world",
			data:           "",
			expectedResult: "hello world",
			verbose:        true,
		},
		{
			name:           "Only data",
			prompt:         "",
			data:           "test data here",
			expectedResult: "Data:\n```\ntest data here\n```",
			verbose:        true,
		},
		{
			name:           "Empty both",
			prompt:         "",
			data:           "",
			expectedResult: "",
			verbose:        false,
		},
		{
			name:           "Summarize with data",
			prompt:         "summarize the content",
			data:           "This is a long document with many paragraphs...",
			expectedResult: "summarize the content in the following data: This is a long document with many paragraphs...",
			verbose:        true,
		},
		{
			name:           "Translate with data",
			prompt:         "translate to Spanish",
			data:           "Hello world",
			expectedResult: "translate to Spanish in the following data: Hello world",
			verbose:        true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := buildUserContent(tt.prompt, tt.data)

				if result != tt.expectedResult {
					if tt.verbose {
						t.Logf("Test: %s", tt.name)
						t.Logf("Prompt: '%s'", tt.prompt)
						t.Logf("Data: '%s'", tt.data)
						t.Logf("Expected: '%s'", tt.expectedResult)
						t.Logf("Got: '%s'", result)
					}
					t.Errorf("Expected content '%s', got '%s'", tt.expectedResult, result)
				} else if tt.verbose {
					t.Logf("✓ Test '%s' passed", tt.name)
					t.Logf("  Result: %s", result)
				}
			},
		)
	}
}

func TestAssembleMessages(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		data           string
		expectedRole   string
		expectedPrefix string
		verbose        bool
	}{
		{
			name:           "Count words example",
			prompt:         "count words",
			data:           "one two three four 5",
			expectedRole:   "user",
			expectedPrefix: "count words in the following data:",
			verbose:        true,
		},
		{
			name:           "Data file example",
			prompt:         "find emails",
			data:           "Contact: john@example.com Phone: 555-0123",
			expectedRole:   "user",
			expectedPrefix: "find emails in the following data:",
			verbose:        true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				messages := assembleMessages(tt.prompt, tt.data)

				if len(messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(messages))
					return
				}

				msg := messages[0]
				if msg.Role != tt.expectedRole {
					t.Errorf("Expected role '%s', got '%s'", tt.expectedRole, msg.Role)
				}

				if !strings.HasPrefix(msg.Content, tt.expectedPrefix) {
					if tt.verbose {
						t.Logf("Test: %s", tt.name)
						t.Logf("Expected prefix: '%s'", tt.expectedPrefix)
						t.Logf("Got content: '%s'", msg.Content)
					}
					t.Errorf(
						"Expected content to start with '%s', got '%s'", tt.expectedPrefix,
						msg.Content,
					)
				} else if tt.verbose {
					t.Logf("✓ Test '%s' passed", tt.name)
					t.Logf("  Message content: %s", msg.Content)
				}
			},
		)
	}
}

func TestDebugActualContent(t *testing.T) {
	// This should match your command: echo 'count words' | ./nuro -p --data 'one two three four 5'
	prompt := "count words"
	data := "one two three four 5"

	t.Logf("Input prompt: '%s' (len=%d)", prompt, len(prompt))
	t.Logf("Input data: '%s' (len=%d)", data, len(data))

	content := buildUserContent(prompt, data)
	t.Logf("Final content sent to API: '%s'", content)
	t.Logf("Final content length: %d", len(content))

	expected := "count words in the following data: one two three four 5"
	if content != expected {
		t.Errorf("Expected: '%s'", expected)
		t.Errorf("Got:      '%s'", content)
	}

	// Test the assembleMessages function too
	messages := assembleMessages(prompt, data)
	if len(messages) == 1 {
		t.Logf("Message role: %s", messages[0].Role)
		t.Logf("Message content: '%s'", messages[0].Content)
	}
}
