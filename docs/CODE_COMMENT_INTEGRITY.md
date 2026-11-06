# 跨内容包代码注释完整性

本合同是 source inventory、target comparison 与 current TranslationUnit validator 的共享边界，不创建 TranslationUnit、语言判断或 lifecycle receipt。Tour 保留 present Page / eligible Go Example；Learn/Docs 一个物理 Page 仍是一个正式 Page TU。内部 code block/comment slot 只用于 parser、保护、还原、校验、审计和精确影响分析。

## 既有 Tour contract

`internal/i18n/preformatted.go` 的 `preformattedBlocks` 复用 Present 缩进结构；`analyzePreformattedGo` 使用 Go parser/scanner，识别完整文件、安全 declaration/statement fragment 与限定 basic-type reference list。`SafeTeachingCommentBodies` 导出完整 body、代码标识符及其坐标，Learn/Docs 的 `teaching.go` 已直接复用。未知片段和 Page block comment 保持 protected。Page comment delimiter、缩进、换行、非注释代码和 referenced identifier identity 不可变；可翻译 body 不改变 Page 边界。

Example 的 `scanGoExampleComments` / `classifyGoExampleComment`、protector 与 validator 对完整 Go 文件执行严格 token/comment 顺序和 immutable bytes 检查；只有 natural payload 开放。Go directives、line directives、legacy build constraints、Present/test/generated markers 和 non-natural annotations 保持 exact。Page 的完整 inline/link pair 可按既有规则随语言移动，不能把 Example 的全局 token-order 规则套给 Page。

既有回归：`preformatted_test.go` 的 classification、literal exclusion、static/block-comment preservation、identifier/layout 与真实 Tour fixtures；`go_example_protect_test.go` / `go_example_validate_test.go` 的 roundtrip、special comments、代码/token/layout 保护；Learn/Docs `complete_code_test.go` 的 full-file/fragment、Goldmark fence/indent 和 HTML pre fixtures。completeness gate 不改变这些正式表示和保护规则，在 current validation 中追加注释完整性检查。

## 唯一 source inventory

`tour-i18n code-comments unsupported --output-dir /tmp` 导出原 extractor 无法支持的完整 source block 记录；`code-comments coverage --output-dir /tmp` 生成 block disposition、扩展后的 source inventory 和独立 frontier source preflight。可选 `--locale <locale>` 只比较这一门 locale 的已证明 slots，输出单独的 `code-comment-<locale>-closure-audit`，不覆盖跨 locale 报告；source 未完整时明确标注 `PARTIAL_UNSUPPORTED`，计数不包含未知语法中的潜在注释。`code-comments audit --output-dir /tmp` 在相同 source disposition 完整后才允许跨 locale 比较，未知块在进入 locale reader 前阻止执行。这些入口均是离线只读工程审计，只写指定目录的 diagnostic JSON/TXT，不写 candidate、Snapshot、authorization、QC、closure、Surface 或 activation。正常重跑由 Maintainer Local terminal 执行。

Tour source 来自 canonical source Catalog；Learn/Docs 来自 current registered package 的 frozen source、stable Page index 与 `WorkflowDocuments`，不扫描 Preview HTML。所有 locale 共用一次 source inventory。source SHA 独立绑定完整正式 Unit；slot identity 绑定 package、source path、parent kind/ID、parser block coordinate、comment ordinal、preceding lexical token coordinate 与 source semantic kind，不依赖行号。上游 source/structure drift 后必须重建 inventory，并重新证明对应关系；此 ID 不承诺跨语义变动自动匹配。

Goldmark 提供 Markdown fenced/indented block；HTML tokenizer 提供 pre text 和 entity decoding，fence 内的 HTML 不重复统计。所有实际 block 都保留 language、extractor、hash 和 unsupported 状态。未声明语言先复用既有 Go safe-fragment contract，其他格式仅按下述 residual source proof 的完整 grammar 或明确 source semantic role 分类。不能从 `//`、路径或普通 prose 猜语言。

