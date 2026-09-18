//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// يمكن استخدام SafeCounter بأمان بصورة متسايرة.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// تزيد Inc قيمة العداد للمفتاح المعطى.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// اقفل بحيث لا تستطيع سوى goroutine واحدة في كل مرة الوصول إلى الخريطة c.v.
	c.v[key]++
	c.mu.Unlock()
}

// تعيد Value القيمة الحالية للعداد للمفتاح المعطى.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// اقفل بحيث لا تستطيع سوى goroutine واحدة في كل مرة الوصول إلى الخريطة c.v.
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
