# te-IN Locale Surface Review Stage A — failed findings

- locale: `te-IN`
- reviewer role: independent ChatGPT GPT-5.6 Sol High Reviewer
- decision: `failed`
- reviewer ZIP SHA-256: `32e63d07f979fdb71aa05eabd62a59b4209188f6b11fc1d362c473457f73562a`
- surface-review package SHA-256: `3955acae149269d5a78b2ccd7c16779935d8c197a2c9c61a41415bb7f381f6a2`
- coverage: UI 113/113; article metadata 7/7; Course SEO 103/103; TranslationUnit context 122/122; other surfaces 22/22
- blocking findings: 9, all Course SEO localized descriptions
- new TranslationUnit defects: none

## Findings

### basics/14
- defect: Course SEO semantic-scope omission + glossary consistency
- current issue: localized description does not explicitly preserve canonical “infers variable types”; ready target discusses variable type inference.
- requirement: preserve inference of variable types from typed values plus default type/precision of untyped numeric constants.
- correction: explicitly retain variable-type inference and mandatory `variable → వేరియబుల్`; do not expand canonical scope.

### moretypes/1
- defect: Course SEO glossary mandatory inconsistency / untreated English
- current issue: description uses `పాయింటర్‌లు` earlier but leaves English `pointer arithmetic`.
- requirement: preserve pointer, address/dereference, nil zero value, and absence of pointer arithmetic.
- correction: apply mandatory `pointer → పాయింటర్` consistently; keep all canonical semantics.
### moretypes/9
- defect: Course SEO glossary mandatory inconsistency
- current issue: `array-literal సింటాక్స్` leaves `array` untranslated.
- requirement: construct a slice literal with array-literal syntax without specifying fixed array length.
- correction: apply mandatory `array → అరే`, preserving both array-literal and fixed-length semantics.

### methods/3
- defect: Course SEO glossary mandatory inconsistency
- current issue: `locally defined non-struct టైప్‌లు` leaves `struct` untranslated.
- requirement: locally defined non-struct types; receiver type belongs to the same package.
- correction: apply mandatory `struct → స్ట్రక్ట్`; preserve same-package rule.

### methods/8
- defect: Course SEO glossary mandatory inconsistency
- current issue: earlier `పాయింటర్ రిసీవర్‌లు` but later `consistent receiver kind`.
- requirement: pointer receiver for mutation/avoiding copies, with consistent receiver kind for methods of a type.
- correction: apply mandatory `receiver → రిసీవర్` consistently; retain consistency requirement.

### methods/10
- defect: Course SEO glossary mandatory inconsistency
- current issue: earlier `ఇంటర్‌ఫేస్‌లు`, later English `interface declarations`.
- requirement: implicit interface implementation and cross-package decoupling of interface declarations from implementations.
- correction: apply mandatory `interface → ఇంటర్‌ఫేస్` consistently; preserve declaration/implementation decoupling.
### methods/21
- defect: Course SEO glossary mandatory inconsistency + target inconsistency
- current issue: `byte slices` conflicts with mandatory `slice → స్లైస్` and ready target’s Telugu byte-slice wording.
- requirement: Reader interface fills byte slices and handles byte count, errors, end-of-stream signal.
- correction: use mandatory slice terminology consistently; preserve all canonical semantics.

### generics/1
- defect: Course SEO canonical semantic-scope omission / keep identity
- current issue: canonical says “generic Go functions”; localized description says only generic functions and drops explicit `Go`.
- requirement: retain Go language qualifier, bracketed type parameters, `comparable` constraint, equality-based slice search.
- correction: explicitly retain `Go` unchanged per keep identity; preserve remaining semantic scope.

### concurrency/5
- defect: Course SEO Telugu naturalness / grammatical agreement
- current issue: `పలువురు సిద్ధంగా ఉంటే ఒక case...` uses a people-count expression for multiple ready cases.
- requirement: select waits on multiple channel operations; executes a ready case; when multiple cases can run, randomly chooses one.
- correction: use natural Telugu for multiple ready cases; preserve select semantics.

## Reviewer scope result
UI, article metadata, public identity, other surfaces, and the remaining 94 Course SEO pages had no blocking finding. Reviewer generated no replacement text.
