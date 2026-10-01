//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter-ды конкурентті түрде пайдалану қауіпсіз.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc берілген кілттің санауышын арттырады.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// c.v сөздігіне бір уақытта тек бір goroutine қатынай алатындай құлыптайды.
	c.v[key]++
	c.mu.Unlock()
}

// Value берілген кілт санауышының ағымдағы мәнін қайтарады.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// c.v сөздігіне бір уақытта тек бір goroutine қатынай алатындай құлыптайды.
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
