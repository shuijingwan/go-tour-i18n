# nb-NO — Independent TranslationUnit QC, Page 61–103

- Role: independent ChatGPT Reviewer, GPT-5.6 Sol High; no nb-NO generation.
- Snapshot: 20260927-nb-NO-qc-001
- Rubric: translation-quality/v1; first QC, no finalized predecessor.
- Scope: Page stable indexes 61–103 (43 complete TranslationUnits).
- Formal reviewer bundle: /tmp/nb-NO-20260927-nb-NO-qc-001-qc-61.zip
- Bundle SHA-256: 132581dc30d2354e70bcc69d5e844262d4ce952ccef50c0717f195f4a07e2b2d
- Bundle input identity: a09601e6f810056dea92b08a42a23df944a53dfa1fe920d3fd7aca9249affdad
- Source/target/validation and all authority/glossary reviewed; bundle current check: PASS.
- Decision: 41 A, 2 B, 0 C, 0 D. Findings are independent and require generation-led revision.

## Per-TranslationUnit ratings

| Stable index | Unit ID | Rating |
|---:|---|:---:|
| 61 | `moretypes/25` | B |
| 62 | `moretypes/26` | A |
| 63 | `moretypes/27` | A |
| 64 | `methods/1` | A |
| 65 | `methods/2` | A |
| 66 | `methods/3` | A |
| 67 | `methods/4` | A |
| 68 | `methods/5` | A |
| 69 | `methods/6` | A |
| 70 | `methods/7` | A |
| 71 | `methods/8` | A |
| 72 | `methods/9` | A |
| 73 | `methods/10` | A |
| 74 | `methods/11` | A |
| 75 | `methods/12` | A |
| 76 | `methods/13` | A |
| 77 | `methods/14` | A |
| 78 | `methods/15` | A |
| 79 | `methods/16` | A |
| 80 | `methods/17` | A |
| 81 | `methods/18` | A |
| 82 | `methods/19` | A |
| 83 | `methods/20` | A |
| 84 | `methods/21` | A |
| 85 | `methods/22` | A |
| 86 | `methods/23` | A |
| 87 | `methods/24` | A |
| 88 | `methods/25` | A |
| 89 | `methods/26` | A |
| 90 | `generics/1` | A |
| 91 | `generics/2` | A |
| 92 | `generics/3` | A |
| 93 | `concurrency/1` | B |
| 94 | `concurrency/2` | A |
| 95 | `concurrency/3` | A |
| 96 | `concurrency/4` | A |
| 97 | `concurrency/5` | A |
| 98 | `concurrency/6` | A |
| 99 | `concurrency/7` | A |
| 100 | `concurrency/8` | A |
| 101 | `concurrency/9` | A |
| 102 | `concurrency/10` | A |
| 103 | `concurrency/11` | A |

## B finding — index 61, moretypes/25: unnatural compound and plural (minor naturalness)

English source title: `* Function closures`

Current Norwegian title: `* Funksjons-closure-er`

The same candidate body uses closure-er as a mechanical plural. Funksjons-closure-er unnaturally combines a Norwegian compound prefix, an English technical loanword, and a hyphenated plural suffix. Adjust title and inflection naturally in Bokmål while retaining the approved closure technical term and all closure/captured-variable teaching semantics.

## B finding — index 93, concurrency/1: incorrect definite noun phrase (minor grammar)

English source: `The evaluation of `f`, `x`, `y`, and `z` happens in the current goroutine and the execution of `f` happens in the new goroutine.`

Current Norwegian: `Uttrykkene `f`, `x`, `y` og `z` evalueres i den nåværende goroutine, mens `f` kjøres i den nye goroutine.`

Both den nåværende goroutine and den nye goroutine are grammatically unnatural in Bokmål because the nominal phrase with den needs a natural definite form. Correct both without changing the goroutine technical identity or the difference between evaluating arguments in the current goroutine and executing f in the new one.

## Other checks

Preserved code, links, inline markup, directives and Go Present semantics; no finding based merely on raw protected underscores or legal in-span markers. concurrency/11 correctly uses lysbilder for three talk-presentation slide links and preserves their targets. No other substantive translation or technical defects identified in this working set.

## After formal recording

- This group is fully recorded: 43/43.
- Full initial QC recorded: 103/122 (99 A, 3 B, 1 C, 0 D).
- Remaining pending: 4 already-recorded non-A Page units plus 19 not-yet-reviewed Example units.
