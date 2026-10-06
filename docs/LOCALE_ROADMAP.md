# Locale 长期路线图

## Versionless locale authority（V2-D 收敛）

v1/v2/v3 是 implementation/content-package milestone，不是 locale language governance namespace。Content package 可以版本化；Locale language system 不版本化。全站只有 `locales/<locale>/glossary.yaml`，不得 Tour/Docs/v2 split。current executable migration、统一 corpus、一次完整 glossary generation/refresh、统一 structured assets、完整 Page TU 与 delta-aware integrated Surface contract 见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。本节是全站扩展入口；下文旧 Tour workflow/evidence 保持兼容，不授权重复初始化 locale 或改写历史证据。

已有 Tour-complete locale 首次扩展：一次 unified glossary refresh → 一次独立 full unified Review → compatibility → 一次 pending structured-assets Generation → 两批 Learn/Docs Page Generation / 两批 Page QC → 一次 integrated Surface Review。unchanged Tour/UI/meta/SEO carry；只有真实 affected scope reopen。新 locale 从零执行同一 locale campaign，glossary 与 structured-assets 各一次，Tour 和 Learn/Docs 各用其 surface-specific parser/TU batches；没有第二次“V2 locale init”。Course SEO 是 Page-derived consumer，按自己的 canonical source/ready target contract 单列，不强行并入 structured-assets；没有 Learn/Docs canonical SEO authority时不创建 SEO target。

新 unified receipt `go-learning/unified-glossary-review/v1` 与 old v1/Tour legacy coverage additive 共存。新 Generation 要求 current unified coverage；旧 completion仍按旧 evidence合法验证。unified current exact优先；多个 current unified receipts、glossary/corpus mismatch fail closed。旧 receipt immutable，不 retroactively upgrade corpus coverage。glossary bytes不变但 corpus变了仍需一次 full unified Review；不猜造 corpus compatibility shortcut。新 glossary byte change须 full unified PASS后才用 V2-B semantic delta/精确 lineage复用旧结果。


本文档是当前 Go 项目固定 65 个 community locale 的长期实施范围、当前调度，以及跨项目 66-locale 商业目标池与默认执行顺序的唯一 planning authority。[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md) 仍是单个新 locale 的正式执行流程 authority；本文只确定规划范围与调度，不替代 locale identity freeze、语言资产、质量审核、Production 或上线流程。

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

