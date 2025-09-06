#!/bin/bash

# Test script for Ollama integration using alias pattern
# This script tests the alias approach: alias nuro-ollama='NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai nuro'

set -e  # Exit on any error

echo "=== Ollama Alias Integration Tests ==="
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

# Create the alias equivalent as a function for testing
nuro-ollama() {
    NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai ./nuro "$@"
}

# Test 1: Basic word count test with alias
echo "=== Test 1: Basic Word Count with Alias ==="
echo "Testing: nuro-ollama -m llama3.1:8b -p \"count words\" --data \"one two three four 5\" --verbose"

nuro-ollama -m llama3.1:8b -p "count words" --data "one two three four 5" --verbose
echo "✓ Test 1 completed"
echo

# Test 2: JSON output with alias
echo "=== Test 2: JSON Output with Alias ==="
echo "Testing JSON output with nuro-ollama"

nuro-ollama -m llama3.1:8b -p "count words" --data "one two three four 5" --json --verbose
echo "✓ Test 2 completed"
echo

# Test 3: Streaming with alias
echo "=== Test 3: Streaming with Alias ==="
echo "Testing streaming with nuro-ollama"

nuro-ollama -m llama3.1:8b -p "write a short poem about code" --stream --verbose
echo
echo "✓ Test 3 completed"
echo

# Test 4: Different prompts with alias
echo "=== Test 4: Various Prompts with Alias ==="
echo "Testing different types of prompts"

echo "--- Summarization test ---"
nuro-ollama -m llama3.1:8b -p "summarize this" --data "The quick brown fox jumps over the lazy dog. This is a classic pangram used in typing practice." --verbose
echo

echo "--- Analysis test ---"
nuro-ollama -m llama3.1:8b -p "explain what this code does" --data "def factorial(n): return 1 if n <= 1 else n * factorial(n-1)" --verbose
echo

echo "✓ Test 4 completed"
echo

# Test 5: Pipeline test with alias
echo "=== Test 5: Pipeline Test with Alias ==="
echo "Testing pipeline input"

echo "Hello world from pipeline" | nuro-ollama -m llama3.1:8b -p "translate to Spanish" --verbose
echo "✓ Test 5 completed"
echo

echo "=== Showing the alias definition you can use ==="
echo "To use this permanently, add this to your shell profile:"
echo "alias nuro-ollama='NURO_API_KEY=ollama NURO_BASE_URL=http://localhost:11434/v1 NURO_PROVIDER=openai nuro'"
echo

echo "=== All Ollama Alias Tests Completed Successfully! ==="