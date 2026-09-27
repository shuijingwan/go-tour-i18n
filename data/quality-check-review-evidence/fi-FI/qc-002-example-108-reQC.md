# fi-FI TranslationUnit re-QC — qc-002 Example stable index 108

- 独立 Reviewer：ChatGPT GPT-5.6 Sol High；没有参与 fi-FI Generation、revision 或 replacement。
- Snapshot：`qc-002`，full 122 Unit（103 Page + 19 Example）；predecessor=`qc-001`。
- Rubric：`translation-quality/v1`；本轮唯一 Unit：`example:concurrency/exercise-web-crawler.go`，stable index=108。
- Revision batch：`chatgpt-fi-FI-005`，attempt=1；qc-001 previous rating=B，finding=F02。
- 当前 Glossary SHA-256：`040bb534d9216c810b6ffeb92efc1c33e536ab003fc848668a9da2efc826c134`。
- 本次 Reviewer ZIP：`/tmp/fi-FI-qc-002-example-108-reQC-reviewer.zip`，SHA-256=`fa89019ebe2475e9a6d842a07c3298a2484e6a43dbeb5f1ed96c57aac36c9143`。
- Bundle identity：`4d552312b7d82be511663d434941da3a607c9ea6b37fd623aee230fd19146019`；12/12 manifest 成员长度及 SHA-256 匹配，ZIP CURRENT。
- 完整源码 SHA-256：`bd6226dcae3b663a3aa1cbf502d840357ac0efb6de46e7a66612ca2834cf5e37`；完整芬兰语 candidate SHA-256：`257fddb57784948a2c8b99b7282b6a119ed4b2c2b1eeaba6f8bc1fc52b8f6897`；validation SHA-256：`a98eb278424bb1f0c7563114986acdf56d0c50753ed1fda4bc3c7da3ee9f6cc0`。
- 按完整的正式 ZIP authority、glossary、122-Unit Snapshot、batch manifest、受保护英文输入、自动验证 evidence 与本 Unit 的完整英文 Go source/芬兰语 target 独立审核。validation=passed 是机器证据，不等于语言评级。
- 本轮 preflight：incremental，`qc-001` 已持久绑定，carry-forward=120；已记录 Page 31 A，本 Unit 是唯一 pending。

## F02 原缺陷与修订检查

### 1. Fetch 的 URL、网页正文及发现的 URL 列表

英文原文：

> // Fetch returns the body of URL and
> // a slice of URLs found on that page.

正式修订译文：

> // Fetch palauttaa URL-osoitetta vastaavan verkkosivun sisällön ja
> // viipaleen tältä sivulta löytyvistä URL-osoitteista.

判定：修订后的 `URL-osoitetta vastaavan verkkosivun sisällön` 明确是该 URL 对应网页的内容，`tältä sivulta` 指向同一页面；`viipale` 与 URL 列表的技术关系忠实，消除了原先不自然、易生歧义的 `sivun URL sisällön`。

### 2. Crawl 起点、递归与最大深度

英文原文：

> // Crawl uses fetcher to recursively crawl
> // pages starting with url, to a maximum of depth.

正式修订译文：

> // Crawl käyttää fetcher-muuttujaa ja käy rekursiivisesti läpi
> // sivuja url-osoitteesta alkaen enintään depth-muuttujan määrittämään syvyyteen.

判定：原来的深度表达已重构，准确区分 `url` 起点、递归处理页面和 `depth` 变量限定的最大深度；没有改变或推断算法的新行为。

## 完整 Example 逐行审核

完整英文/芬兰语文件均为 87 行，9 行差异均局限于相应可翻译的自然语言注释正文，原注释分隔符、前缀缩进、代码行、`//go:build OMIT`、package/import、类型/函数签名、`TODO` 标志、错误与返回处理、字符串、URL、`fakeFetcher` 数据均未改变。逐一核对所有 9 个注释的双语证据：

| 原文件行号 | 英文原文 | 当前芬兰语 | 结论 |
|---:|---|---|---|
| 10 | `// Fetch returns the body of URL and` | `// Fetch palauttaa URL-osoitetta vastaavan verkkosivun sisällön ja` | A |
| 11 | `// a slice of URLs found on that page.` | `// viipaleen tältä sivulta löytyvistä URL-osoitteista.` | A |
| 15 | `// Crawl uses fetcher to recursively crawl` | `// Crawl käyttää fetcher-muuttujaa ja käy rekursiivisesti läpi` | A |
| 16 | `// pages starting with url, to a maximum of depth.` | `// sivuja url-osoitteesta alkaen enintään depth-muuttujan määrittämään syvyyteen.` | A |
| 18 | `// TODO: Fetch URLs in parallel.` | `// TODO: Hae URL-osoitteet rinnakkain.` | A |
| 19 | `// TODO: Don't fetch the same URL twice.` | `// TODO: Älä hae samaa URL-osoitetta kahdesti.` | A |
| 20 | `// This implementation doesn't do either:` | `// Tämä toteutus ei tee kumpaakaan:` | A |
| 40 | `// fakeFetcher is Fetcher that returns canned results.` | `// fakeFetcher on Fetcher, joka palauttaa ennalta määritetyt tulokset.` | A |
| 55 | `// fetcher is a populated fakeFetcher.` | `// fetcher on valmiiksi täytetty fakeFetcher.` | A |

英文中 `TODO: Fetch URLs in parallel`、`TODO: Don't fetch the same URL twice` 和 `This implementation doesn't do either` 分别保持为并行抓取、URL 去重及当前实现均未实现的要求。`fakeFetcher` 仍代表提供预置结果的 Fetcher，`fetcher` 仍是填充数据的 fakeFetcher。词形、复合词、glossary 实际适用项及教学语言均达到可发布质量。

## 正式评级

- `example:concurrency/exercise-web-crawler.go`：**A**。
- F02：RESOLVED；无遗留或新增 B/C/D finding。
- 本次不生成 replacement，不重新审核 Page 31，不改动 qc-001/002 历史记录。首次记录完成后应确认 full Snapshot 的 120 carry-forward + qc-002 两条 A = 122 A，pending=0；仅当正式 scope PASS 方可执行 machine finalization。
