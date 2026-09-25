//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append nil സ്ലൈസുകളിലും പ്രവർത്തിക്കുന്നു.
	s = append(s, 0)
	printSlice(s)

	// ആവശ്യത്തിന് സ്ലൈസ് വളരുന്നു.
	s = append(s, 1)
	printSlice(s)

	// ഒരേസമയം ഒന്നിലധികം ഘടകങ്ങൾ ചേർക്കാം.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
