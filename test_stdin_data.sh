#!/bin/bash

echo "=== Test 1: Simple data via stdin with prompt flag ==="
echo "apple banana cherry" | ./nuro -p "count the words" --verbose

echo -e "\n=== Test 2: Process data simulation ==="
echo -e "PID\tNAME\tCPU\n1234\tchrome\t45%\n5678\tfirefox\t32%" | ./nuro -p "show me the highest cpu process" --verbose

echo -e "\n=== Test 3: JSON data ==="
echo '{"users": [{"name": "john", "age": 30}, {"name": "jane", "age": 25}]}' | ./nuro -p "extract the names" --verbose