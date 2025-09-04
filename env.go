package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type providerResolution struct {
	ProviderName string
	Model        string
	APIKey       string
	BaseURL      string // optional (e.g., custom endpoint)
}

var providerEnv = map[string]string{
	"openai":      "OPENAI_API_KEY",
	"anthropic":   "ANTHROPIC_API_KEY",
	"google":      "GOOGLE_API_KEY",       // Gemini
	"azureopenai": "AZURE_OPENAI_API_KEY", // Azure OpenAI
	"openrouter":  "OPENROUTER_API_KEY",
	"groq":        "GROQ_API_KEY",
	"mistral":     "MISTRAL_API_KEY",
	"together":    "TOGETHER_API_KEY",
	"cohere":      "COHERE_API_KEY",
}

// modelPrefix → provider
var modelHints = []struct {
	prefix   string
	provider string
}{
	{"gpt-", "openai"},
	{"o4", "openai"}, // e.g., o4-mini
	{"gpt4", "openai"},
	{"gpt-4", "openai"},
	{"claude", "anthropic"},
	{"gemini", "google"},
	{"mistral", "mistral"},
	{"mixtral", "mistral"},
	{"llama", "groq"}, // heuristic (also possible via openrouter)
}

func resolveProviderAndModel(modelArg string) (*providerResolution, error) {
	// Highest precedence: NURO_*
	nuroKey := os.Getenv("NURO_API_KEY")
	nuroModel := os.Getenv("NURO_MODEL")
	nuroProv := strings.ToLower(os.Getenv("NURO_PROVIDER"))
	nuroBase := os.Getenv("NURO_BASE_URL")

	// If --model is provided, handle $ENV indirection
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

	// If NURO_* provided, prefer them
	if nuroKey != "" {
		// provider can be derived from NURO_PROVIDER, or inferred from model, or default to openai
		prov := nuroProv
		model := firstNonEmpty(cliModel, nuroModel)
		if prov == "" {
			prov = inferProviderFromModel(model)
			if prov == "" {
				prov = "openai"
			}
		}
		if model == "" {
			// Provide a sensible default if user only set NURO_API_KEY
			if prov == "openai" {
				model = "gpt-4o-mini"
			}
		}
		if model == "" {
			return nil, fmt.Errorf("no model specified; set --model or NURO_MODEL")
		}
		return &providerResolution{
			ProviderName: prov,
			Model:        model,
			APIKey:       nuroKey,
			BaseURL: firstNonEmpty(
				nuroBase, os.Getenv("OPENAI_BASE_URL"),
			), // allow provider-specific override
		}, nil
	}

	// Else: discover common provider keys
	found := make([]string, 0, len(providerEnv))
	for prov, env := range providerEnv {
		if os.Getenv(env) != "" {
			found = append(found, prov)
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf(
			"no provider keys found. Set NURO_API_KEY/NURO_MODEL or one of: %s", envList(),
		)
	}

	// Prefer OpenAI if present, else alphabetical
	sort.Strings(found)
	var chosen string
	if contains(found, "openai") {
		chosen = "openai"
	} else {
		chosen = found[0]
	}

	// If --model provided, we may override/infer provider from model
	if cliModel != "" {
		if p := inferProviderFromModel(cliModel); p != "" && contains(found, p) {
			chosen = p
		} else if p != "" && !contains(found, p) {
			return nil, fmt.Errorf(
				"model '%s' implies provider '%s' but no %s key found", cliModel, p,
				strings.ToUpper(providerEnv[p]),
			)
		}
	}

	key := os.Getenv(providerEnv[chosen])
	base := ""
	switch chosen {
	case "openai":
		base = firstNonEmpty(os.Getenv("OPENAI_BASE_URL"), "")
	}

	model := cliModel
	if model == "" {
		// Provide a default model per provider when none supplied
		switch chosen {
		case "openai":
			model = "gpt-4o-mini"
		default:
			// we don't implement others yet; user should pass --model to be explicit
			model = defaultModelFor(chosen)
		}
	}

	// If model implies a different provider than chosen and both keys exist, we can re-route
	if p2 := inferProviderFromModel(model); p2 != "" && p2 != chosen && contains(found, p2) {
		chosen = p2
		key = os.Getenv(providerEnv[chosen])
	}

	return &providerResolution{
		ProviderName: chosen,
		Model:        model,
		APIKey:       key,
		BaseURL:      base,
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
		return "gpt-4o-mini" // via Azure deployment name
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