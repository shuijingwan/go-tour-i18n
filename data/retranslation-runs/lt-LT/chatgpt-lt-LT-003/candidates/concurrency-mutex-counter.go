//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter saugu naudoti konkurentiškai.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc padidina nurodyto rakto skaitiklį.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Užrakinkite, kad vienu metu tik viena goroutine galėtų pasiekti žodyną c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value grąžina dabartinę nurodyto rakto skaitiklio reikšmę.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Užrakinkite, kad vienu metu tik viena goroutine galėtų pasiekti žodyną c.v.
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
