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

模型应翻译 Page 中开放的自然语言，并保持完整的 `present.Section`。模型必须保留链接 target、present directive、行内代码、预格式化内容和其他受保护结构；不得删除、伪造、复制或改写保护 token 所代表的内容。

标题行仍须保留 `* ` 这一 present 语法前缀。模型可以按目标语言的自然语序调整普通文本，但不得改变页面结构或技术含义。

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

执行一次翻译任务时，模型应读取：

1. `data/retranslation-runs/<locale>/<batch-id>/manifest.json`；
2. manifest 列出的 `inputs/*` 文件；
3. `locales/<locale>/glossary.yaml`。

manifest 是任务身份的权威来源，记录 locale、batch、Translation Unit、source 身份、input 路径与保护 token 数量。模型必须只处理 manifest 列出的 unit，并将输出写入与该 input 对应的文件。

## 5. 输出契约

每个 workflow unit 必须有一个独立 raw response 文件。raw response 必须：

- 只包含该 unit 的完整翻译结果；
- 不添加 Markdown code fence；
- 不添加解释、分析、前言或后记；
- 不输出 JSON；
- 原样且唯一地保留所有已有保护 token；
- 不自行构造保护 token 所代表的代码、directive、链接或其他结构。

Page 的 raw response 使用与 input 对应的 `.article` 文件；Example 的 raw response 使用与 input 对应的 `.txt` 文件。

## 6. Locale 资源

`locales/<locale>/glossary.yaml` 定义该语言的翻译规则，包括强制译法、优先译法、禁止译法和必须保持原样的内容。翻译任务必须使用目标 locale 的 glossary，不得借用其他 locale 的自然语言译法。

公共 UI catalog 属于独立的本地化 workflow，不属于 Page 或 Example 翻译任务，也不应写入本规范定义的 raw response。

## 7. 与 validator、review、promotion 的关系

本文件只定义翻译任务的输入、输出和结构边界。

raw response 后续如何 restore、由 validator 如何校验、如何进行 Translation Quality Review，以及何时 promotion 为 canonical candidate 和 `ready` 状态，分别由相关 workflow 与规范负责。
