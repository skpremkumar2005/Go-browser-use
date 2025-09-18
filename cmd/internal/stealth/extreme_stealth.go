package stealth

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// RequestInterceptionJS returns JavaScript for advanced request interception and modification
func RequestInterceptionJS() string {
	return `
		(function() {
			'use strict';
			
			// ==== ADVANCED REQUEST INTERCEPTION AND HEADER SPOOFING ====
			
			// Dynamic header generation based on current time and request
			const generateDynamicHeaders = (url) => {
				const now = Date.now();
				const entropy = Math.random() * 1000;
				
				return {
					'sec-ch-ua-arch': '"x86"',
					'sec-ch-ua-bitness': '"64"',
					'sec-ch-ua-full-version-list': '"Not_A Brand";v="8.0.0.0", "Chromium";v="120.0.6099.109", "Google Chrome";v="120.0.6099.109"',
					'sec-ch-ua-mobile': '?0',
					'sec-ch-ua-model': '""',
					'sec-ch-ua-platform': '"Windows"',
					'sec-ch-ua-platform-version': '"15.0.0"',
					'sec-ch-ua-wow64': '?0',
					'viewport-width': Math.floor(1920 + Math.random() * 200).toString(),
					'x-client-data': 'CIK2yQEIpLbJAQipncoBCMDdygEIkqHLAQiSocsBCO2uzAEI2L/MAgjy08wBCPjTzAEIk9TMAQjy1MwBCKvYzAEIk9nMAQio2cwBCLfZzAE=',
				};
			};
			
			// Intercept and modify XMLHttpRequest
			if (window.XMLHttpRequest) {
				const originalXHR = window.XMLHttpRequest;
				const originalOpen = originalXHR.prototype.open;
				const originalSetRequestHeader = originalXHR.prototype.setRequestHeader;
				const originalSend = originalXHR.prototype.send;
				
				window.XMLHttpRequest = function() {
					const xhr = new originalXHR();
					let headers = {};
					let method = '';
					let url = '';
					
					xhr.open = function(m, u, ...args) {
						method = m;
						url = u;
						
						// Add realistic timing delay
						const delay = Math.random() * 20 + 5;
						setTimeout(() => {
							originalOpen.call(this, m, u, ...args);
							
							// Set dynamic headers after opening
							const dynamicHeaders = generateDynamicHeaders(u);
							Object.keys(dynamicHeaders).forEach(key => {
								try {
									originalSetRequestHeader.call(this, key, dynamicHeaders[key]);
								} catch(e) {}
							});
						}, delay);
					};
					
					xhr.setRequestHeader = function(header, value) {
						headers[header.toLowerCase()] = value;
						
						// Filter out suspicious headers
						const suspiciousHeaders = ['x-automation', 'x-webdriver', 'x-selenium'];
						if (!suspiciousHeaders.includes(header.toLowerCase())) {
							originalSetRequestHeader.call(this, header, value);
						}
					};
					
					xhr.send = function(...args) {
						// Add human-like delay before sending
						const delay = Math.random() * 30 + 10;
						setTimeout(() => {
							originalSend.apply(this, args);
						}, delay);
					};
					
					return xhr;
				};
				
				// Copy static properties
				Object.keys(originalXHR).forEach(key => {
					window.XMLHttpRequest[key] = originalXHR[key];
				});
			}
			
			// Advanced Fetch API interception
			if (window.fetch) {
				const originalFetch = window.fetch;
				
				window.fetch = function(input, init = {}) {
					const url = typeof input === 'string' ? input : input.url;
					
					// Generate realistic headers
					const baseHeaders = {
						'Accept': 'application/json, text/plain, */*',
						'Accept-Encoding': 'gzip, deflate, br',
						'Accept-Language': 'en-US,en;q=0.9,en-GB;q=0.8',
						'Cache-Control': 'no-cache',
						'Pragma': 'no-cache',
						'Sec-Fetch-Dest': 'empty',
						'Sec-Fetch-Mode': 'cors',
						'Sec-Fetch-Site': 'same-origin'
					};
					
					const dynamicHeaders = generateDynamicHeaders(url);
					
					// Merge headers while preserving existing ones
					const finalHeaders = {
						...baseHeaders,
						...dynamicHeaders,
						...(init.headers || {})
					};
					
					// Remove any automation-related headers
					delete finalHeaders['x-automation'];
					delete finalHeaders['x-webdriver'];
					delete finalHeaders['x-selenium'];
					
					const finalInit = {
						...init,
						headers: finalHeaders
					};
					
					// Add realistic network delay
					const networkDelay = Math.random() * 100 + 20;
					
					return new Promise(resolve => {
						setTimeout(() => {
							resolve(originalFetch.call(this, input, finalInit));
						}, networkDelay);
					});
				};
			}
			
		})();
	`
}

