//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i کی طرف پوائنٹ کریں
	fmt.Println(*p) // پوائنٹر کے ذریعے i پڑھیں
	*p = 21         // پوائنٹر کے ذریعے i مقرر کریں
	fmt.Println(i)  // i کی نئی قدر دیکھیں

	p = &j         // j کی طرف پوائنٹ کریں
	*p = *p / 37   // پوائنٹر کے ذریعے j کو divide کریں
	fmt.Println(j) // j کی نئی قدر دیکھیں
}
