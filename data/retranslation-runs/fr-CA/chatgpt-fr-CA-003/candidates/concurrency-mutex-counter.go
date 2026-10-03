//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter est sûr pour une utilisation concurrente.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrémente le compteur pour la clé donnée.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Verrouiller pour qu’une seule goroutine à la fois puisse accéder à la table associative c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value retourne la valeur courante du compteur pour la clé donnée.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Verrouiller pour qu’une seule goroutine à la fois puisse accéder à la table associative c.v.
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
