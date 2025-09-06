package resolver

import (
	"os"
	"testing"
)

func TestOllamaIntegrationViaOpenAIAdapter(t *testing.T) {
	// Save original env vars to restore later
	origAPIKey := os.Getenv("NURO_API_KEY")
	origBaseURL := os.Getenv("NURO_BASE_URL")
	origProvider := os.Getenv("NURO_PROVIDER")
	defer func() {
		if origAPIKey == "" {
			os.Unsetenv("NURO_API_KEY")
		} else {
			os.Setenv("NURO_API_KEY", origAPIKey)
		}
		if origBaseURL == "" {
			os.Unsetenv("NURO_BASE_URL")
		} else {
			os.Setenv("NURO_BASE_URL", origBaseURL)
		}
		if origProvider == "" {
			os.Unsetenv("NURO_PROVIDER")
		} else {
			os.Setenv("NURO_PROVIDER", origProvider)
		}
	}()

	// Test Ollama configuration using OpenAI adapter
	os.Setenv("NURO_API_KEY", "ollama")
	os.Setenv("NURO_BASE_URL", "http://localhost:11434/v1")
	os.Setenv("NURO_PROVIDER", "openai")

	res, err := ResolveProviderAndModel("llama3.1:8b")
	if err != nil {
		t.Fatalf("Failed to resolve Ollama config: %v", err)
	}

	if res.ProviderName != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", res.ProviderName)
	}

	if res.Model != "llama3.1:8b" {
		t.Errorf("Expected model 'llama3.1:8b', got '%s'", res.Model)
	}

	if res.APIKey != "ollama" {
		t.Errorf("Expected API key 'ollama', got '%s'", res.APIKey)
	}

	if res.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("Expected base URL 'http://localhost:11434/v1', got '%s'", res.BaseURL)
	}

	if res.KeySource != "NURO_API_KEY" {
		t.Errorf("Expected key source 'NURO_API_KEY', got '%s'", res.KeySource)
	}
}

func TestOllamaWithoutExplicitProvider(t *testing.T) {
	// Test that Ollama works when NURO_PROVIDER is not set (should default based on model)
	origAPIKey := os.Getenv("NURO_API_KEY")
	origBaseURL := os.Getenv("NURO_BASE_URL")
	origProvider := os.Getenv("NURO_PROVIDER")
	defer func() {
		if origAPIKey == "" {
			os.Unsetenv("NURO_API_KEY")
		} else {
			os.Setenv("NURO_API_KEY", origAPIKey)
		}
		if origBaseURL == "" {
			os.Unsetenv("NURO_BASE_URL")
		} else {
			os.Setenv("NURO_BASE_URL", origBaseURL)
		}
		if origProvider == "" {
			os.Unsetenv("NURO_PROVIDER")
		} else {
			os.Setenv("NURO_PROVIDER", origProvider)
		}
	}()

	os.Setenv("NURO_API_KEY", "ollama")
	os.Setenv("NURO_BASE_URL", "http://localhost:11434/v1")
	os.Unsetenv("NURO_PROVIDER") // Unset to test default behavior

	res, err := ResolveProviderAndModel("gpt-4o-mini")
	if err != nil {
		t.Fatalf("Failed to resolve Ollama config without explicit provider: %v", err)
	}

	// Should default to openai provider since no NURO_PROVIDER is set and gpt-4o-mini infers openai
	if res.ProviderName != "openai" {
		t.Errorf("Expected default provider 'openai', got '%s'", res.ProviderName)
	}

	if res.Model != "gpt-4o-mini" {
		t.Errorf("Expected model 'gpt-4o-mini', got '%s'", res.Model)
	}
}

func TestOllamaWithDifferentModels(t *testing.T) {
	// Test with various Ollama model names
	origAPIKey := os.Getenv("NURO_API_KEY")
	origBaseURL := os.Getenv("NURO_BASE_URL")
	origProvider := os.Getenv("NURO_PROVIDER")
	defer func() {
		if origAPIKey == "" {
			os.Unsetenv("NURO_API_KEY")
		} else {
			os.Setenv("NURO_API_KEY", origAPIKey)
		}
		if origBaseURL == "" {
			os.Unsetenv("NURO_BASE_URL")
		} else {
			os.Setenv("NURO_BASE_URL", origBaseURL)
		}
		if origProvider == "" {
			os.Unsetenv("NURO_PROVIDER")
		} else {
			os.Setenv("NURO_PROVIDER", origProvider)
		}
	}()

	os.Setenv("NURO_API_KEY", "ollama")
	os.Setenv("NURO_BASE_URL", "http://localhost:11434/v1")
	os.Setenv("NURO_PROVIDER", "openai")

	testModels := []string{
		"llama3.1:8b",
		"llama3.1:70b",
		"mistral:7b",
		"codellama:13b",
		"phi3:mini",
	}

	for _, model := range testModels {
		t.Run(
			"model_"+model, func(t *testing.T) {
				res, err := ResolveProviderAndModel(model)
				if err != nil {
					t.Fatalf("Failed to resolve Ollama config for model %s: %v", model, err)
				}

				if res.ProviderName != "openai" {
					t.Errorf("Expected provider 'openai', got '%s'", res.ProviderName)
				}

				if res.Model != model {
					t.Errorf("Expected model '%s', got '%s'", model, res.Model)
				}

				if res.BaseURL != "http://localhost:11434/v1" {
					t.Errorf("Expected base URL 'http://localhost:11434/v1', got '%s'", res.BaseURL)
				}
			},
		)
	}
}

