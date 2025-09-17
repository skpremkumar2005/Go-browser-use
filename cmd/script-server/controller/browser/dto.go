package browser

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

type ScriptTaskRequest struct {
	Task     string `json:"task"`
	MaxSteps int    `json:"maxSteps,omitempty"`
}

// Enhanced Response Structures
type EnhancedTaskResponse struct {
	Success     bool       `json:"success"`
	TaskId      string     `json:"id"`
	SessionId   string     `json:"sessionId"`
	Status      string     `json:"status"`
	Task        string     `json:"task"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Duration    *int64     `json:"duration,omitempty"`
	// LiveUrl removed - only automation URL is supported
	AutomationUrl string       `json:"live_url"`
	StreamingUrl  string       `json:"streaming_url"`
	WebSocketUrl  string       `json:"websocket_url"`
	BrowserInfo   *BrowserInfo `json:"browser_info,omitempty"`
	Result        *TaskResult  `json:"result,omitempty"`
	Message       string       `json:"message"`
	Error         *ErrorInfo   `json:"error,omitempty"`
}

type BrowserInfo struct {
	BrowserId   string   `json:"browserId"`
	CDPEndpoint string   `json:"cdpEndpoint"`
	Viewport    Viewport `json:"viewport"`
	CurrentUrl  string   `json:"currentUrl,omitempty"`
}

type TaskResult struct {
	Success    bool                   `json:"success"`
	Output     string                 `json:"output"`
	Steps      []TaskStep             `json:"steps,omitempty"`
	TokenUsage *TokenUsage            `json:"token_usage,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type TaskStep struct {
	Step      int       `json:"step"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type TokenUsage struct {
	TotalTokens      int     `json:"total_tokens"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalCost        float64 `json:"total_cost"`
	Model            string  `json:"model"`
}

type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
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

// CDPCommand represents a queued CDP command
type CDPCommand struct {
	ID       int
	Method   string
	Params   map[string]interface{}
	Response chan CDPResponse
	Priority int // 0=low, 1=normal, 2=high
}

// CDPResponse represents a CDP command response
type CDPResponse struct {
	Result map[string]interface{}
	Error  error
}

// CDPClient handles Chrome DevTools Protocol communication
type CDPClient struct {
	wsURL         string
	conn          *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	requestID     int
	mutex         sync.RWMutex
	pageEnabled   bool
	initialized   bool
	commandQueue  chan CDPCommand
	responses     map[int]chan CDPResponse
	responsesMux  sync.RWMutex
	queueWorkers  int
	maxQueueSize  int
	isProcessing  bool
}
