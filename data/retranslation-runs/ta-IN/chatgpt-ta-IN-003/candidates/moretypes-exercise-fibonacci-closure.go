//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci என்பது ஒரு செயற்கூற்றைத் திருப்பித் தரும் செயற்கூறு;
// அந்த செயற்கூறு int மதிப்பைத் திருப்பித் தருகிறது.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
