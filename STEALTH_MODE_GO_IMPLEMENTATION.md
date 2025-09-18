# Stealth Mode Implementation in Go Language

## 🚀 Overview

This document provides a comprehensive guide to implementing the 13-phase stealth browser automation system in Go language. While the original implementation uses Node.js/JavaScript, Go offers unique advantages for high-performance, concurrent browser automation with better resource management and deployment characteristics.

## 🏗️ Go Ecosystem for Browser Automation

### Available Go Libraries

#### 1. **chromedp** (Recommended Primary)
```go
import "github.com/chromedp/chromedp"
```
- **Native CDP Protocol**: Direct Chrome DevTools Protocol implementation
- **High Performance**: Compiled binary, low overhead
- **Concurrent**: Go's goroutines for parallel sessions
- **Stealth Friendly**: Low-level control over browser behavior

#### 2. **rod** (Alternative)
```go
import "github.com/go-rod/rod"
```
- **Developer Friendly**: Higher-level API
- **Stealth Built-in**: Some anti-detection features
- **Concurrent**: Good parallelism support

#### 3. **playwright-go** (Cross-browser)
```go
import "github.com/playwright-community/playwright-go"
```
- **Multi-browser**: Chrome, Firefox, Safari support
- **Feature Rich**: Comprehensive automation features

## 🏛️ Go Architecture Design

### Package Structure
```
stealth-browser-go/
├── cmd/
│   └── stealth-server/
│       └── main.go              # HTTP server entry point
├── internal/
│   ├── browser/
│   │   ├── manager.go           # Browser lifecycle management
│   │   ├── session.go           # Session management
│   │   └── pool.go             # Browser instance pooling
│   ├── stealth/
│   │   ├── config.go           # Stealth configuration
│   │   ├── navigator.go        # DOM patching scripts
│   │   ├── mouse.go            # Human mouse movement
│   │   ├── typing.go           # Human typing simulation
│   │   ├── fingerprint.go      # Fingerprint randomization
│   │   └── detection.go        # Detection recovery
│   ├── cdp/
│   │   ├── client.go           # CDP client wrapper
│   │   └── injector.go         # Script injection
│   └── server/
│       ├── handlers.go         # HTTP handlers
│       └── middleware.go       # Middleware stack
├── pkg/
│   ├── stealth/
│   │   └── api.go              # Public API
│   └── types/
│       └── models.go           # Shared types
├── scripts/
│   ├── navigator-stealth.js    # JavaScript stealth scripts
│   ├── canvas-noise.js         # Canvas fingerprint masking
│   └── human-behavior.js       # Behavior simulation
├── configs/
│   └── stealth.yaml           # Configuration files
├── docker/
│   └── Dockerfile             # Container build
├── go.mod
├── go.sum
└── README.md
```

## 🔧 Phase Implementation in Go

### **Phase 1: Browser Launch Arguments**

```go
// internal/stealth/config.go
package stealth

import (
    "math/rand"
    "time"
)

type StealthConfig struct {
    BrowserArgs      []string
    UserAgents       []string
    ViewportSizes    []Viewport
    MouseConfig      MouseBehaviorConfig
    TypingConfig     TypingBehaviorConfig
    ActionDelays     ActionDelayConfig
    HTTPHeaders      HTTPHeadersConfig
    ResourceLimits   ResourceLimitConfig
    RecoveryConfig   RecoveryConfig
}

type Viewport struct {
    Width  int `json:"width"`
    Height int `json:"height"`
}

var DefaultStealthConfig = StealthConfig{
    BrowserArgs: []string{
        "--disable-blink-features=AutomationControlled",
        "--exclude-switches=enable-automation",
        "--disable-extensions",
        "--disable-default-apps",
        "--disable-web-security",
        "--disable-features=VizDisplayCompositor",
        "--disable-dev-shm-usage",
        "--no-sandbox",
        "--disable-setuid-sandbox",
        "--disable-infobars",
        "--disable-notifications",
        "--disable-popup-blocking",
        "--disable-translate",
        "--disable-background-timer-throttling",
        "--disable-backgrounding-occluded-windows",
        "--disable-renderer-backgrounding",
        "--memory-pressure-off",
        "--disable-background-networking",
        "--disable-client-side-phishing-detection",
        "--disable-sync",
        "--metrics-recording-only",
        "--no-report-upload",
        "--no-crash-upload",
        "--remote-debugging-port=0", // Dynamic port allocation
    },
    
    UserAgents: []string{
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
    },
    
    ViewportSizes: []Viewport{
        {Width: 1920, Height: 1080},
        {Width: 1366, Height: 768},
        {Width: 1536, Height: 864},
        {Width: 1440, Height: 900},
        {Width: 1680, Height: 1050},
        {Width: 1600, Height: 900},
        {Width: 1280, Height: 720},
    },
}

func (c *StealthConfig) GetRandomUserAgent() string {
    rand.Seed(time.Now().UnixNano())
    return c.UserAgents[rand.Intn(len(c.UserAgents))]
}

func (c *StealthConfig) GetRandomViewport() Viewport {
    rand.Seed(time.Now().UnixNano())
    return c.ViewportSizes[rand.Intn(len(c.ViewportSizes))]
}
```