明确的 `json` / `jsonc` / `json-schema` fence 使用同一 Go scanner 的精确 string/comment 边界，加受限 JSON/schema grammar：object/array、JSON atoms、type placeholders、trailing comma；不把 URL、field、identifier、字面字符串当注释。其他语言或未知 grammar 必须报告 `UNSUPPORTED_COMMENT_LANGUAGE`，保留 source coordinate，不以零注释冒充完整覆盖。未支持的 block 不可通过 future hard gate；inventory 中已提取 slot 的计数不包括未知语法中的潜在注释。

semantic kind 与旧 representation protection policy 分开：teaching-comment 是潜在可翻译自然语言；machine-semantic-comment 必须 protected；既有 non-language/structural annotations 按原 contract exact；无法确定为 `unclassified`。current completeness 按 canonical source semantic kind 校验，不能从 target 或旧 protection policy 推断 eligibility。formal input/Unit identity 不变时不创建 representation successor；确需改变正式保护表示时，仍须既有 affected-only representation authority，不能直接改写旧 protected bytes。

## Source block disposition 与覆盖完整性

每个原始 parser block 必须按原顺序得到且仅得到一个 disposition：

- `comment-extracted`：supported comment language，已有精确 comment slots。
- `comment-free-by-grammar`：完整 grammar 或 source-owned AST projection 证明无注释。
- `terminal-transcript`：明确 prompt 或受限 command grammar 的有序 command/output records；不将 output 再解释为程序语言。
- `non-code-output-or-literal`：source semantic markup 或专门 grammar 证明是输出/字面量。
- `machine-semantic-opaque`：source authority 明确其机器语义，整体保持 protected。
- `ambiguous-unsupported`：缺少足够证明；不能当作零注释。

`ValidateSourceCommentBlockSet` 对照原 parser 的 block exact set，绑定数量、顺序、coordinate、原始 code/hash、声明、enclosure、source offset 和 source context/markup；禁止丢块、插块、换序或改写 source。`ValidateCommentBlockDispositions` 检查互斥语义和证明字段，transcript records 还须连续覆盖全部原始 bytes。已分类的 block 不得同时 unsupported；ambiguous 必须保留独立 residual record。任何 ambiguous 都使 completeness gate 失败，即便已知 slots 的比较没有 missing。taxonomy 汇总桶互斥：`comment-extracted` 按显式声明/严格 source 证明分桶，其余按 disposition 分桶；language counts 是另一统计维度，包含 grammar 已证明无注释的语言。

language authority 顺序是：fence info / HTML `language-*` 明示声明，既有 source parser/metadata/template 语义，最后是完整严格 parser acceptance 与无歧义的 source 结构证明。强 authority 不得被较弱推断覆盖。target 文本、locale、Preview、ASCII 关键词、路径猜测或宽松 regex 均不提供 source language authority。不能仅因为没有出现某个 comment delimiter 就声称 comment-free。

基础受限证明包括：

- Tour 参数列表和 range header 使用 Go parser/scanner，并与该 Page 实际引用、SHA 已绑定的 canonical Go Example AST 对应；注释或不匹配的片段仍 unsupported。它只证明无注释，不改变 Tour protector。range 的空白 value 按 Go range assignment 语义规范化。
- 明示 `gomod` / `go.mod`，或 source metadata 精确声明 `go.mod file reference` 时，使用既有 `golang.org/x/mod/modfile.Parse` 的完整严格 grammar；不使用 `ParseLax`，不从文件名或命令外观推断。placeholder、shell、输出和其他 grammar 不通过即保留 ambiguous。comment 坐标来自该 parser，字符串中的 `//` 不是注释；`indirect`、retraction rationale、module deprecation metadata 按机器语义 protected，普通 comment 复用 Tour classifier。新 inventory slots 均保持旧 representation 的 protected policy，不激活翻译边界。

`SourceSelectedTargetCommentBlocks` 仅按已冻结的 source extractor 提取 target；target 不能凭自身 grammar 或 metadata 升级未知 source。无注释 AST projection 的 target code 必须 byte-identical；module target 仍须完整严格 parse。之后复用既有 slot comparison 和 locale-aware validation，不建立另一套机器或 TU lifecycle。

