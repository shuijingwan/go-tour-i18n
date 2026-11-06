# Page protected representation 结构矩阵

本报告是工程诊断，不是语言审核、reopen、validation receipt 或上线授权。范围为变更前实际注册的 153 个 canonical Page：Tour 103、Learn/Docs 49、Homepage 1；3 种表示。当前 package loader 只有 Learn/Docs 与 Site shell，Tour 由基础 Catalog 提供。shared chrome/YAML 等 structured assets 不扩展为语言审核；它们直接调用的共用 protector 改动接受相关夹具检查。独立 Example TU 不计入 Page inventory。

逐构造 occurrence、完整 Unit ID/route 见 [机器审计](PROTECTED_REPRESENTATION_CORPUS_AUDIT.json)。SYS-08 本轮关闭：原16处 /11页均可编辑；当前 systemic defect / validator gap 均为0。既有39个 exposure 与本轮39个 input identity 变化取并集为45个 Unit；这不是45个语言错误，也不授权自动重新翻译。多标签 impact 中45个需要显式 representation identity mapping，既有真实语言 finding 仍只有1个；3个已观察到的 predecessor baseline 复用问题明确标为 unknown。exact-set 见 [FORMAL_IMPACT](PROTECTED_REPRESENTATION_FORMAL_IMPACT.json)，当前结论见 [SYS08_CLOSURE](PROTECTED_REPRESENTATION_SYS08_CLOSURE.md)。47个 soft-wrap Page 保持 translation-only-risk。历史阶段报告保留原结论。

## 十个设计问题的共同答案

1. 可见自然语言是 prose、link label、image alt、已批准的 visible attributes、metadata 和 teaching body；不能只因它位于 markup/code 附近就冻结。
2. machine identity 是 URL、reference key、code bytes/identifier、tag/attribute syntax、anchor、directive、模板控制与源坐标框架。
3. 上述 machine bytes 与每个 opaque token 必须 exact/unique，pair 必须保持正确嵌套与 slot membership。
4. 自然语言保持 editable；例外须明确列为不支持或缺陷，不能静默宣称可翻译。
5. inline link/HTML/emphasis 使用 source-bound pair；reference 额外具有文档级 shared identity。
6. 多 usage 不要求共享同一译法；机器共享的是 target identity。译法一致性仍由 Generation/Reviewer 判断。
7. 以下矩阵的“表达”说明当前 protected raw 能否合法表达自然语言变化；OPEN 项不能。
8. restore 仅使用 frozen source、明确的 token/binding 数据及输出位置，禁止猜 label、搜索相似译法或选择 latest candidate。
9. validator 检查语法、pair、目标与 exact-set，不判断中文如何翻译。tooltip 的残留问题也不能用自动翻译解决。
10. 无 `binding` 使用旧 literal reader；PB1 保留旧 full-reference expansion。PB2 为新 tooltip/shared-reference 语义，transport marker 同步 version；旧 accepted bytes 不重解释。未知版本 fail closed。unresolved bracket prose 仍记录 literal 类型，避免误命中 unused definition。

## Markdown 矩阵

数值主要为 Learn/Docs；Homepage 按其实际 HTML/template extractor 统计，不能将 `.md` 扩展名当成 Goldmark 表示。`S` 为 occurrence，`U` 为 source Page 数。完整 exact-set 位于机器审计同名 construct 下。

