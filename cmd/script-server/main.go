package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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

var (
	cfg                  Config
	scriptSessionManager = &ScriptSessionManager{
		sessions: make(map[string]*ScriptSession),
	}
	browserManager = &BrowserManager{
		sessions:              make(map[string]*BrowserSession),
		nextPort:              9222,
		maxConcurrentSessions: getEnvInt("MAX_CONCURRENT_SESSIONS", 10), // Increased to 10 concurrent sessions
		activeSessions:        0,
	}
	wsManager = &WebSocketManager{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan []byte),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Check allowed origins from environment
			allowedOrigins := getEnv("ALLOWED_ORIGINS", "http://localhost:3000,https://localhost:3000,http://127.0.0.1:3000,https://127.0.0.1:3000")
			origin := r.Header.Get("Origin")
			host := r.Host

			// Allow empty origin (direct API calls)
			if origin == "" {
				return true
			}

			// Check against allowed origins list
			allowedList := strings.Split(allowedOrigins, ",")
			for _, allowed := range allowedList {
				if origin == strings.TrimSpace(allowed) {
					return true
				}
			}

			// Fallback to same origin check
			return strings.Contains(origin, host)
		},
	}
)

func main() {
	// Load configuration
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("🚀 Starting Script-Based Browser Automation Platform")
	log.Printf("📋 Configuration loaded")
	log.Printf("📁 Script directory: %s", cfg.Script.ScriptDir)

	// Clean up old Chrome processes (disabled to prevent CDP issues)
	// cleanupOldChromeProcesses()

	// Start WebSocket manager
	go wsManager.run()

	// Start health monitoring
	browserManager.startHealthMonitoring()
	log.Printf("🏥 Started browser health monitoring")

	// Setup routes
	router := mux.NewRouter()
	setupRoutes(router)

	// Start server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Reset any stuck streaming flags on startup
	resetAllStreamingFlags()

	log.Printf("📡 Server starting on http://%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Fatal(server.ListenAndServe())
}

func loadConfig() error {
	// Load .env file if it exists (try multiple possible locations)
	possiblePaths := []string{".env", "../.env", "../../.env"}
	var loaded bool
	for _, envPath := range possiblePaths {
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("✅ Loaded .env file from %s", envPath)
			loaded = true
			break
		}
	}
	if !loaded {
		log.Printf("⚠️ No .env file found in any of the expected locations, using defaults")
	}

	cfg = Config{
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
			CommonPorts:  parseIntSlice(getEnv("CDP_PORTS", "9222,9223,9224,9225,9226,9227,9228,9229,9230")),
			Timeout:      time.Duration(getEnvInt("CDP_TIMEOUT", 10)) * time.Second,
			PingInterval: time.Duration(getEnvInt("CDP_PING_INTERVAL", 20)) * time.Second,
			PingTimeout:  time.Duration(getEnvInt("CDP_PING_TIMEOUT", 10)) * time.Second,
			CloseTimeout: time.Duration(getEnvInt("CDP_CLOSE_TIMEOUT", 10)) * time.Second,
		},
	}

	return nil
}

func setupRoutes(router *mux.Router) {
	// Add CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Task management
	api.HandleFunc("/browser_use/execute", createScriptTaskHandler).Methods("POST")
	api.HandleFunc("/task/{sessionId}", getScriptTaskHandler).Methods("GET")
	api.HandleFunc("/task/{sessionId}/cancel", cancelScriptTaskHandler).Methods("POST")
	api.HandleFunc("/sessions", listScriptSessionsHandler).Methods("GET")

	// Action execution - User interactions for automation URL

	// Streaming
	api.HandleFunc("/stream-screencast/{sessionId}", streamScreencastHandler).Methods("GET")

	// Live automation page - starts browser-use automation and shows live stream
	api.HandleFunc("/live-automation/{sessionId}", liveAutomationHandler).Methods("GET")

	// Debug endpoint to reset streaming flag
	api.HandleFunc("/reset-stream/{sessionId}", resetStreamHandler).Methods("POST")

	// System status endpoint
	api.HandleFunc("/status", statusHandler).Methods("GET")

	// WebSocket endpoint
	api.HandleFunc("/ws", handleWebSocket)

	// WebSocket endpoint for streaming
	api.HandleFunc("/stream-ws/{sessionId}", handleStreamWebSocket)

	// Serve static files - try multiple paths for frontend
	frontendPaths := []string{"./frontend/", "../frontend/", "../../frontend/"}
	var frontendDir string
	for _, path := range frontendPaths {
		if _, err := os.Stat(path + "index.html"); err == nil {
			frontendDir = path
			break
		}
	}

	if frontendDir != "" {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir(frontendDir)))
	} else {
		// Fallback: serve a simple message if frontend not found
		router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Frontend not found. Please check frontend directory.", http.StatusNotFound)
		})
	}
}

// Browser Management Functions

