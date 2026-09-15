//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter może być bezpiecznie używany współbieżnie.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc zwiększa licznik dla klucza key.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock zapewnia, że tylko jedna goroutine naraz może uzyskać dostęp do mapy c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value zwraca bieżącą wartość licznika dla klucza key.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock zapewnia, że tylko jedna goroutine naraz może uzyskać dostęp do mapy c.v.
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
