# da-DK independent TranslationUnit re-QC — qc-002 — Page 2, 51, 93

- Reviewer: original independent ChatGPT GPT-5.6 Sol High session; no da-DK Generation, revision or replacement participation.
- Rubric: translation-quality/v1. Previous full Snapshot: qc-001. Current full Snapshot: qc-002.
- Revision batch: chatgpt-da-DK-004, three complete Page TranslationUnits.
- Reviewer ZIP SHA-256: 957ed23d4581a9fafa9c0e2a5d87265db5a9248ba94ebff6d3fecba435902e77.
- Bundle identity: be0a529ef6a88eddf5dd48ede908e702c0173484bdb808a0811a78d10076b39f.
- Scope: three full English Page sources and Danish revision candidates, glossary, formal inputs, validation receipts, historical findings, full Snapshot and bundled authorities.
- Independent result: A=3, B=0, C=0, D=0; no new finding. This session has not reviewed Example 108 in qc-002.

## Per-Unit ratings

| Stable index | Unit ID | Previous QC | Current QC | Finding resolution |
| ---: | --- | :---: | :---: | --- |
| 2 | welcome/2 | B | A | F01: accurate multilingual Tour heading. |
| 51 | moretypes/15 | C | A | F02: slice type []T explicitly distinct from element type T. |
| 93 | concurrency/1 | B | A | F03: correct lightweight-thread spelling and retained Go runtime semantics. |

## Detailed independent review

### welcome/2 (index 2) — A

- English heading: * Go local. English body: The tour is available in other languages, followed by 13 Tour translations and next-page/PageDown instructions.
- Revised Danish heading: * Go på andre sprog. Body: Rundturen findes også på andre sprog.
- The revised heading clearly means other languages rather than offline execution. All 13 language links, translated labels, next-page button JavaScript target and PageDown are faithful; no extra or omitted source meaning.

### moretypes/15 (index 51) — A

- English context: func append(s []T, vs ...T) []T. The English prose ambiguously calls s a slice of type T.
- Revised Danish: Den første parameter s til append har typen []T, altså en slice med elementer af typen T.
- This correctly identifies s as type []T and T as the element type. The variadic appended T values, returned slice, backing-array reallocation, protected code and link targets remain correct and natural.

### concurrency/1 (index 93) — A

- English source: A _goroutine_ is a lightweight thread managed by the Go runtime.
- Revised Danish: En _goroutine_ er en letvægtstråd, der styres af Go-kørselsmiljøet.
- The compound spelling is corrected. Legitimate Go Present emphasis is preserved. Evaluation of f/x/y/z in the current goroutine versus execution of f in the new goroutine, shared address-space synchronization and the sync package are accurately distinguished. The final next-page reference matches Tour context.

## Formal evidence cross-check

- Full independent evaluation used the three source/candidate pairs and the locale glossary, not an automatic validation shortcut.
- Source/candidate hashes match the Snapshot. All 14 welcome/2 link targets, 2 moretypes/15 link targets and 1 concurrency/1 link target are unchanged. Preformatted code and Present directives match their English source counterparts.
- Automatic validation receipts each report passed and corroborate structural validity without substituting for language quality review.
- Original 118 A units remain outside this revision working set; they are eligible for carry-forward only under the formal exact-identity scope. The Example candidate must be reviewed in a separate invocation.