### **Phase 2: Browser Manager with Stealth**

```go
// internal/browser/manager.go
package browser

import (
    "context"
    "fmt"
    "log"
    "sync"
    
    "github.com/chromedp/chromedp"
    "github.com/chromedp/chromedp/runner"
    "your-project/internal/stealth"
)

type StealthBrowserManager struct {
    config       *stealth.StealthConfig
    sessions     map[string]*StealthSession
    sessionMutex sync.RWMutex
    logger       *log.Logger
}

type StealthSession struct {
    ID              string
    Context         context.Context
    Cancel          context.CancelFunc
    ChromeContext   context.Context
    ChromeCancel    context.CancelFunc
    UserAgent       string
    Viewport        stealth.Viewport
    CreatedAt       time.Time
    LastActivity    time.Time
    StealthEnabled  bool
    HumanMouse      *stealth.HumanMouseMovement
    HumanTyping     *stealth.HumanTyping
}

func NewStealthBrowserManager(config *stealth.StealthConfig) *StealthBrowserManager {
    return &StealthBrowserManager{
        config:   config,
        sessions: make(map[string]*StealthSession),
        logger:   log.New(os.Stdout, "[StealthBrowser] ", log.LstdFlags),
    }
}

func (m *StealthBrowserManager) CreateStealthSession(sessionID string) (*StealthSession, error) {
    m.sessionMutex.Lock()
    defer m.sessionMutex.Unlock()
    
    // Generate random characteristics
    userAgent := m.config.GetRandomUserAgent()
    viewport := m.config.GetRandomViewport()
    
    // Create Chrome instance with stealth arguments
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("disable-blink-features", "AutomationControlled"),
        chromedp.Flag("exclude-switches", "enable-automation"),
        chromedp.Flag("disable-extensions", true),
        chromedp.UserAgent(userAgent),
        chromedp.WindowSize(viewport.Width, viewport.Height),
    )
    
    // Add all stealth browser arguments
    for _, arg := range m.config.BrowserArgs {
        if strings.Contains(arg, "=") {
            parts := strings.SplitN(arg, "=", 2)
            opts = append(opts, chromedp.Flag(strings.TrimPrefix(parts[0], "--"), parts[1]))
        } else {
            opts = append(opts, chromedp.Flag(strings.TrimPrefix(arg, "--"), true))
        }
    }
    
    allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
    ctx, cancel := chromedp.NewContext(allocCtx)
    
    // Initialize human behavior systems
    humanMouse := stealth.NewHumanMouseMovement()
    humanTyping := stealth.NewHumanTyping()
    
    session := &StealthSession{
        ID:              sessionID,
        Context:         ctx,
        Cancel:          cancel,
        ChromeContext:   allocCtx,
        ChromeCancel:    allocCancel,
        UserAgent:       userAgent,
        Viewport:        viewport,
        CreatedAt:       time.Now(),
        LastActivity:    time.Now(),
        StealthEnabled:  true,
        HumanMouse:      humanMouse,
        HumanTyping:     humanTyping,
    }
    
    // Inject stealth scripts on page load
    err := m.injectStealthScripts(session)
    if err != nil {
        cancel()
        allocCancel()
        return nil, fmt.Errorf("failed to inject stealth scripts: %w", err)
    }
    
    m.sessions[sessionID] = session
    m.logger.Printf("🕵️ Created stealth session: %s with UA: %s, Viewport: %dx%d", 
        sessionID, userAgent, viewport.Width, viewport.Height)
    
    return session, nil
}

func (m *StealthBrowserManager) injectStealthScripts(session *StealthSession) error {
    return chromedp.Run(session.Context,
        chromedp.Navigate("about:blank"),
        chromedp.ActionFunc(func(ctx context.Context) error {
            // Inject navigator stealth script
            _, err := chromedp.RunResponse(ctx, runtime.Evaluate(stealth.NavigatorStealthScript))
            if err != nil {
                return fmt.Errorf("failed to inject navigator stealth: %w", err)
            }
            
            // Inject canvas noise script  
            _, err = chromedp.RunResponse(ctx, runtime.Evaluate(stealth.CanvasNoiseScript))
            if err != nil {
                return fmt.Errorf("failed to inject canvas noise: %w", err)
            }
            
            return nil
        }),
    )
}
```

