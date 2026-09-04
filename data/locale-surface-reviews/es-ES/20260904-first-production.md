# es-ES Locale Surface Review — 2026-09-04 首次 production Stage A evidence

这是 es-ES 首次 production 的 Locale Surface Review 工作记录。目前仅完成并记录 A（Locale-level language quality review）。B（Rendered surface acceptance）和 production verification 尚未执行；本文件不记录最终 Locale Surface Review decision。

## 审核身份

- locale：`es-ES`
- review_id：`20260904-first-production`
- Stage A reviewed baseline commit：`eaaa133c090b632ea6aa348e45f2bab7c9e9f468` (`eaaa133`)
- 当前 worktree `HEAD`：`eaaa133c090b632ea6aa348e45f2bab7c9e9f468` (`eaaa133`)
- configured production public identity：`https://es-go-dev.shuijingwanwq.com/`
- date：`2026-09-04`

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article/lesson entries：`7`
- modules：`5`

TranslationUnit 已完成 automatic validation、ChatGPT Quality Check（全 A）、ChatGPT Final Review（全 A 且 `approved`）和 promotion。本记录不替代这些 TranslationUnit 审核证据。

### Glossary identity

- path：`locales/es-ES/glossary.yaml`
- SHA-256：`0db90221f1e546ec24141c7e02b4f3482b3c3dde1743f7588e75f93089f360d2`

### UI / metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`3a878119cf0d3414fcf6f4ab20abec6459727ad86060f50fd10b0391f46f2964`

es-ES UI target：

- path：`internal/tour/ui/es-ES.json`
- SHA-256：`f2f994f2f1a39e4a6f7274405b42171335d141b9a8e1c5f200f912f1501d0ac5`

Article metadata：

- path：`locales/es-ES/article-metadata.json`
- SHA-256：`744f40dd25eea4571e694d3de9821d80b46373d01de2b71d4352396cc194ae0d`

Course metadata：

- path：`locales/es-ES/course-metadata.json`
- SHA-256：`239dbf2ff339b9360d1309f4c9d812e7b874d1601808119fddea582593350a27`

### Locale-visible stable identity

审核的稳定 build-time 输入：

- `internal/tour/languages.go`：SHA-256 `23a842745781930c230d1e241a5d036b8ea5a905a3de0448e35e35217a4362c9`
- `internal/tour/project.go`：SHA-256 `47bba2660c2fea09af377a092498aedd868dca603f9a5f14ca1ffff7500a9d0b`
- `internal/tour/seo.go`：SHA-256 `dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`
- `production/identity.json`：SHA-256 `1421bc6e2bcc83565e45dc55249a481505143db1d4169542645785e2ec8aecdc`

当前 es-ES identity：

- production hostname：`es-go-dev.shuijingwanwq.com`
- production_state：`first-production`
- CDN：`cloudflare`
- port：`4004`
- shared-assets policy：`shared-cloudflare`
- Playground allowed origin：`https://es-go-dev.shuijingwanwq.com`
- timezone：`Europe/Madrid`
- visible local-time label：`hora local`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 Final Review。ChatGPT GPT-5.6 Sol 对 TranslationUnit 之外的 es-ES locale-level 语言资产完成独立的完整 source ↔ target 审核，并以 `locales/es-ES/glossary.yaml` 为正式术语基线。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/es-ES.json`
- `locales/es-ES/glossary.yaml`

- reviewed：`92/92`
- passed：`92/92`
- failed：`0`
- `plain` / `rich` kind、placeholder、rich markup/link identity：均通过
- forbidden glossary term：`0`

审核覆盖首页、`/tour/`、`/tour/list`、导航和语言选择器、编辑器动作与 runtime message、footer、共享 shell 和 SEO 可见 UI 文案。Surface Review 实际发现并修正过 UI 语言问题；最后一轮修复包含以下 key：

- `site.workflow_units`
- `site.workflow_review`
- `site.workflow_publish`
- `site.official_version`
- `site.architecture_playground`

最终没有遗留 es-ES 公共 UI language blocker。

### Article metadata

完整对照当前 7 个正式 article 的英文根级 `title` / `subtitle` 与 es-ES article metadata。

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

### Course SEO metadata

按每个 Page 的完整英文 source、最终 canonical es-ES target 和 es-ES glossary 审核正式 course metadata。

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

本轮存在真实的多轮 language-quality correction 与完整回流：初版 Surface Review 发现 literal 性别以及 fidelity / technical issues；v2 全量重审实际修改 `20` 个 Page。随后独立 A review 又发现 source-unsupported、cross-page 和 technical wording 问题；v3 再次完整重审 `103/103`，实际修改 `14` 个 Page。最终正式 course metadata commit 为 `ac4b353`。ChatGPT GPT-5.6 Sol 已对当前 v3 独立重新完成 `103/103` Surface Review，未遗留 blocker。

本 evidence 不逐条复写全部 103 条 description；上述计数记录的是逐 Page 的完整覆盖，而非页面抽查。

### A 阶段结论

- unresolved language blocker：`none`
- Stage A result：`passed`

## B. Rendered surface acceptance

结果：`not yet executed`

尚未运行 complete locale preview，尚未执行 browser acceptance，也尚未执行 visual HUMAN gate。本记录不声称任何 preview 或 rendered surface acceptance 已通过。

## Production verification

结果：`not yet executed`

尚未 publish、部署或验收 production；本记录不声称 es-ES 已进入 production 或已完成 production verification。

## Reviewer

- TranslationUnit Quality Check / Final Review：ChatGPT GPT-5.6 Sol
- Locale Surface Review A：ChatGPT GPT-5.6 Sol
- course metadata generation / re-generation：Codex GPT-5.6 Sol High
- evidence 文件整理：Codex Terra light

## Issues 与当前 gate status

- unresolved language blocker：`none`
- B rendered surface acceptance：`not yet executed`
- production verification：`not yet executed`
- overall final decision：尚未记录

最终 Locale Surface Review decision 仅可在所有必需阶段完成后记录为 `passed` 或 `failed`；当前不写入 `decision`。
