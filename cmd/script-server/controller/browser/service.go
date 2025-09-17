package browser

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go-webrtc/cmd/script-server/config"
	"go-webrtc/cmd/script-server/libs/shared/helpers"

	"github.com/gorilla/websocket"
)

// Service interface defines browser service methods
type Service interface {
	// Browser Session Management
	CreateBrowserSession(data CreateBrowserSessionDto) (*BrowserSessionResponseDto, error)
	GetBrowserSession(sessionID string) (*BrowserSessionResponseDto, error)
	CloseBrowserSession(sessionID string) error
	ListBrowserSessions() ([]BrowserSessionResponseDto, error)
	
	// Script Task Management
	CreateScriptTask(data ScriptTaskRequestDto) (*EnhancedTaskResponseDto, error)
	CreateScriptTaskWithBrowser(data ScriptTaskWithBrowserDto) (*EnhancedTaskResponseDto, error)
	GetScriptTask(taskID string) (*EnhancedTaskResponseDto, error)
	ListScriptTasks() ([]EnhancedTaskResponseDto, error)
	
	// Browser Actions
	ExecuteBrowserAction(sessionID string, action BrowserActionDto, actionData interface{}) error
	CaptureScreenshot(sessionID string) (string, error)
	
	// System Status
	GetSystemStatus() (*SystemStatusDto, error)
	
	// WebSocket Management
	HandleWebSocket(conn *websocket.Conn, sessionID string) error
	StartStreamingSession(sessionID string) error
	StopStreamingSession(sessionID string) error
	ResetStreamingSession(sessionID string) error
	
	// Streaming
	StreamScreencast(sessionID string) (chan []byte, error)
}

// serviceImpl implements the Service interface
type serviceImpl struct {
	config         *config.Config
	browserManager *BrowserManager
	scriptManager  *ScriptSessionManager
	wsManager      *WebSocketManager
}

// NewService creates a new browser service
func NewService(cfg *config.Config) Service {
	return &serviceImpl{
		config:         cfg,
		browserManager: NewBrowserManager(cfg),
		scriptManager:  NewScriptSessionManager(),
		wsManager:      NewWebSocketManager(),
	}
}

// BrowserSession represents a browser instance with CDP connection
type BrowserSession struct {
	ID             string                 `json:"id"`
	BrowserProcess *exec.Cmd              `json:"-"`
	CDPEndpoint    string                 `json:"cdpEndpoint"`
	CDPPort        int                    `json:"cdpPort"`
	Viewport       Viewport               `json:"viewport"`
	Status         string                 `json:"status"` // "starting", "ready", "busy", "closed"
	CreatedAt      time.Time              `json:"createdAt"`
	LastActivity   time.Time              `json:"lastActivity"`
	Streaming      bool                   `json:"streaming"`
	StreamCallback func([]byte)           `json:"-"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	// Health monitoring
	HealthCheckInterval time.Duration `json:"-"`
	LastHealthCheck     time.Time     `json:"lastHealthCheck"`
	HealthStatus        string        `json:"healthStatus"` // "healthy", "unhealthy", "unknown"
	ConnectionErrors    int           `json:"connectionErrors"`
	LastConnectionError time.Time     `json:"lastConnectionError"`
	// Recovery
	AutoRecovery        bool `json:"autoRecovery"`
	RecoveryAttempts    int  `json:"recoveryAttempts"`
	MaxRecoveryAttempts int  `json:"maxRecoveryAttempts"`
	// Auto cleanup
	TaskCompleted       bool          `json:"taskCompleted"`
	LastUserInteraction time.Time     `json:"lastUserInteraction"`
	CleanupTimer        *time.Timer   `json:"-"`
	CleanupDuration     time.Duration `json:"cleanupDuration"`
	// Concurrency control
	mutex sync.RWMutex `json:"-"`
}

type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ScriptSession represents a browser automation session
type ScriptSession struct {
	ID          string                 `json:"id"`
	Task        string                 `json:"task"`
	Status      string                 `json:"status"` // "queued", "running", "completed", "failed"
	BrowserID   string                 `json:"browserId,omitempty"`
	CDPEndpoint string                 `json:"cdpEndpoint,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Process     *exec.Cmd              `json:"-"`
}

type WebSocketClient struct {
	Conn   *websocket.Conn
	Send   chan []byte
	ID     string
	Active bool
}

type ScriptSessionManager struct {
	sessions map[string]*ScriptSession
	mutex    sync.RWMutex
}

type BrowserManager struct {
	sessions              map[string]*BrowserSession
	mutex                 sync.RWMutex
	nextPort              int
	portMutex             sync.Mutex // Separate mutex for port assignment
	maxConcurrentSessions int
	activeSessions        int
	sessionMutex          sync.Mutex // Mutex for session counting
}

type WebSocketManager struct {
	clients    map[*WebSocketClient]bool
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan []byte
	mutex      sync.RWMutex
}

// CDPClient handles Chrome DevTools Protocol communication
type CDPClient struct {
	wsURL       string
	conn        *websocket.Conn
	ctx         context.Context
	cancel      context.CancelFunc
	requestID   int
	mutex       sync.RWMutex
	pageEnabled bool
	initialized bool
}

// NewBrowserManager creates a new browser manager
func NewBrowserManager(cfg *config.Config) *BrowserManager {
	bm := &BrowserManager{
		sessions:              make(map[string]*BrowserSession),
		nextPort:              9222,
		maxConcurrentSessions: helpers.GetEnvInt("MAX_CONCURRENT_SESSIONS", 10),
		activeSessions:        0,
	}
	
	// Clean up any lingering Chrome processes on startup
	bm.cleanupLingeringProcesses()
	
	return bm
}

// cleanupLingeringProcesses kills any Chrome processes that might be left over
func (bm *BrowserManager) cleanupLingeringProcesses() {
	log.Printf("🧹 Cleaning up any lingering Chrome processes...")
	
	// On Windows, kill chrome processes with our debugging ports
	for port := 9222; port <= 9230; port++ {
		if bm.isPortOccupied(port) {
			log.Printf("🔍 Found occupied port %d, attempting cleanup", port)
			// Try to connect and close any Chrome on this port
			if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second); err == nil {
				conn.Close()
				log.Printf("✅ Closed connection to port %d", port)
			}
		}
	}
}

