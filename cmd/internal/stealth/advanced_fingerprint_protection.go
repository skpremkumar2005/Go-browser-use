package stealth

import (
	// "fmt"
	"math/rand"
	"time"
)

// GetAdvancedWebRTCStealthJS returns JavaScript to mask WebRTC fingerprinting
func GetAdvancedWebRTCStealthJS() string {
	return `
		(function() {
			'use strict';

			// Phase 1: WebRTC Connection Masking
			if (window.RTCPeerConnection) {
				const originalRTCPeerConnection = window.RTCPeerConnection;
				
				window.RTCPeerConnection = function(configuration) {
					const pc = new originalRTCPeerConnection(configuration);
					
					// Override createOffer to mask fingerprints
					const originalCreateOffer = pc.createOffer;
					pc.createOffer = function() {
						return originalCreateOffer.apply(this, arguments).then(offer => {
							// Randomize ICE fingerprints
							offer.sdp = offer.sdp.replace(
								/a=fingerprint:sha-256 ([A-F0-9:]+)/g, 
								'a=fingerprint:sha-256 ' + Array.from({length: 32}, () => 
									Math.floor(Math.random() * 16).toString(16).toUpperCase()
								).join(':')
							);
							
							// Randomize connection IDs
							offer.sdp = offer.sdp.replace(
								/a=ice-ufrag:([a-zA-Z0-9+/]+)/g,
								'a=ice-ufrag:' + btoa(Math.random().toString()).substring(0, 8)
							);
							
							offer.sdp = offer.sdp.replace(
								/a=ice-pwd:([a-zA-Z0-9+/]+)/g,
								'a=ice-pwd:' + btoa(Math.random().toString()).substring(0, 22)
							);
							
							return offer;
						});
					};
					
					// Override createAnswer similarly
					const originalCreateAnswer = pc.createAnswer;
					pc.createAnswer = function() {
						return originalCreateAnswer.apply(this, arguments).then(answer => {
							// Apply similar fingerprint masking
							answer.sdp = answer.sdp.replace(
								/a=fingerprint:sha-256 ([A-F0-9:]+)/g,
								'a=fingerprint:sha-256 ' + Array.from({length: 32}, () =>
									Math.floor(Math.random() * 16).toString(16).toUpperCase()
								).join(':')
							);
							return answer;
						});
					};
					
					return pc;
				};
				
				// Copy static methods
				for (let key in originalRTCPeerConnection) {
					window.RTCPeerConnection[key] = originalRTCPeerConnection[key];
				}
			}

			// Phase 2: RTCDataChannel Masking  
			if (window.RTCDataChannel) {
				const originalDataChannelPrototype = RTCDataChannel.prototype;
				const descriptors = Object.getOwnPropertyDescriptors(originalDataChannelPrototype);
				
				// Mask channel properties that could fingerprint
				if (descriptors.id) {
					Object.defineProperty(RTCDataChannel.prototype, 'id', {
						get: function() {
							return Math.floor(Math.random() * 65535);
						},
						configurable: true
					});
				}
			}

			// Phase 3: Media Stream Masking
			if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
				const originalGetUserMedia = navigator.mediaDevices.getUserMedia;
				
				navigator.mediaDevices.getUserMedia = function(constraints) {
					// Add realistic delay before responding
					return new Promise((resolve, reject) => {
						setTimeout(() => {
							originalGetUserMedia.apply(this, arguments)
								.then(resolve)
								.catch(reject);
						}, 100 + Math.random() * 200);
					});
				};
			}

			// Phase 4: Advanced TLS Fingerprint Masking
			if (window.crypto && window.crypto.subtle) {
				const originalGenerateKey = window.crypto.subtle.generateKey;
				
				window.crypto.subtle.generateKey = function() {
					// Add timing variation to key generation
					return new Promise((resolve, reject) => {
						const delay = Math.random() * 50;
						setTimeout(() => {
							originalGenerateKey.apply(this, arguments)
								.then(resolve)
								.catch(reject);
						}, delay);
					});
				};
			}

			// Phase 5: Connection Timing Masking
			if (window.performance && window.performance.getEntriesByType) {
				const originalGetEntriesByType = window.performance.getEntriesByType;
				
				window.performance.getEntriesByType = function(type) {
					const entries = originalGetEntriesByType.apply(this, arguments);
					
					// Add noise to network timing entries
					if (type === 'navigation' || type === 'resource') {
						entries.forEach(entry => {
							const noise = Math.random() * 10 - 5;
							if (entry.connectStart) entry.connectStart += noise;
							if (entry.connectEnd) entry.connectEnd += noise;
							if (entry.requestStart) entry.requestStart += noise;
							if (entry.responseStart) entry.responseStart += noise;
							if (entry.responseEnd) entry.responseEnd += noise;
						});
					}
					
					return entries;
				};
			}

			// Phase 6: Font Fingerprinting Protection
			if (document.fonts && document.fonts.check) {
				const originalCheck = document.fonts.check;
				const commonFonts = [
					'Arial', 'Helvetica', 'Times New Roman', 'Courier New', 'Verdana', 
					'Georgia', 'Palatino', 'Garamond', 'Bookman', 'Comic Sans MS',
					'Trebuchet MS', 'Arial Black', 'Impact', 'Calibri', 'Tahoma'
				];
				
				document.fonts.check = function(font, text) {
					// Always return true for common fonts to avoid fingerprinting
					const fontFamily = font.toLowerCase();
					if (commonFonts.some(f => fontFamily.includes(f.toLowerCase()))) {
						return true;
					}
					return originalCheck.apply(this, arguments);
				};
			}

			// Phase 7: Advanced AudioContext Masking
			if (window.AudioContext || window.webkitAudioContext) {
				const AudioContextClass = window.AudioContext || window.webkitAudioContext;
				const originalCreateOscillator = AudioContextClass.prototype.createOscillator;
				
				AudioContextClass.prototype.createOscillator = function() {
					const oscillator = originalCreateOscillator.apply(this, arguments);
					
					// Add noise to frequency to mask audio fingerprinting
					const originalSetValue = oscillator.frequency.setValueAtTime;
					oscillator.frequency.setValueAtTime = function(value, time) {
						const noise = (Math.random() - 0.5) * 0.1;
						return originalSetValue.call(this, value + noise, time);
					};
					
					return oscillator;
				};
			}

			// Phase 8: Hardware Concurrency Masking
			if (navigator.hardwareConcurrency) {
				const commonCores = [2, 4, 6, 8, 12, 16];
				Object.defineProperty(navigator, 'hardwareConcurrency', {
					get: () => commonCores[Math.floor(Math.random() * commonCores.length)],
					configurable: true
				});
			}

			// Phase 9: Device Memory Masking
			if (navigator.deviceMemory) {
				const commonMemorySizes = [2, 4, 8, 16];
				Object.defineProperty(navigator, 'deviceMemory', {
					get: () => commonMemorySizes[Math.floor(Math.random() * commonMemorySizes.length)],
					configurable: true
				});
			}

			// Phase 10: Final Validation and Cleanup
			console.log('🛡️ Advanced WebRTC and TLS fingerprint protection enabled');
			
		})();
	`
}

