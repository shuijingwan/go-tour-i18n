# A Tour of Go 多语言翻译术语规范

## 1. 文档目的与适用范围

本文档定义 `go-tour-i18n` 的术语治理政策，适用于当前和未来的所有 locale。它不是一份完整的 Go 词典，也不表示文中每项政策都已经由当前代码自动强制执行。

术语治理分为两个层次：

1. 全 locale 通用的技术身份保护政策；
2. locale 专用的自然语言翻译政策。

本项目遵循两个核心原则：

> 代码身份必须保持原样，不等于教学概念必须保持英文。

> 结构上不能修改，与语言上不应该翻译，是两个不同问题。

前者通常由 protector 和结构校验保证，后者通常由 locale glossary、翻译提示、术语审核和必要的 validator 规则保证。

## 2. 术语规则类别

### 2.1 `mandatory`

`mandatory` 表示指定 locale 中必须采用的正式译法。例如 zh-CN 中：

- `type assertion` → “类型断言”；
- `type parameter` → “类型参数”；
- `constraint` → “约束”。

政策上的 mandatory 与当前 validator 是否能在全文逐处强制该译法不是同一个概念。规则被列为 mandatory，表示模型输出和人工审核必须遵循；只有实现了相应检查的场景，才能声称它已由 validator 自动保证。不得因为某项规则位于 `mandatory`，就推断当前 validator 已经检查普通正文中的每一次出现。

### 2.2 `preferred`

`preferred` 表示项目已有明确偏好的推荐译法。例如 zh-CN 中：

- `channel` → “通道”；
- `package` → “包”；
- `type inference` → “类型推断”。

即使当前 validator 尚未逐处强制，翻译模型和人工质量审核也应遵循 preferred。存在多个成熟译法时，preferred 可以表达项目选择，而不必宣布其他社区译法在所有语境下都是错误的。

### 2.3 `keep` / do-not-translate

`keep` 正式表示 do-not-translate：对象以其对应技术身份出现在普通、可翻译的自然语言区域中时，仍必须保持规范原始形式，不得本地化。

keep 不是代码保护列表。一个对象只有同时满足以下条件，才应考虑进入 keep：

1. 它会出现在可翻译的自然语言区域；
2. 它在该技术语境中仍不能翻译；
3. protector 不能单独、完整地解决该问题。

只存在于程序字体、预格式化代码、路径或其他严格结构中的对象，通常应由 protector 负责，而不应为了“保险”再复制进 keep。

### 2.4 `forbidden`

`forbidden` 表示候选译文中禁止出现的已知错误译法或错误表达。

forbidden 不表示对应英文必须保留。例如项目可以禁止用“幻灯片”翻译 `slide`，同时要求将其译为“页面”；这不意味着 `slide` 必须保持英文。

forbidden 应用于明确、稳定的错误形式。不要把存在合理语境的普通中文词全局禁止。

## 3. Protector 与 glossary 的职责边界

### 3.1 Protector 负责的对象

Protector 和结构校验负责结构上必须保持身份或字节稳定的对象，包括：

- present directive；
- `.play` 和 `.image` directive；
- 文件路径和资源路径；
- URL、域名和 link target；
- import path；
- program span 和 inline code；
- preformatted 中的 Go 代码、结构、标识符和不可翻译注释（不包括 protector 已安全开放的 teaching comment 自然语言 body）；
- API identifier；
- package identifier；
- 类型名、函数名、方法名和字段名；
- Go keyword 作为代码身份时的原始形式；
- 命令名称和其他技术标识符；
- 其他不能删除、复制、重绑定或改写的结构对象。

### 3.2 Glossary 负责的对象

Glossary 和术语政策负责：

- 普通教学正文中的术语选择；
- locale 中的官方名称显示政策；
- 位于自然语言中仍不得翻译的名称；
- mandatory 和 preferred 译法；
- forbidden 错误译法。

Locale glossary 不只服务于 TranslationUnit 模型输入。它也是该 locale 的正式术语权威来源；公共 UI、首页、`/tour/`、`/tour/list`、导航、语言选择器、编辑器动作、runtime message、article metadata 与 SEO 可见文案都必须遵守同一份 `mandatory`、`preferred`、`forbidden` 和 `keep` 决策。自动 validator 未覆盖这些表层时，由 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 执行发布前人工审核。

Glossary 不能替代 source ↔ target 语言质量比较。术语全部合规的 target 仍可能存在误译、漏译、不自然表达、英文残留或无依据扩写；这些问题必须结合对应英文/source identity 审核。