// getRandomUserAgent returns a realistic user agent string to avoid detection
func getRandomUserAgent() string {
	// Updated user agents with latest versions and more variety
	userAgents := []string{
		// Windows Chrome (most common)
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
		// Windows Edge
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
		// macOS Chrome
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		// macOS Safari
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		// Linux Chrome (less common but realistic)
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	// Use current time as seed for randomization
	return userAgents[time.Now().UnixNano()%int64(len(userAgents))]
}

// getStealthJavaScript returns JavaScript code to inject for enhanced stealth
func getStealthJavaScript() string {
	return `
		// CRITICAL: Override webdriver property (Google's primary detection)
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
			configurable: true
		});

		// Remove automation indicators
		delete window.cdc_adoQpoasnfa76pfcZLmcfl_Array;
		delete window.cdc_adoQpoasnfa76pfcZLmcfl_Promise;
		delete window.cdc_adoQpoasnfa76pfcZLmcfl_Symbol;
		delete window.cdc_adoQpoasnfa76pfcZLmcfl_JSON;
		delete window.cdc_adoQpoasnfa76pfcZLmcfl_Object;
		delete window.cdc_adoQpoasnfa76pfcZLmcfl_Proxy;

		// Override plugins array with realistic plugins
		const mockPlugins = [
			{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
			{ name: 'Chromium PDF Plugin', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: 'Portable Document Format' },
			{ name: 'Microsoft Edge PDF Plugin', filename: 'pdf', description: 'pdf' },
			{ name: 'WebKit built-in PDF', filename: 'internal-pdf-viewer', description: 'pdf' }
		];
		Object.defineProperty(navigator, 'plugins', {
			get: () => mockPlugins,
			configurable: true
		});

		// Override languages with realistic values
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en', 'en-GB'],
			configurable: true
		});

		// Mock chrome runtime object
		if (!window.chrome) {
			window.chrome = {};
		}
		if (!window.chrome.runtime) {
			window.chrome.runtime = {
				onConnect: null,
				onMessage: null,
				connect: function() { return { postMessage: function() {}, onMessage: { addListener: function() {} } }; },
				sendMessage: function() {},
				id: undefined
			};
		}

		// Override permissions API
		if (navigator.permissions && navigator.permissions.query) {
			const originalQuery = navigator.permissions.query.bind(navigator.permissions);
			navigator.permissions.query = (parameters) => {
				return parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters);
			};
		}

		// Randomize screen properties slightly
		const originalScreen = { ...screen };
		Object.defineProperty(screen, 'availHeight', {
			get: () => originalScreen.availHeight + Math.floor(Math.random() * 3 - 1),
			configurable: true
		});
		Object.defineProperty(screen, 'availWidth', {
			get: () => originalScreen.availWidth + Math.floor(Math.random() * 3 - 1),
			configurable: true
		});

		// Override canvas fingerprinting with subtle noise
		const originalGetContext = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function(type, ...args) {
			const context = originalGetContext.apply(this, [type, ...args]);
			
			if (type === '2d') {
				const originalGetImageData = context.getImageData;
				context.getImageData = function(...args) {
					const imageData = originalGetImageData.apply(this, args);
					// Add minimal noise to prevent fingerprinting
					for (let i = 0; i < imageData.data.length; i += Math.floor(Math.random() * 10) + 1) {
						if (Math.random() < 0.001) {
							imageData.data[i] = imageData.data[i] ^ (Math.random() < 0.5 ? 1 : 0);
						}
					}
					return imageData;
				};
			}
			
			return context;
		};

		// Override WebGL fingerprinting
		const originalGetParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(parameter) {
			if (parameter === 37445) { // UNMASKED_VENDOR_WEBGL
				return 'Intel Inc.';
			}
			if (parameter === 37446) { // UNMASKED_RENDERER_WEBGL
				return 'Intel Iris OpenGL Engine';
			}
			return originalGetParameter.apply(this, arguments);
		};

		// Mock battery API
		if ('getBattery' in navigator) {
			navigator.getBattery = () => Promise.resolve({
				charging: true,
				chargingTime: 0,
				dischargingTime: Infinity,
				level: 1
			});
		}

		// Override connection API
		if ('connection' in navigator) {
			Object.defineProperty(navigator, 'connection', {
				get: () => ({
					downlink: 10,
					effectiveType: '4g',
					rtt: 50,
					saveData: false
				}),
				configurable: true
			});
		}

		// Override media devices
		if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
			navigator.mediaDevices.enumerateDevices = () => Promise.resolve([
				{ deviceId: 'default', groupId: 'group1', kind: 'audioinput', label: 'Default - Microphone' },
				{ deviceId: 'default', groupId: 'group2', kind: 'audiooutput', label: 'Default - Speaker' },
				{ deviceId: 'default', groupId: 'group3', kind: 'videoinput', label: 'Default - Camera' }
			]);
		}

		// Add realistic timing variations
		const originalNow = performance.now;
		performance.now = function() {
			return originalNow.apply(this) + Math.random() * 0.1;
		};

		// Override Date.getTimezoneOffset
		const originalGetTimezoneOffset = Date.prototype.getTimezoneOffset;
		Date.prototype.getTimezoneOffset = function() {
			return 300; // EST timezone offset
		};

		// Add mouse movement simulation
		let mouseX = Math.random() * window.innerWidth;
		let mouseY = Math.random() * window.innerHeight;
		
		setInterval(() => {
			mouseX += (Math.random() - 0.5) * 2;
			mouseY += (Math.random() - 0.5) * 2;
			mouseX = Math.max(0, Math.min(window.innerWidth, mouseX));
			mouseY = Math.max(0, Math.min(window.innerHeight, mouseY));
		}, 100 + Math.random() * 200);

		console.log('🥷 Advanced stealth mode activated');
	`
}

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

	// Atomic port assignment to prevent race conditions
	bm.portMutex.Lock()
	port := bm.nextPort
	for {
		if !bm.isPortInUse(port) {
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

	// Launch Chrome with enhanced stealth args for better Google compatibility
	// Generate a random realistic user agent for this session
	randomUserAgent := getRandomUserAgent()

	// Enhanced stealth arguments specifically designed to bypass Google bot detection
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--window-size=" + strconv.Itoa(viewport.Width) + "," + strconv.Itoa(viewport.Height),
		"--disable-gpu", // Keep for Docker/headless compatibility
		"--headless=new",

		// CRITICAL: Core anti-detection flags for Google
		"--disable-blink-features=AutomationControlled",
		"--exclude-switches=enable-automation",
		"--disable-extensions-file-access-check",
		"--disable-extensions-http-throttling",
		"--disable-automation",
		"--disable-save-password-bubble",

		// Realistic user agent and language (randomized per session)
		"--user-agent=" + randomUserAgent,
		"--lang=en-US,en",
		"--accept-lang=en-US,en;q=0.9,en-GB;q=0.8",

		// Enhanced privacy and security settings
		"--disable-web-security",
		"--allow-running-insecure-content",
		"--disable-features=VizDisplayCompositor,AutomationControlled,ScriptStreaming,TranslateUI,BlinkGenPropertyTrees",

		// Advanced stealth browser behavior
		"--no-first-run",
		"--disable-default-apps",
		"--disable-sync",
		"--disable-background-networking",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--disable-ipc-flooding-protection",
		"--disable-field-trial-config",
		"--disable-back-forward-cache",
		"--disable-backing-store-limit",

		// Memory and performance settings
		"--memory-pressure-off",
		"--max_old_space_size=2048",
		"--no-zygote",
		"--disable-accelerated-2d-canvas",
		"--disable-accelerated-jpeg-decoding",
		"--disable-accelerated-mjpeg-decode",
		"--disable-accelerated-video-decode",

		// Google-specific anti-detection
		"--disable-client-side-phishing-detection",
		"--disable-component-update",
		"--disable-hang-monitor",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-domain-reliability",
		"--disable-component-extensions-with-background-pages",
		"--disable-breakpad",
		"--disable-crash-reporter",
		"--disable-extensions",
		"--disable-features=TranslateUI",
		"--disable-ipc-flooding-protection",

		// Credential and keychain settings
		"--password-store=basic",
		"--use-mock-keychain",
		"--disable-password-generation",
		"--disable-password-manager-reauthentication",

		// Logging and metrics (completely disable)
		"--disable-logging",
		"--disable-dev-tools",
		"--silent",
		"--log-level=3",
		"--disable-gpu-sandbox",
		"--metrics-recording-only",
		"--no-default-browser-check",
		"--no-pings",
		"--no-report-upload",

		// Platform and sandbox settings
		"--disable-setuid-sandbox",
		"--ignore-certificate-errors",
		"--ignore-ssl-errors",
		"--ignore-certificate-errors-spki-list",
		"--ignore-certificate-errors-policy-installed",
		"--allow-running-insecure-content",

		// Additional stealth flags for modern detection
		"--disable-site-isolation-trials",
		"--disable-features=VizDisplayCompositor",
		"--run-all-compositor-stages-before-draw",
		"--disable-threaded-animation",
		"--disable-threaded-scrolling",
		"--disable-checker-imaging",
		"--disable-new-content-rendering-timeout",
		"--disable-image-animation-resync",
		"--disable-partial-raster",
		"--disable-skia-runtime-opts",
		"--disable-system-font-check",
		"--disable-cast-streaming-hw-encoding",
		"--disable-gpu-memory-buffer-compositor-resources",
		"--disable-gpu-memory-buffer-video-frames",

		// Fingerprinting protection
		"--fingerprinting-canvas-measuretext-noise",
		"--fingerprinting-canvas-image-data-noise",
		"--fingerprinting-client-rects-noise",
		"--disable-reading-from-canvas",
		"--disable-webgl",
		"--disable-webgl2",
	}

	// Use system Chrome with better server environment detection
	chromePath := getEnv("CHROME_PATH", "")
	if chromePath == "" {
		// Try common Chrome/Chromium paths for server environments
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
			chromePath = "google-chrome" // Final fallback
			log.Printf("⚠️ Using fallback Chrome path: %s", chromePath)
		}
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Env = append(os.Environ(), "DISPLAY="+cfg.Browser.Display)

	// Log the stealth configuration being used
	log.Printf("🎭 Launching Chrome with stealth configuration")
	log.Printf("🕵️ Using random User Agent: %s", randomUserAgent)
	log.Printf("📊 Total stealth args: %d", len(args))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %v", err)
	}

	// Wait for Chrome to start and get the correct WebSocket URL
	time.Sleep(5 * time.Second) // Increased wait time

	// Get the correct WebSocket URL from Chrome with retry logic
	var websocketURL string
	maxRetries := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔍 Attempting to get CDP endpoint (attempt %d/%d)", attempt, maxRetries)

		cdpHost := getEnv("CDP_HOST", "127.0.0.1")
		resp, err := http.Get(fmt.Sprintf("http://%s:%d/json", cdpHost, port))
		if err == nil && resp.StatusCode == 200 {
			var tabs []map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&tabs) == nil && len(tabs) > 0 {
				// Look for the first page tab (not browser tab)
				for _, tab := range tabs {
					if tabType, ok := tab["type"].(string); ok && tabType == "page" {
						if wsURL, ok := tab["webSocketDebuggerUrl"].(string); ok {
							websocketURL = wsURL
							log.Printf("✅ Found page CDP endpoint: %s", websocketURL)
							break
						}
					}
				}

				// If no page tab found, use the first available tab
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

	// Final fallback - this should not be used for browser-use
	if websocketURL == "" {
		log.Printf("⚠️ Could not get CDP endpoint, using fallback")
		cdpHost := getEnv("CDP_HOST", "127.0.0.1")
		websocketURL = fmt.Sprintf("ws://%s:%d/devtools/browser", cdpHost, port)
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
		// Health monitoring
		HealthCheckInterval: 30 * time.Second,
		LastHealthCheck:     time.Now(),
		HealthStatus:        "unknown",
		ConnectionErrors:    0,
		// Recovery
		AutoRecovery:        true,
		RecoveryAttempts:    0,
		MaxRecoveryAttempts: 3,
	}

	bm.sessions[sessionID] = session

	// Apply stealth configuration via CDP
	go func() {
		// Wait a moment for Chrome to fully initialize
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
			
			// Set up page navigation listener to re-inject stealth on every page
			cdpClient.SetupPageNavigationStealth()
		}
	}()

	log.Printf("✅ Created browser session %s on port %d with enhanced stealth", sessionID, port)

	return session, nil
}

func (bm *BrowserManager) isPortInUse(port int) bool {
	// Check if port is assigned to any existing session
	for _, session := range bm.sessions {
		if session.CDPPort == port {
			return true
		}
	}

	// Also check if port is actually available on the system
	cdpHost := getEnv("CDP_HOST", "127.0.0.1")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cdpHost, port), 100*time.Millisecond)
	if err != nil {
		// Port is free, we can use it
		return false
	}
	conn.Close()
	// Port is in use by another process
	return true
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

	// Ensure streaming is stopped
	session.mutex.Lock()
	session.Streaming = false
	session.mutex.Unlock()

	// Cancel cleanup timer if running
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

// startCleanupTimer starts a timer to automatically clean up the session
func (bm *BrowserManager) startCleanupTimer(sessionID string, duration time.Duration) {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if !exists {
		bm.mutex.Unlock()
		return
	}

	// Cancel existing timer if any
	if session.CleanupTimer != nil {
		session.CleanupTimer.Stop()
	}

	// Start new cleanup timer
	session.CleanupTimer = time.AfterFunc(duration, func() {
		log.Printf("⏰ Auto-cleanup timer expired for session %s, cleaning up...", sessionID)
		if err := bm.closeSession(sessionID); err != nil {
			log.Printf("❌ Failed to auto-cleanup session %s: %v", sessionID, err)
		}
	})

	session.CleanupDuration = duration
	bm.mutex.Unlock()

	log.Printf("⏰ Started cleanup timer for session %s (duration: %v)", sessionID, duration)
}

// cancelCleanupTimer cancels the cleanup timer for a session
func (bm *BrowserManager) cancelCleanupTimer(sessionID string) {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if exists && session.CleanupTimer != nil {
		session.CleanupTimer.Stop()
		session.CleanupTimer = nil
		log.Printf("⏰ Cancelled cleanup timer for session %s", sessionID)
	}
	bm.mutex.Unlock()
}

// updateUserInteraction updates the last user interaction time and cancels cleanup timer
func (bm *BrowserManager) updateUserInteraction(sessionID string) {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if !exists {
		bm.mutex.Unlock()
		return
	}

	now := time.Now()
	session.LastUserInteraction = now

	// If task is completed and user is interacting, cancel cleanup timer
	if session.TaskCompleted {
		if session.CleanupTimer != nil {
			session.CleanupTimer.Stop()
			session.CleanupTimer = nil
		}
		log.Printf("👤 User interaction detected for session %s, cleanup timer cancelled", sessionID)
	}
	bm.mutex.Unlock()
}

// Health monitoring and recovery methods
func (bm *BrowserManager) checkSessionHealth(sessionID string) error {
	bm.mutex.RLock()
	session := bm.sessions[sessionID]
	bm.mutex.RUnlock()

	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Check if browser process is still running
	if session.BrowserProcess == nil {
		session.HealthStatus = "unhealthy"
		session.ConnectionErrors++
		session.LastConnectionError = time.Now()
		return fmt.Errorf("browser process is nil")
	}

	if session.BrowserProcess.ProcessState != nil && session.BrowserProcess.ProcessState.Exited() {
		session.HealthStatus = "unhealthy"
		session.ConnectionErrors++
		session.LastConnectionError = time.Now()
		return fmt.Errorf("browser process has exited")
	}

	// Test CDP endpoint connectivity
	cdpClient := NewCDPClient(session.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		session.HealthStatus = "unhealthy"
		session.ConnectionErrors++
		session.LastConnectionError = time.Now()
		cdpClient.Close()
		return fmt.Errorf("CDP connection failed: %v", err)
	}
	cdpClient.Close()

	// Health check passed
	session.HealthStatus = "healthy"
	session.ConnectionErrors = 0
	session.LastHealthCheck = time.Now()
	return nil
}

