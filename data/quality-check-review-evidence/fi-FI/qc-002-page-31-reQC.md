# fi-FI · qc-002 Page 31 定向 TranslationUnit re-QC

- 独立 Reviewer：ChatGPT GPT-5.6 Sol High；未参与 fi-FI Generation、revision 或 replacement。
- 本轮唯一 Unit：stable index **31**，`flowcontrol/9`，kind=`page`。
- Candidate Snapshot：`qc-002`，完整 122 TU；predecessor=`qc-001`，精确 carry-forward=120，pending=2（Page 31 与 Example 108）。
- Revision batch：`chatgpt-fi-FI-004`，attempt=1；原 qc-001 评级 C，原 finding F01。
- Rubric：`translation-quality/v1`。本轮独立结论：**A，F01 RESOLVED，无新增 finding**。
- Reviewer ZIP SHA-256：`650af9ddd8e40978dd9dfee64b20594be366fb3b5d9ed12cfbce7591829a76a7`。
- Bundle identity：`9387a09a72e595ba622517980f4dc9ffe45c368960de7c603bae83bd8fab6394`。
- 当前候选 SHA-256：`bfde779da430709e9b01b501cb302cf4b1d47d58e84f118fbc8c00909939a9ce`，与正式 Snapshot 和 Reviewer ZIP current_target 完全一致。
- ZIP 13 个成员中 manifest 所列 12/12 全部通过字节长度和 SHA-256 核验；完整 authority、glossary、formal batch manifest、英文受保护 input、122 TU Snapshot、validation 及 quality-check context 已检查。validation.status=`passed`，只作为机器证据，不代替本次独立语言判断。

## 独立技术和语言审核

### 原文与修订后译文：F01

英文：

> Another important difference is that Go's switch cases need not be constants, and the values involved need not be integers.

修订后芬兰语（完整对应句）：

> Toinen tärkeä ero on, että Go-kielessä switch-lauseen tapausten ei tarvitse olla vakioita eikä vertailussa käytettävien arvojen tarvitse olla kokonaislukuja.

原来 `eivätkä niiden arvojen kokonaislukuja` 缺少必要谓语，后半项含义不完整；修订后 `eikä vertailussa käytettävien arvojen tarvitse olla kokonaislukuja` 形成正确的芬兰语并列结构，明确表述参与比较的值不必是整数。`switch` case 不必是常量的第一项亦保持完整。F01 确认解决。

### 全 Page 复审

- `switch` 是连续 `if`/`else` 的紧凑形式，匹配首个等于条件表达式的 case：语义准确，标题与术语正确。
- 与 C/C++/Java/JavaScript/PHP 对比：Go 只执行被选中 case，不会默认贯穿后续 case；相当于自动提供 `break`：忠实完整。
- 两项额外特征：case 不必是常量，比较值不必为整数：均忠实且技术准确。
- 芬兰语：句法、复合词、格变化与教程语气均符合要求，未发现增译、漏译或不自然到需要 B/C/D 修订的问题。
- Go Present：标题 `* Switch`、完整顶层 Section、受保护的 `switch`/`if`/`else`/`break` 技术身份、行内代码、段落及 `.play flowcontrol/switch.go` 均保留。基于实际 present 词边界处理，不将合法原始标记、字面间距误作可见语言缺陷。
- 正式 glossary mandatory/preferred/forbidden/keep 均已对本 Page 实际适用项核对；没有 forbidden 表达。

**本 Unit 的正式评级：A。** 没有遗留 finding，不提供 replacement，也不审核 index 108 的 Example。

## 记录边界

本次仅记录 qc-002 的 `flowcontrol/9`，第一次 record 必须携带 `--previous-snapshot-id qc-001`。后续 Example 108 需要新的用户请求和独立 re-QC。未修改旧 Snapshot、旧 evidence 或生成候选。
