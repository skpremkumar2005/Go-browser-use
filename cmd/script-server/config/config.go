package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values
type Config struct {
	Server    ServerConfig
	Browser   BrowserConfig
	Script    ScriptConfig
	Streaming StreamingConfig
	CDP       CDPConfig
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type BrowserConfig struct {
	ViewportWidth  int
	ViewportHeight int
	Display        string
	XvfbDisplay    string
	XvfbScreen     string
	XvfbResolution string
}

type ScriptConfig struct {
	PythonCommand    string
	MaxSteps         int
	ScriptDir        string
	TaskScript       string
	ActionScript     string
	ScreenshotScript string
}

type StreamingConfig struct {
	FrameRate         time.Duration
	CompletionTimeout time.Duration
	LogInterval       int
}

type CDPConfig struct {
	CommonPorts  []int
	Timeout      time.Duration
	PingInterval time.Duration
	PingTimeout  time.Duration
	CloseTimeout time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "localhost"),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
		},
		Browser: BrowserConfig{
			ViewportWidth:  getEnvInt("BROWSER_WIDTH", 1920),
			ViewportHeight: getEnvInt("BROWSER_HEIGHT", 1080),
			Display:        getEnv("DISPLAY", ":99"),
			XvfbDisplay:    getEnv("XVFB_DISPLAY", ":99"),
			XvfbScreen:     getEnv("XVFB_SCREEN", "0"),
			XvfbResolution: getEnv("XVFB_RESOLUTION", "1920x1080x24"),
		},
		Script: ScriptConfig{
			PythonCommand:    getEnv("PYTHON_COMMAND", "python3"),
			MaxSteps:         getEnvInt("MAX_STEPS", 10),
			ScriptDir:        getEnv("SCRIPT_DIR", "../../scripts"),
			TaskScript:       getEnv("TASK_SCRIPT", "browser_task_fixed.py"),
			ActionScript:     getEnv("ACTION_SCRIPT", "browser_action_fixed.py"),
			ScreenshotScript: getEnv("SCREENSHOT_SCRIPT", "browser_screenshot_fixed.py"),
		},
		Streaming: StreamingConfig{
			FrameRate:         time.Duration(getEnvInt("STREAMING_FRAME_RATE", 100)) * time.Millisecond,
			CompletionTimeout: time.Duration(getEnvInt("STREAMING_COMPLETION_TIMEOUT", 30)) * time.Second,
			LogInterval:       getEnvInt("STREAMING_LOG_INTERVAL", 100),
		},
		CDP: CDPConfig{
			CommonPorts:  []int{9222, 9223, 9224, 9225, 9226, 9227, 9228, 9229, 9230},
			Timeout:      time.Duration(getEnvInt("CDP_TIMEOUT", 10)) * time.Second,
			PingInterval: time.Duration(getEnvInt("CDP_PING_INTERVAL", 20)) * time.Second,
			PingTimeout:  time.Duration(getEnvInt("CDP_PING_TIMEOUT", 10)) * time.Second,
			CloseTimeout: time.Duration(getEnvInt("CDP_CLOSE_TIMEOUT", 10)) * time.Second,
		},
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as int with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}