func (bm *BrowserManager) recoverSession(sessionID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	session := bm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if !session.AutoRecovery {
		return fmt.Errorf("auto-recovery disabled for session")
	}

	if session.RecoveryAttempts >= session.MaxRecoveryAttempts {
		return fmt.Errorf("max recovery attempts reached (%d)", session.MaxRecoveryAttempts)
	}

	log.Printf("🔄 Attempting to recover session %s (attempt %d/%d)", sessionID, session.RecoveryAttempts+1, session.MaxRecoveryAttempts)

	// Close existing browser process
	if session.BrowserProcess != nil {
		session.BrowserProcess.Process.Kill()
		session.BrowserProcess.Wait()
	}

	// Recreate browser session
	viewport := session.Viewport
	newSession, err := bm.createBrowserSession(sessionID+"_recovered", viewport)
	if err != nil {
		session.RecoveryAttempts++
		return fmt.Errorf("failed to recreate browser session: %v", err)
	}

	// Update session with new browser info
	session.BrowserProcess = newSession.BrowserProcess
	session.CDPEndpoint = newSession.CDPEndpoint
	session.CDPPort = newSession.CDPPort
	session.Status = "ready"
	session.HealthStatus = "healthy"
	session.ConnectionErrors = 0
	session.RecoveryAttempts++
	session.LastActivity = time.Now()

	log.Printf("✅ Successfully recovered session %s", sessionID)
	return nil
}

func (bm *BrowserManager) startHealthMonitoring() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				bm.mutex.RLock()
				sessions := make([]string, 0, len(bm.sessions))
				for sessionID := range bm.sessions {
					sessions = append(sessions, sessionID)
				}
				bm.mutex.RUnlock()

				for _, sessionID := range sessions {
					if err := bm.checkSessionHealth(sessionID); err != nil {
						log.Printf("⚠️ Health check failed for session %s: %v", sessionID, err)

						// Attempt recovery if enabled
						bm.mutex.RLock()
						session := bm.sessions[sessionID]
						bm.mutex.RUnlock()

						if session != nil && session.AutoRecovery {
							if err := bm.recoverSession(sessionID); err != nil {
								log.Printf("❌ Recovery failed for session %s: %v", sessionID, err)
							}
						}
					}
				}
			case <-time.After(5 * time.Minute):
				// Cleanup check every 5 minutes
				bm.cleanupStaleSessions()
			}
		}
	}()
}

// cleanupStaleSessions removes sessions that have been inactive for too long
func (bm *BrowserManager) cleanupStaleSessions() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	cutoff := time.Now().Add(-30 * time.Minute) // Remove sessions older than 30 minutes
	sessionsToCleanup := make([]string, 0)

	for sessionID, session := range bm.sessions {
		if session.LastActivity.Before(cutoff) && !session.Streaming {
			sessionsToCleanup = append(sessionsToCleanup, sessionID)
		}
	}

	// Clean up sessions outside the main loop to avoid modifying map while iterating
	for _, sessionID := range sessionsToCleanup {
		session := bm.sessions[sessionID]
		log.Printf("🧹 Cleaning up stale session: %s", sessionID)

		// Ensure streaming is stopped
		session.mutex.Lock()
		session.Streaming = false
		session.mutex.Unlock()

		// Close browser process
		if session.BrowserProcess != nil {
			session.BrowserProcess.Process.Kill()
			session.BrowserProcess.Wait()
		}

		// Remove from sessions
		delete(bm.sessions, sessionID)

		// Decrement active session count
		bm.sessionMutex.Lock()
		if bm.activeSessions > 0 {
			bm.activeSessions--
		}
		bm.sessionMutex.Unlock()
	}

	if len(sessionsToCleanup) > 0 {
		log.Printf("🧹 Cleaned up %d stale sessions (active sessions: %d/%d)",
			len(sessionsToCleanup), bm.activeSessions, bm.maxConcurrentSessions)
	}
}

// Task Management Functions

func createScriptTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req ScriptTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse := createErrorResponse("invalid-json", "Invalid JSON in request body", http.StatusBadRequest, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Validate required fields
	if req.Task == "" {
		errorResponse := createErrorResponse("missing-task", "Task field is required", http.StatusBadRequest, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	sessionID := fmt.Sprintf("script_session_%d", time.Now().UnixNano())

	// Create browser session first
	viewport := Viewport{
		Width:  cfg.Browser.ViewportWidth,
		Height: cfg.Browser.ViewportHeight,
	}

	browserSession, err := browserManager.createBrowserSession(sessionID, viewport)
	if err != nil {
		// Check if it's a concurrency limit error
		if strings.Contains(err.Error(), "maximum concurrent sessions") {
			errorResponse := createErrorResponse("concurrency-limit-reached",
				fmt.Sprintf("Server is at capacity. Please try again later. %v", err),
				http.StatusServiceUnavailable, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(errorResponse)
		} else {
			errorResponse := createErrorResponse("browser-creation-failed",
				fmt.Sprintf("Failed to create browser session: %v", err),
				http.StatusInternalServerError, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse)
		}
		return
	}

	// Create script session
	scriptSession := &ScriptSession{
		ID:          sessionID,
		Task:        req.Task,
		Status:      "queued",
		BrowserID:   browserSession.ID,
		CDPEndpoint: browserSession.CDPEndpoint,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	scriptSessionManager.mutex.Lock()
	scriptSessionManager.sessions[sessionID] = scriptSession
	scriptSessionManager.mutex.Unlock()

	log.Printf("✅ Created task session %s: %s", sessionID, req.Task)

	// Start task execution in background
	go executeScriptTask(sessionID, req.Task, req.MaxSteps)

	// Create enhanced response
	response := createEnhancedTaskResponse(scriptSession, browserSession, r)
	response.Status = "started" // Override status for immediate response
	response.Message = "Task started successfully! Browser-use automation available at automation_url"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func executeScriptTask(sessionID, task string, maxSteps int) {
	// Get session with proper locking
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		log.Printf("❌ Session not found: %s", sessionID)
		return
	}

	// Update status atomically with double-check locking
	scriptSessionManager.mutex.Lock()
	if session.Status != "pending" && session.Status != "queued" {
		// Session was already started or completed
		scriptSessionManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already in state: %s", sessionID, session.Status)
		return
	}

	// Double-check: ensure we're not already running
	if session.Status == "running" {
		scriptSessionManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already running, skipping duplicate execution", sessionID)
		return
	}

	session.Status = "running"
	session.UpdatedAt = time.Now()
	scriptSessionManager.mutex.Unlock()

	// Cancel any existing cleanup timer since task is starting
	browserManager.cancelCleanupTimer(session.BrowserID)

	log.Printf("🎯 Processing task for session %s: %s", sessionID, task)

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		log.Printf("❌ Browser session not found: %s", session.BrowserID)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	// Test CDP endpoint before starting automation (with timeout)
	log.Printf("🔍 Testing CDP endpoint before starting automation: %s", browserSession.CDPEndpoint)
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)

	// Use a channel to handle connection timeout
	connectDone := make(chan error, 1)
	go func() {
		connectDone <- cdpClient.Connect()
	}()

	select {
	case err := <-connectDone:
		if err != nil {
			log.Printf("❌ Failed to connect to CDP endpoint: %v", err)
			scriptSessionManager.mutex.Lock()
			session.Status = "failed"
			session.UpdatedAt = time.Now()
			scriptSessionManager.mutex.Unlock()
			return
		}
		cdpClient.Close() // Close test connection
		log.Printf("✅ CDP endpoint validated, browser is ready for automation")
	case <-time.After(10 * time.Second):
		log.Printf("❌ CDP connection timeout after 10 seconds")
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	// Execute Python script with CDP endpoint
	scriptDir := cfg.Script.ScriptDir
	scriptPath := filepath.Join(scriptDir, cfg.Script.TaskScript)

	log.Printf("🐍 Executing Python script: %s", scriptPath)
	log.Printf("🔌 CDP Endpoint: %s", browserSession.CDPEndpoint)
	log.Printf("🎯 Task: %s", task)
	log.Printf("📊 Max Steps: %d", maxSteps)

	args := []string{
		scriptPath,
		sessionID,
		task,
		strconv.Itoa(maxSteps),
		browserSession.CDPEndpoint, // Pass CDP endpoint to browser-use
	}

	cmd := exec.Command(cfg.Script.PythonCommand, args...)
	// Pass all environment variables from .env file to Python script
	cmd.Env = os.Environ()

	log.Printf("🚀 Starting Python script execution...")

	// Execute script and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ Script execution failed: %v", err)
		log.Printf("❌ Script output: %s", string(output))
		log.Printf("❌ CDP endpoint used: %s", browserSession.CDPEndpoint)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.Metadata["error"] = fmt.Sprintf("Script failed: %v\nOutput: %s", err, string(output))
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	// Parse JSON result from output
	var result map[string]interface{}
	outputStr := string(output)
	log.Printf("📄 Python script output: %s", outputStr)

	// Look for JSON in the output - it should be at the end
	lines := strings.Split(outputStr, "\n")
	jsonFound := false

	// First, try to find JSON by looking for lines that start with {
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			// Try to parse this line as JSON
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				jsonFound = true
				log.Printf("✅ Parsed JSON result from line %d: %+v", i, result)
				break
			}
		}
	}

	// If not found, try to find JSON that spans multiple lines
	if !jsonFound {
		// Look for the last occurrence of a complete JSON object
		jsonStart := strings.LastIndex(outputStr, "{")
		if jsonStart != -1 {
			// Find the matching closing brace
			braceCount := 0
			jsonEnd := -1
			for i := jsonStart; i < len(outputStr); i++ {
				if outputStr[i] == '{' {
					braceCount++
				} else if outputStr[i] == '}' {
					braceCount--
					if braceCount == 0 {
						jsonEnd = i + 1
						break
					}
				}
			}

			if jsonEnd > jsonStart {
				jsonStr := outputStr[jsonStart:jsonEnd]
				if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
					jsonFound = true
					log.Printf("✅ Parsed JSON result from multi-line: %+v", result)
				}
			}
		}
	}

	if !jsonFound {
		log.Printf("⚠️ No valid JSON found in output, creating default result")
		// Check if the task actually completed successfully based on the logs
		if strings.Contains(outputStr, "✅ Task completed successfully") ||
			strings.Contains(outputStr, "Task completed: True") {
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

	// Update session status and store result
	scriptSessionManager.mutex.Lock()
	success, _ := result["success"].(bool)
	if success {
		session.Status = "completed"
		// Store the result in metadata for enhanced response
		session.Metadata["result"] = result
	} else {
		session.Status = "failed"
		// Store error information
		if errorMsg, exists := result["error"]; exists {
			session.Metadata["error"] = errorMsg
		}
	}
	session.UpdatedAt = time.Now()
	scriptSessionManager.mutex.Unlock()

	// Mark browser session as task completed and start cleanup timer
	browserSession = browserManager.getSession(session.BrowserID)
	if browserSession != nil {
		browserSession.mutex.Lock()
		browserSession.TaskCompleted = true
		browserSession.LastUserInteraction = time.Now()
		browserSession.mutex.Unlock()

		// Start cleanup timer for 30 seconds after task completion
		browserManager.startCleanupTimer(session.BrowserID, 30*time.Second)
		log.Printf("⏰ Task completed for session %s, cleanup timer started (30s)", sessionID)
	}

	// Broadcast status update to WebSocket clients
	wsManager.broadcastTaskUpdate(sessionID, session.Status)

	log.Printf("✅ Task completed for session %s with status: %s", sessionID, session.Status)
}

func streamScriptScreenshotsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Validate browser session is still active
	if browserSession.CDPEndpoint == "" {
		http.Error(w, "Browser session has no CDP endpoint", http.StatusInternalServerError)
		return
	}

	// Check if already streaming to prevent multiple concurrent streams
	if browserSession.Streaming {
		log.Printf("⚠️ Stream already active for session %s, rejecting new stream request", sessionID)
		http.Error(w, "Stream already active", http.StatusConflict)
		return
	}

	// Mark as streaming
	browserSession.Streaming = true
	defer func() {
		browserSession.Streaming = false
		log.Printf("📸 Stream ended for session %s", sessionID)
	}()

	log.Printf("📹 Starting stream for session %s, browser %s, CDP: %s", sessionID, session.BrowserID, browserSession.CDPEndpoint)

	// Set up MJPEG streaming
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Check browser health before starting stream
	if err := browserManager.checkSessionHealth(session.BrowserID); err != nil {
		log.Printf("❌ Browser health check failed for session %s: %v", session.BrowserID, err)

		// Attempt recovery if enabled
		if browserSession.AutoRecovery {
			log.Printf("🔄 Attempting to recover browser session %s", session.BrowserID)
			if err := browserManager.recoverSession(session.BrowserID); err != nil {
				log.Printf("❌ Recovery failed for session %s: %v", session.BrowserID, err)
				http.Error(w, "Browser session unhealthy and recovery failed", http.StatusInternalServerError)
				return
			}

			// Get updated browser session after recovery
			browserSession = browserManager.getSession(session.BrowserID)
			if browserSession == nil {
				http.Error(w, "Browser session not found after recovery", http.StatusInternalServerError)
				return
			}

			log.Printf("✅ Browser session recovered successfully for %s", session.BrowserID)
		} else {
			http.Error(w, "Browser session unhealthy", http.StatusInternalServerError)
			return
		}
	}

	// Test CDP endpoint accessibility before creating client
	log.Printf("🔍 Testing CDP endpoint accessibility: %s", browserSession.CDPEndpoint)

	// Create persistent CDP connection with retry
	var cdpClient *CDPClient
	var connectErr error
	maxConnectRetries := 3

	for attempt := 1; attempt <= maxConnectRetries; attempt++ {
		log.Printf("🔗 Attempting CDP connection (attempt %d/%d)", attempt, maxConnectRetries)

		cdpClient = NewCDPClient(browserSession.CDPEndpoint)
		connectErr = cdpClient.Connect()

		if connectErr == nil {
			log.Printf("✅ CDP connection successful on attempt %d", attempt)
			break
		}

		log.Printf("⚠️ CDP connection failed (attempt %d): %v", attempt, connectErr)

		if attempt < maxConnectRetries {
			time.Sleep(2 * time.Second)
		}
	}

	if connectErr != nil {
		log.Printf("❌ Failed to connect to CDP after %d attempts: %v", maxConnectRetries, connectErr)
		http.Error(w, fmt.Sprintf("Failed to connect to browser: %v", connectErr), http.StatusInternalServerError)
		return
	}
	defer cdpClient.Close()

	// Validate connection by getting page info
	_, err := cdpClient.SendCommand("Page.getNavigationHistory", map[string]interface{}{})
	if err != nil {
		log.Printf("⚠️ CDP connection validation failed: %v", err)
		log.Printf("⚠️ This may cause streaming issues, but continuing...")
	} else {
		log.Printf("✅ CDP connection validated for browser %s", session.BrowserID)
	}

	frameCount := 0
	// Use slower frame rate to reduce load and improve stability
	frameRate := 200 * time.Millisecond // 5 FPS instead of 10 FPS
	ticker := time.NewTicker(frameRate)
	defer ticker.Stop()

	// Add timeout for streaming (10 minutes max for automation)
	timeout := time.After(10 * time.Minute)

	// Track when task completes
	completionTime := time.Time{}
	lastSuccessfulFrame := time.Now()
	connectionErrors := 0
	maxConnectionErrors := 3 // Reduced from 5 to 3
	consecutiveErrors := 0
	maxConsecutiveErrors := 2 // Stop after 2 consecutive errors

	for {
		select {
		case <-ticker.C:
			// Check if task is completed and track completion time
			scriptSessionManager.mutex.RLock()
			currentStatus := session.Status
			scriptSessionManager.mutex.RUnlock()

			if currentStatus == "completed" && completionTime.IsZero() {
				completionTime = time.Now()
				log.Printf("📸 Task completed, streaming final result for 10 seconds")
			}

			// Stop streaming when task fails, but wait a bit to ensure it's really failed
			if currentStatus == "failed" {
				// Wait 2 seconds to ensure task is really failed and not just parsing error
				time.Sleep(2 * time.Second)

				// Check status again to confirm it's still failed
				scriptSessionManager.mutex.RLock()
				finalStatus := session.Status
				scriptSessionManager.mutex.RUnlock()

				if finalStatus == "failed" {
					log.Printf("📸 Stopping stream for session %s (task confirmed failed)", sessionID)
					browserSession.Streaming = false // Reset flag before returning
					return
				} else {
					log.Printf("📸 Task status changed from failed to %s, continuing stream", finalStatus)
				}
			}

			// Stop streaming after 5 seconds of completion (reduced from 10)
			if currentStatus == "completed" && !completionTime.IsZero() && time.Since(completionTime) > 5*time.Second {
				log.Printf("📸 Stopping stream for session %s (task completed)", sessionID)
				browserSession.Streaming = false // Reset flag before returning
				return
			}

			// Add slight delay after recent user interaction to avoid conflicts
			if time.Since(browserSession.LastUserInteraction) < 2*time.Second {
				time.Sleep(500 * time.Millisecond) // Give Chrome time to process interaction
			}

			// Capture screenshot using persistent CDP connection with retry
			imageData, err := cdpClient.CaptureScreenshotWithRetry(3)
			if err != nil {
				connectionErrors++
				consecutiveErrors++
				log.Printf("⚠️ Failed to capture screenshot after retries (error %d/%d, consecutive %d/%d): %v",
					connectionErrors, maxConnectionErrors, consecutiveErrors, maxConsecutiveErrors, err)

				// Stop if too many consecutive errors (more aggressive)
				if consecutiveErrors >= maxConsecutiveErrors {
					log.Printf("❌ Too many consecutive errors (%d), stopping stream", consecutiveErrors)
					browserSession.Streaming = false // Reset flag before returning
					return
				}

				// Try to reconnect if connection is persistently lost (after multiple errors)
				if connectionErrors >= 3 && connectionErrors%3 == 0 {
					log.Printf("🔄 Attempting to reconnect CDP after %d errors...", connectionErrors)
					cdpClient.Close()

					// Try to reconnect
					newClient := NewCDPClient(browserSession.CDPEndpoint)
					if reconnectErr := newClient.Connect(); reconnectErr == nil {
						cdpClient = newClient
						log.Printf("✅ CDP reconnected successfully")
						consecutiveErrors = 0 // Reset consecutive errors on successful reconnect
					} else {
						log.Printf("❌ CDP reconnection failed: %v", reconnectErr)
					}
				}

				// Stop if too many total connection errors
				if connectionErrors >= maxConnectionErrors {
					log.Printf("❌ Too many total connection errors (%d), stopping stream", connectionErrors)
					browserSession.Streaming = false // Reset flag before returning
					return
				}

				// Check if we haven't had a successful frame in too long
				if time.Since(lastSuccessfulFrame) > 15*time.Second {
					log.Printf("❌ No successful frames for 15 seconds, stopping stream")
					browserSession.Streaming = false // Reset flag before returning
					return
				}

				// Skip this frame, keep last image visible
				continue
			} else {
				// Reset error counts on successful capture
				connectionErrors = 0
				consecutiveErrors = 0
				lastSuccessfulFrame = time.Now()

				// Log health status periodically
				if frameCount > 0 && frameCount%50 == 0 {
					log.Printf("📹 Stream healthy: %d frames captured successfully", frameCount)
				}
			}

			// Validate image data - skip if too small
			if len(imageData) < 1000 {
				log.Printf("⚠️ Screenshot too small (%d bytes), likely blank, skipping frame", len(imageData))
				continue
			}

			// Write MJPEG frame
			fmt.Fprintf(w, "--frame\r\n")
			fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
			fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(imageData))
			w.Write(imageData)
			fmt.Fprintf(w, "\r\n")

			frameCount++
			if frameCount%cfg.Streaming.LogInterval == 0 {
				log.Printf("📹 Streamed %d frames for session %s (browser: %s)", frameCount, sessionID, session.BrowserID)
			}

			// Small delay to reduce load and improve stability
			time.Sleep(100 * time.Millisecond)

		case <-timeout:
			log.Printf("📸 Streaming timeout for session %s", sessionID)
			browserSession.Streaming = false // Reset flag before returning
			return
		}
	}
}

