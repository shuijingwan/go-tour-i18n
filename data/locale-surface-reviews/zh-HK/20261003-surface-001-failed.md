# zh-HK Locale Surface Review Stage A — failed findings

- locale: `zh-HK`
- reviewer role: independent ChatGPT GPT-5.6 Sol High Reviewer
- decision: `failed`
- reviewer ZIP SHA-256: `4f1612b9a7277980911d8bae6eb0bdd27e6e23904e5349196bfc6f3b13a482da`
- surface-review package SHA-256: `ca5367cd09494db030137be60f43661f9d3f43ba8acd401c0e1a192752746fdf`
- coverage: UI 113/113; article metadata 7/7; Course SEO 103/103; TranslationUnit context 122/122; other surfaces 22/22
- blocking findings: 9 — 8 Course SEO localized descriptions and 1 locale profile label
- new TranslationUnit defects: none

## Findings

### Course SEO: welcome/1

- defect: mandatory glossary inconsistency
- current issue: canonical `A Tour of Go` is localized as「Go 導覽」instead of mandatory「Go 語言導覽」.
- correction: use the mandatory formal name without changing canonical semantic scope.

### Course SEO: welcome/2

- defect: mandatory glossary inconsistency
- current issue: canonical `translations of A Tour of Go` uses「Go 導覽」instead of mandatory「Go 語言導覽」.
- correction: use the mandatory formal name without adding or removing canonical semantics.

### Course SEO: welcome/5

- defect: preferred glossary and ready-target inconsistency
- current issue: `sandbox` is localized as「沙箱」instead of preferred「沙盒」used by the current Page target.
- correction: use「沙盒」while preserving the canonical scope.

### Course SEO: basics/2

- defect: canonical semantic-scope change
- current issue:「取代逐項撰寫匯入宣告」turns a preferred alternative into a replacement requirement, although separate import declarations remain valid.
- correction: express the factored form as the preferred or good-style alternative, not as making separate declarations unavailable.

### Course SEO: basics/8

- defect: preferred glossary inconsistency
- current issue: technical `scope` is localized as「範圍」instead of preferred「作用域」.
- correction: use the glossary term while preserving package-or-function scope semantics.

### Course SEO: basics/9

- defect: preferred glossary inconsistency and reduced technical precision
- current issue: `initializer` is localized as「初始值」instead of preferred「初始化式」; an initializer expression is not merely its result value.
- correction: use「初始化式」and preserve the type-inference semantics.

### Course SEO: flowcontrol/1

- defect: preferred glossary inconsistency
- current issue: technical `scope` is localized as「範圍」instead of preferred「作用域」used by the current Page target.
- correction: use the glossary term while preserving the canonical list of for-loop concepts.

### Course SEO: moretypes/3

- defect: unsupported semantic expansion
- current issue:「讀取或操作結構所保存的資料」adds a purpose absent from the canonical description and English source.
- correction: retain only the selected dot-notation field-access semantics.

### locale profile: internal/tour/languages.go localeProfiles["zh-HK"].TimeLabel

- defect: untranslated placeholder in a locale-visible runtime surface
- current issue: `TimeLabel` remains the literal placeholder `TODO`.
- correction: replace it with the formal zh-HK locale-visible time label, rebuild, and re-export the Surface Reviewer Bundle.

## Reviewer scope result

UI, article metadata, the remaining 95 Course SEO pages, TranslationUnit context, and the remaining other surfaces had no blocking finding. The Reviewer generated no replacement text.
