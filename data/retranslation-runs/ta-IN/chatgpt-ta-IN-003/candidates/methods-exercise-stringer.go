//go:build OMIT

package main

import "fmt"

type IPAddr [4]byte

// TODO: "String() string" வழிமுறையை IPAddr-க்கு சேர்க்கவும்.

func main() {
	hosts := map[string]IPAddr{
		"loopback":  {127, 0, 0, 1},
		"googleDNS": {8, 8, 8, 8},
	}
	for name, ip := range hosts {
		fmt.Printf("%v: %v\n", name, ip)
	}
}
