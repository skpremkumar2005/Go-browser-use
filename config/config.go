package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration values
type Config struct {
	// Server configuration
	Server ServerConfig

	// Browser configuration
	Browser BrowserConfig

	// Script configuration
	Script ScriptConfig

	// Streaming configuration
	Streaming StreamingConfig

	// CDP configuration
	CDP CDPConfig
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
	ViewportWidth  int
	ViewportHeight int
	Display        string
	XvfbDisplay    string
	XvfbScreen     string
	XvfbResolution string
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
	FrameRate         time.Duration
	CompletionTimeout time.Duration
	LogInterval       int
}

// CDPConfig holds Chrome DevTools Protocol configuration
type CDPConfig struct {
	CommonPorts  []int
	Timeout      time.Duration
	PingInterval time.Duration
	PingTimeout  time.Duration
	CloseTimeout time.Duration
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "localhost"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Browser: BrowserConfig{
			ViewportWidth:  getIntEnv("BROWSER_VIEWPORT_WIDTH", 1920),
			ViewportHeight: getIntEnv("BROWSER_VIEWPORT_HEIGHT", 1080),
			Display:        getEnv("BROWSER_DISPLAY", ":0"),
			XvfbDisplay:    getEnv("BROWSER_XVFB_DISPLAY", ":99"),
			XvfbScreen:     getEnv("BROWSER_XVFB_SCREEN", "0"),
			XvfbResolution: getEnv("BROWSER_XVFB_RESOLUTION", "1920x1080x24"),
		},
		Script: ScriptConfig{
			PythonCommand:    getEnv("SCRIPT_PYTHON_COMMAND", "python3"),
			MaxSteps:         getIntEnv("SCRIPT_MAX_STEPS", 10),
			ScriptDir:        getEnv("SCRIPT_DIR", "./scripts"),
			TaskScript:       getEnv("SCRIPT_TASK_SCRIPT", "browser_task.py"),
			ActionScript:     getEnv("SCRIPT_ACTION_SCRIPT", "browser_action.py"),
			ScreenshotScript: getEnv("SCRIPT_SCREENSHOT_SCRIPT", "browser_screenshot.py"),
		},
		Streaming: StreamingConfig{
			FrameRate:         getDurationEnv("STREAMING_FRAME_RATE", 200*time.Millisecond),
			CompletionTimeout: getDurationEnv("STREAMING_COMPLETION_TIMEOUT", 30*time.Second),
			LogInterval:       getIntEnv("STREAMING_LOG_INTERVAL", 10),
		},
		CDP: CDPConfig{
			CommonPorts:  getIntSliceEnv("CDP_COMMON_PORTS", []int{9222, 9223, 9224, 9225, 9226, 43031, 58499, 35425, 42101, 47507, 37579, 54399}),
			Timeout:      getDurationEnv("CDP_TIMEOUT", 1*time.Second),
			PingInterval: getDurationEnv("CDP_PING_INTERVAL", 20*time.Second),
			PingTimeout:  getDurationEnv("CDP_PING_TIMEOUT", 10*time.Second),
			CloseTimeout: getDurationEnv("CDP_CLOSE_TIMEOUT", 10*time.Second),
		},
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getIntSliceEnv(key string, defaultValue []int) []int {
	if value := os.Getenv(key); value != "" {
		// Parse comma-separated integers
		parts := strings.Split(value, ",")
		result := make([]int, 0, len(parts))
		for _, part := range parts {
			if intValue, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				result = append(result, intValue)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