当前 Go 项目的 community 语言池严格冻结为 65 门，不存在额外条件语言，不建立观察名单，也不继续增加当前未列出的语言或地区变体。第 5 节独立的跨项目 commercial target pool 增加 `en-US`，不扩展当前 Go scope。特别不得自动增加：

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
→ 按第 5 节冻结商业顺序连续 learn-docs-v1 locale campaign（Go 跳过 en-US，从 zh-CN 开始）
→ frozen-baseline Preview / Production
→ 4 个 tracked issue 完成 + 维护者显式重开 V2-C1
→ V2-C1 只执行一次 upstream source sync（当前 deferred）
→ exact source/package/content-unit reconciliation；只处理真实 stale 范围
→ 足够真实运行数据存在后，只执行一次 empirical re-ranking
→ 以正式实证重排替换当前 pre-real-data 冻结顺序，语言池保持不变
```

Hebrew 已于 2026-10-06 取得自身 `IndexNow bootstrap: PASS` 与 `IndexNow closeout: PASS`，此前对应 deferred issue 已核销。当前不再存在 IndexNow closeout blocker。V2-C1 前主动冻结 `db076098077c07d3cef1b85a2cf56ff52777f587` 为正式 English source authority，允许 campaign/review/activation/Preview/Production；不表示今天最新 master，不增加联网 drift gate，不 fetch/pull/sync。V2-C1 等待 [Architecture 第 10 节](SITE_V2_ARCHITECTURE.md#10-upstream-contract-与-deferred-v2-c1) 的四个 issue 全部达到预期状态并显式重开，不阻塞第一批 locale。最终 sync 只精确处理 affected scope，复用 unaffected evidence/V2-B lineage，不改写历史、不默认全量重译。

此前完成的 Swahili、Kazakh、Persian、Amharic 仍应按既有运营计划申请或维护 go.dev Go local 链接；该事项不改变它们已经 `live` 的完成状态。

## 4. Hebrew 已完成正式实施与质量验收

Hebrew（`he` / עברית）已按统一[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)完成正式实施，不存在 Hebrew-only 旁路 workflow。正式流程已覆盖 glossary、UI catalog、article metadata、122 个 TranslationUnit、独立 QC/re-QC、promotion、schema v2 Course SEO、Locale Surface Review、preview、automated rendered acceptance、visual HUMAN gate、publish、first-production machine/browser acceptance 与 finalization。

最终 Locale Surface Review 为 `20261005-he-stage-a-002`，当前完整 package 决定为 `PASS`。A-001 的 5 个真实 defect 已完成正式修订与复核；另 1 个 `_content/js/playground.js` finding 经完整调用链复核确定为当前 Tour 不可达 generic branch 的 false positive，不需要 Hebrew-only runtime workaround 或 shared UI schema 扩展。

Hebrew 的 RTL、bidi / mixed-direction text、inline code、Go identifiers、LTR technical tokens 周围标点、UI、Course SEO、runtime、mobile / desktop rendered surface 均已通过现有统一 gate 验证。正式 `first-production finalize` 已 PASS，`production/identity.json` 当前验证为 65 locales 且 Hebrew 为 `production_state=live`。

Google Search Console 与 Bing Webmaster Tools 已提交 Hebrew 的正式 `/sitemap.xml`。IndexNow 已于 2026-10-06 使用既有 locale-specific key 和正式入口恢复完成，最终 `IndexNow bootstrap: PASS (locale=he sitemap_urls=105 submitted_urls=105)`、`IndexNow closeout: PASS (locale=he)`；未重新生成 key，也未修改 Production 配置。

## 5. 跨项目 66-locale 广告收入潜力默认执行顺序

维护者于 **2026-10-06** 冻结以下 **pre-real-data expected-ad-revenue commercial execution order**。它从现在开始用于当前 Site v2 locale campaign、future content package，以及未来新的多语言项目；当前已有正式默认顺序，不等待 V2-C1 或真实数据才开始调度。

### 两个独立范围与 source skip rule

当前 Go 项目固定 community scope 仍为 **65 个 live locale**。独立的 cross-project commercial target pool 为 **66 = 当前 65 + `en-US`**：加入 `en-US` 是为了让目标池跨未来不同 canonical source language 的项目复用，不能将美国英语预先排除为所有项目的源语言。

当前官方 Go English authority 仍是 <https://go.dev/tour/> 的 generic `lang="en"`。`en-US` 只是 cross-project target，不是 Go Production locale；不新增 Go hostname、Production identity 或 Tour translation，不修改 65 locale registry、GoLocal、Production 或 routing semantics，也不将 generic `en` technical authority 转换为 `en-US`。

以后新多语言项目默认复用这 66 个目标。**canonical source 已覆盖某个目标 → 跳过该目标，其余 locale 保持原有相对顺序**，不重新编号 base rank。例如：

- source = `en-US`：跳过 rank 1 `en-US`，下一门 `zh-CN`。
- source = `zh-CN`：`en-US` 仍 rank 1，到 `zh-CN` 时跳过，之后继续 `en-GB`。
- source = `ja-JP`：`en-US` → `zh-CN` → `en-GB`，跳过 `ja-JP`，之后继续 `de-DE`。

当前 Go 项目明确采用 scheduling-only rule：generic `en` 视为已经覆盖 cross-project `en-US` target。因此当前 Site v2 / future Go package campaign **不生成 `en-US`，跳过 rank 1，从 rank 2 `zh-CN` 开始**。这只是 commercial scheduling rule，不是 source identity conversion。

### 唯一 ranking objective 与 proxy

基础商业顺序只优化一个最终目标：**长期广告收入潜力由高到低**。它不是语言重要性、人口、GDP、CPM、Google AdSense 支持语言、GoLocal、go.dev 链接申请优先级、翻译成本、Generation 成本或 Reviewer 成本排名。

在没有真实 locale PV / RPM / fill rate / revenue 数据时，planning 使用：

```text
Expected Ad Revenue ≈ Potential Traffic × Monetization Value per Visit
```

语言 / 市场人口规模、互联网受众、主要国家 / 地区、经济体量与购买力、数字广告市场成熟度、技术 / developer audience、regional market coverage 都只是解释预期广告收入可能更高或更低的 proxy；最终 planning metric 仍只有广告收入潜力。

### 平台支持与项目级生态 override

基础商业 ranking 不按单一广告平台重新降权。Google Publisher 当前不支持某语言、Google 当前地区限制、GoLocal Tour 不允许广告、某 locale 需要其他广告平台或某项目尚无合适 monetization provider，都不得改变本次基础顺序。`ru-RU`、`fa-IR`、`sw-TZ`、`kk-KZ`、`am-ET` 等按语言市场本身的长期商业潜力排序。未来可使用 Google、regional/local ad networks、direct sponsorship、affiliate 或其他 monetization mechanism；具体平台不属于基础商业 ranking，也不改变现有项目 publication / advertising policy。

**`sw-TZ` / `kk-KZ` / `fa-IR` / `am-ET` 保留项目级战略 override**：在 go.dev Go local link、官方目录、生态合作、社区推荐或官方语言入口等场景中，覆盖较少的语言可能拥有高于广告收入排名的战略价值。只有具体项目存在维护者明确确认并授权的 official-link / ecosystem opportunity，才可临时将这四门中的相关 locale 提前。override 只影响该项目实际实施顺序，**永不修改 66-locale base commercial ranking**；没有明确生态需求时继续按基础顺序实施。

### 冻结的 exact 66-locale base commercial ranking

| Rank | Locale | 中文语言名称 | Commercial tier |
| ---: | --- | --- | --- |
| 1 | en-US | 英语（美国） | S+ |
| 2 | zh-CN | 简体中文（中国大陆） | S |
| 3 | en-GB | 英语（英国） | S |
| 4 | ja-JP | 日语 | S |
| 5 | de-DE | 德语（德国） | S |
| 6 | en-CA | 英语（加拿大） | S |
| 7 | fr-FR | 法语（法国） | S |
| 8 | en-AU | 英语（澳大利亚） | S |
| 9 | ko-KR | 韩语 | S |
| 10 | es-419 | 西班牙语（拉丁美洲） | S |
| 11 | pt-BR | 葡萄牙语（巴西） | S |
| 12 | ar | 阿拉伯语 | A+ |
| 13 | it-IT | 意大利语 | A+ |
| 14 | es-ES | 西班牙语（西班牙） | A+ |
| 15 | nl-NL | 荷兰语 | A+ |
| 16 | ru-RU | 俄语 | A+ |
| 17 | zh-TW | 繁体中文（台湾） | A+ |
| 18 | de-CH | 德语（瑞士） | A |
| 19 | sv-SE | 瑞典语 | A |
| 20 | en-IN | 英语（印度） | A |
| 21 | id-ID | 印度尼西亚语 | A |
| 22 | fr-CA | 法语（加拿大） | A |
| 23 | de-AT | 德语（奥地利） | A |
| 24 | nb-NO | 书面挪威语（Bokmål） | A |
| 25 | zh-HK | 繁体中文（香港） | A |
| 26 | da-DK | 丹麦语 | A |
| 27 | en-SG | 英语（新加坡） | A |
| 28 | pl-PL | 波兰语 | A |
| 29 | he | 希伯来语 | A |
| 30 | cs-CZ | 捷克语 | A |
| 31 | tr-TR | 土耳其语 | B+ |
| 32 | th-TH | 泰语 | B+ |
| 33 | pt-PT | 葡萄牙语（葡萄牙） | B+ |
| 34 | fi-FI | 芬兰语 | B+ |
| 35 | zh-SG | 简体中文（新加坡） | B+ |
| 36 | hi-IN | 印地语 | B+ |
| 37 | vi-VN | 越南语 | B+ |
| 38 | ms-MY | 马来语 | B |
| 39 | en-ZA | 英语（南非） | B |
| 40 | ca-ES | 加泰罗尼亚语 | B |
| 41 | fil-PH | 菲律宾语 | B |
| 42 | hu-HU | 匈牙利语 | B |
| 43 | el-GR | 希腊语 | B |
| 44 | ro-RO | 罗马尼亚语 | B |
| 45 | bn-BD | 孟加拉语 | B− |
| 46 | ta-IN | 泰米尔语 | B− |
| 47 | te-IN | 泰卢固语 | B− |
| 48 | uk-UA | 乌克兰语 | B− |
| 49 | fa-IR | 波斯语 | B− |
| 50 | bg-BG | 保加利亚语 | C+ |
| 51 | ml-IN | 马拉雅拉姆语 | C+ |
| 52 | mr-IN | 马拉地语 | C+ |
| 53 | kn-IN | 卡纳达语 | C+ |
| 54 | gu-IN | 古吉拉特语 | C+ |
| 55 | pa-IN | 旁遮普语（果鲁穆奇文，印度） | C+ |
| 56 | ur-PK | 乌尔都语 | C+ |
| 57 | kk-KZ | 哈萨克语 | C |
| 58 | hr-HR | 克罗地亚语 | C |
| 59 | sk-SK | 斯洛伐克语 | C |
| 60 | sl-SI | 斯洛文尼亚语 | C |
| 61 | lt-LT | 立陶宛语 | C |
| 62 | et-EE | 爱沙尼亚语 | C |
| 63 | lv-LV | 拉脱维亚语 | C |
| 64 | sr-RS | 塞尔维亚语（西里尔文） | C |
| 65 | sw-TZ | 斯瓦希里语（坦桑尼亚） | D |
| 66 | am-ET | 阿姆哈拉语 | D |

Commercial tier `S+` / `S` / `A+` / `A` / `B+` / `B` / `B−` / `C+` / `C` / `D` 仅帮助维护者理解大致广告收入潜力梯队，不是 machine quality gate，不需要 schema，不影响 Translation / Reviewer / Production gate。相邻 locale 的精确名次不代表统计学上的显著差异。维护者接受：若整个语言池可在约两个月完成，rank 34 与 rank 38 的细微误差不值得继续投入大量研究时间。

此顺序按维护者决定逐项冻结，不调整相邻名次、不替换 locale、不增加候补、不删除目标、不重新开放全球候选研究。不得继续优化、重新研究、重新计算或质疑排序；不建立 JSON schema、CLI、validator、runtime registry 或第二个独立排序 authority。

## 6. 未来一次性 empirical re-ranking

当前第 5 节是已生效的 **pre-real-data frozen execution order**；未来真实数据排序是独立的长期 contract。V2-C1 仍 deferred，只有既有四个 tracked issue 达到维护者预期且维护者显式重开才执行，不因本次 planning 决策改变条件。

V2-C1 后有足够真实运行数据时，仍允许进行 **一次真实数据驱动的最终 empirical re-ranking**。可使用 PV、actual RPM / revenue、fill rate、referral（含 go.dev）、indexing（含 Search impressions / clicks）、国家地区来源、regional variant incremental value、Production / 长期 maintenance cost 与实测 Generation / Reviewer 成本，复核商业价值；这些是未来实证复核输入，不是当前基础顺序的成本排名。

在此之前，不得每天、按新的 CPM 文章或单一广告平台 policy 反复调整当前 66 顺序。未来只有正式的 one-time empirical re-ranking 可以替换当前顺序；不重新开展全球候选研究，不自动增加、替换或删除目标。后续项目直接复用当前冻结范围与顺序，并遵守 canonical source skip rule；未来正式实证重排后复用该次更新后的顺序。
