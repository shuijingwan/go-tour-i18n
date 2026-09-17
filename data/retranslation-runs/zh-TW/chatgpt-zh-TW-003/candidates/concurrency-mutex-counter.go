//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter 可以安全地並行使用。
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc 會遞增指定 key 的計數器。
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// 加鎖，確保一次只有一個 goroutine 可以存取映射 c.v。
	c.v[key]++
	c.mu.Unlock()
}

// Value 會回傳指定 key 的計數器目前值。
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// 加鎖，確保一次只有一個 goroutine 可以存取映射 c.v。
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
