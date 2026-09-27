//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i તરફ નિર્દેશ કરો
	fmt.Println(*p) // પોઇન્ટર દ્વારા i વાંચો
	*p = 21         // પોઇન્ટર દ્વારા i નું મૂલ્ય બદલો
	fmt.Println(i)  // i નું નવું મૂલ્ય જુઓ

	p = &j         // j તરફ નિર્દેશ કરો
	*p = *p / 37   // પોઇન્ટર દ્વારા j ને ભાગો
	fmt.Println(j) // j નું નવું મૂલ્ય જુઓ
}