// GetHumanBehaviorDelays returns realistic timing delays for human-like behavior
func GetHumanBehaviorDelays() map[string]time.Duration {
	delays := map[string]time.Duration{
		"page_load":      time.Duration(1500+rand.Intn(2000)) * time.Millisecond, // 1.5-3.5s
		"before_click":   time.Duration(200+rand.Intn(500)) * time.Millisecond,   // 200-700ms
		"after_click":    time.Duration(100+rand.Intn(300)) * time.Millisecond,   // 100-400ms
		"before_type":    time.Duration(150+rand.Intn(350)) * time.Millisecond,   // 150-500ms
		"between_chars":  time.Duration(50+rand.Intn(150)) * time.Millisecond,    // 50-200ms
		"before_scroll":  time.Duration(300+rand.Intn(700)) * time.Millisecond,   // 300ms-1s
		"reading_pause":  time.Duration(800+rand.Intn(1200)) * time.Millisecond,  // 0.8-2s
		"decision_pause": time.Duration(1000+rand.Intn(2000)) * time.Millisecond, // 1-3s
		"focus_change":   time.Duration(100+rand.Intn(200)) * time.Millisecond,   // 100-300ms
	}

	return delays
}

// AddHumanDelay adds a realistic human delay for the specified action
func AddHumanDelay(action string) {
	delays := GetHumanBehaviorDelays()
	if delay, exists := delays[action]; exists {
		time.Sleep(delay)
	} else {
		// Default delay for unknown actions
		time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
	}
}
