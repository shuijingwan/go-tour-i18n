//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter pode ser utilizado em segurança de forma concorrente.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrementa o contador da chave indicada.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Bloquear para que apenas uma goroutine de cada vez possa aceder ao mapa c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value devolve o valor atual do contador da chave indicada.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Bloquear para que apenas uma goroutine de cada vez possa aceder ao mapa c.v.
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
