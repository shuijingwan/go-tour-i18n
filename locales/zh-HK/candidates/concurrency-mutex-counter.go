//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter 可安全地並行使用。
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc 會遞增指定鍵的計數器。
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// 鎖定，讓每次只有一個 goroutine 可以存取映射 c.v。
	c.v[key]++
	c.mu.Unlock()
}

// Value 會傳回指定鍵的目前計數器值。
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// 鎖定，讓每次只有一個 goroutine 可以存取映射 c.v。
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
