package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Profile represents the configuration for a specific LLM setup
type Profile struct {
	APIKey      string  `json:"api_key,omitempty"`
	BaseURL     string  `json:"base_url,omitempty"`
	Provider    string  `json:"provider,omitempty"`
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// Config represents the structure of the .nuro configuration file
type Config struct {
	Default  string             `json:"default,omitempty"`
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

// FindConfigFile looks for .nuro file in current directory, then in home directory
func FindConfigFile() (string, bool) {
	// First, check current directory
	currentDir, err := os.Getwd()
	if err == nil {
		configPath := filepath.Join(currentDir, ".nuro")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, true
		}
	}

	// Then check home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	configPath := filepath.Join(homeDir, ".nuro")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, true
	}

	return "", false
}

// LoadConfig reads and parses the .nuro configuration file
func LoadConfig() (*Config, error) {
	configPath, found := FindConfigFile()
	if !found {
		return nil, nil // No config file found is not an error
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse .nuro config file: %w", err)
	}

	return &config, nil
}

// GetProfile returns a specific profile by name, with environment variable substitution
func (c *Config) GetProfile(name string) (*Profile, error) {
	if c.Profiles == nil {
		return nil, fmt.Errorf("no profiles defined in config")
	}

	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found in .nuro config", name)
	}

	// Apply environment variable substitution to profile values
	resolved := Profile{
		APIKey:      resolveEnvVars(profile.APIKey),
		BaseURL:     resolveEnvVars(profile.BaseURL),
		Provider:    profile.Provider,
		Model:       resolveEnvVars(profile.Model),
		MaxTokens:   profile.MaxTokens,
		Temperature: profile.Temperature,
		TopP:        profile.TopP,
	}

	return &resolved, nil
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	if c.Profiles == nil {
		return fmt.Errorf("config file must contain 'profiles' object")
	}

	// If a default profile is specified, validate that it exists
	if c.Default != "" {
		if _, exists := c.Profiles[c.Default]; !exists {
			return fmt.Errorf("default profile '%s' not found in 'profiles'", c.Default)
		}
	}

	// Validate each profile
	for name, profile := range c.Profiles {
		// Validate provider
		if profile.Provider != "" {
			validProviders := []string{
				"openai", "anthropic", "google", "azureopenai", "openrouter", "groq", "mistral",
				"together", "cohere", "ollama",
			}
			valid := false
			for _, prov := range validProviders {
				if profile.Provider == prov {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf(
					"invalid provider '%s' in profile '%s': must be one of openai, anthropic, google, azureopenai, openrouter, groq, mistral, together, cohere, ollama",
					profile.Provider, name,
				)
			}
		}

		if profile.MaxTokens < 0 {
			return fmt.Errorf("max_tokens in profile '%s' must be non-negative", name)
		}

		if profile.Temperature < 0 || profile.Temperature > 2.0 {
			return fmt.Errorf("temperature in profile '%s' must be between 0 and 2", name)
		}

		if profile.TopP < 0 || profile.TopP > 1.0 {
			return fmt.Errorf("top_p in profile '%s' must be between 0 and 1", name)
		}
	}

	return nil
}

// Apply applies the default profile's configuration by setting environment variables
func (c *Config) Apply() error {
	if c.Profiles == nil {
		return nil // No profiles to apply
	}

	// Choose profile: CLI flag > config.Default > first profile
	var profileName string
	if c.Default != "" {
		profileName = c.Default
	} else {
		// Pick the first profile as default
		for name := range c.Profiles {
			profileName = name
			break
		}
	}

	// Get and apply the chosen profile
	profile, err := c.GetProfile(profileName)
	if err != nil {
		return err
	}

	return profile.Apply()
}

// ApplyProfile applies a specific named profile's configuration by setting environment variables
func (c *Config) ApplyProfile(name string) error {
	if c.Profiles == nil {
		return fmt.Errorf("no profiles defined in config")
	}

	profile, err := c.GetProfile(name)
	if err != nil {
		return err
	}

	return profile.Apply()
}

// ApplyProfile applies a profile's configuration by setting environment variables
func (p *Profile) Apply() error {
	// Set environment variables
	if p.APIKey != "" {
		if err := os.Setenv("NURO_API_KEY", p.APIKey); err != nil {
			return fmt.Errorf("failed to set NURO_API_KEY: %w", err)
		}
	}
	if p.BaseURL != "" {
		if err := os.Setenv("NURO_BASE_URL", p.BaseURL); err != nil {
			return fmt.Errorf("failed to set NURO_BASE_URL: %w", err)
		}
	}
	if p.Provider != "" {
		if err := os.Setenv("NURO_PROVIDER", p.Provider); err != nil {
			return fmt.Errorf("failed to set NURO_PROVIDER: %w", err)
		}
	}
	if p.Model != "" {
		if err := os.Setenv("NURO_MODEL", p.Model); err != nil {
			return fmt.Errorf("failed to set NURO_MODEL: %w", err)
		}
	}
	if p.MaxTokens > 0 {
		if err := os.Setenv("NURO_MAX_TOKENS", strconv.Itoa(p.MaxTokens)); err != nil {
			return fmt.Errorf("failed to set NURO_MAX_TOKENS: %w", err)
		}
	}
	if p.Temperature > 0 {
		if err := os.Setenv("NURO_TEMPERATURE", fmt.Sprintf("%.2f", p.Temperature)); err != nil {
			return fmt.Errorf("failed to set NURO_TEMPERATURE: %w", err)
		}
	}
	if p.TopP > 0 {
		if err := os.Setenv("NURO_TOP_P", fmt.Sprintf("%.2f", p.TopP)); err != nil {
			return fmt.Errorf("failed to set NURO_TOP_P: %w", err)
		}
	}

	return nil
}

// resolveEnvVars substitutes environment variable references (e.g., "$VAR") in a string
func resolveEnvVars(value string) string {
	if value == "" {
		return ""
	}

	re := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)|\${([A-Za-z_][A-Za-z0-9_]*)}`)

	return re.ReplaceAllStringFunc(
		value, func(match string) string {
			varName := match[1:] // Remove the $ prefix
			if strings.HasPrefix(match, "${") {
				varName = match[2 : len(match)-1] // Extract content between ${}
			}
			if envValue := os.Getenv(varName); envValue != "" {
				return envValue
			}
			// Return original if env var not found or empty
			return match
		},
	)
}
