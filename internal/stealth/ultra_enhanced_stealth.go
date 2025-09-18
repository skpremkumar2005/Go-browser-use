package stealth

import (
	"fmt"
	"math/rand"
	"time"
)

// GetUltraStealthJavaScript returns the most advanced JavaScript stealth code
func GetUltraStealthJavaScript() string {
	baseStealthScript := getBaseStealthScript()
	webrtcProtection := GetAdvancedWebRTCStealthJS()

	return fmt.Sprintf(`
		%s
		
		// Additional WebRTC and Advanced Fingerprint Protection
		%s
	`, baseStealthScript, webrtcProtection)
}

// getBaseStealthScript returns the base stealth script
func getBaseStealthScript() string {
	return `
		(function() {
			'use strict';
			
			const originalConsole = window.console;
			
			// Phase 1: Ultimate WebDriver Detection Prevention 
			const webdriverProps = ['webdriver', '__webdriver_script_fn', '__driver_evaluate', '__webdriver_evaluate', '__selenium_evaluate', '__fxdriver_evaluate', '__driver_unwrapped', '__webdriver_unwrapped', '__selenium_unwrapped', '__fxdriver_unwrapped'];
			webdriverProps.forEach(prop => {
				if (navigator[prop] !== undefined) {
					delete navigator[prop];
				}
				Object.defineProperty(navigator, prop, {
					get: () => undefined,
					configurable: true,
					enumerable: false
				});
			});

			// Phase 2: Advanced Chrome DevTools Protocol Variables Removal
			const cdcProps = Object.getOwnPropertyNames(window).filter(prop => 
				prop.includes('cdc_') || 
				prop.includes('__nightmare') ||
				prop.includes('__phantom') ||
				prop.includes('__selenium') ||
				prop.includes('__webdriver') ||
				prop.includes('calledSelenium') ||
				prop.includes('_selenium') ||
				prop.includes('_webdriver')
			);
			cdcProps.forEach(prop => {
				try {
					delete window[prop];
				} catch (e) {}
			});

			// Phase 3: Runtime Environment Cleansing
			['$chrome_asyncScriptInfo', '$cdc_asdjflasutopfhvcZLmcfl_'].forEach(prop => {
				if (window[prop]) delete window[prop];
			});

			// Phase 4: Advanced Permission API Spoofing
			if (navigator.permissions && navigator.permissions.query) {
				const originalQuery = navigator.permissions.query;
				navigator.permissions.query = function(parameters) {
					if (parameters.name === 'notifications') {
						return Promise.resolve({ state: 'granted' });
					}
					return originalQuery.apply(navigator.permissions, arguments);
				};
			}

			// Phase 5: Enhanced Plugin Array with Real Behavior
			const createMockPlugin = (name, filename, description) => ({
				name,
				filename,
				description,
				length: 1,
				item: function(index) { return index === 0 ? this : null; },
				namedItem: function(name) { return name === this.name ? this : null; },
				refresh: function() {},
				[Symbol.iterator]: function*() { yield this; }
			});

			const realisticPlugins = [
				createMockPlugin('Chrome PDF Plugin', 'internal-pdf-viewer', 'Portable Document Format'),
				createMockPlugin('Chromium PDF Plugin', 'mhjfbmdgcfjbbpaeojofohoefgiehjai', 'Portable Document Format'), 
				createMockPlugin('Microsoft Edge PDF Plugin', 'pdf', 'pdf'),
				createMockPlugin('WebKit built-in PDF', 'webkit-pdf-plugin', 'pdf')
			];

			Object.defineProperty(navigator, 'plugins', {
				get: () => realisticPlugins,
				configurable: false,
				enumerable: true
			});

			// Phase 6: Advanced Screen Property Randomization
			const screenProps = {
				width: 1920,
				height: 1080,
				colorDepth: 24,
				pixelDepth: 24,
				availWidth: null,
				availHeight: null
			};
			screenProps.availWidth = screenProps.width - Math.floor(Math.random() * 100);
			screenProps.availHeight = screenProps.height - Math.floor(Math.random() * 150);

			Object.keys(screenProps).forEach(prop => {
				Object.defineProperty(screen, prop, {
					get: () => screenProps[prop],
					configurable: true
				});
			});

			// Phase 7: Advanced Canvas Fingerprint Masking with Realistic Noise
			const getImageData = HTMLCanvasElement.prototype.toDataURL;
			HTMLCanvasElement.prototype.toDataURL = function(...args) {
				const context = this.getContext('2d');
				if (context) {
					const imageData = context.getImageData(0, 0, this.width, this.height);
					for (let i = 0; i < imageData.data.length; i += 4) {
						const noise = Math.floor(Math.random() * 10) - 5;
						imageData.data[i] = Math.min(255, Math.max(0, imageData.data[i] + noise));
						imageData.data[i + 1] = Math.min(255, Math.max(0, imageData.data[i + 1] + noise));
						imageData.data[i + 2] = Math.min(255, Math.max(0, imageData.data[i + 2] + noise));
					}
					context.putImageData(imageData, 0, 0);
				}
				return getImageData.apply(this, args);
			};

			// Phase 8: WebGL Advanced Fingerprint Protection
			const originalGetParameter = WebGLRenderingContext.prototype.getParameter;
			const webglVendors = ['Intel Inc.', 'NVIDIA Corporation', 'AMD', 'Microsoft Corporation'];
			const webglRenderers = [
				'Intel(R) UHD Graphics 620',
				'NVIDIA GeForce GTX 1060', 
				'AMD Radeon RX 580',
				'Intel(R) HD Graphics 4000'
			];
			
			WebGLRenderingContext.prototype.getParameter = function(parameter) {
				if (parameter === 37445) { // UNMASKED_VENDOR_WEBGL
					return webglVendors[Math.floor(Math.random() * webglVendors.length)];
				}
				if (parameter === 37446) { // UNMASKED_RENDERER_WEBGL  
					return webglRenderers[Math.floor(Math.random() * webglRenderers.length)];
				}
				return originalGetParameter.apply(this, arguments);
			};

			// Phase 9: Enhanced Media Device Simulation
			if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
				const originalEnumerateDevices = navigator.mediaDevices.enumerateDevices;
				navigator.mediaDevices.enumerateDevices = function() {
					return Promise.resolve([
						{
							deviceId: 'default',
							kind: 'audioinput',
							label: 'Default - Microphone Array (Intel® Smart Sound Technology)',
							groupId: 'group1'
						},
						{
							deviceId: 'communications',
							kind: 'audioinput', 
							label: 'Communications - Microphone Array (Intel® Smart Sound Technology)',
							groupId: 'group1'
						},
						{
							deviceId: 'camera1',
							kind: 'videoinput',
							label: 'Integrated Camera (04f2:b5ce)',
							groupId: 'group2'
						}
					]);
				};
			}

			// Phase 10: Battery API Advanced Spoofing
			if (navigator.getBattery) {
				navigator.getBattery = function() {
					const batteryLevels = [0.21, 0.47, 0.63, 0.89, 0.92];
					const chargingStates = [true, false];
					const chargingTimes = [3600, 5400, 7200, Infinity];
					const dischargingTimes = [18000, 21600, 28800, 32400];

					return Promise.resolve({
						level: batteryLevels[Math.floor(Math.random() * batteryLevels.length)],
						charging: chargingStates[Math.floor(Math.random() * chargingStates.length)],
						chargingTime: chargingTimes[Math.floor(Math.random() * chargingTimes.length)],
						dischargingTime: dischargingTimes[Math.floor(Math.random() * dischargingTimes.length)],
						addEventListener: function() {},
						removeEventListener: function() {}
					});
				};
			}

			// Phase 11: Network Connection API Enhanced Spoofing
			const connectionTypes = ['4g', '3g', 'wifi'];
			const effectiveTypes = ['4g', '3g', 'slow-2g', '2g'];
			
			if (navigator.connection) {
				const mockConnection = {
					downlink: 10 + Math.random() * 40,
					effectiveType: effectiveTypes[Math.floor(Math.random() * effectiveTypes.length)],
					rtt: 50 + Math.floor(Math.random() * 150),
					type: connectionTypes[Math.floor(Math.random() * connectionTypes.length)],
					saveData: Math.random() > 0.7,
					addEventListener: function() {},
					removeEventListener: function() {}
				};

				Object.defineProperty(navigator, 'connection', {
					get: () => mockConnection,
					configurable: true
				});
			}

			// Phase 12: Advanced Language and Locale Masking
			const languages = [
				['en-US', 'en', 'en-GB', 'es'],
				['en-US', 'en'],
				['en-GB', 'en', 'en-US'],
				['en-US', 'en', 'fr', 'de']
			];
			const selectedLanguages = languages[Math.floor(Math.random() * languages.length)];

			Object.defineProperty(navigator, 'languages', {
				get: () => selectedLanguages,
				configurable: true
			});
			
			Object.defineProperty(navigator, 'language', {
				get: () => selectedLanguages[0],
				configurable: true
			});

			// Phase 13: Performance API Timing Randomization
			if (window.performance && window.performance.now) {
				const originalNow = window.performance.now;
				let timeOffset = Math.random() * 100;
				
				window.performance.now = function() {
					return originalNow.apply(this, arguments) + timeOffset + Math.random() * 0.1;
				};
			}

			// Phase 14: Chrome Runtime Enhanced Mocking
			if (!window.chrome) window.chrome = {};
			if (!window.chrome.runtime) {
				window.chrome.runtime = {
					onConnect: { addListener: function() {}, removeListener: function() {} },
					onMessage: { addListener: function() {}, removeListener: function() {} },
					connect: function() {
						return {
							onMessage: { addListener: function() {} },
							onDisconnect: { addListener: function() {} },
							postMessage: function() {}
						};
					},
					sendMessage: function() {},
					id: 'mhjfbmdgcfjbbpaeojofohoefgiehjai'
				};
			}

			// Phase 15: Geolocation API Enhancement
			if (navigator.geolocation) {
				const originalGetCurrentPosition = navigator.geolocation.getCurrentPosition;
				navigator.geolocation.getCurrentPosition = function(success, error, options) {
					setTimeout(() => {
						if (success) {
							success({
								coords: {
									accuracy: 20 + Math.random() * 80,
									altitude: null,
									altitudeAccuracy: null,
									heading: null,
									latitude: 37.7749 + (Math.random() - 0.5) * 0.01,
									longitude: -122.4194 + (Math.random() - 0.5) * 0.01,
									speed: null
								},
								timestamp: Date.now()
							});
						}
					}, 100 + Math.random() * 200);
				};
			}

			// Phase 16: Enhanced toString Override Protection
			const descriptors = ['webdriver', 'plugins', 'languages', 'language'];
			descriptors.forEach(prop => {
				const descriptor = Object.getOwnPropertyDescriptor(navigator, prop);
				if (descriptor && descriptor.get) {
					descriptor.get.toString = function() {
						return 'function get ' + prop + '() { [native code] }';
					};
				}
			});

			// Phase 17: Console Detection Prevention and Mouse Activity Simulation
			let mouseActivity = Date.now();
			
			['log', 'debug', 'info', 'warn', 'error'].forEach(method => {
				const original = originalConsole[method];
				originalConsole[method] = function(...args) {
					const message = args.join(' ');
					if (!message.includes('stealth') && !message.includes('automation') && !message.includes('webdriver')) {
						return original.apply(originalConsole, args);
					}
				};
			});

			// Mouse movement simulation
			document.addEventListener('mousemove', () => {
				mouseActivity = Date.now();
			});

			// Simulate background mouse activity
			setInterval(() => {
				if (Date.now() - mouseActivity > 30000) {
					const event = new MouseEvent('mousemove', {
						view: window,
						bubbles: true,
						cancelable: true,
						clientX: Math.random() * window.innerWidth,
						clientY: Math.random() * window.innerHeight
					});
					document.dispatchEvent(event);
					mouseActivity = Date.now();
				}
			}, 30000);

			// Phase 18: WebRTC Advanced Fingerprint Masking
			if (window.RTCPeerConnection) {
				const originalCreateOffer = RTCPeerConnection.prototype.createOffer;
				RTCPeerConnection.prototype.createOffer = function() {
					return originalCreateOffer.apply(this, arguments).then(offer => {
						// Modify SDP to mask fingerprint
						offer.sdp = offer.sdp.replace(/a=fingerprint:sha-256 ([A-F0-9:]+)/g, 
							'a=fingerprint:sha-256 ' + Array.from({length: 32}, () => 
								Math.floor(Math.random() * 16).toString(16).toUpperCase()
							).join(':'));
						return offer;
					});
				};
			}

			// Phase 19: Advanced Headless Detection Prevention
			Object.defineProperty(navigator, 'webkitTemporaryStorage', {
				get: () => ({ queryUsageAndQuota: function() {} }),
				configurable: true
			});

			Object.defineProperty(navigator, 'webkitPersistentStorage', {
				get: () => ({ queryUsageAndQuota: function() {} }),
				configurable: true
			});

			// Phase 20: Final Environment Validation
			if (window.outerHeight === 0) {
				Object.defineProperty(window, 'outerHeight', {
					get: () => window.innerHeight + 85,
					configurable: true
				});
			}
			
			if (window.outerWidth === 0) {
				Object.defineProperty(window, 'outerWidth', {
					get: () => window.innerWidth + 16,
					configurable: true
				});
			}

		})();
	`
}

// GetAdvancedTiming returns human-like timing for various actions
func GetAdvancedTiming() map[string]time.Duration {
	return map[string]time.Duration{
		"page_load_wait": time.Duration(2000+rand.Intn(3000)) * time.Millisecond, // 2-5 seconds
		"click_delay":    time.Duration(100+rand.Intn(300)) * time.Millisecond,   // 100-400ms
		"type_delay":     time.Duration(50+rand.Intn(150)) * time.Millisecond,    // 50-200ms per char
		"scroll_delay":   time.Duration(200+rand.Intn(500)) * time.Millisecond,   // 200-700ms
		"human_pause":    time.Duration(1000+rand.Intn(2000)) * time.Millisecond, // 1-3 seconds
	}
}
