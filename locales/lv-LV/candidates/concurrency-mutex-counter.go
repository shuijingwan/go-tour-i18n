//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter var droši lietot laiksakritīgi.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc palielina skaitītāja vērtību atslēgai key.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Bloķēt, lai vārdnīcai c.v vienlaikus varētu piekļūt tikai viena goroutine.
	c.v[key]++
	c.mu.Unlock()
}

// Value atgriež skaitītāja pašreizējo vērtību atslēgai key.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Bloķēt, lai vārdnīcai c.v vienlaikus varētu piekļūt tikai viena goroutine.
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
