# zh-CN Locale Surface Review — 2026-09-16 Header navigation

## Review identity

- locale: `zh-CN`
- review_id: `20260916-header-navigation`
- reviewer: `ChatGPT GPT-5.6 Sol`
- date: `2026-09-16`
- decision: `passed`
- reviewed commit: `c6c49a7511364a82284858ee98cc7991554e628b`
- package: `/tmp/zh-CN-surface-review-v2.json`

## Formal input identity

- glossary: `locales/zh-CN/glossary.yaml`
- glossary SHA-256: `0f8c5619ed8288ae9df2a380b2b98d34df1e05d36e846517de36f0b75f218e96`
- English UI SHA-256: `9a6e6877cd195144d6121f93cdabd387bdeb35c60a3f1afa9b8f417c7b6dabda`
- zh-CN UI SHA-256: `a4f67de71e15ee0b0ae12330061bbd67c61c1a2e1b27aac36e843df771f7def2`
- article metadata SHA-256: `b35eec52636238836fb9d904dbf03a627a2ccd95dc4279567e13001f32b836a6`
- course metadata SHA-256: `63f531c77b603ed252056386fa7aed7efc5eb924b33636077a9fe80d22a690a1`
- catalog/source SHA-256: `9b6f9e5a5d698b4417aee5fc64239408461fb47876a69c3c803b3e51f25f285f`
- languages config SHA-256: `8affaeba1e7232a5e7062940551c6d53ce99357ef138be0a03e54b56b21f9428`
- project config SHA-256: `78ae201af9207dbb74b266fb64614b82688195e8fff3b71f44b618b2a6ac23e3`
- SEO config SHA-256: `dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`
- production public identity SHA-256: `5848326f01e7c7a7d0dc0c0143873318be8b3aec69de9bc1525557b2e0ac2518`
- production hostname: `go-dev.shuijingwanwq.com`
- production public URL: `https://go-dev.shuijingwanwq.com/`

## Scope

本轮对当前 deterministic Locale Surface Review package 进行了完整语言质量审核。

Mechanical package coverage:

- Pages: 103
- UI messages: 113
- articles: 7
- TranslationUnits represented for metadata context: 122
- other surfaces: 22

审核覆盖完整 glossary、English ↔ locale UI catalog、article metadata、
103 个 Page source / canonical target / course description，以及 homepage、
navigation、language selector、runtime、templates、partials、project、SEO
和 production public identity context。

本轮正式变化包括课程 Header 的“关于此项目”入口、语言选择器及相关共享
navigation/runtime surface。

初次审核发现 `module.concurrency.description` 中 source 的 `goroutines`
被 target 写为 `goroutine`，与 zh-CN glossary keep 决策不一致。
修正为 `goroutines` 后重新导出完整审核包。

修正后的 package 与前一完整 package 逐字段比较，仅有：

- `inputs.ui_locale_sha256` 更新；
- `module.concurrency.description` 中 `goroutine` → `goroutines`。

其余正式审核输入与内容保持一致。

## Language-quality checks

- complete current package coverage: passed
- UI source ↔ target fidelity: passed
- UI naturalness: passed
- glossary consistency: passed
- forbidden terminology regression: none
- article metadata: passed
- course metadata lineage: passed
- homepage / navigation / language selector / runtime / list / SEO language: passed
- unsupported expansion or affiliation claims: none
- untranslated language-content regression: none
- language-quality blockers: 0

## Decision

Locale Surface Review A: **passed**.

本审核不替代 TranslationUnit Quality Check。

## Rendered surface acceptance

- preview URL: `http://127.0.0.1:37477/`
- projection: locale=`zh-CN`, ready=122, pending=0, blocked=0, pages=103, articles=7
- preview identity: passed
- SEO/routes: passed
- desktop rendered surface: passed
- editor Run / Format / Reset: passed
- SPA: passed
- mobile `/tour/moretypes/1`: passed
- automated preview acceptance: **passed (`PREVIEW SURFACE ACCEPTANCE: PASS`)**
- Preview HUMAN visual gate: **passed**
- HUMAN visual scope: desktop and mobile overall layout and visual composition
- unresolved preview blocker: none

维护者于 2026-09-16 确认：

> zh-CN 正式 preview 桌面端和移动端视觉检查通过，没有发现异常。

## Production verification

- production URL: `https://go-dev.shuijingwanwq.com/`
- maintenance deployment: not yet performed for this reviewed commit
- production verification: not yet performed
- production result is not claimed by this pre-deploy review evidence

本轮 maintenance deploy 完成后，应以实际 production machine/browser
acceptance 结果补充本节；不得提前记录为 passed。

## Issues

- language-quality blockers: none
- preview blockers: none
- production blockers: none discovered at this pre-deploy stage
