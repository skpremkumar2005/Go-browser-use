package stealth

import "fmt"

// GetStealthJavaScript returns the best stealth system - now uses Unified Platform architecture
func GetStealthJavaScript() string {
	return GetUnifiedPlatformCompleteStealthJS()
}

// GetEnhancedStealthJavaScript returns advanced JavaScript stealth injection code
func GetEnhancedStealthJavaScript() string {
	return `
		(function() {
			'use strict';
			
			// Phase 1: Core WebDriver Detection Prevention
			if (navigator.webdriver !== undefined) {
				delete navigator.webdriver;
			}
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
				configurable: true
			});

			// Phase 2: Remove Chrome Automation Variables
			const cdcProps = [
				'cdc_adoQpoasnfa76pfcZLmcfl_Array',
				'cdc_adoQpoasnfa76pfcZLmcfl_Promise', 
				'cdc_adoQpoasnfa76pfcZLmcfl_Symbol',
				'cdc_adoQpoasnfa76pfcZLmcfl_JSON',
				'cdc_adoQpoasnfa76pfcZLmcfl_Object',
				'cdc_adoQpoasnfa76pfcZLmcfl_Proxy'
			];
			cdcProps.forEach(prop => delete window[prop]);

			// Phase 3: Enhanced Plugin Simulation
			const mockPlugins = [
				{
					name: 'Chrome PDF Plugin',
					filename: 'internal-pdf-viewer',
					description: 'Portable Document Format',
					length: 1,
					item: () => null,
					namedItem: () => null
				},
				{
					name: 'Chromium PDF Plugin', 
					filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai',
					description: 'Portable Document Format',
					length: 1,
					item: () => null,
					namedItem: () => null
				},
				{
					name: 'Microsoft Edge PDF Plugin',
					filename: 'pdf',
					description: 'pdf',
					length: 1,
					item: () => null,
					namedItem: () => null
				}
			];
			
			Object.defineProperty(navigator, 'plugins', {
				get: () => mockPlugins,
				configurable: true
			});

			// Phase 4: Advanced Language and Locale Spoofing
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en', 'en-GB', 'es'],
				configurable: true
			});
			
			Object.defineProperty(navigator, 'language', {
				get: () => 'en-US',
				configurable: true
			});

			// Phase 5: Enhanced Chrome Runtime Mocking
			if (!window.chrome) {
				window.chrome = {};
			}
			
			const mockRuntime = {
				onConnect: {
					addListener: () => {},
					removeListener: () => {},
					hasListener: () => false
				},
				onMessage: {
					addListener: () => {},
					removeListener: () => {},
					hasListener: () => false
				},
				connect: () => ({
					name: '',
					sender: undefined,
					onDisconnect: {
						addListener: () => {},
						removeListener: () => {}
					},
					onMessage: {
						addListener: () => {},
						removeListener: () => {}
					},
					postMessage: () => {}
				}),
				sendMessage: () => {},
				id: undefined,
				getManifest: () => undefined,
				getURL: (path) => 'chrome-extension://invalid/' + path
			};
			
			Object.defineProperty(window.chrome, 'runtime', {
				get: () => mockRuntime,
				configurable: true
			});

			// Phase 6: Advanced Screen and Device Spoofing
			const originalScreen = { ...screen };
			const screenNoise = () => Math.floor(Math.random() * 5) - 2;
			
			Object.defineProperty(screen, 'availHeight', {
				get: () => originalScreen.availHeight + screenNoise(),
				configurable: true
			});
			
			Object.defineProperty(screen, 'availWidth', {
				get: () => originalScreen.availWidth + screenNoise(),
				configurable: true
			});
			
			Object.defineProperty(screen, 'height', {
				get: () => originalScreen.height + screenNoise(),
				configurable: true
			});
			
			Object.defineProperty(screen, 'width', {
				get: () => originalScreen.width + screenNoise(),
				configurable: true
			});

			// Phase 7: Advanced Canvas Fingerprint Masking
			const originalGetContext = HTMLCanvasElement.prototype.getContext;
			const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
			const originalGetImageData = CanvasRenderingContext2D.prototype.getImageData;
			
			HTMLCanvasElement.prototype.getContext = function(type, ...args) {
				const context = originalGetContext.apply(this, [type, ...args]);
				
				if (type === '2d') {
					context.getImageData = function(...args) {
						const imageData = originalGetImageData.apply(this, args);
						const data = imageData.data;
						
						// Add sophisticated noise pattern
						for (let i = 0; i < data.length; i += 4) {
							if (Math.random() < 0.001) {
								const noise = Math.random() * 2 - 1;
								data[i] = Math.min(255, Math.max(0, data[i] + noise));     // R
								data[i + 1] = Math.min(255, Math.max(0, data[i + 1] + noise)); // G
								data[i + 2] = Math.min(255, Math.max(0, data[i + 2] + noise)); // B
							}
						}
						
						return imageData;
					};
				}
				
				return context;
			};
			
			HTMLCanvasElement.prototype.toDataURL = function(...args) {
				// Add minimal noise to canvas output
				const originalResult = originalToDataURL.apply(this, args);
				if (Math.random() < 0.1) {
					// Slightly modify the base64 string occasionally
					return originalResult.slice(0, -1) + String.fromCharCode(65 + Math.floor(Math.random() * 26));
				}
				return originalResult;
			};

			// Phase 8: Enhanced WebGL Fingerprint Protection
			const originalGetParameter = WebGLRenderingContext.prototype.getParameter;
			WebGLRenderingContext.prototype.getParameter = function(parameter) {
				// Randomize GPU vendor/renderer info
				if (parameter === 37445) { // UNMASKED_VENDOR_WEBGL
					const vendors = ['Intel Inc.', 'Google Inc.', 'AMD'];
					return vendors[Math.floor(Math.random() * vendors.length)];
				}
				if (parameter === 37446) { // UNMASKED_RENDERER_WEBGL
					const renderers = [
						'Intel Iris OpenGL Engine',
						'ANGLE (Intel, Intel(R) HD Graphics 620, OpenGL 4.1)',
						'AMD Radeon Pro 560X OpenGL Engine'
					];
					return renderers[Math.floor(Math.random() * renderers.length)];
				}
				return originalGetParameter.apply(this, arguments);
			};

			// Phase 9: Advanced Permission API Handling
			if (navigator.permissions && navigator.permissions.query) {
				const originalQuery = navigator.permissions.query.bind(navigator.permissions);
				navigator.permissions.query = function(parameters) {
					// Handle common permission queries naturally
					if (parameters.name === 'notifications') {
						return Promise.resolve({ state: 'default' });
					}
					if (parameters.name === 'geolocation') {
						return Promise.resolve({ state: 'prompt' });
					}
					if (parameters.name === 'camera') {
						return Promise.resolve({ state: 'prompt' });
					}
					if (parameters.name === 'microphone') {
						return Promise.resolve({ state: 'prompt' });
					}
					return originalQuery(parameters);
				};
			}

			// Phase 10: Enhanced Media Device Simulation
			if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
				navigator.mediaDevices.enumerateDevices = () => Promise.resolve([
					{
						deviceId: 'default',
						groupId: 'group1',
						kind: 'audioinput',
						label: 'Built-in Microphone'
					},
					{
						deviceId: 'default', 
						groupId: 'group2',
						kind: 'audiooutput',
						label: 'Built-in Speakers'
					},
					{
						deviceId: 'camera1',
						groupId: 'group3',
						kind: 'videoinput',
						label: 'Built-in Camera'
					}
				]);
			}

			// Phase 11: Battery API Spoofing with Realistic Values
			if ('getBattery' in navigator) {
				navigator.getBattery = () => Promise.resolve({
					charging: Math.random() > 0.5,
					chargingTime: Math.random() > 0.5 ? Infinity : Math.random() * 3600,
					dischargingTime: Math.random() * 14400,
					level: 0.2 + Math.random() * 0.8 // 20-100%
				});
			}

			// Phase 12: Enhanced Connection API with Dynamic Values
			if ('connection' in navigator) {
				Object.defineProperty(navigator, 'connection', {
					get: () => ({
						downlink: 5 + Math.random() * 10, // 5-15 Mbps
						effectiveType: ['4g', '3g', 'slow-2g'][Math.floor(Math.random() * 3)],
						rtt: 50 + Math.random() * 100, // 50-150ms
						saveData: false
					}),
					configurable: true
				});
			}

			// Phase 13: Advanced Timing Attack Prevention
			const originalNow = performance.now;
			performance.now = function() {
				return originalNow.apply(this) + (Math.random() - 0.5) * 0.1;
			};
			
			const originalGetTime = Date.prototype.getTime;
			Date.prototype.getTime = function() {
				return originalGetTime.apply(this) + Math.floor((Math.random() - 0.5) * 2);
			};

			// Phase 14: Mouse Movement and Activity Simulation
			let mouseSimulation = {
				x: Math.random() * window.innerWidth,
				y: Math.random() * window.innerHeight,
				lastMove: Date.now()
			};
			
			// Simulate subtle mouse movements
			setInterval(() => {
				if (Date.now() - mouseSimulation.lastMove > 5000) {
					mouseSimulation.x += (Math.random() - 0.5) * 3;
					mouseSimulation.y += (Math.random() - 0.5) * 3;
					mouseSimulation.x = Math.max(0, Math.min(window.innerWidth, mouseSimulation.x));
					mouseSimulation.y = Math.max(0, Math.min(window.innerHeight, mouseSimulation.y));
					mouseSimulation.lastMove = Date.now();
				}
			}, 2000 + Math.random() * 3000);

			// Phase 15: Enhanced toString Method Overrides
			const originalToString = Function.prototype.toString;
			Function.prototype.toString = function() {
				if (this === navigator.permissions.query) {
					return 'function query() { [native code] }';
				}
				if (this === WebGLRenderingContext.prototype.getParameter) {
					return 'function getParameter() { [native code] }';
				}
				if (this === HTMLCanvasElement.prototype.getContext) {
					return 'function getContext() { [native code] }';
				}
				return originalToString.apply(this, arguments);
			};

			// Phase 16: Geolocation API Enhancement
			if (navigator.geolocation) {
				const originalGetCurrentPosition = navigator.geolocation.getCurrentPosition;
				navigator.geolocation.getCurrentPosition = function(success, error, options) {
					// Simulate realistic geolocation behavior
					setTimeout(() => {
						if (error && Math.random() > 0.8) {
							error({ code: 1, message: 'User denied Geolocation' });
						} else if (success) {
							success({
								coords: {
									latitude: 40.7128 + (Math.random() - 0.5) * 0.1,
									longitude: -74.0060 + (Math.random() - 0.5) * 0.1,
									accuracy: 10 + Math.random() * 40,
									altitude: null,
									altitudeAccuracy: null,
									heading: null,
									speed: null
								},
								timestamp: Date.now()
							});
						}
					}, 500 + Math.random() * 2000);
				};
			}

			// Phase 17: Console Detection Prevention
			const originalLog = console.log;
			console.log = function(...args) {
				// Filter out stealth-related logs in production
				const logStr = args.join(' ');
				if (!logStr.includes('stealth') && !logStr.includes('🥷')) {
					originalLog.apply(console, args);
				}
			};

			console.log('🥷 Ultra stealth mode activated with 17 phases');
		})();
	`
}

