//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter በተጓዳኝነት ለመጠቀም ደህና ነው።
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc ለተሰጠው key ቆጣሪውን ይጨምራል።
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock አድርግ፣ በአንድ ጊዜ አንድ goroutine ብቻ ካርታ c.v ላይ እንዲደርስ።
	c.v[key]++
	c.mu.Unlock()
}

// Value ለተሰጠው key የቆጣሪውን የአሁኑን እሴት ይመልሳል።
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock አድርግ፣ በአንድ ጊዜ አንድ goroutine ብቻ ካርታ c.v ላይ እንዲደርስ።
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
