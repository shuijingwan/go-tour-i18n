//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter kann sicher nebenläufig verwendet werden.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc erhöht den Zähler für den angegebenen Schlüssel.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Sperren, damit jeweils nur eine goroutine auf die Map c.v zugreifen kann.
	c.v[key]++
	c.mu.Unlock()
}

// Value gibt den aktuellen Wert des Zählers für den angegebenen Schlüssel zurück.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Sperren, damit jeweils nur eine goroutine auf die Map c.v zugreifen kann.
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
