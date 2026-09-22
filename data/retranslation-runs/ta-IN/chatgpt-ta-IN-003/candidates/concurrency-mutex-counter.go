//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounter-ஐ உடன்நிகழ்வாகப் பாதுகாப்பாகப் பயன்படுத்தலாம்.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// கொடுக்கப்பட்ட key-க்கான எண்ணியை Inc அதிகரிக்கிறது.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lock செய்து, ஒரே ஒரு goroutine மட்டுமே ஒரே நேரத்தில் c.v இணைபடத்தை அணுகுமாறு செய்க.
	c.v[key]++
	c.mu.Unlock()
}

// கொடுக்கப்பட்ட key-க்கான எண்ணியின் தற்போதைய மதிப்பை Value திருப்பித் தருகிறது.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lock செய்து, ஒரே ஒரு goroutine மட்டுமே ஒரே நேரத்தில் c.v இணைபடத்தை அணுகுமாறு செய்க.
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