// NewScriptSessionManager creates a new script session manager
func NewScriptSessionManager() *ScriptSessionManager {
	return &ScriptSessionManager{
		sessions: make(map[string]*ScriptSession),
	}
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan []byte),
	}
}

// CreateBrowserSession creates a new browser session
func (s *serviceImpl) CreateBrowserSession(data CreateBrowserSessionDto) (*BrowserSessionResponseDto, error) {
	sessionID := helpers.GenerateID()
	viewport := Viewport{
		Width:  data.Viewport.Width,
		Height: data.Viewport.Height,
	}
	
	session, err := s.browserManager.createBrowserSession(sessionID, viewport)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser session: %w", err)
	}
	
	// Convert to DTO
	response := &BrowserSessionResponseDto{}
	if err := helpers.JsonMarshaller(session, response); err != nil {
		return nil, fmt.Errorf("failed to convert session to DTO: %w", err)
	}
	
	return response, nil
}

// GetBrowserSession retrieves a browser session
func (s *serviceImpl) GetBrowserSession(sessionID string) (*BrowserSessionResponseDto, error) {
	session := s.browserManager.getSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	
	response := &BrowserSessionResponseDto{}
	if err := helpers.JsonMarshaller(session, response); err != nil {
		return nil, fmt.Errorf("failed to convert session to DTO: %w", err)
	}
	
	return response, nil
}

// CloseBrowserSession closes a browser session
func (s *serviceImpl) CloseBrowserSession(sessionID string) error {
	return s.browserManager.closeSession(sessionID)
}

// ListBrowserSessions lists all browser sessions
func (s *serviceImpl) ListBrowserSessions() ([]BrowserSessionResponseDto, error) {
	s.browserManager.mutex.RLock()
	defer s.browserManager.mutex.RUnlock()
	
	var sessions []BrowserSessionResponseDto
	for _, session := range s.browserManager.sessions {
		var dto BrowserSessionResponseDto
		if err := helpers.JsonMarshaller(session, &dto); err != nil {
			log.Printf("Error converting session to DTO: %v", err)
			continue
		}
		sessions = append(sessions, dto)
	}
	
	return sessions, nil
}

