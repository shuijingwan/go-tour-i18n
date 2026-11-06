# Multilingual language-defect exhaustion 工程 API

`internal/languageaudit.Scan` 是跨 locale/surface 的纯函数，不写文件、不调用网络、不产生 finding 或 Reviewer judgment。正式 current validator 使用下文 versioned locale authority 解释其结果：deterministic defects 阻断，review candidates 进入既有 Reviewer transport。它不关闭既有 finding，不替代 Independent Reviewer，不创建平行 lifecycle。

## 输入与 policy

调用者提供 source/target `Document`（有序 `Block`、typed `Piece`）和显式 `Policy`。locale metadata 只证明 locale 身份，不能独自推导间隔、标点或语言风格。`Locale` 是报告身份；scanner 不比较 locale 字符串，也不包含路径、中文 target 文本或 Surface finding 特例。

`ProseScripts` 和 `SpacingRule` 使用 Go `unicode.Scripts` 的 Unicode primary-script 名称。script adjacency 只在明确指定的 script pair、technical representation kind 和直接 lexical boundary 上检查。code、link label、domain/identifier 和已有 keep identity 都可以提供 technical edge；不同 script 相邻本身不是 defect。combining marks 保留并跟随其字母；bidi/joiner controls 不被擅自解释成缺失空格。

例如汉字 profile 可以显式要求 Latin technical identity 与 prose 分隔；日文 profile 可以不选择这条规则。Arabic、Cyrillic、Hangul、Devanagari 与 Latin 或其它 script 的组合使用相同 API。Latin-script locale 仍能使用 source phrase overlap 枚举候选，不假设 Latin 就是英文。未知 script、未知 piece kind、不完整 policy 或非法 UTF-8 返回 error。

`SentenceEnd`、`SentenceTerminators`、`ClosingQuotes`、`ForbiddenClausePunctuation` 都是 profile 数据。句末规则仅应用于具有目标 prose script 的可见 prose block 的末尾/结束引号前；不检查 code、URL 内部句点，也不把所有 ASCII `.` 视为非法。ellipsis 不按单个句点枚举。dash 检查要求两侧都有该 profile 的自然语言 prose，且匹配区间不属于 code/identity。

只读工程 probe 的 profile 仍是明确选择的 audit 假设。正式调用仅接受 `locale.json.language_audit` authority；不能把 fixture profile、locale 名字或工程示例自动升级为所有 locale 的语言标准。

## 两类输出

- `deterministic-defect`：typed representation 上可确定的显式 profile 违反。目前包括 technical/prose script adjacency、配置禁止的句末标点及 clause punctuation。只有 profile 显式禁止 punctuation-only paragraph 时，后者才属于此类。
- `review-candidate`：必须保留语言判断。目前包括 source-prose overlap、protected visible word 的 overlap、heading 中的 protected prose symbol、punctuation-only paragraph，以及 source 连续段落中的相同 link 在 target 被移成独立段落的形状。

residual 当前枚举 source 中已出现的最大连续词语 overlap，不执行语言识别，也不声称穷尽 source 中没有出现过的其它未译外语。阈值由 profile 明确选择，同 script 的共享词也只构成候选。residual 枚举不自动决定 Go、Kubernetes、schema、Command Prompt、issue 等是否应翻译。`ProtectedWord` 仍可进入候选；冻结一个 lexical atom 不等于已证明它在所有上下文中都是应保留的技术名称。相反，explicit code、href、identifier/domain 和调用者提供的已确认 `ProtectedIdentities` 不成为 residual defect。opaque action、literal/identity 即使不含字母，也会截断 residual phrase，不把两个不连续的语言 span 拼成一个候选。返回的候选可以包含合法专名或共享词：它们不是错误率声明，也不授权修改。