| 构造 | S / U | 可编辑 / exact protected | 绑定、表达与校验 | 当前 classification；实现、测试与风险 |
| --- | --- | --- | --- | --- |
| plain prose | 980 Paragraph slots / 38 | prose / block gaps | 整块语言；deterministic restore；禁止新结构 | SAFE_CURRENT（slot框架）；`markdownBlocksMode`、`validatePageSlotSyntax`；`TestReconstructTargetLexicalAtoms`。source 仅按共享 semantic-role policy 识别 machine identity；`TestSemanticRoleNaturalLanguage`、`TestSemanticRoleMachineControls` 和153 Page exact mutation audit 已覆盖 SYS-08 |
| inline code | 1214 / 35 | 周围 prose / 完整 span | atom 可整件移动；内部不能翻译 | SAFE_CURRENT；`protectBlockReferences`、`codeSpanEnd`；`TestStructuralBindingFamilies`。旧 lexer 对 code 中 bracket/backtick run 的误读已修复（SYS-04） |
| fenced code | 292 / 29 | 批准的 Go teaching bodies / 其余代码和 fence info | 显式非 Go fence 不获得 Go 语义；无 info 时沿用安全 Go 推断 | SAFE_CURRENT；`teachingSpansMode`；`TestTeachingFenceLanguageAuthority`。SYS-05 已修复；当前 289 个无 info、3 个 json，0 个显式 go/golang |
| indented code | 3 / 1 | 同上 | 同一 Go analyzer，无新状态机 | SAFE_CURRENT；`TestCompleteCodeTeachingProtection` |
| inline link | 615 个可编辑 label binding；含机器-only link 共 635 | label / delimiters、URL | pair + AST 检查目的地；允许整对移动，拒绝 label 注入导致解链 | SAFE_CURRENT；`StructuralBinding`、`markdownBindingShape`；`TestStructuralBindingCandidateBindingValidation`。VAL-01 已修复；不是仅检查 token 出现 |
| collapsed reference | 2 / 1 | usage 的可见 label / URL、opaque source binding | PB2 从首个 implicit usage 的可见 label 同步 definition 与所有引用；key 冲突或不能作为 reference key 时保留原 key 并输出 full reference，不约束译法。PB1 reader 不变 | SAFE_CURRENT；`TestStructuralBindingFamilies/collapsed`、`TestStructuralBindingFrozenFIPS140`、`TestReferenceLabelsMayShareTranslatedSpelling` |
| shortcut reference | 0 / 0 | 同上 | 同样支持确定性展开；6 个仅有 bracket 的源构造实际为未解析 literal，不能误报 shortcut | SAFE_CURRENT；`shortcut` 与 `literal` fixtures；原 lexer 支持 bracket，Goldmark 决定是否真是 reference |
| full reference | 0 / 0 | visible label / explicit ID | 每对 label 绑定 frozen shared ID | SAFE_CURRENT；`full` fixture |
| definition | 2 / 1 | key 不是显示文案；当前无 title / key、URL、framing | parser context 精确解析；不要求翻译隐藏 ID | SAFE_CURRENT；`markdownStructuralBinding`、`shared` fixture。definition 本身不生成平行生命周期 |
| multiple usages / one definition | 0 / 0 | 各 usage label 独立可译 / shared key | 机器保证所有 usage 指向同一个 frozen ID；可见文案允许语法变体 | SAFE_CURRENT；`shared` fixture；不要求模型手工复制 ID |
| unused definition | 0 / 0 | 无可见 prose / 完整定义 | 不因未使用而删除；restore 保留原字节 | SAFE_CURRENT；`unused` fixture |
| inline/reference image | 1 inline / 1；reference 0 | alt label / image marker、URL或ID | 与 link 共用 StructuralBinding；类型不能互换 | SAFE_CURRENT；`image` fixture |
| optional link/image/definition title | 0 / 0 | title prose editable / destination、delimiter exact | PB2 inline title interior、reference-definition title slot；Goldmark 验证 binding 与冻结 title 要求，拒绝 URL mutation、quote escape 注入及 definition 失效 | SAFE_CURRENT，SYS-07 FIXED；`TestMarkdownInlineTooltipRepresentation`、`TestMarkdownReferenceTooltipRepresentation`、`TestHistoricalTooltipAndPB1Readers`，含单双引号/括号/escaped quote/多行；无当前 affected Unit |
| emphasis / strong | 35 / 16；158 / 27 | 内部 prose / 源 delimiter 与 program child | 共用 inline slots；既有 emphasis AST binding | SAFE_CURRENT；`emphasis.go`、`TestMarkdownEmphasisLanguageProtection`、`TestStructuralBindingOtherRegisteredMarkdownFamilies` |
| soft break | 2250 AST breaks / 35；跨 HTML slot 的 wrap 合计2731 / 47 | 标点和 prose / newline、indent、quote prefix atom | wrap 在同一 slot 内可移动；中文标点可以放在正确一侧，不必成为独立一行 | TRANSLATION_ONLY_RISK；`TestStructuralBindingSoftBoundaryFreedom`、既有 `TestFormalPageBindingMovement`；47 个 exact source exposures 不是47个译文缺陷 |
| hard break | 0 / 0 | prose / required hard-break shape | block signature 检查 hard-break；positive fixture 保留两个空格和 LF | SAFE_CURRENT；`TestStructuralBindingOtherRegisteredMarkdownFamilies` |
| list direct text | 734 slots / 32；AST 594 items | item prose / list hierarchy | paragraph/TextBlock 属于 item；item slot 不等于 AST item 数 | SAFE_CURRENT；`markdownBlocksMode`、HTML mixed-list fixtures |
| blockquote | 4 / 3 | quote prose / prefix、hierarchy | source-bound prefix/wrap；新 block 注入拒绝 | SAFE_CURRENT；`TestStructuralBindingOtherRegisteredMarkdownFamilies` |
| Markdown table | 0 / 0 | cell prose / table shape | Table extension、TableCell slots；不把 HTML table 混算 | SAFE_CURRENT；同上 table fixture |
| headings/explicit anchor | 347 headings / 37；162 anchors / 21 | title prose / level、anchor | anchor 必须 suffix；不能移动到正文 | SAFE_CURRENT；`TestFormalPageCrossSlotAndSuffix` |
| escapes/entities/templates | 4 escapes / 2；14 entities / 8；6 slot actions / 2 | 周围 prose / atom raw | 标点可在 atom 外重排；模板内容不变 | SAFE_CURRENT；`protectBlockReferences`、structure-negative fixtures；实体编码与模板值不作语言猜测 |
| autolink、未知 directive/tag | 0 个 accepted autolink | 无新增可编辑协议 | 当前 bounded lexer fail closed；不因 Goldmark 有功能就宣布正式支持 | NOT_APPLICABLE；`TestMarkdownInlineHTMLAttributeProtectionBoundaries` 保留既有 rejection |

