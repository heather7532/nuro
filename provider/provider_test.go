package provider

import "testing"

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