// GetExtremeSteathBrowserArgs returns browser arguments with even more aggressive stealth
func GetExtremeStealthBrowserArgs(sessionID string, port int, viewport Viewport) []string {
	userAgent := DefaultStealthConfig.GetRandomUserAgent()

	// Start with ultra-stealth args and add more aggressive ones
	baseArgs := GetUltraStealthBrowserArgs(sessionID, port, viewport)

	// Additional extreme stealth arguments
	extremeArgs := []string{
		// Anti-fingerprinting measures
		"--disable-blink-features=AutomationControlled,ExecCommandInJavaScript",
		"--disable-domain-reliability-reporting",
		"--disable-component-extensions-with-background-pages",
		"--disable-default-apps",
		"--disable-desktop-notifications",
		"--disable-extensions-http-throttling",
		"--disable-field-trial-config",
		"--disable-background-networking",
		"--disable-renderer-backgrounding",
		"--disable-backgrounding-occluded-windows",
		"--disable-back-forward-cache",
		"--disable-ipc-flooding-protection",

		// Advanced Chrome detection bypass
		"--disable-features=AutomationControlled,VizDisplayCompositor,TranslateUI,BlinkGenPropertyTrees,MediaRouter,OptimizationHints,AudioServiceOutOfProcess",
		"--disable-hang-monitor",
		"--disable-prompt-on-repost",
		"--disable-client-side-phishing-detection",
		"--disable-component-update",
		"--disable-sync",

		// Memory and process flags
		"--memory-pressure-off",
		"--renderer-process-limit=1",
		"--max-active-webgl-contexts=1",
		"--disable-accelerated-2d-canvas",
		"--disable-accelerated-jpeg-decoding",
		"--disable-accelerated-mjpeg-decode",
		"--disable-accelerated-video-decode",
		"--disable-accelerated-video-encode",
		"--disable-gpu-rasterization",
		"--disable-gpu-sandbox",

		// Network and security
		"--disable-web-security",
		"--allow-running-insecure-content",
		"--ignore-certificate-errors-spki-list",
		"--ignore-ssl-errors",
		"--ignore-certificate-errors-policy-installed",
		"--allow-cross-origin-auth-prompt",

		// Logging and reporting
		"--disable-logging",
		"--disable-dev-tools",
		"--log-level=3",
		"--silent",
		"--no-report-upload",
		"--disable-crash-reporter",
		"--crash-dumps-dir=/tmp",

		// User agent spoofing with rotation
		"--user-agent=" + userAgent,

		// Random viewport variation
		fmt.Sprintf("--window-size=%d,%d",
			viewport.Width,
			viewport.Height),

		// Additional stealth flags discovered from research
		"--disable-features=WebOTP,WebPayments,WebUSB,VoiceInteraction,PictureInPicture,SensorExtraClasses",
		"--disable-permissions-api",
		"--disable-presentation-api",
		"--disable-remote-fonts",
		"--disable-speech-api",
		"--hide-scrollbars",
		"--mute-audio",
		"--no-experiments",
		"--no-pings",
		"--no-zygote",
		"--single-process",

		// Advanced fingerprinting protection
		"--fingerprinting-canvas-measuretext-noise",
		"--fingerprinting-canvas-image-data-noise",
		"--fingerprinting-client-rects-noise",
		"--canvas-msaa-sample-count=0",
		"--disable-reading-from-canvas",

		// WebRTC protection
		"--disable-webrtc-multiple-routes",
		"--disable-webrtc-hw-decoding",
		"--disable-webrtc-hw-encoding",
		"--enforce-webrtc-ip-permission-check",
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",

		// Final evasion techniques
		"--simulate-outdated-no-au='Tue, 31 Dec 2099 23:59:59 GMT'",
		"--aggressive-cache-discard",
		"--enable-automation=false",
	}

	// Combine base args with extreme args, removing duplicates
	allArgs := append(baseArgs, extremeArgs...)
	seen := make(map[string]bool)
	result := []string{}

	for _, arg := range allArgs {
		key := strings.Split(arg, "=")[0]
		if !seen[key] {
			seen[key] = true
			result = append(result, arg)
		}
	}

	return result
}

// GenerateSessionFingerprint creates a consistent fingerprint for the session
func GenerateSessionFingerprint(sessionID string) map[string]interface{} {
	rand.Seed(time.Now().UnixNano())

	return map[string]interface{}{
		"screen": map[string]int{
			"width":      1920 + rand.Intn(400),
			"height":     1080 + rand.Intn(300),
			"colorDepth": 24,
			"pixelDepth": 24,
		},
		"timezone": rand.Intn(12) - 6, // Random timezone offset
		"language": []string{"en-US", "en", "en-GB"}[rand.Intn(3)],
		"platform": []string{"Win32", "MacIntel", "Linux x86_64"}[rand.Intn(3)],
		"webgl": map[string]string{
			"vendor":   []string{"Intel Inc.", "NVIDIA Corporation", "AMD"}[rand.Intn(3)],
			"renderer": []string{"Intel HD Graphics", "GeForce GTX 1060", "Radeon RX 580"}[rand.Intn(3)],
		},
	}
}
