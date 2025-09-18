package stealth

import (
	"fmt"
	"strconv"
	"strings"
)

// GetUltraStealthBrowserArgs returns the most advanced browser arguments for maximum stealth
func GetUltraStealthBrowserArgs(sessionID string, port int, viewport Viewport) []string {
	userAgent := DefaultStealthConfig.GetRandomUserAgent()

	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),

		// Essential headless and sandboxing
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-gpu",

		// Window and viewport
		"--window-size=" + strconv.Itoa(viewport.Width) + "," + strconv.Itoa(viewport.Height),
		"--window-position=0,0",

		// CRITICAL: Primary automation detection bypass
		"--disable-blink-features=AutomationControlled",
		"--exclude-switches=enable-automation",
		"--disable-automation",
		"--disable-extensions-file-access-check",
		"--disable-extensions-http-throttling",

		// User agent and locale
		"--user-agent=" + userAgent,
		"--lang=en-US,en",
		"--accept-lang=en-US,en;q=0.9,en-GB;q=0.8,fr;q=0.7",

		// Advanced feature disabling for stealth
		"--disable-features=AutomationControlled,VizDisplayCompositor,ScriptStreaming,TranslateUI,BlinkGenPropertyTrees,AudioServiceOutOfProcess,MediaRouter,OptimizationHints,Translate,VizServiceSharingFromGPUProcess",

		// Privacy and security
		"--disable-web-security",
		"--allow-running-insecure-content",
		"--ignore-certificate-errors",
		"--ignore-ssl-errors",
		"--ignore-certificate-errors-spki-list",
		"--ignore-certificate-errors-spki-list-experimental",
		"--allow-cross-origin-auth-prompt",

		// First run and setup prevention
		"--no-first-run",
		"--no-service-autorun",
		"--password-store=basic",
		"--use-mock-keychain",
		"--disable-component-extensions-with-background-pages",
		"--disable-default-apps",
		"--disable-extensions",

		// Sync and Google services
		"--disable-sync",
		"--disable-background-networking",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--disable-client-side-phishing-detection",
		"--disable-hang-monitor",
		"--disable-prompt-on-repost",
		"--disable-domain-reliability",
		"--disable-component-update",

		// IPC and process management
		"--disable-ipc-flooding-protection",
		"--disable-field-trial-config",
		"--disable-back-forward-cache",
		"--disable-backing-store-limit",
		"--no-zygote",

		// Memory and performance
		"--memory-pressure-off",
		"--max_old_space_size=4096",
		"--disable-accelerated-2d-canvas",
		"--disable-accelerated-jpeg-decoding",
		"--disable-accelerated-mjpeg-decode",
		"--disable-accelerated-video-decode",
		"--disable-accelerated-video-encode",

		// Media and devices
		"--disable-audio-output",
		"--mute-audio",
		"--disable-background-media-suspend",
		"--disable-media-session-api",
		"--autoplay-policy=no-user-gesture-required",

		// Logging and debugging
		"--disable-logging",
		"--disable-dev-tools",
		"--log-level=3",
		"--silent",

		// Additional stealth measures
		"--disable-permissions-api",
		"--disable-presentation-api",
		"--disable-push-messaging",
		"--disable-speech-api",
		"--hide-scrollbars",
		"--disable-wake-on-wifi",
		"--disable-background-mode",
		"--force-color-profile=srgb",

		// Network and connectivity
		"--aggressive-cache-discard",
		"--disable-background-downloads",
		"--disable-add-to-shelf",
		"--disable-datasaver-prompt",
		"--disable-desktop-notifications",
		"--disable-device-discovery-notifications",

		// Advanced fingerprint resistance
		"--fingerprinting-canvas-measuretext-noise",
		"--fingerprinting-canvas-image-data-noise",
		"--fingerprinting-client-rects-noise",
		"--disable-reading-from-canvas",
		"--canvas-msaa-sample-count=0",

		// WebRTC stealth
		"--disable-webrtc-multiple-routes",
		"--disable-webrtc-hw-decoding",
		"--disable-webrtc-hw-encoding",
		"--enforce-webrtc-ip-permission-check",
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",

		// Additional Chrome policies
		"--simulate-outdated-no-au='Tue, 31 Dec 2099 23:59:59 GMT'",
		"--disable-features=WebOTP,WebPayments,WebUSB,VoiceInteraction",

		// Ultimate stealth flags
		"--disable-site-isolation-trials",
		"--disable-features=TranslateUI,BlinkGenPropertyTrees,ImprovedCookieControls,SameSiteByDefaultCookies,CookiesWithoutSameSiteMustBeSecure",
		"--aggressive",
		"--no-experiments",
		"--no-report-upload",
		"--disable-breakpad",
		"--disable-crash-reporter",
		"--crash-dumps-dir=/tmp",
		"--disable-logging-redirect",
	}

	return args
}

// GetAdvancedStealthHeaders returns realistic HTTP headers
func GetAdvancedStealthHeaders() map[string]string {
	userAgent := DefaultStealthConfig.GetRandomUserAgent()

	// Extract Chrome version from user agent for realistic headers
	chromeVersion := "120"
	if strings.Contains(userAgent, "Chrome/") {
		parts := strings.Split(userAgent, "Chrome/")
		if len(parts) > 1 {
			version := strings.Split(parts[1], " ")[0]
			if len(version) > 0 {
				chromeVersion = strings.Split(version, ".")[0]
			}
		}
	}

	return map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Encoding":           "gzip, deflate, br",
		"Accept-Language":           "en-US,en;q=0.9,en-GB;q=0.8,fr;q=0.7",
		"Cache-Control":             "max-age=0",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Sec-Ch-Ua":                 fmt.Sprintf(`"Not_A Brand";v="8", "Chromium";v="%s", "Google Chrome";v="%s"`, chromeVersion, chromeVersion),
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        `"Windows"`,
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                userAgent,
	}
}

// GetStealthNetworkConditions returns realistic network conditions
func GetStealthNetworkConditions() map[string]interface{} {
	return map[string]interface{}{
		"offline":            false,
		"downloadThroughput": 100 * 1024 * 1024 / 8, // 100 Mbps in bytes/s
		"uploadThroughput":   10 * 1024 * 1024 / 8,  // 10 Mbps in bytes/s
		"latency":            20,                    // 20ms latency
	}
}