// WebSocket Management

func (ws *WebSocketManager) run() {
	for {
		select {
		case client := <-ws.register:
			ws.mutex.Lock()
			ws.clients[client] = true
			ws.mutex.Unlock()
			log.Printf("📡 WebSocket client connected. Total clients: %d", len(ws.clients))

		case client := <-ws.unregister:
			ws.mutex.Lock()
			if _, ok := ws.clients[client]; ok {
				delete(ws.clients, client)
				close(client.Send)
			}
			ws.mutex.Unlock()
			log.Printf("📡 WebSocket client disconnected. Total clients: %d", len(ws.clients))

		case message := <-ws.broadcast:
			ws.mutex.RLock()
			for client := range ws.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(ws.clients, client)
				}
			}
			ws.mutex.RUnlock()
		}
	}
}

// Broadcast task status update to all connected clients
func (ws *WebSocketManager) broadcastTaskUpdate(taskId, status string) {
	message := map[string]interface{}{
		"type":      "task_update",
		"taskId":    taskId,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal task update message: %v", err)
		return
	}

	ws.broadcast <- messageBytes
	log.Printf("📡 Broadcasted task update: %s -> %s", taskId, status)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &WebSocketClient{
		Conn:   conn,
		Send:   make(chan []byte, 256),
		ID:     fmt.Sprintf("client_%d", time.Now().UnixNano()),
		Active: true,
	}

	wsManager.register <- client

	go client.writePump()
	go client.readPump()
}

func handleStreamWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Validate browser session is still active
	if browserSession.CDPEndpoint == "" {
		http.Error(w, "Browser session has no CDP endpoint", http.StatusInternalServerError)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Stream WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("📹 WebSocket stream connected for session %s", sessionID)

	// Create CDP client with retry logic
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)

	// Try to connect with retries
	var connectErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("🔗 Attempting CDP connection (attempt %d/3)", attempt)
		if connectErr = cdpClient.Connect(); connectErr == nil {
			log.Printf("✅ CDP connection successful on attempt %d", attempt)
			break
		}
		log.Printf("⚠️ CDP connection failed (attempt %d): %v", attempt, connectErr)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if connectErr != nil {
		log.Printf("❌ Failed to connect to CDP after 3 attempts: %v", connectErr)
		// Check if connection is still open before sending error
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Failed to connect to browser"}`)); err != nil {
			log.Printf("⚠️ Failed to send error message to closed WebSocket: %v", err)
		}
		return
	}
	defer cdpClient.Close()

	// Stream frames
	frameCount := 0
	ticker := time.NewTicker(500 * time.Millisecond) // 2 FPS to reduce load and prevent timeouts
	defer ticker.Stop()

	// Mark this session as streaming to prevent conflicts
	browserSession.mutex.Lock()
	if browserSession.Streaming {
		browserSession.mutex.Unlock()
		log.Printf("⚠️ Stream already active for session %s, rejecting new stream request", sessionID)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Stream already active"}`))
		return
	}
	browserSession.Streaming = true
	browserSession.mutex.Unlock()

	// Ensure streaming flag is reset when done
	defer func() {
		browserSession.mutex.Lock()
		browserSession.Streaming = false
		browserSession.mutex.Unlock()
		log.Printf("📸 Stream ended for session %s", sessionID)
	}()

	// Set up ping/pong to detect disconnections
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Channel to handle graceful shutdown
	done := make(chan struct{})

	// Start goroutine to handle ping/pong and detect disconnections
	go func() {
		defer close(done)
		for {
			select {
			case <-pingTicker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("📸 WebSocket ping failed, client disconnected: %v", err)
					return
				}
			case <-r.Context().Done():
				log.Printf("📸 WebSocket stream client disconnected for session %s", sessionID)
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			// Check if task is completed
			scriptSessionManager.mutex.RLock()
			currentStatus := session.Status
			scriptSessionManager.mutex.RUnlock()

			// Stop streaming when task completes or fails
			if currentStatus == "completed" || currentStatus == "failed" {
				log.Printf("📸 Stopping WebSocket stream for session %s (task %s)", sessionID, currentStatus)
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"task_complete","status":"`+currentStatus+`"}`)); err != nil {
					log.Printf("⚠️ Failed to send task complete message: %v", err)
				}
				return
			}

			// Capture screenshot with timeout and retry
			imageData, err := cdpClient.CaptureScreenshotWithRetry(3)
			if err != nil {
				log.Printf("⚠️ Failed to capture screenshot after retries: %v", err)
				// Try to reconnect CDP client
				if reconnectErr := cdpClient.Reconnect(); reconnectErr != nil {
					log.Printf("❌ Failed to reconnect CDP client: %v", reconnectErr)
					conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"CDP connection lost"}`))
					return
				}
				log.Printf("✅ CDP client reconnected successfully")
				continue
			}

			// Validate image data
			if len(imageData) < 1000 {
				log.Printf("⚠️ Screenshot too small (%d bytes), skipping frame", len(imageData))
				continue
			}

			// Send frame via WebSocket
			frameCount++
			if frameCount%25 == 0 { // Reduced logging frequency
				log.Printf("📹 WebSocket stream healthy: %d frames sent", frameCount)
			}

			// Convert to base64 and send
			frameData := map[string]interface{}{
				"type": "frame",
				"data": base64.StdEncoding.EncodeToString(imageData),
			}

			// Set write deadline to prevent hanging
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(frameData); err != nil {
				log.Printf("❌ Failed to send frame via WebSocket: %v", err)
				return
			}

		case <-done:
			log.Printf("📸 WebSocket stream done for session %s", sessionID)
			return
		case <-r.Context().Done():
			log.Printf("📸 Client disconnected for session %s", sessionID)
			return
		}
	}
}

func (c *WebSocketClient) readPump() {
	defer func() {
		wsManager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process WebSocket messages
		var wsMessage map[string]interface{}
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("⚠️ Failed to parse WebSocket message: %v", err)
			continue
		}

		// Handle action messages
		if messageType, ok := wsMessage["type"].(string); ok && messageType == "action" {
			c.handleActionMessage(wsMessage)
		}
	}
}

// Handle action messages from WebSocket
func (c *WebSocketClient) handleActionMessage(message map[string]interface{}) {
	taskId, ok := message["taskId"].(string)
	if !ok {
		log.Printf("⚠️ Missing taskId in WebSocket action message")
		return
	}

	action, ok := message["action"].(string)
	if !ok {
		log.Printf("⚠️ Missing action in WebSocket action message")
		return
	}

	data, ok := message["data"].(map[string]interface{})
	if !ok {
		log.Printf("⚠️ Missing data in WebSocket action message")
		return
	}

	// Get session
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[taskId]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		log.Printf("⚠️ Session not found for WebSocket action: %s", taskId)
		return
	}

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		log.Printf("⚠️ Browser session not found for WebSocket action: %s", session.BrowserID)
		return
	}

	// Execute action using Go CDP
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		log.Printf("❌ Failed to connect to CDP for WebSocket action: %v", err)
		return
	}
	defer cdpClient.Close()

	// Execute action based on type
	var err error
	
	// Update last user interaction timestamp
	browserSession.LastUserInteraction = time.Now()
	
	switch action {
	case "click":
		if x, xOk := data["x"].(float64); xOk {
			if y, yOk := data["y"].(float64); yOk {
				log.Printf("🖱️ WebSocket click: (%.0f, %.0f)", x, y)
				err = cdpClient.Click(x, y)
			}
		}
	case "type":
		if text, ok := data["text"].(string); ok {
			log.Printf("⌨️ WebSocket type: '%s'", text)
			err = cdpClient.TypeText(text)
		}
	case "key":
		if key, ok := data["key"].(string); ok {
			log.Printf("⌨️ WebSocket key: '%s'", key)
			err = cdpClient.PressKey(key)
		}
	case "scroll":
		if deltaX, xOk := data["deltaX"].(float64); xOk {
			if deltaY, yOk := data["deltaY"].(float64); yOk {
				log.Printf("📜 WebSocket scroll: deltaX=%.0f deltaY=%.0f", deltaX, deltaY)
				err = cdpClient.Scroll(deltaX, deltaY)
			}
		}
	}

	if err != nil {
		log.Printf("❌ WebSocket action failed: %v", err)
	} else {
		log.Printf("✅ WebSocket action executed successfully: %s", action)

		// Update user interaction for cleanup timer management
		browserManager.updateUserInteraction(session.BrowserID)
	}
}

func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Handler functions for other endpoints

func getScriptTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		errorResponse := createErrorResponse("session-not-found", "Session not found", http.StatusNotFound, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)

	// Create enhanced response
	response := createEnhancedTaskResponse(session, browserSession, r)

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
		if errorData, exists := session.Metadata["error"]; exists {
			if errorMsg, ok := errorData.(string); ok {
				response.Error = &ErrorInfo{
					Type:    "task-execution-failed",
					Message: errorMsg,
				}
			}
		}
	case "cancelled":
		response.Message = "Task was cancelled"
		response.Success = false
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func listScriptSessionsHandler(w http.ResponseWriter, r *http.Request) {
	scriptSessionManager.mutex.RLock()
	sessions := make([]*ScriptSession, 0, len(scriptSessionManager.sessions))
	for _, session := range scriptSessionManager.sessions {
		sessions = append(sessions, session)
	}
	scriptSessionManager.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func cancelScriptTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	scriptSessionManager.mutex.Lock()
	session := scriptSessionManager.sessions[sessionID]
	if session != nil {
		session.Status = "cancelled"
		session.UpdatedAt = time.Now()
		if session.Process != nil {
			session.Process.Process.Kill()
		}
	}
	scriptSessionManager.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// Live page handler - serves the full-page live streaming interface
// Manual live page handler removed - only browser-use automation is supported

// Live automation handler - starts browser-use automation and shows live stream
func liveAutomationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session information
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session information
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Check if automation is already running
	if session.Status == "running" {
		log.Printf("🤖 Automation already running for session: %s", sessionID)
	} else if session.Status == "queued" || session.Status == "completed" || session.Status == "failed" {
		// Validate browser session is ready before starting automation
		if browserSession.BrowserProcess == nil {
			log.Printf("❌ Browser process is nil, cannot start automation for session: %s", sessionID)
			http.Error(w, "Browser process not ready", http.StatusInternalServerError)
			return
		}

		// Check if browser process is still running
		if browserSession.BrowserProcess.ProcessState != nil && browserSession.BrowserProcess.ProcessState.Exited() {
			log.Printf("❌ Browser process has exited, cannot start automation for session: %s", sessionID)
			http.Error(w, "Browser process has exited", http.StatusInternalServerError)
			return
		}

		// Test CDP endpoint before starting automation
		log.Printf("🔍 Testing CDP endpoint before starting automation: %s", browserSession.CDPEndpoint)
		testClient := NewCDPClient(browserSession.CDPEndpoint)
		if testErr := testClient.Connect(); testErr != nil {
			log.Printf("❌ CDP endpoint not accessible, cannot start automation: %v", testErr)
			testClient.Close()
			http.Error(w, fmt.Sprintf("Browser not accessible: %v", testErr), http.StatusInternalServerError)
			return
		}
		testClient.Close()
		log.Printf("✅ CDP endpoint validated, browser is ready for automation")

		// Start browser-use automation
		log.Printf("🤖 Starting browser-use automation for session: %s", sessionID)

		// Update status to running
		scriptSessionManager.mutex.Lock()
		session.Status = "running"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()

		// Start automation in background with a small delay to ensure browser is ready
		go func() {
			time.Sleep(2 * time.Second)                    // Give browser time to fully initialize
			executeScriptTask(sessionID, session.Task, 10) // Default max steps
		}()
	}

	// Prepare template data
	templateData := struct {
		TaskID       string
		Task         string
		Status       string
		CreatedAt    string
		BrowserID    string
		IsAutomation bool
	}{
		TaskID:       session.ID,
		Task:         session.Task,
		Status:       session.Status,
		CreatedAt:    session.CreatedAt.Format("2006-01-02 15:04:05"),
		BrowserID:    session.BrowserID,
		IsAutomation: true, // Flag to indicate this is automation mode
	}

	// Read the live.html template
	templatePath := "frontend/live.html"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// Try alternative paths
		altPaths := []string{"../frontend/live.html", "../../frontend/live.html"}
		for _, path := range altPaths {
			if _, err := os.Stat(path); err == nil {
				templatePath = path
				break
			}
		}
	}

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Parse and execute template
	tmpl, err := template.New("live").Parse(string(templateContent))
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}

	log.Printf("🤖 Served live automation page for session: %s", sessionID)
}

// New screencast streaming handler (similar to unified-browser-platform)
func streamScreencastHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Validate browser session is still active
	if browserSession.CDPEndpoint == "" {
		http.Error(w, "Browser session has no CDP endpoint", http.StatusInternalServerError)
		return
	}

	// Check if already streaming to prevent multiple concurrent streams
	if browserSession.Streaming {
		log.Printf("⚠️ Stream already active for session %s, rejecting new stream request", sessionID)
		http.Error(w, "Stream already active", http.StatusConflict)
		return
	}

	// Mark as streaming
	browserSession.Streaming = true
	defer func() {
		browserSession.Streaming = false
		log.Printf("📸 Stream ended for session %s", sessionID)
	}()

	// Note: Streaming flag will be reset by defer function or manual reset endpoint

	log.Printf("📹 Starting screenshot stream for session %s, browser %s, CDP: %s", sessionID, session.BrowserID, browserSession.CDPEndpoint)

	// Set up MJPEG streaming
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create CDP client with retry logic
	var cdpClient *CDPClient
	var connectErr error
	maxConnectRetries := 3

	for attempt := 1; attempt <= maxConnectRetries; attempt++ {
		log.Printf("🔗 Attempting CDP connection (attempt %d/%d)", attempt, maxConnectRetries)

		cdpClient = NewCDPClient(browserSession.CDPEndpoint)
		connectErr = cdpClient.Connect()

		if connectErr == nil {
			log.Printf("✅ CDP connection successful on attempt %d", attempt)
			break
		}

		log.Printf("⚠️ CDP connection failed (attempt %d): %v", attempt, connectErr)
		cdpClient.Close()

		if attempt < maxConnectRetries {
			time.Sleep(2 * time.Second)
		}
	}

	if connectErr != nil {
		log.Printf("❌ Failed to connect to CDP after %d attempts: %v", maxConnectRetries, connectErr)
		http.Error(w, fmt.Sprintf("Failed to connect to browser: %v", connectErr), http.StatusInternalServerError)
		return
	}
	defer cdpClient.Close()

	// Add timeout for streaming (10 minutes max for automation)
	timeout := time.After(10 * time.Minute)
	frameCount := 0

	for {
		select {
		case <-time.After(100 * time.Millisecond): // 10 FPS
			// Check if task is completed
			scriptSessionManager.mutex.RLock()
			currentStatus := session.Status
			scriptSessionManager.mutex.RUnlock()

			// Stop streaming when task fails
			if currentStatus == "failed" {
				log.Printf("📸 Stopping stream for session %s (task failed)", sessionID)
				return
			}

			// Stop streaming when task completes
			if currentStatus == "completed" {
				log.Printf("📸 Stopping stream for session %s (task completed)", sessionID)
				return
			}

			// Capture screenshot with retry logic
			imageData, err := cdpClient.CaptureScreenshotWithRetry(3)
			if err != nil {
				log.Printf("⚠️ Failed to capture screenshot after retries: %v", err)
				// Try to reconnect CDP client
				if reconnectErr := cdpClient.Reconnect(); reconnectErr != nil {
					log.Printf("❌ Failed to reconnect CDP client: %v", reconnectErr)
					// Skip this frame, keep last image visible
					continue
				}
				log.Printf("✅ CDP client reconnected successfully")
				// Skip this frame, will try again next iteration
				continue
			}

			// Validate image data - skip if too small
			if len(imageData) < 1000 {
				log.Printf("⚠️ Screenshot too small (%d bytes), likely blank, skipping frame", len(imageData))
				continue
			}

			// Send frame
			frameCount++
			if frameCount%50 == 0 {
				log.Printf("📹 Stream healthy: %d frames captured successfully", frameCount)
			}

			// Write MJPEG frame
			w.Write([]byte("--frame\r\n"))
			w.Write([]byte("Content-Type: image/jpeg\r\n"))
			w.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(imageData))))
			w.Write(imageData)
			w.Write([]byte("\r\n"))

			// Check if client disconnected
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Note: Stream health is monitored by frontend, not backend

		case <-timeout:
			log.Printf("📸 Streaming timeout for session %s", sessionID)
			return

		case <-r.Context().Done():
			log.Printf("📸 Client disconnected for session %s", sessionID)
			// Small delay to ensure proper cleanup
			time.Sleep(100 * time.Millisecond)
			return
		}
	}
}

// Reset stream handler for debugging
func resetStreamHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := browserManager.getSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Reset streaming flag
	browserSession.Streaming = false
	log.Printf("🔄 Manually reset streaming flag for session %s", sessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Streaming flag reset",
		"sessionId": sessionID,
	})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// Get system status
	browserManager.mutex.RLock()
	activeBrowserSessions := len(browserManager.sessions)
	browserManager.mutex.RUnlock()

	scriptSessionManager.mutex.RLock()
	activeScriptSessions := len(scriptSessionManager.sessions)
	scriptSessionManager.mutex.RUnlock()

	browserManager.sessionMutex.Lock()
	concurrentSessions := browserManager.activeSessions
	maxSessions := browserManager.maxConcurrentSessions
	browserManager.sessionMutex.Unlock()

	// Get memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := map[string]interface{}{
		"status": "healthy",
		"concurrency": map[string]interface{}{
			"active_sessions":     concurrentSessions,
			"max_sessions":        maxSessions,
			"utilization_percent": float64(concurrentSessions) / float64(maxSessions) * 100,
		},
		"sessions": map[string]interface{}{
			"browser_sessions": activeBrowserSessions,
			"script_sessions":  activeScriptSessions,
		},
		"memory": map[string]interface{}{
			"alloc_mb": memStats.Alloc / 1024 / 1024,
			"sys_mb":   memStats.Sys / 1024 / 1024,
			"num_gc":   memStats.NumGC,
		},
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
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

func parseIntSlice(value string) []int {
	var result []int
	parts := strings.Split(value, ",")
	for _, part := range parts {
		if num, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, num)
		}
	}
	return result
}

// URL Generation Helper Functions
func generateBaseURL(r *http.Request) string {
	// Get the host from the request
	host := r.Host
	if host == "" {
		host = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	}

	// Determine protocol
	protocol := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s", protocol, host)
}

// generateLiveURL removed - only automation URL is supported

func generateAutomationURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/live-automation/%s", baseURL, sessionId)
}

func generateStreamingURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/stream/%s", baseURL, sessionId)
}

func generateWebSocketURL(baseURL string) string {
	// Convert http to ws, https to wss
	if strings.HasPrefix(baseURL, "https://") {
		return strings.Replace(baseURL, "https://", "wss://", 1) + "/ws"
	}
	return strings.Replace(baseURL, "http://", "ws://", 1) + "/ws"
}

// Enhanced Response Helper Functions
func createEnhancedTaskResponse(session *ScriptSession, browserSession *BrowserSession, r *http.Request) *EnhancedTaskResponse {
	baseURL := generateBaseURL(r)

	response := &EnhancedTaskResponse{
		Success:   true,
		TaskId:    session.ID,
		SessionId: session.ID,
		Status:    session.Status,
		Task:      session.Task,
		CreatedAt: session.CreatedAt,
		// LiveUrl removed - only automation URL is supported
		StreamingUrl: generateStreamingURL(baseURL, session.ID),
		WebSocketUrl: generateWebSocketURL(baseURL),
		Message:      "Task started successfully! Browser-use automation available at automation_url",
	}

	// Add automation URL for production use
	response.AutomationUrl = generateAutomationURL(baseURL, session.ID)

	// Add browser info if available
	if browserSession != nil {
		response.BrowserInfo = &BrowserInfo{
			BrowserId:   browserSession.ID,
			CDPEndpoint: browserSession.CDPEndpoint,
			Viewport:    browserSession.Viewport,
		}
	}

	// Add completion info if task is completed
	if session.Status == "completed" || session.Status == "failed" {
		completedAt := session.UpdatedAt
		response.CompletedAt = &completedAt

		duration := completedAt.Sub(session.CreatedAt).Milliseconds()
		response.Duration = &duration

		// Parse result from metadata if available
		if resultData, exists := session.Metadata["result"]; exists {
			if resultMap, ok := resultData.(map[string]interface{}); ok {
				response.Result = parseTaskResult(resultMap)
			}
		}
	}

	return response
}

