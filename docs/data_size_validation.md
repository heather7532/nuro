# nuro Data Size Validation

## Overview

nuro now includes data size validation to protect users from accidentally sending large amounts of data to LLMs, which could result in unexpected costs.

## Thresholds

- **Warning Threshold**: 50KB - Shows a warning but continues processing
- **Error Threshold**: 500KB - Blocks processing unless `--force` is used

## Flags

- `--force` / `-f`: Override data size limits and send large data anyway

## Behavior

### Small Data (< 50KB)
- No warnings or restrictions
- Processes normally

### Medium Data (50KB - 500KB)
- Shows a warning message about potential costs
- Continues processing
- Example: `nuro: WARNING: Data size 58.6KB (60001 bytes) is large and may increase LLM costs.`

### Large Data (> 500KB)
- Blocks processing with an error message
- Requires `--force` flag to proceed
- Provides suggestions for data reduction
- Example error:
  ```
  data size 585.9KB (600001 bytes) exceeds safe limit (500.0KB). This could be expensive to send to LLM.
  Use --force/-f to proceed anyway, or reduce data size.
  Consider filtering with: head, tail, grep, jq, or similar tools
  ```

## Examples

### Normal usage (small data)
```bash
echo "hello world" | nuro -p "count words" --verbose
# Output includes: nuro: data size=12B (12 bytes)
```

### Medium data with warning
```bash
# Creates ~60KB of data - triggers warning but continues
python3 -c "print('test ' * 12000)" | nuro -p "analyze this"
```

### Large data blocked
```bash
# Creates ~600KB of data - blocks unless --force is used
python3 -c "print('large ' * 100000)" | nuro -p "analyze this"
# Error: data size exceeds safe limit
```

### Force override
```bash
# Same large data but with --force flag
python3 -c "print('large ' * 100000)" | nuro -p "analyze this" --force
# Proceeds with warning: nuro: WARNING: Large data size 585.9KB forced with --force flag. This may be expensive.
```

### Real-world example
```bash
# Process list - usually under limits
ps -eo pid,comm,%cpu | head -20 | nuro -p "show me the top CPU users" --verbose
# Shows: nuro: data size=580B (580 bytes) - no warnings
```

## Data Sources

The validation applies to data from any source:
- **stdin**: `cat large_file.txt | nuro -p "summarize"`
- **--data flag**: `nuro -p "analyze" --data "$(cat large_file.txt)"`
- **--data-file**: `nuro -p "analyze" --data-file large_file.txt`

## Verbose Mode

When using `--verbose`, you'll always see the data size:
- `nuro: data size=12B (12 bytes)` for small data
- `nuro: data size=58.6KB (60001 bytes)` for larger data

This helps you understand how much data you're sending to the LLM.

## Filtering Suggestions

For large datasets, consider filtering before sending to nuro:

```bash
# Instead of sending entire process list
ps -ef | nuro -p "analyze processes"  # May be very large

# Filter first
ps -eo pid,comm,%cpu --sort=-%cpu | head -20 | nuro -p "show top CPU users"

# For JSON data
cat large.json | jq '.important_field' | nuro -p "analyze this data"

# For log files  
tail -100 /var/log/system.log | nuro -p "find errors"
```

This protects against expensive accidental sends while still allowing users to override when needed.