### **Phase 3: Human Mouse Movement in Go**

```go
// internal/stealth/mouse.go
package stealth

import (
    "context"
    "math"
    "math/rand"
    "time"
    
    "github.com/chromedp/chromedp"
)

type MouseBehaviorConfig struct {
    CurveStrength       float64
    SpeedVariation      float64
    MinSteps           int
    MaxSteps           int
    PauseProbability   float64
    OvershootProbability float64
    OvershootDistance   int
    FastSpeed          SpeedRange
    NormalSpeed        SpeedRange
    SlowSpeed          SpeedRange
}

type SpeedRange struct {
    Min int `json:"min"`
    Max int `json:"max"`
}

type Point struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}

type HumanMouseMovement struct {
    config       MouseBehaviorConfig
    lastPosition Point
}

func NewHumanMouseMovement() *HumanMouseMovement {
    return &HumanMouseMovement{
        config: MouseBehaviorConfig{
            CurveStrength:        0.3,
            SpeedVariation:       0.4,
            MinSteps:            10,
            MaxSteps:            30,
            PauseProbability:    0.2,
            OvershootProbability: 0.15,
            OvershootDistance:   5,
            FastSpeed:           SpeedRange{Min: 1, Max: 3},
            NormalSpeed:         SpeedRange{Min: 3, Max: 8},
            SlowSpeed:           SpeedRange{Min: 8, Max: 15},
        },
        lastPosition: Point{X: 0, Y: 0},
    }
}

func (h *HumanMouseMovement) GenerateBezierCurve(start, end Point, steps int) []Point {
    points := make([]Point, 0, steps+1)
    
    // Create control points for natural curve
    midpoint := Point{
        X: (start.X + end.X) / 2,
        Y: (start.Y + end.Y) / 2,
    }
    
    // Add randomness to control points
    controlPoint1 := Point{
        X: start.X + (midpoint.X-start.X)*h.config.CurveStrength + (rand.Float64()-0.5)*50,
        Y: start.Y + (midpoint.Y-start.Y)*h.config.CurveStrength + (rand.Float64()-0.5)*50,
    }
    
    controlPoint2 := Point{
        X: end.X + (midpoint.X-end.X)*h.config.CurveStrength + (rand.Float64()-0.5)*50,
        Y: end.Y + (midpoint.Y-end.Y)*h.config.CurveStrength + (rand.Float64()-0.5)*50,
    }
    
    // Generate curve points using cubic Bézier formula
    for i := 0; i <= steps; i++ {
        t := float64(i) / float64(steps)
        point := h.cubicBezier(t, start, controlPoint1, controlPoint2, end)
        points = append(points, point)
    }
    
    return points
}

func (h *HumanMouseMovement) cubicBezier(t float64, p0, p1, p2, p3 Point) Point {
    oneMinusT := 1 - t
    
    x := math.Pow(oneMinusT, 3)*p0.X +
        3*math.Pow(oneMinusT, 2)*t*p1.X +
        3*oneMinusT*math.Pow(t, 2)*p2.X +
        math.Pow(t, 3)*p3.X
        
    y := math.Pow(oneMinusT, 3)*p0.Y +
        3*math.Pow(oneMinusT, 2)*t*p1.Y +
        3*oneMinusT*math.Pow(t, 2)*p2.Y +
        math.Pow(t, 3)*p3.Y
        
    return Point{X: math.Round(x), Y: math.Round(y)}
}

func (h *HumanMouseMovement) CalculateSpeed(distance float64) int {
    var speed SpeedRange
    
    if distance < 100 {
        speed = h.config.FastSpeed
    } else if distance < 300 {
        speed = h.config.NormalSpeed
    } else {
        speed = h.config.SlowSpeed
    }
    
    // Add speed variation
    variation := 1 + (rand.Float64()-0.5)*h.config.SpeedVariation
    delay := float64(speed.Min) + rand.Float64()*float64(speed.Max-speed.Min)
    delay *= variation
    
    return int(math.Max(1, math.Round(delay)))
}

func (h *HumanMouseMovement) MoveToHumanLike(ctx context.Context, targetX, targetY float64) error {
    start := h.lastPosition
    end := Point{X: targetX, Y: targetY}
    
    // Calculate distance and steps
    distance := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))
    steps := h.config.MinSteps + rand.Intn(h.config.MaxSteps-h.config.MinSteps)
    
    // Generate Bézier curve path
    path := h.GenerateBezierCurve(start, end, steps)
    
    // Move along the path
    for i, point := range path {
        if i == 0 {
            continue // Skip starting point
        }
        
        // Calculate delay based on distance
        stepDistance := math.Sqrt(math.Pow(point.X-path[i-1].X, 2) + math.Pow(point.Y-path[i-1].Y, 2))
        delay := h.CalculateSpeed(stepDistance)
        
        // Add pause probability
        if rand.Float64() < h.config.PauseProbability {
            time.Sleep(time.Duration(delay*2) * time.Millisecond)
        }
        
        // Move mouse to point
        err := chromedp.Run(ctx, 
            chromedp.MouseEvent(chromedp.MouseMove, point.X, point.Y),
        )
        if err != nil {
            return fmt.Errorf("failed to move mouse to (%f, %f): %w", point.X, point.Y, err)
        }
        
        time.Sleep(time.Duration(delay) * time.Millisecond)
    }
    
    // Add overshoot simulation
    if rand.Float64() < h.config.OvershootProbability {
        overshootX := targetX + (rand.Float64()-0.5)*float64(h.config.OvershootDistance)
        overshootY := targetY + (rand.Float64()-0.5)*float64(h.config.OvershootDistance)
        
        err := chromedp.Run(ctx, chromedp.MouseEvent(chromedp.MouseMove, overshootX, overshootY))
        if err == nil {
            time.Sleep(50 * time.Millisecond)
            chromedp.Run(ctx, chromedp.MouseEvent(chromedp.MouseMove, targetX, targetY))
        }
    }
    
    h.lastPosition = end
    return nil
}
```

