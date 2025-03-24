package utils

import (
	"fmt"
	"os"
	"strings"
)

func DeleteSubfolder(path string) error {
	return os.RemoveAll(path) // Recursively deletes folder and contents.
}

func GetEnvOrDefault(key, value string) string {
	os.Setenv(key, value)

	return value
}

// Validate required API keys
func ValidateAPIKeys() error {
	required := []string{
		"HELM_API_CREATE_API_KEY",
		"HELM_API_DELETE_API_KEY",
		"HELM_API_UPDATE_API_KEY",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return fmt.Errorf("master %s missing", strings.ToLower(strings.TrimPrefix(key, "HELM_API_")))
		}
	}
	return nil
}

// Helper function
func GetEnvOrValue(envKey, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to check if a substring is present in a string.
func Contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > len(substr) && Contains(str[1:], substr))
}
