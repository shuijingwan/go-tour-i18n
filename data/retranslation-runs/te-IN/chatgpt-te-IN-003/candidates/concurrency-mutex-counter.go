//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter ను ఏకకాలంలో సురక్షితంగా ఉపయోగించవచ్చు.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc ఇచ్చిన key కు సంబంధించిన కౌంటర్‌ను పెంచుతుంది.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// ఒకేసారి ఒక్క goroutine మాత్రమే మ్యాప్ c.v ను యాక్సెస్ చేయగలిగేలా Lock చేయండి.
	c.v[key]++
	c.mu.Unlock()
}

// Value ఇచ్చిన key కు సంబంధించిన కౌంటర్ ప్రస్తుత విలువను రిటర్న్ చేస్తుంది.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// ఒకేసారి ఒక్క goroutine మాత్రమే మ్యాప్ c.v ను యాక్సెస్ చేయగలిగేలా Lock చేయండి.
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