### **Phase 4: Human Typing in Go**

```go
// internal/stealth/typing.go
package stealth

import (
    "context"
    "fmt"
    "math/rand"
    "strings"
    "time"
    
    "github.com/chromedp/chromedp"
)

type TypingBehaviorConfig struct {
    CharDelays struct {
        FastTyping   SpeedRange
        NormalTyping SpeedRange
        SlowTyping   SpeedRange
    }
    SpecialCharDelay    SpeedRange
    WordPauseDelay      SpeedRange
    SentencePauseDelay  SpeedRange
    CommonWords         []string
}

type HumanTyping struct {
    config      TypingBehaviorConfig
    typingSpeed string
}

func NewHumanTyping() *HumanTyping {
    return &HumanTyping{
        config: TypingBehaviorConfig{
            CharDelays: struct {
                FastTyping   SpeedRange
                NormalTyping SpeedRange
                SlowTyping   SpeedRange
            }{
                FastTyping:   SpeedRange{Min: 50, Max: 120},
                NormalTyping: SpeedRange{Min: 80, Max: 200},
                SlowTyping:   SpeedRange{Min: 150, Max: 350},
            },
            SpecialCharDelay:   SpeedRange{Min: 100, Max: 300},
            WordPauseDelay:     SpeedRange{Min: 200, Max: 500},
            SentencePauseDelay: SpeedRange{Min: 800, Max: 1500},
            CommonWords: []string{
                "the", "and", "for", "are", "but", "not", "you", "all",
                "can", "had", "her", "was", "one", "our", "out", "day",
                "get", "has", "him", "his", "how", "man", "new", "now",
                "old", "see", "two", "way", "who", "boy",
            },
        },
        typingSpeed: "NormalTyping",
    }
}

func (h *HumanTyping) IsCommonWord(word string) bool {
    word = strings.ToLower(strings.TrimSpace(word))
    for _, commonWord := range h.config.CommonWords {
        if word == commonWord {
            return true
        }
    }
    return false
}

func (h *HumanTyping) GetCharacterDelay(char rune, previousChar rune, word string) int {
    var baseDelay SpeedRange
    
    switch h.typingSpeed {
    case "FastTyping":
        baseDelay = h.config.CharDelays.FastTyping
    case "SlowTyping":
        baseDelay = h.config.CharDelays.SlowTyping
    default:
        baseDelay = h.config.CharDelays.NormalTyping
    }
    
    delay := float64(baseDelay.Min) + rand.Float64()*float64(baseDelay.Max-baseDelay.Min)
    
    // Fast typing for common words
    if h.IsCommonWord(word) {
        delay *= 0.7 // 30% faster
    }
    
    // Special characters take longer
    if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
         (char >= '0' && char <= '9') || char == ' ') {
        delay += rand.Float64() * float64(h.config.SpecialCharDelay.Max)
    }
    
    // Typing combinations that are harder (different hands)
    if previousChar != 0 && h.isDifficultCombination(previousChar, char) {
        delay *= 1.3 // 30% slower
    }
    
    // Random variation
    delay *= 0.8 + rand.Float64()*0.4 // ±20% variation
    
    return int(math.Max(10, math.Round(delay)))
}

func (h *HumanTyping) isDifficultCombination(char1, char2 rune) bool {
    // Simple heuristic: characters typed with different hands
    leftHand := "qwertasdfgzxcvb"
    rightHand := "yuiophjklnm"
    
    char1Lower := strings.ToLower(string(char1))
    char2Lower := strings.ToLower(string(char2))
    
    char1IsLeft := strings.Contains(leftHand, char1Lower)
    char2IsLeft := strings.Contains(leftHand, char2Lower)
    
    // Same hand combinations can be faster
    return char1IsLeft != char2IsLeft
}

func (h *HumanTyping) TypeHumanLike(ctx context.Context, selector, text string) error {
    words := strings.Fields(text)
    
    for wordIndex, word := range words {
        // Add word pause delay
        if wordIndex > 0 {
            wordDelay := h.config.WordPauseDelay.Min + 
                       rand.Intn(h.config.WordPauseDelay.Max-h.config.WordPauseDelay.Min)
            time.Sleep(time.Duration(wordDelay) * time.Millisecond)
        }
        
        var previousChar rune
        for _, char := range word {
            // Simulate typing mistake (2% chance)
            if rand.Float64() < 0.02 {
                // Type wrong character first
                typo := h.generateTypo(char)
                err := chromedp.Run(ctx, chromedp.SendKeys(selector, string(typo), chromedp.ByQuery))
                if err != nil {
                    return fmt.Errorf("failed to type typo character: %w", err)
                }
                
                time.Sleep(100 * time.Millisecond)
                
                // Backspace and correct
                err = chromedp.Run(ctx, chromedp.KeyEvent("\b"))
                if err != nil {
                    return fmt.Errorf("failed to backspace: %w", err)
                }
                
                time.Sleep(50 * time.Millisecond)
            }
            
            // Get character delay
            delay := h.GetCharacterDelay(char, previousChar, word)
            
            // Type the character
            err := chromedp.Run(ctx, chromedp.SendKeys(selector, string(char), chromedp.ByQuery))
            if err != nil {
                return fmt.Errorf("failed to type character %c: %w", char, err)
            }
            
            time.Sleep(time.Duration(delay) * time.Millisecond)
            previousChar = char
        }
        
        // Add space between words (except last word)
        if wordIndex < len(words)-1 {
            err := chromedp.Run(ctx, chromedp.SendKeys(selector, " ", chromedp.ByQuery))
            if err != nil {
                return fmt.Errorf("failed to type space: %w", err)
            }
        }
        
        // Add sentence pause if word ends with sentence-ending punctuation
        if strings.HasSuffix(word, ".") || strings.HasSuffix(word, "!") || strings.HasSuffix(word, "?") {
            sentenceDelay := h.config.SentencePauseDelay.Min + 
                           rand.Intn(h.config.SentencePauseDelay.Max-h.config.SentencePauseDelay.Min)
            time.Sleep(time.Duration(sentenceDelay) * time.Millisecond)
        }
    }
    
    return nil
}

func (h *HumanTyping) generateTypo(char rune) rune {
    // Adjacent keys on QWERTY keyboard - simplified mapping
    adjacentKeys := map[rune][]rune{
        'q': {'w', 'a'},
        'w': {'q', 'e', 's', 'a'},
        'e': {'w', 'r', 'd', 's'},
        'r': {'e', 't', 'f', 'd'},
        't': {'r', 'y', 'g', 'f'},
        'y': {'t', 'u', 'h', 'g'},
        'u': {'y', 'i', 'j', 'h'},
        'i': {'u', 'o', 'k', 'j'},
        'o': {'i', 'p', 'l', 'k'},
        'p': {'o', 'l'},
        // Add more mappings as needed
    }
    
    if adjacent, exists := adjacentKeys[char]; exists && len(adjacent) > 0 {
        return adjacent[rand.Intn(len(adjacent))]
    }
    
    return char
}
```

