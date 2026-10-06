# Locale 长期路线图

## Versionless locale authority（V2-D 收敛）

v1/v2/v3 是 implementation/content-package milestone，不是 locale language governance namespace。Content package 可以版本化；Locale language system 不版本化。全站只有 `locales/<locale>/glossary.yaml`，不得 Tour/Docs/v2 split。current executable migration、统一 corpus、一次完整 glossary generation/refresh、统一 structured assets、完整 Page TU 与 delta-aware integrated Surface contract 见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。本节是全站扩展入口；下文旧 Tour workflow/evidence 保持兼容，不授权重复初始化 locale 或改写历史证据。

已有 Tour-complete locale 首次扩展：一次 unified glossary refresh → 一次独立 full unified Review → compatibility → 一次 pending structured-assets Generation → 两批 Learn/Docs Page Generation / 两批 Page QC → 一次 integrated Surface Review。unchanged Tour/UI/meta/SEO carry；只有真实 affected scope reopen。新 locale 从零执行同一 locale campaign，glossary 与 structured-assets 各一次，Tour 和 Learn/Docs 各用其 surface-specific parser/TU batches；没有第二次“V2 locale init”。Course SEO 是 Page-derived consumer，按自己的 canonical source/ready target contract 单列，不强行并入 structured-assets；没有 Learn/Docs canonical SEO authority时不创建 SEO target。

新 unified receipt `go-learning/unified-glossary-review/v1` 与 old v1/Tour legacy coverage additive 共存。新 Generation 要求 current unified coverage；旧 completion仍按旧 evidence合法验证。unified current exact优先；多个 current unified receipts、glossary/corpus mismatch fail closed。旧 receipt immutable，不 retroactively upgrade corpus coverage。glossary bytes不变但 corpus变了仍需一次 full unified Review；不猜造 corpus compatibility shortcut。新 glossary byte change须 full unified PASS后才用 V2-B semantic delta/精确 lineage复用旧结果。


本文档是社区 locale 的长期实施范围、当前 Go Tour 剩余调度，以及未来多语言项目默认顺序的唯一 authority。[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md) 仍是单个新 locale 的正式执行流程 authority；本文只确定规划范围与调度，不替代 locale identity freeze、语言资产、质量审核、Production 或上线流程。

## 1. 当前已完成：65 个 live community locale

当前仓库已有 **65 个 `production_state=live` community locale**：

```text
zh-CN  ja-JP  de-DE  fr-FR  ko-KR  es-ES  it-IT  nl-NL  pt-BR  tr-TR
sv-SE  pl-PL  zh-TW  id-ID  vi-VN  ar     th-TH  hi-IN  bn-BD  ur-PK
uk-UA  ro-RO  cs-CZ  ta-IN  te-IN  ms-MY  fil-PH el-GR  hu-HU  bg-BG
mr-IN  ml-IN  kn-IN  es-419 gu-IN  pa-IN  nb-NO  da-DK  fi-FI  sr-RS
sk-SK  hr-HR  sl-SI  lt-LT  ca-ES  et-EE  lv-LV
sw-TZ  kk-KZ  fa-IR  am-ET  ru-RU  pt-PT  fr-CA  zh-HK  de-CH
de-AT  zh-SG  en-GB  en-CA  en-AU  en-IN  en-SG  en-ZA  he
```

该清单只记录当前完成状态，不重写这些 locale 的历史或实现身份。已实施 locale 的 hostname、CDN、service、port、public URL 与 lifecycle 等 machine fact 继续以 `production/identity.json` 为 authority。

## 2. 最终固定语言范围：65 个 community locale

固定 **65 个 community locale** 已全部完成并进入 `production_state=live`。Hebrew（`he` / עברית）作为第 65 个、也是最后一个固定 community language target，已完成正式实施与首次 Production finalization。

已实施 locale 的 canonical locale、`html_lang`、Autonym、EnglishName、hostname、port、service、public URL、timezone、runtime profile 与 lifecycle 等 machine fact 继续只以 `production/identity.json` 为 authority；本文不复制这些实现字段。

### 已完成的 Google 广告支持非英语商业扩展

| Planning target | Language / regional standard |
| --- | --- |
| `de-AT` | Austrian German |
| `zh-SG` | Singapore Simplified Chinese |

### 已完成的英语商业区域扩展

- `en-GB`
- `en-CA`
- `en-AU`
- `en-IN`
- `en-SG`
- `en-ZA`

当前 Go Tour 官方 English 继续使用 <https://go.dev/tour/> 的 generic `lang="en"`。上述 regional English locale 是独立商业区域扩展，不以取得 go.dev Go local 链接为前提；本项目不新增独立 `en-US` Go Tour。

### 已完成的非 Google Publisher 支持语言

以下 4 门语言已经完成并进入 `live`，应尽早申请 go.dev 的 Go local 链接：

- Swahili
- Kazakh
- Persian
- Amharic

这四项的正式实现身份已由 `production/identity.json` 冻结；本节只保留完成状态和 go.dev 链接申请事项，不复制 machine identity。

### 固定范围边界

固定语言池严格冻结为 65 门，不存在额外条件语言，不建立观察名单，也不继续增加当前未列出的语言或地区变体。特别不得自动增加：

- `nn-NO`
- `en-NZ`
- `en-IE`
- `nl-BE`
- `fr-BE`
- `fr-CH`
- `zh-MY`
- 其他 English regional locales
- Bosnian
- Macedonian
- Icelandic
- 其他仅因“存在合法 locale”而发现的候选

