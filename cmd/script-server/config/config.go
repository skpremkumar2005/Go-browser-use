package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Global configuration instance
var AppConfig *Config

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	Browser   BrowserConfig
	Script    ScriptConfig
	Streaming StreamingConfig
	CDP       CDPConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// BrowserConfig holds browser-related configuration
type BrowserConfig struct {
	Width                 int
	Height                int
	ViewportWidth         int
	ViewportHeight        int
	Display               string
	XvfbDisplay           string
	XvfbScreen            string
	XvfbResolution        string
	ChromePath            string
	MaxConcurrentSessions int
}

// ScriptConfig holds script execution configuration
type ScriptConfig struct {
	PythonCommand    string
	MaxSteps         int
	ScriptDir        string
	TaskScript       string
	ActionScript     string
	ScreenshotScript string
}

// StreamingConfig holds streaming-related configuration
type StreamingConfig struct {
	FrameRate         int
	CompletionTimeout time.Duration
	LogInterval       time.Duration
}

// CDPConfig holds Chrome DevTools Protocol configuration
type CDPConfig struct {
	Ports        []string
	Host         string
	Timeout      time.Duration
	PingInterval time.Duration
	PingTimeout  time.Duration
	CloseTimeout time.Duration
}

// LoadConfig initializes and loads all configuration from environment variables
func LoadConfig() *Config {
	// Load environment variables from .env file (try multiple paths)
	envPaths := []string{"../../.env", ".env", "../.env"}
	envLoaded := false

	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("✅ Loaded environment variables from %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Printf("⚠️ Warning: Could not load .env file from any of the paths: %v", envPaths)
	}

	config := &Config{
		Server: ServerConfig{
			Port:         GetEnvWithDefault("PORT", "8080"),
			Host:         GetEnvWithDefault("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  parseDurationWithDefault("SERVER_READ_TIMEOUT", "30s"),
			WriteTimeout: parseDurationWithDefault("SERVER_WRITE_TIMEOUT", "30s"),
		},
		Browser: BrowserConfig{
			Width:          parseIntWithDefault("BROWSER_WIDTH", 1920),
			Height:         parseIntWithDefault("BROWSER_HEIGHT", 1080),
			ViewportWidth:  parseIntWithDefault("BROWSER_WIDTH", 1920),
			ViewportHeight: parseIntWithDefault("BROWSER_HEIGHT", 1080),
			Display:        GetEnvWithDefault("DISPLAY", ":99"),
			XvfbDisplay:    GetEnvWithDefault("XVFB_DISPLAY", ":99"),
			XvfbScreen:     GetEnvWithDefault("XVFB_SCREEN", "0"),
			XvfbResolution: GetEnvWithDefault("XVFB_RESOLUTION", "1920x1080x24"),
			ChromePath:     GetEnvWithDefault("CHROME_PATH", "/usr/bin/google-chrome"),
			// ChromePath:            GetEnvWithDefault("CHROME_PATH", "C:/Program Files/Google/Chrome/Application/chrome.exe"),
			MaxConcurrentSessions: parseIntWithDefault("MAX_CONCURRENT_SESSIONS", 5),
		},
		Script: ScriptConfig{
			PythonCommand:    detectPythonPath(),
			MaxSteps:         parseIntWithDefault("MAX_STEPS", 50),
			ScriptDir:        GetEnvWithDefault("SCRIPT_DIR", "libs/utils/shared/scripts"),
			TaskScript:       GetEnvWithDefault("TASK_SCRIPT", "browser_task_fixed.py"),
			ActionScript:     GetEnvWithDefault("ACTION_SCRIPT", "browser_action_fixed.py"),
			ScreenshotScript: GetEnvWithDefault("SCREENSHOT_SCRIPT", "browser_screenshot_fixed.py"),
		},
		Streaming: StreamingConfig{
			FrameRate:         parseIntWithDefault("STREAMING_FRAME_RATE", 2),
			CompletionTimeout: parseDurationWithDefault("STREAMING_COMPLETION_TIMEOUT", "300s"),
			LogInterval:       parseDurationWithDefault("STREAMING_LOG_INTERVAL", "10s"),
		},
		CDP: CDPConfig{
			Ports:        parseCDPPorts(),
			Host:         GetEnvWithDefault("CDP_HOST", "127.0.0.1"),
			Timeout:      parseDurationWithDefault("CDP_TIMEOUT", "30s"),
			PingInterval: parseDurationWithDefault("CDP_PING_INTERVAL", "30s"),
			PingTimeout:  parseDurationWithDefault("CDP_PING_TIMEOUT", "10s"),
			CloseTimeout: parseDurationWithDefault("CDP_CLOSE_TIMEOUT", "5s"),
		},
	}

	AppConfig = config
	logConfiguration(config)
	return config
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	if AppConfig == nil {
		return LoadConfig()
	}
	return AppConfig
}

// GetAllowedOrigins returns the list of allowed CORS origins
func GetAllowedOrigins() []string {
	allowedOrigins := GetEnvWithDefault("ALLOWED_ORIGINS", "*")
	if allowedOrigins == "*" {
		return []string{"*"}
	}
	return strings.Split(allowedOrigins, ",")
}

// Helper functions for environment variable parsing

// detectPythonPath detects the Python executable path on the system
func detectPythonPath() string {
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

func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		log.Printf("⚠️ Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

func parseDurationWithDefault(key, defaultValue string) time.Duration {
	value := GetEnvWithDefault(key, defaultValue)
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

func parseCDPPorts() []string {
	portsEnv := GetEnvWithDefault("CDP_PORTS", "9222,9223,9224,9225,9226")
	return strings.Split(portsEnv, ",")
}

// logConfiguration logs the loaded configuration for debugging
func logConfiguration(config *Config) {
	log.Printf("🔧 Configuration loaded:")
	log.Printf("   Server: %s:%s (Read: %v, Write: %v)",
		config.Server.Host, config.Server.Port,
		config.Server.ReadTimeout, config.Server.WriteTimeout)
	log.Printf("   Browser: %dx%d, Display: %s, Chrome: %s",
		config.Browser.Width, config.Browser.Height,
		config.Browser.Display, config.Browser.ChromePath)
	log.Printf("   Scripts: %s, Dir: %s, Max Steps: %d",
		config.Script.PythonCommand, config.Script.ScriptDir, config.Script.MaxSteps)
	log.Printf("   Streaming: %d FPS, Timeout: %v",
		config.Streaming.FrameRate, config.Streaming.CompletionTimeout)
	log.Printf("   CDP: Host: %s, Ports: %v, Timeout: %v",
		config.CDP.Host, config.CDP.Ports, config.CDP.Timeout)
	log.Printf("   Max Concurrent Sessions: %d", config.Browser.MaxConcurrentSessions)
}

// Validation functions

func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	// Validate browser configuration
	if c.Browser.Width <= 0 || c.Browser.Height <= 0 {
		return fmt.Errorf("browser dimensions must be positive")
	}

	if c.Browser.MaxConcurrentSessions <= 0 {
		return fmt.Errorf("max concurrent sessions must be positive")
	}

	// Validate script configuration
	if c.Script.MaxSteps <= 0 {
		return fmt.Errorf("max steps must be positive")
	}

	// Validate streaming configuration
	if c.Streaming.FrameRate <= 0 {
		return fmt.Errorf("streaming frame rate must be positive")
	}

	// Validate CDP configuration
	if len(c.CDP.Ports) == 0 {
		return fmt.Errorf("at least one CDP port must be configured")
	}

	return nil
}

// Environment variable helpers for backward compatibility

func GetServerPort() string {
	return GetConfig().Server.Port
}

func GetServerHost() string {
	return GetConfig().Server.Host
}

func GetBrowserWidth() int {
	return GetConfig().Browser.Width
}

func GetBrowserHeight() int {
	return GetConfig().Browser.Height
}

func GetMaxConcurrentSessions() int {
	return GetConfig().Browser.MaxConcurrentSessions
}

func GetPythonCommand() string {
	return GetConfig().Script.PythonCommand
}

func GetMaxSteps() int {
	return GetConfig().Script.MaxSteps
}

func GetScriptDir() string {
	return GetConfig().Script.ScriptDir
}

func GetTaskScript() string {
	return GetConfig().Script.TaskScript
}

func GetActionScript() string {
	return GetConfig().Script.ActionScript
}

func GetScreenshotScript() string {
	return GetConfig().Script.ScreenshotScript
}

func GetStreamingFrameRate() int {
	return GetConfig().Streaming.FrameRate
}

func GetStreamingCompletionTimeout() time.Duration {
	return GetConfig().Streaming.CompletionTimeout
}

func GetCDPPorts() []string {
	return GetConfig().CDP.Ports
}

func GetCDPHost() string {
	return GetConfig().CDP.Host
}

func GetCDPTimeout() time.Duration {
	return GetConfig().CDP.Timeout
}
