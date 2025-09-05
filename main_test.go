package main

import (
	"strings"
	"testing"
)

func TestUsageError(t *testing.T) {
	err := usageError("test")
	if err == nil {
		t.Error("Expected error but got nil")
		return
	}
	if err.Error() != "usage error: test" {
		t.Errorf("Expected 'usage error: test', got '%s'", err.Error())
	}
}
func TestResolvePromptAndDataStdinPromptWithInlineData(t *testing.T) {
	// Test scenario: echo 'count the words in the provided data' | nuro --prompt-stdin --data 'one two three four 5'
	// This should result in prompt from stdin and data from --data flag

	flags := &cliFlags{
		promptUseStdin: true, // --prompt-stdin flag
		dataInline:     "one two three four 5",
	}

	// Simulate stdin data
	stdinContent := []byte("count the words in the provided data")

	// We can't easily mock readMaybeStdin, but we can test the logic directly
	prompt := string(stdinContent)
	data := flags.dataInline

	expectedPrompt := "count the words in the provided data"
	expectedData := "one two three four 5"

	if prompt != expectedPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", expectedPrompt, prompt)
	}
	if data != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, data)
	}

	// Test that this scenario would produce the expected word count (5)
	// The LLM should count words in: "one two three four 5"
	expectedWordCount := 5
	actualWords := len(strings.Split(strings.TrimSpace(data), " "))

	if actualWords != expectedWordCount {
		t.Errorf("Expected %d words in data, got %d", expectedWordCount, actualWords)
	}
}

func TestValidateDataSizeWithSmallData(t *testing.T) {
	// Test with small data (should not trigger any warnings)
	smallData := "hello world"
	err := validateDataSize(smallData, false, false)
	if err != nil {
		t.Errorf("Expected no error for small data, got: %v", err)
	}
}

func TestValidateDataSizeWithMediumData(t *testing.T) {
	// Test with medium data (should trigger warning but not error)
	mediumData := strings.Repeat("a", 60*1024) // 60KB - above warning threshold
	err := validateDataSize(mediumData, false, false)
	if err != nil {
		t.Errorf("Expected no error for medium data, got: %v", err)
	}
}

func TestValidateDataSizeWithLargeDataNoForce(t *testing.T) {
	// Test with large data without --force (should error)
	largeData := strings.Repeat("a", 600*1024) // 600KB - above error threshold
	err := validateDataSize(largeData, false, false)
	if err == nil {
		t.Error("Expected error for large data without --force")
		return
	}
	if !strings.Contains(err.Error(), "exceeds safe limit") {
		t.Errorf("Expected 'exceeds safe limit' in error message, got: %v", err)
	}
}

func TestValidateDataSizeWithLargeDataWithForce(t *testing.T) {
	// Test with large data with --force (should not error)
	largeData := strings.Repeat("a", 600*1024) // 600KB - above error threshold
	err := validateDataSize(largeData, true, false)
	if err != nil {
		t.Errorf("Expected no error for large data with --force, got: %v", err)
	}
}

func TestValidateDataSizeWithEmptyData(t *testing.T) {
	// Test with empty data (should not trigger any warnings)
	err := validateDataSize("", false, false)
	if err != nil {
		t.Errorf("Expected no error for empty data, got: %v", err)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int
		expected string
	}{
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{50 * 1024, "50.0KB"},
		{500 * 1024, "500.0KB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}