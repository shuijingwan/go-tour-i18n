# Protected representation closure 工程结果

> 本文件保留上一轮阶段结论。当前 SYS-08 已关闭，见 [PROTECTED_REPRESENTATION_SYS08_CLOSURE.md](PROTECTED_REPRESENTATION_SYS08_CLOSURE.md)；不得将以下历史 FAILED 当成当前结论。

本轮 **FAILED**，不能交付正式 reconciliation。SYS-06/07 已实现并通过当前 corpus 与定向 fixtures；SYS-08 没有完成共享 semantic-authority policy 替换。除了原11处（8页），读取既有 candidate pool 的 exact contexts 又确认3个 `and/or`、1个 `U.S` 和1个 `FAQ` 自然语言冻结，共16处、11页；这不是新增回归，而是既有 source policy 的进一步证据。Surface replacement regression 的 fixture 问题已修复，正向综合测试通过。没有以局部 PASS 代替整体闭合。

本轮沿用已有 taxonomy 和 deterministic audit framework，未进行 repository-wide discovery。入口仍是 [矩阵](PROTECTED_REPRESENTATION_MATRIX.md)、[153 Page corpus audit](PROTECTED_REPRESENTATION_CORPUS_AUDIT.json) 和 [39 Unit impact exact-set](PROTECTED_REPRESENTATION_FORMAL_IMPACT.json)。

## 实际工程变化

Homepage 新 Page 表示将完整 inline runs 合并为语言 slot，共用现有 HTML protector、pair stack、属性 binding 和 syntax validator。当前 Homepage 有23个完整 runs，旧 inline-code boundary 从1降到0，旧3个 mixed paragraphs 不再有固定 text-node slots。code、link、重复/nested inline、前后中文标点及语序移动有正向 fixtures；带不同 URL/title 的两个 link 整体交换，也能 deterministic restore/validate。SVG/icon 在新 inline-run role 中保留原 machine/hidden 意义；跨 block 的 anchor 留在 block framework。没有重设计已关闭的 Learn/Docs HTML/teaching 路径。冻结的 `ExtractShellLanguageSource` 与旧 text-node adapter reader 保留，新 Page 使用 `ExtractShellPageRepresentation`。

PB2 将 Markdown inline tooltip 内文与 URL/framing 分开，reference-definition title 使用同一 Page 的语义 slot。单双引号、括号、escaped quote、多行 title、无 title、URL mutation、delimiter injection 都有 fixtures。冻结的旧 title 仍按旧 token raw 读取；混合 reader 保留非 editable title 的约束。机器不提供 title 译文。

PB2 reference 具有文档级 definition source coordinates。模型选择 usage 可见 label，机器按首个 source-owned implicit usage 同步 definition 和共享 usages。正常情况可以保留 collapsed/shortcut spelling；不同目的地使用同一个中文 label，或 label 不能作为 reference key 时，机器使用各自原 key 的 full reference，不要求模型改变译法。该行为基于明确 source binding 与 parser-approved definition coordinates，不使用相似 label 猜测。

fips140 夹具已输出 `[CMVP 验证证书][]` 及匹配 `[CMVP 验证证书]: <原 URL>`，destination shape 不变。这里只证明文本 transport；“验证证书”是 fixture 的输入，未生成真实 replacement。整页仍因 SYS-08 的 `NOTE` 冻结而不能报告 representation PASS。

SYS-08 的实现仍位于 `linguistic.go:protectBlockReferences` 的 source lexical 分支：uppercase、dot/slash/underscore、camel spelling 会设置 `tech`。它没有验证 semantic role。仅解锁4个字面词、按 locale 特判，或整体释放候选 atom，都不能证明 API/flag/package/enum identity 正确，因此本轮没有实施这种替代。16处 exact contexts 和 technical candidate pool 继续保留在 audit JSON；没有宣称分类审计完整，也没有新增语言 finding。原词表11处不是全域完备证明。仍需要把 plain-prose guessing 替换为结构/明确 technical identity authority；本轮没有完成这一实现。

soft-boundary 最终分类是 **TRANSLATION_ONLY_RISK**：47个 exact source Page 的 prose/标点可合法位于 soft-wrap/code/link 两侧，inline run fixtures 允许自然移动；模型仍可能选择不自然 placement。validator 检查 structure/binding，不判断语序、标点或译法。这不是新增 systemic defect，历史4页 local A 不被改写。

