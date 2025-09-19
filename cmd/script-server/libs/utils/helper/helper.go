package helper

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Environment variable helper functions

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func GetEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// URL generation helper functions

func GenerateBaseURL(host, port string, isHTTPS bool) string {
	protocol := "http"
	if isHTTPS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%s", protocol, host, port)
}

func GenerateAutomationURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/browser_use/live-automation/%s", baseURL, sessionId)
}

func GenerateStreamingURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/browser_use/stream-screencast/%s", baseURL, sessionId)
}

func GenerateWebSocketURL(baseURL string) string {
	if strings.HasPrefix(baseURL, "https://") {
		return strings.Replace(baseURL, "https://", "wss://", 1) + "/ws"
	}
	return strings.Replace(baseURL, "http://", "ws://", 1) + "/ws"
}

// String utility functions

func SanitizeString(input string) string {
	// Remove potentially dangerous characters
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\r", " ")
	input = strings.ReplaceAll(input, "\t", " ")

	// Trim whitespace
	return strings.TrimSpace(input)
}

func TruncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}
	return input[:maxLength-3] + "..."
}

// Array utility functions

func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// File and path utility functions

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// Validation functions

func IsValidPort(port string) bool {
	if portNum, err := strconv.Atoi(port); err == nil {
		return portNum > 0 && portNum <= 65535
	}
	return false
}

func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// Error handling utilities

func LogError(context string, err error) {
	if err != nil {
		fmt.Printf("❌ Error in %s: %v\n", context, err)
	}
}

func LogInfo(message string, args ...interface{}) {
	fmt.Printf("ℹ️ "+message+"\n", args...)
}

func LogWarning(message string, args ...interface{}) {
	fmt.Printf("⚠️ "+message+"\n", args...)
}

func LogSuccess(message string, args ...interface{}) {
	fmt.Printf("✅ "+message+"\n", args...)
}

