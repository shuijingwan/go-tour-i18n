# gu-IN TranslationUnit 定向 re-QC：Example 107–108

- Reviewer identity: original long-term independent gu-IN ChatGPT GPT-5.6 Sol High; never involved in gu-IN Generation, revision or replacement.
- Rubric: translation-quality/v1; revision Reviewer invocation covers only the two Example Units named below.
- Locale: gu-IN (ગુજરાતી). Previous full Snapshot: `20260927-gu-IN-qc-001`; current full Snapshot: `20260927-gu-IN-qc-002`.
- Revision batch: `chatgpt-gu-IN-005`; both automatic validations passed.
- CURRENT Reviewer ZIP: `/tmp/gu-IN-20260927-gu-IN-qc-002-re-qc-example-2.zip`.
- ZIP SHA-256: `aada4c47d8d5ab802181fd487d99815b29048b0e2e7c9dd8db70d767dc407417`.
- Bundle identity: `d56ae1f57b0cfba2adaecc2266e41279f5165953e1288aeb2b362472f7edd3fc`.
- Snapshot manifest SHA-256: `746f1ee72a7a9635e31f7ce1d03df15e6979ab31b117087353c7439fdca9b236`.
- Batch manifest SHA-256: `6a1b033ec2ab1b5e25e3edda628bee335fab4942390ba0f1891064e033c6bb24`.
- Complete formal glossary SHA-256: `9e9698f4281fb2be23f5532d3e6804f85eec5dbc25759044ce8b233d7e85c377`.
- Inventory: 14/14 declared ZIP members SHA-256/size PASS, plus manifest.json, ZIP CRC PASS.
- Repository `quality-check reviewer-bundle-check`: CURRENT; `quality-check preflight`: incremental, existing six Page results and 114 historic A carry-forward, only #107/#108 pending.
- All full English originals, full Gujarati revised candidates, formal previous findings, relevant revision inputs, snapshot-selected identities and both passing validation receipts cross-checked.
- Scope boundary: historical 120 effective A untouched and not re-reviewed. No candidate editing, formal QC record, finalization, promotion, Production or Git operation.

## Individual ratings

| Stable index | Complete TranslationUnit | Previous rating | Current rating | Validation | Finding |
| ---: | --- | :---: | :---: | :---: | --- |
| 107 | `example:concurrency/exercise-equivalent-binary-trees.go` | B | **A** | passed | Original B resolved; no new finding |
| 108 | `example:concurrency/exercise-web-crawler.go` | B | **A** | passed | Original B resolved; no new finding |

## Full-unit independent language and machine-semantic review

### #107 `example:concurrency/exercise-equivalent-binary-trees.go` — A (previous B)

- Previous formal finding: The Gujarati teaching comments redundantly use both બંને ટ્રીમાં and t1 અને t2 માં, obscuring the relationship between trees and values. Revise only these two comments to express clearly that Same checks whether t1 and t2 contain the same values; preserve identifiers, code and protected structure.
- Original source SHA-256: `184cbbd1e662b64de6e6928d4af6f2469c7bccdbc9a8b5f477c1b694b9ab02b1`.
- Revised candidate SHA-256: `ac9e77e4ed3b5f8ff2d1aee62823d5bcb112c595b20a6db5025ab938be325805`.
- Automatic validation SHA-256: `4e67802466adae123e676dbd029044ff803da392d0d7c3cbda5e9b78180358df` (passed).
- Relevant revision input SHA-256: `7a7b9604ab994ccecd30d445793424bc7db50ec6b4b77d9d082d42f1e9cc5685`.
- Revision batch / attempt: `chatgpt-gu-IN-005` / 1.
- Machine-protected comparison: all 11 non-comment lines byte-identical; 5 comment delimiters/indent prefixes identical; Go build directive byte-identical.
- Independent reviewer verdict: The revised two-line Same comment now connects both named trees t1 and t2 directly to the values they contain: "Same તપાસે છે કે / t1 અને t2 નામનાં બંને ટ્રીમાં એકસરખાં મૂલ્યો છે કે નહીં." This resolves the previous redundant locative wording without changing the mathematical comparison. Walk still accurately traverses t and sends its values to channel ch. Both full file and protected structure are intact; no new issue.