未知 shell/transcript、SQL、配置或 plain output 若没有可靠 parser 或 source semantic authority，保持 exact residual set，不用 `#`、`$`、`--` 等外观强行命名。source coverage 未完整时不得声称 gate-ready；active validator 对未知 coverage fail closed，不能批量改写历史 locale evidence。

### Residual source proof

`coverage --predecessor <diagnostic.json>` 读取 predecessor exact block set，验证 package、parent、stable index、route、source path/hash、block hash/code、coordinate、enclosure 与 declaration 不变；已证明 block 的 disposition、language 和 proof 必须原样保留。只对 predecessor 的 ambiguous 集合汇总新增证明。完整 source 才覆盖 canonical closure evidence；未完整不得启动跨 locale audit。predecessor 是诊断 lineage，不是正式 lifecycle receipt。

新增证明按 source authority 使用以下受限语法，不按 route/source path 添加例外：

- `terminal-transcript/v1` 支持实际 corpus 的 `$ `、`~ `、Windows drive/path prompt、`mysql>`，逐字节保存有序 command/output records。quote-aware command envelope 只证明 opaque command 的词法边界，不声称是完整 shell/cmd/SQL parser。output 始终是 literal，不能从其中的 `#`、`//`、`--` 提取代码注释。quoted `#`、URL fragment、普通参数值保留；word-start comment、heredoc、反引号、复杂/可能隐藏 comment 的 command substitution 拒绝。只接受实际需要的 plain-word `$(...)` 子集；其他输入保持 ambiguous。
- 无 prompt 的 `documentation-command/v1` 是有限完整生产式：`cd`、单路径 `mkdir`、environment assignment / `export`，以及文档实际使用的 Go `install/get/vet`、`mod init/edit`。不从任意 executable 名称扩展 shell。command bytes 整体 protected，quoted export 还须通过完整 quoted-value 与 command-envelope 检查。空白/output records 也进入 exact-set，禁止遗漏字节。
- `tour-go-template-literal/v1` 用 Go template AST 只接受 literal `raw` 字符动作，再由既有 Go fragment parser/scanner 检查完整投影；同宽 padding 保持 comment 坐标，原模板 bytes 留在 skeleton。动作不得藏在 Go string/comment 内；控制流、其他函数、未知 action 或 Go parse failure 均拒绝。不执行任意模板逻辑。
- 无语言声明的具体 module/workspace 示例使用官方 `modfile.Parse` / `ParseWork` 完整 grammar，共用原 module comment walker；不要求路径可下载，不使用忽略未知语句的 `ParseLax` 作 blanket proof。格式 reference 中的 `<var>` markup 明确 metavariable notation，且无 comment token 时才整体 `machine-semantic-opaque`；它不冒充具体 go.mod semantic validation。无 markup 的未知 placeholder 仍不能过关。
- `tour-scanner-json-documentation/v1` 继续复用唯一 restricted JSON scanner/grammar，只增加 object-member excerpt 的外层结构投影与 member-position `...` elision。原 bytes/hash/comment 坐标不变；string 中的 URL/`//` 不是注释。target 只能复用 source 已选的 extractor。
- `mysql-tutorial-grammar/v1` 是独立注明范围的受限 SQL tokenizer/完整生产式，仅支持当前 tutorial 的 DROP/CREATE/INSERT/SELECT 形状、columns/values/WHERE；quoted string 与 backtick identifier 拥有自己的 delimiter。真实 SQL comment 或未知生产式保持 ambiguous，不以 substring 搜索证明无注释。
- `source-literal-grammar/v1` 验证 rooted file-tree、angle-delimited path metavariable、semantic-version / placeholder listing、官方 module path/version pair、fuzz typed-literal data、HTTP headers + JSON body 等 exact grammar；HTTP documentation 的显示 Content-Length 保留为 literal，不声称 pretty-printed body 长度相同。tuple、API quotation、mutation trace 和明确 output declaration 还必须绑定相邻 frozen source paragraph，不能从 target prose 或远处的 keyword 猜测。Markdown context 保留 soft/hard line break，避免将相邻词拼接。

synthetic unknown block 必须保持 `ambiguous-unsupported`。这些 classifier 只提供 source-side authority，不改正式 protector、不翻译 comment、不创建 TU。active gate 复用这些证明；source coverage 完整性和相关回归是其前提，未知输入仍拒绝。