func createErrorResponse(errorType, message string, code int, r *http.Request) *EnhancedTaskResponse {
	baseURL := generateBaseURL(r)

	return &EnhancedTaskResponse{
		Success:   false,
		TaskId:    "",
		SessionId: "",
		Status:    "failed",
		Task:      "",
		CreatedAt: time.Now(),
		// LiveUrl removed - only automation URL is supported
		StreamingUrl: "",
		WebSocketUrl: generateWebSocketURL(baseURL),
		Message:      "Task failed to start",
		Error: &ErrorInfo{
			Type:    errorType,
			Message: message,
			Code:    code,
		},
	}
}

func parseTaskResult(resultMap map[string]interface{}) *TaskResult {
	result := &TaskResult{
		Success:  false,
		Output:   "",
		Steps:    []TaskStep{},
		Metadata: make(map[string]interface{}),
	}

	// Parse success
	if success, ok := resultMap["success"].(bool); ok {
		result.Success = success
	}

	// Parse output
	if output, ok := resultMap["output"].(string); ok {
		result.Output = output
	} else if extractedContent, ok := resultMap["extracted_content"].(string); ok {
		result.Output = extractedContent
	}

	// Parse token usage if available
	if tokenUsage, ok := resultMap["token_usage"].(map[string]interface{}); ok {
		result.TokenUsage = parseTokenUsage(tokenUsage)
	}

	// Copy other metadata
	for key, value := range resultMap {
		if key != "success" && key != "output" && key != "extracted_content" && key != "token_usage" {
			result.Metadata[key] = value
		}
	}

	return result
}

func parseTokenUsage(tokenMap map[string]interface{}) *TokenUsage {
	usage := &TokenUsage{
		Model: "gpt-4.1", // Default model
	}

	if total, ok := tokenMap["total_tokens"].(float64); ok {
		usage.TotalTokens = int(total)
	}
	if prompt, ok := tokenMap["prompt_tokens"].(float64); ok {
		usage.PromptTokens = int(prompt)
	}
	if completion, ok := tokenMap["completion_tokens"].(float64); ok {
		usage.CompletionTokens = int(completion)
	}
	if cost, ok := tokenMap["total_cost"].(float64); ok {
		usage.TotalCost = cost
	}
	if model, ok := tokenMap["model"].(string); ok {
		usage.Model = model
	}

	return usage
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// NewCDPClient creates a new CDP client
func NewCDPClient(wsURL string) *CDPClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &CDPClient{
		wsURL:     wsURL,
		ctx:       ctx,
		cancel:    cancel,
		requestID: 1,
	}
}

// Connect establishes WebSocket connection to CDP
func (c *CDPClient) Connect() error {
	u, err := url.Parse(c.wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %v", err)
	}

	// Set connection timeout
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to CDP: %v", err)
	}

	// Set read/write deadlines
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	c.mutex.Lock()
	c.conn = conn
	c.mutex.Unlock()

	// Initialize Page domain once
	if !c.initialized {
		_, err = c.SendCommand("Page.enable", map[string]interface{}{})
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to enable Page domain: %v", err)
		}
		c.pageEnabled = true
		c.initialized = true
		log.Printf("🔗 Connected to CDP at %s (Page domain enabled)", c.wsURL)
	} else {
		log.Printf("🔗 Reconnected to CDP at %s", c.wsURL)
	}

	return nil
}

// Close closes the CDP connection
func (c *CDPClient) Close() error {
	c.cancel()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		// Set a close deadline
		c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// sendRequest sends a CDP request
func (c *CDPClient) sendRequest(request map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	// Set write deadline
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Send request
	return c.conn.WriteJSON(request)
}

// SendCommand sends a CDP command and waits for response
func (c *CDPClient) SendCommand(method string, params map[string]interface{}) (map[string]interface{}, error) {
	c.mutex.Lock()
	requestID := c.requestID
	c.requestID++
	conn := c.conn
	c.mutex.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("CDP connection not established")
	}

	request := map[string]interface{}{
		"id":     requestID,
		"method": method,
		"params": params,
	}

	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// Send request
	if err := conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send CDP request: %v", err)
	}

	// Read messages until we get the response with matching ID (with timeout)
	timeout := time.After(30 * time.Second) // Increased timeout
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("CDP command timeout after 30 seconds")
		default:
			// Set read deadline - increased for better stability during interactions
			conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // Increased from 3s to 10s

			_, message, err := conn.ReadMessage()
			if err != nil {
				return nil, fmt.Errorf("failed to read CDP response: %v", err)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(message, &response); err != nil {
				continue // Skip invalid messages
			}

			// Check if this is a response to our request
			if id, exists := response["id"]; exists {
				if responseID, ok := id.(float64); ok && int(responseID) == requestID {
					// This is our response
					if errorData, exists := response["error"]; exists {
						return nil, fmt.Errorf("CDP error: %v", errorData)
					}
					return response, nil
				}
			}
			// If not our response, continue reading
		}
	}
}

// CaptureScreenshot captures a screenshot using CDP
func (c *CDPClient) CaptureScreenshot() ([]byte, error) {
	if !c.initialized {
		return nil, fmt.Errorf("CDP client not initialized")
	}

	// Capture screenshot with high quality
	params := map[string]interface{}{
		"format":      "jpeg",
		"quality":     95,
		"fromSurface": true,
	}

	response, err := c.SendCommand("Page.captureScreenshot", params)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %v", err)
	}

	// Extract base64 data - try different response formats
	var dataStr string
	var found bool

	// Try result.data format first
	if result, exists := response["result"]; exists {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if data, exists := resultMap["data"]; exists {
				if dataStr, ok = data.(string); ok {
					found = true
				}
			}
		}
	}

	// Fallback: try direct data field
	if !found {
		if data, exists := response["data"]; exists {
			if str, ok := data.(string); ok {
				dataStr = str
				found = true
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no valid data found in screenshot response")
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot: %v", err)
	}

	// Validate screenshot size
	if len(imageData) < 1000 {
		return nil, fmt.Errorf("screenshot too small (%d bytes), likely blank", len(imageData))
	}

	return imageData, nil
}

// CaptureScreenshotWithRetry attempts to capture a screenshot with retries
func (c *CDPClient) CaptureScreenshotWithRetry(maxRetries int) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		imageData, err := c.CaptureScreenshot()
		if err == nil {
			return imageData, nil
		}

		lastErr = err
		log.Printf("⚠️ Screenshot attempt %d/%d failed: %v", attempt, maxRetries, err)

		// Wait before retry (exponential backoff)
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * 100 * time.Millisecond
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed to capture screenshot after %d attempts: %v", maxRetries, lastErr)
}

// InjectStealthScript injects stealth JavaScript to bypass bot detection
func (c *CDPClient) InjectStealthScript() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	stealthScript := getStealthJavaScript()

	// Inject the stealth script
	params := map[string]interface{}{
		"source": stealthScript,
	}

	_, err := c.SendCommand("Runtime.evaluate", params)
	if err != nil {
		return fmt.Errorf("failed to inject stealth script: %v", err)
	}

	log.Printf("🥷 Stealth JavaScript injected successfully")
	return nil
}

// SetupStealthEnvironment configures the browser environment for maximum stealth
func (c *CDPClient) SetupStealthEnvironment() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	// Enable Runtime domain for script injection
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	// Inject stealth script
	if err := c.InjectStealthScript(); err != nil {
		return err
	}

	// Set additional stealth headers (removed unused variable)

	// Set user agent override with additional client hints
	params := map[string]interface{}{
		"userAgent": getRandomUserAgent(),
		"acceptLanguage": "en-US,en;q=0.9,en-GB;q=0.8",
		"platform": "Win32",
		"userAgentMetadata": map[string]interface{}{
			"brands": []map[string]interface{}{
				{"brand": "Not_A Brand", "version": "8"},
				{"brand": "Chromium", "version": "120"},
				{"brand": "Google Chrome", "version": "120"},
			},
			"fullVersion": "120.0.0.0",
			"platform": "Windows",
			"platformVersion": "10.0.0",
			"architecture": "x86",
			"model": "",
			"mobile": false,
			"bitness": "64",
		},
	}

	_, err = c.SendCommand("Network.setUserAgentOverride", params)
	if err != nil {
		log.Printf("⚠️ Failed to set user agent override: %v", err)
	}

	// Enable Network domain for header manipulation
	_, err = c.SendCommand("Network.enable", map[string]interface{}{})
	if err != nil {
		log.Printf("⚠️ Failed to enable Network domain: %v", err)
	}

	// Set viewport to common resolution
	viewportParams := map[string]interface{}{
		"width":  1920,
		"height": 1080,
		"deviceScaleFactor": 1,
		"mobile": false,
	}

	_, err = c.SendCommand("Emulation.setDeviceMetricsOverride", viewportParams)
	if err != nil {
		log.Printf("⚠️ Failed to set viewport: %v", err)
	}

	// Set timezone to common US timezone
	timezoneParams := map[string]interface{}{
		"timezoneId": "America/New_York",
	}

	_, err = c.SendCommand("Emulation.setTimezoneOverride", timezoneParams)
	if err != nil {
		log.Printf("⚠️ Failed to set timezone: %v", err)
	}

	// Set locale override
	localeParams := map[string]interface{}{
		"locale": "en-US",
	}

	_, err = c.SendCommand("Emulation.setLocaleOverride", localeParams)
	if err != nil {
		log.Printf("⚠️ Failed to set locale: %v", err)
	}

	log.Printf("🛡️ Stealth environment configured successfully")
	return nil
}

// SetupPageNavigationStealth sets up listeners to re-inject stealth on every page navigation
func (c *CDPClient) SetupPageNavigationStealth() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	// Add script to evaluate on new document creation
	stealthScript := getStealthJavaScript()
	params := map[string]interface{}{
		"source": stealthScript,
	}

	// Add script that will run on every page load
	_, err := c.SendCommand("Page.addScriptToEvaluateOnNewDocument", params)
	if err != nil {
		return fmt.Errorf("failed to add stealth script to new documents: %v", err)
	}

	// Also inject immediately for current page
	_, err = c.SendCommand("Runtime.evaluate", params)
	if err != nil {
		log.Printf("⚠️ Failed to inject stealth script on current page: %v", err)
	}

	log.Printf("🔄 Page navigation stealth listener activated")
	return nil
}

