# Protected Page representation 审计与工程交接

> 当前 SYS-08 工程结果见 [PROTECTED_REPRESENTATION_SYS08_CLOSURE.md](PROTECTED_REPRESENTATION_SYS08_CLOSURE.md)：systemic 1→0、false-machine 16→0，153 Page audit PASS。下面是原8→3阶段的历史记录，保留当时 FAILED 结论；不代表当前状态。046 与所有 formal artifacts 未修改。

整体结果 **FAILED**。已完成 registered canonical corpus inventory、taxonomy、5类共享 representation 修复与1类 validator 修复，但还有3类 representation blocker；046 的 QC-B replacement authority 也不能由现有 Integrated Surface successor 直接替代。不能宣称 remaining systemic defects=NONE。

详细结构矩阵见 [PROTECTED_REPRESENTATION_MATRIX.md](PROTECTED_REPRESENTATION_MATRIX.md)，机器 exact-set 见 [PROTECTED_REPRESENTATION_CORPUS_AUDIT.json](PROTECTED_REPRESENTATION_CORPUS_AUDIT.json)。这些是工程诊断，未建立任何正式语言 evidence 或授权。

## 冻结范围与结果

```text
STAGE=protected-representation-systemic-audit-and-fix
HISTORICAL_INCIDENT_COUNT=5
HTML_INLINE_BOUNDARY_FIX_COMPLETE=yes
FENCED_CODE_COMMENT_FIX_COMPLETE=yes
SOFT_BOUNDARY_PUNCTUATION_SYSTEMIC_RISK=yes
REFERENCE_LABEL_BINDING_FIX_REQUIRED=yes
CLASSIFICATION_REGRESSION_COVERAGE_COMPLETE=no
REGISTERED_PAGE_REPRESENTATION_COUNT=3
TOTAL_REGISTERED_PAGE_UNITS=153
SYSTEMIC_DEFECT_COUNT_BEFORE=8
SYSTEMIC_DEFECT_COUNT_AFTER=3
VALIDATOR_GAP_COUNT_BEFORE=1
VALIDATOR_GAP_COUNT_AFTER=0
OLD_REPRESENTATION_COMPATIBILITY=PASS
PROTECTED_MACHINE_IDENTITY=PASS
TRANSLATABLE_NATURAL_LANGUAGE=FAILED
PAIR_BINDING_MACHINE_ENFORCED=PASS
FIPS140_REPRESENTATION=FAILED
FIPS140_REFERENCE_BINDING=PASS
BATCH_046_LIFECYCLE_BRANCH=representation-reconciliation
BATCH_046_MUTATED=no
FORMAL_ARTIFACT_MUTATION=none
CODE_MUTATION=known
STOP=before-formal-revision-reexport-or-successor
```

HTML / fenced completeness 的 yes 限于矩阵明确的已批准支持面；block comments 不宣称支持翻译。soft-boundary risk 的 yes 来自 Homepage 的 SYS-06；Learn/Docs 历史 C 的4页局部修订与4个A不被改写。reference-required=yes 表示本轮确实需要并实施了共享修复。fips140 的reference夹具通过，但该页的普通admonition `NOTE`仍被source spelling heuristic冻结，因此整页representation不能PASS。旧E的三类regressions通过不等于source classification完备。

current locale=`zh-CN`，Snapshot=`learn-docs-v1-qc-014`，review=`20261015-zh-CN-learn-docs-qc-014-re-qc-page-001`，rating=`B`，batch=`chatgpt-zh-CN-046`；仍为0/1 output，未import/process、无replacement、无successor Snapshot、无QC/Surface record、无closure/activation/Production/commit。

