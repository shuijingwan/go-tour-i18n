//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter का समवर्ती रूप से उपयोग सुरक्षित है।
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc दिए गए key के काउंटर को बढ़ाता है।
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock करें ताकि एक समय में केवल एक goroutine मैप c.v को एक्सेस कर सके।
	c.v[key]++
	c.mu.Unlock()
}

// Value दिए गए key के काउंटर की वर्तमान वैल्यू लौटाता है।
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock करें ताकि एक समय में केवल एक goroutine मैप c.v को एक्सेस कर सके।
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
