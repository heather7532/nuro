package resolver

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/heather7532/nuro/provider"
)

var providerEnv = map[string]string{
	"openai":      "OPENAI_API_KEY",
	"anthropic":   "ANTHROPIC_API_KEY",
	"google":      "GOOGLE_API_KEY",
	"azureopenai": "AZURE_OPENAI_API_KEY",
	"openrouter":  "OPENROUTER_API_KEY",
	"groq":        "GROQ_API_KEY",
	"mistral":     "MISTRAL_API_KEY",
	"together":    "TOGETHER_API_KEY",
	"cohere":      "COHERE_API_KEY",
}

var modelHints = []struct {
	prefix   string
	provider string
}{
	{"gpt-", "openai"},
	{"o4", "openai"},
	{"gpt4", "openai"},
	{"gpt-4", "openai"},
	{"claude", "anthropic"},
	{"gemini", "google"},
	{"mistral", "mistral"},
	{"mixtral", "mistral"},
	{"llama", "groq"},
}

func ResolveProviderAndModel(modelArg string) (*provider.ProviderResolution, error) {
	// Handle model argument with $ENV indirection
	var cliModel string
	if modelArg != "" {
		if strings.HasPrefix(modelArg, "$") {
			ref := strings.TrimPrefix(modelArg, "$")
			cliModel = os.Getenv(ref)
			if cliModel == "" {
				return nil, fmt.Errorf("model env '%s' is empty or unset", ref)
			}
		} else {
			cliModel = modelArg
		}
	}

	// Check for NURO_* variables first (highest precedence)
	if nuroKey := os.Getenv("NURO_API_KEY"); nuroKey != "" {
		return resolveWithNuroVars(nuroKey, cliModel)
	}

	// Auto-discover from common provider keys
	return autoDiscoverProvider(cliModel)
}

func resolveWithNuroVars(nuroKey, cliModel string) (*provider.ProviderResolution, error) {
	nuroModel := os.Getenv("NURO_MODEL")
	nuroProv := strings.ToLower(os.Getenv("NURO_PROVIDER"))
	nuroBase := os.Getenv("NURO_BASE_URL")

	prov := nuroProv
	model := firstNonEmpty(cliModel, nuroModel)

	if prov == "" {
		prov = inferProviderFromModel(model)
		if prov == "" {
			prov = "openai"
		}
	}

	if model == "" {
		if prov == "openai" {
			model = "gpt-4o-mini"
		} else {
			return nil, fmt.Errorf("no model specified; set --model or NURO_MODEL")
		}
	}

	return &provider.ProviderResolution{
		ProviderName: prov,
		Model:        model,
		APIKey:       nuroKey,
		BaseURL:      firstNonEmpty(nuroBase, os.Getenv("OPENAI_BASE_URL")),
		KeySource:    "NURO_API_KEY",
	}, nil
}

func autoDiscoverProvider(cliModel string) (*provider.ProviderResolution, error) {
	found := make([]string, 0, len(providerEnv))
	for prov, env := range providerEnv {
		if os.Getenv(env) != "" {
			found = append(found, prov)
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf(
			"no provider keys found. Set NURO_API_KEY/NURO_MODEL or one of: %s",
			envList(),
		)
	}

	// Prefer OpenAI if present, else alphabetical
	sort.Strings(found)
	chosen := found[0]
	if contains(found, "openai") {
		chosen = "openai"
	}

	// Override provider if model implies a different one
	if cliModel != "" {
		if p := inferProviderFromModel(cliModel); p != "" && contains(found, p) {
			chosen = p
		} else if p != "" && !contains(found, p) {
			return nil, fmt.Errorf(
				"model '%s' implies provider '%s' but no %s key found",
				cliModel, p, strings.ToUpper(providerEnv[p]),
			)
		}
	}

	key := os.Getenv(providerEnv[chosen])
	model := cliModel
	if model == "" {
		model = defaultModelFor(chosen)
	}

	baseURL := ""
	if chosen == "openai" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	return &provider.ProviderResolution{
		ProviderName: chosen,
		Model:        model,
		APIKey:       key,
		BaseURL:      baseURL,
		KeySource:    strings.ToUpper(providerEnv[chosen]),
	}, nil
}

func envList() string {
	var parts []string
	for _, env := range providerEnv {
		parts = append(parts, env)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func contains(ss []string, t string) bool {
	for _, s := range ss {
		if s == t {
			return true
		}
	}
	return false
}

func inferProviderFromModel(model string) string {
	m := strings.ToLower(model)
	for _, h := range modelHints {
		if strings.HasPrefix(m, h.prefix) {
			return h.provider
		}
	}
	return ""
}

func defaultModelFor(provider string) string {
	switch provider {
	case "openai":
		return "gpt-4o-mini"
	case "anthropic":
		return "claude-3-5-sonnet"
	case "google":
		return "gemini-1.5-pro"
	case "groq":
		return "llama3-70b-8192"
	case "mistral":
		return "mistral-large-latest"
	case "openrouter":
		return "openrouter/auto"
	case "together":
		return "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"
	case "cohere":
		return "command-r-plus"
	case "azureopenai":
		return "gpt-4o-mini"
	default:
		return "unknown"
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
