//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter בטוח לשימוש בו־זמני.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc מגדילה את המונה עבור המפתח הנתון.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// נעלו כך שרק goroutine אחד בכל פעם יוכל לגשת למפה c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value מחזירה את הערך הנוכחי של המונה עבור המפתח הנתון.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// נעלו כך שרק goroutine אחד בכל פעם יוכל לגשת למפה c.v.
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
