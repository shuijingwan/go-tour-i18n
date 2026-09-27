//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i ਵੱਲ ਇਸ਼ਾਰਾ ਕਰੋ
	fmt.Println(*p) // ਪੌਇੰਟਰ ਰਾਹੀਂ i ਨੂੰ ਪੜ੍ਹੋ
	*p = 21         // ਪੌਇੰਟਰ ਰਾਹੀਂ i ਦਾ ਮੁੱਲ ਸੈੱਟ ਕਰੋ
	fmt.Println(i)  // i ਦਾ ਨਵਾਂ ਮੁੱਲ ਵੇਖੋ

	p = &j         // j ਵੱਲ ਇਸ਼ਾਰਾ ਕਰੋ
	*p = *p / 37   // ਪੌਇੰਟਰ ਰਾਹੀਂ j ਨੂੰ ਭਾਗ ਦਿਓ
	fmt.Println(j) // j ਦਾ ਨਵਾਂ ਮੁੱਲ ਵੇਖੋ
}
