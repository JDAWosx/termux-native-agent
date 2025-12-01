import json
import subprocess
import sys
import argparse
import time
import os
from openai import OpenAI

# --- Configuration ---
def load_config():
    with open("config.json", "r") as f:
        return json.load(f)

CONFIG = load_config()
LLM_SETTINGS = CONFIG["llm_settings"]

client = OpenAI(
    api_key=LLM_SETTINGS["api_key"],
    base_url=LLM_SETTINGS["base_url"]
)

# --- Tools ---

def execute_shell(command):
    """Executes a shell command and returns the output."""
    print(f"Executing shell: {command}")
    try:
        result = subprocess.run(command, shell=True, capture_output=True, text=True, timeout=10)
        return f"STDOUT: {result.stdout}\nSTDERR: {result.stderr}"
    except Exception as e:
        return f"Error: {str(e)}"

def make_call(phone_number):
    """Initiates a phone call."""
    print(f"Calling: {phone_number}")
    subprocess.run(["termux-telephony-call", phone_number])
    return f"Call initiated to {phone_number}"

def send_sms(phone_number, message):
    """Sends an SMS message."""
    print(f"Sending SMS to {phone_number}: {message}")
    subprocess.run(["termux-sms-send", "-n", phone_number, message])
    return f"SMS sent to {phone_number}"

TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "execute_shell",
            "description": "Execute a shell command on the Termux system. Use this to manage files, check system status, or fetch data.",
            "parameters": {
                "type": "object",
                "properties": {
                    "command": {"type": "string", "description": "The shell command to execute"}
                },
                "required": ["command"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "make_call",
            "description": "Initiate a GSM phone call to a specific number.",
            "parameters": {
                "type": "object",
                "properties": {
                    "phone_number": {"type": "string", "description": "The phone number to call"}
                },
                "required": ["phone_number"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "send_sms",
            "description": "Send an SMS text message.",
            "parameters": {
                "type": "object",
                "properties": {
                    "phone_number": {"type": "string", "description": "The recipient's phone number"},
                    "message": {"type": "string", "description": "The message content"}
                },
                "required": ["phone_number", "message"]
            }
        }
    }
]

# --- I/O Interfaces ---

def text_input(prompt="You: "):
    return input(prompt)

def text_output(text):
    print(f"Agent: {text}")

def voice_input(prompt=None):
    if prompt:
        print(prompt)
    # Uses termux-speech-to-text
    try:
        result = subprocess.run(["termux-speech-to-text"], capture_output=True, text=True)
        text = result.stdout.strip()
        print(f"You (Voice): {text}")
        return text
    except FileNotFoundError:
        print("Error: termux-speech-to-text not found.")
        return ""

def voice_output(text):
    print(f"Agent (Voice): {text}")
    subprocess.run(["termux-tts-speak", text])

# --- Agent Loop ---

def run_conversation(mode):
    messages = [{"role": "system", "content": "You are a helpful AI assistant running natively on Android via Termux. You have access to device tools like phone calling, SMS, and the shell."}]
    
    get_input = voice_input if mode == "voice" else text_input
    send_output = voice_output if mode == "voice" else text_output

    print(f"--- TNA Started in {mode.upper()} mode ---")
    if mode == "voice":
        send_output("System online. I am listening.")

    while True:
        try:
            user_input = get_input()
            if not user_input:
                continue
            
            if user_input.lower() in ["exit", "quit"]:
                break

            messages.append({"role": "user", "content": user_input})

            # First API call
            response = client.chat.completions.create(
                model=LLM_SETTINGS["model_name"],
                messages=messages,
                tools=TOOLS,
                tool_choice="auto"
            )
            
            response_message = response.choices[0].message
            tool_calls = response_message.tool_calls

            if tool_calls:
                # Add the model's response (with tool calls) to history
                messages.append(response_message)
                
                for tool_call in tool_calls:
                    function_name = tool_call.function.name
                    function_args = json.loads(tool_call.function.arguments)
                    
                    function_response = None
                    
                    if function_name == "execute_shell":
                        function_response = execute_shell(function_args.get("command"))
                    elif function_name == "make_call":
                        function_response = make_call(function_args.get("phone_number"))
                    elif function_name == "send_sms":
                        function_response = send_sms(function_args.get("phone_number"), function_args.get("message"))
                    
                    # Add tool response to history
                    messages.append(
                        {
                            "tool_call_id": tool_call.id,
                            "role": "tool",
                            "name": function_name,
                            "content": function_response,
                        }
                    )
                
                # Second API call (get final answer)
                second_response = client.chat.completions.create(
                    model=LLM_SETTINGS["model_name"],
                    messages=messages
                )
                final_reply = second_response.choices[0].message.content
                messages.append({"role": "assistant", "content": final_reply})
                send_output(final_reply)
            
            else:
                messages.append(response_message)
                send_output(response_message.content)

        except KeyboardInterrupt:
            print("\nStopping...")
            break
        except Exception as e:
            print(f"\nError: {e}")
            break

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Termux Native Agent")
    parser.add_argument("--mode", choices=["text", "voice"], default="text", help="Interaction mode")
    args = parser.parse_args()

    run_conversation(args.mode)
