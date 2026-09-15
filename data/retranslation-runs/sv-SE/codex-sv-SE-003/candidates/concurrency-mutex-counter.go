//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter är säker att använda samtidigt.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc ökar räknaren för den angivna nyckeln.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lås så att endast en goroutine åt gången kan komma åt mappen c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value returnerar räknarens aktuella värde för den angivna nyckeln.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lås så att endast en goroutine åt gången kan komma åt mappen c.v.
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
