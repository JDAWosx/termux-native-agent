package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"google.golang.org/genai"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// --- Helper: Persistent Memory ---
const MemoryFile = "agent_memory.json"

func loadMemory() map[string]string {
	data, err := os.ReadFile(MemoryFile)
	if err != nil {
		return make(map[string]string)
	}
	var mem map[string]string
	json.Unmarshal(data, &mem)
	if mem == nil {
		return make(map[string]string)
	}
	return mem
}

func saveMemory(mem map[string]string) {
	data, _ := json.MarshalIndent(mem, "", "  ")
	os.WriteFile(MemoryFile, data, 0644)
}

// --- Tool Handlers ---

// 1. ExecuteShell
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

// 2. MakeCall
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

// 3. SendSMS
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

// 4. Launch App (Deep Linking)
type AppInput struct {
	AppName string `json:"app_name"`
	Query   string `json:"query"`
}
type AppOutput struct {
	Status string `json:"status"`
}

func launchAppHandler(ctx tool.Context, input AppInput) (AppOutput, error) {
	log.Printf("[Tool] Launching App: %s (Query: %s)", input.AppName, input.Query)
	
	var cmd *exec.Cmd
	
	switch input.AppName {
	case "spotify":
		cmd = exec.Command("termux-open-url", "spotify:search:"+input.Query)
	case "maps":
		cmd = exec.Command("termux-open-url", "geo:0,0?q="+input.Query)
	case "youtube":
		cmd = exec.Command("termux-open-url", "https://www.youtube.com/results?search_query="+input.Query)
	case "browser":
		cmd = exec.Command("termux-open-url", input.Query) // Treats query as URL
	case "camera":
		cmd = exec.Command("termux-camera-photo", "/dev/null") // Just launches it? No, this captures.
		// To launch camera app:
		cmd = exec.Command("am", "start", "-a", "android.media.action.IMAGE_CAPTURE")
	default:
		// Try generic launch
		return AppOutput{Status: "App not supported for deep linking yet."}, nil
	}

	if err := cmd.Run(); err != nil {
		return AppOutput{Status: fmt.Sprintf("Failed to launch %s: %v", input.AppName, err)}, nil
	}
	return AppOutput{Status: fmt.Sprintf("Launched %s", input.AppName)}, nil
}

// 5. Inspect Surroundings (Vision)
type VisionInput struct {
	Prompt string `json:"prompt"`
}
type VisionOutput struct {
	Description string `json:"description"`
}

// Note: This requires a secondary client to perform the vision task inside the tool.
// We'll re-use the env API key.
func visionHandler(ctx tool.Context, input VisionInput) (VisionOutput, error) {
	log.Printf("[Tool] Inspecting Surroundings...")
	
	// 1. Capture Image
	tmpFile := fmt.Sprintf("capture_%d.jpg", time.Now().Unix())
	cmd := exec.Command("termux-camera-photo", "-c", "0", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return VisionOutput{Description: fmt.Sprintf("Failed to capture photo: %v. Output: %s", err, string(output))}, nil
	}
	defer os.Remove(tmpFile)

	// 2. Read Image
	imgData, err := os.ReadFile(tmpFile)
	if err != nil {
		return VisionOutput{Description: "Failed to read captured image file."}, nil
	}

	// 3. Analyze with Gemini (Nested Call)
	apiKey := os.Getenv("GOOGLE_API_KEY")
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return VisionOutput{Description: "Failed to connect to AI for vision analysis."}, nil
	}

	// Determine prompt
	userPrompt := input.Prompt
	if userPrompt == "" {
		userPrompt = "Describe what you see in this image in detail."
	}

	// Construct Content with Parts (Text + Image)
	parts := []*genai.Part{
		{Text: userPrompt},
		{InlineData: &genai.Blob{MIMEType: "image/jpeg", Data: imgData}},
	}
	contents := []*genai.Content{
		{Parts: parts},
	}

	resp, err := client.Models.GenerateContent(context.Background(), "gemini-2.0-flash", contents, nil)
	if err != nil {
		return VisionOutput{Description: fmt.Sprintf("AI Vision Analysis Failed: %v", err)}, nil
	}

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				return VisionOutput{Description: part.Text}, nil
			}
		}
	}

	return VisionOutput{Description: "Image captured, but no description generated."}, nil
}

// 6. Memory Tools
type SaveMemInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type SaveMemOutput struct {
	Status string `json:"status"`
}
func saveMemHandler(ctx tool.Context, input SaveMemInput) (SaveMemOutput, error) {
	mem := loadMemory()
	mem[input.Key] = input.Value
	saveMemory(mem)
	return SaveMemOutput{Status: "Saved to memory."}, nil
}

type ReadMemInput struct {
	Key string `json:"key"`
}
type ReadMemOutput struct {
	Value string `json:"value"`
}
func readMemHandler(ctx tool.Context, input ReadMemInput) (ReadMemOutput, error) {
	mem := loadMemory()
	val, ok := mem[input.Key]
	if !ok {
		return ReadMemOutput{Value: "Not found."}, nil
	}
	return ReadMemOutput{Value: val}, nil
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

	launchTool, _ := functiontool.New(functiontool.Config{
		Name: "launch_app",
		Description: "Launch an Android app or deep link. Apps: spotify, maps, youtube, browser, camera.",
	}, launchAppHandler)

	visionTool, _ := functiontool.New(functiontool.Config{
		Name: "inspect_surroundings",
		Description: "Take a photo with the camera and analyze it using AI vision. Use this to 'see' things.",
	}, visionHandler)

	saveMemTool, _ := functiontool.New(functiontool.Config{
		Name: "remember_fact",
		Description: "Save a fact to long-term memory.",
	}, saveMemHandler)

	readMemTool, _ := functiontool.New(functiontool.Config{
		Name: "recall_fact",
		Description: "Recall a fact from long-term memory.",
	}, readMemHandler)


	// Create Agent
	a, err := llmagent.New(llmagent.Config{
		Name:        "termux_agent",
		Model:       model,
		Description: "An autonomous agent running on Android Termux.",
		Instruction: `You are a helpful AI assistant running natively on Android. 
		You have access to the following tools:
		1. Phone & SMS: Communicate with the world.
		2. Shell: Control the device system.
		3. Launch App: Open Spotify, Maps, etc.
		4. Vision: "inspect_surroundings" to take photos and see what is in front of you.
		5. Memory: "remember_fact" and "recall_fact" to persist information.

		Use these tools proactively to help the user.`,
		Tools: []tool.Tool{
			shellTool,
			callTool,
			smsTool,
			launchTool,
			visionTool,
			saveMemTool,
			readMemTool,
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
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}