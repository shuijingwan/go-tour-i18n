//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// A SafeCounter biztonságosan használható konkurens módon.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Az Inc növeli az adott kulcshoz tartozó számlálót.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Zárold, hogy egyszerre csak egy goroutine férhessen hozzá a c.v leképezéshez.
	c.v[key]++
	c.mu.Unlock()
}

// A Value visszaadja az adott kulcshoz tartozó számláló aktuális értékét.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Zárold, hogy egyszerre csak egy goroutine férhessen hozzá a c.v leképezéshez.
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