**Complete English Go original:**

```go
//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk walks the tree t sending all values
// from the tree to the channel ch.
func Walk(t *tree.Tree, ch chan int)

// Same determines whether the trees
// t1 and t2 contain the same values.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
```

**Complete revised Gujarati Go candidate:**

```go
//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, ટ્રી t માંથી પસાર થઈને તેના બધાં મૂલ્યો
// ટ્રીમાંથી ચેનલ ch પર મોકલે છે.
func Walk(t *tree.Tree, ch chan int)

// Same તપાસે છે કે
// t1 અને t2 નામનાં બંને ટ્રીમાં એકસરખાં મૂલ્યો છે કે નહીં.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
```

### #108 `example:concurrency/exercise-web-crawler.go` — A (previous B)

- Previous formal finding: The TODO translation of Fetch URLs in parallel uses singular URL ને, failing to make clear that several URLs must be fetched in parallel. Adjust only this TODO to express the plural object while preserving TODO, parallel semantics and protected bytes.
- Original source SHA-256: `bd6226dcae3b663a3aa1cbf502d840357ac0efb6de46e7a66612ca2834cf5e37`.
- Revised candidate SHA-256: `e3755e7812b4867ea5121474cec1205fa17eba7ad61602e82144c2cb45abada2`.
- Automatic validation SHA-256: `e8243723063cca7518227541c477ba2180b348c4b04979c87a00ee2c23781bf0` (passed).
- Relevant revision input SHA-256: `53554a1126b74da6e3bcb93c625eff1ec44f560e2817f1875202423a7d6009ab`.
- Revision batch / attempt: `chatgpt-gu-IN-005` / 1.
- Machine-protected comparison: all 77 non-comment lines byte-identical; 10 comment delimiters/indent prefixes identical; Go build directive byte-identical.
- Independent reviewer verdict: "TODO: એકથી વધુ URL સમાંતર રીતે મેળવો." explicitly directs parallel fetching of more than one URL. The other TODO still forbids fetching the same URL twice. The next comment correctly states this current implementation does neither. Fetch returns the page body and found URL slice; Crawl recursively follows pages until maximum depth; fakeFetcher and populated fetcher descriptions remain faithful. The glossary keeps URL untranslated and distinguishes parallelism from concurrency. Full file and all non-comment machine semantics are unchanged; no new issue.

**Complete English Go original:**

```go
//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Fetch URLs in parallel.
	// TODO: Don't fetch the same URL twice.
	// This implementation doesn't do either:
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

// fakeFetcher is Fetcher that returns canned results.
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

// fetcher is a populated fakeFetcher.
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
```

**Complete revised Gujarati Go candidate:**

```go
//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch, URL નો બોડી ભાગ અને
	// તે પાના પર મળેલા URL ની સ્લાઇસ પરત કરે છે.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl, fetcher નો ઉપયોગ કરીને
// url થી શરૂ થતાં પાનાંઓને depth ની મહત્તમ ઊંડાઈ સુધી પુનરાવર્તિત રીતે ક્રૉલ કરે છે.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: એકથી વધુ URL સમાંતર રીતે મેળવો.
	// TODO: એક જ URL ફરીથી મેળવશો નહીં.
	// આ અમલીકરણ આ બંનેમાંથી કંઈ કરતું નથી:
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

// fakeFetcher એ તૈયાર પરિણામો પરત કરતું Fetcher છે.
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

// fetcher એ ડેટાથી ભરેલું fakeFetcher છે.
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
```

## Decision and strict handoff

- Independent Example re-QC: **A=2, B=0, C=0, D=0; PASS**. Both previous B findings resolved; no introduced faults.
- Formal Reviewer evidence is saved by `review-evidence save` and checked by `review-evidence check` separately from this independent language judgment.
- Unique next action: original gu-IN Generation session or maintainer Local terminal must recheck CURRENT and preflight, then record the two A ratings on `20260927-gu-IN-qc-002` with preserved predecessor `20260927-gu-IN-qc-001`; inspect exact machine scope afterward.
- Expected after successful record: 114 carried-forward A plus six recorded Page A plus two Example A = 122 effective A, pending=0. Reviewer does not run record/finalize/promotion/Production/Git.
