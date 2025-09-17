package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// JsonMarshaller marshals data from one struct to another via JSON
func JsonMarshaller(source interface{}, target interface{}) error {
	jsonData, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}
	
	err = json.Unmarshal(jsonData, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}
	
	return nil
}

// GetEnv gets environment variable with default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets environment variable as integer with default value
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvDuration gets environment variable as duration with default value
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GenerateID generates a unique identifier
func GenerateID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// Contains checks if a slice contains a specific item
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsInt checks if an int slice contains a specific item
func ContainsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}



