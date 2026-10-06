# Glossary Compatibility / Freshness Gates（Site v2-B）

## Versionless locale authority（V2-D 收敛）

v1/v2/v3 是 implementation/content-package milestone，不是 locale language governance namespace。Content package 可以版本化；Locale language system 不版本化。全站只有 `locales/<locale>/glossary.yaml`，不得 Tour/Docs/v2 split。current executable migration、统一 corpus、一次完整 glossary generation/refresh、统一 structured assets、完整 Page TU 与 delta-aware integrated Surface contract 见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。本节是全站扩展入口；下文旧 Tour workflow/evidence 保持兼容，不授权重复初始化 locale 或改写历史证据。

已有 Tour-complete locale 首次扩展：一次 unified glossary refresh → 一次独立 full unified Review → compatibility → 一次 pending structured-assets Generation → 两批 Learn/Docs Page Generation / 两批 Page QC → 一次 integrated Surface Review。unchanged Tour/UI/meta/SEO carry；只有真实 affected scope reopen。新 locale 从零执行同一 locale campaign，glossary 与 structured-assets 各一次，Tour 和 Learn/Docs 各用其 surface-specific parser/TU batches；没有第二次“V2 locale init”。Course SEO 是 Page-derived consumer，按自己的 canonical source/ready target contract 单列，不强行并入 structured-assets；没有 Learn/Docs canonical SEO authority时不创建 SEO target。

新 unified receipt `go-learning/unified-glossary-review/v1` 与 old v1/Tour legacy coverage additive 共存。新 Generation 要求 current unified coverage；旧 completion仍按旧 evidence合法验证。unified current exact优先；多个 current unified receipts、glossary/corpus mismatch fail closed。旧 receipt immutable，不 retroactively upgrade corpus coverage。glossary bytes不变但 corpus变了仍需一次 full unified Review；不猜造 corpus compatibility shortcut。新 glossary byte change须 full unified PASS后才用 V2-B semantic delta/精确 lineage复用旧结果。


本规范是 downstream compatibility 的正式执行入口。唯一全站术语表仍为 `locales/<locale>/glossary.yaml`；不创建 Tour、Learn、Docs 或其他 surface 的 override。兼容性分析只复用历史已审核成果，不作语言审核，也不授权新的 Generation 使用旧术语表。正式语言 Generation / 独立 Reviewer 模型政策仍为 GPT-5.6 Sol High。

## 不可变 bytes 与完整 Review

每次修改已审核 glossary **之前**，先执行 prepare/archive：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility archive --locale <locale>
```

它要求原 `RequireCurrentGlossaryReview` 的完整 byte-current PASS，保存 exact bytes 到 `data/glossary-history/<locale>/<sha256>.yaml`。locale 由路径及正式 Review authority 绑定；文件名 SHA 必须与实际内容一致。相同 bytes 幂等复用，不同 bytes 拒绝覆盖；路径 traversal、父目录/文件 symlink、非普通文件拒绝。不依赖 Git history、网络或聊天。旧 bytes 缺失时 assess/resolver fail closed，不允许仅凭 SHA 重建。

之后才由 Generation session 修改 glossary，并由独立 Reviewer **重新完整审核**。comment/order/format 等任意 byte change 也使旧 receipt stale；Glossary Reviewer Bundle 始终包含当前完整 bytes。新 `glossary-review record --decision passed` 会先归档本次完整已审核 bytes，再保存原格式 receipt；旧 receipt 不变。这个自动归档不能找回修改前丢失的版本，不能替代 prepare。

V2-B 的当前 cohort bootstrap 使用正式 Go CLI：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility archive --all
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility config-baseline --all
```

`--all` 从现有 live authority 确定 65 locale，先预检整个请求集合。此次生成 65 份 glossary baseline archive，以及覆盖 65 locale 的 83 份可证明原 config bytes 的独立 baseline；另 25 个无法证明原 config bytes 的历史 gate 不补造 baseline。所有动态 evidence 按 locale 独立，shared config 原始 bytes 只存入 content-addressed immutable common archive。上述 bootstrap 不改变任何 glossary、Review、QC、Surface 或 content completion 文件。

## 正式 evidence 与 current-check

取得新 full Glossary Review CURRENT PASS 后：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility assess \
  --locale <locale> --old-sha256 <archived-old-sha256>
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility check \
  --locale <locale> --evidence data/glossary-compatibility/<locale>/<identity>.json
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility status \
  --locale <locale> --old-sha256 <archived-old-sha256> --scope tu:<unit-id>
