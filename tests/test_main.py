import sys
import os
import pytest
from unittest.mock import MagicMock, patch

# Add parent directory to path to import main
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import main

# --- Tests for Tools ---

def test_execute_shell(mocker):
    mock_run = mocker.patch("subprocess.run")
    mock_run.return_value.stdout = "test output"
    mock_run.return_value.stderr = ""
    
    result = main.execute_shell("echo test")
    
    assert "STDOUT: test output" in result
    mock_run.assert_called_with("echo test", shell=True, capture_output=True, text=True, timeout=10)

def test_make_call(mocker):
    mock_run = mocker.patch("subprocess.run")
    
    result = main.make_call("555-0123")
    
    assert "Call initiated" in result
    mock_run.assert_called_with(["termux-telephony-call", "555-0123"])

def test_send_sms(mocker):
    mock_run = mocker.patch("subprocess.run")
    
    result = main.send_sms("555-0123", "Hello")
    
    assert "SMS sent" in result
    mock_run.assert_called_with(["termux-sms-send", "-n", "555-0123", "Hello"])

# --- Tests for Agent Logic (Mocking OpenAI) ---

@patch("main.client")
@patch("main.text_input")
@patch("main.text_output")
def test_run_conversation_text_simple(mock_output, mock_input, mock_client):
    # Scenario: User says "hi", Agent says "hello"
    
    # Mock User Input: "hi", then "exit"
    mock_input.side_effect = ["hi", "exit"]
    
    # Mock OpenAI Response
    mock_response = MagicMock()
    mock_response.choices[0].message.content = "Hello there!"
    mock_response.choices[0].message.tool_calls = None
    
    mock_client.chat.completions.create.return_value = mock_response
    
    main.run_conversation("text")
    
    # Verify "Hello there!" was output
    mock_output.assert_any_call("Hello there!")

@patch("main.client")
@patch("main.text_input")
@patch("main.text_output")
@patch("main.make_call")
def test_run_conversation_tool_use(mock_make_call, mock_output, mock_input, mock_client):
    # Scenario: User says "Call 123", Agent calls function, then says "Done"
    
    mock_input.side_effect = ["Call 123", "exit"]
    
    # 1. First API response: Tool Call
    # Create the function mock specifically to ensure .name attribute is set correctly
    function_mock = MagicMock()
    function_mock.name = "make_call"
    function_mock.arguments = '{"phone_number": "123"}'

    tool_call_mock = MagicMock()
    tool_call_mock.id = "call_1"
    tool_call_mock.function = function_mock

    msg1 = MagicMock()
    msg1.content = None
    msg1.tool_calls = [tool_call_mock]
    
    # 2. Second API response: Final Reply
    msg2 = MagicMock()
    msg2.choices[0].message.content = "I have called 123."
    
    # Mock the API to return msg1 then msg2
    # Note: create() is called twice in the loop for a tool call workflow
    mock_client.chat.completions.create.side_effect = [
        MagicMock(choices=[MagicMock(message=msg1)]), # First call returns tool request
        MagicMock(choices=[MagicMock(message=msg2.choices[0].message)]) # Second call returns final answer
    ]
    
    mock_make_call.return_value = "Call initiated"
    
    main.run_conversation("text")
    
    # Verify tool was executed
    mock_make_call.assert_called_with("123")
    
    # Verify final output
    mock_output.assert_any_call("I have called 123.")
