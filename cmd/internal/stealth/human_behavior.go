package stealth

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

// HumanBehavior provides methods for simulating human-like interactions
type HumanBehavior struct {
	config *StealthConfig
}

// NewHumanBehavior creates a new HumanBehavior instance
func NewHumanBehavior(config *StealthConfig) *HumanBehavior {
	return &HumanBehavior{
		config: config,
	}
}

// Point represents a 2D coordinate
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GenerateHumanDelay creates realistic delays between actions
func (h *HumanBehavior) GenerateHumanDelay() time.Duration {
	return h.config.GetRandomActionDelay()
}

// GeneratePageLoadDelay creates realistic page load waiting times
func (h *HumanBehavior) GeneratePageLoadDelay() time.Duration {
	return h.config.GetRandomPageLoadDelay()
}

// GenerateElementWaitDelay creates realistic element waiting times
func (h *HumanBehavior) GenerateElementWaitDelay() time.Duration {
	return h.config.GetRandomElementWait()
}

// GenerateTypingDelays creates human-like typing patterns
func (h *HumanBehavior) GenerateTypingDelays(text string) []time.Duration {
	delays := make([]time.Duration, len(text))
	words := splitIntoWords(text)

	charIndex := 0
	for _, word := range words {
		wordSpeed := h.getWordTypingSpeed(word)

		for _, char := range word {
			baseDelay := h.getCharacterDelay(char, wordSpeed)

			// Add mistake simulation
			if rand.Float64() < h.config.TypingConfig.MistakeChance {
				// Longer delay for mistake + correction
				baseDelay = baseDelay * 3
			}

			delays[charIndex] = baseDelay
			charIndex++
		}

		// Add space delay if not last word
		if charIndex < len(text) && text[charIndex] == ' ' {
			delays[charIndex] = h.getWordPause()
			charIndex++
		}
	}

	return delays
}

// GenerateBezierPath creates natural mouse movement paths
func (h *HumanBehavior) GenerateBezierPath(start, end Point) []Point {
	distance := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))
	steps := h.config.MouseConfig.MinSteps + int(distance/20)

	if steps > h.config.MouseConfig.MaxSteps {
		steps = h.config.MouseConfig.MaxSteps
	}

	// Create control points for natural curve
	controlPoint1 := Point{
		X: start.X + (end.X-start.X)*0.25 + (rand.Float64()-0.5)*50*h.config.MouseConfig.CurveStrength,
		Y: start.Y + (end.Y-start.Y)*0.25 + (rand.Float64()-0.5)*50*h.config.MouseConfig.CurveStrength,
	}

	controlPoint2 := Point{
		X: start.X + (end.X-start.X)*0.75 + (rand.Float64()-0.5)*50*h.config.MouseConfig.CurveStrength,
		Y: start.Y + (end.Y-start.Y)*0.75 + (rand.Float64()-0.5)*50*h.config.MouseConfig.CurveStrength,
	}

	points := make([]Point, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		points[i] = h.cubicBezier(t, start, controlPoint1, controlPoint2, end)
	}

	return points
}

// GenerateMouseSpeed calculates human-like mouse movement speed
func (h *HumanBehavior) GenerateMouseSpeed(distance float64) time.Duration {
	var baseSpeed time.Duration

	// Speed varies based on distance
	if distance < 100 {
		baseSpeed = 5 * time.Millisecond
	} else if distance < 300 {
		baseSpeed = 10 * time.Millisecond
	} else {
		baseSpeed = 20 * time.Millisecond
	}

	// Add speed variation
	variation := 1 + (rand.Float64()-0.5)*h.config.MouseConfig.SpeedVariation
	return time.Duration(float64(baseSpeed) * variation)
}

// ShouldAddMousePause determines if a pause should be added during mouse movement
func (h *HumanBehavior) ShouldAddMousePause() bool {
	return rand.Float64() < h.config.MouseConfig.PauseProbability
}

// ShouldAddOvershoot determines if mouse overshoot should be simulated
func (h *HumanBehavior) ShouldAddOvershoot() bool {
	return rand.Float64() < h.config.MouseConfig.OvershootProbability
}

// GenerateOvershoot creates overshoot coordinates
func (h *HumanBehavior) GenerateOvershoot(target Point) Point {
	angle := rand.Float64() * 2 * math.Pi
	distance := float64(h.config.MouseConfig.OvershootDistance) * (0.5 + rand.Float64()*0.5)

	return Point{
		X: target.X + math.Cos(angle)*distance,
		Y: target.Y + math.Sin(angle)*distance,
	}
}

// GenerateScrollPattern creates human-like scrolling behavior
func (h *HumanBehavior) GenerateScrollPattern(totalScroll float64) []ScrollAction {
	var actions []ScrollAction

	remaining := totalScroll
	for remaining > 0 {
		// Variable scroll amounts
		scrollAmount := 100 + rand.Float64()*200
		if scrollAmount > remaining {
			scrollAmount = remaining
		}

		actions = append(actions, ScrollAction{
			Amount: scrollAmount,
			Delay:  50*time.Millisecond + time.Duration(rand.Float64()*100)*time.Millisecond,
		})

		remaining -= scrollAmount

		// Occasional pause during scrolling
		if rand.Float64() < 0.3 {
			actions = append(actions, ScrollAction{
				Amount: 0,
				Delay:  200*time.Millisecond + time.Duration(rand.Float64()*300)*time.Millisecond,
			})
		}
	}

	return actions
}

// ScrollAction represents a scrolling action
type ScrollAction struct {
	Amount float64
	Delay  time.Duration
}

// Helper methods

func (h *HumanBehavior) cubicBezier(t float64, p0, p1, p2, p3 Point) Point {
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

func (h *HumanBehavior) getWordTypingSpeed(word string) string {
	// Common words are typed faster
	commonWords := []string{
		"the", "and", "for", "are", "but", "not", "you", "all",
		"can", "had", "her", "was", "one", "our", "out", "day",
		"get", "has", "him", "his", "how", "man", "new", "now",
		"old", "see", "two", "way", "who", "boy", "did", "its",
	}

	wordLower := strings.ToLower(word)
	for _, common := range commonWords {
		if wordLower == common {
			return "FastTyping"
		}
	}

	// Technical/unusual words are typed slower
	if len(word) > 8 {
		return "SlowTyping"
	}

	return "NormalTyping"
}

func (h *HumanBehavior) getCharacterDelay(char rune, speed string) time.Duration {
	var speedRange SpeedRange

	switch speed {
	case "FastTyping":
		speedRange = h.config.TypingConfig.FastTyping
	case "SlowTyping":
		speedRange = h.config.TypingConfig.SlowTyping
	default:
		speedRange = h.config.TypingConfig.NormalTyping
	}

	baseDelay := speedRange.Min + rand.Intn(speedRange.Max-speedRange.Min)

	// Special characters take longer
	if !isAlphanumeric(char) {
		baseDelay = int(float64(baseDelay) * 1.3)
	}

	return time.Duration(baseDelay) * time.Millisecond
}

func (h *HumanBehavior) getWordPause() time.Duration {
	min := h.config.TypingConfig.WordPause.Min
	max := h.config.TypingConfig.WordPause.Max
	return time.Duration(min+rand.Intn(max-min)) * time.Millisecond
}

func splitIntoWords(text string) []string {
	return strings.Fields(text)
}

func isAlphanumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == ' '
}
