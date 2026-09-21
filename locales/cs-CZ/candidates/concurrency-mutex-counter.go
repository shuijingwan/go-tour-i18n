//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter lze bezpečně používat souběžně.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc zvýší čítač pro daný klíč.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Zamkneme, aby k mapě c.v mohla současně přistupovat jen jedna goroutine.
	c.v[key]++
	c.mu.Unlock()
}

// Value vrátí aktuální hodnotu čítače pro daný klíč.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Zamkneme, aby k mapě c.v mohla současně přistupovat jen jedna goroutine.
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
