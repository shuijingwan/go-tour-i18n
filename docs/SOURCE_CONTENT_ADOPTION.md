# 新内容包与 structured source successor authority

本合同只治理 canonical source 和 pending language scope，不创建语言输出、Reviewer rating、finalization、closure、Surface 或 activation。既有 finalized package 保持原 inventory、UnitID 和历史 evidence；新增页面使用单独的 registered package，继续复用共享 TranslationUnit 生命周期。不能把新增 source 塞入旧 Snapshot，不能以 source adoption 代替语言接受。

## Frozen Page package

`site-content page-package-adopt --input <proposal> --input-sha256 <exact SHA>` 默认只读。输入 schema 为 `go-learning/page-package-source-adoption/v1`，包含新的 `package`、新的 `surface` 和完整、有序、非空的 `ordered_sources` path/SHA。来源必须是当前 frozen manifest 中的 `upstream-exact` Page；已注册的 source、route、package 或 surface 冲突均拒绝。canonical route 由既有 `pageRoute` 计算，不能从目标语言推断。

preflight 生成 canonical receipt、完整 ordered Page Documents、正式 UnitID 和 protected input，作为 pending Generation scope 的机器证据。显式 `--apply --prior-identity <predecessor scope identity>` 才发布 source authority：content-addressed receipt 在 `data/content-package-adoptions/receipts/`，唯一 current pointer 在 `current/<package>.json`。`page-package-adoption-check --input <receipt> --input-sha256 <SHA>` 验证 source、manifest、template dependencies、原 frozen Document/source coordinates、exact membership 和 identity；current loader 通过已发布 pointer 选择该 foundation。frozen representation 不因今日 protector 改变而重写；current workflow 独立按同一 physical source 重建 Page，真实 representation drift 通过[既有 replacement authority](TRANSLATION_UNIT_REPLACEMENT.md#qc-bcd-的-representation-successor-authority)精确衔接，不重新 adoption，不把旧 UnitID 冒充 current。

loader 只构造 `TranslationPackageDescriptor` 和 Page adapter；Generation、import/restore、validation、Snapshot、QC、revision、A-only 和 descriptor package closure 均使用现有共享实现。新包每个物理 Page 一个 TU；内部槽位不成为 TU。initial working sets 使用共享 Page 分组和上限，不改变既有包的 frozen initial membership。future package 的 coverage family 使用自身 exact canonical routes，不继承其他包的 wildcard scope。

## Structured source 的受限正文扩展

`site-content structured-source-adopt --input <proposal> --input-sha256 <SHA>` 默认只读。schema `go-learning/structured-source-content-successor/v1` 包含：现有 `package`、`previous_source` path/SHA、独立 repository-relative `successor_body` path/SHA 和显式 `insert_at` byte coordinate。旧 physical source 保留不动；current source 使用 hash-bound body reference，逻辑 source path、route 和 package 不变。只允许在所有既有 semantic slots 之后插入 source-owned canonical prose；原 prefix/suffix 必须逐字节一致，不能移除、改写或重新分类原正文。

旧槽位复用 source-adopted frozen Document 的 exact ID、coordinate、source 和 protection；新增槽位只从当前 parser 接受的插入区间产生。先验证旧 source round trip，再验证组合后的新 source round trip 和 symmetric protected structure。不能用今日 protector 重写旧表示的 token identity，也不能让语言模型猜 mapping。未绑定此 source authority 的 raw drift 仍然 stale。

`--apply --prior-identity <predecessor scope identity>` 发布 content-addressed receipt 和 write-once pointer；`structured-source-check --input <receipt> --input-sha256 <SHA>` 要求该 authority CURRENT。scope inspection 可在 publish 前用 `structured-source-adopt ... --locale <locale>`，或发布后用 `structured-source-scope --input <receipt> --input-sha256 <SHA> --locale <locale>`；二者只读，机械绑定该 locale 的 accepted closure、Surface、Generation、旧 source/Document、完整 target restore 和逐槽位 target，不授予新语言接受。

沿用现有 `StructuredAssessment` / `StructuredAsset` / `StructuredOutput`，只增加可选的 `structured_source_successor` 与 `carried_slots`：Plan 冻结旧 accepted slots，`Document` 保留完整本地语义上下文；Generation 仅返回剩余新增槽位，不能提交或修改 carried slots。import 合并 frozen carry 与新增 exact set，随后复用同一 restore、protected structure 和 glossary validation。current-check 必须重新取得 accepted baseline 和当前 source authority；历史 reader 保留原 immutable source、Generation 和 closure lineage。旧 glossary 若使 carry slot 需要修订，必须明确处理，不能伪装成 source-only carry。

## Corpus、Review 与执行边界

source adoption 后旧 corpus/Review 不自动覆盖新内容。`site-content adopted-corpus --prior-sha256 <current corpus file SHA>` 默认只读；`--apply` 复用既有 corpus-successor 的 archive、CAS、lock、pending marker 和 atomic publication，不创建第二种 corpus 生命周期。新增包通过 corpus 的可选 `package_authorities` 绑定 exact source receipt 和 contributor paths；source successor 保留旧 contributor 的历史版本，只更新 current source projection。`site-content corpus-check` 必须能从当前 authority 重建同一 corpus。

新 corpus identity 需要正式 glossary refresh（如有实际术语变化）及完整 Independent Unified Review；不能伪造旧 Review CURRENT。source/current-check PASS、pending Generation scope 可机械导出，不代表该 Review 或语言 Generation 已完成。之后 Page 与 structured assets 分别走已有正式入口，不耦合 package，不为特定 URL 建旁路。

current corpus 的唯一 projection 由注册的 source epoch 决定：已有 corpus successor 必须从 exact archived predecessor、current source authorities 和正式 delta 完整重建；unaffected contributors 保留原 frozen projection。`BuildUnifiedCorpus`、`corpus-check` 与 `terminology-check` 共用此选择与重建路径，不以今日 parser 的全量重提取另定义一份 current identity。全量提取仍用于 bootstrap 或显式的新 source/terminology authority 建立，不自动改写历史 epoch。implementation/protection 演进本身不授权 checker 替换已冻结的 contexts；真正 source 或声明的 parser authority 变化必须经正式 successor，不可隐藏 mismatch。

沿用 frozen discovery projection 仍须验证当前完整 terminology manifest/snapshot 的 exact SHA、source inventory、declared parser，以及 discovery contributors 的完整 ID/path/package/purpose/parser 集合；source/current pointer、receipt、archive、corpus canonical bytes/self identity 与全部 raw dependencies 继续 fresh 验证。仅存在 receipt、已 decode 或 prior PASS 均不足以复用；transitional source-before-corpus、缺 evidence、错误 hash 或 authority drift 必须 fail closed。

历史 representation proof 与 current eligibility 分开验证。accepted closure 的历史 finalization reader 使用 `RepresentationSuccessorVerifier.VerifyHistorical`，按已落盘 reconciliation receipt 绑定的 archived glossary 重建同一 representation transition；仍验证 receipt、Snapshot/QC、source/input、原 raw/candidate/validation、映射与 materialized proof 的 exact bytes/hash/identity，并将 archive 和全部引用绑定进历史 proof。current corpus 改变使旧 Review stale，不撤销完整且未变的历史 proof；任何真实 artifact drift 仍 fail closed。

`Verify`、新 reconciliation 和正式 Generation 继续使用 current glossary / full Review eligibility；历史入口不授予 current authorization 或 publication eligibility，历史上下文不能被当前入口复用来绕过 gate。source-only carry scope PASS 只证明 pending/carry 的 exact set 可计算，不代表当前 Unified Glossary Review 或 Generation ready；不得重标旧 Review、伪造新 Review，或以 `/tmp` scope 代替 lifecycle authority。

## CAS、恢复与 replay

source proposal、body、receipt 均不可覆盖，默认 source apply 的 current pointer 也不可替换。相同 input、相同 predecessor identity、相同已发布 pointer 是幂等 replay；不同 pointer、source/manifest/proposal drift、unsafe path、wrong SHA 或 conflicting membership 均 fail closed。pointer 发布前的 content-addressed orphan receipt 不改变 current authority，可在检查 current pointer 后复用同一 exact proposal；禁止更换 input 后盲重放。source apply 不创建 candidate、Generation attempt 或语言接受。

structured source 设置阶段若需纠正尚未消费的新正文，必须另存 body/proposal/receipt，显式使用 `structured-source-adopt ... --apply --prior-identity <base identity> --prior-pointer-sha256 <exact old pointer SHA>`。此受限 CAS 仅允许相同 package、相同 frozen predecessor 的 source 草案纠正：current corpus 与任一 immutable corpus-successor 都不得引用旧 source receipt；与 corpus publication 共用排他 lock。消费后禁止此路径，不得用于已经生成或接受的语言修订。旧 receipt/body/proposal 保留，新 pointer 原子发布；相同已发布新 pointer 的 replay 幂等。corpus commit 前再次验证其全部 source references，拒绝并发 source pointer 漂移。

corpus apply 成功后使用 `corpus-check` 确认，不再以旧 prior SHA 重放：已无 source delta 时 apply 拒绝。出现 `.corpus-successor-pending`、遗留 lock、current pointer / canonical rebuild 不一致或不明写入结果时，报告 `mutation=unknown` 并停止，先检查 exact receipt、pointer、archive 与当前 SHA；不得自动删除 marker/lock、回退 source、覆盖 corpus 或重跑语言阶段。历史 glossary/Review、accepted target、Snapshot、QC/finalization、closure、Surface 和 activation 不改写。

## 已消费 structured source 的 replacement successor

已发布且被 canonical corpus-successor 精确消费的 repository-owned structured source head，如需删除、重组或替换正文，使用 `structured-source-replace`，不能冒充 append 或 unconsumed correction。仅支持 structured semantic Document；physical Page TU、frozen upstream source、没有正式 source head/consumption proof 的输入拒绝。logical path、package、route 不变，不新增 TranslationUnit 或语言生命周期。

proposal schema 为 `go-learning/consumed-structured-source-replacement/v1`，字段为：

- `package`；`predecessor_authority`（immutable source receipt path/SHA）；`predecessor_identity_sha256`（该 receipt 的 self identity）。
- `prior_pointer`（现有 source pointer path/SHA；内容必须精确为 canonical 编码的 predecessor reference）。
- `consuming_corpus_successor`（已安装、当前 corpus bytes 精确匹配其 result 的 immutable corpus-successor path/SHA）。
- `predecessor_package_source_heads`（同 package 全部 current structured heads，按 reference path 排序，包含 predecessor，不能遗漏 peers）。
- `successor_body`（独立、不可覆盖的 repository-relative body path/SHA）。
- `replaced_slots`（按 `old_slot_id` 排序的 `{old_slot_id,new_slot_id}` 数组）。这是 source owner 声明的 retirement/replacement 对，不声称新旧语义或 target 等价，不猜配对。

机器用 current parser 生成完整新 Document，不复制历史 ID 到新语义。仅 parser ID、locator、source/protection identity 全相同的槽位进入 `carry_slot_ids`，reconstruction offsets 可改变。显式 replaced pairs、剩余 removed old IDs、剩余 added new IDs 独立绑定；重复、缺失、把 unchanged ID 冒充 changed pair 均拒绝。未声明对应关系的旧/新槽位分别视为 removed/added，不做语言 heuristic。新 source round trip 必须逐字节一致。

沿用 `go-learning/structured-source-content-successor/v1` receipt 和同一 `data/structured-source-successors/receipts/`、`current/<source-path-hash>.json`。additive `consumed_replacement` 绑定 predecessor receipt/identity、原 pointer、consuming corpus SHA/identity、完整 prior package heads/descriptor、replaced pairs、removed IDs；append receipt 不增加字段，不改变 bytes/identity。`previous_source` 仍绑定最初 logical source foundation；replacement 的 `previous_document` 则绑定紧邻 predecessor head 的完整 current Document。历史链通过 exact references 读取。

```sh
# 默认只读；applied=false、mutation=none
 go run -mod=readonly ./cmd/tour-i18n site-content structured-source-replace \
  --input <proposal.json> --input-sha256 <proposal-file-SHA>

# 维护者另获授权后才执行；replacement 的 prior identity 是 source receipt identity
 go run -mod=readonly ./cmd/tour-i18n site-content structured-source-replace \
  --input <same-proposal.json> --input-sha256 <same-proposal-file-SHA> \
  --apply --prior-identity <predecessor-receipt-identity> \
  --prior-pointer-sha256 <exact-original-pointer-file-SHA>

# 发布后沿用 current reader；被 supersede 的 receipt 继续供 historical reader 读取
 go run -mod=readonly ./cmd/tour-i18n site-content structured-source-check \
  --input <new-source-receipt.json> --input-sha256 <receipt-file-SHA>
```

source authority 跨 locale，replacement command 不接受 `--locale`，readonly 不接受 CAS flags。stdout plan 包含 `authority`、`planned_receipt` path/SHA、`result_pointer_sha256`、`already_current`、`applied`、`mutation`。

apply 与 corpus publication、source writers 共用 `.glossary-corpus.lock`；校验 immutable receipt 后重新构建输入、核对 corpus/peer pointers，落 durable marker `data/.structured-source-replacement-pending`，再检查 exact dependencies，原子 rename pointer 并同步目录。所有历史 source/body/proposal/corpus、target、Generation/Review/closure/Surface 原样保留，不生成语言输出。

**恢复/replay：** 同一 proposal、原 predecessor identity、原 prior pointer SHA，且 pointer 已是 exact result，返回 `already_current=true`、`mutation=none`，不覆盖 receipt。pointer 仍为 exact prior 且没有 marker 时，canonical orphan receipt 可原字节复用；先复核 preflight，再显式 apply。不同 head、错误 prior、proposal/body、corpus 或 peer drift 均拒绝。lock/marker 遗留、I/O 不明、pointer 既非 exact prior 又非 exact result，均停止并按 `mutation=unknown` 诊断，禁止自动删 marker/lock 或盲重放。marker 保存 prior pointer reference、result receipt reference、proposal reference：先核对 SHA、重建 receipt、比较 pointer 是否等于 prior 或 canonical result reference，再由维护者单独授权恢复；没有自动 recovery/apply shortcut。历史 reader 不把 partial current publication 当 CURRENT。

source apply 后，旧 corpus 的 source pointer reference 变 stale，current eligibility 失败是预期行为。下一阶段仍用同一个 `adopted-corpus --prior-sha256 ... [--apply]` publisher；仅当 basis bytes 精确等于 replacement receipt 的 consumed corpus 时，允许读取该 historical basis，替换完整 affected contributor，验证其他 exact references。`ReadHistoricalAdoptedCorpusSuccessor` 按原 projection authorities 和 archived predecessor 重建旧 corpus；旧 pointer bytes 只能从当时 immutable authority reference 确定性恢复，不能用 current head 冒充旧 head。source adoption 的历史 proof 可读取 corpus identity/contributor projection；新的 adoption 和真正 current language eligibility 仍执行完整 corpus/Review gate。

replacement partition 不是语言 PASS。后续 structured plan 只能从已机器验证的 accepted ancestor 恢复已有 target，且仅保留与新 source exact unchanged、确实存在 accepted target 的 slots；未接受的旧 source-only slots 继续 pending。changed/replaced/added slots 必须正式 Generation/validation/独立 Review，removed slots 不拼进 target。之后才是 current package closure、Integrated Surface、activation。新的 corpus identity 仍按 current contract 完成 glossary refresh/full Independent Unified Review；historical PASS 不绕过它。

replacement 的完整 affected contributor context IDs 绑定 immutable replacement authority identity；即使仅重组相同文案、术语文本未变，也形成新的 current semantic projection/corpus identity，不能让旧 full Review 因文本碰巧一致而继续 current。这不改 source slot IDs 或 target 文本，且不会影响 append-only 的原 contributor identity。
