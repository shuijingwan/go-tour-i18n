# 翻译任务规范

本规范定义 A Tour of Go 多语言翻译任务的输入、输出和结构约束。它不定义命令操作、自动校验、质量审核、promotion 或具体术语译法。

## 1. Translation Unit

翻译任务以一个完整 workflow Translation Unit 为最小单位。Translation Unit 有两种：

- **Page**：一个完整顶层 `present.Section`。
- **Example**：一个完整 `.go` 文件。

不得把一个 workflow unit 拆分为句子、段落或多个独立模型输出。一个 unit 对应一份输入、一次原始模型输出和一份恢复后的 candidate。

## 2. Page 翻译任务

Page 的源文件片段使用 `.article`，其流程中的 artifact 语义如下：

| 阶段 | 格式 |
| --- | --- |
| source | `.article` |
| model input | 受保护的 `.article` |
| model output | `raw-responses/*.article` |
| candidate | restore 后的 `.article` |

模型应翻译 Page 中开放的自然语言，并保持完整的 `present.Section`。模型必须保留链接 target、present directive、行内代码、preformatted 的代码与结构以及其他受保护内容；不得删除、伪造、复制或改写保护 token 所代表的内容。

### Page 标题

Page 标题属于该 TranslationUnit 的可翻译自然语言。标题行必须保留 `* ` 这一 present 语法前缀，标题中的普通自然语言必须翻译；其中的 Go 关键字、技术标识符、glossary `keep` 内容及其他不可翻译技术身份保持原样。不得把“保留 `* ` 前缀”解释为“保留英文标题”。模型可以按目标语言的自然语序调整标题和正文中的普通文本，但不得改变页面结构或技术含义。

例如：

```text
source: * Switch with no condition
ja-JP:  * 条件なしの switch

source: * Switch
ja-JP:  * Switch
```

第二个标题可以保留，因为 `switch` 是 Go 关键字；第一个标题中的普通自然语言仍须翻译。

### ko-KR 行内字体 span 的句法重构

本节只适用于 ko-KR Page 中可翻译的自然语言。legacy `golang.org/x/tools/present` 的 inline code 和 emphasis 字体解析依赖词边界；当 closing font span 后直接附着 Hangul 助词或语尾时，restore 为保持字体 span 可能插入空格。因此，这类边界必须在翻译阶段通过自然的韩语句法重构解决。

- 不得为了通过 validator 人为插入空格、零宽字符或其他字符。
- 不得修改 inline code、emphasis 或 protected technical identity。
- 当自然韩语原本需要在 font span 后直接附着助词或语尾时，应改用自然的独立韩语名词，或采用其他不改变原意的句法重构。
- 同样必须注意 Hangul 紧邻 opening font span 左侧的情况，并优先自然重构句子。
- 必须首先保证语义准确和韩语自然度；不得为了结构安全产生翻译腔，也不得新增原文不存在的技术解释。

例如，可以根据具体上下文采用以下句法方向：

```text
`PageUp` 키를 사용합니다
`Vertex` 타입이고
`v` 식별자인
채널은 *버퍼* 기능을 지원합니다
`make` 함수의 두 번째 인수
```

这些只是句法重构示例，不是固定术语映射；实际译文仍须根据原文语义和上下文自然组织。

### Page preformatted 中的 teaching comment

Page 的 preformatted block 不是一律禁止翻译。对于 protector 能安全识别的 Go preformatted，普通 `//` teaching comment 的自然语言 body 属于开放翻译区域，应当翻译；comment delimiter、缩进和换行等结构仍受保护。

下列内容继续保持受保护或不可翻译：

- Go 代码、语法和整体布局；
- comment delimiter 与 block comment；
- 标识符，以及 teaching comment 中引用代码身份的标识符；
- `// int`、`// OK`、`// len(...)` 等机器语义或静态技术注释；
- protector 判定不能安全开放的其他 preformatted 内容。

Automatic validator 应允许安全 Go preformatted 中已开放的 teaching comment body 使用目标语言，不得仅因自然语言注释被翻译而报告违规；代码、结构、标识符和不可翻译内容仍必须保持不变。

