//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter је безбедан за конкурентну употребу.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc увећава бројач за дати кључ.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Закључајте тако да само један goroutine у датом тренутку може да приступи мапи c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value враћа тренутну вредност бројача за дати кључ.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Закључајте тако да само један goroutine у датом тренутку може да приступи мапи c.v.
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