paragraph split 的 source link binding 可以机器确定，但“该处是否允许语义重组”仍交 Reviewer。合法 Markdown soft-wrap 不新增 AST paragraph。HTML/Markdown code block、script/style、`aria-hidden` 装饰内容和模板 action 不作为普通 prose；初始 `hidden` 的 tab panel/dialog 仍可能通过交互可见，必须扫描。普通 HTML 源码换行在 visible view 中保留为空白，不误当作未知模板输出；代码注释语言完整性继续由既有 code-comment authority 处理。

## 现有 representation 的只读 adapter

`internal/sitecontent.ProjectLanguageAudit` 复用 Goldmark AST/当前 Markdown rendering options、HTML tokenizer/正式 block vocabulary、inline protector、template range lexer、dynamic attribute mask 和 registered shell YAML extractor。没有新增 HTML/Markdown/YAML regex parser，也不改变 reconstruction、editable slots 或 Unit IDs。Prose 中的已证实技术身份复用现有 `classifyProseTokenInRole` 的 namespace/URL/API authority，不通过大写、下划线或点号猜测身份，也不启用 named-symbol 位置推断。纯 scanner 的 `Identity` 必须由 adapter 或调用者提供这种正面证据。

HTML link label 和 code 是 typed visible pieces；有正面 identity 证据的 link label 使用 `TechnicalLink`，既保留 href binding/technical edge，又不进入 residual prose。href、lang、canonical URL、键名和机器属性不进入 prose。quoted `alt`/`title`/`aria-label` 及 social description/title metadata 独立成为 block，避免把属性文案拼进 link label。template actions 仅验证/遮蔽，不执行；其未知 rendering 是 opaque barrier。YAML 仅消费既有 extractor 声明的语言 slots，不扫描 attribution、company 或其它 locked identity。

diagnostic 给出 owner、block locator/context、parser source range、visible rune range 与局部 evidence。Markdown 经 AST rendering 后的 source range 为 `-1`，locator 明确带 `markdown-rendered/`；visible rune offset 不等于 raw byte offset，尤其不能用它编辑 entity、UTF-8 或 YAML 编码。projection 不支持或 parser 拒绝的 representation 返回 error，不允许将未扫描区域包装为 PASS。

## Surface010 只读 regression

opt-in test `TestLanguageExhaustionSurface010Readonly` 从 exact recorded receipt 的 finding set 发现资产，校验 receipt/Markdown refs、TU candidate SHA，或沿既有 frozen closure reader 精确解析 structured owner 并重建 target。当前 raw source 必须匹配该 fixture；不会重跑 Reviewer、currentness workflow 或创建 Bundle。真实 owner/path 的 expectation table 只存在于这个 regression test，不进入 scanner/policy。

`GO_TOUR_LANGUAGE_EXHAUSTION_READONLY=1` 启用 probe；可通过 `GO_TOUR_LANGUAGE_EXHAUSTION_EVIDENCE` 指定 `/tmp/` 下的工程 JSON。报告包括每个 class 的 deterministic/candidate counts，以及每项 owner/path/locator/rune location。`fixture_machine_covered` / `fixture_review_candidate_covered` 是 recorded finding 的重叠覆盖数，不是互斥分类；完整资产扫描得到的 diagnostics counts 另列，并不冒充 Reviewer 的 exact finding 数量。重复扫描必须完全一致。测试的 PASS 只证明 fixture 枚举/分类合同，并不代表这些 target 的语言 PASS。

原只读 probe 的 PASS 只证明工程枚举；正式 integration contract 与首次真实使用 gate 见下文。候选永远不直接升级为正式 FAILED。

## Cross-surface Tour adapter

Tour 继续使用同一个 `languageaudit.Document` / `Scan` 和相同 locale policy；scanner 不认识 Tour、Learn/Docs 或 Homepage。`ProjectTourLanguageAudit` 只组合 adapter：既有 Present parser/`present.Style` → 既有 HTML visible view → scanner。Section title 遵循实际 Tour action template 的 plain escaping；正文、列表和 link label 遵循 Present 的 font/link grammar，不把 Markdown 语法套进 Present。未映射的 element 返回 error，不宣称完整覆盖。

