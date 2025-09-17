package browser

import (
	"go-webrtc/cmd/script-server/libs/shared/helpers"

	"github.com/labstack/echo/v4"
)

// Validation functions for browser controller

// CreateBrowserSessionValidation validates create browser session request
func CreateBrowserSessionValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var payload CreateBrowserSessionDto
		if err := c.Bind(&payload); err != nil {
			return helpers.RespBadRequest(c, "Invalid request payload")
		}

		// Set default viewport if not provided
		if payload.Viewport.Width == 0 {
			payload.Viewport.Width = 1920
		}
		if payload.Viewport.Height == 0 {
			payload.Viewport.Height = 1080
		}

		if err := c.Validate(&payload); err != nil {
			return helpers.RespBadRequest(c, err.Error())
		}

		c.Set("createBrowserSession", payload)
		return next(c)
	}
}

// ScriptTaskRequestValidation validates script task request
func ScriptTaskRequestValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var payload ScriptTaskRequestDto
		if err := c.Bind(&payload); err != nil {
			return helpers.RespBadRequest(c, "Invalid request payload")
		}

		// Set default max steps if not provided
		if payload.MaxSteps == 0 {
			payload.MaxSteps = 10
		}

		if err := c.Validate(&payload); err != nil {
			return helpers.RespBadRequest(c, err.Error())
		}

		c.Set("scriptTaskRequest", payload)
		return next(c)
	}
}

// BrowserActionValidation validates browser action request
func BrowserActionValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var payload BrowserActionDto
		if err := c.Bind(&payload); err != nil {
			return helpers.RespBadRequest(c, "Invalid request payload")
		}

		if err := c.Validate(&payload); err != nil {
			return helpers.RespBadRequest(c, err.Error())
		}

		// Validate action-specific data based on action type
		switch payload.Action {
		case "click":
			var clickData ClickActionDto
			if err := helpers.JsonMarshaller(payload.Data, &clickData); err != nil {
				return helpers.RespBadRequest(c, "Invalid click action data")
			}
			c.Set("actionData", clickData)
		case "type":
			var typeData TypeActionDto
			if err := helpers.JsonMarshaller(payload.Data, &typeData); err != nil {
				return helpers.RespBadRequest(c, "Invalid type action data")
			}
			c.Set("actionData", typeData)
		case "key":
			var keyData KeyActionDto
			if err := helpers.JsonMarshaller(payload.Data, &keyData); err != nil {
				return helpers.RespBadRequest(c, "Invalid key action data")
			}
			c.Set("actionData", keyData)
		case "scroll":
			var scrollData ScrollActionDto
			if err := helpers.JsonMarshaller(payload.Data, &scrollData); err != nil {
				return helpers.RespBadRequest(c, "Invalid scroll action data")
			}
			c.Set("actionData", scrollData)
		case "navigate":
			var navigateData NavigateActionDto
			if err := helpers.JsonMarshaller(payload.Data, &navigateData); err != nil {
				return helpers.RespBadRequest(c, "Invalid navigate action data")
			}
			c.Set("actionData", navigateData)
		default:
			return helpers.RespBadRequest(c, "Unsupported action type")
		}

		c.Set("browserAction", payload)
		return next(c)
	}
}

// ValidateSessionID validates session ID parameter
func ValidateSessionID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sessionID := c.Param("sessionId")
		if sessionID == "" {
			return helpers.RespBadRequest(c, "Session ID is required")
		}
		c.Set("sessionId", sessionID)
		return next(c)
	}
}

// ValidateTaskID validates task ID parameter
func ValidateTaskID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID := c.Param("taskId")
		if taskID == "" {
			return helpers.RespBadRequest(c, "Task ID is required")
		}
		c.Set("taskId", taskID)
		return next(c)
	}
}