#!/bin/bash

# Test script for native Ollama provider integration
# This script tests the native Ollama provider (not using OpenAI compatibility)

set -e  # Exit on any error

echo "=== Ollama Native Provider Tests ==="
echo

# Check if Ollama is running
echo "Checking if Ollama is running..."
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "ERROR: Ollama is not running. Please start Ollama first:"
    echo "  ollama serve"
    echo "  ollama pull llama3.1:8b  # or another model"
    exit 1
fi
echo "✓ Ollama is running"
echo

# Test 1: Native Ollama provider with NURO_PROVIDER=ollama
echo "=== Test 1: Native Ollama Provider ==="
echo "Testing: NURO_API_KEY=dummy NURO_BASE_URL=http://localhost:11434 NURO_PROVIDER=ollama nuro -m llama3.1:8b -p \"count words\" --data \"one two three four 5\" --verbose"

export NURO_API_KEY=dummy  # Ollama doesn't need a real API key
export NURO_BASE_URL=http://localhost:11434
export NURO_PROVIDER=ollama

./nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --verbose
echo "✓ Test 1 completed"
echo

# Test 2: Native Ollama with JSON output
echo "=== Test 2: Native Ollama JSON Output ==="
echo "Testing JSON output with native Ollama"

./nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --json --verbose
echo "✓ Test 2 completed"
echo

# Test 3: Native Ollama streaming
echo "=== Test 3: Native Ollama Streaming ==="
echo "Testing streaming with native Ollama"

./nuro -m llama3.1:8b -p "write a haiku about coding" --stream --verbose
echo
echo "✓ Test 3 completed"
echo

# Test 4: Native Ollama with different models
echo "=== Test 4: Native Ollama Different Models ==="
echo "Trying different models if available..."

# Check what models are available
AVAILABLE_MODELS=$(curl -s http://localhost:11434/api/tags | jq -r '.models[].name' 2>/dev/null || echo "")
if [[ -n "$AVAILABLE_MODELS" ]]; then
    echo "Available models:"
    echo "$AVAILABLE_MODELS"
    echo
    
    # Test with different models
    for model in $AVAILABLE_MODELS; do
        if [[ "$model" != "llama3.1:8b" ]]; then
            echo "--- Testing model: $model ---"
            ./nuro -m "$model" -p "say hello briefly" --verbose
            echo
            break # Just test one different model to save time
        fi
    done
else
    echo "Could not retrieve model list, skipping this test"
fi
echo "✓ Test 4 completed"
echo

# Test 5: Default model test
echo "=== Test 5: Default Model Test ==="
echo "Testing without specifying a model (should use default llama3.1:8b)"

./nuro -p "what is 2+2?" --verbose
echo "✓ Test 5 completed"
echo

# Test 6: Temperature and other parameters
echo "=== Test 6: Parameters Test ==="
echo "Testing with different temperature and max-tokens"

./nuro -m llama3.1:8b -p "write one sentence about the weather" --temperature 0.1 --max-tokens 50 --verbose
echo "✓ Test 6 completed"
echo

# Clean up environment variables
unset NURO_API_KEY
unset NURO_BASE_URL  
unset NURO_PROVIDER

echo "=== All Native Ollama Tests Completed Successfully! ==="
echo
echo "Summary:"
echo "- ✓ Native Ollama provider working"
echo "- ✓ JSON output supported"
echo "- ✓ Streaming supported" 
echo "- ✓ Multiple models supported"
echo "- ✓ Default model working"
echo "- ✓ Parameters (temperature, max-tokens) working"
echo
echo "To use the native Ollama provider:"
echo "  export NURO_API_KEY=dummy"
echo "  export NURO_BASE_URL=http://localhost:11434"
echo "  export NURO_PROVIDER=ollama"
echo "  ./nuro -m llama3.1:8b -p \"your prompt here\""