//go:build OMIT

package main

import "fmt"

type IPAddr [4]byte

// TODO: IPAddr-এ একটি "String() string" মেথড যোগ করুন।

func main() {
	hosts := map[string]IPAddr{
		"loopback":  {127, 0, 0, 1},
		"googleDNS": {8, 8, 8, 8},
	}
	for name, ip := range hosts {
		fmt.Printf("%v: %v\n", name, ip)
	}
}
