//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter 可安全地并发使用。
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc 将给定键对应的计数器加一。
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// 加锁，使同一时刻只有一个 goroutine 能访问映射 c.v。
	c.v[key]++
	c.mu.Unlock()
}

// Value 返回给定键对应计数器的当前值。
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// 加锁，使同一时刻只有一个 goroutine 能访问映射 c.v。
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
