package stealth

import (
	"math/rand"
	"time"
)

// GetUltimateStealthJS returns the most advanced stealth JavaScript to defeat 2025 Google detection
func GetUltimateStealthJS() string {
	return `
		(function() {
			'use strict';
			
			// ==== ULTIMATE STEALTH SYSTEM 2025 ====
			
			// Complete automation signature elimination
			const eliminateAutomationSignatures = () => {
				// Remove ALL possible automation indicators
				const signatures = [
					'webdriver', '__webdriver_script_fn', '__driver_evaluate', '__webdriver_evaluate',
					'__selenium_evaluate', '__fxdriver_evaluate', '__driver_unwrapped', '__webdriver_unwrapped',
					'__selenium_unwrapped', '__fxdriver_unwrapped', '__nightmare', '__phantomas', 'callPhantom',
					'_phantom', 'phantom', '__wdProfiler', 'webdriverCommand', 'webdriver-evaluate',
					'selenium-evaluate', 'webdriver-evaluate-response', 'domAutomation', 'domAutomationController',
					'__webdriver_script_func', '__webdriver_script_function', 'webdriver_id', '__selenium_id',
					'$cdc_asdjflasutopfhvcZLmcfl_', 'cdc_adoQpoasnfa76pfcZLmcfl_Array',
					'cdc_adoQpoasnfa76pfcZLmcfl_Promise', 'cdc_adoQpoasnfa76pfcZLmcfl_Symbol',
					'$chrome_asyncScriptInfo', '__$webdriverAsyncExecutor', '__webdriverFunc',
					'calledSelenium', '__selenium_evaluation', '__fxdriver_evaluation', '__driver_evaluation'
				];
				
				// Deep object cleanup
				signatures.forEach(sig => {
					try {
						if (window[sig]) { delete window[sig]; Object.defineProperty(window, sig, {value: undefined}); }
						if (navigator[sig]) { delete navigator[sig]; Object.defineProperty(navigator, sig, {value: undefined}); }
						if (document[sig]) { delete document[sig]; Object.defineProperty(document, sig, {value: undefined}); }
					} catch(e) {}
				});
				
				// Advanced CDP pattern removal
				const cdpPatterns = [
					/^cdc_[a-zA-Z0-9_]+$/i, /^__nightmare/i, /^__phantom/i, /^_selenium/i, /^_webdriver/i,
					/^webdriver/i, /^chrome_asyncScriptInfo/i, /^__chrome_asyncScriptInfo/i,
					/^__webdriver_script_fn/i, /^__webdriver_script_func/i, /^selenium/i, /^__selenium/i
				];
				
				const cleanObject = (obj) => {
					if (!obj) return;
					Object.getOwnPropertyNames(obj).forEach(prop => {
						if (cdpPatterns.some(pattern => pattern.test(prop))) {
							try {
								delete obj[prop];
								Object.defineProperty(obj, prop, {value: undefined, writable: false});
							} catch(e) {}
						}
					});
				};
				
				cleanObject(window);
				cleanObject(document);
				cleanObject(navigator);
				setInterval(() => { cleanObject(window); cleanObject(document); cleanObject(navigator); }, 100);
			};
			
			// Advanced permission spoofing
			const spoofPermissions = () => {
				if (navigator.permissions && navigator.permissions.query) {
					const originalQuery = navigator.permissions.query;
					navigator.permissions.query = async function(descriptor) {
						// Return realistic permission states
						const permissionStates = {
							'notifications': Math.random() > 0.5 ? 'granted' : 'default',
							'geolocation': 'prompt',
							'camera': 'prompt',
							'microphone': 'prompt',
							'background-sync': 'granted',
							'persistent-storage': 'prompt'
						};
						
						const state = permissionStates[descriptor.name] || 'prompt';
						return Promise.resolve({ state: state });
					};
				}
			};
			
			// Sophisticated WebRTC protection
			const protectWebRTC = () => {
				// Override RTCPeerConnection to prevent IP leaks
				if (window.RTCPeerConnection) {
					const OriginalRTCPeerConnection = window.RTCPeerConnection;
					window.RTCPeerConnection = function(...args) {
						const pc = new OriginalRTCPeerConnection(...args);
						
						// Block ICE candidate gathering
						const originalCreateOffer = pc.createOffer;
						pc.createOffer = function(options) {
							return originalCreateOffer.call(this, {
								...options,
								iceRestart: false
							}).then(offer => {
								// Remove real IP addresses from SDP
								offer.sdp = offer.sdp.replace(/([0-9]{1,3}\.){3}[0-9]{1,3}/g, '192.168.1.1');
								return offer;
							});
						};
						
						return pc;
					};
				}
				
				// Block other WebRTC APIs
				if (navigator.getUserMedia) navigator.getUserMedia = undefined;
				if (navigator.webkitGetUserMedia) navigator.webkitGetUserMedia = undefined;
				if (navigator.mozGetUserMedia) navigator.mozGetUserMedia = undefined;
				if (navigator.mediaDevices) navigator.mediaDevices.getUserMedia = () => Promise.reject(new Error('Permission denied'));
			};
			
			// Advanced timing attack prevention
			const protectTiming = () => {
				// Randomize performance.now() with realistic jitter
				const originalNow = performance.now;
				let baseOffset = Math.random() * 100;
				performance.now = function() {
					const real = originalNow.call(this);
					const jitter = (Math.random() - 0.5) * 0.1;
					return real + baseOffset + jitter;
				};
				
				// Protect Date.now() as well
				const originalDateNow = Date.now;
				Date.now = function() {
					const real = originalDateNow.call(this);
					const jitter = Math.floor((Math.random() - 0.5) * 10);
					return real + jitter;
				};
				
				// Override setTimeout/setInterval to add realistic delays
				const originalSetTimeout = window.setTimeout;
				window.setTimeout = function(callback, delay, ...args) {
					const jitter = Math.random() * 2;
					return originalSetTimeout(callback, delay + jitter, ...args);
				};
			};
			
			// Advanced network behavior masking
			const maskNetworkBehavior = () => {
				// Override XMLHttpRequest to add realistic headers and timing
				const OriginalXHR = window.XMLHttpRequest;
				window.XMLHttpRequest = function() {
					const xhr = new OriginalXHR();
					const originalOpen = xhr.open;
					const originalSend = xhr.send;
					
					xhr.open = function(method, url, ...args) {
						// Add realistic headers
						originalOpen.apply(this, arguments);
						
						// Set realistic headers that browsers send
						this.setRequestHeader('Accept', 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8');
						this.setRequestHeader('Accept-Language', 'en-US,en;q=0.9');
						this.setRequestHeader('Accept-Encoding', 'gzip, deflate, br');
						this.setRequestHeader('Cache-Control', 'no-cache');
						this.setRequestHeader('Pragma', 'no-cache');
						
						// Random DNT header
						if (Math.random() > 0.5) {
							this.setRequestHeader('DNT', '1');
						}
						
						// Random Upgrade-Insecure-Requests
						if (url.startsWith('http:')) {
							this.setRequestHeader('Upgrade-Insecure-Requests', '1');
						}
					};
					
					xhr.send = function(data) {
						// Add realistic delay before sending
						setTimeout(() => {
							originalSend.call(this, data);
						}, Math.random() * 50);
					};
					
					return xhr;
				};
				
				// Override fetch with realistic options
				const originalFetch = window.fetch;
				window.fetch = function(url, options = {}) {
					// Add realistic headers
					const defaultHeaders = {
						'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8',
						'Accept-Language': 'en-US,en;q=0.9',
						'Accept-Encoding': 'gzip, deflate, br',
						'Cache-Control': 'no-cache',
						'Pragma': 'no-cache',
						'Sec-Fetch-Dest': 'document',
						'Sec-Fetch-Mode': 'navigate',
						'Sec-Fetch-Site': 'none',
						'Sec-Fetch-User': '?1'
					};
					
					options.headers = { ...defaultHeaders, ...(options.headers || {}) };
					
					// Add realistic delay
					return new Promise(resolve => {
						setTimeout(() => {
							resolve(originalFetch(url, options));
						}, Math.random() * 100);
					});
				};
			};
			
			// Browser inconsistency elimination
			const eliminateInconsistencies = () => {
				// Ensure navigator properties are consistent
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
					set: () => {},
					configurable: true
				});
				
				// Remove automation indicators from chrome object
				if (window.chrome && window.chrome.runtime) {
					// Make it look like a real extension environment
					const mockExtensionId = 'mhjfbmdgcfjbbpaeojofohoefgiehjai'; // Real Chrome PDF extension ID
					
					window.chrome.runtime.onConnect.addListener = function(callback) {
						setTimeout(() => {
							callback({
								name: 'content-script',
								sender: { id: mockExtensionId }
							});
						}, 100 + Math.random() * 500);
					};
				}
				
				// Spoof plugins array to look realistic
				Object.defineProperty(navigator, 'plugins', {
					get: () => {
						const plugins = [
							{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer' },
							{ name: 'Native Client', filename: 'internal-nacl-plugin' }
						];
						plugins.length = 2;
						return plugins;
					},
					configurable: true
				});
			};
			
			// Ultimate mouse and keyboard simulation
			const ultimateHumanSimulation = () => {
				let mouseX = Math.floor(Math.random() * window.innerWidth);
				let mouseY = Math.floor(Math.random() * window.innerHeight);
				let isMoving = false;
				
				// Sophisticated mouse movement with acceleration and deceleration
				const moveMouseHuman = () => {
					if (isMoving) return;
					isMoving = true;
					
					const targetX = Math.random() * window.innerWidth;
					const targetY = Math.random() * window.innerHeight;
					const steps = 20 + Math.random() * 30;
					let currentStep = 0;
					
					const moveStep = () => {
						currentStep++;
						const progress = currentStep / steps;
						
						// Ease-in-out curve for realistic movement
						const easeProgress = progress < 0.5 ? 
							2 * progress * progress : 
							1 - Math.pow(-2 * progress + 2, 3) / 2;
						
						mouseX = mouseX + (targetX - mouseX) * easeProgress * 0.1;
						mouseY = mouseY + (targetY - mouseY) * easeProgress * 0.1;
						
						// Add micro-movements (human tremor)
						mouseX += (Math.random() - 0.5) * 2;
						mouseY += (Math.random() - 0.5) * 2;
						
						document.dispatchEvent(new MouseEvent('mousemove', {
							clientX: Math.floor(mouseX),
							clientY: Math.floor(mouseY),
							bubbles: true
						}));
						
						if (currentStep < steps) {
							setTimeout(moveStep, 16 + Math.random() * 8);
						} else {
							isMoving = false;
						}
					};
					
					moveStep();
				};
				
				// Random human-like actions
				const performRandomActions = () => {
					// Mouse movements every 1-3 seconds
					setInterval(moveMouseHuman, 1000 + Math.random() * 2000);
					
					// Random clicks (very occasional)
					setInterval(() => {
						if (Math.random() < 0.05) {
							document.dispatchEvent(new MouseEvent('click', {
								clientX: Math.floor(mouseX),
								clientY: Math.floor(mouseY),
								bubbles: true
							}));
						}
					}, 5000 + Math.random() * 10000);
					
					// Random keyboard events
					const keys = ['Tab', 'ArrowDown', 'ArrowUp', 'PageDown', 'PageUp', 'Home', 'End'];
					setInterval(() => {
						if (Math.random() < 0.1) {
							const key = keys[Math.floor(Math.random() * keys.length)];
							document.dispatchEvent(new KeyboardEvent('keydown', {
								key: key,
								bubbles: true
							}));
						}
					}, 3000 + Math.random() * 7000);
					
					// Random scroll behavior
					let scrollDirection = 1;
					setInterval(() => {
						if (Math.random() < 0.3) {
							const scrollAmount = (50 + Math.random() * 100) * scrollDirection;
							window.scrollBy({
								top: scrollAmount,
								behavior: Math.random() > 0.5 ? 'smooth' : 'auto'
							});
							
							// Change direction occasionally
							if (Math.random() < 0.2) {
								scrollDirection *= -1;
							}
						}
					}, 2000 + Math.random() * 4000);
				};
				
				// Start human simulation after page is ready
				if (document.readyState === 'loading') {
					document.addEventListener('DOMContentLoaded', () => {
						setTimeout(performRandomActions, 1000 + Math.random() * 2000);
					});
				} else {
					setTimeout(performRandomActions, 500 + Math.random() * 1000);
				}
			};
			
			// Apply all protections immediately
			eliminateAutomationSignatures();
			spoofPermissions();
			protectWebRTC();
			protectTiming();
			maskNetworkBehavior();
			eliminateInconsistencies();
			ultimateHumanSimulation();
			
			// Continuous protection
			setInterval(eliminateAutomationSignatures, 500);
			setInterval(eliminateInconsistencies, 1000);
			
			console.log('🛡️ Ultimate Stealth System 2025 - ACTIVE');
		})();`
}

