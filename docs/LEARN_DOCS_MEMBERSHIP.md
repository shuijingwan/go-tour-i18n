# Learn/Docs current membership 与历史组合

`learn-docs-v1` 是统一 current Learn/Docs package，包含51个 Page TU及既有7份 structured Learn data。原49页的 stable index完全保留；`/doc/`、`/doc/install` 依正式 source adoption依次追加为50、51。所有新 locale只执行这一套共享 TU lifecycle，initial Generation及 initial QC各为30+21两次独立 invocation。max60不允许合并这两组。Tour Page/Example、status与 Course SEO仍按各自原 scope工作。

`docs-entry-v1` 只保留为历史 migration identity，不再出现在 current package descriptor、initial exporter、Snapshot、QC、finalization、closure或 Surface的独立调度中。原 adoption、representation successor、失败零输出 batch、Generation/QC及 finalization全部原样保留；不得按 route寻找替代 candidate，也不因 package grouping改变而重做语言审核。

## Source membership authority

`data/content-package-memberships/current.json` 引用 immutable、content-addressed `go-learning/package-membership-successor/v1` receipt。它绑定有序的完整 predecessor package identity、有序 Page source path/SHA、原 registry及新 registry archive。current `data/learn-docs-pages.json` 必须与新 archive逐字节同 SHA；append-only prefix、source exact set、重复/冲突/缺页/多页均机器校验。registry source grouping不重写原 Content Scope、source adoption或 unified corpus的 contributor identity，不制造新 glossary epoch。source/corpus投影仍消费原 source authority；current lifecycle/coverage投影消费 membership successor。

current catalog仅公开合并后的 descriptor。历史 descriptor由独立 historical reader显式选择，保留 original authority/order/adapter；它不能成为 future scheduling入口。相同历史 descriptor在嵌套读取中仍是一个 authority，真正的不同或歧义 descriptor继续拒绝。

## 不新增 A 的历史 finalization composition

已有 locale若两个 disjoint predecessor均已 finalized A，使用只读命令：

```sh
go run -mod=readonly ./cmd/tour-i18n site-content package-composition \
  --locale '<locale>' --package learn-docs-v1 \
  --page-finalization '<first-predecessor-finalization.json>' \
  --page-finalization '<second-predecessor-finalization.json>'
```

顺序、package identity和 Page全集来自 membership receipt；不是任意拼接 finalization。结果是 readonly closure input proof，`mutation=none`，不创建 Snapshot、rating、finalization、current authorization或 closure。

每个 predecessor分别执行完整 historical integrity验证和当前 re-attestation，包含 source、原 Unit identity、candidate/validation、batch/attempt/raw/provenance、原独立 A/finding lineage、current glossary/full Unified Review、当前 adapter/protector/validator及 latest candidate选择。各自必须全部 compatible，affected/stale/unknown均拒绝；然后逐 Unit exact绑定 current descriptor，拒绝 overlap、重复声称、gap、extra及 source mismatch。所有 transitive history/current dependencies在返回前重新检查。

preflight PASS不授权 lifecycle mutation。维护者另获 closure授权后，沿用现有 `site-content package-closure`，以同样 repeatable `--page-finalization`代替单个 `--input`，并保留全部 `--generation <structured evidence>`。不能同时传 `--input`；structured范围不因 Page composition省略。closure仍是同一个 `go-learning/package-language-closure/v1`，只 additive绑定 `page_finalization_composition`及 immutable membership reference。所有 reader/current-check重算原 proof；没有第二套 acceptance/state machine，也没有新的语言 A。

future locale正常51页 finalization继续用原单个 `--input`，不需要 composition。Surface-facing只接收统一 `learn-docs-v1` closure，逐 Page provenance仍指向真正原 Snapshot；Surface Reviewer不负责判断 package migration。retry、replacement、bridge、QC rating/finding与角色隔离均不改变。

membership successor 的 Candidate Snapshot 与 successor QC scope 共用正式 ordered predecessor projection。`quality-check scope` 的 `--previous-snapshot-id` 仍指 revision lifecycle 的 primary predecessor，不要求它包含 sibling package 的历史 Units。每个 member 分别验证完整 finalized A、原始 evidence 与 current authorization；secondary predecessor 只从该 member 的 exact accepted candidate set 解析。缺少任一完整 proof、membership/order/source 漂移或重叠均拒绝，不能按 path/count union 或缺省 A 补齐。historical machine reconciliation 在自己的 frozen member scope 验证完整 initial QC；current Snapshot 仍必须覆盖完整 current membership。只有 candidate/validation/attempt 等 identity 完全相同且当前 eligible 的 A 可以 carry，revised Unit 必须 fresh re-QC。Reviewer Bundle 与 JIT current-check 重建同一 scope，carry 仅为 lineage context，fresh judgment working set 只包含 pending、同 kind、最多60个 Units，不继承 initial partition；不新增语言判断、Snapshot/QC schema 或第二个 predecessor CLI flag。

composition 的 structured accepted baseline 可能通过原 Generation 引用其他 package 的历史 closure；其 frozen package ID/SHA 仍须能从 historical descriptor registry 精确解析，不能只注册 membership 的两个 Page predecessors。原 source-adoption/structured-only successor 的历史 descriptor 与当前 descriptor 并存，不能把 current identity 写进旧 closure。解析及提前拒绝边界见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。

## 恢复与重放

preflight失败为无 mutation，保存 exact blocker；修正工程/真正 current eligibility后再只读重算，不重做已成功 source adoption或语言历史。缺当前 glossary/Review或当前 validation失败不能用历史 A绕过。closure使用现有 immutable write-once publisher；已存在输出或终态不明时先核对现有 authority，不覆盖、不盲重放。membership receipt是仓库 source governance变更，后续变更必须保留旧 receipt/registry archives与 descriptor identity，不修改历史语言 evidence。