```

CLI 输出 JSON；`status` 为 `exact`、`compatible`、`affected`、`missing evidence` 或 `stale evidence`，附 enum reason 与有序 chain。`exact` 仅表示调用方两个 glossary identity 相等，消费者仍须验证自己的 source/candidate/provenance/full Review gates。只有 `exact` / `compatible` 可以接受；其他结果不授予复用权限。

Schema identity 为 `go-learning/glossary-compatibility/v1`，JSON Schema 为 `data/glossary-compatibility.schema.json`。正式文件在 `data/glossary-compatibility/<locale>/<evidence_identity_sha256>.json`，不可覆盖。old/new archive refs 各自含真实 path/SHA；new full Review ref 绑定完整 passed receipt 的 path/file SHA；另外包含 normalization/algorithm version、normalized delta、exact context inventory、affected/compatible scope、prior lineage 和 self identity。self identity 是 `evidence_identity_sha256` 置空后的 Go contract JSON SHA，file reference SHA 则是保存后的完整原始 JSON bytes SHA，两者用途不同。

共享 Go API `ResolveGlossaryCompatibility` 重建当前 formal inventory、delta、classification 与 self identity，严格比较全部 evidence。unknown fields、duplicate JSON keys、trailing JSON、错误 locale/version/path/hash、missing archive/review、空/重复/遗漏 scope、overlap、tampered reason、断链/循环/分支及 context mismatch 都 fail closed。未知版本不会部分容忍。

## Semantic delta 与影响规则

Normalization `tour-glossary-normalization/v1` 使用当前 glossary loader 的窄行格式，保留其实际消费的 literal key/value；支持 `mandatory`、`preferred`、`forbidden`、`keep`。排序后比较 category/term/old/new，新增/删除分别保留 null。full-line comments、section/entry order、表示空白不算语义变化，但不降低完整 Review 的 byte gate。重复 section/entry、locale mismatch、encoding ambiguity、unsupported legacy `terms` 或不支持的语法均拒绝 normalization。

Impact algorithm `tour-visible-glossary-impact/v1` 复用 validator 的 `visibleCandidateTextBytes`、present program-font/link/directive/preformatted 保护语义、Go Example comment parser，以及 forbidden 的 Unicode/apostrophe boundary。rich UI 使用现有 markup whitelist 提取 label 文本并去掉 link target；plain UI 保留 literal 可见文字。

- Delta 为空：`semantic_delta_empty`，所有 exact context compatible。
- 只有约束删除：`constraint_removed`，不机械要求重译。
- 新增/改变 mandatory/preferred/keep：在真实可翻译 source 中出现或存在保守的大小写、空白、标点、词形碰撞时 affected；不存在时 `changed_source_term_absent`。keep 当前版本不尝试语言判断 compliant，出现就 affected。
- 新增 forbidden：visible target 的现有 boundary 规则命中时 affected；URL、link target、program font、directive、protected preformatted/Go code 不作为译文命中。可翻译 Go comments 和 link labels 仍检查。
- 无可靠 source→target mapping 的 mixed Go/JS/template surface：任何新增/改变约束均为 `unknown_parser` affected；不扫描 raw source 后宣称 absent。空 delta/纯放宽仍 compatible。

Inventory 精确包含 `tu:<UnitID>`、`seo-page:<page-id>`（完整 source/canonical target）、`seo:<page-id>`（v2 canonical English description + localized description）、`ui:<key>`、`article:<article-file>`、`surface:<surface-id>`，保存 source/target 与实际文件 refs/hash。每类 formal exact-set 由现有 Catalog/UI/article/Course/surface parser 确定。UI、article、Course asset 的 whole-file refs 是保守的当前性绑定：其中任一改变需重新 assess；不在扫描遗漏时默认 compatible。

## Lineage、QC 与恢复

Resolver 从原 SHA 沿唯一连续 old→new chain 到实际当前 glossary；每一跳对请求 scope compatible、receipt/archives/context/self identity 可验证才放行。历史中间 glossary 的完整 passed Review 仍绑定其 immutable bytes，链末端必须有当前完整 Review PASS。检查历史中间 Snapshot 时也验证它后面的路径能安全到达当前 glossary。不同 new SHA 分支一律 ambiguous，不猜路径；同一边的 stale context 版本不能替代唯一 current 版本。

Artifact repair 后原 inventory stale，重新 assess 产生新的 immutable evidence。同一历史边可对新 exact inventory重新分析，必要时递归保存新的 predecessor evidence；旧 evidence 不覆盖。重新评估使用真实 archived old/new bytes 与原 full Review，不生成 Reviewer 结论。修改过 candidate/source 的 Unit 仍因精确 identity 改变而不能继承旧 A。各历史边仍逐跳检查，不能跳过曾受影响的中间约束选择快捷路径。

新 Snapshot 仍完整、绑定当前 glossary bytes，并要求完整 Review CURRENT。旧 Snapshot 不修改。QC scope 比较 source/candidate/batch/validation/attempt/page identity；跨 SHA 只有 scope compatibility 成功的 A 可 carry-forward。affected/missing/stale Unit 进入 pending；发生 protector drift 时走原 `--glossary-stale` 精确同 kind recovery，否则先建立当前 Snapshot，再由正式 QC 的 B/C/D finding 授权 revision。Page/Example 分离、每批最多 60、protected validation、A-only finalization 不变。

新 finalization 绑定当前 Snapshot/glossary，Unit 中保留原 `source_snapshot_id`、当前 exact Snapshot identity，以及 optional `compatibility_chain` refs；原 QC response 不伪造。QC Reviewer Bundle 保存实际 scope、当前完整 glossary、相关 immutable evidence/archive/Review bytes。历史 finalization 保持原 bytes；若原链本身已 stale，不假装其 current，使用新的当前 Snapshot/finalization 收口。

约束删除导致 protector 输出变化时，只能经绑定历史 Snapshot/真实 input/candidate/validation 的 archived glossary 恢复原 protected input；随后仍验证原 raw response restore 以及当前 canonical validator。不能据此处理受影响 Unit或让 retry/new Generation 使用旧输入。

## Course SEO、Surface 与 config projection

新 v4 `record-a` 先把本次 project/seo exact bytes 保存到 `data/surface-config-history/<sha>.go`，再写新 gate；检查时重建这些原始 bytes 的 projection，要求与 gate 内 projection 一致。schema v4 自己保存的 AST/hash 不能脱离真实 archived config bytes 单独放行。

Course SEO loader/refresh/revise 对每 page 的历史 `glossary_sha256`，要求 exact current SHA 或同时通过 `seo:` 与 `seo-page:` lineage。其余 source、canonical description、target、route、schema、provenance gates 不变。refresh 只替换实际 stale pages，保留其余 page 的历史 SHA/provenance；同一个文件可以合法包含不同 generation glossary SHA。asset 修改后必须重新 assess 原 glossary 边，才能重新证明未修改 pages 的 context 当前。

Surface A 的 glossary mismatch 只有 `scope=*` 全部 context compatible 才接受；任何 TU/UI/article/SEO/other surface affected、或后续文案 identity 改变都会 stale。新 v4 另外绑定 `tour-language-context/v1` identity：formal source/canonical target exact-set，以及 Tour/runtime/first-party JS/template/partial 原文；registry/project/SEO 由各自 semantic projection 验证，避免 raw-file hash 重新引入 debt。历史 v1/v2/v3 没有此字段，跨 glossary 必须用不可改写的 V2-A completion 原 refs + 完整 package/context digest 证明原 source/public 文案未变；缺少可恢复完整 closure 时 fail closed。`internal/contentidentity` 共享原 wire DTO，`sitecontent` 使用 type aliases，未改原 schema/JSON encoding。重新 assess 不能把后来修改的 other-surface 文案变成旧 Surface A。修复后按原 Surface workflow 取得必要独立审核。

新 Surface `record-a` 写 schema v4，保留历史 whole config SHA，并嵌入 `tour-surface-config-projection/v1`。中性 helper 使用 Go AST/gofmt 提取原 declarations（含 values/fields/function bodies），忽略普通 comment/format/order；旧 declaration 必须逐项原样存在。仅允许未被旧 declarations 引用的新增普通函数和 literal constants。新 variables/types/imports/methods/init、compiler directive 或未知 AST 不证明安全；原 canonical/SEO/public values或函数语义变化都 stale。这是保守的 Tour semantic contract，不是简单忽略 shared file hash。

历史 v1/v2/v3 可在原 project/seo whole SHA 仍 exact 时记录：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-compatibility config-baseline \
  --locale <locale> --review-id <historical-review-id>
```

