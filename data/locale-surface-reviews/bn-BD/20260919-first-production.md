# bn-BD Locale Surface Review — First Production

## Review identity

- locale: `bn-BD`
- stage: `Locale-level language quality review (Stage A)`
- reviewed repository commit: `849770e99e590ca3f63dbfb2037a9814d9948ddb`
- deterministic review package: `/tmp/bn-BD-surface-review.json`
- package SHA-256: `08ae29f0514f23ea3f3f19eccc125488128ca4711ee72a62964e6c0e47dee08a`
- reviewer: `ChatGPT GPT-5.6 Sol High`
- date: `2026-09-19`
- production state: `first-production`

## Reviewed identities

- glossary: `locales/bn-BD/glossary.yaml`, SHA-256 `5635897781910bff59fc99e01bbb0e667524acf3bc6aff4abdac30474bac155c`
- English UI catalog: `internal/tour/ui/en.json`, SHA-256 `9a6e6877cd195144d6121f93cdabd387bdeb35c60a3f1afa9b8f417c7b6dabda`
- bn-BD UI catalog: `internal/tour/ui/bn-BD.json`, SHA-256 `91eaf9b004b01d949e50b3654a8794a3928deb37c114b96e6d9d1c0ca593b5e8`
- article metadata: `locales/bn-BD/article-metadata.json`, SHA-256 `9a4c3710d689bda889c34614d8637cb535a433c2d13b125a2195014b2b82984f`
- schema v2 course metadata: `locales/bn-BD/course-metadata.json`, SHA-256 `9f9f82f9792a7632b42087389058551c8d7f9ee3b0946e1d4a58631ee5775484`
- canonical English source descriptions: `data/course-seo/source-descriptions.json`, SHA-256 `1a46d981268ba75767986b3c212be06b6f678179f882efbc1813dedfad0a171d`
- production public identity: locale `bn-BD`, hostname `bn-go-dev.shuijingwanwq.com`, URL `https://bn-go-dev.shuijingwanwq.com/`

## A. Locale-level language quality review

The current deterministic Surface Review package was reviewed in full against the current repository authority and complete bn-BD glossary.

- UI catalog: `113/113` — passed
- article metadata: `7/7` — passed
- schema v2 Course SEO: `103/103` — passed
- TranslationUnit context: `122/122` — no new TranslationUnit defect found
- other first-party locale/runtime/template surfaces: `22/22` — passed
- glossary / identity consistency: `PASS`
- unresolved language blocker: `none`
- issues: `none`
- Stage A decision: `passed`

For Course SEO, all `103/103` schema v2 Pages were reviewed with the complete English Page source, canonical English description, complete final Bengali target, current glossary, and localized Bengali description. No semantic-scope change, unsupported expansion, cross-Page contamination, route mismatch, generic/duplicate description, keyword stuffing, technical error, or glossary defect was found.

The exporter validated the complete `122/122` ready TranslationUnit set before producing the package. The package schema serializes complete Page context for `103/103` Pages; the `19/19` Examples are not independently serialized as Surface Review language objects. Their formal TranslationUnit language-quality authority remains the completed A-only TranslationUnit Quality Check. No new TranslationUnit defect was exposed by the locale-level composition review.

The `22/22` other surfaces were reviewed, including language registry/profile, Tour shell/runtime, production Playground transport, project/SEO identity, first-party Playground/Tour JavaScript, templates, and partials. Legacy hard-coded English strings in the generic Playground branch are not used by the current Tour public execution path and therefore are not a bn-BD locale-surface blocker.

Forbidden terms `Golang`, `গোরুটিন`, `গো রুটিন`, and `গো প্লেগ্রাউন্ড` occur only in the glossary's own forbidden declarations; actual reviewed Bengali surfaces contain zero occurrences.

## B. Rendered surface acceptance

### Automated preview acceptance

- formal entry: `scripts/verify-preview-browser.py`
- preview URL: `http://127.0.0.1:45097/`
- preview identity: `PASS`
- SEO/routes: `PASS`
- desktop rendered surface: `PASS`
- editor Run / Format / Reset: `PASS`
- SPA: `PASS`
- mobile `/tour/moretypes/1`: `PASS`
- overall: `PREVIEW SURFACE ACCEPTANCE: PASS`
- issues: `none`

### Preview HUMAN visual gate

- maintainer confirmation: `passed`
- desktop overall layout: `passed`
- mobile overall layout: `passed`
- no blocking visual anomaly observed
- overall: `passed`

Stage A language-quality review, automated preview acceptance, and Preview HUMAN visual gate are complete and passed. Production machine/browser acceptance remains a later independent gate.

<!-- first-production-finalization:start -->
- production receipt identity: `locale=bn-BD hostname=bn-go-dev.shuijingwanwq.com release=20260919-bn-BD-ab8bc41`
- production machine acceptance: `passed`
- production browser acceptance: `passed`
- unresolved production blocker: `none`
- overall final decision: `passed`
- decision: `passed`
<!-- first-production-finalization:end -->