// GetUltimateStealthArguments returns the most comprehensive browser arguments
func GetUltimateStealthArguments() []string {
	return []string{
		// Core automation hiding (CRITICAL)
		"--disable-blink-features=AutomationControlled",
		"--exclude-switches=enable-automation",
		"--disable-automation",
		"--enable-automation=false",

		// Network behavior masking
		"--disable-features=VizDisplayCompositor,TranslateUI,BlinkGenPropertyTrees",
		"--disable-features=ScriptStreaming,V8OptimizeJavascript,NetworkService",
		"--disable-features=LazyImageLoading,AudioServiceOutOfProcess",
		"--disable-features=OptimizationGuideModelDownloading,OptimizationHintsFetching",
		"--disable-ipc-flooding-protection",

		// HTTP/2 and connection masking
		"--disable-http2",
		"--disable-quic",
		"--force-fieldtrials=*BackgroundTimerThrottling/Disabled/",
		"--aggressive-cache-discard",
		"--enable-fast-unload",

		// TLS fingerprint masking
		"--ssl-version-fallback-min=tls1.2",
		"--ignore-ssl-errors=true",
		"--ignore-certificate-errors=true",
		"--allow-running-insecure-content",
		"--disable-web-security",

		// Request timing randomization
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--disable-back-forward-cache",
		"--force-color-profile=srgb",

		// Memory and process masking
		"--memory-pressure-off",
		"--max-old-space-size=4096",
		"--process-per-site",
		"--disable-dev-shm-usage",

		// Hardware fingerprint masking
		"--disable-accelerated-2d-canvas",
		"--disable-accelerated-jpeg-decoding",
		"--disable-accelerated-mjpeg-decode",
		"--disable-accelerated-video-decode",
		"--disable-accelerated-video-encode",
		"--disable-gpu-rasterization",
		"--disable-gpu-sandbox",
		"--disable-gpu",

		// Google-specific detection blocks
		"--disable-component-update",
		"--disable-domain-reliability",
		"--disable-sync",
		"--disable-background-networking",
		"--disable-client-side-phishing-detection",
		"--disable-component-extensions-with-background-pages",
		"--disable-default-apps",
		"--disable-background-mode",

		// Extension and plugin masking
		"--disable-extensions",
		"--disable-extensions-file-access-check",
		"--disable-extensions-http-throttling",
		"--disable-plugins-discovery",
		"--disable-bundled-ppapi-flash",

		// Media masking
		"--mute-audio",
		"--disable-audio-output",
		"--autoplay-policy=no-user-gesture-required",
		"--disable-features=MediaRouter",

		// Privacy and tracking prevention
		"--enable-privacy-sandbox-ads-apis=false",
		"--disable-features=PrivacySandboxSettings4",
		"--disable-features=InterestFeedContentSuggestions",
		"--no-pings",
		"--no-referrers",

		// Logging suppression
		"--disable-logging",
		"--disable-dev-tools",
		"--log-level=3",
		"--silent",
		"--no-crash-upload",

		// Window positioning randomization
		"--window-position=" + generateRandomWindowPosition(),

		// Non-headless window stability flags
		"--disable-infobars",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-translate",

		// Additional stealth flags
		"--disable-renderer-accessibility",
		"--disable-hang-monitor",
		"--disable-prompt-on-repost",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-translate",
		"--disable-features=Translate",
	}
}

// generateRandomWindowPosition creates realistic window positioning
func generateRandomWindowPosition() string {
	positions := []string{
		"0,0", "100,50", "200,100", "50,25", "150,75", "250,125",
		"25,10", "75,35", "125,60", "175,85", "225,110",
	}
	return positions[rand.Intn(len(positions))]
}

// GetAntiDetectionUserAgents returns the most current undetected user agents
func GetAntiDetectionUserAgents() []string {
	return []string{
		// Latest Chrome versions (September 2025) with exact version strings used by real browsers
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",

		// Edge versions (avoiding Chrome pattern detection)
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0",

		// Firefox (different engine entirely)
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:131.0) Gecko/20100101 Firefox/131.0",

		// Mobile user agents for diversity
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; SM-G998U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