## QC-B bridge 与046

现有 replacement reopen 增加 `authority_source=quality-check` 及 `replacement_context.quality_check`；没有第二套 state machine。原 Snapshot manifest、QC results、candidate、validation、input、rating、finding 和工程必要性 proof 都有 exact binding；要求完整初审、真实 successor input drift 和 exact scope。Surface、Preview、旧 reopen/finalization authority 不能混入。

临时 fixture 已完成 B → reconciliation authorization → successor revision protected baseline → import/process → new Snapshot pending → independent re-QC；旧 evidence bytes 保持不变。这里的所有正式形态操作均在 `t.TempDir()`，未触及真实 locale。错 rating/Snapshot/Unit/finding、混合 Surface authority 被拒绝。普通 B/C/D revision、Preview reopen/retry 和 legacy Surface negative controls 的定向测试通过。

39个 exposure 的 primary disposition 为：1 reconciliation required（fips140）、37 byte/semantically unaffected、1 unknown（Homepage 的实际 accepted-candidate/formal migration 未证明）；reattest-only 为0。`true_language_defect=1` 是正交的既有 QC-B evidence flag，与 fips140 的 reconciliation 集合重叠，不是第40个 Unit。37个 candidate 的 byte-exact 结论不授权改 ID、carry A、reopen、Generation 或 formal reattestation。

046 manifest SHA=`a7d005c29e630e58fdbdc0c7d6de9f0e2f5c2c33f1f4e39b51cb4b4a2143df95`；manifest/input 未改，无 raw/import/process。合法方向保持：046 immutable preserved → 明确 representation reconciliation → successor Unit representation → 消费**同一个 qc-014 B finding**的新 revision batch → validation + independent re-QC。不是 same-batch-refresh。

正式命令当前为 `NONE`。SYS-08 和 classification completeness 未收口，不能交维护者执行 reconciliation，也没有生成真实工程 proof/历史表示正式授权输入。

## structured evidence drift

分类为 **over-broad-dependency**。错误前缀来自通用 `readRegularReference`；实际 drifted evidence 是 shell source foundation：

`data/site-shell/successor/sources/30d7c2ef242aa2aa13e64aa5a8917baca377710cb0ef77a68dba3cfb27694272.json`

经由 adoption `data/site-shell/successor/adoptions/2ec50f613ab4b48abf1dd58078b47c89626bae8d6050d24944aa389bbac258c6.json` 读取，并非已经证明 structured Generation 输出语义变化。原依赖为 `internal/sitecontent/formal_page.go` 整文件 SHA=`1847269422e692248ef2556db07aeab05b4d46dcae6bfe41c3ceebf3a303067c`；当前文件 SHA=`5109232b1ee3765fa4f9ea6686d72ced263e2a321ff60dd0387d33ab0838113a`。

冻结5个 shell sources 的 SourceSHA、完整 slots、role/context/encoding、protected raw/role/pair、StructureSHA 与 frozen Page 经原 extractor 重建仍精确一致。真正被该 foundation 消费的是这组有限 canonical inputs 的 extraction projection；新 PB2/inline Page reader 没有改变它们的旧解释。依赖已改为可重建的 `ShellExtractionContract + Sources` semantic projection，SHA=`2ecc16b133a9059469a3f5b1cd8735ed2f91808d1d045073ec06d128fc73457c`。新 foundation 可以显式写 additive `parser_semantic_identity_sha256`；旧 foundation 不改写，reader 重算其完整冻结 projection。

只有明确列出的 source extractor implementation dependencies 使用这个边界；module/library 及未知依赖仍检查原文件 SHA。source/slot/protection output drift 拒绝，单纯 implementation comment edit 不再伪 stale。current TU validator/reconstruction 仍由 adapter authority 独立 hash，新增 helper 已纳入。旧 evidence 的 identity/SHA/provenance 和所有 language artifacts 未刷新。无需 formal recovery/re-attestation/regeneration 来修复这一个过宽 dependency gate；真实新 Page 表示的 formal currentness 仍须独立处理。

## 兼容、测试与真实失败

