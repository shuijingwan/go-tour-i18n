//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i कडे निर्देश करा
	fmt.Println(*p) // पॉइंटरद्वारे i वाचा
	*p = 21         // पॉइंटरद्वारे i सेट करा
	fmt.Println(i)  // i चे नवे मूल्य पाहा

	p = &j         // j कडे निर्देश करा
	*p = *p / 37   // पॉइंटरद्वारे j ला भागा
	fmt.Println(j) // j चे नवे मूल्य पाहा
}
