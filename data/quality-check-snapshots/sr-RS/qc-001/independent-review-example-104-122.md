# sr-RS Independent TranslationUnit QC — Example 104–122

- Reviewer: independent ChatGPT GPT-5.6 Sol High; did not participate in sr-RS Generation or replacement.
- Snapshot: qc-001; scope: Example stable index 104–122, 19 complete TranslationUnits.
- Reviewer Bundle: /tmp/sr-RS-qc-001-qc-104-122.zip
- Bundle SHA-256: 2699405ccd7e3d4098509bbde2b7fcfd31bdea09aafdd65574992badab7b469d
- Bundle identity: a3280b70702c0cfcd20338b163060ef5d6e9f32fb63964d457dd39b4b242e062
- Rubric: translation-quality/v1
- Reviewer checked the ZIP SHA-256, all 48 manifest members, Snapshot/glossary identity, all 19 source/candidate/validation identities, and all validation evidence. Candidate changes were confined to allowed // teaching comments.
- Result: A=16, B=3, C=0, D=0. This scope is not A-only.

## Individual results

| Stable index | Unit ID | Rating | Finding |
|---:|---|:---:|---|
| 104 | example:basics/numeric-constants.go | B | F01 |
| 105 | example:basics/type-inference.go | A | — |
| 106 | example:concurrency/channels.go | A | — |
| 107 | example:concurrency/exercise-equivalent-binary-trees.go | A | — |
| 108 | example:concurrency/exercise-web-crawler.go | B | F02 |
| 109 | example:concurrency/mutex-counter.go | A | — |
| 110 | example:flowcontrol/if-and-else.go | B | F03 |
| 111 | example:generics/index.go | A | — |
| 112 | example:generics/list.go | A | — |
| 113 | example:methods/exercise-reader.go | A | — |
| 114 | example:methods/exercise-stringer.go | A | — |
| 115 | example:methods/interfaces-are-satisfied-implicitly.go | A | — |
| 116 | example:methods/interfaces.go | A | — |
| 117 | example:moretypes/append.go | A | — |
| 118 | example:moretypes/exercise-fibonacci-closure.go | A | — |
| 119 | example:moretypes/pointers.go | A | — |
| 120 | example:moretypes/slice-len-cap.go | A | — |
| 121 | example:moretypes/slices-of-slice.go | A | — |
| 122 | example:moretypes/struct-literals.go | A | — |

## Findings

### F01 — stable index 104 — example:basics/numeric-constants.go — B
The candidate phrase "од 1 праћене са 100 нула" is an awkward calque. Revise to natural Serbian expressing that the binary number consists of one 1 followed by 100 zeroes, preserving 1, 100, the binary meaning, and all Go code/numerals.

### F02 — stable index 108 — example:concurrency/exercise-web-crawler.go — B
The teaching comments use unnatural Serbian around kept identity URL, including "Не преузимајте исту URL двапут" and "Fetch враћа тело странице URL". Revise the URL noun phrases/case/gender naturally while preserving URL, Fetch, Crawl, all code identities, the fetched-body meaning, and the requirement not to fetch the same URL twice.

### F03 — stable index 110 — example:flowcontrol/if-and-else.go — B
The comment "овде не можете да користите v, међутим" uses awkward sentence-final међутим following English word order. Use natural Serbian contrastive word order while preserving identifier v exactly and the scope meaning that v cannot be used here.

## State after this review

All 19 Example TranslationUnits in the final initial-QC working set were independently reviewed. Together with the first two recorded groups, the full qc-001 first-pass totals are A=115, B=6, C=1, D=0. All seven non-A units must enter concentrated revision, split by Page and Example and followed by automatic validation, a new full Snapshot, and independent re-QC of only the pending changed scope.