`i18n.ReadTourLanguageAuditUnits` 通过完整 native workflow membership、locale metadata、status exact set、canonical ready path/source SHA 和既有 candidate validator 选择 TU。不枚举 candidate/staging 目录，不凭历史 A/PASS 假定语言无缺陷。Page 和 eligible Example 分别计数；非 eligible Go Example 仍按 source-owned code 验证为没有开放语言 payload，不伪装成新的语言 TU。

Page preformatted 与 Example 的语言部分复用 source-owned comment inventory/disposition、immutable Go structure 和 source/target comment mapping。只有 source 正式开放的 teaching-comment payload 进入 prose；target 是否含英文不能决定开放范围。Go syntax、字符串/字符字面量、机器注释及静态代码不进入残留语言或 locale punctuation 检查。普通 prose 中缺少技术身份正面证据的名称或代码形状仍可能成为 review candidate，不能只凭拼写把它们豁免，更不能把候选当语言错误。

源代码实际绑定的 comment identifier 通过 `languageaudit.BindIdentities` 映射为 typed pieces。该 API 仅公开已存在的 exact identity/Unicode boundary normalizer，供任何 adapter 使用；不增加 script/locale 规则或新的 defect class。`ProjectLanguageAuditText` 表示 UI/metadata/comment 的 plain strings，不递归解释其中看起来像 HTML 的字符。

Tour UI 使用 `ui.LoadFromFS` 的完整 working-tree English/locale catalog 与 kind/coverage 校验。rich message 走既有 constrained markup；plain message 保持 plain。source 声明的 `{name}` placeholder 按既有 UI placeholder grammar 校验并作为 opaque runtime boundary，不把变量名当英文 prose。article title/subtitle 使用 native article metadata membership/reader；course description 使用 native complete metadata reader。schema v1 使用对应完整 Page source 作为 overlap context；schema v2 使用其 canonical source description，不把 v2 contract 强加给 v1。

opt-in `TestLanguageExhaustionTourReadonly` 使用与 Surface010 probe 相同的 `auditFixturePolicy` 和当前完整 glossary keep。当前 zh-CN fixture 只是该显式 profile 的第二种 surface，不新增 Tour locale policy。启用变量继续为 `GO_TOUR_LANGUAGE_EXHAUSTION_READONLY=1`，通过 `GO_TOUR_LANGUAGE_EXHAUSTION_EVIDENCE` 将工程报告写到 `/tmp/` 子目录。测试两次重新加载完整 current inventory，再比较 normalized representation 与全部 scan diagnostics；不只重扫几个 sample。

报告区分 physical asset、TU、UI/metadata record 与 visible field counts；每项 hit 保留 locale、surface、owner、适用的 unit ID/kind、class、location、reason 和 classification。UI catalog、article metadata、course metadata 是独立资产，不把 message/description 当新 TU。候选可含合法专名、共享词或未获 representation 身份证据的技术用语，数量不等于错误数。PASS 表示覆盖、分类、重复性与保护边界成立，不表示 Tour 语言审核通过；所有发现只供下一阶段 assessment，不进行 finding recovery 或正式集成。

## 正式接入合同（首次真实使用前仍须 command-contract gate）

locale policy authority 位于既有 `locales/<locale>/locale.json` 的 `language_audit` 字段，schema 为 `go-learning/locale-language-audit-policy/v1`，包含递增的 `policy_version` 和完整 `policy`。policy 的 locale 必须与 metadata 一致；profile、Unicode prose scripts、spacing kind/pair、句末替换、terminators、closing quotes、clause punctuation、paragraph 与 residual 设置均为显式数据。缺失、未知字段/版本、非法 script 或不完整配置 fail closed；没有 locale-name switch、默认中文/英文或历史 A 豁免。技术 keep 不由 policy 自创，仍从当前 glossary/representation identity authority 注入。

