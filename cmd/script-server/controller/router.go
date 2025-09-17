package controller

import (
	"go-webrtc/cmd/script-server/config"
	browser "go-webrtc/cmd/script-server/controller/browser"

	"github.com/labstack/echo/v4"
)

// InitRoutes initializes all API routes
func InitRoutes(api *echo.Group, service browser.Service, cfg *config.Config) {
	// Mount browser routes under /browser using the handler
	browser.NewHandler(service, cfg).Route(api.Group("/browser"))
}