#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_FILE="$SCRIPT_DIR/cpu_hogger.csv"

start_ts=$(date +%s.%N)

./bin/serverledge-cli invoke -f cpu_hogger -p 'duration:180' -p 'memory:800'

end_ts=$(date +%s.%N)

{
    echo "timestamp"
    echo "$start_ts"
    echo "$end_ts"
} > "$OUTPUT_FILE"

echo "Done"
