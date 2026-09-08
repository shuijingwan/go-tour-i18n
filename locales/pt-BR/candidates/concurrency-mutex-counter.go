//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter é seguro para uso concorrente.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrementa o contador para a chave fornecida.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Bloqueie para que apenas uma goroutine por vez possa acessar o map c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value retorna o valor atual do contador para a chave fornecida.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Bloqueie para que apenas uma goroutine por vez possa acessar o map c.v.
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
