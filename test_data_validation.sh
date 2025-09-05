#!/bin/bash

echo "=== nuro Data Size Validation Tests ==="
echo

echo "1. Small data (should work fine):"
echo "hello world" | ./bin/nuro -p "count words" --verbose
echo

echo "2. Medium data (~60KB, should show warning):"
python3 -c "print('test data ' * 8000)" | ./bin/nuro -p "analyze this" --verbose
echo

echo "3. Large data (~600KB, should require --force):"
echo "This should fail without --force flag..."
python3 -c "print('large data ' * 60000)" | ./bin/nuro -p "analyze this" --verbose 2>&1 | head -5
echo

echo "4. Large data with --force (should proceed with warning):"
echo "This should work with --force flag..."
echo "test data for force flag" | ./bin/nuro -p "count words" --verbose --force
echo

echo "5. Real-world example - process list:"
ps -eo pid,comm,%cpu | head -10 | ./bin/nuro -p "show me the top processes" --verbose