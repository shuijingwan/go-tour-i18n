//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter は並行に使用しても安全です。
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc は指定したキーのカウンタを 1 増やします。
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// 一度に 1 つの goroutine だけがマップ c.v にアクセスできるようロックします。
	c.v[key]++
	c.mu.Unlock()
}

// Value は指定したキーのカウンタの現在値を返します。
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// 一度に 1 つの goroutine だけがマップ c.v にアクセスできるようロックします。
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
