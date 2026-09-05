# it-IT Locale Surface Review — 2026-09-05 首次 production Stage A 记录

这是 it-IT 首次 production 的 Locale Surface Review 工作记录。本文件仅记录已完成的 A（Locale-level language quality review）；preview rendered acceptance、visual HUMAN gate、publish 和 production verification 尚未执行。

## 审核身份

- locale：`it-IT`
- review_id：`20260905-first-production`
- Stage A reviewed baseline commit：`ffa44556d4df72aaa36f0810659303fe6e1373c6` (`ffa4455`)
- configured production public identity：`https://it-go-dev.shuijingwanwq.com/`
- date：`2026-09-05`

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article/lesson entries：`7`
- modules：`5`

TranslationUnit 已完成 automatic validation、ChatGPT Quality Check（全 A）、finalization 和 promotion。本记录不替代这些 TranslationUnit 审核证据。

### Glossary identity

- path：`locales/it-IT/glossary.yaml`
- SHA-256：`cd3b2ba87059a4df7db51cdfe457617819816655fdece5b50abc0aaf4c2b6dd1`

### UI / metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`3a878119cf0d3414fcf6f4ab20abec6459727ad86060f50fd10b0391f46f2964`

it-IT UI target：

- path：`internal/tour/ui/it-IT.json`
- SHA-256：`41d6586dc1086e43639f551064a8034185eb33ca3f7b6ae3b9dbc642f4fb55f0`

Article metadata：

- path：`locales/it-IT/article-metadata.json`
- SHA-256：`268c72cd4640876e9f1eddbf758474cb9a053dc765ce39e4254ed7ed1758414d`

Course metadata：

- path：`locales/it-IT/course-metadata.json`
- SHA-256：`aa74f8125047b48aab284060c6dfbab376a642de7ba1b668a9a02fa01048954f`
- generator contract：`course-seo-description-v1`
- generation / revision provenance：`codex` / `gpt-5.6-sol-high` / `2026-09-05T09:54:44Z`

### Locale-visible stable identity

审核的稳定 build-time 输入：

- `locales/it-IT/locale.json`：SHA-256 `8e7160fd4035477702d797bcb7f038af090abdbe5b090e4ce60553194a03a75e`
- `data/tour-pages.tsv`：SHA-256 `50b6244ef1d8115b2332cd5fa20640ce12fef8db931a1e313b410ade586777b6`
- `internal/tour/languages.go`：SHA-256 `d0ac72af8f3da200b20c01e0a453c75f0c6fd753c2ca636b59b1a42f4249c864`
- `internal/tour/project.go`：SHA-256 `47bba2660c2fea09af377a092498aedd868dca603f9a5f14ca1ffff7500a9d0b`
- `internal/tour/seo.go`：SHA-256 `dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`
- `production/identity.json`：SHA-256 `809201cb19a87cbdf961095eb5cde60022616731a88f76a6ba79fac1838afa01`

当前 it-IT identity：

- production hostname：`it-go-dev.shuijingwanwq.com`
- production_state：`first-production`
- CDN：`cloudflare`
- port：`4005`
- shared-assets policy：`shared-cloudflare`
- Playground allowed origin：`https://it-go-dev.shuijingwanwq.com`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 finalization。ChatGPT GPT-5.6 Sol 对 TranslationUnit 之外的 it-IT locale-level 语言资产完成独立的完整 source ↔ target 审核，并以 `locales/it-IT/glossary.yaml` 为正式术语基线。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/it-IT.json`
- `locales/it-IT/glossary.yaml`

- reviewed：`92/92`
- passed：`92/92`
- failed：`0`
- `plain` / `rich` kind、placeholder、rich markup/link identity：均通过
- forbidden glossary term：`0`

审核覆盖首页、`/tour/`、`/tour/list`、导航和语言选择器、编辑器动作与 runtime message、footer、共享 shell 和 SEO 可见 UI 文案。首轮 Surface Review 发现 locale-level UI language-quality 问题；Codex GPT-5.6 Sol High 随后完成 92/92 全量复审并实际修订 `22` 个 key。ChatGPT GPT-5.6 Sol 对修订后的当前正式 catalog 重新审核，未遗留公共 UI language blocker。

### Article metadata

完整对照当前 7 个正式 article 的英文根级 `title` / `subtitle` 与 it-IT article metadata。

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

### Course SEO metadata

按每个 Page 的完整英文 source、最终 canonical it-IT target 和 it-IT glossary 审核正式 course metadata。

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

首轮 Surface Review 发现 course metadata language-quality 问题；Codex GPT-5.6 Sol High 对完整 `103/103` 实际复审并修订 `25` 条 description，随后重新 assemble 为当前正式 metadata。ChatGPT GPT-5.6 Sol 已对当前 `103/103` 重新完成独立审核，未遗留 blocker。

本 evidence 不逐条复写全部 103 条 description；上述计数记录的是逐 Page 的完整覆盖，而非页面抽查。

### A 阶段结论

- unresolved language blocker：`none`
- Stage A result：`passed`

## 尚未执行的后续阶段

- machine-readable A gate：尚未执行 `surface-review record-a`，因此没有 A-gate receipt。
- preview rendered acceptance：`not executed`
- visual HUMAN gate：`not executed`
- publish：`not executed`
- production deployment / machine acceptance / browser acceptance：`not executed`
- ads 与 production acceptance：`not executed`

<!-- first-production-finalization:start -->
- production receipt identity: `PENDING`
- production machine acceptance: `PENDING`
- production browser acceptance: `PENDING`
- production visual HUMAN gate: `PENDING`
- unresolved production blocker: `PENDING`
- overall final decision: `PENDING`
- decision: `pending`
<!-- first-production-finalization:end -->

## Reviewer

- Locale Surface Review A：ChatGPT GPT-5.6 Sol
- course metadata generation / revision：Codex GPT-5.6 Sol High

## Issues 与当前 gate status

- unresolved language blocker：`none`
- preview automated rendered acceptance：`not executed`
- visual HUMAN gate：`not executed`
- unresolved preview rendered blocker：`not assessed`
- unresolved production blocker：`not assessed`
- production verification：`not executed`
- overall final decision：`PENDING`

## Decision

Stage A：`decision = passed`。首次 production 的整体最终 decision 仍为 `PENDING`，直至完成后续 required gates。