38个Learn/Docs Page的protected表示identity及input bytes发生变化，39个machine exposure是这38页加未修复的Homepage；不是39个已证明的译文缺陷。新旧ID、route、source SHA在JSON的`representation_identity_changes`，exact 39-set在`affected_current_units`。当前确认的source construct blockers为9页：Homepage以及SYS-08的8页（含fips140），exact IDs/routes在`confirmed_source_defect_units_after`。tooltip缺陷由夹具证明，当前source occurrence=0。47页soft-wrap translation-only exposures在`potential_translation_only_risks`，没有自动reopen任何TU。

SYS-08 的11处确定性证据为普通自然语言 `e.g`、`i.e`、admonition `NOTE`/`IMPORTANT`，完整source slot、token、raw与ID/routes在`source_natural_language_frozen_as_atom`。`source_lexical_atoms`同时保留958个ASCII word-shape atom候选（含合法machine identifiers）；这是可检查的候选池，不是958个语言缺陷或全域ownership证明。`PUBLIC/PRIVATE/URGENT`等named security tracks、`U.S`/`FAQ`等其它语境不能仅靠uppercase/dot规则自动作keep或解锁决定。没有为消除11处问题写locale/词语特判，也没有取消CLI/API/URL/code identity保护；source machine ownership的通用分类仍需工程收口。

## 共享实现与兼容

`ProtectedToken.Binding` 分离显示语言与机器绑定；新 formal Page transport marker携带binding version，并进入原有SourceSHA/Unit ID/input SHA计算。新reader对Markdown link/image/literal逐pair解析，restore后比较全页destination/type/title multiset，拒绝label破坏binding、误命中unused definition、URL变化及新增/消失链接。完整pair仍可移动；token exact/unique与slot membership没有放宽。

collapsed/shortcut label改变时，机器确定性输出full-reference spelling，保留原definition key和URL；不翻译隐藏ID，不猜label，不要求模型同步多处identity。模型仍决定所有开放prose。使用相同shared pair机制处理inline、full-reference和image；没有新建任何TranslationUnit生命周期。

HTML startTag的quoted visible attributes在HTML/Markdown中使用同一binding；accepted direct run也拥有其属性坐标，避免重复spans。code-span lexer按完整同长度backtick run匹配，bracket parser跳过code内容。新source解析的teaching eligibility遵守显式fence language；历史persisted slot验证沿用旧eligibility，不把新语言策略倒灌到旧raw。Page/Shell reattestation及未来shell parser-reference builder显式包含新增shared binding文件；没有修改任何已冻结reference。

旧reader：frozen `binding`缺省时，完全按原raw/role/pair还原。旧JSON序列化不增加字段，旧FormalPage重组不插入PB marker。canonical English bytes、raw responses、candidate、Snapshot、finalization、quality evidence、reopen receipt均未改写。

新reader：只有versioned protected binding启用新语义，未知version拒绝。source本身roundtrip字节不变；label变化后可能采用等价full-reference syntax，精确URL/ID继续由机器保护。

export/currentness：Unit ID与protected input bytes真实改变，使旧manifest/Bundle自然stale；不能仅刷新transport authority。retry不能把原immutable batch换成新版token set；必须走已证明的representation replacement authorization。现有Bundle currentness、input SHA/token count、source identity检查保留。

revision predecessor：现有source-coordinate `RebaseRevisionPredecessor`将旧raw映射到新token，只接受byte-exact reviewed candidate。实测49个qc-014原raw/candidate全通过，38个changed Page全部rebase后仍byte-exact，且38个input均有真实byte drift。没有从restored candidate猜token，也没有latest candidate shortcut。

历史validation：读取Snapshot显式selected batch、validation SHA、attempt、RawResponsePath与candidate SHA；只验证原bytes如何解码，不保存revalidation receipt，不把历史PASS升级为current授权。独立Reviewer职责保持不变。

冻结的pre-change diagnostic fixture为 `internal/sitecontent/testdata/protected-page-before.json.gz`；仅含canonical representation，不含新译文或正式receipt。gzip使用无时间戳形式，SHA-256=`3338b21ae9bd201a7ccc9ba321b934b164cbeb6e5f2a20a922015556c6e64395`。普通历史reader test可直接消费该fixture；本轮额外用worktree中的真实qc-014 raw/candidate完成49/49验证。

