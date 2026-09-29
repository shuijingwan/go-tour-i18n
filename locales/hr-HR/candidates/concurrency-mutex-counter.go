//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter je siguran za konkurentnu uporabu.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc povećava brojač za zadani ključ.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Zaključaj kako bi samo jedna goroutine odjednom mogla pristupiti mapi c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value vraća trenutačnu vrijednost brojača za zadani ključ.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Zaključaj kako bi samo jedna goroutine odjednom mogla pristupiti mapi c.v.
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
