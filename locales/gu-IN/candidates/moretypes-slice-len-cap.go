//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// સ્લાઇસની લંબાઈ શૂન્ય કરવા તેનું સ્લાઇસિંગ કરો.
	s = s[:0]
	printSlice(s)

	// તેની લંબાઈ વધારો.
	s = s[:4]
	printSlice(s)

	// તેના શરૂઆતનાં બે મૂલ્યો દૂર કરો.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