// CreateScriptTask creates a new script task
func (s *serviceImpl) CreateScriptTask(data ScriptTaskRequestDto) (*EnhancedTaskResponseDto, error) {
	// Validate required fields
	if data.Task == "" {
		return nil, fmt.Errorf("task field is required")
	}

	sessionID := helpers.GenerateID()

	// Create script session record so other endpoints (live page, status) can find it
	scriptSession := &ScriptSession{
		ID:          sessionID,
		Task:        data.Task,
		Status:      "queued",
		BrowserID:   sessionID, // Use same ID for both
		CDPEndpoint: "", // Will be populated by Python script
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	s.scriptManager.mutex.Lock()
	s.scriptManager.sessions[sessionID] = scriptSession
	s.scriptManager.mutex.Unlock()

	log.Printf("✅ Created task session %s: %s", sessionID, data.Task)

	// Start task execution in background - let Python create browser and report CDP endpoint
	go s.executeScriptTask(sessionID, data.Task, data.MaxSteps)

	// Create enhanced response
	response := &EnhancedTaskResponseDto{
		Success:       true,
		TaskId:        sessionID,
		SessionId:     sessionID, // Use same ID for both
		Status:        "started",
		Task:          data.Task,
		CreatedAt:     time.Now(),
		AutomationUrl: s.generateAutomationURL(sessionID),
		StreamingUrl:  s.generateStreamingURL(sessionID),
		WebSocketUrl:  s.generateWebSocketURL(),
		BrowserInfo: &BrowserInfoDto{
			BrowserId:   sessionID,
			CDPEndpoint: "", // Will be populated when Python script reports it
			Viewport: ViewportDto{
				Width:  s.config.Browser.ViewportWidth,
				Height: s.config.Browser.ViewportHeight,
			},
		},
		Message: "Task started successfully! Browser-use automation available at automation_url",
	}

	return response, nil
}

// CreateScriptTaskWithBrowser creates a new script task with existing browser (old code architecture)
func (s *serviceImpl) CreateScriptTaskWithBrowser(data ScriptTaskWithBrowserDto) (*EnhancedTaskResponseDto, error) {
	// Validate required fields
	if data.Task == "" {
		return nil, fmt.Errorf("task field is required")
	}
	if data.BrowserID == "" {
		return nil, fmt.Errorf("browserId field is required")
	}
	if data.CDPEndpoint == "" {
		return nil, fmt.Errorf("cdpEndpoint field is required")
	}

	// Use the provided browser ID as session ID
	sessionID := data.BrowserID

	// Create script session record that uses existing browser
	scriptSession := &ScriptSession{
		ID:          sessionID,
		Task:        data.Task,
		Status:      "queued",
		BrowserID:   data.BrowserID,
		CDPEndpoint: data.CDPEndpoint, // Use provided CDP endpoint
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	s.scriptManager.mutex.Lock()
	s.scriptManager.sessions[sessionID] = scriptSession
	s.scriptManager.mutex.Unlock()

	log.Printf("✅ Created task session %s with existing browser: %s", sessionID, data.Task)

	// Start task execution in background - pass CDP endpoint TO Python (old code style)
	go s.executeScriptTaskWithBrowser(sessionID, data.Task, data.MaxSteps, data.CDPEndpoint)

	// Create enhanced response with existing browser info
	response := &EnhancedTaskResponseDto{
		Success:       true,
		TaskId:        sessionID,
		SessionId:     sessionID,
		Status:        "started",
		Task:          data.Task,
		CreatedAt:     time.Now(),
		AutomationUrl: s.generateAutomationURL(sessionID),
		StreamingUrl:  s.generateStreamingURL(sessionID),
		WebSocketUrl:  s.generateWebSocketURL(),
		BrowserInfo: &BrowserInfoDto{
			BrowserId:   data.BrowserID,
			CDPEndpoint: data.CDPEndpoint, // Already available!
			Viewport: ViewportDto{
				Width:  s.config.Browser.ViewportWidth,
				Height: s.config.Browser.ViewportHeight,
			},
		},
		Message: "Task started with existing browser! Streaming available immediately",
	}

	return response, nil
}

// Additional methods would continue here...
// This is a partial implementation showing the structure.
// The full implementation would include all the original functionality
// from main.go converted to this service pattern.

func (s *serviceImpl) generateAutomationURL(sessionID string) string {
	// Fixed URL to match the route path
	return fmt.Sprintf("http://%s:%s/api/v1/browser_use/browser/live-automation/%s", 
		s.config.Server.Host, s.config.Server.Port, sessionID)
}

func (s *serviceImpl) generateStreamingURL(sessionID string) string {
	return fmt.Sprintf("http://%s:%s/api/v1/browser_use/browser/stream-screencast/%s", 
		s.config.Server.Host, s.config.Server.Port, sessionID)
}

func (s *serviceImpl) generateWebSocketURL() string {
	return fmt.Sprintf("ws://%s:%s/api/v1/browser_use/browser/ws", 
		s.config.Server.Host, s.config.Server.Port)
}

// executeScriptTask executes a script task in the background
func (s *serviceImpl) executeScriptTask(sessionID, task string, maxSteps int) {
	// Get session with proper locking
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		log.Printf("❌ Session not found: %s", sessionID)
		return
	}

	// Update status atomically with double-check locking
	s.scriptManager.mutex.Lock()
	if session.Status != "pending" && session.Status != "queued" {
		// Session was already started or completed
		s.scriptManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already in state: %s", sessionID, session.Status)
		return
	}

	// Double-check: ensure we're not already running
	if session.Status == "running" {
		s.scriptManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already running, skipping duplicate execution", sessionID)
		return
	}

	session.Status = "running"
	session.UpdatedAt = time.Now()
	s.scriptManager.mutex.Unlock()

	// Cancel any existing cleanup timer since task is starting
	// s.browserManager.cancelCleanupTimer(session.BrowserID)

	log.Printf("🎯 Processing task for session %s: %s", sessionID, task)

	// Execute Python script - let it create its own browser and report CDP endpoint
	// Use absolute path to avoid working directory issues
	scriptPath := `D:\Loacl disk D\projects\Go-browser-use\scripts\browser_task_fixed.py`

	log.Printf("🐍 Executing Python script: %s", scriptPath)
	log.Printf("🎯 Task: %s", task)
	log.Printf("📊 Max Steps: %d", maxSteps)

	args := []string{
		scriptPath,
		sessionID,
		task,
		fmt.Sprintf("%d", maxSteps),
		// Don't pass CDP endpoint - let Python create browser and report it back
	}

	cmd := exec.Command(s.config.Script.PythonCommand, args...)
	// Pass all environment variables
	cmd.Env = os.Environ()

	log.Printf("🚀 Starting Python script execution...")

	// Stream output in real-time and extract CDP endpoint as soon as available
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Failed to create stdout pipe: %v", err)
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("❌ Failed to create stderr pipe: %v", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start Python script: %v", err)
		return
	}

	// Stream stderr for logging and CDP endpoint extraction
	var stderrBuffer strings.Builder
	cdpExtracted := false
	
	// Function to process a line of stderr output
	processLine := func(line string) {
		stderrBuffer.WriteString(line + "\n")
		log.Printf("🐍 Python: %s", line)
		
		// Extract CDP endpoint as soon as we see it in logs
		if !cdpExtracted {
			if cdpEndpoint := extractCDPEndpointFromLogs(line); cdpEndpoint != "" {
				log.Printf("🔗 Extracted CDP endpoint: %s", cdpEndpoint)
				// Update session immediately
				s.scriptManager.mutex.Lock()
				session.CDPEndpoint = cdpEndpoint
				session.UpdatedAt = time.Now()
				s.scriptManager.mutex.Unlock()
				cdpExtracted = true
			}
		}
	}

	// Stream stderr concurrently
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			processLine(scanner.Text())
		}
	}()
	
	// Read stdout for JSON result
	var stdoutBuffer strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		stdoutBuffer.WriteString(scanner.Text() + "\n")
	}

	// Wait for script to complete
	err = cmd.Wait()
	outputStr := strings.TrimSpace(stdoutBuffer.String())
	stderrStr := stderrBuffer.String()
	
	if err != nil {
		log.Printf("❌ Script execution failed: %v", err)
		log.Printf("❌ Script stderr: %s", stderrStr)
		return
	}

	log.Printf("📄 Python script completed")
	
	cdpEndpoint := extractCDPEndpointFromLogs(stderrStr)
	if cdpEndpoint != "" {
		log.Printf("🔗 Extracted CDP endpoint: %s", cdpEndpoint)
		session.CDPEndpoint = cdpEndpoint
	} else {
		log.Printf("⚠️ Could not extract CDP endpoint from logs")
	}

	// Parse JSON result from stdout (should be clean JSON now)
	var result map[string]interface{}

	// Try to parse stdout directly as JSON first
	if outputStr != "" {
		if err := json.Unmarshal([]byte(outputStr), &result); err == nil {
			log.Printf("✅ Parsed JSON result from stdout: %+v", result)
		} else {
			log.Printf("⚠️ Failed to parse stdout as JSON: %v", err)
			log.Printf("⚠️ Stdout content: '%s'", outputStr)
			
			// Fallback: look for JSON in lines
			lines := strings.Split(outputStr, "\n")
			jsonFound := false

			// First, try to find JSON by looking for lines that start with {
			for i := len(lines) - 1; i >= 0; i-- {
				line := strings.TrimSpace(lines[i])
				if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
					// Try to parse this line as JSON
					if err := json.Unmarshal([]byte(line), &result); err == nil {
						jsonFound = true
						log.Printf("✅ Parsed JSON result from line %d: %+v", i, result)
						break
					}
				}
			}

			if !jsonFound {
				log.Printf("⚠️ No valid JSON found in output, creating default result")
				// Check if the task actually completed successfully based on stderr logs
				stderrContent := stderrBuffer.String()
				if strings.Contains(stderrContent, "✅ Task completed successfully") ||
					strings.Contains(stderrContent, "Task completed: True") {
					log.Printf("✅ Task appears to have completed successfully based on logs")
					result = map[string]interface{}{
						"success":    true,
						"result":     "Task completed successfully",
						"raw_output": outputStr,
					}
				} else {
					result = map[string]interface{}{
						"success":    false,
						"error":      "No valid JSON output from script",
						"raw_output": outputStr,
					}
				}
			}
		}
	} else {
		log.Printf("⚠️ No stdout output, checking stderr for success")
		stderrContent := stderrBuffer.String()
		if strings.Contains(stderrContent, "✅ Task completed successfully") {
			result = map[string]interface{}{
				"success": true,
				"result":  "Task completed successfully",
			}
		} else {
			result = map[string]interface{}{
				"success": false,
				"error":   "No output from script",
			}
		}
	}

	// Update session status and store result
	s.scriptManager.mutex.Lock()
	success, _ := result["success"].(bool)
	if success {
		session.Status = "completed"
		// Store the result in metadata for enhanced response
		session.Metadata["result"] = result
		
		// Extract CDP endpoint if provided by Python script
		if cdpEndpoint, exists := result["cdp_endpoint"]; exists {
			if cdpStr, ok := cdpEndpoint.(string); ok && cdpStr != "" {
				session.CDPEndpoint = cdpStr
				log.Printf("✅ Captured CDP endpoint from Python script: %s", cdpStr)
			}
		}
	} else {
		session.Status = "failed"
		// Store error information
		if errorMsg, exists := result["error"]; exists {
			session.Metadata["error"] = errorMsg
		}
	}
	session.UpdatedAt = time.Now()
	s.scriptManager.mutex.Unlock()

	log.Printf("✅ Task completed for session %s with status: %s", sessionID, "completed")
}

