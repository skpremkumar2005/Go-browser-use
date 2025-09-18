package stealth

import (
	"log"
	"strconv"
	"time"
)

// StealthManager manages all stealth features for browser sessions
type StealthManager struct {
	config        *StealthConfig
	humanBehavior *HumanBehavior
	logger        *log.Logger
}

// SessionProfile contains stealth settings for a specific session
type SessionProfile struct {
	UserAgent   string
	Viewport    Viewport
	BrowserArgs []string
	StealthJS   string
}

// NewStealthManager creates a new stealth manager instance
func NewStealthManager() *StealthManager {
	config := &DefaultStealthConfig

	return &StealthManager{
		config:        config,
		humanBehavior: NewHumanBehavior(config),
		logger:        log.New(log.Writer(), "[StealthManager] ", log.LstdFlags),
	}
}

// GenerateSessionProfile creates a unique stealth profile for a browser session
func (sm *StealthManager) GenerateSessionProfile(sessionID string) *SessionProfile {
	userAgent := sm.config.GetRandomUserAgent()
	viewport := sm.config.GetRandomViewport()

	sm.logger.Printf("🕵️ Generated stealth profile for session %s: UA=%s, Viewport=%dx%d",
		sessionID, userAgent[:50]+"...", viewport.Width, viewport.Height)

	return &SessionProfile{
		UserAgent:   userAgent,
		Viewport:    viewport,
		BrowserArgs: GetAdvancedBrowserArgs(userAgent, viewport),
		StealthJS:   GetEnhancedStealthJavaScript(),
	}
}

// GetHumanDelay returns a human-like delay for actions
func (sm *StealthManager) GetHumanDelay() time.Duration {
	return sm.humanBehavior.GenerateHumanDelay()
}

// GetPageLoadDelay returns a realistic page load wait time
func (sm *StealthManager) GetPageLoadDelay() time.Duration {
	return sm.humanBehavior.GeneratePageLoadDelay()
}

// GetElementWaitDelay returns a realistic element wait time
func (sm *StealthManager) GetElementWaitDelay() time.Duration {
	return sm.humanBehavior.GenerateElementWaitDelay()
}

// GetTypingDelays returns human-like typing delays for text
func (sm *StealthManager) GetTypingDelays(text string) []time.Duration {
	return sm.humanBehavior.GenerateTypingDelays(text)
}

// GenerateMousePath creates natural mouse movement path
func (sm *StealthManager) GenerateMousePath(startX, startY, endX, endY float64) []Point {
	start := Point{X: startX, Y: startY}
	end := Point{X: endX, Y: endY}
	return sm.humanBehavior.GenerateBezierPath(start, end)
}

// ShouldAddMousePause determines if mouse movement should include pauses
func (sm *StealthManager) ShouldAddMousePause() bool {
	return sm.humanBehavior.ShouldAddMousePause()
}

// GenerateScrollPattern creates human-like scrolling behavior
func (sm *StealthManager) GenerateScrollPattern(totalScroll float64) []ScrollAction {
	return sm.humanBehavior.GenerateScrollPattern(totalScroll)
}

// EnhanceUserAgent adds randomization to user agent string
func (sm *StealthManager) EnhanceUserAgent(baseUA string) string {
	// Add slight variations to make each session unique
	versions := []string{"120.0.0.0", "119.0.0.0", "118.0.0.0", "121.0.0.0"}

	for _, version := range versions {
		if !contains(baseUA, version) {
			continue
		}

		// Replace with a nearby version
		newVersion := generateNearbyVersion(version)
		return replace(baseUA, version, newVersion)
	}

	return baseUA
}

// GetStealthBrowserArgs returns browser arguments optimized for stealth
func (sm *StealthManager) GetStealthBrowserArgs(sessionID string, port int, viewport Viewport) []string {
	profile := sm.GenerateSessionProfile(sessionID)

	args := profile.BrowserArgs

	// Add dynamic port
	args = append(args, "--remote-debugging-port="+strconv.Itoa(port))

	sm.logger.Printf("🚀 Generated %d stealth browser arguments for session %s",
		len(args), sessionID)

	return args
}

// GetEnhancedStealthScript returns the advanced stealth JavaScript
func (sm *StealthManager) GetEnhancedStealthScript() string {
	return GetEnhancedStealthJavaScript()
}

// LogStealthActivation logs stealth mode activation
func (sm *StealthManager) LogStealthActivation(sessionID string) {
	sm.logger.Printf("🥷 Enhanced stealth mode activated for session: %s", sessionID)
	sm.logger.Printf("✅ Features enabled: 17-phase JS injection, human behavior simulation, advanced fingerprint masking")
}

// Helper functions

func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[len(str)-len(substr):] == substr ||
		len(str) > len(substr) && findSubstring(str, substr) != -1
}

func findSubstring(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func replace(str, old, new string) string {
	idx := findSubstring(str, old)
	if idx == -1 {
		return str
	}
	return str[:idx] + new + str[idx+len(old):]
}

func generateNearbyVersion(version string) string {
	// Simple version variation - in production, use more sophisticated logic
	versions := map[string]string{
		"120.0.0.0": "120.0.6099.109",
		"119.0.0.0": "119.0.6045.105",
		"118.0.0.0": "118.0.5993.70",
		"121.0.0.0": "121.0.6167.85",
	}

	if newVersion, exists := versions[version]; exists {
		return newVersion
	}

	return version
}
