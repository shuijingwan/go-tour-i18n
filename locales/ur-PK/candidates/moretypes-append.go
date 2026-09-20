//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append، nil سلائسز پر کام کرتا ہے۔
	s = append(s, 0)
	printSlice(s)

	// ضرورت کے مطابق سلائس بڑھتی ہے۔
	s = append(s, 1)
	printSlice(s)

	// ہم ایک وقت میں ایک سے زیادہ element شامل کر سکتے ہیں۔
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