只有维护者明确重新开启全局语言范围评估，才允许改变上述固定语言池。普通实施阻塞、单一市场 policy 变化或发现新的合法 locale 都不得自动扩展、替换或建立候补队列。

## 3. 当前 Go Tour 收口调度

当前 Go Tour 的 **65 个固定 community locale 已全部完成正式语言实施并进入 `live`**，不再存在 remaining locale target，也不得自动重新开放全球候选研究或建立候补队列。

Hebrew 的 Google Search Console 与 Bing Webmaster Tools sitemap 已提交；此前暂缓的 IndexNow closeout 已于 2026-10-06 使用既有 locale-specific key 和正式入口恢复完成，取得 `IndexNow bootstrap: PASS (locale=he sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=he)`。同轮 `et-EE`、`de-CH`、`en-GB`、`en-AU` 的 deferred IndexNow closeout 也全部恢复并取得各自 PASS，仓库记录中的剩余 IndexNow closeout 已全部完成。

当前一次性收口顺序为：

```text
固定 65 个 locale 已全部 live
→ 剩余 IndexNow closeout 已全部完成
→ V2-A Architecture / Content Scope
→ V2-B compatibility / freshness gates
→ V2-D runtime / localization workflow foundation
→ 正式模型 benchmark
→ 连续 learn-docs-v1 locale campaign
→ frozen-baseline Preview / Production
→ 4 个 tracked issue 完成 + 维护者显式重开 V2-C1
→ V2-C1 只执行一次 upstream source sync（当前 deferred）
→ exact source/package/content-unit reconciliation；只处理真实 stale 范围
→ 基于真实运行数据只执行一次跨项目商业排序
→ 冻结以后多语言项目的语言池与默认顺序
```

Hebrew 已于 2026-10-06 取得自身 `IndexNow bootstrap: PASS` 与 `IndexNow closeout: PASS`，此前对应 deferred issue 已核销。当前不再存在 IndexNow closeout blocker。V2-C1 前主动冻结 `db076098077c07d3cef1b85a2cf56ff52777f587` 为正式 English source authority，允许 campaign/review/activation/Preview/Production；不表示今天最新 master，不增加联网 drift gate，不 fetch/pull/sync。V2-C1 等待 [Architecture 第 10 节](SITE_V2_ARCHITECTURE.md#10-upstream-contract-与-deferred-v2-c1) 的四个 issue 全部达到预期状态并显式重开，不阻塞第一批 locale。最终 sync 只精确处理 affected scope，复用 unaffected evidence/V2-B lineage，不改写历史、不默认全量重译。

此前完成的 Swahili、Kazakh、Persian、Amharic 仍应按既有运营计划申请或维护 go.dev Go local 链接；该事项不改变它们已经 `live` 的完成状态。

## 4. Hebrew 已完成正式实施与质量验收

Hebrew（`he` / עברית）已按统一[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)完成正式实施，不存在 Hebrew-only 旁路 workflow。正式流程已覆盖 glossary、UI catalog、article metadata、122 个 TranslationUnit、独立 QC/re-QC、promotion、schema v2 Course SEO、Locale Surface Review、preview、automated rendered acceptance、visual HUMAN gate、publish、first-production machine/browser acceptance 与 finalization。

最终 Locale Surface Review 为 `20261005-he-stage-a-002`，当前完整 package 决定为 `PASS`。A-001 的 5 个真实 defect 已完成正式修订与复核；另 1 个 `_content/js/playground.js` finding 经完整调用链复核确定为当前 Tour 不可达 generic branch 的 false positive，不需要 Hebrew-only runtime workaround 或 shared UI schema 扩展。

Hebrew 的 RTL、bidi / mixed-direction text、inline code、Go identifiers、LTR technical tokens 周围标点、UI、Course SEO、runtime、mobile / desktop rendered surface 均已通过现有统一 gate 验证。正式 `first-production finalize` 已 PASS，`production/identity.json` 当前验证为 65 locales 且 Hebrew 为 `production_state=live`。

Google Search Console 与 Bing Webmaster Tools 已提交 Hebrew 的正式 `/sitemap.xml`。IndexNow 已于 2026-10-06 使用既有 locale-specific key 和正式入口恢复完成，最终 `IndexNow bootstrap: PASS (locale=he sitemap_urls=105 submitted_urls=105)`、`IndexNow closeout: PASS (locale=he)`；未重新生成 key，也未修改 Production 配置。

## 5. Go Tour 完成后的一次性跨项目商业排序与冻结

固定 65 个 community locale 已全部完成。维护者仍只进行 **一次最终跨项目商业排序**；最终排序/冻结可等待 V2-C1 与真实 Site v2 运行数据，不阻塞第一批 Site v2 locale 实施。可使用的数据包括：

- PV；
- Google Search impressions / clicks；
- 国家地区来源；
- RPM / 广告收入；
- go.dev referral；
- 索引表现；
- regional variant 实际增量；
- Generation / Reviewer 成本；
- Production / 长期维护成本。

该排序用于确定以后多语言项目的默认实施顺序，不重新开展全球候选研究，也不自动增加、替换或删除固定范围中的语言。排序完成后，把固定 65 门语言池与跨项目默认顺序一起冻结。

后续新多语言项目直接复用冻结后的范围与顺序，不为每个项目重新做全球语言排名。如果项目 canonical source 已经覆盖某个固定目标，则跳过该目标的 Generation，其余顺序保持不变。对于以 `en-US` 为 canonical source 的项目，`en-US` 作为源语言单独存在，不计入本路线图的 community locale 固定目标。
