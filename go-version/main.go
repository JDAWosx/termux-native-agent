package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"google.golang.org/genai"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// --- Tool Handlers ---

// ExecuteShell
type ShellInput struct {
	Command string `json:"command"`
}
type ShellOutput struct {
	Result string `json:"result"`
}

func shellHandler(ctx tool.Context, input ShellInput) (ShellOutput, error) {
	log.Printf("[Tool] Executing Shell: %s", input.Command)
	cmd := exec.Command("sh", "-c", input.Command)
	output, err := cmd.CombinedOutput()
	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\nError: %v", err)
	}
	return ShellOutput{Result: result}, nil
}

// MakeCall
type CallInput struct {
	PhoneNumber string `json:"phone_number"`
}
type CallOutput struct {
	Status string `json:"status"`
}

func callHandler(ctx tool.Context, input CallInput) (CallOutput, error) {
	log.Printf("[Tool] Calling: %s", input.PhoneNumber)
	cmd := exec.Command("termux-telephony-call", input.PhoneNumber)
	if err := cmd.Run(); err != nil {
		return CallOutput{Status: fmt.Sprintf("Failed to call: %v", err)}, nil
	}
	return CallOutput{Status: fmt.Sprintf("Call initiated to %s", input.PhoneNumber)}, nil
}

// SendSMS
type SMSInput struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}
type SMSOutput struct {
	Status string `json:"status"`
}

func smsHandler(ctx tool.Context, input SMSInput) (SMSOutput, error) {
	log.Printf("[Tool] Sending SMS to %s: %s", input.PhoneNumber, input.Message)
	cmd := exec.Command("termux-sms-send", "-n", input.PhoneNumber, input.Message)
	if err := cmd.Run(); err != nil {
		return SMSOutput{Status: fmt.Sprintf("Failed to send SMS: %v", err)}, nil
	}
	return SMSOutput{Status: fmt.Sprintf("SMS sent to %s", input.PhoneNumber)}, nil
}

func main() {
	ctx := context.Background()

	// Check for API Key
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is not set.")
	}

	// Initialize Gemini Model
	model, err := gemini.NewModel(ctx, "gemini-2.0-flash", &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Define Tools
	shellTool, _ := functiontool.New(functiontool.Config{
		Name:        "execute_shell",
		Description: "Execute a shell command on the Termux system.",
	}, shellHandler)

	callTool, _ := functiontool.New(functiontool.Config{
		Name:        "make_call",
		Description: "Initiate a GSM phone call.",
	}, callHandler)

	smsTool, _ := functiontool.New(functiontool.Config{
		Name:        "send_sms",
		Description: "Send an SMS text message.",
	}, smsHandler)

	// Create Agent
	a, err := llmagent.New(llmagent.Config{
		Name:        "termux_agent",
		Model:       model,
		Description: "An autonomous agent running on Android Termux.",
		Instruction: "You are a helpful AI assistant running natively on Android. You have access to device tools like Phone, SMS, and Shell. Use them when requested.",
		Tools: []tool.Tool{
			shellTool,
			callTool,
			smsTool,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Launch
	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(a),
	}

	l := full.NewLauncher()
	// Use console mode by default if no args provided? 
	// The launcher usually parses args.
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
