package stealth

import (
	"time"
) // StealthIntegration provides easy integration with existing browser management
type StealthIntegration struct {
	manager *StealthManager
}

// Global stealth integration instance
var Integration *StealthIntegration

func init() {
	Integration = &StealthIntegration{
		manager: NewStealthManager(),
	}
}

// GetEnhancedStealthJS returns the ultimate stealth JavaScript with all countermeasures
func GetEnhancedStealthJS() string {
	// Use the ultimate stealth system for maximum protection
	return GetUltimateStealthJS()
} // GetStealthUserAgent returns a randomized user agent
func GetStealthUserAgent() string {
	return Integration.manager.config.GetRandomUserAgent()
}

// GetStealthViewport returns a randomized viewport
func GetStealthViewport() Viewport {
	return Integration.manager.config.GetRandomViewport()
}

// GetStealthBrowserArguments returns ultimate stealth browser arguments with Unified Platform architecture
func GetStealthBrowserArguments(sessionID string, port int, width, height int) []string {
	viewport := Viewport{Width: width, Height: height}

	// Get enhanced unified platform arguments (combines all previous techniques)
	unifiedArgs := GetUnifiedPlatformBrowserArgs()

	// Get milspec arguments for session-specific settings
	milspecArgs := GetMilspecStealthArgs(sessionID, port, viewport, nil)

	// Add ultimate stealth arguments for network-level detection bypass
	ultimateArgs := GetUltimateStealthArguments()

	// Add Google bypass arguments
	googleArgs := GetGoogleBypassArguments()

	// Combine all argument sets with unified platform as base
	combined := make([]string, 0, len(unifiedArgs)+len(milspecArgs)+len(ultimateArgs)+len(googleArgs))
	combined = append(combined, unifiedArgs...)
	combined = append(combined, milspecArgs...)
	combined = append(combined, ultimateArgs...)
	combined = append(combined, googleArgs...)

	return combined
}

// GetHumanActionDelay returns a human-like delay between actions
func GetHumanActionDelay() time.Duration {
	return Integration.manager.GetHumanDelay()
}

// GetHumanPageLoadDelay returns a human-like page load delay
func GetHumanPageLoadDelay() time.Duration {
	return Integration.manager.GetPageLoadDelay()
}

// LogStealthMode logs stealth mode activation
func LogStealthMode(sessionID string) {
	Integration.manager.LogStealthActivation(sessionID)
}

// GetUltraStealthHeaders returns realistic HTTP headers for advanced stealth
func GetUltraStealthHeaders() map[string]string {
	return GetAdvancedStealthHeaders()
}

// GetStealthNetworkEmulation returns realistic network emulation settings
func GetStealthNetworkEmulation() map[string]interface{} {
	return GetStealthNetworkConditions()
}