## 046 的 authority 判断

[Task Spec](TRANSLATION_TASK_SPEC.md)明确：无output仅允许在原manifest/authorization不变时同批刷新；representation产生replacement Unit ID时例外，必须使用[Page replacement lineage](TRANSLATION_UNIT_REPLACEMENT.md)。因此046不能same-batch-refresh，也不能跳过reconciliation直接新建revision batch。

还有一个真实bridge缺口：`internal/i18n/translation_unit_replacement.go` 的replacement successor要求`IntegratedAuthority`、`IntegratedSurface`及相同旧finding。现有 `20261015-zh-CN-preview-fips140-reopen-001`来自Preview，绑定qc-012的missing-link finding；046则绑定qc-014新的QC-B localization finding。不能拿原Preview finding替代当前B，也不能伪造Integrated Surface receipt或已finalized的qc-014。

本轮只判定正确branch和保存工程诊断，没有落盘任何successor authorization。38个identity变化的scope需要明确区分已通过rebase的机器兼容性与未来正式QC/授权；不让通用修复隐式reopen38页。

代码变更后的真实authority failure仍为：

```text
structured evidence hash drift: internal/sitecontent/formal_page.go
```

这是正式reader正确拒绝stale execution authority。本轮after scan使用变更前实际注册的完整inventory，并逐文件核对canonical SHA；不是修改/绕过authority reader。没有宣称real Bundle CURRENT或正式工作流可恢复。普通shell的Go cache曾报告read-only filesystem；对相同targeted命令按sandbox规则重跑后通过，不是未解决的代码测试失败。

## Targeted tests 与 focused timing

已执行以下定向测试；全部PASS。所有synthetic export/process/record夹具均在`t.TempDir()`，没有对真实locale执行这些动作。

- `internal/sitecontent`：`TestRegisteredPageRepresentationAudit`；`TestStructuralBindingFamilies`（inline/collapsed/shortcut/full/shared/unused/image/literal/pseudo-code/code-bracket/backtick）；`TestStructuralBindingHTMLAttributesAndBlocks`；`TestStructuralBindingSoftBoundaryFreedom`；`TestStructuralBindingOtherRegisteredMarkdownFamilies`；`TestStructuralBindingHistoricalLiteralReader`；`TestStructuralBindingFrozenFIPS140`；`TestStructuralBindingHistoricalCorpus`；`TestStructuralBindingStandaloneReferenceRoundtrip`；`TestStructuralBindingCandidateBindingValidation`。
- 同package既有focused regressions：`TestCompleteCodeTeachingProtection`、`TestCompleteCodeUnsafeFragments`、`TestCompleteCodeUnicodeIdentifierBinding`、`TestFormalPageBindingMovement`、`TestFormalPageCrossSlotAndSuffix`、`TestMarkdownEmphasisLanguageProtection`、`TestMarkdownInlineHTMLVisibleAttributes`、`TestMarkdownInlineHTMLAttributeInjectionRejected`、`TestMarkdownInlineHTMLAttributeProtectionBoundaries`、`TestFormalPageInlineDirectLanguage`、`TestFormalPageOpaqueHTMLLanguage`、`TestReconstructUTF8Trim`、`TestReconstructTargetLexicalAtoms`、`TestReconstructSourceProtectionStrict`、`TestReconstructStructureStrict`。
- 新增边界诊断：`TestTeachingFenceLanguageAuthority`、`TestTeachingFenceHistoricalEligibility`、`TestStructuralBindingAdapterAuthorityDependency`；最后一个测试证明shared binding文件变化会使两个adapter authority都变化。`TestRepresentationKnownBlockers`通过表示三类OPEN事实仍成立，**不表示已支持该功能**。
- `internal/i18n`：`TestTourPageRepresentationAudit`（103个canonical Page source restore + structure validator）；`TestInitialGenerationBundleDoesNotRequireCandidates`；`TestRevisionGenerationBundleOrdinaryRatings`（B/C/D）；`TestPreviewRevisionGenerationBundleInfersFrozenPackageScope`；`TestRevisionGenerationBundleProtectedOutputAndRetry`（restore_failed/validation_failed）；`TestPreformattedTeachingCommentClassification`；`TestPreformattedScannerExcludesCommentLikeLiterals`；`TestUnrecognizedPreformattedBlockIsStatic`；`TestPreformattedStaticCommentsAndBlockCommentsCannotChange`；`TestMoretypes18HistoricalResponsesRestoreNineInlineCodes`；`TestCompleteLinkStructuresMayReorderButCannotRebind`。

