#!/bin/bash

echo "📱 Termux-Native-Agent Setup"
echo "=============================="

# Update packages
echo "[*] Updating package lists..."
pkg update -y && pkg upgrade -y

# Install system dependencies
echo "[*] Installing system dependencies..."
pkg install -y python termux-api libffi openssl

# Check if Termux:API app is installed (simple check if command exists, though it requires the app to function)
if ! command -v termux-tts-speak &> /dev/null; then
    echo "[!] termux-api package not found. Please ensure you have installed the Termux:API app from F-Droid/Play Store and the package."
fi

# Install Python dependencies
echo "[*] Installing Python dependencies..."
pip install --upgrade pip
pip install openai requests

echo "[*] Creating default configuration..."
if [ ! -f config.json ]; then
    cat <<EOF > config.json
{
    "llm_settings": {
        "provider": "openai",
        "base_url": "https://api.openai.com/v1",
        "api_key": "sk-...",
        "model_name": "gpt-4o-mini"
    }
}
EOF
    echo "    - Created config.json"
else
    echo "    - config.json already exists, skipping."
fi

echo ""
echo "✅ Setup complete!"
echo "Please edit 'config.json' with your API key."
echo "Run './run_agent.sh text' to start."
