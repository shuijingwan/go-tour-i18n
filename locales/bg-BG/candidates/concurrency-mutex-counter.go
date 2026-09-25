//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter може безопасно да се използва конкурентно.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc увеличава брояча за дадения ключ.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Заключваме, така че само една goroutine в даден момент да има достъп до асоциативния масив c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value връща текущата стойност на брояча за дадения ключ.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Заключваме, така че само една goroutine в даден момент да има достъп до асоциативния масив c.v.
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
