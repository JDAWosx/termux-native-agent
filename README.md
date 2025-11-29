# Termux-Native-Agent (TNA)

Termux-Native-Agent (TNA) is a lightweight, modular AI agent framework designed to run natively on Android devices.
Unlike other agents that require heavy Docker containers or root access, TNA utilizes the native Android subsystem via Termux:API to bridge the gap between Large Language Models (LLMs) and physical hardware. It can make phone calls, send text messages, and manage the device shell, all while maintaining a continuous voice conversation with the user.

## Key Features
* Model Agnostic: Built on the OpenAI API standard. Hot-swap between GPT-4o, Claude, or run completely offline with Ollama (Llama 3, Mistral) running locally on your device.
* Native Telephony Tools: The agent can autonomously initiate GSM phone calls and send SMS messages using your device's SIM card.
* Live Voice Loop: Features a continuous ReAct loop with low-latency Speech-to-Text (STT) and Text-to-Speech (TTS) for hands-free interaction.
* Zero Bloat: No Docker. No VM. Runs purely on Python and standard Termux packages (pkg).
* Shell Autonomy: Capable of executing shell commands to manage files, fetch web data, or diagnose system status.

## Architecture
TNA operates on a standard ReAct (Reasoning + Acting) loop:
* Input: Captures audio via Android's native Speech Recognition service (efficient battery usage).
* Reasoning: Sends context + available tools to the LLM (Cloud or Local).
* Tool Execution: If the LLM decides to act (e.g., "Call Mom"), TNA intercepts the function call and executes it via subprocess wrappers around termux-api.
* Feedback: The tool output (e.g., "Call started") is fed back into the conversation history.
* Output: The agent responds via Android's native Text-to-Speech engine.

## Security & Disclaimer
* Cost: Using cloud APIs (OpenAI) incurs costs. Check your usage limits.
* Safety: The run_shell tool gives the LLM access to your Termux shell. While it cannot access root (unless your device is rooted), it can modify files within the Termux scope. Use with caution.
* Privacy: If using local models (Ollama), no data leaves your device. If using OpenAI, data is processed according to their privacy policy.

## License
Distributed under the MIT License. See LICENSE for more information.