func TestOllamaEnvironmentPrecedence(t *testing.T) {
	// Test that NURO_* variables take precedence over regular provider env vars
	origAPIKey := os.Getenv("NURO_API_KEY")
	origBaseURL := os.Getenv("NURO_BASE_URL")
	origProvider := os.Getenv("NURO_PROVIDER")
	origOpenAIKey := os.Getenv("OPENAI_API_KEY")
	origOpenAIBase := os.Getenv("OPENAI_BASE_URL")
	defer func() {
		if origAPIKey == "" {
			os.Unsetenv("NURO_API_KEY")
		} else {
			os.Setenv("NURO_API_KEY", origAPIKey)
		}
		if origBaseURL == "" {
			os.Unsetenv("NURO_BASE_URL")
		} else {
			os.Setenv("NURO_BASE_URL", origBaseURL)
		}
		if origProvider == "" {
			os.Unsetenv("NURO_PROVIDER")
		} else {
			os.Setenv("NURO_PROVIDER", origProvider)
		}
		if origOpenAIKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", origOpenAIKey)
		}
		if origOpenAIBase == "" {
			os.Unsetenv("OPENAI_BASE_URL")
		} else {
			os.Setenv("OPENAI_BASE_URL", origOpenAIBase)
		}
	}()

	// Set both NURO_* and OPENAI_* vars to ensure NURO_* takes precedence
	os.Setenv("NURO_API_KEY", "ollama")
	os.Setenv("NURO_BASE_URL", "http://localhost:11434/v1")
	os.Setenv("NURO_PROVIDER", "openai")
	os.Setenv("OPENAI_API_KEY", "sk-real-openai-key")
	os.Setenv("OPENAI_BASE_URL", "https://api.openai.com/v1")

	res, err := ResolveProviderAndModel("llama3.1:8b")
	if err != nil {
		t.Fatalf("Failed to resolve with precedence test: %v", err)
	}

	// Should use NURO_* values, not OPENAI_* values
	if res.APIKey != "ollama" {
		t.Errorf("Expected NURO_API_KEY value 'ollama', got '%s'", res.APIKey)
	}

	if res.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("Expected NURO_BASE_URL value 'http://localhost:11434/v1', got '%s'", res.BaseURL)
	}

	if res.KeySource != "NURO_API_KEY" {
		t.Errorf("Expected key source 'NURO_API_KEY', got '%s'", res.KeySource)
	}
}

func TestNativeOllamaProvider(t *testing.T) {
	// Test native Ollama provider resolution
	origAPIKey := os.Getenv("NURO_API_KEY")
	origBaseURL := os.Getenv("NURO_BASE_URL")
	origProvider := os.Getenv("NURO_PROVIDER")
	defer func() {
		if origAPIKey == "" {
			os.Unsetenv("NURO_API_KEY")
		} else {
			os.Setenv("NURO_API_KEY", origAPIKey)
		}
		if origBaseURL == "" {
			os.Unsetenv("NURO_BASE_URL")
		} else {
			os.Setenv("NURO_BASE_URL", origBaseURL)
		}
		if origProvider == "" {
			os.Unsetenv("NURO_PROVIDER")
		} else {
			os.Setenv("NURO_PROVIDER", origProvider)
		}
	}()

	// Test native Ollama configuration
	os.Setenv("NURO_API_KEY", "dummy") // Ollama doesn't need a real API key
	os.Setenv("NURO_BASE_URL", "http://localhost:11434")
	os.Setenv("NURO_PROVIDER", "ollama")

	res, err := ResolveProviderAndModel("llama3.1:8b")
	if err != nil {
		t.Fatalf("Failed to resolve native Ollama config: %v", err)
	}

	if res.ProviderName != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%s'", res.ProviderName)
	}

	if res.Model != "llama3.1:8b" {
		t.Errorf("Expected model 'llama3.1:8b', got '%s'", res.Model)
	}

	if res.APIKey != "dummy" {
		t.Errorf("Expected API key 'dummy', got '%s'", res.APIKey)
	}

	if res.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected base URL 'http://localhost:11434', got '%s'", res.BaseURL)
	}

	if res.KeySource != "NURO_API_KEY" {
		t.Errorf("Expected key source 'NURO_API_KEY', got '%s'", res.KeySource)
	}
}

func TestNativeOllamaDefaultModel(t *testing.T) {
	// Test that native Ollama uses correct default model
	origAPIKey := os.Getenv("NURO_API_KEY")
	origProvider := os.Getenv("NURO_PROVIDER")
	origModel := os.Getenv("NURO_MODEL")
	defer func() {
		if origAPIKey == "" {
			os.Unsetenv("NURO_API_KEY")
		} else {
			os.Setenv("NURO_API_KEY", origAPIKey)
		}
		if origProvider == "" {
			os.Unsetenv("NURO_PROVIDER")
		} else {
			os.Setenv("NURO_PROVIDER", origProvider)
		}
		if origModel == "" {
			os.Unsetenv("NURO_MODEL")
		} else {
			os.Setenv("NURO_MODEL", origModel)
		}
	}()

	os.Setenv("NURO_API_KEY", "dummy")
	os.Setenv("NURO_PROVIDER", "ollama")
	os.Setenv("NURO_MODEL", "llama3.1:8b") // Set via env instead of CLI arg

	res, err := ResolveProviderAndModel("") // No model specified via CLI
	if err != nil {
		t.Fatalf("Failed to resolve native Ollama config with env model: %v", err)
	}

	if res.ProviderName != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%s'", res.ProviderName)
	}

	if res.Model != "llama3.1:8b" {
		t.Errorf("Expected model 'llama3.1:8b', got '%s'", res.Model)
	}
}