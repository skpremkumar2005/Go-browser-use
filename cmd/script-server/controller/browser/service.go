package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-webrtc/cmd/internal/stealth"
	"go-webrtc/cmd/script-server/libs/utils/helper"

	"github.com/gorilla/websocket"
)

var (
	cfg                  map[string]interface{}
	scriptSessionManager *ScriptSessionManager
	browserManager       *BrowserManager
	wsManager            *WebSocketManager
	upgrader             websocket.Upgrader

)

// InitializeGlobalVariables initializes all global variables with configuration
func InitializeGlobalVariables(configuration map[string]interface{}) {
	cfg = configuration
	
	scriptSessionManager = &ScriptSessionManager{
		sessions: make(map[string]*ScriptSession),
	}
	
	browserManager = &BrowserManager{
		sessions:              make(map[string]*BrowserSession),
		nextPort:              9222,
		maxConcurrentSessions: configuration["Browser"].(map[string]interface{})["MaxConcurrentSessions"].(int),
		activeSessions:        0,
		cdpClientCache:        make(map[string]*CDPClient),
		cdpCacheMutex:         sync.Mutex{},
	}
	
	wsManager = &WebSocketManager{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan []byte),
	}
	
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			allowedOrigins := helper.GetAllowedOrigins()
			origin := r.Header.Get("Origin")
			host := r.Host

			if origin == "" {
				return true
			}

			for _, allowed := range allowedOrigins {
				if origin == strings.TrimSpace(allowed) || allowed == "*" {
					return true
				}
			}

			return strings.Contains(origin, host)
		},
	}
}

// Getter functions for global variables
func GetConfig() map[string]interface{} {
	return cfg
}

func GetScriptSessionManager() *ScriptSessionManager {
	return scriptSessionManager
}

func GetBrowserManager() *BrowserManager {
	return browserManager
}

func GetWSManager() *WebSocketManager {
	return wsManager
}

func GetUpgrader() *websocket.Upgrader {
	return &upgrader
}

// Browser Management Functions

func getRandomUserAgent() string {
	return stealth.GetStealthUserAgent()
}

func getStealthJavaScript() string {
	return stealth.GetEnhancedStealthJS()
}

func (bm *BrowserManager) CreateBrowserSession(sessionID string, viewport Viewport) (*BrowserSession, error) {
	bm.sessionMutex.Lock()
	if bm.activeSessions >= bm.maxConcurrentSessions {
		bm.sessionMutex.Unlock()
		return nil, fmt.Errorf("maximum concurrent sessions reached (%d/%d)", bm.activeSessions, bm.maxConcurrentSessions)
	}
	bm.activeSessions++
	bm.sessionMutex.Unlock()

	bm.mutex.Lock()
	if existingSession, exists := bm.sessions[sessionID]; exists {
		log.Printf("🧹 Cleaning up existing session %s to prevent conflicts", sessionID)
		if existingSession.BrowserProcess != nil {
			existingSession.BrowserProcess.Process.Kill()
			existingSession.BrowserProcess.Wait()
		}
		delete(bm.sessions, sessionID)
		bm.sessionMutex.Lock()
		if bm.activeSessions > 0 {
			bm.activeSessions--
		}
		bm.sessionMutex.Unlock()
	}
	bm.mutex.Unlock()

	bm.portMutex.Lock()
	port := bm.nextPort
	for {
		if !bm.isPortInUse(port) {
			break
		}
		port++
		if port > 9999 {
			bm.portMutex.Unlock()
			bm.sessionMutex.Lock()
			bm.activeSessions--
			bm.sessionMutex.Unlock()
			return nil, fmt.Errorf("no available CDP ports")
		}
	}
	bm.nextPort = port + 1
	bm.portMutex.Unlock()

	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	args := stealth.GetStealthBrowserArguments(sessionID, port, viewport.Width, viewport.Height)

	chromePath := cfg["Browser"].(map[string]interface{})["ChromePath"].(string)
	if chromePath == "" {
		possiblePaths := []string{
			"google-chrome-stable",
			"google-chrome",
			"chromium-browser",
			"chromium",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
		}

		for _, path := range possiblePaths {
			if _, err := exec.LookPath(path); err == nil {
				chromePath = path
				log.Printf("🔍 Found Chrome at: %s", chromePath)
				break
			}
		}

		if chromePath == "" {
			chromePath = "google-chrome"
			log.Printf("⚠️ Using fallback Chrome path: %s", chromePath)
		}
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Env = append(os.Environ(), "DISPLAY="+cfg["Browser"].(map[string]interface{})["Display"].(string))

	log.Printf("🎭 Launching Chrome with ultra-stealth configuration")
	log.Printf("🕵️ Using stealth user agent from pool")
	log.Printf("📊 Total stealth args: %d", len(args))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %v", err)
	}

	time.Sleep(5 * time.Second)

	var websocketURL string
	maxRetries := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔍 Attempting to get CDP endpoint (attempt %d/%d)", attempt, maxRetries)

		cdpHost := cfg["CDP"].(map[string]interface{})["Host"].(string)
		resp, err := http.Get(fmt.Sprintf("http://%s:%d/json", cdpHost, port))
		if err == nil && resp.StatusCode == 200 {
			var tabs []map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&tabs) == nil && len(tabs) > 0 {
				for _, tab := range tabs {
					if tabType, ok := tab["type"].(string); ok && tabType == "page" {
						if wsURL, ok := tab["webSocketDebuggerUrl"].(string); ok {
							websocketURL = wsURL
							log.Printf("✅ Found page CDP endpoint: %s", websocketURL)
							break
						}
					}
				}

				if websocketURL == "" && len(tabs) > 0 {
					if wsURL, ok := tabs[0]["webSocketDebuggerUrl"].(string); ok {
						websocketURL = wsURL
						log.Printf("✅ Using first available CDP endpoint: %s", websocketURL)
					}
				}
			}
			resp.Body.Close()
		} else {
			log.Printf("⚠️ Failed to get CDP info (attempt %d): %v", attempt, err)
		}

		if websocketURL != "" {
			break
		}

		if attempt < maxRetries {
			time.Sleep(2 * time.Second)
		}
	}

	if websocketURL == "" {
		log.Printf("⚠️ Could not get CDP endpoint, using fallback")
		cdpHost := cfg["CDP"].(map[string]interface{})["Host"].(string)
		websocketURL = fmt.Sprintf("ws://%s:%d/devtools/browser", cdpHost, port)
	}

	session := &BrowserSession{
		ID:             sessionID,
		BrowserProcess: cmd,
		CDPEndpoint:    websocketURL,
		CDPPort:        port,
		Viewport:       viewport,
		Status:         "ready",
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		Streaming:      false,
		Metadata:       make(map[string]interface{}),
		HealthCheckInterval: 30 * time.Second,
		LastHealthCheck:     time.Now(),
		HealthStatus:        "unknown",
		ConnectionErrors:    0,
		AutoRecovery:        true,
		RecoveryAttempts:    0,
		MaxRecoveryAttempts: 3,
	}

	bm.sessions[sessionID] = session

	go func() {
		time.Sleep(3 * time.Second)
		
		cdpClient := NewCDPClient(websocketURL)
		if err := cdpClient.Connect(); err != nil {
			log.Printf("⚠️ Failed to connect CDP for stealth setup: %v", err)
			return
		}
		defer cdpClient.Close()

		if err := cdpClient.SetupStealthEnvironment(); err != nil {
			log.Printf("⚠️ Failed to setup stealth environment: %v", err)
		} else {
			log.Printf("🥷 Advanced stealth mode activated for session %s", sessionID)
			cdpClient.SetupPageNavigationStealth()
		}
	}()

	log.Printf("✅ Created browser session %s on port %d with enhanced stealth", sessionID, port)
	stealth.LogStealthMode(sessionID)

	return session, nil
}

