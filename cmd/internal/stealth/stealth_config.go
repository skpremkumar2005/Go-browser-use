package stealth

import (
	"math/rand"
	"time"
)

// StealthConfig holds all stealth configuration
type StealthConfig struct {
	UserAgents    []string
	ViewportSizes []Viewport
	MouseConfig   MouseBehaviorConfig
	TypingConfig  TypingBehaviorConfig
	ActionDelays  ActionDelayConfig
}

type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type MouseBehaviorConfig struct {
	CurveStrength        float64
	SpeedVariation       float64
	MinSteps             int
	MaxSteps             int
	PauseProbability     float64
	OvershootProbability float64
	OvershootDistance    int
}

type TypingBehaviorConfig struct {
	FastTyping    SpeedRange
	NormalTyping  SpeedRange
	SlowTyping    SpeedRange
	MistakeChance float64
	WordPause     SpeedRange
}

type ActionDelayConfig struct {
	MinActionDelay  time.Duration
	MaxActionDelay  time.Duration
	MinPageLoadWait time.Duration
	MaxPageLoadWait time.Duration
	MinElementWait  time.Duration
	MaxElementWait  time.Duration
}

type SpeedRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// DefaultStealthConfig provides default stealth configuration
var DefaultStealthConfig = StealthConfig{
	UserAgents: []string{
		// Ultimate stealth user agents - September 2025 (most current and undetected)
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		// Edge variants to avoid Chrome-only patterns
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0",
		// Firefox for engine diversity
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:131.0) Gecko/20100101 Firefox/131.0",
		// Mobile variants for pattern diversity
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; SM-G998U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
	},
	ViewportSizes: []Viewport{
		{Width: 1920, Height: 1080},
	},
	MouseConfig: MouseBehaviorConfig{
		CurveStrength:        0.3,
		SpeedVariation:       0.4,
		MinSteps:             15,
		MaxSteps:             35,
		PauseProbability:     0.15,
		OvershootProbability: 0.1,
		OvershootDistance:    8,
	},
	TypingConfig: TypingBehaviorConfig{
		FastTyping:    SpeedRange{Min: 50, Max: 120},
		NormalTyping:  SpeedRange{Min: 80, Max: 200},
		SlowTyping:    SpeedRange{Min: 150, Max: 350},
		MistakeChance: 0.02,
		WordPause:     SpeedRange{Min: 200, Max: 500},
	},
	ActionDelays: ActionDelayConfig{
		MinActionDelay:  1500 * time.Millisecond,
		MaxActionDelay:  3500 * time.Millisecond,
		MinPageLoadWait: 2000 * time.Millisecond,
		MaxPageLoadWait: 5000 * time.Millisecond,
		MinElementWait:  500 * time.Millisecond,
		MaxElementWait:  1500 * time.Millisecond,
	},
}

// GetRandomUserAgent returns a random user agent
func (c *StealthConfig) GetRandomUserAgent() string {
	rand.Seed(time.Now().UnixNano())
	return c.UserAgents[rand.Intn(len(c.UserAgents))]
}

// GetRandomViewport returns a random viewport size
func (c *StealthConfig) GetRandomViewport() Viewport {
	rand.Seed(time.Now().UnixNano())
	return c.ViewportSizes[rand.Intn(len(c.ViewportSizes))]
}

// GetRandomActionDelay returns a random delay for human-like timing
func (c *StealthConfig) GetRandomActionDelay() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := c.ActionDelays.MinActionDelay
	max := c.ActionDelays.MaxActionDelay
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// GetRandomPageLoadDelay returns a random page load wait time
func (c *StealthConfig) GetRandomPageLoadDelay() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := c.ActionDelays.MinPageLoadWait
	max := c.ActionDelays.MaxPageLoadWait
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// GetRandomElementWait returns a random element wait time
func (c *StealthConfig) GetRandomElementWait() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := c.ActionDelays.MinElementWait
	max := c.ActionDelays.MaxElementWait
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
