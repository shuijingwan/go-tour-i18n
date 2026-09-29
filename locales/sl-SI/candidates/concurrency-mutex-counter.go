//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter je varen za sočasno uporabo.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc poveča števec za podani ključ.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Zakleni, da lahko do slovarja c.v naenkrat dostopa samo ena goroutine.
	c.v[key]++
	c.mu.Unlock()
}

// Value vrne trenutno vrednost števca za podani ključ.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Zakleni, da lahko do slovarja c.v naenkrat dostopa samo ena goroutine.
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