`Go Playground` 一类在不同语言中可能翻译、部分本地化或保持原文的名称，必须由每个 locale 显式决定并写入其 glossary。不同 locale 不要求字面处理相同；同一 locale 必须全站一致。新增这项治理规则不意味着自动改写 zh-CN 或 ja-JP 的现有译文，既有语言应在独立术语维护任务中评估具体变更。

不能为了保险，把已经由 protector 保护的 Go keyword、builtin、API、路径和包名再次复制成巨大的 keep 表。这样既会造成双份维护，也会把代码身份与自然语言概念混为一谈。

## 4. 已冻结的专门术语政策

### 4.1 `goroutine`

`goroutine` 是 Go 原生专有名称，也是 do-not-translate 的典型案例。

zh-CN 正式显示规则如下：

- 使用：`goroutine`；
- 不将“Go 程”或“协程”设为正式译名；
- 首次教学性出现时，可以写成“goroutine（由 Go 运行时管理的轻量级线程）”或语义等效的解释；
- 后续继续使用 `goroutine`。

该规则不同于 `channel` → “通道”、`mutex` → “互斥锁”、`package` → “包”等教学概念的中文化规则。

### 4.2 `map`

`map` 不属于 do-not-translate。必须区分它的代码身份和教学概念身份。

作为 Go keyword 或代码结构时保持 `map`，例如：

```go
map[string]int
```

在普通自然语言中表示教学概念时，zh-CN 使用“映射”。例如 `Maps are like slices...` 应译为“映射与切片类似……”。不能因为 Go keyword 必须保持 `map`，就让自然语言正文长期中英混用。

### 4.3 `channel` 与 `pipeline`

zh-CN 采用以下术语边界：

- `channel` → “通道”；
- `pipeline` → “管道”。

不要使用“管道”作为 `channel` 的正式术语。英文 `conduit` 用于解释 channel 时，可以根据上下文译为“通路”“传输通路”“传递值的机制”等自然中文，避免形成 `channel = pipeline` 的概念混淆。

“管道”不能被设为全局 forbidden，因为它是 `pipeline` 的正确译法。若需要禁止 channel 的错误定义，应使用足够精确、不会误伤其他概念的规则。

## 5. 不应该加入 keep 的典型教学术语

下列对象虽然是 Go 或编程专业术语，但属于可翻译的教学自然语言概念，不应仅因其专业性而保留英文：

| 英文 | zh-CN 教学译法 |
| --- | --- |
| `package` | 包 |
| `interface` | 接口 |
| `interface value` | 接口值 |
| `interface type` | 接口类型 |
| `concrete type` | 具体类型 |
| `constraint` | 约束 |
| `type parameter` | 类型参数 |
| `channel` | 通道 |
| `mutex` | 互斥锁 |
| `pointer` | 指针 |
| `slice` | 切片 |
| `map` | 映射 |
| `receiver` | 接收者 |
| `method` | 方法 |
| `function` | 函数 |
| `loop` | 循环 |
| `iteration` | 迭代 |
| `type inference` | 类型推断 |
| `constant` | 常量 |
| `built-in` | 内置 |
| `standard library` | 标准库 |

这些概念在代码区域中的 keyword、标识符或语法形式仍应由 protector 保持原样。例如自然语言中的 `package` 译为“包”，代码中的 `package` token 不变；自然语言中的 `mutex` 译为“互斥锁”，标识符 `sync.Mutex` 不变。

本表仅用于说明分类原则，不取代 locale glossary，也不是完整术语表。

## 6. 官方品牌和技术固定名称

官方品牌核心名称、工具名、环境变量名和规范技术缩写原则上保持其标准形式。例如：

- `Go` 是官方品牌核心名称，通常保持 `Go`；
- `gofmt` 是工具名，保持原样；
- `GOPATH` 是规范环境变量名称，保持原样；
- `URL`、`ASCII`、`CPU`、`UTC` 等规范缩写原则上保持标准大写形式。

这不表示上述所有对象都必须机械加入 glossary keep。是否加入仍需逐项判断：

1. 是否实际出现在普通可翻译正文；
2. protector 是否已经覆盖；
3. 是否确实需要模型级约束；
4. 是否存在普通英语或其他技术身份导致的歧义。

本政策不声称这些名称已经由当前 YAML keep 自动实现。

## 7. Go keyword 与 builtin

Go keyword 和 builtin 应根据技术身份判断，不建立机械的字符串级“永远不翻译”规则。

- `select` 作为 Go keyword 或语句名称时保持 `select`，例如“`select` 语句”；普通英语 `select an item` 应正常翻译。
- `make` 作为 Go builtin 时保持 `make`；普通英语 `make it possible` 应正常翻译。