独立文件 `data/locale-surface-reviews/<locale>/<review-id>.config-baseline.json` 使用 `go-learning/tour-surface-config-baseline/v1`，绑定原 gate path/file SHA、原 whole-file SHA、原 config bytes（`data/surface-config-history/<sha>.go`）、结构化 projections/self identity。current-check 重建原/当前 projection 并比较；无法取得真实原 bytes 时保持 stale。既有 v1/v2/v3 registry/English UI/Production compatibility 继续按原 gate schema组合，不产生新的 registry兼容权限。

Surface Reviewer transport 继续导出真实完整当前文件；包的原 inputs projection 形状保持，避免为新增 evidence 机械改变当前 65 locale closure。`sitecontent` 的历史 completion 在确有 glossary/config变化时，验证 locale-owned全部原 refs，并只用已验证 archive/baseline 重建原 glossary/config package 部分，要求重建后的**完整原 context digest** 与历史 completion 一致。不能改写历史 completion或忽略其他 source/key-set/deployment/context变化。`internal/sitecontent` 的 FrozenUpstreamCommit 引用 debt 保留，V2-B 不做 namespace refactor。

## 执行与验证边界

评估和 current-check 是本地 deterministic Go workflow，无语言生成、无网络、无 Production。新 Generation Bundle、initial/revision/retry、locale assets、Course localization/maintenance仍要求完整 CURRENT Glossary Review + 当前完整 bytes，旧 artifact compatible 不使旧 transport current。批量 bootstrap 只保存 immutable machine baseline，不代表重新审核。

测试只运行新增 compatibility tests、直接相关历史 gate/validator/CLI focused tests，以及一次允许的 `content-scope check --all`。不运行 repository-wide、browser、build、preview、publish、deploy 或 upstream sync。本阶段完成即停止，不进入 V2-C/V2-D，不 commit/push。
