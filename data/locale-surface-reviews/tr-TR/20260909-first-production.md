# tr-TR Locale Surface Review — 2026-09-09 首次 production Stage A 记录

这是 tr-TR 首次 production 的 Locale Surface Review 工作记录。A（Locale-level language quality review）和 B（Rendered surface acceptance）已完成；production finalization 的可变正式结论仅由文末 machine-finalized block 记录。

## 审核身份

- locale：`tr-TR`
- review_id：`20260909-first-production`
- Stage A reviewed baseline commit：`40954fd77f435576dfbaedbf91b310cd262e1da7`
- configured production public identity：`https://tr-go-dev.shuijingwanwq.com/`
- date：`2026-09-09`

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article/lesson entries：`7`

TranslationUnit 已完成 automatic validation、ChatGPT Quality Check（122/122 A）、machine finalization 和 promotion。本记录不替代这些 TranslationUnit 审核证据。

完整 build 已通过：

```text
locale=tr-TR ready=122 pending=0 blocked=0 pages=103 articles=7
```

### Glossary identity

- path：`locales/tr-TR/glossary.yaml`
- SHA-256：`ada18f72633760d40f259d5c10147ec14a0fa764668dc77f8488fab94d68757c`

### UI / metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`3a878119cf0d3414fcf6f4ab20abec6459727ad86060f50fd10b0391f46f2964`

tr-TR UI target：

- path：`internal/tour/ui/tr-TR.json`
- SHA-256：`43f135530b3e6256fa83e3ed491fc42b186f4f68a8d11d039ecc6234fc0fe174`

Article metadata：

- path：`locales/tr-TR/article-metadata.json`
- SHA-256：`6c64a1f019a1e7fc0a5e02a229d31a28b6bc08f3d729da878425d1794600b2cb`

Course metadata：

- path：`locales/tr-TR/course-metadata.json`
- SHA-256：`e9d81b68a0d29ad1e6a488d2d3da9f337c54c487380215f3e57efbddddbd7a5b`
- generator contract：`course-seo-description-v1`
- generation / revision provenance：`codex` / `gpt-5.6-sol-high` / `2026-09-09T14:43:52Z`

### Locale-visible stable identity

- `locales/tr-TR/locale.json`：SHA-256 `d79beeaabcca1b907360c9270cec17092613ff05965ab3edcf360ec8b12ed1fc`
- `data/tour-pages.tsv`：SHA-256 `50b6244ef1d8115b2332cd5fa20640ce12fef8db931a1e313b410ade586777b6`
- `internal/tour/languages.go`：SHA-256 `552aa2f4ff08c0e199022d425c476f0c4d6053cd53c20cda550b5d44b741bb1e`
- `internal/tour/project.go`：SHA-256 `47bba2660c2fea09af377a092498aedd868dca603f9a5f14ca1ffff7500a9d0b`
- `internal/tour/seo.go`：SHA-256 `dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`

当前稳定 public identity：

- locale：`tr-TR`
- production hostname：`tr-go-dev.shuijingwanwq.com`
- production public URL：`https://tr-go-dev.shuijingwanwq.com/`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 machine finalization。ChatGPT GPT-5.6 Sol 对 TranslationUnit 之外的 tr-TR locale-level 语言资产完成完整 source ↔ target 审核，并以 `locales/tr-TR/glossary.yaml` 为正式术语基线。

### 公共 UI catalog

完整审核：

- `internal/tour/ui/en.json`
- `internal/tour/ui/tr-TR.json`
- `locales/tr-TR/glossary.yaml`

结果：

- reviewed：`92/92`
- passed：`92/92`
- failed：`0`
- key / plain-rich kind / placeholder / rich markup identity：均通过
- forbidden glossary term：未发现 blocker

首轮 Stage A 发现 2 个语言质量问题：

- `module.using_tour.description`
- `site.continue_learning_description`

Codex GPT-5.6 Sol High 根据正式 source/context 和完整 glossary 修订后，ChatGPT GPT-5.6 Sol 对当前正式 catalog 完成重新审核，两项问题均已解决。

### Article metadata

完整对照当前 7 个正式 article 的英文根级 title / subtitle 与 tr-TR article metadata：

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

未发现需要修订的问题。

### Course SEO metadata

按照每个 Page 的完整英文 source、最终 canonical tr-TR target、完整 glossary 和正式 description 完成审核：

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

首轮 Stage A 发现 `14` 条 description 存在语言质量问题，主要包括：

