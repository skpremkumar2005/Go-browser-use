package browser

import "github.com/labstack/echo/v4"

// Routes defines the routing structure for browser operations
type Routes struct {
	handler Handler  // Use the fixed handler
}

// NewRoutes creates a new routes instance with the fixed handler
func NewRoutes(handler Handler) *Routes {
	return &Routes{
		 handler: handler,
	}
}

// Setup configures all browser-related routes
func (r *Routes) Setup(g *echo.Group) {
	r.handler.Route(g)
}

func (h *handler) Route(g *echo.Group) {
	// Browser Session Management


	//MAIN APIS

	g.POST("/execute", h.ExecuteTaskAndStream)



	
	g.POST("/sessions", h.CreateBrowserSession, CreateBrowserSessionValidation)
	g.GET("/sessions/:sessionId", h.GetBrowserSession, ValidateSessionID)
	g.GET("/sessions", h.ListBrowserSessions)

	// Script Task Management
	g.POST("/tasks", h.CreateScriptTask, ScriptTaskRequestValidation)
	g.GET("/tasks/:taskId", h.GetScriptTask, ValidateTaskID)
	g.GET("/tasks", h.ListScriptTasks)

	// Browser Actions
	g.POST("/sessions/:sessionId/actions", h.ExecuteBrowserAction, ValidateSessionID, BrowserActionValidation)
	g.GET("/sessions/:sessionId/screenshot", h.CaptureScreenshot, ValidateSessionID)

	// System Status
	g.GET("/status", h.GetSystemStatus)
	
	// Get active session - helps frontend find current session
	g.GET("/active-session", h.GetActiveSession)

	// WebSocket and Streaming
	g.GET("/ws/:sessionId", h.HandleWebSocketFixed, ValidateSessionID)
	g.GET("/websocket-stream/:sessionId", h.HandleWebSocketFixed, ValidateSessionID) // New WebSocket streaming endpoint with fixed handling
	g.GET("/stream/:sessionId", h.StartStreamingSession, ValidateSessionID)
	g.DELETE("/stream/:sessionId", h.StopStreamingSession, ValidateSessionID)
	g.POST("/stream/:sessionId/reset", h.ResetStreamingSession, ValidateSessionID)
       
	// MJPEG Streaming for live.html (FIXED)
	g.GET("/stream-screencast/:sessionId", h.StreamScreencast, ValidateSessionID)
       
	// Execute endpoint (same as old code) - creates task and immediately streams


	// Recovery endpoint - helps frontend recover from lost sessions
	// g.POST("/recover/:sessionId", h.RecoverSession, ValidateSessionID)

	// Live Automation Page - serves live.html template (fixed path to match URL generation)
	g.GET("/live-automation/:sessionId", h.LiveAutomationPage, ValidateSessionID)
}