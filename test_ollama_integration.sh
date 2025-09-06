#!/bin/bash

# Test script for Ollama integration with nuro
# This script tests the Ollama integration using NURO env vars to use the OpenAI adapter with Ollama's compat API

set -e  # Exit on any error

echo "=== Ollama Integration Tests ==="
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

# Test 1: Direct environment variable usage
echo "=== Test 1: Direct Environment Variables ==="
echo "Testing: NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai nuro -m llama3.1:8b -p \"count words\" --data \"one two three four 5\" --verbose"

export NURO_API_KEY=ollama
export NURO_BASE_URL=http://localhost:11434/v1
export NURO_PROVIDER=openai

./nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --verbose
echo "✓ Test 1 completed"
echo

# Test 2: JSON output
echo "=== Test 2: JSON Output ==="
echo "Testing JSON output with Ollama"

./nuro -m llama3.1:8b -p "count words" --data "one two three four 5" --json --verbose
echo "✓ Test 2 completed"
echo

# Test 3: Streaming output
echo "=== Test 3: Streaming Output ==="
echo "Testing streaming with Ollama"

./nuro -m llama3.1:8b -p "write a haiku about programming" --stream --verbose
echo
echo "✓ Test 3 completed"
echo

# Test 4: Different model test (if available)
echo "=== Test 4: Different Ollama Model (if available) ==="
echo "Trying to use a different model if available..."

# Check what models are available
AVAILABLE_MODELS=$(curl -s http://localhost:11434/api/tags | jq -r '.models[].name' 2>/dev/null || echo "")
if [[ -n "$AVAILABLE_MODELS" ]]; then
    echo "Available models:"
    echo "$AVAILABLE_MODELS"
    echo
    
    # Try to find a different model
    FIRST_MODEL=$(echo "$AVAILABLE_MODELS" | head -n 1)
    echo "Testing with model: $FIRST_MODEL"
    ./nuro -m "$FIRST_MODEL" -p "say hello" --verbose
else
    echo "Could not retrieve model list, skipping this test"
fi
echo "✓ Test 4 completed"
echo

# Clean up environment variables
unset NURO_API_KEY
unset NURO_BASE_URL  
unset NURO_PROVIDER

echo "=== All Ollama Integration Tests Completed Successfully! ==="