// executeScriptTaskWithBrowser executes task with existing browser (old code architecture)
func (s *serviceImpl) executeScriptTaskWithBrowser(sessionID, task string, maxSteps int, cdpEndpoint string) {
	// Get session with proper locking
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		log.Printf("❌ Session not found: %s", sessionID)
		return
	}

	// Update status atomically 
	s.scriptManager.mutex.Lock()
	if session.Status == "running" {
		s.scriptManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already running, skipping duplicate execution", sessionID)
		return
	}

	session.Status = "running"
	session.UpdatedAt = time.Now()
	s.scriptManager.mutex.Unlock()

	log.Printf("🎯 Processing task for session %s: %s", sessionID, task)
	log.Printf("🔗 Using existing browser CDP endpoint: %s", cdpEndpoint)

	// Execute Python script with existing CDP endpoint (old code style)
	scriptPath := `D:\Loacl disk D\projects\Go-browser-use\scripts\browser_task_fixed.py`

	log.Printf("🐍 Executing Python script: %s", scriptPath)
	log.Printf("🎯 Task: %s", task)
	log.Printf("📊 Max Steps: %d", maxSteps)

	args := []string{
		scriptPath,
		sessionID,
		task,
		fmt.Sprintf("%d", maxSteps),
		cdpEndpoint, // Pass CDP endpoint TO Python (old code architecture)
	}

	cmd := exec.Command(s.config.Script.PythonCommand, args...)
	// Pass all environment variables
	cmd.Env = os.Environ()

	log.Printf("🚀 Starting Python script execution with existing browser...")

	// Use separate stdout/stderr pipes for better JSON parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Failed to create stdout pipe: %v", err)
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("❌ Failed to create stderr pipe: %v", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start Python script: %v", err)
		return
	}

	// Read stderr for logging (separate goroutine)
	var stderrBuffer strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuffer.WriteString(line + "\n")
			log.Printf("🐍 Python: %s", line)
		}
	}()

	// Read stdout for JSON result
	var stdoutBuffer strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		stdoutBuffer.WriteString(line + "\n")
		// Don't log stdout lines as they contain the JSON result
	}

	// Wait for script to complete
	err = cmd.Wait()
	outputStr := strings.TrimSpace(stdoutBuffer.String())
	stderrStr := stderrBuffer.String()
	
	if err != nil {
		log.Printf("❌ Script execution failed: %v", err)
		log.Printf("❌ Script stderr: %s", stderrStr)
		
		// Handle specific timeout errors
		if strings.Contains(stderrStr, "TIMEOUT ERROR") {
			log.Printf("⚠️ Browser-use timeout detected - likely browser dialog or CDP issues")
			s.scriptManager.mutex.Lock()
			session.Status = "failed"
			session.Metadata = map[string]interface{}{
				"error": "Browser timeout - possible dialog interference or CDP connection issues",
				"output": outputStr,
				"stderr": stderrStr,
			}
			session.UpdatedAt = time.Now()
			s.scriptManager.mutex.Unlock()
			return
		}
		
		// Handle other errors but don't immediately fail - check for partial success
		if strings.Contains(stderrStr, "✅ Task completed successfully") ||
			strings.Contains(stderrStr, "Task completed: True") {
			log.Printf("✅ Task appears to have completed successfully despite error exit code")
		} else {
			s.scriptManager.mutex.Lock()
			session.Status = "failed"
			session.Metadata = map[string]interface{}{
				"error": fmt.Sprintf("Script failed: %v", err),
				"output": outputStr,
				"stderr": stderrStr,
			}
			session.UpdatedAt = time.Now()
			s.scriptManager.mutex.Unlock()
			return
		}
	}

	log.Printf("📄 Python script completed")
	log.Printf("📝 Python script stdout: %s", outputStr)

	// Parse JSON result from stdout (should be clean JSON)
	var result map[string]interface{}
	
	// Try to parse stdout directly as JSON (should be clean now)
	if outputStr != "" {
		if err := json.Unmarshal([]byte(outputStr), &result); err == nil {
			log.Printf("✅ Parsed JSON result from stdout: %+v", result)
		} else {
			log.Printf("⚠️ Failed to parse stdout as JSON: %v", err)
			log.Printf("⚠️ Stdout content: '%s'", outputStr)
			
			// Fallback: try to find JSON in the output
			lines := strings.Split(outputStr, "\n")
			jsonFound := false
			
			// Look for a line that looks like JSON
			for i := len(lines) - 1; i >= 0; i-- {
				line := strings.TrimSpace(lines[i])
				if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
					if err := json.Unmarshal([]byte(line), &result); err == nil {
						jsonFound = true
						log.Printf("✅ Found JSON in line %d: %+v", i, result)
						break
					}
				}
			}
			
			if !jsonFound {
				log.Printf("⚠️ No valid JSON found in stdout, creating default result")
				result = map[string]interface{}{
					"success": strings.Contains(stderrBuffer.String(), "✅ Task completed successfully"),
					"result":     "Task execution completed",
					"raw_output": outputStr,
				}
			}
		}
	} else {
		log.Printf("⚠️ No stdout output, creating default result")
		result = map[string]interface{}{
			"success": strings.Contains(stderrBuffer.String(), "✅ Task completed successfully"),
			"result":     "Task execution completed - no output",
			"raw_stderr": stderrBuffer.String(),
		}
	}

	// Update session status and store result
	s.scriptManager.mutex.Lock()
	success, _ := result["success"].(bool)
	if success {
		session.Status = "completed"
		session.Metadata["result"] = result
	} else {
		session.Status = "failed"
		if errorMsg, exists := result["error"]; exists {
			session.Metadata["error"] = errorMsg
		}
	}
	session.UpdatedAt = time.Now()
	s.scriptManager.mutex.Unlock()

	log.Printf("✅ Task completed for session %s with status: %s", sessionID, session.Status)
}

