//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter придатний для безпечного конкурентного використання.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc збільшує лічильник для вказаного ключа.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Блокуємо, щоб лише одна goroutine за раз могла звертатися до мапи c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value повертає поточне значення лічильника для вказаного ключа.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Блокуємо, щоб лише одна goroutine за раз могла звертатися до мапи c.v.
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
