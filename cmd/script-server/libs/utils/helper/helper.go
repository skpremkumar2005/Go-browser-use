package helper

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	return fmt.Sprintf("%s/api/live-automation/%s", baseURL, sessionId)
}

func GenerateStreamingURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/stream-screencast/%s", baseURL, sessionId)
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
