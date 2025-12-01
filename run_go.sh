#!/bin/bash

# Get the directory of the script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Check if binary exists
if [ ! -f "$DIR/go-version/termux-agent-go" ]; then
    echo "[*] Building Go Agent..."
    cd "$DIR/go-version"
    go build -o termux-agent-go main.go
    cd "$DIR"
fi

# Load API Key from config.json if available and not in env
if [ -z "$GOOGLE_API_KEY" ] && [ -f "$DIR/config.json" ]; then
    # Attempt to extract api_key (assuming it might be in there, though the python config structure is different)
    # The Python config has "api_key" under "llm_settings". 
    # Simplest way is to ask user to set it or just try to grep it.
    # Let's just warn the user.
    echo "Note: Please ensure GOOGLE_API_KEY is set in your environment for the Go agent."
fi

echo "🚀 Starting Termux-Native-Agent (Go Version)..."
echo "Usage: ./run_go.sh console"
"$DIR/go-version/termux-agent-go" "$@"