// GetAdvancedBrowserArgs returns enhanced Chrome arguments for stealth mode
func GetAdvancedBrowserArgs(userAgent string, viewport Viewport) []string {
	return []string{
		// Core stealth arguments (maintain existing)
		"--disable-blink-features=AutomationControlled",
		"--exclude-switches=enable-automation",
		"--disable-automation",
		"--disable-extensions-file-access-check",
		"--disable-extensions-http-throttling",
		"--disable-save-password-bubble",

		// Enhanced user agent and language
		"--user-agent=" + userAgent,
		"--lang=en-US,en",
		"--accept-lang=en-US,en;q=0.9,en-GB;q=0.8,es;q=0.7",

		// Viewport settings
		"--window-size=" + fmt.Sprintf("%d,%d", viewport.Width, viewport.Height),

		// Advanced privacy settings
		"--disable-web-security",
		"--allow-running-insecure-content",
		"--disable-features=VizDisplayCompositor,AutomationControlled,ScriptStreaming,TranslateUI,BlinkGenPropertyTrees",

		// Enhanced stealth behavior
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

		// Memory and performance
		"--memory-pressure-off",
		"--max_old_space_size=2048",
		"--no-zygote",

		// Google-specific enhancements
		"--disable-client-side-phishing-detection",
		"--disable-component-update",
		"--disable-hang-monitor",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-domain-reliability",
		"--disable-component-extensions-with-background-pages",
		"--disable-breakpad",
		"--disable-crash-reporter",
		"--disable-features=TranslateUI",

		// Enhanced fingerprint resistance
		"--disable-accelerated-2d-canvas",
		"--disable-accelerated-jpeg-decoding",
		"--disable-accelerated-mjpeg-decode",
		"--disable-accelerated-video-decode",
		"--disable-gpu-sandbox",
		"--disable-software-rasterizer",

		// Additional stealth flags
		"--password-store=basic",
		"--use-mock-keychain",
		"--disable-password-generation",
		"--disable-password-manager-reauthentication",
		"--disable-dev-shm-usage",
		"--no-sandbox",
		"--headless=new",
		"--disable-gpu",
	}
}
