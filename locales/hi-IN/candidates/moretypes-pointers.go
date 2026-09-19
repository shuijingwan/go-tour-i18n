//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i की ओर पॉइंट करें
	fmt.Println(*p) // पॉइंटर के जरिए i पढ़ें
	*p = 21         // पॉइंटर के जरिए i सेट करें
	fmt.Println(i)  // i की नई वैल्यू देखें

	p = &j         // j की ओर पॉइंट करें
	*p = *p / 37   // पॉइंटर के जरिए j को विभाजित करें
	fmt.Println(j) // j की नई वैल्यू देखें
}
