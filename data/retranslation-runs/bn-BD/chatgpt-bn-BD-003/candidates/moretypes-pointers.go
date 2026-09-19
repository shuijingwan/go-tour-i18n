//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i-কে নির্দেশ করুন
	fmt.Println(*p) // পয়েন্টারের মাধ্যমে i পড়ুন
	*p = 21         // পয়েন্টারের মাধ্যমে i সেট করুন
	fmt.Println(i)  // i-এর নতুন মান দেখুন

	p = &j         // j-কে নির্দেশ করুন
	*p = *p / 37   // পয়েন্টারের মাধ্যমে j ভাগ করুন
	fmt.Println(j) // j-এর নতুন মান দেখুন
}