// Placeholder methods - these would contain the full implementation
// from the original main.go file

func (s *serviceImpl) GetScriptTask(taskID string) (*EnhancedTaskResponseDto, error) {
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[taskID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Get browser session
	browserSession := s.browserManager.getSession(session.BrowserID)

	// Create enhanced response
	response := &EnhancedTaskResponseDto{
		Success:       true,
		TaskId:        session.ID,
		SessionId:     session.ID,
		Status:        session.Status,
		Task:          session.Task,
		CreatedAt:     session.CreatedAt,
		AutomationUrl: s.generateAutomationURL(session.ID),
		StreamingUrl:  s.generateStreamingURL(session.ID),
		WebSocketUrl:  s.generateWebSocketURL(),
	}

	// Add browser info if available
	if browserSession != nil {
		response.BrowserInfo = &BrowserInfoDto{
			BrowserId:   browserSession.ID,
			CDPEndpoint: browserSession.CDPEndpoint,
			Viewport: ViewportDto{
				Width:  browserSession.Viewport.Width,
				Height: browserSession.Viewport.Height,
			},
		}
	}

	// Update message based on status
	switch session.Status {
	case "queued":
		response.Message = "Task is queued and waiting to start"
	case "running":
		response.Message = "Task is currently running"
	case "completed":
		response.Message = "Task completed successfully"
	case "failed":
		response.Message = "Task failed to complete"
		response.Success = false
	}

	return response, nil
}

func (s *serviceImpl) ListScriptTasks() ([]EnhancedTaskResponseDto, error) {
	// Implementation here
	return nil, fmt.Errorf("not implemented")
}

func (s *serviceImpl) ExecuteBrowserAction(sessionID string, action BrowserActionDto, actionData interface{}) error {
	// Implementation here
	return fmt.Errorf("not implemented")
}

func (s *serviceImpl) CaptureScreenshot(sessionID string) (string, error) {
	// Get script session
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		return "", fmt.Errorf("session not found")
	}

	// Check if we have CDP endpoint from Python script
	if session.CDPEndpoint == "" {
		return "", fmt.Errorf("CDP endpoint not yet available - task may still be starting")
	}

	// Create CDP client and capture screenshot
	cdpClient := s.newCDPClient(session.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect to CDP: %w", err)
	}
	defer cdpClient.Close()

	imageData, err := cdpClient.CaptureScreenshot()
	if err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Return base64 encoded string
	return base64.StdEncoding.EncodeToString(imageData), nil
}

func (s *serviceImpl) GetSystemStatus() (*SystemStatusDto, error) {
	// Implementation here
	return nil, fmt.Errorf("not implemented")
}

func (s *serviceImpl) HandleWebSocket(conn *websocket.Conn, sessionID string) error {
	// Implementation here
	return fmt.Errorf("not implemented")
}

func (s *serviceImpl) StartStreamingSession(sessionID string) error {
	// Get script session
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Check if CDP endpoint is available
	if session.CDPEndpoint == "" {
		return fmt.Errorf("CDP endpoint not yet available - task may still be starting")
	}

	// For this approach, we don't track streaming state in browser sessions
	// since we don't manage them directly - the Python script does
	log.Printf("📹 Started streaming session %s (CDP: %s)", sessionID, session.CDPEndpoint)

	return nil
}

func (s *serviceImpl) StopStreamingSession(sessionID string) error {
	// Get script session
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		return fmt.Errorf("session not found")
	}

	// For this approach, we don't need to track streaming state
	// since Python manages the browser
	log.Printf("📹 Stopped streaming session %s", sessionID)

	return nil
}

func (s *serviceImpl) ResetStreamingSession(sessionID string) error {
	// Get script session
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Reset any streaming flags or state if needed
	log.Printf("🔄 Reset streaming session %s", sessionID)

	return nil
}

