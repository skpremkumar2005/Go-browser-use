package main

import (
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go-webrtc/cmd/script-server/config"
	"go-webrtc/cmd/script-server/controller"
	"go-webrtc/cmd/script-server/controller/browser"
	"go-webrtc/cmd/script-server/libs/shared/helpers"
)

// CustomValidator wraps the go-playground validator
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the struct
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	// Load environment variables
	if err := loadEnvironment(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Echo server
	e := echo.New()

	// Setup validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Setup middleware
	setupMiddleware(e)

	// Setup routes
	api := e.Group("/api/v1/browser_use")
	service := browser.NewService(cfg)
	controller.InitRoutes(api, service, cfg)

	// Setup static file serving
	setupStaticFiles(e)

	// Start server
	address := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s", address)
	
	if err := e.Start(address); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// loadEnvironment loads environment variables from .env file
func loadEnvironment() error {
	// Try to load .env file from multiple possible locations
	envPaths := []string{
		".env",
		"../.env",
		"../../.env",
		"../../../.env",
	}
	
	var err error
	for _, path := range envPaths {
		if err = godotenv.Load(path); err == nil {
			log.Printf("Loaded environment from %s", path)
			return nil
		}
	}
	
	// If no .env file is found, that's okay - we'll use system environment variables
	return err
}

// setupMiddleware configures Echo middleware
func setupMiddleware(e *echo.Echo) {
	// Logger middleware with custom format
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${method} ${uri} ${status} ${latency_human} ${bytes_in}/${bytes_out}\n",
	}))

	// Recover middleware
	e.Use(middleware.Recover())

	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: getAllowedOrigins(),
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-Requested-With",
		},
		AllowCredentials: true,
	}))

	// Request timeout middleware - exclude WebSocket and streaming routes
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
		Skipper: func(c echo.Context) bool {
			// Skip timeout for WebSocket routes and streaming routes
			path := c.Request().URL.Path
			return strings.Contains(path, "/ws/") || 
				   strings.HasSuffix(path, "/ws") ||
				   strings.Contains(path, "/stream-screencast/") ||
				   strings.Contains(path, "/stream/")
		},
	}))

	// Body limit middleware
	e.Use(middleware.BodyLimit("10M"))

	// Secure middleware
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	// Custom error handler
	e.HTTPErrorHandler = customErrorHandler
}

// getAllowedOrigins returns the list of allowed CORS origins
func getAllowedOrigins() []string {
	allowedOrigins := helpers.GetEnv("ALLOWED_ORIGINS", "*")
	if allowedOrigins == "*" {
		return []string{"*"}
	}
	
	// Split comma-separated origins
	origins := []string{}
	originStr := helpers.GetEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")
	for _, origin := range strings.Split(originStr, ",") {
		if strings.TrimSpace(origin) != "" {
			origins = append(origins, strings.TrimSpace(origin))
		}
	}
	
	return origins
}

// customErrorHandler handles errors in a consistent format
func customErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	// Handle Echo HTTP errors
	if he, ok := err.(*echo.HTTPError); ok {
		message := "An error occurred"
		if msg, ok := he.Message.(string); ok {
			message = msg
		}
		
		switch he.Code {
		case 400:
			helpers.RespBadRequest(c, message)
		case 401:
			helpers.RespUnauthorized(c, message)
		case 404:
			helpers.RespNotFound(c, message)
		case 409:
			helpers.RespConflict(c, message)
		default:
			helpers.RespFailure(c, message, err)
		}
		return
	}

	// Handle other errors
	helpers.RespFailure(c, "Internal server error", err)
}

// setupStaticFiles configures static file serving
func setupStaticFiles(e *echo.Echo) {
	// Serve static files from multiple possible locations
	staticPaths := []string{
		"./frontend",
		"../frontend",
		"../../frontend",
		"./static",
		"../static",
		"../../static",
	}
	
	for _, path := range staticPaths {
		e.Static("/static", path)
		log.Printf("Attempting to serve static files from %s", path)
	}
	
	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return helpers.RespSuccess(c, "Server is healthy", map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		})
	})
	
	// API documentation endpoint
	e.GET("/", func(c echo.Context) error {
		return c.HTML(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Go Browser Use API</title>
				<style>
					body { 
						font-family: Arial, sans-serif; 
						max-width: 800px; 
						margin: 50px auto; 
						padding: 20px; 
						line-height: 1.6; 
					}
					h1 { color: #333; }
					.endpoint { 
						background: #f4f4f4; 
						padding: 10px; 
						margin: 10px 0; 
						border-radius: 4px; 
					}
					.method { 
						font-weight: bold; 
						color: #007ACC; 
					}
				</style>
			</head>
			<body>
				<h1>Go Browser Use API</h1>
				<p>Welcome to the Go Browser Use API. This service provides browser automation capabilities.</p>
				
				<h2>Available Endpoints</h2>
				
				<h3>Browser Sessions</h3>
				<div class="endpoint">
					<span class="method">POST</span> /api/v1/browser/sessions - Create browser session
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/sessions - List browser sessions
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/sessions/{sessionId} - Get browser session
				</div>
				<div class="endpoint">
					<span class="method">DELETE</span> /api/v1/browser/sessions/{sessionId} - Close browser session
				</div>
				
				<h3>Script Tasks</h3>
				<div class="endpoint">
					<span class="method">POST</span> /api/v1/browser/tasks - Create script task
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/tasks - List script tasks
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/tasks/{taskId} - Get script task
				</div>
				
				<h3>Browser Actions</h3>
				<div class="endpoint">
					<span class="method">POST</span> /api/v1/browser/sessions/{sessionId}/actions - Execute browser action
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/sessions/{sessionId}/screenshot - Capture screenshot
				</div>
				
				<h3>System</h3>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/status - Get system status
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /health - Health check
				</div>
				
				<h3>Live Automation</h3>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/live/automation/{sessionId} - Live automation page
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/stream/{sessionId} - Start streaming
				</div>
				<div class="endpoint">
					<span class="method">GET</span> /api/v1/browser/ws - WebSocket connection
				</div>
			</body>
			</html>
		`)
	})
}