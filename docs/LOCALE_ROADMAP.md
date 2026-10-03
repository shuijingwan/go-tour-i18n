# Locale 长期路线图

本文档是社区 locale 的长期实施范围、当前 Go Tour 剩余调度，以及未来多语言项目默认顺序的唯一 authority。[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md) 仍是单个新 locale 的正式执行流程 authority；本文只确定规划范围与调度，不替代 locale identity freeze、语言资产、质量审核、Production 或上线流程。

## 1. 当前已完成：56 个 live community locale

当前仓库已有 **56 个 `production_state=live` community locale**：

```text
zh-CN  ja-JP  de-DE  fr-FR  ko-KR  es-ES  it-IT  nl-NL  pt-BR  tr-TR
sv-SE  pl-PL  zh-TW  id-ID  vi-VN  ar     th-TH  hi-IN  bn-BD  ur-PK
uk-UA  ro-RO  cs-CZ  ta-IN  te-IN  ms-MY  fil-PH el-GR  hu-HU  bg-BG
mr-IN  ml-IN  kn-IN  es-419 gu-IN  pa-IN  nb-NO  da-DK  fi-FI  sr-RS
sk-SK  hr-HR  sl-SI  lt-LT  ca-ES  et-EE  lv-LV
sw-TZ  kk-KZ  fa-IR  am-ET  ru-RU  pt-PT  fr-CA  zh-HK  de-CH
```

该清单只记录当前完成状态，不重写这些 locale 的历史或实现身份。已实施 locale 的 hostname、CDN、service、port、public URL 与 lifecycle 等 machine fact 继续以 `production/identity.json` 为 authority。

## 2. 最终固定语言范围：64 个 community locale

在当前 56 个 live locale 基础上，再固定增加以下 **8 个 locale / language targets**，使 community locale 固定总数达到 **64**。这些目标是长期固定范围，不是可直接用于实现或 Production 的 identity；每个目标正式启动时仍须进入新增 Locale 执行手册，冻结其 canonical locale、`html_lang`、autonym、English name、hostname、port、service、public URL 及其他正式身份。不得根据本表提前猜测或写入这些字段。

### Google 广告支持的非英语商业扩展

| Planning target | Language / regional standard |
| --- | --- |
| `de-AT` | Austrian German |
| `zh-SG` | Singapore Simplified Chinese |

### 英语商业区域扩展

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

除[第 4 节](#4-hebrew-唯一-conditional-gate)的 Hebrew 唯一条件项外，不建立观察名单，也不继续增加当前未列出的语言或地区变体。特别不得自动增加：

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

## 3. 当前 Go Tour 剩余执行调度

当前 Go Tour 的项目调度与未来跨项目商业排序分离，按以下顺序完成剩余目标。

### 第一阶段：已完成，申请 go.dev 链接

1. Swahili
2. Kazakh
3. Persian
4. Amharic

这四门已经完成且为 `live`。尽早提交 go.dev Go local 链接申请，使上游审核等待时间与后续 locale 实施并行；链接申请不阻塞第二阶段继续实施。

### 第二阶段：固定商业扩展

依次完成：

已完成：`ru-RU`、`pt-PT`、`fr-CA`、`zh-HK`、`de-CH`。

剩余目标依次为：

1. `de-AT`
2. `zh-SG`
3. `en-GB`
4. `en-CA`
5. `en-AU`
6. `en-IN`
7. `en-SG`
8. `en-ZA`

`ru-RU` 不再作为阻塞后续 locale 的路线图 gate。若某个目标在正式执行中出现真实 failure，按对应 runbook 保存 evidence 和恢复，不因此重新开放全球候选研究或从固定池中自动寻找替代项。

每个 locale 的具体启动、Generation / Reviewer、质量 gate、Production 与上线仍完全服从新增 Locale 执行手册及其引用规范。本路线图不授权提前创建 locale、冻结 Production identity 或跳过任何 gate。

### 第三阶段：Hebrew feasibility experiment

固定 64 个 community locale 全部完成后，最后执行[第 4 节](#4-hebrew-唯一-conditional-gate)规定的一次 Hebrew feasibility experiment。

当前项目的全部目标与 Hebrew gate 处理完毕后，再进行一次统一 upstream source sync；不在每批新增语言之间重复同步。若 Hebrew PASS，先完成其正式新增 locale 流程再做该次统一 sync；若 FAIL，则在记录停止结论后进入统一 sync。

## 4. Hebrew 唯一 conditional gate

Hebrew 是固定 64 个 community locale 之外唯一的 conditional language。只有前述 64 个 locale 全部完成后，才执行一次 translation feasibility experiment：

- Generation 使用 GPT-5.6 Sol + High；
- Reviewer 使用与 Generation 独立的 GPT-5.6 Sol + High session；
- 验证自然语言、Go 技术术语、RTL、inline code / Go identifiers、UI / SEO 等 Production 级质量；
- experiment 只决定是否进入正式实施，不能替代正式 Glossary Review、TranslationUnit Quality Check、Locale Surface Review 或其他正式 QC。

结果只有两条路径：

- **PASS**：Hebrew 才进入新增 Locale 执行手册，届时再冻结全部正式 identity 并执行完整 gate；完成后 community locale 总数最多为 **65**。
- **FAIL**：停止，不实施 Hebrew，不建立长期 watchlist，也不以其他语言替补。

本次路线图不预先指定 Hebrew 的 canonical locale、`html_lang`、autonym 或任何 Production identity。

## 5. Go Tour 完成后的一次性跨项目商业排序与冻结

Go Tour 的固定 64 个 locale、Hebrew gate，以及 PASS 后的 Hebrew 正式实施（如适用）全部完成并取得真实运行数据后，维护者只进行 **一次最终跨项目商业排序**。可使用的数据包括：

- PV；
- Google Search impressions / clicks；
- 国家地区来源；
- RPM / 广告收入；
- go.dev referral；
- 索引表现；
- regional variant 实际增量；
- Generation / Reviewer 成本；
- Production / 长期维护成本。

该排序用于确定以后多语言项目的默认实施顺序，不重新开展全球候选研究，也不自动增加、替换或删除固定范围中的语言。排序完成后，将 Hebrew gate 的最终结果纳入实际语言池，并把语言池与跨项目默认顺序一起冻结。

后续新多语言项目直接复用冻结后的范围与顺序，不为每个项目重新做全球语言排名。如果项目 canonical source 已经覆盖某个固定目标，则跳过该目标的 Generation，其余顺序保持不变。对于以 `en-US` 为 canonical source 的项目，`en-US` 作为源语言单独存在，不计入本路线图的 community locale 固定目标。
