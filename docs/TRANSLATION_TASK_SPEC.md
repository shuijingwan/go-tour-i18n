# 翻译任务规范

Learn/Docs current Page scope为统一51 Page，initial Generation与初审各30+21两次独立 invocation。`docs-entry-v1`仅是历史 compatibility identity，不另开新 locale阶段；package composition不改变 TranslationUnit source/input或语言角色，详见[统一 membership](LEARN_DOCS_MEMBERSHIP.md)。

全项目执行共同遵守[共享翻译单元生命周期与五角色协作](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)：所有内容表面复用同一生命周期；页面差异限于适配器；传输安全与逐单元机器校验分层；先恢复、先复用，中文说明，交接只给一个执行者、动作和停止条件。本文中 Tour 的 present / Example 细节属于 Tour 适配边界，不另建 Learn/Docs 状态机。Learn/Docs 两批生成和机器校验全部完成后才初审，两批初审全部完成后集中修订。角色与工具分工不替代任何正式 machine authority 或语言质量 gate。

本规范定义全项目正式翻译单元的输入、输出和结构约束；以下 present / `.go` 细节是 Tour 适配规则。它不定义命令操作或具体术语译法。

## 1. Translation Unit

翻译任务以一个完整 workflow Translation Unit 为最小单位。Translation Unit 有两种：

- **Page**：Tour 是一个完整顶层 `present.Section`；Learn/Docs 是一个完整物理 Markdown/HTML 页面。后者内部槽位不是独立翻译单元。
- **Example**：一个完整 `.go` 文件。

不得把一个 workflow unit 拆分为句子、段落或多个独立模型输出。一个 unit 对应一份输入、一次原始模型输出和一份恢复后的 candidate。

Markdown inline HTML 的 quoted `alt`、`title`、`aria-label` 与 HTML surface 遵守同一可编辑自然语言属性合同；模型可以翻译其中开放的 prose，仍输出 protected raw response。tag、属性名、machine 属性值、引号边界、属性顺序、空白及 void/self-closing syntax 保持 protected；不可展开 token 或在 restored candidate 上手工补属性，matching quote / attribute injection 继续拒绝。opaque `code/pre/script/style` 不因此开放。

## 2. Tour Page 适配规则

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

manifest 是任务身份的权威来源，记录 locale、batch、Translation Unit、source 身份、input 路径与保护 token 数量。新流程的 revision feedback 只来自 Quality Check B/C/D 的 rating 和 finding；它不属于 promotion evidence，也不替代新 candidate 的 QC/finalization。历史 manifest 中的 Final Review feedback 字段保持可读，不被重写。Glossary-induced protected-input stale recovery 不伪造 revision feedback；它以独立 `reexport_reason=glossary_input_stale` 和 previous batch/input identity 说明当前 protected input 为何需要重新生成。模型必须读取真实 provenance/feedback，只处理 manifest 列出的 unit，并将输出写入对应文件。

`manifest.json`、manifest 列出的全部 `inputs/*` 与 `locales/<locale>/glossary.yaml` 是不可拆分的正式输入。任何一部分缺失，都不属于合规的正式 TranslationUnit 翻译执行。Glossary 必须在模型开始翻译前读取并用于生成译文，不是仅供 validator 在输出后检查的材料；不得因用户 Prompt 未重复提醒而省略，也不得用聊天上下文中的旧规则代替仓库当前内容。

正式 batch 还必须由 `retranslation export` 在创建前确认 current Glossary Review coverage。该前置 gate 只证明 glossary 本身已独立审核，不改变上述三部分输入契约，也不证明本 batch candidate 已通过 automatic validation 或 TranslationUnit Quality Check。

曾有翻译实验漏读 glossary，导致全部候选都产生 forbidden 译法；因此输入完整性本身是正式执行契约的一部分。

### 精确修订任务语义

Quality Check / re-QC / Surface finding 授权的 exact revision 是 **finding-anchored + locally complete**。finding-anchored 指依据 persisted finding 的 **defect category / predicate + bounded scope** 修订；locally complete 指在同一 current Unit、同一 category、同一授权 scope 内完成 **same-class complete sweep**，修复所有明确匹配的实例，不只修 Reviewer 明列或最显眼的几项。多个独立 category 分别处置；实际 candidate defect 修复，source/protected 原貌、合法结构或 Reviewer 误识别有依据地保持。

