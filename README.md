# nuro

nuro is a light CLI that simplifies calling LLM APIs. With a unified interface, you can focus on your prompt,
and nuro will take care of the complexity of talking directly to the provider API.

## Getting Started

Here are a few examples to help you get started with nuro:

### Basic Usage with Ollama (OpenAI Compatibility Mode)
```bash
# Set up environment variables
export NURO_API_KEY=ollama
export NURO_BASE_URL=http://localhost:11434/v1  
export NURO_PROVIDER=openai

# Run a basic command
echo "{\"text\": \"Heather\"}" | nuro -p "write a haiku for {{text}}" --json
```

### Native Ollama Provider
```bash
# Use the native Ollama provider
export NURO_API_KEY=dummy
export NURO_BASE_URL=http://localhost:11434
export NURO_PROVIDER=ollama

# Basic completion
text="one two three four 5"
nuro -m llama3.1:8b -p "count words" --data "$text" --verbose

# Streaming output
nuro -m llama3.1:8b -p "write a haiku about programming" --stream

# Using stdin
ls -la | nuro -p "explain these files in a table" --verbose
```

### Using Aliases
```bash
# Create a convenient alias
alias nuro-ollama='NURO_API_KEY=dummy NURO_BASE_URL=http://localhost:11434 NURO_PROVIDER=ollama nuro'

# Use the alias
nuro-ollama -m phi3:mini -p "explain quantum computing basics" --verbose
```

### Different Input Methods
```bash
# Prompt from stdin, data from flag
echo "Summarize this:" | nuro -p --data "The quick brown fox jumps over the lazy dog. This is a classic pangram."

# Model from environment variable
export MYMODEL=llama3.1:8b
nuro -m $MYMODEL -p "Explain black holes"

# Using data from a file
nuro -m mistral:7b -p "translate to French" --data-file my_document.txt
```

### Advanced Usage
```bash
# JSON output with full details
nuro -m llama3.1:8b -p "explain this code" --data 'func main() { fmt.Println("Hello") }' --json

# Stream with different model
nuro -m codellama:13b -p "write a simple Go function that adds two numbers" --stream

# Analysis of code
echo 'def factorial(n): return 1 if n <= 1 else n * factorial(n-1)' | \
  nuro -m llama3.1:8b -p "explain what this code does"
```

## Prerequisites
1. **Install Ollama**: Download from [ollama.ai](https://ollama.ai)
2. **Start Ollama Server**: `ollama serve`
3. **Pull Models**: `ollama pull llama3.1:8b`, `ollama pull mistral:7b`, etc.
4. **Verify Ollama is Running**: `curl http://localhost:11434/api/tags`

For more detailed information, check the [Ollama Integration Guide](docs/ollama_integration.md).

## Supported

### Provider Integration Methods

- **OpenAI Compatibility Mode**
  - Uses Ollama's OpenAI-compatible endpoint (`/v1/chat/completions`)
  - Configuration:
    - `NURO_API_KEY=ollama`
    - `NURO_BASE_URL=http://localhost:11434/v1`
    - `NURO_PROVIDER=openai`

- **Native Ollama Provider**
  - Uses Ollama's native API endpoint (`/api/generate`)
  - Configuration:
    - `NURO_API_KEY=dummy` (Ollama doesn't need a real API key)
    - `NURO_BASE_URL=http://localhost:11434`
    - `NURO_PROVIDER=ollama`

### Features

| Feature | Status |
|---------|--------|
| **Model Specification** | ✅ Supported via `-m` flag or `NURO_MODEL` env var |
| **Environment Variable Resolution** | ✅ Supported with NURO_* precedence |
| **Data Input** | ✅ Supported via `--data`, `--data-file`, or stdin |
| **Streaming Output** | ✅ Supported with `--stream` flag |
| **JSON Output** | ✅ Supported with `--json` flag |
| **Stdin/Stdout** | ✅ Full support for pipes and redirects |
| **Local Model Support** | ✅ All models available in local Ollama installation |
| **Temperature Control** | ✅ Supported via `--temperature` flag |
| **Max Tokens** | ✅ Supported via `--max-tokens` flag |
| **Top-p Sampling** | ✅ Supported via `--top-p` flag |
| **Request Timeout** | ✅ Supported via `--timeout` flag |
| **Verbose Mode** | ✅ Supported via `--verbose` flag |

### Supported Models

Any model available in your local Ollama installation can be used. Common examples:

- `llama3.1:8b`
- `llama3.1:70b`
- `mistral:7b`
- `codellama:13b`
- `phi3:mini`

Check available models:
```bash
curl -s http://localhost:11434/api/tags | jq '.models[].name'
```

### Testing

The project includes comprehensive test scripts:

- `test_ollama_integration.sh` - Tests OpenAI compatibility mode
- `test_ollama_alias.sh` - Tests alias pattern
- `test_ollama_native.sh` - Tests native Ollama provider

Run tests with:
```bash
chmod +x test_ollama_*.sh
./test_ollama_integration.sh
./test_ollama_alias.sh
./test_ollama_native.sh
```

For a detailed guide, see the [Ollama Integration Documentation](docs/ollama_integration.md).