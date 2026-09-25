//go:build OMIT

package main

import "fmt"

const (
	// 1 ബിറ്റ് 100 സ്ഥാനങ്ങൾ ഇടത്തേക്ക് ഷിഫ്റ്റ് ചെയ്ത് ഒരു വളരെ വലിയ സംഖ്യ സൃഷ്ടിക്കുക.
	// മറ്റൊരു രീതിയിൽ പറഞ്ഞാൽ, 1-ന് പിന്നാലെ 100 പൂജ്യങ്ങളുള്ള ബൈനറി സംഖ്യ.
	Big = 1 << 100
	// അത് വീണ്ടും 99 സ്ഥാനങ്ങൾ വലത്തേക്ക് ഷിഫ്റ്റ് ചെയ്യുക; അപ്പോൾ 1<<1, അഥവാ 2 ലഭിക്കും.
	Small = Big >> 99
)

func needInt(x int) int { return x*10 + 1 }
func needFloat(x float64) float64 {
	return x * 0.1
}

func main() {
	fmt.Println(needInt(Small))
	fmt.Println(needFloat(Small))
	fmt.Println(needFloat(Big))
}