// AddStealthToAllFrames injects stealth script into all frames
func (c *CDPClient) AddStealthToAllFrames() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	// Get all frames
	response, err := c.SendCommand("Page.getFrameTree", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to get frame tree: %v", err)
	}

	stealthScript := getStealthJavaScript()

	// Inject into main frame
	params := map[string]interface{}{
		"source": stealthScript,
	}
	_, err = c.SendCommand("Runtime.evaluate", params)
	if err != nil {
		log.Printf("⚠️ Failed to inject stealth into main frame: %v", err)
	}

	// Try to inject into child frames if they exist
	if result, ok := response["result"]; ok {
		if frameTree, ok := result.(map[string]interface{}); ok {
			if childFrames, ok := frameTree["childFrames"]; ok {
				if frames, ok := childFrames.([]interface{}); ok {
					for _, frame := range frames {
						if frameMap, ok := frame.(map[string]interface{}); ok {
							if frameInfo, ok := frameMap["frame"]; ok {
								if frameData, ok := frameInfo.(map[string]interface{}); ok {
									if frameId, ok := frameData["id"].(string); ok {
										frameParams := map[string]interface{}{
											"source":  stealthScript,
											"frameId": frameId,
										}
										_, err = c.SendCommand("Runtime.evaluate", frameParams)
										if err != nil {
											log.Printf("⚠️ Failed to inject stealth into frame %s: %v", frameId, err)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Reconnect attempts to reconnect the CDP client
func (c *CDPClient) Reconnect() error {
	// Close existing connection
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Reset initialization state
	c.initialized = false
	c.pageEnabled = false

	// Attempt to reconnect
	return c.Connect()
}

// resetAllStreamingFlags resets all streaming flags on startup to prevent conflicts
func resetAllStreamingFlags() {
	log.Printf("🧹 Resetting all streaming flags on startup...")

	// Reset all browser session streaming flags
	browserManager.mutex.Lock()
	for _, browserSession := range browserManager.sessions {
		browserSession.Streaming = false
	}
	browserManager.mutex.Unlock()

	log.Printf("✅ All streaming flags reset")
}

// cleanupOldChromeProcesses kills old Chrome processes to free up resources
func cleanupOldChromeProcesses() {
	log.Printf("🧹 Cleaning up old Chrome processes...")

	// Only kill Chrome processes that are NOT browser-use automation processes
	// Look for Chrome processes without remote-debugging-port (regular Chrome)
	cmd := exec.Command("pgrep", "-f", "chrome")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️ Could not check Chrome processes: %v", err)
		return
	}

	processes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(processes) == 0 || (len(processes) == 1 && processes[0] == "") {
		log.Printf("✅ No Chrome processes to clean up")
		return
	}

	// Count browser-use Chrome processes (with remote-debugging-port)
	cmd = exec.Command("pgrep", "-f", "remote-debugging-port")
	browserUseOutput, err := cmd.Output()
	if err == nil {
		browserUseProcesses := strings.Split(strings.TrimSpace(string(browserUseOutput)), "\n")
		browserUseCount := len(browserUseProcesses)
		if len(browserUseProcesses) == 1 && browserUseProcesses[0] == "" {
			browserUseCount = 0
		}

		totalChrome := len(processes)
		if len(processes) == 1 && processes[0] == "" {
			totalChrome = 0
		}

		regularChrome := totalChrome - browserUseCount

		log.Printf("📊 Chrome process summary:")
		log.Printf("   Total Chrome processes: %d", totalChrome)
		log.Printf("   Browser-use processes: %d", browserUseCount)
		log.Printf("   Regular Chrome processes: %d", regularChrome)

		// Only clean up if there are too many regular Chrome processes
		if regularChrome > 10 {
			log.Printf("🧹 Too many regular Chrome processes (%d), cleaning up...", regularChrome)
			// Kill Chrome processes that don't have remote-debugging-port
			cmd = exec.Command("pkill", "-f", "chrome")
			cmd.Run()
			time.Sleep(2 * time.Second)
			log.Printf("✅ Chrome cleanup completed")
		} else {
			log.Printf("✅ Chrome process count is reasonable, no cleanup needed")
		}
	}
}

// Click performs a mouse click at the specified coordinates using JavaScript
func (c *CDPClient) Click(x, y float64) error {
	// Enable Runtime domain
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	// Create a click event using JavaScript
	jsCode := fmt.Sprintf(`
		var event = new MouseEvent('click', {
			view: window,
			bubbles: true,
			cancelable: true,
			clientX: %f,
			clientY: %f
		});
		var element = document.elementFromPoint(%f, %f);
		if (element) {
			element.dispatchEvent(event);
		}
	`, x, y, x, y)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute click: %v", err)
	}

	log.Printf("🖱️ Clicked at coordinates (%.0f, %.0f) via JavaScript", x, y)
	return nil
}

// TypeText types text at the current cursor position using JavaScript
func (c *CDPClient) TypeText(text string) error {
	// Enable Runtime domain
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	// Escape text for JavaScript
	escapedText := strings.ReplaceAll(text, `\`, `\\`)
	escapedText = strings.ReplaceAll(escapedText, `"`, `\"`)
	escapedText = strings.ReplaceAll(escapedText, "\n", `\n`)
	escapedText = strings.ReplaceAll(escapedText, "\r", `\r`)

	// Type text using JavaScript
	jsCode := fmt.Sprintf(`
		var activeElement = document.activeElement;
		if (activeElement && (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA' || activeElement.contentEditable === 'true')) {
			var start = activeElement.selectionStart || 0;
			var end = activeElement.selectionEnd || 0;
			var value = activeElement.value || activeElement.textContent || '';
			var newValue = value.substring(0, start) + "%s" + value.substring(end);
			if (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA') {
				activeElement.value = newValue;
			} else {
				activeElement.textContent = newValue;
			}
			activeElement.selectionStart = activeElement.selectionEnd = start + %d;
			activeElement.dispatchEvent(new Event('input', { bubbles: true }));
			activeElement.dispatchEvent(new Event('change', { bubbles: true }));
		} else {
			// If no active element, try to find and focus the first input/textarea
			var input = document.querySelector('input, textarea, [contenteditable="true"]');
			if (input) {
				input.focus();
				input.value = input.value + "%s";
				input.dispatchEvent(new Event('input', { bubbles: true }));
			}
		}
	`, escapedText, len(text), escapedText)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute type: %v", err)
	}

	log.Printf("⌨️ Typed text: %s via JavaScript", text)
	return nil
}

// PressKey presses a specific key using JavaScript
func (c *CDPClient) PressKey(key string) error {
	// Enable Runtime domain
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	// Map common keys to their key codes
	keyCodeMap := map[string]int{
		"Enter":      13,
		"Tab":        9,
		"Escape":     27,
		"Backspace":  8,
		"Delete":     46,
		"ArrowUp":    38,
		"ArrowDown":  40,
		"ArrowLeft":  37,
		"ArrowRight": 39,
		"Space":      32,
	}

	keyCode, exists := keyCodeMap[key]
	if !exists {
		keyCode = int(key[0]) // Use ASCII value for single characters
	}

	// Create key event using JavaScript
	jsCode := fmt.Sprintf(`
		var activeElement = document.activeElement;
		var event = new KeyboardEvent('keydown', {
			key: '%s',
			code: '%s',
			keyCode: %d,
			which: %d,
			bubbles: true,
			cancelable: true
		});
		if (activeElement) {
			activeElement.dispatchEvent(event);
		} else {
			document.dispatchEvent(event);
		}
		
		// Also trigger keyup
		var eventUp = new KeyboardEvent('keyup', {
			key: '%s',
			code: '%s',
			keyCode: %d,
			which: %d,
			bubbles: true,
			cancelable: true
		});
		if (activeElement) {
			activeElement.dispatchEvent(eventUp);
		} else {
			document.dispatchEvent(eventUp);
		}
	`, key, key, keyCode, keyCode, key, key, keyCode, keyCode)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute key press: %v", err)
	}

	log.Printf("⌨️ Pressed key: %s via JavaScript", key)
	return nil
}

// Scroll scrolls the page using JavaScript
func (c *CDPClient) Scroll(deltaX, deltaY float64) error {
	// Enable Runtime domain
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	// Scroll using JavaScript
	jsCode := fmt.Sprintf(`
		window.scrollBy(%f, %f);
	`, deltaX, deltaY)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute scroll: %v", err)
	}

	log.Printf("📜 Scrolled by (%.0f, %.0f) via JavaScript", deltaX, deltaY)
	return nil
}

// Navigate navigates to a URL
func (c *CDPClient) Navigate(url string) error {
	// Enable Page domain
	_, err := c.SendCommand("Page.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Page domain: %v", err)
	}

	// Navigate to URL
	_, err = c.SendCommand("Page.navigate", map[string]interface{}{
		"url": url,
	})
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %v", url, err)
	}

	log.Printf("🌐 Navigated to: %s", url)
	return nil
}

// GetPageTitle gets the current page title
func (c *CDPClient) GetPageTitle() (string, error) {
	// Enable Page domain
	_, err := c.SendCommand("Page.enable", map[string]interface{}{})
	if err != nil {
		return "", fmt.Errorf("failed to enable Page domain: %v", err)
	}

	// Get page title
	response, err := c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": "document.title",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get page title: %v", err)
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid result response")
	}

	value, ok := result["value"].(string)
	if !ok {
		return "", fmt.Errorf("invalid title value")
	}

	return value, nil
}

// GetPageURL gets the current page URL
func (c *CDPClient) GetPageURL() (string, error) {
	// Enable Page domain
	_, err := c.SendCommand("Page.enable", map[string]interface{}{})
	if err != nil {
		return "", fmt.Errorf("failed to enable Page domain: %v", err)
	}

	// Get page URL
	response, err := c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": "window.location.href",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get page URL: %v", err)
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid result response")
	}

	value, ok := result["value"].(string)
	if !ok {
		return "", fmt.Errorf("invalid URL value")
	}

	return value, nil
}