对于 ko-KR teaching comment，原样保留的完整 ASCII Go identifier 后可以按正常韩语语法直接附着一个或多个 Hangul 字符，不要求为了 validator 插入空格、零宽字符、反引号或额外名词。该后缀不属于 Go identifier 的技术身份；ASCII identifier 自身的字节、大小写、数量、顺序和所属注释仍必须保持不变，追加 ASCII 字母、数字或下划线仍属于标识符改写并必须拒绝。该窄规则只适用于 ko-KR candidate 的可翻译 teaching comment body，不放宽非注释 Go code、其他 protected structure 或其他 locale 的 identifier 校验。

## 3. Example 翻译任务

Example 的源是完整 `.go` 文件，其流程中的 artifact 语义如下：

| 阶段 | 格式 |
| --- | --- |
| source | `.go` |
| model input | 受保护的 `.txt` |
| model output | `raw-responses/*.txt` |
| candidate | restore 后的 `.go` |

模型不是生成新的 Go 文件。模型只能翻译允许翻译的自然语言注释。

模型必须保持以下内容不变：

- Go 语法；
- `package` 和 import；
- 标识符；
- 字符串；
- 文件结构与布局；
- 代码、注释分隔符、机器语义注释及其他非翻译内容。

模型输出的 `.txt` 是受保护文本表示；只有工作流 restore 后形成的 `.go` 文件才是 candidate。

## 4. 输入契约

一次正式 TranslationUnit 翻译的模型输入由以下三部分共同构成：

1. `data/retranslation-runs/<locale>/<batch-id>/manifest.json`；
2. manifest 列出的 `inputs/*` 文件；
3. `locales/<locale>/glossary.yaml`。

manifest 是任务身份的权威来源，记录 locale、batch、Translation Unit、source 身份、input 路径与保护 token 数量。新流程的 revision feedback 只来自 Quality Check B/C/D 的 rating 和 finding；它不属于 promotion evidence，也不替代新 candidate 的 QC/finalization。历史 manifest 中的 Final Review feedback 字段保持可读，不被重写。模型必须读取真实反馈，只处理 manifest 列出的 unit，并将输出写入对应文件。

`manifest.json`、manifest 列出的全部 `inputs/*` 与 `locales/<locale>/glossary.yaml` 是不可拆分的正式输入。任何一部分缺失，都不属于合规的正式 TranslationUnit 翻译执行。Glossary 必须在模型开始翻译前读取并用于生成译文，不是仅供 validator 在输出后检查的材料；不得因用户 Prompt 未重复提醒而省略，也不得用聊天上下文中的旧规则代替仓库当前内容。

正式 batch 还必须由 `retranslation export` 在创建前确认 current Glossary Review coverage。该前置 gate 只证明 glossary 本身已独立审核，不改变上述三部分输入契约，也不证明本 batch candidate 已通过 automatic validation 或 TranslationUnit Quality Check。

曾有翻译实验漏读 glossary，导致全部候选都产生 forbidden 译法；因此输入完整性本身是正式执行契约的一部分。

### Deterministic Generation Bundle transport

首次大批量 TranslationUnit Generation 必须由维护者 Local terminal 把上述不可拆分输入导出为 provider-neutral ZIP 并 handoff 给 Generation session；不得以逐文件传输或模型重新扫描工作树代替。后续小批 revision / retry 继续使用同一 bundle contract，但默认由具备仓库终端能力的当前 AI execution environment 完成机械导出：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
  --locale <locale> --batch-id <batch-id> \
  --output /tmp/<locale>-<batch-id>-generation.zip
```

retry 仅对当前 `restore_failed` / `validation_failed` Unit 增加 `--unit-id <unit-id>`；bundle 自动绑定下一连续 attempt、当前 validation/result 和完整原 batch 输入。`manifest.json` 声明 schema、task kind、locale、batch、Unit kind、attempt、exact expected outputs、全部成员 inventory/hash 与聚合 input identity。ZIP 内仍包含原始 batch manifest、manifest 列出的全部 inputs、完整 glossary 和当前 authority；ZIP 不替代这些 semantic authority。ChatGPT 与 Codex 都读取同一 contract，不因 provider 改变内容边界。

将完整生成结果先放入按 locale + batch 隔离的本地隐藏 staging。具备同一本地文件访问能力时，initial、revision 与 retry 默认直接目录 import，不制作临时 Result ZIP：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle import \
  --bundle /tmp/<locale>-<batch-id>-generation.zip \
  --input-dir /tmp/.go-tour-i18n-generation/<locale>/<batch-id> \
  --provider <chatgpt|codex> --model <provider-specific-formal-model>
```

