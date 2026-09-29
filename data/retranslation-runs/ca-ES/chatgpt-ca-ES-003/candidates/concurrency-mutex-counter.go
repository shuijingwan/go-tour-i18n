//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter es pot utilitzar de manera segura concurrentment.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrementa el comptador per a la clau indicada.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Bloqueja perquè només una goroutine alhora pugui accedir al mapa c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value retorna el valor actual del comptador per a la clau indicada.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Bloqueja perquè només una goroutine alhora pugui accedir al mapa c.v.
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