**finding-anchored 不等于 enumerated-example-only。** Reviewer examples 是 evidence，不默认构成 exhaustive revision boundary；只有 finding 明确写为 **exact-instance-only** 时，才只授权列出的具体实例。Generation 先识别正式 category/predicate 与 bounded scope，再修复明列实例，并扫描该 scope 内其他明确同类实例。此类未逐项点名的 same-class fixes 仍是 finding-anchored revision，不是 scope expansion。finding 写法与 Reviewer completeness 见[质量审核合同](TRANSLATION_QUALITY_REVIEW.md#bcd-finding-完整性与-successor-re-qc)。

“local”由 finding 的语义与授权边界决定，不仅指 examples 周围的几句话。scope 明确覆盖当前 Page 全文时，必须扫描该 Page 全文中的同类问题；scope 仅限某 section、表格列、comment 区域或 exact instances 时，不得自行扩大。修复还须核对所在句子及直接受影响的紧密上下文，避免留下同类缺陷或引入新的局部不完整。这不授权跨 Unit、改其他问题类别、一般措辞偏好、无关标点/style、普通润色、已正确内容或整页重译。category/predicate、scope 或某实例是否同类无法可靠确定时，不猜、不扩大范围，保持不确定处原文并报告 ambiguity / blocker，不能把未关闭问题视为已完成。

same-class sweep 不改变正式 Unit/slot authorization 或 protected representation：canonical source、code/identifier/link/API identity、directive、protected literal/token、machine semantics、required markup 和结构保护全部保持；tokens 仍须 exact and unique，允许的完整结构移动仍遵守既有绑定。即使某处符合 predicate，也不能打开未授权 teaching-comment slot、opaque span 或自行重建 token mapping；缺合法 editable location 时 fail closed，报告 representation blocker，沿用既有 bridge/replacement/retry authority。

每个 revised Unit 提交前必须完成 finding closure check，重新对照 source、revised candidate、persisted finding 与 protected/structure requirements，确认逐 category 处置、授权 scope 内 same-class closure 和直接受影响的局部完整性；具体执行与例外见[Generation finding closure 执行规范](CHATGPT_LANGUAGE_GENERATION.md#generation-finding-closure-执行规范)。这是 Generation 的任务完成性责任，不是正式 QC 或通过决定；输出仍是完整 Unit，不新增审核 evidence、状态、validator 规则或 Bundle/invocation size 上限。历史 finding/evidence 不回写、不迁移。

initial working-set membership 仅是调度元数据，不是永久 TU identity / revision partition。Learn/Docs initial Generation / initial QC 保留固定组和每组30限制；两批初审完整结束后的 B/C/D revision 与正式 Surface reopen revision 可跨 initial group，仍须同 kind、最多60、current predecessor/finding/identity authorization 和全部安全 gate。successor re-QC 按 current `quality_check_required` pending、stable index 顺序、同 kind、最多60选择并 JIT；record 使用相同阶段语义。glossary-stale recovery 保留既有固定分组及 protected-input/currentness gate，retry/recovery 不变。

### Deterministic Generation Bundle transport

缺失历史 Generation archive 的 narrowly scoped provenance 已有独立的 [`generation-bundle legacy-provenance` 正式 authority](LEGACY_GENERATION_PROVENANCE_AUTHORITY_DESIGN.md#正式-implementation-command-contract)；zh-CN json 的 exact authority 已完成 first-real-use 与 write-once record，其 path/SHA/identity 见 [approved-interval contract v2](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md#已记录-authority-与当前-case)。它只证明指定 per-Unit historical response 的 surviving provenance，不恢复原 Bundle、不证明旧 Bundle CURRENT、不授权原 invocation replay/import 或语言质量。approved-interval consumer 已接入该 authority；普通无 selection 的 consumers 继续执行既有 archive/codec gate。优先级固定为 `NORMAL_ARCHIVE > EXPLICIT_RECORDED_LEGACY_PROVENANCE`：archive 缺失才可显式绑定 path/SHA 并调用 `VerifyLegacyGenerationProvenance`、传 exact requested options，不扫描 latest/按 Unit 自动发现。只消费 `historical-response/v1` allowlist，不补齐六项 unknown facts；archive 重现必须比 covered facts，冲突停止、匹配走 normal。

Surface revision 还必须满足独立的 [revision transport capability](SURFACE_REVISION_TRANSPORT.md) 合同；CURRENT inputs 与可表达 persisted finding 是两个不同 gate。没有 exact recorded capability authority 的新 Surface Bundle 在 handoff 前 fail closed，不能仅凭 historical raw restore PASS 或过去 CURRENT 推断可编辑。

首次大批量 TranslationUnit Generation 必须由维护者 Local terminal 把上述不可拆分输入导出为 provider-neutral ZIP 并 handoff 给 Generation session；不得以逐文件传输或模型重新扫描工作树代替。后续小批 revision / retry 继续使用同一 bundle contract，机械导出同样默认由维护者 Local terminal 执行；AI 按[Call-Zero-First](WORKFLOW_HANDOFF.md#call-zero-first-交接合同)提供完整命令：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
  --locale <locale> --batch-id <batch-id> \
  --output /tmp/<locale>-<batch-id>-generation.zip
```

`--batch-id` 是已存在正式 batch 的 identity，不是 manifest 文件路径；`--output` 是新的 ZIP transport 路径，支持绝对路径或相对当前工作目录的路径。export 重新验证完整 current authority 后只发布该 ZIP，不改 batch、Snapshot、reopen 或语言 evidence；同一输入重建相同 bytes，但已存在的 output 路径一律拒绝覆盖。准备失败不发布 transport，发布时以新路径不可覆盖为边界；失败后保留原 batch/evidence，不自动 recovery 或创建新 batch。

TranslationUnit Generation Bundle 的 canonical ZIP 使用 **Deflate、显式 level 5**；固定 member 顺序、1980-01-01 UTC 时间与0644 mode，在同一 Go compressor 实现下 exact inputs 重建相同 bytes，不承诺不同 Go/compressor 版本输出相同压缩字节。只对此 family opt-in；Result、Reviewer、locale、retry handoff 等其它 family 的 canonical writer 保持 Store。reader 兼容历史 Store 与新 Deflate，仍核验 CRC、path、exact inventory/hash；上限为8192个成员（含 manifest）与134217728 bytes **总解包 payload**，不按压缩后的大小放宽。member names/逻辑 bytes/semantic authority graph 不因压缩变化；ZIP SHA 仍绑定真实物理 bytes。

**READABLE 不等于 CURRENT**：历史 Store archive 继续可读，current-check 仍只比较当前 canonical exporter 的完整 ZIP bytes，不以双格式 comparison 维持旧 Store 的 CURRENT。archive observation 对 Deflate bounded decode 后核验逻辑 hash，不把 compressed span 当作逻辑 bytes；Store 保留已验证原始 span 的复用。retry handoff 只按已记录 SHA 精确恢复原 Store 或 Deflate-5 constituent，不替换 invocation identity。旧 ZIP、import/model provenance、历史 replay evidence 不重打包或回写。本次 canonical encoding/reader/currentness 修改使旧 first-real-use verification 失效；真实使用前须重新 gate，工程实现不授权真实064 re-export 或 Generation。

retry 仅对当前 `restore_failed` / `validation_failed` Unit 增加 `--unit-id <unit-id>`；bundle 自动绑定下一连续 attempt、当前 validation/result 和完整原 batch 输入。`manifest.json` 声明 schema、task kind、locale、batch、Unit kind、attempt、exact expected outputs、全部成员 inventory/hash 与聚合 input identity。ZIP 内仍包含原始 batch manifest、manifest 列出的全部 inputs、完整 glossary 和当前 authority；ZIP 不替代这些 semantic authority。ChatGPT 与 Codex 都读取同一 contract，不因 provider 改变内容边界。

export 与 current-check 是独立 invocation，使用同一 complete package-selection contract。CLI 每次重建 source/catalog；显式 `--content-package tour-v1` 选择 canonical Tour-native default complete scope（全部 native Page、eligible Example、canonical ordering、无 foreign adapter），不需要也不创建 registration。Tour Catalog 必须没有 explicit `TranslationScope`；不得清空 partial、different-order 或 foreign scope 后 widening。其它显式 package 必须经既有 `UseTranslationPackage()` 选择 exact complete registered descriptor；未知 package 或已有不匹配 scope 拒绝。recorded current-validation revision 的正式 export/current-check 调用均须显式提供一次同一 `--content-package`，不能从 batch 的少数 Unit 推断 package 全集；current-validation reopen 与 revision export 复用相同 complete selector。canonical exporter 在读取 predecessor context 前独立证明完整 package scope，并要求 batch 每个 Unit 都属于该 scope；historical reader 不能借恢复 predecessor catalog 覆盖调用者显式选择的另一 package。

current-check 重新执行 canonical exporter，核对 Files inventory、聚合 identity 与完整 ZIP bytes，不复用上次 export 的 PASS。单次 revision graph 构建可以复用已经验证的同一 current-validation reopen 和 frozen QC evaluation，但返回前仍重新读取 receipt 并核验全部冻结历史引用。既有 invocation memoization 只复用 freshly supplied exact bytes 的确定性转换，不缓存 disk observation、currentness 或 authorization，不跨 invocation 保存结果。性能测试须分离两个 invocation 的 memo scope，不能用同一 memo scope 或旧 ZIP 的历史 CURRENT 代替当前检查；已安装输出的 batch 继续拒绝重新导出 initial/revision Bundle，benchmark 只能使用隔离副本。

`translation-unit-revision` 必须另外携带每个 selected Unit 的 **predecessor current candidate**：从 batch 授权的 `PreviousSnapshotID` 精确解析 Snapshot `CandidatePath` / `CandidateSHA256`，验证 Unit/kind/source、reviewed candidate、finding authorization、validation 与 attempt。不得选用另一批次的 latest candidate。完整候选冻结在 `formal/candidates/<input basename>`，对应 predecessor Snapshot manifest 在 `formal/predecessors/<snapshot-id>/manifest.json`；二者进入既有 Files inventory/hash 与 aggregate input identity。候选缺失、SHA 不符、授权不符或 Bundle 被改动，export/current-check 均 fail closed。候选是独立 formal input，不拼入 protected English input，也不属于 ExpectedOutputs；ExpectedOutputs 仍只包含 replacement。

revision Bundle 区分三层表示：`formal/inputs/<basename>` 是 protected English/source input；`formal/candidates/<basename>` 是 Reviewer 审核的 restored predecessor candidate，只用于 semantic/structure/finding closure 对照；`formal/protected-candidates/<basename>` 是实际编辑模板，即产生该 candidate 的正式 passed protected raw response。exporter 从 Snapshot 绑定的 validation `RawResponsePath` / `Attempt` 取得唯一原批次、原 attempt 的 raw bytes，用其正式 representation decoder restore，并按既有 process EOF framing 核对精确 Snapshot candidate；缺失、错误 attempt/token set 或还原不一致均 fail closed。protected baseline 与 predecessor validation 也进入 Files inventory/hash、aggregate identity 和 canonical current-check；不靠重新保护 candidate 猜测 token mapping。

`quality_check` 与 `surface_review` 的 finding/revision authority 分别验证，predecessor protected representation 则由其 immutable same-Unit lineage 决定。两者均可沿 exact predecessor Snapshot 所选 manifest/attempt 复用已有 same-unit bridge 的 historical decoder；必须 freshly 验证 bridge receipt、同 Unit/kind/source/input、每层 candidate/validation/raw、import receipt/archive 和全部 frozen references，且 raw 必须 exact restore reviewed candidate。`formal/representation-lineage/<unit-id>.json` 与 passed protected raw、candidate 一同进入 Files inventory/hash；current-check 独立重建同一 graph，返回前重验依赖。不复制旧 QC finding 为 Surface authority，不创建新 bridge，不开放额外 slot；没有合法 lineage 的 Surface revision 继续使用普通 protector，不能借他人 lineage 或从 candidate 反推 mapping。此规则不扩大 Preview、replacement 或 current-validation 的专属 transport contract。

`current_validation` 的 exact teaching-comment finding 有一个 revision-only transport 投影：exporter 在完成上述历史 raw → candidate 精确验证后，按 shared reopen receipt 的逐 Unit slot identities、source-selected parser 和 exact payload coordinates 派生 `formal/protected-candidates/<basename>`。其中仅授权的 teaching-comment body 和其既有 identifier bindings 可编辑；其它正文、代码、literal、directive、delimiter、protected comment 和未授权 teaching comment 均为 opaque boundary。原 passed raw 不变，另存于 `formal/historical-protected-candidates/<basename>`；`formal/comment-revision-transport/<basename>.json` 保存派生 mapping、predecessor/source/receipt identities 和 exact editable slots。它是 Bundle input，不是新的正式 TU、output schema 或 representation successor；正式 Unit ID/source/input/manifest 不变。

`current_validation` 不自动等于 teaching-comment finding。若 exact finding 没有授权 comment slots（例如普通 prose boundary），保持原 passed raw 为 `formal/protected-candidates`，不伪造 `formal/comment-revision-transport` 或额外 historical/projection split。上述 comment 专属文件只在其 authority 适用时存在；缺少授权不能用这些路径的存在伪装成可编辑能力。

Generation 只编辑投影中开放的 body，使用该投影的 token set，逐 token exact/unique，并保持各 binding 在原 slot 内；不从 restored Page 重建 token mapping。机器通过现有 body protector/restore 解码局部文本，再按 opaque boundaries 恢复完整 Page，最后执行 active canonical validator。删改/重复 token、跨 slot、修改未授权区域、missing/extra/changed slot authority 或不可证明的 raw encoding mapping 均 fail closed。projection/mapping 与其实现依赖进入 Bundle identity/current-check；普通 initial、QC B/C/D、Surface/Preview revision 不获得此例外。没有正式 outputs 时，修复 transport 后重导同一 batch，旧 ZIP 保留为历史事实。

Generation 从 protected predecessor baseline 做 finding-anchored、locally complete 修改，输出始终是 **protected raw response**，保留全部 tokens exactly and uniquely，再走既有 restore → candidate → validation。不得直接输出 restored candidate，不从 English source 整页重译。完整 frozen semantic inputs 同时包括三种表示、Snapshot、previous finding/revision authorization、完整 glossary、authority 与 identity/provenance context；完整 CURRENT Bundle 到手后只读取 Bundle，语言 semantic RDC calls 默认 0。

execution `task_kind=translation-unit-retry` 不改变原 batch 的 semantic task。由 manifest `RevisionFeedbackSource` / `PreviousSnapshotID` 等确定 revision origin；revision-origin retry 保留完整原 batch protected inputs、两种 predecessor candidate、Snapshot/validation、finding authorization、glossary/authority，并附当前失败 Unit 的 retry validation/result。context 指明 attempt N 与原 exact revision/finding closure 语义，ExpectedOutputs 仍仅为该失败 Unit。QC 和 Surface reopen 共用此机制；initial retry 与 glossary-stale retry 不套用不存在的 revision predecessor/finding 合同。

若旧 Bundle 缺项或 authority 更新，但原 revision batch 的 manifest/authorization 未变且没有正式 raw responses/import，可以为**同一 batch**重新执行 Generation Bundle export；无需新建 revision batch。current-check 按当前完整合同重建 canonical Bundle，使旧 ZIP 自然 stale/non-canonical，不手工补 ZIP、删除旧证据或覆盖正式输出。

已经installed的approved derived proof后来stale，不适用上述同批刷新。新的 Generation/recovery
invocation 仍按下述恢复合同；Candidate Snapshot 的 historical-PASS admission 与新 invocation
权限分开，以本文后述 Snapshot 专属合同为准。
[Installed derived-transport recovery设计v1](INSTALLED_DERIVED_TRANSPORT_STALE_RECOVERY_DESIGN.md)
规定新的窄authority、原batch完整decoder-context closure及全范围successor invocation；
要求真实新Language Generation，不把旧raw作为新output，不修改旧proof比较或同batch历史。
实现仍须独立first-real-use gate后才可真实使用；synthetic PASS不授权真实恢复、单Unit裁剪或新reopen。

历史 construction dependency 的 exact bytes 若仅存在于外部历史副本，仍不能直接作为
formal reader 输入。[Historical bytes formalization 合同](HISTORICAL_CONSTRUCTION_BYTES_FORMALIZATION_DESIGN.md)
另行设计 supplemental byte authority、content-addressed repository publication 与 acquisition
audit；尚未实现或授权采用，不修改旧 archive/receipt/proof，也不授予 Generation/QC/recovery 权限。

上述同批刷新不适用于 formal Page protected representation 已产生 replacement UnitID 的情况：旧 batch 不改写，须通过[正式 Page replacement lineage](TRANSLATION_UNIT_REPLACEMENT.md)建立显式 reconciliation successor reopen 与新 revision batch，再精确复用 unchanged 输出。

若 attempt 1 已 import/process 且失败，必须保留 raw responses、provenance、result 和 validation；仅对真实失败 Unit 导出下一连续 attempt 的单 Unit retry Bundle，Generation 输出 protected revision response 后按既有 retry import/process 验证。不得覆盖 attempt 1、重新 initial import、revalidate 绕过 restore 或另建 batch；成功 Unit 保留，仅继续 retry 真实失败 Unit。

如果一次 Generation invocation、消息流或本地 transport 在正式 import 前中断，新的模型重试不得直接从零重新生成整批。先核对正式 `raw-responses/` / retry attempt、generation import provenance、`result.json`、`validation/` 与后续 evidence；若正式状态尚未落地且原 Generation Bundle 仍 CURRENT，则使用 recovery preflight 从旧 hidden staging、临时单文件或 interrupted tar/tar.gz artifact 中恢复可独立验证的完整 Unit：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle recover \
  --bundle /tmp/<locale>-<batch-id>-generation.zip \
  --source <old-staging-or-interrupted-artifact> \
  --source <optional-second-source> \
  --output-dir /tmp/.go-tour-i18n-generation/<locale>/<batch-id>-recovered
```

`recover` 先重验 CURRENT bundle 与正式 batch state，不覆盖 formal output；随后逐 expected output 验证 filename identity、UTF-8/no-BOM/single-LF、protected restore、glossary 与 machine candidate。损坏 archive 的总体读取/完整性错误保留为 issue，但错误发生前已经完整读取并通过上述验证的成员可以独立复用；同一 output 若出现不同的有效字节则 fail closed。命令输出 `COMPLETE` / `PARTIAL` / `NONE` 与精确 `missing:` 列表；`PARTIAL` 是成功的恢复诊断，不表示 formal import 已完成。Generation 只生成 `missing` Unit 并补入新的 recovered staging，最后仍必须让该目录满足 exact set，再走普通 `generation-bundle import`、`process` / `retry` 与 automatic validation。recovery staging 不新增 provenance/schema，正式 provenance 仍只由后续 import 记录真实 provider/model。

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

### Same-unit representation bridge 的 ordinary QC template

Surface-bounded predecessor 也可通过独立记录的 ordinary QC bridge 提供此 template，但 Surface finding/regions 不作为本轮 QC authority。source-owned frozen mapping 与 passed raw 由机器组合，新 template 必须 exact restore 当前 reviewed candidate；旧 Surface editor 本身仍拒绝 generic QC 使用。模型只消费新的 bridge template 和自身 QC finding，不重建 mapping、不扩大 finding locality，不修改旧 Surface authority 或零输出 batch。构造、currentness 和 fail-closed 边界见 [Surface→QC bridge 合同](TRANSLATION_UNIT_REPLACEMENT.md#surface-bounded-predecessor--independent-ordinary-qc-bridge)。

正式 `quality_check` revision 可显式绑定已记录的 `representation_bridge`。这仅修复 restricted predecessor 的编辑能力，不改变 formal source/input/UnitID、reviewed candidate、原 rating/finding 或 attempt。`formal/protected-candidates` 此时是机器 authority 已证明能恢复原 reviewed bytes 的 ordinary template；`formal/representation-bridge` 冻结 receipt 和完整 dependency graph，`formal/historical-protected-candidates` 保留旧 restricted raw。模型只能编辑提供的 template，保持 tokens exact/unique，并做 finding-anchored locally complete revision；不得重建 mapping 或重新保护 candidate。无 receipt 不自动使用此分支，initial、普通可编辑 QC、Surface/Preview 与 current-validation comment contract 保持原义。构造、失败语义与 retry 见[同 Unit bridge contract](TRANSLATION_UNIT_REPLACEMENT.md#同一-formal-unit-的受限-transport-bridge)。

### Current-validation predecessor codec recovery

whole approved interval consumer 以 [contract v2](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md#derived-transport-command-contract-v2已实现真实使用待-gate) 为入口：legacy provenance 与 approved-interval representation authority 独立，whole parent immutable/opaque，不建立 target atom mapping。必须以 typed machine coordinates 证明全部 editable finding regions 与 parent 无交集，重建 exact restores，并保留 current glossary/representation、active validation、独立 re-QC/A-only。export-only `--approved-interval-selection <repository-relative JSON>` 只能出现一次，要求同一显式完整 `--content-package`；current-check 不接受 selector/override，只从 `formal/approved-interval-selection.json` 的 frozen exact bytes 独立重建。proof schema 为 `go-learning/approved-interval-transport/v1`，member 为 `formal/approved-interval-transport/<current-unit-id>.json`；unknown schema/mode、missing/extra/tampered proof 均 fail closed。process/retry 从真实 initial imported Bundle 重建同一 decoder/provenance（Snapshot 的独立 admission 见下文），不回退普通 Page restore；retry 继承相同 selection/proof，真实新 import 的 provider/model 与 historical legacy authority 分离。引用和 proof 均不回写064 manifest。实现不授权真实调用，仍须独立 first-real-use gate。

**Candidate Snapshot historical-PASS admission：**历史 evidence integrity 与当前 implementation
能否逐字节重建旧 derived proof 分开。仅 Snapshot 可在原 approved batch 完整 full-PASS、
所有 Unit 仍为同批 latest selection 时，验证 canonical result/validation 原 bytes、exact initial
receipt/archive/inventory/proof、全部最终 attempt/raw/import history，以及 frozen decoder 对
candidate 的 exact restore。随后独立重建当前 finding/representation 语义，要求完整 typed
projection/inverse、language/provenance 与非 implementation dependencies exact；仅既有
code-owned approved implementation dependency refs 与 validator implementation identity 可不同。
嵌套 `current-validation-predecessor` transport 同样必须由 current implementation fresh rebuild；
仅 canonical producer implementation path inventory 内的 SHA、producer identity 与 adapter authority
identity 可 drift，old/current SHA 均须有效且分别绑定 historical Bundle/current input exact bytes。
完整 typed semantic body（含三种 representation identity、Template/TemplateSHA256、source、
Snapshot/reopen/selected scope）及全部非 implementation dependencies 仍须 exact，并重新验证依赖。
missing/extra/orphan implementation member 或多个 transport 的 binding 冲突均拒绝。全部 typed
comparison 通过后，仅为 historical invocation identity 比对使用 exact original transport/member
bytes；不把历史 transport 声明 CURRENT，也不用历史 implementation 执行 current reconstruction。
frozen inverse 仍须 exact restore candidate，每个最终 candidate 仍须当前 active validator PASS。
result、validation 与全部引用的 SHA 在本次读取和 commit 前重验；missing/tampered/unknown evidence、semantic/source/membership/候选变化、
不完整 PASS、冲突 successor 或当前 validation failure 继续拒绝。历史 proof 不被改写或标为
CURRENT，不把旧 PASS 当作 QC/A，不豁免 source、glossary、membership 或 candidate eligibility。
此 admission 不授权 retry/revalidate 刷新 proof、同批 re-export 或新的 Generation invocation；
有 transport-recovery authority 的 batch 继续使用原严格 authority/currentness reader。

current-validation predecessor 的 protected raw 可能属于 sealed Surface/inherited bridge 表示，不能拿普通 current Page decoder 假定其可编辑。普通 restore 失败后，仅按 [representation-only transport contract](TRANSLATION_UNIT_REPLACEMENT.md#current-validation-predecessor-representation-transport) 确定性组合既有 source-owned codec；不能借 QC-only exclusive bridge 改写本轮 authority。Bundle 的 `formal/current-validation-predecessor-transport` 冻结 exact bindings，`formal/historical-protected-candidates` 保留历史 raw；Generation 只编辑 supplied `formal/protected-candidates`，不重建 mapping。comment-revision transport 仍仅开放原 reopen 授权的 slots。当前 source/formal input/Unit delta 非零、missing import/codec 或未知新 protection 都拒绝；没有可以从 engineering impact 的 `unknown=false` 推导的 PASS。export/current-check 与 import/process/后续 Snapshot 使用同一 response decoder reconstruction，仍须独立 validation/re-QC/A-only。无 formal metadata delta 的 pending batch 可保留；任何未关闭 Unit 继续阻止整批 Bundle，不自动补 receipt 或输出。
