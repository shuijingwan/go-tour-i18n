//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter kan brukes trygt samtidig av flere goroutiner.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc øker telleren for den angitte nøkkelen.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lås slik at bare én goroutine om gangen kan få tilgang til map-en c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value returnerer tellerens gjeldende verdi for den angitte nøkkelen.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lås slik at bare én goroutine om gangen kan få tilgang til map-en c.v.
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
