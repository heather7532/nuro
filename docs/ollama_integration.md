# Ollama Integration for nuro

This document describes the Ollama integration options available in nuro.

## Overview

nuro supports integrating with Ollama in two ways:

1. **Via OpenAI Compatibility API** - Uses Ollama's OpenAI-compatible endpoint with nuro's existing OpenAI adapter
2. **Via Native Ollama Provider** - Uses Ollama's native API directly with a dedicated Ollama provider

## Option 1: OpenAI Compatibility Mode

This approach uses Ollama's OpenAI-compatible API endpoint (`/v1/chat/completions`) with nuro's existing OpenAI provider.

### Configuration

Set these environment variables:

```bash
export NURO_API_KEY=ollama
export NURO_BASE_URL=http://localhost:11434/v1  
export NURO_PROVIDER=openai
```

### Usage Examples

```bash
# Basic usage
nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --verbose

# JSON output
nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --json

# Streaming
nuro -m llama3.1:8b -p "write a haiku about programming" --stream
```

### Alias Pattern

You can create a convenient alias:

```bash
alias nuro-ollama='NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai nuro'
```

Then use it like:

```bash
nuro-ollama -m llama3.1:8b -p "count words" --data "one two three four 5"
```

## Option 2: Native Ollama Provider

This approach uses Ollama's native API endpoint (`/api/generate`) with a dedicated Ollama provider.

### Configuration

Set these environment variables:

```bash
export NURO_API_KEY=dummy        # Ollama doesn't need a real API key
export NURO_BASE_URL=http://localhost:11434
export NURO_PROVIDER=ollama
```

### Usage Examples

```bash
# Basic usage
nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --verbose

# JSON output
nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --json

# Streaming
nuro -m llama3.1:8b -p "write a haiku about programming" --stream

# Using default model (llama3.1:8b)
nuro -p "what is 2+2?"
```

### Alias Pattern for Native Provider

```bash
alias nuro-ollama='NURO_API_KEY=dummy NURO_BASE_URL=http://localhost:11434 NURO_PROVIDER=ollama nuro'
```

## Differences Between Approaches

| Feature | OpenAI Compatibility | Native Ollama |
|---------|---------------------|---------------|
| API Endpoint | `/v1/chat/completions` | `/api/generate` |
| Message Format | OpenAI chat format | Ollama prompt format |
| Authentication | Token-based (dummy) | None required |
| Base URL | `http://localhost:11434/v1` | `http://localhost:11434` |
| Provider Value | `openai` | `ollama` |
| Token Usage Stats | Limited | Full Ollama stats |

## Prerequisites

1. **Install Ollama**: Download from [ollama.ai](https://ollama.ai)

2. **Start Ollama Server**: 
   ```bash
   ollama serve
   ```

3. **Pull Models**: 
   ```bash
   ollama pull llama3.1:8b
   ollama pull mistral:7b
   # ... other models
   ```

4. **Verify Ollama is Running**:
   ```bash
   curl http://localhost:11434/api/tags
   ```

## Testing

The project includes comprehensive test scripts for both approaches:

### Test Scripts

1. **`test_ollama_integration.sh`** - Tests OpenAI compatibility mode
2. **`test_ollama_alias.sh`** - Tests OpenAI compatibility mode using alias pattern  
3. **`test_ollama_native.sh`** - Tests native Ollama provider

### Running Tests

```bash
# Make scripts executable
chmod +x test_ollama_*.sh

# Test OpenAI compatibility mode
./test_ollama_integration.sh

# Test alias pattern
./test_ollama_alias.sh  

# Test native Ollama provider
./test_ollama_native.sh

# Run Go unit tests
go test ./... -v
```

## Supported Models

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

## Troubleshooting

### Ollama Not Running
```
ERROR: Ollama is not running. Please start Ollama first:
  ollama serve
  ollama pull llama3.1:8b  # or another model
```

### Model Not Available
Pull the model first:
```bash
ollama pull llama3.1:8b
```

### Connection Refused
Ensure Ollama is running on the correct host and port. Default is `localhost:11434`.

### Wrong Provider
Make sure `NURO_PROVIDER` matches your intended approach:
- `openai` for compatibility mode
- `ollama` for native provider

## Examples

### Word Counting
```bash
# OpenAI compatibility mode
NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai \
  nuro -m llama3.1:8b -p "count words" --data "one two three four 5"

# Native Ollama provider  
NURO_API_KEY=dummy NURO_BASE_URL=http://localhost:11434 NURO_PROVIDER=ollama \
  nuro -m llama3.1:8b -p "count words" --data "one two three four 5"
```

### Code Analysis
```bash
echo 'def factorial(n): return 1 if n <= 1 else n * factorial(n-1)' | \
  nuro-ollama -m llama3.1:8b -p "explain what this code does"
```

### Data Summarization
```bash
cat large_file.txt | \
  nuro-ollama -m llama3.1:8b -p "summarize the key points"
```

This integration provides flexible options for using Ollama with nuro, whether you prefer the OpenAI-compatible approach or the native Ollama API.