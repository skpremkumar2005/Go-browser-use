package stealth

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
)

// ProxyConfig represents a proxy server configuration
type ProxyConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Type     string // "http", "socks5", etc.
}

// ProxyRotator manages proxy rotation for enhanced anonymity
type ProxyRotator struct {
	proxies      []ProxyConfig
	currentIndex int
	useProxies   bool
}

// NewProxyRotator creates a new proxy rotator
func NewProxyRotator(proxies []ProxyConfig) *ProxyRotator {
	return &ProxyRotator{
		proxies:      proxies,
		currentIndex: 0,
		useProxies:   len(proxies) > 0,
	}
}

// GetNextProxy returns the next proxy in rotation
func (pr *ProxyRotator) GetNextProxy() *ProxyConfig {
	if !pr.useProxies || len(pr.proxies) == 0 {
		return nil
	}

	proxy := &pr.proxies[pr.currentIndex]
	pr.currentIndex = (pr.currentIndex + 1) % len(pr.proxies)
	return proxy
}

// GetProxyBrowserArgs returns browser arguments for proxy usage
func GetProxyBrowserArgs(proxy *ProxyConfig) []string {
	if proxy == nil {
		return []string{}
	}

	args := []string{
		fmt.Sprintf("--proxy-server=%s://%s:%d", proxy.Type, proxy.Host, proxy.Port),
	}

	if proxy.Username != "" && proxy.Password != "" {
		// Note: Chrome doesn't support proxy auth via command line
		// This would need to be handled via proxy-auto-config (PAC) file
		log.Printf("🔐 Proxy authentication detected - consider using PAC file")
	}

	return args
}

// GetVPNSimulationArgs returns args to simulate VPN-like behavior without actual VPN
func GetVPNSimulationArgs() []string {
	return []string{
		// Use system DNS instead of forcing specific DNS server
		// Removed problematic "--host-resolver-rules=MAP * 8.8.8.8" that breaks navigation
		"--dns-prefetch-disable", // Disable DNS prefetching for privacy

		// Additional privacy flags that VPN users might enable
		"--disable-background-networking",
		"--disable-domain-reliability",
		"--disable-component-update",
		"--disable-sync",
		"--incognito",

		// Simulate geographic location variation
		"--lang=" + getRandomLocale(),
		"--accept-lang=" + getRandomAcceptLanguage(),
	}
}

// getRandomLocale returns a random locale for geographic variation
func getRandomLocale() string {
	locales := []string{
		"en-US,en", "en-GB,en", "en-CA,en", "en-AU,en",
		"fr-FR,fr", "de-DE,de", "es-ES,es", "it-IT,it",
		"nl-NL,nl", "pt-PT,pt", "ru-RU,ru", "ja-JP,ja",
	}
	return locales[rand.Intn(len(locales))]
}

// getRandomAcceptLanguage returns a random Accept-Language header
func getRandomAcceptLanguage() string {
	acceptLangs := []string{
		"en-US,en;q=0.9",
		"en-GB,en;q=0.9,en-US;q=0.8",
		"en-US,en;q=0.9,fr;q=0.8",
		"en-US,en;q=0.9,es;q=0.8,fr;q=0.7",
		"en-GB,en;q=0.9,en-US;q=0.8,fr;q=0.7",
	}
	return acceptLangs[rand.Intn(len(acceptLangs))]
}