637 个 AST links 与 617 个可编辑-slot link bindings 的差额已逐路径核对：`querying.md` 的6个纯 code-label list links、`threat-model.md` 的10个 machine package-path list links 整体在 boundary 中；`security/fuzz/index.md` 的4个 link 在完整 opaque `<code>` atom 中。它们不是20个漏保护 label。615 inline + 2 collapsed = 617；另有1个 image binding。

## HTML 与 Site shell 矩阵

| 构造 | 当前 source occurrence | 可编辑 / exact protected、pair | classification；实现、测试与风险 |
| --- | --- | --- | --- |
| direct text + inline code/link | `slot/html/*`、`slot/html-block/*` exact runs 见 JSON；214 code、160 anchor tokens | prose 与完整 inline binding 同 slot；代码atom、tag/href保持 exact | SAFE_CURRENT；`htmlBlocksMode` 的 leaf + direct run 路径不只支持 li；`TestStructuralBindingHTMLAttributesAndBlocks` 覆盖 p/li/div/section/blockquote/dt/dd/td/th/figcaption/summary/button/label |
| list mixed content | 131 li / 10 Page | direct run、inner paragraph、nested list 分别保留；完整 inline child 不吞掉 prose | SAFE_CURRENT；历史 A 的7页有 repository evidence；`TestFormalPageInlineDirectLanguage` 有6种 child ×3种 nested shape。SYS-03 修复 accepted run 与 attribute 重叠 |
| nested inline HTML | a/strong/em/span 等 actual token counts 见 JSON | tag pair 嵌套；其间 prose editable；opaque child 仍为 atom | SAFE_CURRENT；`restoreBlock` pair stack、13 tag table fixtures |
| machine attributes | 同各 actual tag occurrence | href/src/id/class、引号、属性名/顺序/markup exact | SAFE_CURRENT；`validateInlineHTMLAttributeBindings` 与 mutation-negative fixtures |
| visible attributes | alt22 / 10；aria-label6 / 1；title3 / 1 | alt/title/aria-label prose editable；开闭引号及相邻属性 machine-bound | SAFE_CURRENT；Markdown 与 HTML 共用 `startTag`。SYS-02 已修复；当前 corpus 的已批准 visible slots 无额外 identity 变化。模板-only 属性没有固定 prose，不误报丢失 |
| opaque HTML/code regions | pre61 / 11；textarea1 / 1；script2 / 2；其他 token 见 JSON | 除批准 teaching body 外全部 exact | SAFE_CURRENT；`htmlProtectedRanges`、`TestFormalPageOpaqueHTMLLanguage`。不开放 script/style、任意 code string 或模板控制 |
| Homepage mixed prose | 源有3 mixed paragraphs / 1；旧1个 code boundary → 新0 | 新 Page 采用23个完整 inline runs，共用 HTML pair/atom 与 slot syntax；code/link、重复/nested elements 可随 prose 移动，标点可处于两侧 | SAFE_CURRENT，SYS-06 FIXED；`ExtractShellPageRepresentation`、`bindInlineShellTokens`、`TestHomepageInlineRepresentation`。冻结 extractor/old text-node reader 保留；跨 block anchor 继续使用 block framework；SVG/icon 保持已有 machine/hidden role |
| metadata | Learn/Docs title49、breadcrumbTitle1；Homepage title/summary各1 | 文案editable，keys/encoding/path/template flags exact | SAFE_CURRENT；两个 extractor 保持各自已注册 schema；没有借用 Markdown parser 解释 Homepage front matter |
| shared dynamic identity | Homepage 模板绑定及 dynamic-only attrs | action/field path exact；静态 prose editable | SAFE_CURRENT；既有 `TestShellFrozenTargetValidation`。structured source 的内容不扩展为语言审核 |

