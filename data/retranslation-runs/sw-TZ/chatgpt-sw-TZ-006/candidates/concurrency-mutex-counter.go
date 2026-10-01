//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter ni salama kutumiwa katika utekelezaji wa majukumu kwa kupishana au sambamba.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc huongeza kaunta ya key iliyotolewa.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Funga ili goroutine moja tu kwa wakati aweze kufikia ramani c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value hurejesha thamani ya sasa ya kaunta ya key iliyotolewa.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Funga ili goroutine moja tu kwa wakati aweze kufikia ramani c.v.
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
