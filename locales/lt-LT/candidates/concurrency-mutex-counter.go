//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter galima saugiai naudoti konkurentinio vykdymo metu.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc padidina nurodyto rakto skaitiklio reikšmę.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Užrakinkite, kad vienu metu žodyną c.v galėtų pasiekti tik viena goroutine.
	c.v[key]++
	c.mu.Unlock()
}

// Value grąžina dabartinę nurodyto rakto skaitiklio reikšmę.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Užrakinkite, kad vienu metu žodyną c.v galėtų pasiekti tik viena goroutine.
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