跨 locale audit 只消费 current accepted/current authorized bytes；unavailable 保留原状态，不诊断或回退历史目标。一次 audit 同时写 zh-CN exact comparison 和 machine Reviewer scope proposal，避免重复扫描。proposal 按 roadmap、package、stable Page/Example order，只包含 carry/residue 或确定性结构问题；source 证明完整不代表语言质量通过，也不启动 Reviewer。missing/extra/protected mutation/block-only structure diagnostics 同样进入 affected-only proposal。

### Future frontier source-only preflight

预检只从 frozen source manifest 读取 `/doc/` 与 `/doc/install` 的 exact source SHA/bytes，使用同一 HTML tokenizer、source classifier 和 disposition invariant。`SourceHTMLCommentBlocks` 将上游 HTML metadata comment 交给 tokenizer 作 opaque comment，允许在正式 Page adoption 之前审计 `<pre>`；不会放宽正式 `PageCommentBlocks` 的 metadata schema。它不创建正式 Unit，不改变 package authority，不计入当前 Page totals，不读 target。单独的 frontier JSON 保存 manifest identity、source identity、block records 与 counts；ambiguous 是 future adoption blocker，零 code block 的页面也保留明确记录。

## Currentness 与 comparison

Locale 顺序只读 `LOCALE_ROADMAP.md` frozen commercial order；current language registry 验证 exact set。base language 来自该 registry 对应 locale metadata 的 `html_lang` primary subtag，不维护 English locale 白名单。

Package completion 仅用于发现 scope。accepted bytes 必须由共享 current authorization / current canonical finalized reader 证明，绑定 exact source、candidate path/SHA 和完整正式 Unit set，并检查 current candidate selection。旧 Snapshot、文件存在、目录字典序、promotion 或 Preview 不是 currentness authority。无法证明时输出 `CURRENT_TARGET_UNAVAILABLE`；没有该 package 的 locale 输出 `NOT_ACCEPTED`，不计作语言失败。

对每个 source slot 保存 source/target comment、present、exact carry、structure correspondence、protected unchanged、missing/extra 和 review flags。source classification 冻结，target 不重新判断翻译 eligibility。block 数量/坐标/语言、comment style/token coordinate 和非注释 skeleton 必须对应；不能 fuzzy join、猜 missing slot 或把后续 comment 错位后称为 match。

missing、extra、protected mutation、structure mismatch、ambiguous mapping、unsupported grammar、unclassified 均使 current validator FAIL。非 English base 的 teaching-comment exact source carry 是 deterministic machine validation failure；English base 允许 exact carry。不同 bytes 只能证明没有 exact carry，不能证明翻译正确。保留 source 连续自然语言短语的 mixed comment 可标为 `english_residue_suspect`，仅是 Reviewer hint；单独 API、JSON、OSV、Go、identifier、field 等 English token 不作为 failure。

## Gate activation、恢复与 affected-only follow-up

`COMMENT_GATE_ACTIVE=yes`。canonical 接入点为 `internal/i18n/go_example_validate.go` 的 `validateTranslationUnitWithGlossary`：先执行原 representation/glossary validation，再调用 `ValidateTranslationUnitCommentCoverage`，复用 `SourceCommentBlockDisposition`、`SourceSelectedTargetCommentBlocks`、`CompareCommentBlocks` 和 `ValidateCommentCoverage`。Generation candidate validation 与 `AssessCurrentTranslation` 共用该入口，不按 locale/package 名称分支。Page/Shell adapter 实现 `TranslationCommentAdapter` 提供原 parser coordinates 和 source metadata；future registered Page package 自动继承同一检查，未知 adapter 缺少 source authority 时拒绝。

`CommentLocaleBaseLanguage` 从 current workflow locale registry 对应的 `locales/<locale>/locale.json` 读取 `html_lang` primary subtag，验证 locale/metadata identity；不维护 English locale 列表。base 为 `en` 时 exact teaching carry 合法，其余 base 必须拒绝；结构、protected 和 unknown failures 对所有 base 相同。target extraction 始终使用 source 已选 extractor，target 不得升级 language/grammar 或重判 teaching/protected semantics。

