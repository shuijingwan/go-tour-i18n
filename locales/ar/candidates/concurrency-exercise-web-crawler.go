//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// تعيد Fetch محتوى URL و
	// شريحة من عناوين URL الموجودة في تلك الصفحة.
	Fetch(url string) (body string, urls []string, err error)
}

// تستخدم Crawl ‏fetcher للزحف تكراريًا إلى
// الصفحات بدءًا من url، حتى حد أقصى مقداره depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: اجلب عناوين URL بالتوازي.
	// TODO: لا تجلب URL نفسه مرتين.
	// هذا التنفيذ لا يفعل أيًا من الأمرين:
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

// fakeFetcher هو Fetcher يعيد نتائج ثابتة مسبقًا.
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

// fetcher هو fakeFetcher مهيأ بالبيانات.
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
