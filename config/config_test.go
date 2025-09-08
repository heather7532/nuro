package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, dir string, json string) string {
	t.Helper()
	path := filepath.Join(dir, ".nuro")
	if err := os.WriteFile(path, []byte(json), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadValidateAndGetProfile(t *testing.T) {
	tmp := t.TempDir()

	// env used by the config for substitution
	t.Setenv("CUSTOM_API_KEY", "custom_test_key")
	t.Setenv("CUSTOM_BASE_URL", "http://test.local:11434")
	t.Setenv("CUSTOM_MODEL", "gpt-4o-mini")

	cfgJSON := `{
  "default": "test1",
  "profiles": {
    "test1": {
      "api_key": "$CUSTOM_API_KEY",
      "base_url": "${CUSTOM_BASE_URL}",
      "provider": "openai",
      "model": "$CUSTOM_MODEL",
      "max_tokens": 1500,
      "temperature": 1.0,
      "top_p": 0.8
    }
  }
}`
	_ = writeTempConfig(t, tmp, cfgJSON)

	// run from the temp dir so LoadConfig finds ./.nuro
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// restore working dir when test exits
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}
	if cfg.Default != "test1" {
		t.Fatalf("expected default 'test1', got %q", cfg.Default)
	}

	// Validate should pass
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate unexpected error: %v", err)
	}

	// GetProfile should resolve env vars
	p, err := cfg.GetProfile("test1")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if p.APIKey != "custom_test_key" {
		t.Errorf("APIKey not resolved: %q", p.APIKey)
	}
	if p.BaseURL != "http://test.local:11434" {
		t.Errorf("BaseURL not resolved: %q", p.BaseURL)
	}
	if p.Provider != "openai" {
		t.Errorf("Provider mismatch: %q", p.Provider)
	}
	if p.Model != "gpt-4o-mini" {
		t.Errorf("Model not resolved: %q", p.Model)
	}
	if p.MaxTokens != 1500 {
		t.Errorf("MaxTokens mismatch: %d", p.MaxTokens)
	}
	if p.Temperature != 1.0 {
		t.Errorf("Temperature mismatch: %v", p.Temperature)
	}
	if p.TopP != 0.8 {
		t.Errorf("TopP mismatch: %v", p.TopP)
	}
}

func TestConfigValidationErrorForBadProvider(t *testing.T) {
	tmp := t.TempDir()
	cfgJSON := `{
  "default": "bad",
  "profiles": {
    "bad": {
      "provider": "not-a-provider",
      "model": "anything"
    }
  }
}`
	_ = writeTempConfig(t, tmp, cfgJSON)

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil cfg")
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid provider, got nil")
	}
}

func TestApplyProfileSetsEnv(t *testing.T) {
	tmp := t.TempDir()

	cfgJSON := `{
  "default": "test1",
  "profiles": {
    "test1": {
      "api_key": "testkey123",
      "base_url": "https://api.test.com",
      "provider": "openai",
      "model": "gpt-4o-mini",
      "max_tokens": 1500,
      "temperature": 1.0,
      "top_p": 0.8
    }
  }
}`
	_ = writeTempConfig(t, tmp, cfgJSON)

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Clear any existing NURO_* env to avoid false positives
	for _, k := range []string{
		"NURO_API_KEY", "NURO_BASE_URL", "NURO_PROVIDER", "NURO_MODEL",
		"NURO_MAX_TOKENS", "NURO_TEMPERATURE", "NURO_TOP_P",
	} {
		_ = os.Unsetenv(k)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil cfg")
	}

	// This uses Config.ApplyProfile which internally resolves then Profile.Apply()
	if err := cfg.ApplyProfile("test1"); err != nil {
		t.Fatalf("ApplyProfile: %v", err)
	}

	// Check envs set by Profile.Apply (note temperature/top_p formatted to 2 decimals)
	if os.Getenv("NURO_API_KEY") != "testkey123" {
		t.Errorf("NURO_API_KEY not set correctly, got %q", os.Getenv("NURO_API_KEY"))
	}
	if os.Getenv("NURO_BASE_URL") != "https://api.test.com" {
		t.Errorf("NURO_BASE_URL not set correctly, got %q", os.Getenv("NURO_BASE_URL"))
	}
	if os.Getenv("NURO_PROVIDER") != "openai" {
		t.Errorf("NURO_PROVIDER not set correctly, got %q", os.Getenv("NURO_PROVIDER"))
	}
	if os.Getenv("NURO_MODEL") != "gpt-4o-mini" {
		t.Errorf("NURO_MODEL not set correctly, got %q", os.Getenv("NURO_MODEL"))
	}
	if os.Getenv("NURO_MAX_TOKENS") != "1500" {
		t.Errorf("NURO_MAX_TOKENS not set correctly, got %q", os.Getenv("NURO_MAX_TOKENS"))
	}
	if os.Getenv("NURO_TEMPERATURE") != "1.00" {
		t.Errorf("NURO_TEMPERATURE not set correctly, got %q", os.Getenv("NURO_TEMPERATURE"))
	}
	if os.Getenv("NURO_TOP_P") != "0.80" {
		t.Errorf("NURO_TOP_P not set correctly, got %q", os.Getenv("NURO_TOP_P"))
	}
}

func TestResolveEnvVarsStandalone(t *testing.T) {
	// defensive unit test for the internal resolver behavior your config.go uses
	t.Setenv("FOO", "bar")
	t.Setenv("BAZ", "qux")
	in := "x $FOO y ${BAZ} z $MISSING"
	out := resolveEnvVars(in)
	if !strings.Contains(out, "x bar y qux z") {
		t.Fatalf("resolveEnvVars failed, got: %q", out)
	}
}