HTML tag count 是 tokenizer 经过 Markdown code/escape 排除、opaque region 排除后的 token 数，不是浏览器 DOM node 数；malformed-but-frozen optional-end HTML 保持原 bytes。不能用 raw grep 的伪 tag 替代这些 counts。新 fixtures 的未知 tag、quote injection、URL/attribute mutation 均拒绝。

## Code 与 Tour 矩阵

`code/*` exact lexical/policy counts 也进入机器审计。`go-safe` 来自实际 `analyzePreformattedGo`；`static-lexer` 只是 Go scanner 在 opaque code 区域找到的词法形状，**不证明该片段是 Go**，不能据它开放翻译。

| 构造 | 源范围与权限 | classification；实现、测试、false-positive/negative 边界 |
| --- | --- | --- |
| complete Go / safe fragment | declaration、statement、完整文件共用 analyzer；仅批准 teaching line body editable；identifier exact且可随语言语序整件移动 | SAFE_CURRENT；`SafeTeachingCommentBodies`、`protectTeachingBody`；`TestCompleteCodeTeachingProtection`、Unicode identifier test。不是只修4篇；当前Learn/Docs有141个body、21页，旧“约19页”不是当前exact count |
| line comment | natural teaching body可译；directive、annotation、code-like/non-natural comment exact | SAFE_CURRENT；classification tests；source-only eligibility，target不重新归类为 machine atom |
| block comment | 现阶段整个 containing snippet Static；不提供 block-comment 翻译槽 | NOT_APPLICABLE（语言开放）；code identity仍SAFE_CURRENT；`TestCompleteCodeUnsafeFragments`、`TestPreformattedStaticCommentsAndBlockCommentsCannotChange`。这是明确的 conservative feature boundary，不宣称支持block翻译 |
| quoted/raw strings | strings中的 `//`、`/*`、伪 Markdown 不产生 prose slot | SAFE_CURRENT；Go scanner而非regex；`TestPreformattedScannerExcludesCommentLikeLiterals`、code span family fixtures |
| shell/config/explicit non-Go fence | static code；即使payload碰巧parse为Go，也不能获得teaching语义 | SAFE_CURRENT；SYS-05修复；`TestTeachingFenceLanguageAuthority`；无locale特殊逻辑 |
| executable vs Static | 是代码表示策略，不是新TranslationUnit状态；Page内代码与独立Example生命周期不混用 | SAFE_CURRENT；preformatted tests。独立19个eligible Example不纳入本轮Page scope |
| Tour present prose/link/code/directive | 103 Page；338 inline code /77页；82links /27页；94directives /94页；25list items /6页 | SAFE_CURRENT；`protectTranslation`、`protectedTranslation.restore`、`validateCandidateWithGlossary`；103个实际canonical Page全部source restore + validator通过 |
| Tour link payload/pair、legacy program-font | label prose editable；link target及program payload绑定 | SAFE_CURRENT；`TestCompleteLinkStructuresMayReorderButCannotRebind`、`TestMoretypes18HistoricalResponsesRestoreNineInlineCodes`。不能把Markdown reference语法套用到present |
| UTF-8 trim / target CLI / raw comparison | rune trim；source lexical protection与target structure对称区分 | HISTORICAL_COMPATIBILITY_ONLY（缺陷已关闭）；`TestReconstructUTF8Trim`、`TestReconstructTargetLexicalAtoms`、`TestReconstructSourceProtectionStrict`、`TestReconstructStructureStrict`；没有重新修E |
| source semantic roles | 原16处 /11页误冻结已归零；完整 current atom inventory 见 JSON | SAFE_CURRENT，SYS-08 FIXED。结构/code/target 先由 parser 持有；prose 仅在明确 technical position、冻结标准库 namespace、格式/协议/技术 identity authority 中保护。uppercase、camel、dot、slash 本身不构成 identity。`semantic_role.go`、`semantic_go_packages.go`；positive16/16 + 真 machine negative controls + frozen11页 reader |

