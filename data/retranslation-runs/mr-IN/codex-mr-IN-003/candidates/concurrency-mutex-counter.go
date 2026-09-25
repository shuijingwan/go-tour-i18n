//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter चा concurrent वापर सुरक्षित आहे.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc दिलेल्या key साठी counter वाढवते.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock करा, म्हणजे एका वेळी फक्त एक goroutine c.v मॅप वापरू शकेल.
	c.v[key]++
	c.mu.Unlock()
}

// Value दिलेल्या key साठी counter चे सध्याचे मूल्य परत करते.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock करा, म्हणजे एका वेळी फक्त एक goroutine c.v मॅप वापरू शकेल.
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
