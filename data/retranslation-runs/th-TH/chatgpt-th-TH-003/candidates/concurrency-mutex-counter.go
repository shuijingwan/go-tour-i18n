//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter ปลอดภัยสำหรับการใช้งานพร้อมกัน
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc เพิ่มค่าตัวนับสำหรับคีย์ที่กำหนด
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// ล็อกเพื่อให้ในแต่ละครั้งมี goroutine เพียงตัวเดียวที่เข้าถึงแมป c.v ได้
	c.v[key]++
	c.mu.Unlock()
}

// Value คืนค่าปัจจุบันของตัวนับสำหรับคีย์ที่กำหนด
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// ล็อกเพื่อให้ในแต่ละครั้งมี goroutine เพียงตัวเดียวที่เข้าถึงแมป c.v ได้
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
