//go:build OMIT

package main

import "fmt"

type IPAddr [4]byte

// TODO: IPAddr에 "String() string" 메서드를 추가하세요.

func main() {
	hosts := map[string]IPAddr{
		"loopback":  {127, 0, 0, 1},
		"googleDNS": {8, 8, 8, 8},
	}
	for name, ip := range hosts {
		fmt.Printf("%v: %v\n", name, ip)
	}
}
