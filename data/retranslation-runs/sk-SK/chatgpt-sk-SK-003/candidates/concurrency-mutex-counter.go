//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter možno bezpečne používať súbežne.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc zvýši počítadlo pre daný kľúč.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Zamkni, aby k mape c.v mohol naraz pristupovať iba jeden goroutine.
	c.v[key]++
	c.mu.Unlock()
}

// Value vracia aktuálnu hodnotu počítadla pre daný kľúč.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Zamkni, aby k mape c.v mohol naraz pristupovať iba jeden goroutine.
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