### **Phase 5: JavaScript Stealth Scripts**

```go
// internal/stealth/navigator.go
package stealth

const NavigatorStealthScript = `
(function() {
    'use strict';
    
    // Phase 2: Navigator Object Patching - Hide automation signatures
    
    // 1. Remove webdriver property
    if (navigator.webdriver !== undefined) {
        delete navigator.webdriver;
    }
    
    // 2. Define webdriver as false (non-configurable)
    Object.defineProperty(navigator, 'webdriver', {
        get: () => false,
        configurable: false
    });
    
    // 3. Remove automation-related properties
    if (window.chrome && window.chrome.runtime) {
        delete window.chrome.runtime.onConnect;
        delete window.chrome.runtime.onMessage;
    }
    
    // 4. Patch the permissions API
    if (navigator.permissions && navigator.permissions.query) {
        const originalQuery = navigator.permissions.query;
        navigator.permissions.query = function(parameters) {
            if (parameters.name === 'notifications') {
                return Promise.resolve({ state: Notification.permission });
            }
            return originalQuery.apply(this, arguments);
        };
    }
    
    // 5. Mock Chrome runtime to prevent detection
    if (!window.chrome) {
        window.chrome = {};
    }
    
    window.chrome.runtime = {
        connect: function() {
            return {
                onMessage: { addListener: function() {}, removeListener: function() {} },
                onDisconnect: { addListener: function() {}, removeListener: function() {} },
                postMessage: function() {}
            };
        },
        sendMessage: function() {},
        onMessage: { addListener: function() {}, removeListener: function() {} }
    };
    
    // 6. Hide Puppeteer-specific properties
    if (window.__puppeteer_evaluation_script__) {
        delete window.__puppeteer_evaluation_script__;
    }
    
    // 7. Override toString methods to hide proxy traces
    const originalToString = Function.prototype.toString;
    Function.prototype.toString = function() {
        if (this === navigator.permissions.query) {
            return 'function query() { [native code] }';
        }
        return originalToString.apply(this, arguments);
    };
    
    console.log('🔒 Stealth navigator patches applied successfully');
})();
`

const CanvasNoiseScript = `
(function() {
    'use strict';
    
    // Phase 10: Canvas Fingerprint Masking - Add subtle noise to canvas rendering
    
    const originalGetContext = HTMLCanvasElement.prototype.getContext;
    const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
    const originalGetImageData = CanvasRenderingContext2D.prototype.getImageData;
    
    // Noise generation function
    function addCanvasNoise(imageData) {
        const data = imageData.data;
        const noiseLevel = 0.1; // Very subtle noise
        
        for (let i = 0; i < data.length; i += 4) {
            // Add very small random variations to RGB values
            const noise = (Math.random() - 0.5) * noiseLevel;
            data[i] = Math.max(0, Math.min(255, data[i] + noise));         // Red
            data[i + 1] = Math.max(0, Math.min(255, data[i + 1] + noise)); // Green  
            data[i + 2] = Math.max(0, Math.min(255, data[i + 2] + noise)); // Blue
            // Alpha channel (data[i + 3]) remains unchanged
        }
        
        return imageData;
    }
    
    // Override getImageData to add noise
    CanvasRenderingContext2D.prototype.getImageData = function(...args) {
        const imageData = originalGetImageData.apply(this, args);
        return addCanvasNoise(imageData);
    };
    
    console.log('🎨 Canvas fingerprint masking applied successfully');
})();
`
```