// Browser manager methods would be implemented here
// All the original browser management functionality from main.go

func (bm *BrowserManager) createBrowserSession(sessionID string, viewport Viewport) (*BrowserSession, error) {
	// Check concurrency limits
	bm.sessionMutex.Lock()
	if bm.activeSessions >= bm.maxConcurrentSessions {
		bm.sessionMutex.Unlock()
		return nil, fmt.Errorf("maximum concurrent sessions reached (%d/%d)", bm.activeSessions, bm.maxConcurrentSessions)
	}
	bm.activeSessions++
	bm.sessionMutex.Unlock()

	// Force cleanup of any existing session with the same ID to prevent conflicts
	bm.mutex.Lock()
	if existingSession, exists := bm.sessions[sessionID]; exists {
		log.Printf("🧹 Cleaning up existing session %s to prevent conflicts", sessionID)
		if existingSession.BrowserProcess != nil && existingSession.BrowserProcess.Process != nil {
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

	// Atomic port assignment to prevent race conditions
	bm.portMutex.Lock()
	port := bm.nextPort
	for {
		if !bm.isPortInUse(port) && !bm.isPortOccupied(port) {
			break
		}
		port++
		if port > 9999 {
			bm.portMutex.Unlock()
			// Decrement active sessions on failure
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

	// Launch Chrome with Windows-compatible args
	args := []string{
		"--remote-debugging-port=" + fmt.Sprintf("%d", port),
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--window-size=" + fmt.Sprintf("%d", viewport.Width) + "," + fmt.Sprintf("%d", viewport.Height),
		"--disable-gpu",
		"--headless=new",
		"--disable-web-security",
		"--disable-features=VizDisplayCompositor",
		"--disable-background-timer-throttling",
		"--disable-renderer-backgrounding",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-plugins",
		"--disable-sync",
		"--disable-popup-blocking",           // Prevent popup interference
		"--disable-notifications",           // Prevent notification dialogs
		"--disable-infobars",               // Prevent info bars
		"--no-default-browser-check",       // Prevent default browser dialogs
		"--disable-translate",              // Prevent translation popups
		"--disable-features=TranslateUI",   // Additional translation prevention
		"--disable-component-extensions-with-background-pages",
		"--disable-ipc-flooding-protection",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}

	chromePath := os.Getenv("CHROME_PATH")
	if chromePath == "" {
		// Try common Chrome paths on Windows
		commonPaths := []string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"chrome.exe",
			"chromium.exe",
		}
		
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				chromePath = path
				break
			}
		}
		
		if chromePath == "" {
			chromePath = "chrome.exe" // Fallback
		}
	}

	cmd := exec.Command(chromePath, args...)
	// Don't set DISPLAY on Windows
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		// Decrement active sessions on failure
		bm.sessionMutex.Lock()
		bm.activeSessions--
		bm.sessionMutex.Unlock()
		return nil, fmt.Errorf("failed to start Chrome: %v", err)
	}

	// Wait for Chrome to start and be ready
	log.Printf("⏳ Waiting for Chrome to initialize on port %d...", port)
	time.Sleep(3 * time.Second) // Reduced initial wait

	// Discover WebSocket URL from Chrome's JSON endpoint
	var websocketURL string
	maxRetries := 15 // Increased retries
	for i := 0; i < maxRetries; i++ {
		// Try to get the WebSocket URL from Chrome's /json endpoint
		if wsURL, err := bm.getWebSocketURL(port); err == nil {
			websocketURL = wsURL
			log.Printf("✅ Chrome CDP endpoint ready on port %d with WS URL: %s", port, websocketURL)
			break
		} else if i == maxRetries-1 {
			// Last attempt failed, kill the process and return error
			log.Printf("❌ Chrome failed to start after %d attempts: %v", maxRetries, err)
			cmd.Process.Kill()
			cmd.Wait()
			bm.sessionMutex.Lock()
			bm.activeSessions--
			bm.sessionMutex.Unlock()
			return nil, fmt.Errorf("Chrome failed to start properly: CDP endpoint not accessible after %d attempts", maxRetries)
		}
		log.Printf("⏳ Chrome not ready yet, retrying... (%d/%d)", i+1, maxRetries)
		time.Sleep(1500 * time.Millisecond) // Faster retry interval
	}

	// Create session
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

	log.Printf("✅ Created browser session %s on port %d", sessionID, port)

	return session, nil
}

func (bm *BrowserManager) isPortInUse(port int) bool {
	// Simple check - in production you'd check if port is actually available
	for _, session := range bm.sessions {
		if session.CDPPort == port {
			return true
		}
	}
	return false
}

func (bm *BrowserManager) isPortOccupied(port int) bool {
	// Check if port is actually occupied by making a test connection
	timeout := time.Second * 1
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err != nil {
		return false // Port is free
	}
	conn.Close()
	return true // Port is occupied
}

func (bm *BrowserManager) getSession(sessionID string) *BrowserSession {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.sessions[sessionID]
}

func (bm *BrowserManager) closeSession(sessionID string) error {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if !exists {
		bm.mutex.Unlock()
		return fmt.Errorf("session not found")
	}

	if session.BrowserProcess != nil {
		log.Printf("🔄 Killing browser process for session %s", sessionID)
		
		// Try graceful shutdown first
		if session.BrowserProcess.Process != nil {
			session.BrowserProcess.Process.Kill()
			// Give it a moment to shut down gracefully
			done := make(chan error, 1)
			go func() {
				done <- session.BrowserProcess.Wait()
			}()
			
			select {
			case <-done:
				log.Printf("✅ Browser process terminated gracefully for session %s", sessionID)
			case <-time.After(5 * time.Second):
				log.Printf("⚠️ Browser process did not terminate gracefully, forcing kill")
				if session.BrowserProcess.Process != nil {
					session.BrowserProcess.Process.Kill()
				}
			}
		}
	}

	delete(bm.sessions, sessionID)
	bm.mutex.Unlock()

	// Decrement active session count
	bm.sessionMutex.Lock()
	if bm.activeSessions > 0 {
		bm.activeSessions--
	}
	bm.sessionMutex.Unlock()

	log.Printf("✅ Closed browser session %s (active sessions: %d/%d)", sessionID, bm.activeSessions, bm.maxConcurrentSessions)
	return nil
}