最终 corpus projection：153/153，约1.67s；49 historical selected raw/candidate 精确复现，38个 changed Learn/Docs baseline byte-exact rebase（约7.04s）。新增 old fixtures 包括 Homepage text-node、literal inline/reference tooltip 和 PB1 reference。无 metadata 的旧 reader、PB1 full-reference reader 与旧 accepted bytes 均未重解释。gzip fixture 未修改。

实际 current registered reader（不用 before projection）也通过153页，约4.98s；Tour/code补充约0.28s。最终独立100次 fips140 restore benchmark：old `11620583 ns/op, 6990558 B/op, 22830 allocs/op`；new `24341717 ns/op, 13747173 B/op, 57598 allocs/op`。新增binding/文档级同步约增加12.72ms/次，未达到分钟级；没有扩大性能审计。日志为 `/tmp/go-tour-representation-closure-timing.log`。

定向测试清单和精确命令见本报告末尾命令块及测试文件；没有运行 `go test ./...`、browser 或实际 lifecycle。最终 stdout 保存在 `/tmp/go-tour-representation-closure-corpus-test.log`、`/tmp/go-tour-representation-closure-focused-test.log`、`/tmp/go-tour-representation-closure-registered-test.log`。Go cache 只读导致的 sandbox setup failure 对原命令重跑后消除，不算代码 regression。

Surface 综合测试首次在前置 current-authorization gate 拒绝 `affected=0 stale=6 unknown=0`，隔离时还出现 predecessor 缺 identity-matching A。根因是旧 fixture 没有落盘 synthetic physical source，且直接消费已扩大的真实 corpus authority；current source-file gate 因此正确拒绝。fixture 改用已有 fresh-authority 模式并写入6份 synthetic canonical source 后，原 `TestSurfaceReplacementMixedAggregateRevisionAndRecovery` 全部通过（约34.16s），包括 Integrated Surface、replacement/recovery、独立 re-QC 和原 package closure assertions。没有放宽 runtime gate，没有改真实 authority。临时隔离 wrapper 已移除。

必要的下一步仍是 Codex 工程：完成 SYS-08 的 semantic-role machine policy，之后重新运行本范围定向 closure；本轮停止于任何 formal reconciliation 之前。

## 改动文件与验证入口

本轮仅改动下列文件；其余既有139-file WIP未整理。`git diff --stat` 含进入本轮前的修改，不能视为本轮独占统计。

共34个本轮改动文件，机器完整路径数组也在 audit JSON 的 `files_changed`。全 worktree 的 tracked diff 是 `139 files changed, 72277 insertions(+), 16152 deletions(-)`；完整 stat/status 分别保存于 `/tmp/go-tour-representation-closure-diff-stat.txt`、`/tmp/go-tour-representation-closure-status.txt`（528行，含大量既有 untracked WIP）。`git diff --check` 通过；未 add/commit/reset/clean/stash/checkout。

- `cmd/tour-i18n/surface_reopen.go`、`qc_replacement_options_test.go`
- `internal/i18n/quality_check_reopen.go`、`quality_check_replacement.go`、`translation_unit_replacement.go`、`preview_failure.go`、`retranslation_export.go`、`generation_revision_inputs.go`、`historical_finalization.go`
- `internal/sitecontent/linguistic.go`、`structural_binding.go`、`markdown_references.go`、`units.go`、`formal_page.go`、`page_structure.go`、`shell_inline.go`、`shell_reconstruction.go`、`shell_adoption.go`、`shell_successor.go`、`shell_parser_dependency.go`、`translation_adapter.go`、`page_replacement.go`
- `internal/sitecontent/structural_binding_test.go`、`representation_audit_test.go`、`representation_closure_test.go`、`quality_check_replacement_test.go`、`shell_parser_dependency_test.go`、`page_replacement_test.go`
- `docs/TRANSLATION_UNIT_REPLACEMENT.md`、`PROTECTED_REPRESENTATION_MATRIX.md`、`PROTECTED_REPRESENTATION_CORPUS_AUDIT.json`、`PROTECTED_REPRESENTATION_FORMAL_IMPACT.json`、`PROTECTED_REPRESENTATION_AUDIT.md`、本报告。