`locale init` 增加必填 `--language-policy <repository-relative JSON>`；输入是上述完整 policy authority，必须匹配待初始化 locale。维护者在正式 Generation 前审定此数据，init 校验并复制进既有 locale metadata，沿用原 staging/no-overwrite/rollback；不创建 policy receipt、全局 registry 或新 lifecycle。旧 locale 缺 policy 时停止，经审定后使用下文 [existing-locale adoption](#existing-locale-policy-adoption)，不能在 validator 内自动补全。本轮工程不为真实 locale 写 policy，不发布任何语言 authority。

current TU machine validation 在既有结构、protected identity、comment gate 之后消费同一 scanner；native Tour 与 registered adapters 通过 normalized view 协议提供 source/target。缺 adapter capability fail closed。current structured output/Generation 验证也执行同一 policy/scanner，覆盖完整重建 target、metadata 与 shared composition，不只检查开放 slot。只有 `deterministic-defect` 阻断；`review-candidate` 永远不改变 validation/QC/Surface decision，也不产生 finding/reopen authority。

policy metadata SHA、scanner/view/adapter 实现 bytes 进入现有 machine authority/current validation 依赖。旧 accepted evidence 保持 immutable；historical frozen proof 不获得新 current policy 要求，但当前 eligibility 必须重新验证。新确定 defect 按既有 affected/current-validation/reopen/Surface recovery contract 处理，不重写旧 A，不 silent carry，不自动制造 revision finding。

QC、Tour Surface 与 Integrated Surface Reviewer transport 增加完整 `machine/language-review-candidates.json` auxiliary evidence，使用 `go-learning/language-review-attention/v1`；它是 transport attention，不是 receipt/judgment。报告绑定 locale policy Reference 与执行实现 identity，包含 locale、surface、owner、适用的 unit ID/kind、class、locator、bounded context 和 `classification=review-candidate`。所有当前输出先通过 deterministic gate；历史 carried context 同样不能借旧 A 绕过 current gate。Reviewer 仍须完整阅读 source/target/glossary，candidate list 仅帮助 exhaustiveness；零 candidate 也不表示语言 PASS。完整 inventory/hash/current-check 继续重建 transport，旧 Bundle 不自动升级。

本变更修改 init input、Generation/validation/currentness 与 Reviewer transport acceptance；第一次真实使用必须重新完成 current canonical docs → implementation → direct CLI/contract tests gate。工程测试只使用临时 fixture；不运行真实 zh-CN validation，不导出真实 Bundle，不修复既有 Surface/Tour 缺陷。


### 当前 command / transport 合同矩阵

| 入口 | 本次变化 | mutation / stop |
| --- | --- | --- |
| `locale init --locale <locale> --language-name <name> --english-name <name> --language-policy <repository-relative-json>` | 必填显式 policy；输入 authority schema v1，locale exact；absolute、`..`、symlink、非 regular、unknown/missing 字段拒绝 | 沿用一次性 scaffold/no-overwrite/rollback；成功 stdout 仍为原 result JSON；停止于首次 Generation 前 |
| `status init`、TU/locale Generation export/current-check | policy 必须存在且有效，完整 locale metadata 进入 Generation transport | 原写入/readonly、exact replay/CAS、stdout 不变；缺 policy 不继续 |
| TU process/retry/current validation、structured import/Generation check/recovery | 当前 policy + normalized view；仅 deterministic defect 阻断，依赖 SHA/currentness 继续 exact | 原 lifecycle、publication、idempotence、prior/CAS 不变；失败保留原 evidence，不制造修订授权 |
| QC `reviewer-bundle` / `reviewer-bundle-check` | 在既有 v1 manifest inventory 中必带 attention auxiliary 与 exact policy metadata；完整 source/target 保留 | transport-only，current-check 重建 exact bytes；不同内容不覆盖既有 evidence |
| Tour `surface-review export` / `reviewer-bundle` | review-package 与 Reviewer Bundle schema_version 升为 2；完整 Examples 与 attention，UI/article/course descriptions 均扫描 | transport-only；不是 Stage A judgment |
| Integrated `surface-reviewer-bundle` / check | 既有 v1 file inventory 增加 attention auxiliary；fresh/carried 的 current deterministic gate 不可省略 | transport-only，scope/receipt schema 不变；停止于 Independent Reviewer 前 |

所有 check 的 currentness/recovery/replay 仍由各自 canonical command 合同约束；这个矩阵不授权重放已知/未知 mutation。新 validator 若使已有内容不再 eligible，必须停止并走既有 recovery，不能删除、覆盖或盲重放。首次真实使用前尚须 Coordinator 完成完整 Formal Command Contract v3 gate；本工程阶段不提供真实 mutation 命令。

<a id="existing-locale-policy-adoption"></a>

## Existing locale：language-policy-adopt canonical contract

新 locale 使用原 `locale init --language-policy`。已经存在但尚无 policy 的 locale 使用独立的 `locale language-policy-adopt`；validator 不自动补全，init 不覆盖 existing locale。此 command 仅扩展既有 `locales/<locale>/locale.json.language_audit`，不新增 locale metadata、policy registry、receipt/pointer 或 lifecycle。维护者须先审定完整 policy 数据；机器校验不替代语言/script policy 的审定。

```text
tour-i18n locale language-policy-adopt --locale <locale> --input <repository-relative-policy.json>
tour-i18n locale language-policy-adopt --locale <locale> --input <repository-relative-policy.json> --apply --prior-sha256 <preflight-prior> --input-sha256 <preflight-input-policy-sha> --result-sha256 <preflight-result>
```

`OPERATION` 默认是 readonly preflight，只有显式 `--apply` 才 publication。`INPUT_KIND` 是完整 `go-learning/locale-language-audit-policy/v1` authority JSON；本次 initial adoption 仅接受 `policy_version=1`，所有 policy 字段必须显式完整、locale exact，复用 generic schema/script validation。不能按 locale 名字推导任何规则。严格拒绝 malformed/duplicate/unknown JSON 字段、尾随 JSON、不支持的 schema/version、locale mismatch。已有不同 policy 的 successor/update 不在本 command 范围。

`INPUT_PATH_CONTRACT=repository-relative`：例如 `engineering/policy.json` 合法；即使位于 repo 内的 `/home/user/repo/engineering/policy.json` 也非法。拒绝 absolute、非 canonical path、空 component、`.`、任意 `..` traversal、backslash、路径任一 component 的 symlink、非 regular file。target 固定为已存在、regular、无 symlink 的 `locales/<locale>/locale.json`；严格按 current `Locale` schema/identity 校验，要求 locale/html_lang exact、language_name/english_name 非空、phase=scaffold、translation_unit=present.Section。不存在或 malformed 的 metadata 不能借 adoption 初始化/修复。`language_audit:null` 属于非法已有 authority，不当作 missing。

允许的 flags 只有 `--locale`、`--input`、`--apply`、`--prior-sha256`、`--input-sha256`、`--result-sha256`。两个 mode 都必填 locale/input；apply 额外必填三项 lowercase exact SHA-256；preflight 禁止这些 apply-only SHA flags。重复 flag、unknown flag、positional arguments 全部拒绝；没有 force、overwrite、record、output 或修改其它 metadata 的 flag。

成功 stdout 为一个完整 JSON result object（缩进、末尾 LF），`schema=go-learning/locale-language-audit-adoption-result/v1`，字段 exact 如下；不是 receipt 或 wrapper：

| 字段 | 含义 |
| --- | --- |
| `schema` | 上述 result schema |
| `locale` | exact target locale |
| `policy_schema` | `go-learning/locale-language-audit-policy/v1` |
| `policy_version` | initial adoption 支持的 `1` |
| `input_policy_sha256` | input file 的完整 raw bytes SHA，包含 whitespace |
| `prior_sha256` | 本次观察到的 existing metadata 完整 raw bytes SHA |
| `result_sha256` | 确定性 expected metadata bytes SHA；already-current 时等于 prior |
| `already_current` | 本次写前检查已包含语义 exact 的同一 authority；first adoption 为 false；不证明哪次 invocation 写入 |
| `applied` | 仅本 invocation 完成 atomic replacement、directory sync 与结果重读确认时为 true |
| `mutation` | preflight/no-op 为 `none`；已确认 publication 为 `known` |

缺 policy 的 result construction 只在 root object 的最终 `}` 前插入 `,"language_audit":<canonical authority JSON>`；原文件的所有 bytes、字段顺序、whitespace 与尾部保持 exact，不重新 marshal 整份 metadata。新 authority 使用 production type 的 canonical JSON。already-current comparison 忽略 JSON whitespace/object key order，保留 rule arrays 的 exact ordering/values；即使 input 格式不同，已有 metadata bytes 也不重写。

### CAS、并发与 publication

preflight `MUTATION_EXPECTATION=none`，不创建目录或暂存文件。apply 的有写入路径 `MUTATION_EXPECTATION=known`，唯一持久化写入为 target metadata；不能修改 language_name、english_name、html_lang、phase、translation_unit、glossary、status、candidate、QC、Snapshot、authorization、Surface、closure、content-scope 或任何其它语言结果。

apply 使用 Linux locale directory inode 的非阻塞 exclusive flock 串行化此 publisher，不建立 lock file；其它平台 apply fail closed，readonly preflight 仍可用。锁内重读 input/target，并在同目录 private staging（exclusive create、保留 mode、完整 write/fsync）之后、atomic rename 之前再次重读。检查完整 prior/input/result 三项 SHA，以及 input/metadata/parent-directory identity；变化即停止，不自动刷新 prior 或扩大 expected result。目录 handle 固定 publication 与 cleanup 的位置；成功后 fsync directory 并重读 exact result bytes。

并发 adoption publisher 不能越过同一目录锁。其它 repository writer 也不得绕锁并发写此 metadata；观察到的 concurrent input/metadata/namespace mutation 在 publication 前 fail closed，publication 后发现不一致则 `mutation=unknown`，保留现场而不 rollback/覆盖。此锁不是全仓库其它 command 的全局事务锁。

幂等与 failure stop：

- missing policy + matching prior/input/result + 合法 input：只产生一个 exact metadata result。
- exact same policy：preflight 返回 `already_current=true, applied=false, mutation=none`；持有当前 preflight 三项 pins 的 apply 同样 no-op。**apply 仍严格要求 current prior CAS，即使 policy 相同，旧 prior 也拒绝**；重复观察可重新 preflight，不能把旧 prior 自动更新为新 prior 后继续写。
- existing different policy：两个 mode 均 fail closed，不提供覆盖/successor。
- prior、input 或 expected result 不匹配：publication 前停止；input 即使只改变 whitespace，旧 input SHA 也不能复用。
- preflight 或 publication 前失败：stderr 错误标注 `mutation=none`，无成功 JSON；暂存文件清理，不改变 authority。
- atomic replacement attempt、durability/final verification 或 staging cleanup 无法确认：stderr 标注 `mutation=unknown`。若已确认 publication 但 stdout 发送失败，stderr 标注 `mutation=known`；终端/evidence 丢失仍按未知终态诊断。

recovery/replay 必须先读取 actual metadata，与保留的 exact prior/result SHA 和 input SHA 对照。终态不明时 preserve/diagnose，禁止盲重放。actual bytes 等于 expected result 只证明 observed already-current state，不能倒推出前次 invocation 的 provenance；不同于 prior/result 的 bytes 不可覆盖或自动修复。历史语言、QC、Surface evidence 保持 immutable；adoption 不赋予历史 A 豁免，也不代替新的 current eligibility/validation。

`SUCCESS_STOP`：adoption/preflight result 可审查，停止于任何 downstream Generation/validation/Bundle 之前。`FAILURE_STOP`：保留 input、terminal evidence 和 metadata，先诊断 mutation provenance。本 command 是 new formal CLI：本工程完成后仍须另行执行 current canonical docs → implementation → direct CLI/contract tests 的 first-real-use gate；此文档示例不授权真实 adoption。