// MilspecStealthArgs returns military-specification stealth arguments
func GetMilspecStealthArgs(sessionID string, port int, viewport Viewport, proxy *ProxyConfig) []string {
	rand.Seed(time.Now().UnixNano())
	userAgent := DefaultStealthConfig.GetRandomUserAgent()

	baseArgs := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),

		// Core stealth - MANDATORY for Google bypass
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-gpu",

		// CRITICAL: Primary automation detection prevention
		"--disable-blink-features=AutomationControlled",
		"--exclude-switches=enable-automation",
		"--disable-automation",
		"--enable-automation=false",

		// Google-specific detection countermeasures
		"--disable-features=VizDisplayCompositor",
		"--disable-features=ScriptStreaming",
		"--disable-features=V8OptimizeJavascript",
		"--disable-features=NetworkService",
		"--disable-features=LazyImageLoading",
		"--disable-component-extensions-with-background-pages",
		"--disable-default-apps",
		"--disable-background-mode",
		"--force-color-profile=srgb",
		"--disable-renderer-accessibility",

		// Viewport with minimal variation for non-headless stability
		fmt.Sprintf("--window-size=%d,%d",
			viewport.Width,   // ±10px variation
			viewport.Height), // ±10px variation
		"--window-position=0,0",

		// User agent with session consistency
		"--user-agent=" + userAgent,

		// Locale randomization for geographic dispersion
		"--lang=" + getRandomLocale(),
		"--accept-lang=" + getRandomAcceptLanguage(),

		// ADVANCED: Feature disabling for maximum stealth
		"--disable-features=AutomationControlled,VizDisplayCompositor,TranslateUI,BlinkGenPropertyTrees,AudioServiceOutOfProcess,MediaRouter,OptimizationHints,Translate,VizServiceSharingFromGPUProcess,WebOTP,WebPayments,WebUSB,VoiceInteraction,PictureInPicture,SensorExtraClasses",

		// Privacy and security hardening
		"--disable-web-security",
		"--allow-running-insecure-content",
		"--ignore-certificate-errors",
		"--ignore-ssl-errors",
		"--ignore-certificate-errors-spki-list",
		"--ignore-certificate-errors-policy-installed",
		"--allow-cross-origin-auth-prompt",

		// Google-specific evasion
		"--disable-client-side-phishing-detection",
		"--disable-component-update",
		"--disable-domain-reliability",
		"--disable-sync",
		// Removed duplicate "--disable-background-networking" (already in VPN simulation)
		"--disable-hang-monitor",
		"--disable-prompt-on-repost",

		// Process and memory management
		"--disable-extensions",
		"--disable-extensions-file-access-check",
		"--disable-extensions-http-throttling",
		"--disable-component-extensions-with-background-pages",
		"--disable-default-apps",
		"--no-first-run",
		"--no-service-autorun",
		"--password-store=basic",
		"--use-mock-keychain",

		// Advanced timing and behavior
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--disable-ipc-flooding-protection",
		"--disable-field-trial-config",
		"--disable-back-forward-cache",
		"--disable-backing-store-limit",

		// Memory optimization
		"--memory-pressure-off",
		"--max_old_space_size=4096",
		// Removed --no-zygote and --single-process as they break CDP connection

		// CDP compatibility (ensure remote debugging works)
		"--enable-automation=false", // Hide automation flag while keeping CDP
		"--disable-dev-shm-usage",   // Prevent shared memory issues

		// Network connectivity fixes (ensure proper navigation)
		"--allow-running-insecure-content",
		"--disable-web-security",
		"--disable-features=TranslateUI",
		"--no-default-browser-check",
		"--no-first-run",
		"--ignore-ssl-errors=true",
		"--ignore-certificate-errors=true",

		// Media and hardware masking
		"--disable-accelerated-2d-canvas",
		"--disable-accelerated-jpeg-decoding",
		"--disable-accelerated-mjpeg-decode",
		"--disable-accelerated-video-decode",
		"--disable-accelerated-video-encode",
		"--disable-gpu-rasterization",
		"--disable-gpu-sandbox",
		"--mute-audio",
		"--disable-audio-output",
		"--autoplay-policy=no-user-gesture-required",

		// Advanced fingerprinting resistance
		"--fingerprinting-canvas-measuretext-noise",
		"--fingerprinting-canvas-image-data-noise",
		"--fingerprinting-client-rects-noise",
		"--disable-reading-from-canvas",
		"--canvas-msaa-sample-count=0",

		// WebRTC stealth (critical for IP masking)
		"--disable-webrtc-multiple-routes",
		"--disable-webrtc-hw-decoding",
		"--disable-webrtc-hw-encoding",
		"--enforce-webrtc-ip-permission-check",
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",

		// Logging and telemetry suppression
		"--disable-logging",
		"--disable-dev-tools",
		"--log-level=3",
		"--silent",
		"--no-report-upload",
		"--disable-crash-reporter",
		"--disable-breakpad",
		"--crash-dumps-dir=/tmp",
		"--disable-logging-redirect",

		// Final evasion layer
		"--simulate-outdated-no-au='Tue, 31 Dec 2099 23:59:59 GMT'",
		"--aggressive-cache-discard",
		"--no-experiments",
		"--no-pings",
		"--aggressive",
		"--hide-scrollbars",
		"--disable-permissions-api",
		"--disable-presentation-api",
		"--disable-remote-fonts",
		"--disable-speech-api",
		"--disable-wake-on-wifi",
		"--disable-background-mode",
		"--force-color-profile=srgb",
	}

	// Add proxy args if proxy is configured
	if proxy != nil {
		proxyArgs := GetProxyBrowserArgs(proxy)
		baseArgs = append(baseArgs, proxyArgs...)
	} else {
		// Add VPN simulation args if no real proxy
		vpnArgs := GetVPNSimulationArgs()
		baseArgs = append(baseArgs, vpnArgs...)
	}

	return baseArgs
}
