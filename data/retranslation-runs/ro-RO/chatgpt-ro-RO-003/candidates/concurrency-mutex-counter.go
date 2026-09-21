//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter poate fi folosit în siguranță concurent.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc incrementează contorul pentru cheia dată.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Blochează astfel încât doar o goroutine la un moment dat să poată accesa map-ul c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value returnează valoarea curentă a contorului pentru cheia dată.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Blochează astfel încât doar o goroutine la un moment dat să poată accesa map-ul c.v.
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
