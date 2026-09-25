//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch हे URL शी संबंधित पृष्ठाचा मजकूर आणि
	// त्या पृष्ठावर सापडलेल्या URL चा स्लाइस परत करते.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl ही fetcher वापरून
// url पासून सुरू होणारी पाने कमाल depth पर्यंत पुनरावृत्तीने क्रॉल करते.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URL समांतरपणे मिळवा.
	// TODO: तोच URL दोनदा मिळवू नका.
	// या अंमलबजावणीत यांपैकी काहीही केलेले नाही:
	if depth <= 0 {
		return
	}
	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("found: %s %q\n", url, body)
	for _, u := range urls {
		Crawl(u, depth-1, fetcher)
	}
	return
}

func main() {
	Crawl("https://golang.org/", 4, fetcher)
}

// fakeFetcher हा बनवून ठेवलेले परिणाम परत करणारा Fetcher आहे.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher हा आधीच भरलेला fakeFetcher आहे.
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
