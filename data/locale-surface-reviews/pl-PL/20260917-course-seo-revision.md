# pl-PL Locale Surface Review — Course SEO revision

- locale: `pl-PL`
- review_id: `20260917-course-seo-revision`
- reviewer: `ChatGPT GPT-5.6 Sol High`
- date: `2026-09-17`
- reviewed commit: `1c8d4de` (`fix: 支持课程 SEO 描述质量修订`)
- decision: **passed**
- stage: `locale-level-language-quality-review`

## Deterministic review package

- path: `/tmp/pl-PL-surface-review.json`
- SHA-256: `15e967af8be27b42d2ba39c0dfd8c95b37f286a53fa086c7e2c6b05a63445762`
- lines: `4257`
- bytes: `440205`
- package locale: `pl-PL`
- course metadata schema version: `2`

The package was read in full. Its SHA-256, byte/line identity, coverage, and formal input identities were mechanically checked before the language review. The working tree was clean and `HEAD` was `1c8d4de` when the review began.

## Formal input identities

- `ui_english_sha256`: `9a6e6877cd195144d6121f93cdabd387bdeb35c60a3f1afa9b8f417c7b6dabda`
- `ui_locale_sha256`: `91df39a712ebc260e83e114d4a30b09c30298d10f5742a679a677ab08232d75f`
- `glossary_sha256`: `18cd40600f0ae27c531d278cd4d22d4c6693f5003a10adfc4ecb9e50b2407b7e`
- `article_metadata_sha256`: `82460396ca49363dc6f1a39d56a0f0ddb6f03fe9ef5bf9f886fcd6b21fd8b97c`
- `course_metadata_sha256`: `e922f7ce8e5bd4d1e7d27235ecf6a20359266d4d8f590d893b56ebb1e69d224a`
- `course_source_descriptions_sha256`: `1a46d981268ba75767986b3c212be06b6f678179f882efbc1813dedfad0a171d`
- `course_source_description_review_sha256`: `9377e85b56bef45c5e1a1832186be7bc804816d0ac9149147db97267371b736f`
- `catalog_source_sha256`: `9b6f9e5a5d698b4417aee5fc64239408461fb47876a69c3c803b3e51f25f285f`
- `languages_config_sha256`: `268a60c5f67fcbf449ad6cfe737813b624ec3f67b8742fd10d39d356c1b73ded`
- `project_config_sha256`: `78ae201af9207dbb74b266fb64614b82688195e8fff3b71f44b618b2a6ac23e3`
- `seo_config_sha256`: `dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`
- `production_public_identity_sha256`: `c5f82c615bc18102862097919c67ec026d08d291cd1b70d03594010fd2d92758`

Production public identity in the package is `pl-PL` / `pl-go-dev.shuijingwanwq.com` / `https://pl-go-dev.shuijingwanwq.com/`.

## Canonical English source-description authority

- review_id: `canonical-en-002`
- stage: `canonical-english-course-seo-description-review`
- decision: `passed`
- reviewer: `ChatGPT GPT-5.6 Sol High`
- current authority SHA-256: `9377e85b56bef45c5e1a1832186be7bc804816d0ac9149147db97267371b736f`

`go run -mod=readonly ./cmd/tour-i18n course-metadata source review-check` returned PASS for this current authority before Stage A language review.

## Review coverage

The current package and formal source/context were reviewed completely; this was not sampling and the previous pl-PL Surface Review conclusion was not reused as the language decision.

- UI messages: `113/113`
- article metadata: `7/7`
- schema v2 Course SEO Pages: `103/103`
- TranslationUnit context represented by the formal package/status gate: `122/122` (`103` Pages + `19` Examples); this Stage A did not replace or re-run TranslationUnit A/B/C/D QC
- other locale-visible surfaces: `22/22`

For every schema v2 Course SEO Page, the review used the complete English Page source, current canonical English description, complete Polish canonical target, complete current glossary, localized Polish description, and route/identity. The revised `concurrency/10` description was specifically re-checked against all of these inputs in addition to the other 102 Pages.

The 22 other surfaces included the language registry/profile, Tour shell/runtime transport, project/SEO/public identity context, shared first-party Playground runtime, all first-party Tour JavaScript runtime files, formal templates, and Angular partials present in the package. Reachability was checked for hard-coded shared runtime text: the current Tour constructs `HTTPTransport()` directly; the unused shared `playground(opts)` path is not a current Tour-visible localization surface, while HTTPTransport's current visible status/error messages are catalog-backed.

## Stage A language-quality result

`Locale-level language quality review = passed`

- language-quality blockers: `0`
- TranslationUnit revision required: `no`
- Course SEO revision required: `no`
- unresolved issues: `none`

UI source ↔ Polish target pairs were checked for glossary decisions, naturalness, technical identity, plain/rich kind, placeholders, and markup. Article metadata was checked source ↔ target. Course SEO was checked for canonical-source alignment, semantic-scope fidelity, consistency with the complete Polish target, glossary compliance, technical accuracy, natural Polish, omissions/mistranslations, unsupported expansion, generic/duplicate descriptions, and route/page identity. Other public surfaces were checked for locale/hostname/project identity, official/unofficial boundaries, runtime composition, navigation/language-selector text, and first-party template/runtime localization behavior.

As supplemental mechanical checks, UI placeholder mismatches were `0`, rich-markup mismatches were `0`, normalized duplicate Course SEO descriptions were `0`, all `103` routes and Page IDs were unique, and all per-Page glossary identities matched the current glossary. Apparent `Golang` forbidden-term substring hits occurred only inside protected `golang.org` URLs/import paths, not as natural-language terminology.

## Rendered surface acceptance

- preview URL: `http://127.0.0.1:39491/`
- preview repository HEAD: `9ab70a1` (`docs: 记录波兰语课程 SEO 修订审核`)
- automated preview acceptance: **passed (`PREVIEW SURFACE ACCEPTANCE: PASS`)**
- Preview HUMAN visual gate: **passed**
- unresolved preview blockers: `none`

Automated acceptance passed preview identity, SEO/routes, desktop rendered surface, editor Run / Format / Reset, SPA behavior, and the mobile `/tour/moretypes/1` regression surface. The HUMAN visual gate confirmed the desktop/mobile overall presentation had no blocking visual anomaly. This section does not claim Production verification passed.

## Production verification

- production release repository HEAD: `1d66296078de16df9136bb321eb6ac06fe801516`
- release: `20260917T044813Z-pl-PL-1d66296078de`
- production URL: `https://pl-go-dev.shuijingwanwq.com/`
- CDN: `cloudflare`
- maintenance production receipt result: **passed**
- deploy: **PASS**
- exact-hostname purge: **PASS**
- production machine acceptance: **PASS**
- production browser acceptance: **PASS**
- sitemap: **105/105 PASS**
- socket boundary: **PASS**
- production browser desktop routes: **PASS**
- production browser mobile `/tour/moretypes/1`: **PASS**
- production browser Run / Format / Reset / SPA / ads: **PASS**
- unresolved production blockers: `none`

The sitemap verifier observed one transient `curl-exit-28` for `/tour/concurrency/1`; the bounded retry recovered on attempt 2/5 and the complete 105/105 sitemap verification passed.

The same production release workflow also updated the shared non-Chinese assets required by the current repository. `SHA256SUMS`, `tour/static/css/app.css`, and `tour/static/js/support.js` were the changed paths; exact-URL Cloudflare purge passed, changed-path cache verification passed, public SHA-256 verification passed for all `14/14` shared assets, and all `3/3` non-allowlist boundary paths remained HTTP 404.

`PRODUCTION RELEASE BATCH: PASS`.
