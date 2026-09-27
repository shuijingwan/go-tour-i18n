//go:build OMIT

package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeCounteria voi käyttää turvallisesti samanaikaisesti.
type SafeCounter struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc kasvattaa annetun avaimen laskurin arvoa.
func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	// Lukitse, jotta vain yksi goroutine voi kerrallaan käyttää sanakirjaa c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value palauttaa annetun avaimen laskurin nykyisen arvon.
func (c *SafeCounter) Value(key string) int {
	c.mu.Lock()
	// Lukitse, jotta vain yksi goroutine voi kerrallaan käyttää sanakirjaa c.v.
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
