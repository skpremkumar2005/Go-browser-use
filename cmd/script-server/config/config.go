package config

var GlobalEnv = map[string]interface{}{
	"Server": map[string]interface{}{
		"Port":         "3000",
		"Host":         "0.0.0.0",
		"ReadTimeout":  "30s",
		"WriteTimeout": "30s",
	},
	"Browser": map[string]interface{}{
		"Width":                 1920,
		"Height":                1080,
		"ViewportWidth":         1920,
		"ViewportHeight":        1080,
		"Display":               ":99",
		"XvfbDisplay":           ":99",
		"XvfbScreen":            "0",
		"XvfbResolution":        "1920x1080x24",
		"ChromePath":            "C:/Program Files/Google/Chrome/Application/chrome.exe",
		// "ChromePath":            "/usr/bin/google-chrome",
		"MaxConcurrentSessions": 5,
	},
	"Script": map[string]interface{}{
		"PythonCommand":    "python",
		"MaxSteps":         50,
		"ScriptDir":        "libs/utils/shared/scripts",
		"TaskScript":       "browser_task_fixed.py",
		"ActionScript":     "browser_action_fixed.py",
		"ScreenshotScript": "browser_screenshot_fixed.py",
	},
	"Streaming": map[string]interface{}{
		"FrameRate":         2,
		"CompletionTimeout": "300s",
		"LogInterval":       "10s",
	},
	"CDP": map[string]interface{}{
		"Ports":        []string{"9222", "9223", "9224", "9225", "9226"},
		"Host":         "127.0.0.1",
		"Timeout":      "30s",
		"PingInterval": "30s",
		"PingTimeout":  "10s",
		"CloseTimeout": "5s",
	},
	"AllowedOrigins": "*",
}
