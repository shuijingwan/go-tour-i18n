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

manifest 是任务身份的权威来源，记录 locale、batch、Translation Unit、source 身份、input 路径与保护 token 数量。对于 revision batch，每个 Unit 还记录 previous Quality Check Snapshot、rating 和 finding provenance；它提供修订反馈，但不属于 promotion evidence，也不替代 Final Review。模型必须只处理 manifest 列出的 unit，并将输出写入与该 input 对应的文件。

`manifest.json`、manifest 列出的全部 `inputs/*` 与 `locales/<locale>/glossary.yaml` 是不可拆分的正式输入。任何一部分缺失，都不属于合规的正式 TranslationUnit 翻译执行。Glossary 必须在模型开始翻译前读取并用于生成译文，不是仅供 validator 在输出后检查的材料；不得因用户 Prompt 未重复提醒而省略，也不得用聊天上下文中的旧规则代替仓库当前内容。

曾有翻译实验漏读 glossary，导致全部候选都产生 forbidden 译法；因此输入完整性本身是正式执行契约的一部分。

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