直接导入会在本地完整读取并冻结 exact expected output set；正式安装使用的就是已经验证过的字节。跨环境传输、RDC 不可用或目录导入不适用时，保留 outputs ZIP → `result-pack` → 正式 Result ZIP → `import --result` 回退。旧 Result ZIP 仍兼容：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle result-pack \
  --bundle /tmp/<locale>-<batch-id>-generation.zip \
  --provider <chatgpt|codex> --model <provider-specific-formal-model> \
  --input-dir <exact-output-directory> \
  --output /tmp/<locale>-<batch-id>-result.zip

go run -mod=readonly ./cmd/tour-i18n generation-bundle import \
  --bundle /tmp/<locale>-<batch-id>-generation.zip \
  --result /tmp/<locale>-<batch-id>-result.zip
```

目录 import 与 Result ZIP import 共用 current canonical bundle、原 ZIP 完整性与全部 authority/input identity、locale/batch/task/attempt、expected outputs、真实 provider/model、exact file set、路径、UTF-8/no-BOM/single-LF、protected restore、glossary 与 machine candidate 检查。目录 symlink、额外目录/FIFO/非普通文件、缺失或额外文件、stale 输入、身份不符及已有正式输出均 fail closed。initial/revision 完整 staging 后原子安装且 no-replace；当前整批目录安装要求 Linux amd64 内核支持 `renameat2(RENAME_NOREPLACE)`，其他平台 fail closed。retry 保持单文件 no-overwrite。目录 import 将既有 `GenerationResultBundleManifest` 持久保存在 `data/retranslation-runs/<locale>/<batch-id>/generation-imports/initial-manifest.json`，retry manifest 按 unit 与 attempt 命名；记录 provider/model、Generation Bundle SHA-256、input identity、attempt 与输出 hashes；检查过程中若状态不确定，pending marker 会阻止 import 重放及 `process` / `retry`，必须先核实实际文件。目录与 Result ZIP 两种导入都只完成机器安全检查，不判断语言质量；随后仍由 `retranslation process` / `retry` 生成正式 automatic validation evidence，并继续独立 A-only QC。

## 5. 输出契约

每个 workflow unit 必须有一个独立 raw response 文件。raw response 必须：

- 只包含该 unit 的完整翻译结果；
- 不添加 Markdown code fence；
- 不添加解释、分析、前言或后记；
- 不输出 JSON；
- 原样且唯一地保留所有已有保护 token；
- 不自行构造保护 token 所代表的代码、directive、链接或其他结构；
- 以恰好一个 LF 结束，EOF 不得有额外空行。

Page 的 raw response 使用与 input 对应的 `.article` 文件；Example 的 raw response 使用与 input 对应的 `.txt` 文件。

### Protected token 内容保持规则

翻译过程中，protected token 是不可展开内容。即使模型可以根据上下文推断 token 对应的内容，也禁止：

- 恢复真实文本；
- 重写真实文本；
- 补充代码示例；
- 添加原文不存在的技术说明。

token 前后的自然语言可以正常翻译，但 token 本身必须原样且唯一地保留。翻译目标是保持原始 present 结构，不是根据教程上下文重新编写页面。

禁止新增原文不存在的代码块、preformatted section、`.play` directive 或示例说明。允许翻译既有安全 Go preformatted 中已开放的 teaching comment body，不等于允许新增、删除或重排 preformatted 内容。

正确：

```text
⟪GTI18N_xxx⟫ はエクスポートされた名前です。
```

错误：根据上下文补充 `package math` 的 `Pi` 定数说明，或新增：

```text
var i int
j := i
```

这些错误示例将 token 展开或补充了原文不存在的内容，不能作为 raw response 交付。

`basics/14` 曾出现类型推论失败：模型根据上下文补充了原文不存在的代码块，导致 `preformatted block section mismatch`。该案例说明，合理的教程上下文推断不能成为新增受保护内容或页面结构的理由。

### Raw response 文件格式

raw response 是 retranslation process 的直接输入。Page raw response 的路径为：

```text
data/retranslation-runs/<locale>/<batch-id>/raw-responses/*.article
```

该文件必须是纯 present article 文本。允许包含：

- present article 原始结构；
- 翻译后的自然语言；
- present directive；
- 链接结构；
- 代码和技术标识。

禁止包含：

- JSON wrapper；
- GitHub API 返回对象；
- Markdown code fence；
- 文件说明文字；
- 额外解释内容。

正确示例：

```text
* Hello, 世界

Go 言語ツアーへようこそ。
```

错误示例：

```json
{
  "content": "* Hello, 世界...",
  "encoding": "utf-8"
}
```

## 6. Locale 资源

`locales/<locale>/glossary.yaml` 定义该语言的 `mandatory`、`preferred`、`forbidden` 和 `keep` 规则。翻译任务必须在生成译文前完整读取并遵守目标 locale 的 glossary，不得借用其他 locale 的自然语言译法。

公共 UI catalog 属于独立的本地化 workflow，不属于 Page 或 Example 翻译任务，也不应写入本规范定义的 raw response。

## 7. 与 validator、review、promotion 的关系

本文件只定义翻译任务的输入、输出和结构边界。

raw response 后续如何 restore、由 validator 如何校验、如何生成完整 locale Candidate Snapshot、如何进行 Translation Quality Review，以及何时 promotion 为 canonical candidate 和 `ready` 状态，分别由相关 workflow 与规范负责。

### Generation 执行者主动诊断 validator

每个长期 Generation 执行者都对“自然译文与机器规则是否正确相容”承担主动诊断责任。目标语言自然句法与 validator 冲突，或执行者怀疑 validator 漏检时，不得只按报错机械改译文，也不得只凭聊天上下文宣称规则有误；应核对完整 source、完整 candidate、全部保护 token 及其 identity、restore/validation evidence，并在需要时使用仓库锁定的真实 Go Present parser/render 行为或 Go 源码语义验证判断。

诊断必须明确区分：

- 真实的翻译语义、完整性或自然度缺陷；
- 非法 Go 代码、present 结构或 Section topology 改变；
- 受保护内容、token identity、链接绑定、preformatted 或机器语义破坏；
- validator 的确定性误报或漏报；
- Generation 执行者自行增加、但正式规则并不要求的过度预检。

Page 与 Example 的 token 顺序边界不同：

- 普通 Page 可按目标语言自然语序整体移动完整 inline-code pair、完整链接及对应自然语言；完整链接移动时必须保持 label-target 绑定。不得缺失、重复或交叉 token，不得拆开 opener/closer、重新绑定链接、非法移动 directive、破坏 preformatted 结构或改变 Section topology。Page 不采用 Example 的全局固定 token 顺序约束，也不得由 Generation 执行者自行增加该约束。
- Example 继续保持严格 token 顺序、完整 Go 文件结构和所有非注释机器语义。只有正式识别为可翻译 teaching comment 的自然语言可以改变；Go 代码、标识符、字符串、comment delimiter、机器语义注释及其顺序不得改变。

规则修正应优先采用确定性、通用且最小的方案。只有证据证明某种目标语言特有的语法机制无法由通用方案正确处理时，才可提出 narrowly scoped locale-specific 规则；提案必须明确 locale、允许的精确翻译区域和结构、禁止的邻接案例及影响范围，不能为单个失败样本整体放宽共享 validator。

每项 validator 改进提案必须保存：

1. 完整 source 与完整 candidate；
2. 真实 restore/validation 失败或漏检 evidence；
3. 预期的 Go Present 解析/渲染行为或 Go 源码语义；
4. 至少一个确实应通过的 positive fixture；
5. 足以保护现有边界的 negative fixtures，包括适用的 token 缺失/重复/交叉、link rebind、directive 错位、preformatted/Section topology 破坏、代码或 identifier 改写。

不得为绕过疑似误报而硬改自然译文、插入不自然空格、删除自然语言、伪造 retry、擅自重译已审核通过的成果，或降低 A-only Quality Check、Glossary Review、Locale Surface Review 等正式门槛。Generation 执行者可以完成诊断并保存具体的最小修改方案；生产 validator 或共享校验规则的代码修改必须先取得维护者明确批准，再交由获授权的仓库级 Codex 改造任务实施。

只有已确定属于规则误报、restore 已成功、原 candidate 本来有效，且规则修复已获批并正式实施时，才按 [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md) 使用 `retranslation revalidate`。真正的译文质量缺陷仍走 revision、automatic validation 与独立 re-QC；不得改写旧 validation、candidate、review evidence、provenance 或 finalization。历史 pa-IN `basics/5` attempt 1 已为 `passed`，只说明当时的 directive placement 争议应以完整 evidence 和真实 parser 行为诊断，不授权改写该历史或预设现行 validator 存在 bug。
