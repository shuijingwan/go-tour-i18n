//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter एकाच वेळी वापरल्यावरही सुरक्षित राहतो.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc दिलेल्या कीसाठी काउंटरची संख्या वाढवतो.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// लॉक घ्या, जेणेकरून एका वेळी फक्त एक goroutine मॅप c.v मध्ये प्रवेश करू शकेल.
	c.v[key]++
	c.mu.Unlock()
}

// Value दिलेल्या कीसाठी काउंटरचे सध्याचे मूल्य परत करतो.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// लॉक घ्या, जेणेकरून एका वेळी फक्त एक goroutine मॅप c.v मध्ये प्रवेश करू शकेल.
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