### **Phase 6: HTTP Server with Stealth API**

```go
// internal/server/handlers.go
package server

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/gorilla/mux"
    "your-project/internal/browser"
    "your-project/internal/stealth"
)

type StealthServer struct {
    browserManager *browser.StealthBrowserManager
    logger         *log.Logger
}

type ExecuteRequest struct {
    Task     string            `json:"task"`
    MaxSteps int              `json:"maxSteps"`
    Options  map[string]interface{} `json:"options"`
    SessionID string           `json:"sessionId,omitempty"`
}

type ExecuteResponse struct {
    Success    bool   `json:"success"`
    ID         string `json:"id"`
    SessionID  string `json:"sessionId"`
    SessionReused bool `json:"session_reused"`
    LiveURL    string `json:"live_url"`
    Message    string `json:"message,omitempty"`
    Error      string `json:"error,omitempty"`
}

func NewStealthServer() *StealthServer {
    config := &stealth.DefaultStealthConfig
    browserManager := browser.NewStealthBrowserManager(config)
    
    return &StealthServer{
        browserManager: browserManager,
        logger:         log.New(os.Stdout, "[StealthServer] ", log.LstdFlags),
    }
}

func (s *StealthServer) ExecuteTask(w http.ResponseWriter, r *http.Request) {
    var req ExecuteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.respondError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    if req.Task == "" {
        s.respondError(w, http.StatusBadRequest, "task is required")
        return
    }
    
    // Generate session ID if not provided
    sessionID := req.SessionID
    if sessionID == "" {
        sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
    }
    
    // Create or reuse stealth session
    session, err := s.browserManager.CreateStealthSession(sessionID)
    if err != nil {
        s.logger.Printf("Failed to create stealth session: %v", err)
        s.respondError(w, http.StatusInternalServerError, 
            fmt.Sprintf("Failed to create browser session: %v", err))
        return
    }
    
    // Generate response URLs
    baseURL := fmt.Sprintf("%s://%s", r.URL.Scheme, r.Host)
    if baseURL == "://" {
        baseURL = "http://localhost:3000" // Default for development
    }
    
    liveURL := fmt.Sprintf("%s/stream/%s?sessionId=%s", baseURL, sessionID, sessionID)
    
    response := ExecuteResponse{
        Success:       true,
        ID:           fmt.Sprintf("task_%d", time.Now().UnixNano()),
        SessionID:    sessionID,
        SessionReused: false, // New session in this example
        LiveURL:      liveURL,
        Message:      fmt.Sprintf("Task started! Browser streaming available at: %s", liveURL),
    }
    
    // Execute task asynchronously
    go s.executeTaskAsync(session, req.Task, req.MaxSteps)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *StealthServer) executeTaskAsync(session *browser.StealthSession, task string, maxSteps int) {
    ctx, cancel := context.WithTimeout(session.Context, 5*time.Minutes)
    defer cancel()
    
    s.logger.Printf("🚀 Executing task asynchronously: %s", task)
    
    // Example task execution - navigate to a URL
    // In a real implementation, you would integrate with browser-use or similar
    err := s.performSampleTask(ctx, session, task)
    if err != nil {
        s.logger.Printf("❌ Task execution failed: %v", err)
        return
    }
    
    s.logger.Printf("✅ Task completed successfully: %s", task)
}

func (s *StealthServer) performSampleTask(ctx context.Context, session *browser.StealthSession, task string) error {
    // Example: Navigate to Google and search
    if strings.Contains(strings.ToLower(task), "google") {
        return chromedp.Run(ctx,
            chromedp.Navigate("https://www.google.com"),
            chromedp.WaitVisible(`input[name="q"]`, chromedp.ByQuery),
            chromedp.ActionFunc(func(ctx context.Context) error {
                // Use human-like typing
                return session.HumanTyping.TypeHumanLike(ctx, `input[name="q"]`, "stealth browser automation")
            }),
            chromedp.Click(`input[value="Google Search"]`, chromedp.ByQuery),
            chromedp.WaitVisible(`#search`, chromedp.ByQuery),
        )
    }
    
    return fmt.Errorf("task not implemented: %s", task)
}

