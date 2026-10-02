//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // ወደ i ጠቁም
	fmt.Println(*p) // ጠቋሚውን በመጠቀም i ን አንብብ
	*p = 21         // ጠቋሚውን በመጠቀም i ን አዘጋጅ
	fmt.Println(i)  // የ i አዲሱን እሴት ተመልከት

	p = &j         // ወደ j ጠቁም
	*p = *p / 37   // ጠቋሚውን በመጠቀም j ን ክፈል
	fmt.Println(j) // የ j አዲሱን እሴት ተመልከት
}
