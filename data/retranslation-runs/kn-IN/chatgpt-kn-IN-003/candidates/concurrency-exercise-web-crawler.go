//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch ಫಂಕ್ಷನ್ URL ನ ವಿಷಯವನ್ನು ಹಾಗೂ
	// ಆ ಪುಟದಲ್ಲಿ ಕಂಡುಬಂದ URL ಗಳ ಸ್ಲೈಸ್ ಅನ್ನು ಹಿಂತಿರುಗಿಸುತ್ತದೆ.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl ಫಂಕ್ಷನ್ fetcher ಅನ್ನು ಬಳಸಿ ಪುನರಾವರ್ತಿತವಾಗಿ
// url ನಿಂದ ಆರಂಭಿಸಿ ಗರಿಷ್ಠ depth ಆಳದವರೆಗೆ ಪುಟಗಳನ್ನು ಕ್ರಾಲ್ ಮಾಡುತ್ತದೆ.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URL ಗಳನ್ನು ಸಮಾನಾಂತರವಾಗಿ ಪಡೆದುಕೊಳ್ಳಿ.
	// TODO: ಒಂದೇ URL ಅನ್ನು ಎರಡು ಬಾರಿ ಪಡೆಯಬೇಡಿ.
	// ಈ ಅನುಷ್ಠಾನವು ಮೇಲಿನ ಎರಡೂ ಕೆಲಸಗಳನ್ನು ಮಾಡುವುದಿಲ್ಲ:
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

// fakeFetcher ಎನ್ನುವುದು ಪೂರ್ವನಿರ್ಧರಿತ ಫಲಿತಾಂಶಗಳನ್ನು ಹಿಂತಿರುಗಿಸುವ Fetcher ಆಗಿದೆ.
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

// fetcher ಎನ್ನುವುದು ಮೊದಲೇ ಡೇಟಾ ತುಂಬಿರುವ fakeFetcher ಆಗಿದೆ.
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
