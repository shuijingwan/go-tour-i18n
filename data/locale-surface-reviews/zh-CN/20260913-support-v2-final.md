# zh-CN Locale Surface Review — 2026-09-13 Support v2 final

## Review identity

- locale: zh-CN
- review_id: 20260913-support-v2-final
- reviewer: ChatGPT GPT-5.6 Sol
- decision: passed
- package: /tmp/go-tour-support-v2-final-review/zh-CN.json

## Scope

本轮对当前 deterministic Locale Surface Review package 进行了完整语言质量审核。

Mechanical package coverage:

- Pages: 103
- UI messages: 112
- articles: 7
- TranslationUnits represented for metadata context: 122
- other surfaces: 22

审核覆盖完整 glossary、English ↔ locale UI catalog、article metadata、
103 个 Page source / canonical target / course description，以及 homepage、
navigation、language selector、runtime、templates、partials、project、SEO
和 production public identity context。

本轮 Support v2 的正式变化集中在 UI catalog 与 project-visible support
configuration。课程 TranslationUnit、course metadata、article metadata、
catalog/source identity、language registry、SEO 和稳定 public identity
未发现新的语言质量问题。

## Support v2

Support v2 当前 source ↔ target 文案已经完整审核。

通过项目包括：

- 广告说明与运营成本说明
- 支持当前语言版本的表述
- support reference
- USDC / USDT asset、network 与 receiving-side minimum 文案
- Base / Tron(TRC20) network-only warning
- wrong asset/network loss warning
- Binance / OKX UID platform internal transfer
- address / reference / UID copy accessible labels
- localized copied Toast
- desktop/mobile context 下的组合语义

正式 mainland support surface 使用微信支付和支付宝；international crypto / UID methods 不进入 zh-CN 首页。

平台内转账说明保持简洁，不承诺永久免费、固定手续费或固定可用性。

未发现 unsupported affiliation、endorsement、payment guarantee、
anonymity、regulatory 或 asset-transfer claim。

## Language-quality checks

- complete current package coverage: passed
- UI source ↔ target fidelity: passed
- UI naturalness: passed
- glossary consistency: passed
- forbidden terminology regression: none
- article metadata: passed
- course metadata lineage: passed
- homepage / navigation / runtime / list / SEO language: passed
- support audience semantics: passed
- payment/network identity: passed
- copy interaction language: passed
- unsupported expansion or affiliation claims: none
- untranslated language-content regression: none
- language-quality blockers: 0

## Decision

Locale Surface Review A: passed.

本审核不替代 TranslationUnit Quality Check，
也不构成 rendered preview acceptance 或 HUMAN visual gate。

## Rendered surface acceptance

- automated preview acceptance: passed
- preview visual HUMAN gate: passed