func (bm *BrowserManager) isPortInUse(port int) bool {
	for _, session := range bm.sessions {
		if session.CDPPort == port {
			return true
		}
	}

	cdpHost := getEnv("CDP_HOST", "127.0.0.1")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cdpHost, port), 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func (bm *BrowserManager) GetSession(sessionID string) *BrowserSession {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.sessions[sessionID]
}

func (bm *BrowserManager) CloseSession(sessionID string) error {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if !exists {
		bm.mutex.Unlock()
		return fmt.Errorf("session not found")
	}
	// Get CDP port before deleting session
	cdpPort := session.CDPPort
	bm.mutex.Unlock() // Release BrowserManager lock BEFORE acquiring session lock
	
	// Now safely acquire session lock
	session.mutex.Lock()
	session.Streaming = false
	session.mutex.Unlock()
	
	// Re-acquire BrowserManager lock for cleanup
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if session.CleanupTimer != nil {
		session.CleanupTimer.Stop()
		session.CleanupTimer = nil
	}

	if session.BrowserProcess != nil {
		log.Printf("🔄 Killing browser process for session %s", sessionID)
		session.BrowserProcess.Process.Kill()
		session.BrowserProcess.Wait()
		log.Printf("✅ Browser process terminated for session %s", sessionID)
	}

	// Clean up any cached CDP client for this session
	bm.cdpCacheMutex.Lock()
	if client, ok := bm.cdpClientCache[sessionID]; ok && client != nil {
		client.Close()
		delete(bm.cdpClientCache, sessionID)
		log.Printf("🧹 Cleaned up cached CDP client for session %s", sessionID)
	}
	bm.cdpCacheMutex.Unlock()

	delete(bm.sessions, sessionID)
	bm.mutex.Unlock()

	bm.sessionMutex.Lock()
	if bm.activeSessions > 0 {
		bm.activeSessions--
	}
	bm.sessionMutex.Unlock()
	// Kill any remaining processes on the CDP port
	if cdpPort > 0 {
		log.Printf("🧹 Cleaning up CDP port %d for session %s", cdpPort, sessionID)
		cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", cdpPort))
		out, err := cmd.Output()
		if err != nil {
			log.Printf("⚠️ No processes found on port %d (this is normal): %v", cdpPort, err)
		} else {
			pid := strings.TrimSpace(string(out))
			if pid != "" {
				killCmd := exec.Command("kill", pid)
				if err := killCmd.Run(); err != nil {
					log.Printf("❌ Failed to kill process %s on port %d: %v", pid, cdpPort, err)
				} else {
					log.Printf("✅ Successfully killed process %s on port %d", pid, cdpPort)
				}
			}
		}
	}


	log.Printf("✅ Closed browser session %s (active sessions: %d/%d)", sessionID, bm.activeSessions, bm.maxConcurrentSessions)
	return nil
}
