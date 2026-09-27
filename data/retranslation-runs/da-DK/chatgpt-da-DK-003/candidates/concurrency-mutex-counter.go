//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter kan bruges sikkert ved samtidig adgang.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc øger tælleren for den angivne nøgle.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lås, så kun én goroutine ad gangen kan tilgå mappet c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value returnerer tællerens aktuelle værdi for den angivne nøgle.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lås, så kun én goroutine ad gangen kan tilgå mappet c.v.
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
