//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter সমসাময়িকভাবে ব্যবহার করা নিরাপদ।
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc প্রদত্ত key-এর কাউন্টার বাড়ায়।
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock করুন, যাতে এক সময়ে কেবল একটি goroutine ম্যাপ c.v অ্যাক্সেস করতে পারে।
	c.v[key]++
	c.mu.Unlock()
}

// Value প্রদত্ত key-এর কাউন্টারের বর্তমান মান রিটার্ন করে।
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock করুন, যাতে এক সময়ে কেবল একটি goroutine ম্যাপ c.v অ্যাক্সেস করতে পারে।
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
