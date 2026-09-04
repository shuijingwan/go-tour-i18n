//go:build OMIT

package main

import "fmt"

func main() {
	v := 42 // ¡cámbiame!
	fmt.Printf("v is of type %T\n", v)
}