// Command execution helper functions

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func ExecuteCommandWithTimeout(command string, args []string, timeout time.Duration) (string, error) {
	cmd := exec.Command(command, args...)
	var out strings.Builder
	cmd.Stdout = &out

	err := cmd.Start()
	if err != nil {
		return "", err
	}

	timer := time.AfterFunc(timeout, func() {
		cmd.Process.Kill()
	})
	defer timer.Stop()

	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// Retry mechanism for command execution

func RetryCommand(command string, args []string, retries int, delay time.Duration) (string, error) {
	var err error
	var result string
	for i := 0; i < retries; i++ {
		result, err = ExecuteCommand(command, args...)
		if err == nil {
			return result, nil
		}
		time.Sleep(delay)
	}
	return "", fmt.Errorf("command failed after %d retries: %v", retries, err)
}

// detectPythonPath detects the Python executable path on the system
func DetectPythonPath() string {
	// Check if PYTHON_COMMAND is set in environment
	if pythonCmd := os.Getenv("PYTHON_COMMAND"); pythonCmd != "" {
		log.Printf("🐍 Using Python command from environment: %s", pythonCmd)
		return pythonCmd
	}

	// Try common Python executable names on Windows and other systems
	possiblePaths := []string{
		"python",      // Most common on Windows
		"python3",     // Common on Linux/Mac
		"py",          // Python Launcher on Windows
		"python.exe",  // Explicit Windows executable
		"python3.exe", // Explicit Windows Python 3
	}

	for _, path := range possiblePaths {
		if _, err := exec.LookPath(path); err == nil {
			log.Printf("🔍 Found Python at: %s", path)
			return path
		}
	}

	// Fallback to python3 (original default)
	log.Printf("⚠️ Python not found in PATH, using fallback: python3")
	log.Printf("💡 Consider installing Python or setting PYTHON_COMMAND environment variable")
	return "python3"
}

// GetEnvWithDefault is an alias for GetEnv for backward compatibility
func GetEnvWithDefault(key, defaultValue string) string {
	return GetEnv(key, defaultValue)
}

// parseIntWithDefault parses an integer from environment variable with default
func ParseIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		log.Printf("⚠️ Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// parseDurationWithDefault parses a duration from environment variable with default
func ParseDurationWithDefault(key, defaultValue string) time.Duration {
	value := GetEnv(key, defaultValue)
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	// Fallback: try parsing as seconds if it's just a number
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	log.Printf("⚠️ Invalid duration value for %s: %s, using default: %s", key, value, defaultValue)
	if defaultDuration, err := time.ParseDuration(defaultValue); err == nil {
		return defaultDuration
	}

	return 30 * time.Second // Ultimate fallback
}

// parseCDPPorts parses CDP ports from environment variable
func ParseCDPPorts() []string {
	portsEnv := GetEnv("CDP_PORTS", "9222,9223,9224,9225,9226")
	return strings.Split(portsEnv, ",")
}

// logConfiguration logs the loaded configuration for debugging
func LogConfiguration(config map[string]interface{}) {
	log.Printf("🔧 Configuration loaded:")
	if server, ok := config["Server"].(map[string]interface{}); ok {
		log.Printf("   Server: %s:%s (Read: %v, Write: %v)",
			server["Host"], server["Port"], server["ReadTimeout"], server["WriteTimeout"])
	}
	if browser, ok := config["Browser"].(map[string]interface{}); ok {
		log.Printf("   Browser: %dx%d, Display: %s, Chrome: %s",
			int(getNumericValue(browser["Width"])), int(getNumericValue(browser["Height"])), browser["Display"], browser["ChromePath"])
	}
	if script, ok := config["Script"].(map[string]interface{}); ok {
		log.Printf("   Scripts: %s, Dir: %s, Max Steps: %d",
			script["PythonCommand"], script["ScriptDir"], int(getNumericValue(script["MaxSteps"])))
	}
	if streaming, ok := config["Streaming"].(map[string]interface{}); ok {
		log.Printf("   Streaming: %d FPS, Timeout: %v",
			int(getNumericValue(streaming["FrameRate"])), streaming["CompletionTimeout"])
	}
	if cdp, ok := config["CDP"].(map[string]interface{}); ok {
		log.Printf("   CDP: Host: %s, Ports: %v, Timeout: %v",
			cdp["Host"], cdp["Ports"], cdp["Timeout"])
	}
	if maxSess := getNumericValue(config["MaxConcurrentSessions"]); maxSess > 0 {
		log.Printf("   Max Concurrent Sessions: %d", int(maxSess))
	}
}

// GetAllowedOrigins returns the list of allowed CORS origins
func GetAllowedOrigins() []string {
	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "*")
	if allowedOrigins == "*" {
		return []string{"*"}
	}
	return strings.Split(allowedOrigins, ",")
}

// ValidateConfig validates the configuration map
func ValidateConfig(config map[string]interface{}) error {
	// Validate server configuration
	if server, ok := config["Server"].(map[string]interface{}); ok {
		if port, ok := server["Port"].(string); ok && port == "" {
			return fmt.Errorf("server port cannot be empty")
		}
	}

	// Validate browser configuration
	if browser, ok := config["Browser"].(map[string]interface{}); ok {
		width := getNumericValue(browser["Width"])
		if width <= 0 {
			return fmt.Errorf("browser width must be positive")
		}
		height := getNumericValue(browser["Height"])
		if height <= 0 {
			return fmt.Errorf("browser height must be positive")
		}
		maxSess := getNumericValue(browser["MaxConcurrentSessions"])
		if maxSess <= 0 {
			return fmt.Errorf("max concurrent sessions must be positive")
		}
	}

	// Validate script configuration
	if script, ok := config["Script"].(map[string]interface{}); ok {
		maxSteps := getNumericValue(script["MaxSteps"])
		if maxSteps <= 0 {
			return fmt.Errorf("max steps must be positive")
		}
	}

	// Validate streaming configuration
	if streaming, ok := config["Streaming"].(map[string]interface{}); ok {
		frameRate := getNumericValue(streaming["FrameRate"])
		if frameRate <= 0 {
			return fmt.Errorf("streaming frame rate must be positive")
		}
	}

	// Validate CDP configuration
	if cdp, ok := config["CDP"].(map[string]interface{}); ok {
		if ports, ok := cdp["Ports"].([]interface{}); ok && len(ports) == 0 {
			return fmt.Errorf("at least one CDP port must be configured")
		}
	}

	return nil
}

// getNumericValue extracts a numeric value from interface{} that could be int or float64
func getNumericValue(val interface{}) float64 {
	switch v := val.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}
