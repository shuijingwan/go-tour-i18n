//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// استفادهٔ هم‌زمان از SafeCounter ایمن است.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc شمارندهٔ کلید داده‌شده را افزایش می‌دهد.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// قفل کنید تا در هر لحظه فقط یک goroutine بتواند به نگاشت c.v دسترسی داشته باشد.
	c.v[key]++
	c.mu.Unlock()
}

// Value مقدار کنونی شمارنده را برای کلید داده‌شده برمی‌گرداند.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// قفل کنید تا در هر لحظه فقط یک goroutine بتواند به نگاشت c.v دسترسی داشته باشد.
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