```sh
GO_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-representation-registered-current-final.json \
go test -mod=readonly ./internal/sitecontent -run '^TestRegisteredPageRepresentationAudit$' -count=1 -v

GO_TOUR_REPRESENTATION_BEFORE=/tmp/go-tour-representation-before.json \
GO_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-representation-closure.json \
go test -mod=readonly ./internal/sitecontent \
  -run '^(TestRegisteredPageRepresentationAudit|TestStructuralBindingHistoricalCorpus)$' -count=1 -v

GO_TOUR_PAGE_CODE_AUDIT_INPUT=/tmp/go-tour-representation-closure.json \
GO_TOUR_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-tour-representation-closure.json \
go test -mod=readonly ./internal/i18n -run '^TestTourPageRepresentationAudit$' -count=1 -v

go test -mod=readonly ./internal/sitecontent -run '^$' \
  -bench '^BenchmarkStructuralBindingFIPS140Restore$' -benchtime=100x -count=1
git diff --check
```

其它已执行的精确定向命令（PASS；综合 Surface fixture 修复后的原测试也已 PASS）：

```sh
go test -mod=readonly ./internal/sitecontent -run '^(TestStructuralBinding.*|TestHomepageInlineRepresentation|TestMarkdown.*TooltipRepresentation|TestHistoricalTooltipAndPB1Readers|TestRepresentationKnownBlockers|TestTeachingFenceLanguageAuthority|TestTeachingFenceHistoricalEligibility|TestCompleteCodeTeachingProtection|TestCompleteCodeUnsafeFragments|TestCompleteCodeUnicodeIdentifierBinding|TestFormalPageBindingMovement|TestFormalPageCrossSlotAndSuffix|TestMarkdownEmphasisLanguageProtection|TestMarkdownInlineHTMLVisibleAttributes|TestMarkdownInlineHTMLAttributeInjectionRejected|TestMarkdownInlineHTMLAttributeProtectionBoundaries|TestFormalPageInlineDirectLanguage|TestFormalPageOpaqueHTMLLanguage|TestReconstructUTF8Trim|TestReconstructTargetLexicalAtoms|TestReconstructSourceProtectionStrict|TestReconstructStructureStrict|TestQualityCheckRepresentationReconciliation|TestShellParserSemanticDependency|TestReconcilePageReplacementFailsClosed|TestFrozenPageReplacementReconciliation)$' -count=1

go test -mod=readonly ./internal/sitecontent -run '^(TestHomepageInlineRepresentation|TestMarkdown.*TooltipRepresentation|TestReferenceLabelsMayShareTranslatedSpelling|TestHistoricalTooltipAndPB1Readers|TestStructuralBindingFrozenFIPS140|TestQualityCheckRepresentationReconciliation|TestShellParserSemanticDependency)$' -count=1

go test -mod=readonly ./internal/sitecontent -run '^TestSurfaceReplacementMixedAggregateRevisionAndRecovery$' -count=1

go test -mod=readonly ./internal/i18n -run '^(TestInitialGenerationBundleDoesNotRequireCandidates|TestRevisionGenerationBundleOrdinaryRatings|TestPreviewRevisionGenerationBundleInfersFrozenPackageScope|TestRevisionGenerationBundleProtectedOutputAndRetry|TestPreviewFailureSharedReopenRevisionAndIndependentQC|TestPreviewFailureSharedFailClosed|TestSurfaceReplacementRejectsUnboundClaims|TestPreformattedTeachingCommentClassification|TestPreformattedScannerExcludesCommentLikeLiterals|TestUnrecognizedPreformattedBlockIsStatic|TestPreformattedStaticCommentsAndBlockCommentsCannotChange|TestMoretypes18HistoricalResponsesRestoreNineInlineCodes|TestCompleteLinkStructuresMayReorderButCannotRebind)$' -count=1

go test -mod=readonly ./cmd/tour-i18n -run '^(TestParseQualityCheckReplacementOptions|TestParseQualityCheckSurfaceReopenOptions|TestParsePreviewFailureReopenOptions)$' -count=1
```

`TestRepresentationKnownBlockers` 当前通过表示 SYS-08 的原11个 seed 仍冻结，以及旧 Homepage reader保留旧表示；不是 classification closure 的 PASS。额外5处由已有 corpus候选池的完整 source contexts 确认，均进入更新后的 JSON exact-set。
