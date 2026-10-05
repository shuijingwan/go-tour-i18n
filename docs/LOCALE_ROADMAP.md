# Locale 长期路线图

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

Hebrew 的 Google Search Console 与 Bing Webmaster Tools sitemap 已提交；IndexNow provisioning 已成功，但公网 key verification 在两次正式尝试中分别遇到 `EOF` 与 `connection reset by peer`，因此 IndexNow closeout 暂缓并按 `docs/DEFERRED_ISSUES.md` 中的真实 issue evidence 后续恢复。该失败不回滚 Hebrew 已通过的 Surface Review、preview、first-production machine/browser acceptance 或 `production_state=live`。

当前一次性收口顺序为：

```text
固定 65 个 locale 已全部 live
→ 完成 Hebrew 暂缓的 IndexNow closeout
→ 只执行一次 upstream source sync
→ 基于真实运行数据只执行一次跨项目商业排序
→ 冻结以后多语言项目的语言池与默认顺序
```

在 Hebrew IndexNow closeout 尚未取得自身 `IndexNow bootstrap: PASS` 与 `IndexNow closeout: PASS` 前，不把它伪记为 search-engine closeout 完成；也不因该非阻塞第三方 submission failure 重开 locale Production lifecycle。

此前完成的 Swahili、Kazakh、Persian、Amharic 仍应按既有运营计划申请或维护 go.dev Go local 链接；该事项不改变它们已经 `live` 的完成状态。

## 4. Hebrew 已完成正式实施与质量验收

Hebrew（`he` / עברית）已按统一[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)完成正式实施，不存在 Hebrew-only 旁路 workflow。正式流程已覆盖 glossary、UI catalog、article metadata、122 个 TranslationUnit、独立 QC/re-QC、promotion、schema v2 Course SEO、Locale Surface Review、preview、automated rendered acceptance、visual HUMAN gate、publish、first-production machine/browser acceptance 与 finalization。

最终 Locale Surface Review 为 `20261005-he-stage-a-002`，当前完整 package 决定为 `PASS`。A-001 的 5 个真实 defect 已完成正式修订与复核；另 1 个 `_content/js/playground.js` finding 经完整调用链复核确定为当前 Tour 不可达 generic branch 的 false positive，不需要 Hebrew-only runtime workaround 或 shared UI schema 扩展。

Hebrew 的 RTL、bidi / mixed-direction text、inline code、Go identifiers、LTR technical tokens 周围标点、UI、Course SEO、runtime、mobile / desktop rendered surface 均已通过现有统一 gate 验证。正式 `first-production finalize` 已 PASS，`production/identity.json` 当前验证为 65 locales 且 Hebrew 为 `production_state=live`。

Google Search Console 与 Bing Webmaster Tools 已提交 Hebrew 的正式 `/sitemap.xml`。IndexNow 当前仅剩非阻塞 closeout 恢复：保留已有 locale-specific key 和已成功 provisioning，后续只使用正式入口继续，不重新生成 key、不重复修改 Production 配置。

## 5. Go Tour 完成后的一次性跨项目商业排序与冻结

固定 65 个 community locale 全部完成、Hebrew 已按完整正式流程进入 `live`、一次统一 upstream source sync 完成并取得真实运行数据后，维护者只进行 **一次最终跨项目商业排序**。可使用的数据包括：

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
