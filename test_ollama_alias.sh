#!/bin/bash

# Test script for Ollama integration using the current environment settings.
# This version does NOT override your environment. It requires you to have
# NURO_API_KEY and NURO_BASE_URL (and optionally NURO_PROVIDER) set in your shell.

set -euo pipefail

echo "=== Ollama Alias Integration Tests (using current environment) ==="

echo "Environment summary:"
echo "  NURO_API_KEY=${NURO_API_KEY:-<unset>}"
echo "  NURO_BASE_URL=${NURO_BASE_URL:-<unset>}"
echo "  NURO_PROVIDER=${NURO_PROVIDER:-<unset>}"

echo
# Require at least NURO_API_KEY and NURO_BASE_URL to be present
if [ -z "${NURO_API_KEY:-}" ] || [ -z "${NURO_BASE_URL:-}" ]; then
    cat <<EOF
ERROR: Required environment variables are not set.
Please set NURO_API_KEY and NURO_BASE_URL to run these tests.

For local Ollama (recommended for development), you can run:

  export NURO_API_KEY=ollama
  export NURO_BASE_URL=http://localhost:11434/v1
  export NURO_PROVIDER=openai   # optional: 'openai' for Ollama OpenAI adapter

Then re-run this script.
EOF
    exit 1
fi

# Derive a health-check URL for the Ollama server. NURO_BASE_URL may include /v1
# (OpenAI-compat mode) or be the native Ollama URL (no /v1). Use /api/tags on the
# underlying server to check availability.
base="$NURO_BASE_URL"
# Strip trailing slash
base="${base%/}"
if [[ "$base" == */v1 ]]; then
    server_check="${base%/v1}/api/tags"
else
    server_check="$base/api/tags"
fi

echo "Checking if Ollama server is reachable at: $server_check"
if ! curl -s "$server_check" > /dev/null 2>&1; then
    echo "ERROR: Ollama server not reachable at $server_check"
    echo "If you're running Ollama locally, start it with: ollama serve"
    exit 1
fi

echo "✓ Ollama server reachable"

echo
# Helper to run the local ./nuro binary using the current environment
nuro_run() {
    if [ ! -x ./nuro ]; then
        echo "ERROR: ./nuro binary not found or not executable. Build it first: make build or go build -o nuro ."
        exit 1
    fi
    ./nuro "$@"
}

# Test 1: Basic word count test
echo "=== Test 1: Basic Word Count ==="
echo "Running: nuro -m llama3.1:8b -p 'count words' --data 'one two three four 5' --verbose"
nuro_run -m llama3.1:8b -p "count words" --data "one two three four 5" --verbose
echo "✓ Test 1 completed"
echo

# Test 2: JSON output
echo "=== Test 2: JSON Output ==="
echo "Running: nuro -m llama3.1:8b -p 'count words' --data 'one two three four 5' --json --verbose"
nuro_run -m llama3.1:8b -p "count words" --data "one two three four 5" --json --verbose
echo "✓ Test 2 completed"
echo

# Test 3: Streaming
echo "=== Test 3: Streaming ==="
echo "Running: nuro -m llama3.1:8b -p 'write a short poem about code' --stream --verbose"
nuro_run -m llama3.1:8b -p "write a short poem about code" --stream --verbose
echo
echo "✓ Test 3 completed"
echo

# Test 4: Various prompts
echo "=== Test 4: Various Prompts ==="

echo "--- Summarization test ---"
nuro_run -m llama3.1:8b -p "summarize this" --data "The quick brown fox jumps over the lazy dog. This is a classic pangram used in typing practice." --verbose

echo
echo "--- Analysis test ---"
nuro_run -m llama3.1:8b -p "explain what this code does" --data "def factorial(n): return 1 if n <= 1 else n * factorial(n-1)" --verbose

echo "✓ Test 4 completed"
echo

# Test 5: Pipeline test
echo "=== Test 5: Pipeline Test ==="
echo "Running pipeline: echo 'Hello world from pipeline' | nuro -m llama3.1:8b -p 'translate to Spanish' --verbose"
echo "Hello world from pipeline" | nuro_run -m llama3.1:8b -p "translate to Spanish" --verbose

echo "✓ Test 5 completed"
echo

cat <<'EOF'
To permanently use an alias similar to the previous script, add this to your shell profile:

  alias nuro-ollama='NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai nuro'

But this test suite intentionally uses your current environment variables and does not override them.
EOF

echo "=== All Ollama Alias Tests Completed ==="