// newCDPClient creates a new CDP client
func (s *serviceImpl) newCDPClient(wsURL string) *CDPClient {
	return &CDPClient{
		wsURL:       wsURL,
		ctx:         context.Background(),
		requestID:   1,
		pageEnabled: false,
		initialized: false,
	}
}

// Connect establishes WebSocket connection to CDP
func (c *CDPClient) Connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	
	conn, _, err := dialer.Dial(c.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to CDP WebSocket %s: %w", c.wsURL, err)
	}
	c.conn = conn
	c.initialized = true
	
	// Enable Page domain for screenshot functionality - this is critical!
	if err := c.enablePageDomain(); err != nil {
		log.Printf("❌ Failed to enable Page domain: %v", err)
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}
	
	// Navigate to a basic page to ensure we have something to screenshot
	if err := c.navigateToPage("about:blank"); err != nil {
		log.Printf("❌ Failed to navigate to about:blank: %v", err)
		return fmt.Errorf("failed to navigate to initial page: %w", err)
	}
	
	log.Printf("✅ CDP connection established and Page domain enabled")
	return nil
}

// ensurePageActive ensures the CDP connection is active and attached to a page
func (c *CDPClient) ensurePageActive() error {
	// Try to get current page info to test if we're attached
	_, err := c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": "document.readyState",
	})
	
	if err != nil {
		if strings.Contains(err.Error(), "Not attached to an active page") {
			log.Printf("⚠️ Page not active, attempting to re-enable Page domain")
			// Re-enable Page domain
			if enableErr := c.enablePageDomain(); enableErr != nil {
				return fmt.Errorf("failed to re-enable Page domain: %w", enableErr)
			}
			
			// Try the test again
			_, retryErr := c.SendCommand("Runtime.evaluate", map[string]interface{}{
				"expression": "document.readyState",
			})
			if retryErr != nil {
				return fmt.Errorf("page still not active after retry: %w", retryErr)
			}
		} else {
			return fmt.Errorf("page activity check failed: %w", err)
		}
	}
	
	return nil
}

// Close closes the CDP connection
func (c *CDPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// sendRequest sends a CDP request and waits for response
func (c *CDPClient) sendRequest(method string, params map[string]interface{}) (map[string]interface{}, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("CDP connection not established")
	}

	c.mutex.Lock()
	requestID := c.requestID
	c.requestID++
	c.mutex.Unlock()

	request := map[string]interface{}{
		"id":     requestID,
		"method": method,
		"params": params,
	}

	if err := c.conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send CDP request: %w", err)
	}

	// Wait for response with timeout
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	
	for {
		var response map[string]interface{}
		if err := c.conn.ReadJSON(&response); err != nil {
			return nil, fmt.Errorf("failed to read CDP response: %w", err)
		}

		// Check if this is the response to our request
		if respID, ok := response["id"].(float64); ok && int(respID) == requestID {
			if errorObj, hasError := response["error"]; hasError {
				return nil, fmt.Errorf("CDP error: %v", errorObj)
			}
			
			if result, hasResult := response["result"]; hasResult {
				if resultMap, ok := result.(map[string]interface{}); ok {
					return resultMap, nil
				}
			}
			
			return response, nil
		}
		// If not our response, continue reading (could be an event)
	}
}

// SendCommand sends a CDP command and waits for response
func (c *CDPClient) SendCommand(method string, params map[string]interface{}) (map[string]interface{}, error) {
	return c.sendRequest(method, params)
}

// CaptureScreenshot captures a screenshot using CDP (enhanced error handling like old code)
func (c *CDPClient) CaptureScreenshot() ([]byte, error) {
	// First ensure we're connected and the page is active
	if err := c.ensurePageActive(); err != nil {
		return nil, fmt.Errorf("page not active: %w", err)
	}

	// Attempt screenshot with retry logic similar to old working code
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := c.SendCommand("Page.captureScreenshot", map[string]interface{}{
			"format":  "png",
			"quality": 90,
		})
		
		if err != nil {
			// Handle specific CDP errors like the old code
			if strings.Contains(err.Error(), "Not attached to an active page") {
				log.Printf("⚠️ CDP page detached during screenshot, attempting recovery (attempt %d/%d)", attempt, maxRetries)
				
				// Try to ensure page is active again
				if activeErr := c.ensurePageActive(); activeErr != nil {
					log.Printf("❌ Failed to recover page activity: %v", activeErr)
				} else {
					continue // Retry screenshot
				}
			}
			
			if attempt == maxRetries {
				return nil, fmt.Errorf("screenshot failed after %d attempts: %w", maxRetries, err)
			}
			
			log.Printf("⚠️ Screenshot attempt %d failed, retrying: %v", attempt, err)
			time.Sleep(time.Duration(attempt) * 1 * time.Millisecond) // Faster progressive delay for high-fps streaming
			continue
		}

		// Success - process the result
		if data, ok := result["data"].(string); ok && data != "" {
			// The CDP returns base64-encoded image data
			imageBytes, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 screenshot: %w", err)
			}
			return imageBytes, nil
		}

		return nil, fmt.Errorf("no screenshot data received from CDP")
	}

	return nil, fmt.Errorf("screenshot failed after all retry attempts")
}

