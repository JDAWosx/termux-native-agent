#!/bin/bash

# Get the directory of the script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"

MODE=$1

if [ -z "$MODE" ]; then
    echo "Usage: ./run_agent.sh [text|voice]"
    exit 1
fi

# Activate environment if we had venv (keeping it simple for now as per 'Zero Bloat' claim)

echo "🚀 Starting Termux-Native-Agent in $MODE mode..."

python main.py --mode "$MODE"
