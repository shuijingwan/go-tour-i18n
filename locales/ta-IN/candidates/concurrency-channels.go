//go:build OMIT

package main

import "fmt"

func sum(s []int, c chan int) {
	sum := 0
	for _, v := range s {
		sum += v
	}
	c <- sum // கூட்டுத்தொகையை c-க்கு அனுப்பு
}

func main() {
	s := []int{7, 2, 8, -9, 4, 0}

	c := make(chan int)
	go sum(s[:len(s)/2], c)
	go sum(s[len(s)/2:], c)
	x, y := <-c, <-c // c-இலிருந்து பெறு

	fmt.Println(x, y, x+y)
}