## 缺陷 class 与剩余 blockers

class 数量按独立根因计，不是 affected Page 数。历史A/B/E已关闭，不重复计入本轮 before。新增夹具证明的潜伏构造即使当前occurrence=0仍计一个class。

| class | before classification | 本轮结果 | 当前source受影响范围 |
| --- | --- | --- | --- |
| SYS-01 implicit reference label 被当作 binding ID | SYSTEMIC_REPRESENTATION_DEFECT | FIXED；显示label与opaque reference分离 | fips140，2 usages /1 Page |
| SYS-02 HTML inline visible attribute 被整体锁定 | SYSTEMIC_REPRESENTATION_DEFECT | FIXED；共用属性binding | 当前0；13 tag families夹具确认 |
| SYS-03 HTML run 与属性坐标重复产生 overlapping spans | SYSTEMIC_REPRESENTATION_DEFECT | FIXED；accepted runs也拥有属性坐标 | 当前0；非leaf div/section等夹具确认 |
| SYS-04 bracket/code-span delimiter词法边界不一致 | SYSTEMIC_REPRESENTATION_DEFECT | FIXED；equal-length delimiter run，bracket跳过code | 当前0；label内孤立`]`、长backtick、伪reference夹具 |
| SYS-05 explicit fence language 未约束 Go comment eligibility | SYSTEMIC_REPRESENTATION_DEFECT | FIXED；只对新解析执行语言约束 | 当前0；3个json fence当前均Static |
| SYS-06 Homepage text-node slots 固定inline code/link/标点 | SYSTEMIC_REPRESENTATION_DEFECT | FIXED | Homepage 1；3 mixed p采用完整run，code boundary=0；old reader保留 |
| SYS-07 Markdown tooltip title 与 URL一起冻结 | SYSTEMIC_REPRESENTATION_DEFECT | FIXED | 当前0；inline/definition delimiter、escape、多行、历史兼容 fixtures |
| SYS-08 source spelling heuristic把普通语言冻结为machine atom | SYSTEMIC_REPRESENTATION_DEFECT | FIXED | 原16处 /11页均可编辑；保留旧 inventory reader；exact mutation、technical negative controls 和 historical fixtures 见 SYS08 report |
| VAL-01 Markdown token唯一但不证明rendered link/image binding | VALIDATOR_GAP | FIXED；versioned per-pair AST + restored-document target multiset；literal不能被unused definition意外捕获 | 617 link +1 image +6 literal bindings；38 Page的表示identity变化 |

原轮 `SYSTEMIC_DEFECT_COUNT_BEFORE=8`，前轮8→3→1，本轮 SYS-08 入口1→0；validator gap仍为0。共享 semantic-role policy 已替代新 Page 的 spelling heuristic。当前源码支持边界之外的语法没有为凑测试扩充。

## 历史事件 A—E 复核

- A：实际历史报告与7页夹具一致；通用HTML direct run修复不是仅li规则。新增属性与run协调修复见SYS-02/03。Homepage使用另一extractor，其SYS-06仍单列。
- B：完整Go文件与片段共用analyzer；4个seed routes都在49页source projection内。当前141个Learn/Docs teaching slots/21页的coordinate coverage通过，strings/rawstrings/directive保持机器身份；非Go fence补上SYS-05。block comments依然明确不支持翻译。
- C：`qc-011`的4个exact findings与`qc-012`的4个A记录实际存在。当前Learn/Docs和新Homepage inline runs允许标点合法放在wrap/link/code两侧；47页仍为 translation-only source exposure。最终分类为 translation-only-risk，validator不替代语言Reviewer。
- D：canonical fips140 source SHA仍为 `c171065cde5e7a0a9597d89e59bb9e8cb3e614ff096e032b3580ebf2709902e4`；2 definitions、2 collapsed usages；不是只对CMVP写regex。
- E：UTF-8 trim、target-side `CLI/API/HTTP/CPU/URL`、source protection及structure negative fixtures实际通过，未重做已关闭修复。SYS-08 现由共享 role policy、16个 exact source contexts、technical negative controls、11页历史 reader 与153 Page mutation audit 覆盖；classification regression coverage complete=yes。