func (s *StealthServer) respondError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *StealthServer) SetupRoutes() *mux.Router {
    r := mux.NewRouter()
    
    r.HandleFunc("/api/browser-use/execute", s.ExecuteTask).Methods("POST")
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    }).Methods("GET")
    
    return r
}
```

### **Phase 7: Main Server Entry Point**

```go
// cmd/stealth-server/main.go
package main

import (
    "log"
    "net/http"
    "os"
    
    "your-project/internal/server"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }
    
    stealthServer := server.NewStealthServer()
    router := stealthServer.SetupRoutes()
    
    log.Printf("🕵️ Stealth Browser Server starting on port %s", port)
    log.Printf("🔗 API endpoint: http://localhost:%s/api/browser-use/execute", port)
    
    if err := http.ListenAndServe(":"+port, router); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}
```

### **Phase 8: Go Module Configuration**

```go
// go.mod
module stealth-browser-go

go 1.21

require (
    github.com/chromedp/chromedp v0.9.3
    github.com/gorilla/mux v1.8.0
    github.com/gorilla/websocket v1.5.0
)

require (
    github.com/chromedp/cdproto v0.0.0-20230802225258-3cf4e6d46a89
    github.com/gobwas/httphead v0.1.0 // indirect
    github.com/gobwas/pool v0.2.1 // indirect
    github.com/gobwas/ws v1.2.1 // indirect
    github.com/josharian/intern v1.0.0 // indirect
    github.com/mailru/easyjson v0.7.7 // indirect
    golang.org/x/sys v0.6.0 // indirect
)
```

## 🚀 Deployment and Usage

### **Docker Configuration**

```dockerfile
# docker/Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stealth-server cmd/stealth-server/main.go

