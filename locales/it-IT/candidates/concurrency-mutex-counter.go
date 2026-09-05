//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter può essere usato in modo sicuro in concorrenza.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrementa il contatore per la chiave specificata.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Blocca l'accesso in modo che una sola goroutine alla volta possa accedere alla mappa c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value restituisce il valore corrente del contatore per la chiave specificata.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Blocca l'accesso in modo che una sola goroutine alla volta possa accedere alla mappa c.v.
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