同理，代码中的 keyword、builtin 和预声明标识符应保持规范形式；普通自然语言中与其同形的英语单词仍按句意翻译。项目不维护完整 keyword 或 builtin keep 字典。

## 8. 全 locale 通用规则与 locale 专用规则

### 8.1 全 locale 通用规则

以下规则属于跨语言技术身份保护政策，主要由 protector、结构校验和通用翻译政策承担：

- URL target、域名和 link target 不改；
- 文件路径和资源路径不改；
- import path 不改；
- API identifier 和 package identifier 不改；
- 类型名、函数名、方法名和字段名不改；
- Go keyword 作为代码身份时不改；
- 命令名称不改；
- 技术标识符的大小写不改；
- directive 及其结构身份不改。

`goroutine` 体现 Go 原生专有名称的跨 locale 保留原则。不同 locale 可以决定首次出现时是否附加本地语言解释，但不应把本地解释变成替代规范名称的正式译名。

### 8.2 Locale 专用规则

Locale glossary 决定目标语言中的自然语言译法。例如 zh-CN 包括：

- `channel` → “通道”；
- `package` → “包”；
- `constraint` → “约束”；
- `type assertion` → “类型断言”；
- `type switch` → “类型选择”；
- `map` → “映射”。

`goroutine` 首次出现时附加何种中文解释，也属于 zh-CN 显示策略。

## 9. 政策状态与实现状态

本文档定义目标术语政策，但当前仓库尚未让所有政策在代码中自动生效。必须区分以下四个层次：

- **Policy**：本文档规定项目希望采用的长期术语行为；
- **Prompt**：翻译请求提供给模型的规则和建议；
- **Protector**：在翻译前后确定性保护结构和技术身份；
- **Validator**：对最终候选执行的自动检查。

某项政策存在，不表示 prompt、protector 和 validator 已经全部实现它。反之，某个字符串当前被 protector 硬编码保护，也不表示它已经形成完整、可维护的 glossary 政策。

### 9.1 当前 glossary 实现

当前 `locales/zh-CN/glossary.yaml` 声明了：

- `mandatory`；
- `preferred`；
- `terms`；
- `forbidden`；
- `keep`。

但当前 loader 实际只解析：

- `mandatory`；
- `preferred`；
- `forbidden`。

`terms` 当前未加载，YAML 中的 `keep` 当前也未加载。当前实际 keep 来源是翻译保护代码中的硬编码规则，而不是 YAML keep。因此不得声称修改 YAML keep 就会改变当前运行时行为，也不得声称本文所有政策已经由代码自动保证。

### 9.2 已知实现缺口

后续独立实现阶段需要：

1. 决定 `terms` 是否保留独立语义，或并入 `preferred` / `mandatory`；
2. 决定 YAML keep 是否成为唯一配置源；
3. 消除 YAML keep 与翻译保护代码硬编码的双份维护；
4. 根据正式政策决定 validator 是否需要增加必要且可可靠执行的检查。

本文档不决定具体代码修复方式。

## 10. 新增术语的判断流程

遇到新术语时，按以下顺序判断：

1. **它是结构对象或代码身份吗？**  如果是，优先由 protector 处理，不进入普通 glossary。
2. **它是普通教学概念吗？**  如果是，优先确定 locale 的 mandatory 或 preferred 译法。
3. **它虽然位于自然语言中，但仍必须保留规范原文吗？**  如果是，再考虑 keep / do-not-translate。
4. **某个错误译法需要明确禁止吗？**  如果是，评估加入 forbidden，并确保规则不会误伤合理语境。
5. **它是否只是当前模型偶然不一致？**  如果只是单次波动，不要立即新增规则；先判断是否属于稳定、长期的术语政策。

## 11. 保持简单的架构方向

当前不引入：

- 完整 Go keyword keep 字典；
- 完整 builtin keep 字典；
- 完整标准库 API keep 字典；
- `contextual_keep` NLP 引擎；
- 复杂术语数据库；
- Web 审校平台。

项目继续采用“小而稳定的术语规则 + 结构保护 + locale glossary + 自动 validator”的方向。只有出现明确、可重复、适合自动判断的问题时，才增加相应规则或校验。

## 12. 术语政策参考

术语选择应综合参考：

- A Tour of Go 固定官方 upstream；
- 目标语言技术社区的常用译法；
- 项目已有术语实践和实际候选；
- [为什么 channel 最终选择“通道”：A Tour of Go 中文术语统一的一次实践](https://www.shuijingwanwq.com/2026/08/15/25521/)。

社区存在多个成熟译法时，项目可以选择统一标准，但不应仅根据当前候选中的出现次数判断，也不应把项目选择描述为整个社区唯一正确的答案。
