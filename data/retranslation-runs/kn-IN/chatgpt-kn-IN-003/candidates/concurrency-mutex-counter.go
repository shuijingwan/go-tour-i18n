//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter ಅನ್ನು ಸಹವರ್ತಿ ಕಾರ್ಯನಿರ್ವಹಣೆಯಲ್ಲಿಯೂ ಸುರಕ್ಷಿತವಾಗಿ ಬಳಸಬಹುದು.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc ನೀಡಲಾದ ಕೀಗಾಗಿ ಕೌಂಟರ್‌ನ ಮೌಲ್ಯವನ್ನು ಒಂದರಷ್ಟು ಹೆಚ್ಚಿಸುತ್ತದೆ.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// ಒಂದು ಸಮಯದಲ್ಲಿ ಒಂದೇ goroutine ಮಾತ್ರ c.v ಮ್ಯಾಪ್ ಅನ್ನು ಪ್ರವೇಶಿಸುವಂತೆ ಲಾಕ್ ಮಾಡಿ.
	c.v[key]++
	c.mu.Unlock()
}

// Value ನೀಡಲಾದ ಕೀಗಾಗಿ ಕೌಂಟರ್‌ನ ಪ್ರಸ್ತುತ ಮೌಲ್ಯವನ್ನು ಹಿಂತಿರುಗಿಸುತ್ತದೆ.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// ಒಂದು ಸಮಯದಲ್ಲಿ ಒಂದೇ goroutine ಮಾತ್ರ c.v ಮ್ಯಾಪ್ ಅನ್ನು ಪ್ರವೇಶಿಸುವಂತೆ ಲಾಕ್ ಮಾಡಿ.
	defer c.mu.Unlock()
	return c.v[key]
}

func main() {
	c := SafeCounter{v: make(map[string]int)}
	for i := 0; i < 1000; i++ {
		go c.Inc("somekey")
	}

	time.Sleep(time.Second)
	fmt.Println(c.Value("somekey"))
}
