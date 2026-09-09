//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter eşzamanlı kullanım için güvenlidir.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc, belirtilen key için sayacı artırır.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// c.v eşlemesine aynı anda yalnızca bir goroutine erişebilsin diye kilitleyin.
	c.v[key]++
	c.mu.Unlock()
}

// Value, belirtilen key için sayacın geçerli değerini döndürür.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// c.v eşlemesine aynı anda yalnızca bir goroutine erişebilsin diye kilitleyin.
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