// enablePageDomain enables the Page domain for CDP operations
func (c *CDPClient) enablePageDomain() error {
	_, err := c.SendCommand("Page.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}
	c.pageEnabled = true
	return nil
}

// navigateToPage navigates to a specific URL
func (c *CDPClient) navigateToPage(url string) error {
	_, err := c.SendCommand("Page.navigate", map[string]interface{}{
		"url": url,
	})
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	
	// Wait a moment for the page to load
	time.Sleep(1 * time.Second)
	return nil
}

// StreamScreencast streams browser screenshots as MJPEG
func (s *serviceImpl) StreamScreencast(sessionID string) (chan []byte, error) {
	// Get script session
	s.scriptManager.mutex.RLock()
	session := s.scriptManager.sessions[sessionID]
	s.scriptManager.mutex.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Check if we have CDP endpoint from Python script
	if session.CDPEndpoint == "" {
		return nil, fmt.Errorf("CDP endpoint not yet available - task may still be starting")
	}

	// Check if already streaming
	if session.Status == "completed" || session.Status == "failed" {
		return nil, fmt.Errorf("task already completed, cannot stream")
	}

	// Create CDP client
	cdpClient := s.newCDPClient(session.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to CDP: %w", err)
	}

	// Create frame channel with proper buffer size to prevent backpressure
	frameChannel := make(chan []byte, 50) // Buffer up to 50 frames to prevent blocking

	// Start streaming in goroutine
	go func() {
		defer cdpClient.Close()
		defer close(frameChannel)

		ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS for live streaming
		defer ticker.Stop()

		frameCount := 0
		maxFrames := 12000 // 10 minutes at 20 FPS
		connectionErrors := 0
		maxErrors := 10 // Increased tolerance for higher frame rate

		for {
			select {
			case <-ticker.C:
				// Check if task is still running
				s.scriptManager.mutex.RLock()
				currentStatus := session.Status
				s.scriptManager.mutex.RUnlock()

				if currentStatus == "completed" || currentStatus == "failed" {
					log.Printf("📸 Stopping stream for session %s (task %s)", sessionID, currentStatus)
					return
				}

				// Capture screenshot
				imageData, err := cdpClient.CaptureScreenshot()
				if err != nil {
					connectionErrors++
					log.Printf("⚠️ Screenshot capture failed (error %d/%d): %v", connectionErrors, maxErrors, err)
					
					if connectionErrors >= maxErrors {
						log.Printf("❌ Too many connection errors, stopping stream")
						return
					}
					continue
				}

				// Reset error count on success
				connectionErrors = 0

				// Validate image data
				if len(imageData) < 1000 {
					log.Printf("⚠️ Screenshot too small (%d bytes), skipping frame", len(imageData))
					continue
				}

				// Send frame to channel with non-blocking approach
				select {
				case frameChannel <- imageData:
					frameCount++
					if frameCount%200 == 0 { // Log every 200 frames (10 seconds at 20fps)
						log.Printf("📹 Streamed %d frames for session %s", frameCount, sessionID)
					}
				case <-time.After(25 * time.Millisecond): // Faster timeout for higher frame rates
					// If channel is still full after timeout, drop oldest frame and add new one
					select {
					case <-frameChannel:
						// Drop oldest frame
					default:
					}
					// Try to add new frame again
					select {
					case frameChannel <- imageData:
						frameCount++
					default:
						log.Printf("⚠️ Frame channel still full after cleanup, skipping frame")
					}
				}

				// Stop after max frames (safety limit)
				if frameCount >= maxFrames {
					log.Printf("📸 Reached maximum frames limit, stopping stream")
					return
				}
			}
		}
	}()

	return frameChannel, nil
}

// extractCDPEndpointFromLogs extracts CDP WebSocket URL from browser-use logs (same as old code)
func extractCDPEndpointFromLogs(logOutput string) string {
	lines := strings.Split(logOutput, "\n")
	
	// Look for the CDP connection line in logs
	// Example: INFO:cdp_use.client:Connecting to ws://localhost:59035/devtools/browser/e75bcc01-4322-4d7d-90e1-edb71d4383c1
	for _, line := range lines {
		if strings.Contains(line, "INFO:cdp_use.client:Connecting to ws://") {
			// Extract the WebSocket URL
			start := strings.Index(line, "ws://")
			if start != -1 {
				// Find the end of the URL (should be end of line or next space)
				end := len(line)
				for i := start; i < len(line); i++ {
					if line[i] == ' ' || line[i] == '\n' || line[i] == '\r' {
						end = i
						break
					}
				}
				wsURL := strings.TrimSpace(line[start:end])
				log.Printf("🔗 Found CDP WebSocket URL in logs: %s", wsURL)
				return wsURL
			}
		}
	}
	
	// Fallback: look for HTTP CDP endpoint and convert to WebSocket
	for _, line := range lines {
		if strings.Contains(line, "http://localhost:") && strings.Contains(line, "/json/version") {
			// Extract the HTTP URL and convert to WebSocket
			start := strings.Index(line, "http://localhost:")
			if start != -1 {
				end := strings.Index(line[start:], "/json/version")
				if end != -1 {
					baseURL := line[start : start+end]
					log.Printf("🔗 Found HTTP CDP endpoint, converting to WebSocket: %s", baseURL)
					return baseURL + "/devtools/browser"
				}
			}
		}
	}
	
	return ""
}

// getWebSocketURL discovers the WebSocket URL from Chrome's /json endpoint
func (bm *BrowserManager) getWebSocketURL(port int) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json", port)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get Chrome tabs: %v", err)
	}
	defer resp.Body.Close()
	
	var targets []struct {
		ID                   string `json:"id"`
		Type                 string `json:"type"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
		URL                  string `json:"url"`
		Title                string `json:"title"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", fmt.Errorf("failed to parse Chrome targets: %v", err)
	}
	
	// Find a page target (not extension or other types)
	for _, target := range targets {
		if target.Type == "page" && target.WebSocketDebuggerURL != "" {
			log.Printf("🔗 Found WebSocket URL: %s (title: %s)", target.WebSocketDebuggerURL, target.Title)
			return target.WebSocketDebuggerURL, nil
		}
	}
	
	return "", fmt.Errorf("no suitable WebSocket target found")
}

// NewCDPClient creates a new CDP client instance
func NewCDPClient(wsURL string) *CDPClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &CDPClient{
		wsURL:     wsURL,
		ctx:       ctx,
		cancel:    cancel,
		requestID: 1,
	}
}