//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// Tüüpi SafeCounter saab konkurrentselt ohutult kasutada.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc suurendab võtmele key vastavat loendurit.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lukusta, et korraga pääseks ainult üks goroutine kujutisele c.v ligi.
	c.v[key]++
	c.mu.Unlock()
}

// Value tagastab võtmele key vastava loenduri praeguse väärtuse.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lukusta, et korraga pääseks ainult üks goroutine kujutisele c.v ligi.
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