FROM chromedp/headless-shell:latest

# Install required packages
RUN apt-get update && apt-get install -y \
    ca-certificates \
    fonts-liberation \
    libappindicator3-1 \
    libasound2 \
    libatk-bridge2.0-0 \
    libdrm2 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libxss1 \
    lsb-release \
    xdg-utils \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/stealth-server .
COPY scripts/ ./scripts/

EXPOSE 3000

CMD ["./stealth-server"]
```

### **Usage Example**

```bash
# Build and run
go mod tidy
go run cmd/stealth-server/main.go

# Test the API
curl -X POST http://localhost:3000/api/browser-use/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task": "search google for stealth automation",
    "maxSteps": 25
  }'
```

## 🔍 Advantages of Go Implementation

### **Performance Benefits**
- **Compiled Binary**: No runtime interpretation overhead
- **Memory Efficient**: Better garbage collection than Node.js
- **Concurrent**: Native goroutines for parallel sessions
- **Resource Management**: Better control over system resources

### **Deployment Benefits**
- **Single Binary**: Easy deployment without dependencies
- **Cross Platform**: Compile for different architectures
- **Container Friendly**: Smaller container images
- **Cloud Native**: Better integration with Kubernetes

### **Development Benefits**
- **Type Safety**: Compile-time error detection
- **Performance**: Faster execution than interpreted languages
- **Concurrency**: Built-in support for concurrent operations
- **Standard Library**: Rich standard library for networking

### **Stealth Benefits**
- **Lower Detection**: Less common for web automation
- **System Integration**: Better OS-level integration
- **Resource Control**: More precise resource management
- **Custom CDP**: Direct Chrome DevTools Protocol control

## 🛡️ Security and Detection Evasion

### **Go-Specific Advantages**
1. **Memory Safety**: Automatic memory management reduces crashes
2. **Network Stack**: Custom HTTP client implementations
3. **Concurrency**: Independent session isolation
4. **System Calls**: Lower-level system interaction capability
5. **Binary Obfuscation**: Compiled binaries harder to reverse engineer

### **Implementation Recommendations**
1. **Use chromedp**: Provides direct CDP access with better control
2. **Session Isolation**: Separate goroutines and contexts per session
3. **Resource Pooling**: Implement browser instance pooling
4. **Monitoring**: Add comprehensive logging and metrics
5. **Recovery**: Implement robust error handling and recovery

## 📈 Scaling Considerations

### **Horizontal Scaling**
```go
// Kubernetes deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stealth-browser-go
spec:
  replicas: 5
  selector:
    matchLabels:
      app: stealth-browser-go
  template:
    metadata:
      labels:
        app: stealth-browser-go
    spec:
      containers:
      - name: stealth-browser-go
        image: stealth-browser-go:latest
        ports:
        - containerPort: 3000
        env:
        - name: MAX_CONCURRENT_SESSIONS
          value: "3"
        - name: MEMORY_LIMIT_MB
          value: "2048"
        resources:
          limits:
            memory: "4Gi"
            cpu: "2"
          requests:
            memory: "2Gi" 
            cpu: "1"
```

### **Load Balancing**
- **Session Affinity**: Route requests to same instance for session reuse
- **Health Checks**: Implement comprehensive health checking
- **Circuit Breakers**: Add circuit breaker patterns for resilience
- **Metrics**: Expose Prometheus metrics for monitoring

---

*This Go implementation provides a robust, performant, and scalable alternative to the JavaScript stealth system, leveraging Go's strengths in concurrent programming, system resource management, and deployment flexibility.*