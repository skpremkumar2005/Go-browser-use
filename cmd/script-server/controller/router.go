package controller

import (
	"net/http"

	"github.com/gorilla/mux"

	"go-webrtc/cmd/script-server/controller/browser"
)

// SetupRoutes mounts sub-controllers under a top-level router.
// Example: calling SetupRoutes(api) will mount the browser controller at /browser
func SetupRoutes(router *mux.Router) {
    // Mount browser controller under /browser_use
    browserSub := router.PathPrefix("/browser_use").Subrouter()
    browser.SetupRoutes(browserSub)



    // Add favicon route - serve the project logo
    router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "./browser_use/docs/logo/light.svg")
    }).Methods("GET")

    // Future controllers can be mounted here, e.g.
    // authSub := router.PathPrefix("/auth").Subrouter()
    // auth.SetupRoutes(authSub)
}