current re-attestation 将新 gate failure 记为当前 Unit 的 `affected / current_validation_failed`，绑定 policy/extractor/adapter dependency identity 与 locale metadata SHA；complete authorization 不能把这类 Unit 当 compatible carry。历史 A、Snapshot、原 validation、finalization 和原 provenance 保持 immutable。历史完整性 reader 保留原 representation/input/candidate proof，不把今日 completeness failure 伪装成旧 A 被推翻；当前 eligibility 仍必须单独经过 active gate。

exact carry、missing、extra、structure/protected failure 由机器确定 affected scope，不需要 Reviewer 先确认“是否漏译”。后续仅按明确授权的 affected Page/Example 进入 Generation exact revision、active validation、Independent re-QC 和既有 A-only successor lifecycle；Reviewer 判断修订译文的正确性、自然度与术语。mixed-language residue suspect 仍只作 Reviewer hint。activation 本身不创建 reopen、revision、Snapshot、QC 或 authorization，不 whole-package invalidation、不重做 unaffected A，也不建立第二套 TU lifecycle。

这类 current machine failure 使用 [current-validation shared reopen authority](CURRENT_TRANSLATION_AUTHORIZATION.md#current-validation-的-affected-only-reopen-authority)。只读 `quality-check current-validation-reopen` 从完整 fresh re-attestation 绑定 exact affected Units、各自的 canonical failure 和 comment slot identities/source/target；不把多个 Page 压缩成一个泛化“翻译注释”finding。未来明确授权的 record/export 继续走既有 `quality-check-reopens` 和 `--surface-reopen-id`，保留历史 A，只修列明的教学注释及局部必要上下文；不生成假的 Reviewer rating 或 representation successor。

Generation revision transport 必须实际开放该 receipt 授权的 exact teaching bodies，不能让 validator 要求修订而 opaque predecessor token 又禁止表达。共享 Bundle 使用 revision-only 的候选投影，复用 source-selected comment coordinates 和既有 body protector/restore；仅精确授权 body 可编辑，code/identifier/literal/directive/delimiter、其它注释与正文继续保护。派生 mapping/实现依赖纳入 current-check，原 source/input/raw/candidate/reopen/manifest 历史不变；output 仍是正常 protected raw → 完整 Page restore → active canonical validation。不能证明 slot、raw encoding 或 exact-set 时 fail closed，不扩大可编辑范围，不改变正式 TU boundary，也不放宽 initial 或其它 revision authority。

授权 body 的稳定身份来自冻结 source 的 block/comment coordinate，不来自可翻译 body bytes。body restore 复用既有语义分类器的封闭技术术语 namespace，允许普通技术词的合法语言形态变化；这不授权新增 code-like identifier、调用、路径、字面量或 structural binding。source 已绑定的身份仍须保留 exact unique token，不能在译文中额外复制。body 改变后仍执行 source-selected slot correspondence 和 active canonical validator；不为具体 route 或 locale 建立例外。

current-validation process 在一次 invocation 内只准备一次完整 revision projection，并复用同一 authority 下已验证的 frozen history / QC evaluation。该对象不跨调用保存，也不是历史 PASS 缓存。逐 Unit 的 restore、token、source SHA、provenance、canonical validation 与 glossary/完整 Unified Review gate 均保留；提交前重新读取 exact dependency bytes、current validator/locale authority，重建完整 revision graph 并比较，且再次校验 receipt、完整历史引用和 review currentness。检测到期间漂移时，在发布 candidate/validation/result 前 fail closed；不改 schema、receipt 或历史 A，不自动恢复任何正式失败 Unit。

真实 stale/missing/currentness failure 保留原 bytes 和最小诊断；修复 reader 的接线问题后重跑只读审计，真实 authority drift 则停止并交 Coordinator，不能以旧目标或改写 evidence 恢复。unsupported 不是无注释；缺 application URL 不是已提交/已合并；任何 partial audit 都必须明确其边界。最终所有正式语言 successors 完成后才安排一次 fresh Preview、automated acceptance 与 Maintainer HUMAN gate。