上述lifecycle adapter contract测试与新Page family fixtures分层组合；未为initial/revision/reopen/retry复制parser或状态机，也没有以测试名冒充真实046执行。没有运行package-wide suite、`go test ./...`、browser、build、Production或deployment。

扫描命令（diagnostic outputs；不会写正式locale artifacts）：

```sh
GO_TOUR_REPRESENTATION_BEFORE=/tmp/go-tour-representation-before.json \
GO_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-representation-after.json \
go test -mod=readonly ./internal/sitecontent \
  -run '^(TestRegisteredPageRepresentationAudit|TestStructuralBindingHistoricalCorpus)$' -count=1 -v

GO_TOUR_PAGE_CODE_AUDIT_INPUT=/tmp/go-tour-representation-after.json \
GO_TOUR_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-tour-representation.json \
go test -mod=readonly ./internal/i18n -run '^TestTourPageRepresentationAudit$' -count=1 -v

go test -mod=readonly ./internal/sitecontent -run '^$' \
  -bench '^BenchmarkStructuralBindingFIPS140Restore$' -benchtime=100x -count=1
```

`/tmp/go-tour-representation-before.json`是本轮pre-change scan保存的exact inventory，不得以当前parser重建后冒充before；gzip fixture保留同一完整数据。after投影153页约1.24s，Tour及code词法补充约0.20s。最终100次单Page benchmark：old `11789368 ns/op`、`6990554 B/op`、`22830 allocs/op`；new `15064017 ns/op`、`9772402 B/op`、`43151 allocs/op`。增加约3.275ms/restore（27.8%）与2.782MB临时allocation；该Page仍为约15ms，不是分钟级workflow。结果保存于 `/tmp/go-tour-representation-timing.txt`；没有做repository performance audit。

本轮 `git diff --check`：PASS。完整worktree的stat/status包含大量进入本轮前的WIP，分别保存于 `/tmp/go-tour-representation-final-diff-stat.txt`、`/tmp/go-tour-representation-final-status.txt`；不能当作本轮独占diff。新增文件不因`git diff --stat`不显示untracked而漏报，下节列出了实际16个文件。

## 本轮实际改动文件

既有WIP中的8个文件仅追加本轮改动：`internal/sitecontent/formal_page.go`、`linguistic.go`、`units.go`、`page_structure.go`、`teaching.go`、`translation_adapter.go`、`shell_reconstruction.go`、`shell_successor.go`。

新增8个文件：`internal/sitecontent/structural_binding.go`、`internal/sitecontent/structural_binding_test.go`、`internal/sitecontent/representation_audit_test.go`、`internal/i18n/representation_audit_test.go`、`internal/sitecontent/testdata/protected-page-before.json.gz`、本报告、结构矩阵及机器审计JSON。没有git add/reset/clean/stash/checkout/commit。

唯一下一步执行者是Codex工程：保持046和formal artifacts冻结，先收口SYS-06/SYS-07/SYS-08及QC-B representation-reconciliation的合法bridge设计/实现；其后才能交维护者执行正式authority reconciliation / revision reexport。停止点仍在任何formal successor/reexport之前。没有将真实问题标记为用户已同意defer。