- Congratulations Page 跨 Page enrichment；
- source 未支持的因果或用途扩写；
- 个别 source 范围扩大；
- 土耳其语技术表达不自然。

Codex GPT-5.6 Sol High 随后重新读取完整 103/103 Page 正式输入：

- 实际修改：`14`
- 完整复核后保持字节不变：`89`
- byte duplicate：`0`
- normalized duplicate：`0`
- Unicode 长度范围：`70–173`

重新 assemble 后正式 `locales/tr-TR/course-metadata.json` 为 current 103/103 完整资产。ChatGPT GPT-5.6 Sol 对 14 条实际修订重新审核，全部通过；其余 89 条沿用首轮逐 Page 完整审核结论。

### 首页 / Tour shell / list / 导航 / language selector / runtime / SEO

Stage A 机械提取覆盖 `123` 项其他 locale-visible surface，并逐项结合正式 source identity、tr-TR target 和 glossary 进行语言审核。

首轮发现一个 shared Playground runtime blocker：

- frontend 自行生成英文 `status N.`；
- manual Kill 自行生成英文 `killed`；
- 与已本地化 `execution.exited` 拼接后会形成 mixed-language output。

共享实现已修复：

- non-zero exit 保留语言中立的数字 exit status；
- manual stop 不再生成英文 `killed` body；
- production compile proxy 保留 upstream numeric Status；
- stdout / stderr / server/compiler 原始错误语义不变；
- 未新增 UI key；
- transport event type / code execution protocol semantics 不变。

Targeted tests：

- `TestTurkishCatalogMatchesEnglishSource`：PASS
- `TestProductionCompileProxy`：PASS
- `TestPlaygroundUsesBootstrappedExitedMessage`：PASS
- `TestHTTPTransportRuntimeLocalizationInBrowser`：PASS

修复后的 shared runtime surface 经 ChatGPT GPT-5.6 Sol 复审通过。

### TranslationUnit 回流检查

本次 Locale Surface Review 未发现需要回 TranslationUnit revision batch 的 candidate 问题。

- TranslationUnit revision required：`no`
- canonical candidate direct edit：`none`

### A 阶段结论

- UI：`92/92 passed`
- Article metadata：`7/7 passed`
- Course SEO metadata：`103/103 passed`
- other locale-visible surfaces：已完整审核
- unresolved language blocker：`none`
- Stage A result：`passed`

## B. Rendered surface acceptance

结果：`passed`

### Preview identity

- locale：`tr-TR`
- preview loopback URL：`http://127.0.0.1:34829/`
- reviewed baseline commit：`40954fd77f435576dfbaedbf91b310cd262e1da7`

### Automated preview rendered acceptance

正式浏览器验收结果：

```text
[preview-browser] preview identity: PASS
[preview-browser] SEO/routes: PASS
[preview-browser] desktop rendered surface: PASS
[preview-browser] editor Run / Format / Reset: PASS
[preview-browser] SPA: PASS
[preview-browser] mobile /tour/moretypes/1: PASS
PREVIEW SURFACE ACCEPTANCE: PASS
```

- preview automated rendered acceptance：`passed`

### Visual HUMAN gate

结果：`passed`

维护者本人在 automated preview acceptance 通过后完成桌面与移动端视觉检查，确认本次 release 无阻塞性视觉异常。

维护者观察到两处既有的小型 header 对齐观感问题；本次判断为非阻塞问题，后续作为共享样式维护项独立处理，不在 tr-TR 首次 production 流程中临时修改。

- preview visual HUMAN gate：`passed`
- unresolved preview rendered blocker：`none`

## Reviewer

- Locale Surface Review A：ChatGPT GPT-5.6 Sol
- UI / course metadata / shared runtime revision：Codex GPT-5.6 Sol High
- deterministic assemble / build：maintainer local terminal

## Decision

Stage A：`decision = passed`

Rendered surface acceptance：`decision = passed`

首次 production 的整体最终结论见文末 machine-finalized block。

<!-- first-production-finalization:start -->
- production receipt identity: `locale=tr-TR hostname=tr-go-dev.shuijingwanwq.com release=20260909-tr-TR-3806beee`
- production machine acceptance: `passed`
- production browser acceptance: `passed`
- production visual HUMAN gate: `passed` (maintainer confirmation)
- unresolved production blocker: `none`
- overall final decision: `passed`
- decision: `passed`
<!-- first-production-finalization:end -->
