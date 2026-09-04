//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter es seguro para el uso concurrente.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrementa el contador correspondiente a key.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock garantiza que solo una goroutine a la vez pueda acceder al mapa c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value devuelve el valor actual del contador correspondiente a key.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock garantiza que solo una goroutine a la vez pueda acceder al mapa c.v.
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
