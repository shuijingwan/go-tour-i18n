# 缺失原 Generation ZIP 的 legacy provenance authority 设计

**Maintainer contract review 已批准 implementation；本轮不授权真实 record。** 下方 feasibility/design 保留为设计依据，当前可执行语义以文末「正式 implementation command contract」为准。authority kind 为 **`recorded-receipt`**，仅供独立 historical-response provenance 验证；当前 Generation/representation consumers 尚未接入，064/json 继续 blocked。首次真实使用前必须重新执行 first-real-use gate。

当前执行合同仍见 [Translation task](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)、[Runbook](RETRANSLATION_RUNBOOK.md#current-validation-predecessor-representation-readiness)、[replacement](TRANSLATION_UNIT_REPLACEMENT.md#current-validation-predecessor-representation-transport) 和 [handoff / first-real-use](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。[approved-interval 设计](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md) 是下一独立 authority layer，本文不实现它。

## 先做 feasibility matrix

以下为 2026-11-05 单 Unit 只读 probe 的观察，不能代替未来 consumer 的 fresh checks。`AVAILABLE` 指该行指定的 facts/bytes 是否存在；`RECONSTRUCTIBLE_WITHOUT_GUESSING=yes` 指可精确读取或从 surviving bytes 确定性复算，不表示可以重建 ZIP。标量行的 SHA256 是**承载该字段的 receipt bytes hash**，不是对字段字符串重哈希。`NEEDED_FOR_CURRENT_RECOVERY` 指本次 narrow historical-response consumption 所需历史事实，不包含下一层尚未获准的 transport 能力。

路径缩写仅为表格排版，按以下文字展开；`R#member` 表示 R 的 ZIP member，`/field` 表示 JSON 字段，不是文件系统路径：

```text
OLD_ID=99824b13c26ffe219230ad0171e93d850003d16aa9f780c2f06c41bc46845cb3
UNIT_ID=b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c
B=data/retranslation-runs/zh-CN/chatgpt-zh-CN-031
N=data/retranslation-runs/zh-CN/chatgpt-zh-CN-064
M=data/translation-unit-reconciliations/zh-CN/learn-docs-qc014-machine-successors
S=data/representation-reconciliation-reviews/zh-CN/20261016-zh-CN-learn-docs-json-reconciliation-001
R=S/bundle.zip
H=docs/protected-representation-evidence/qc014-historical-representations.json
Q14=data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-014/manifest.json
Q23=data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-023/manifest.json
REOPEN=data/quality-check-reopens/zh-CN/zh-CN-learn-docs-qc023-language-audit-current-validation.json
G=data/generation-archives/zh-CN/6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542.zip
```

`FORMALLY_AUTHENTICATED_BY` 是已有正式 hash/reference 的信任链，不声称密码学签名，也不从成功 import 推导未知 ZIP 成员。用户给定的可信 receipt SHA 是本设计的信任起点之一；R 的独立已记录 semantic receipt 又冻结了相同 receipt 的 bytes。

| FACT | SURVIVING_ARTIFACT | PATH | SHA256 | FORMALLY_AUTHENTICATED_BY | AVAILABLE | NEEDED_FOR_CURRENT_RECOVERY | RECONSTRUCTIBLE_WITHOUT_GUESSING |
| --- | --- | --- | --- | --- | --- | --- | --- |
| predecessor batch manifest | 原 manifest bytes，16 Page revision | B/manifest.json | `87fecc65c7a25319ffe6007645198ce572d5da93c276ae2484b4f08afd37f0ee` | M/receipt.json original_manifest；R 中 exact 副本 | yes | yes | yes |
| formal input | OLD_ID 的原 protected source input | B/inputs/OLD_ID.article | `06554ff7462184a90b908d394519b5a5a5fc1a72d14177f61b9b583efcd15bab` | immutable batch input_sha256；M replacement.old_input_sha256；H mapping 精确复算 | yes | yes | yes |
| source | 完整 English source bytes | _content/doc/tutorial/json.md | `9bb5bed860a71bdb4dc9776c9eff2e5f4a40f1ca9851851bfd1619c1793970be` | batch/validation/Q14/Q23/replacement source_sha256；R 的 predecessor-source.txt/current-source.txt | yes | yes | yes |
| raw response | 真实 OLD_ID attempt 1 output，8230 bytes | B/raw-responses/OLD_ID.article | `6f9ec7c679de88d7c47b649237166b1f89c888daa38332d1d0b2b5097b8a701e` | import outputs hash/size + canonical install path rule + validation；M original_raw；R exact 副本 | yes | yes | yes |
| result.json | 原 process result bytes | B/result.json | `7de559b4db451289a83898da04917c1c482518a110db020129ff16241a38ecca` | M original_result；R exact 副本；逐 Unit passed/path 与 validation 相同 | yes | yes | yes |
| import receipt | 已接受的 immutable receipt bytes | B/generation-imports/initial-manifest.json | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | 维护者给定可信 SHA；S/receipt.json → R inventory 中相同 bytes | yes | yes | yes |
| provider=chatgpt | receipt 字段 | B/generation-imports/initial-manifest.json /provider | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | trusted import receipt；正式 provider/model/batch policy | yes | yes | yes |
| model=gpt-5.6-sol-high | receipt 字段 | B/generation-imports/initial-manifest.json /model | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | trusted import receipt；不能由今天默认模型补填 | yes | yes | yes |
| batch=chatgpt-zh-CN-031 | receipt 字段及批次目录 | B/generation-imports/initial-manifest.json /batch_id | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | receipt/manifest/result/validation/Q14/Q23 同时绑定 | yes | yes | yes |
| attempt=1 | receipt 字段 | B/generation-imports/initial-manifest.json /attempt | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | receipt + passed validation + Snapshot final attempt；不是 machine reconciliation 新 attempt | yes | yes | yes |
| task kind=translation-unit-revision | receipt 字段 | B/generation-imports/initial-manifest.json /task_kind | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | receipt；031 manifest revision_feedback_source=quality_check | yes | yes | yes |
| recorded output path/hash/size | outputs/OLD_ID.article；上列 raw SHA；8230 | B/generation-imports/initial-manifest.json /outputs | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | receipt **只记录 bundle_path**；install path 另由 immutable input basename、attempt=1 标准安装规则、validation.RawResponsePath 和 result/Snapshot 唯一证明 | yes | yes | yes |
| GenerationBundleSHA256 的值 | receipt attests import 时 ZIP hash | B/generation-imports/initial-manifest.json /generation_bundle_sha256 | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | trusted receipt；值为 `6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542` | yes | yes | yes |
| exact original ZIP bytes | 无 surviving bytes | G | recorded=`6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542`；实际不可重哈希 | receipt 只 attests existed at import time；维护者 EXACT_ARCHIVE_MATCHES=0 | no | no | no |
| InputIdentitySHA256 的值 | opaque historical 字段 | B/generation-imports/initial-manifest.json /input_identity_sha256 | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | trusted receipt；值 `fdc5e84fc20f2912a2f6a401b176820d3ec06291744684def6c7cc5a03ca62dc`；不能复算成员集合 | yes | yes | yes |
| GenerationManifest hash/size | receipt attests member 曾存在 | B/generation-imports/initial-manifest.json /generation_manifest | `e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26` | recorded SHA `55db87dbffd719194483e82f53c7600cd4bcf7a48668f4b9a7cb310474db6b0d`，size=14024，bundle_path=generation-manifest.json | yes | yes | yes |
| actual Generation manifest bytes | 未在 surviving formal evidence 发现 | 原 ZIP manifest.json；旧 Result transport generation-manifest.json | recorded=`55db87dbffd719194483e82f53c7600cd4bcf7a48668f4b9a7cb310474db6b0d`；无实际 bytes hash | receipt 只记录 hash/size；R 不含此 member；不能逆 hash 求内容 | no | no | no |
| original ZIP member inventory | 无完整 Files/ExpectedOutputs bytes | G#manifest.json | unknown | receipt 不携带完整 Generation Files inventory；只有 output inventory 与 aggregate hash | no | no | no |
| authority files from original ZIP | 不知道那次 invocation 的 exact docs/context bytes | G#authority/*、context/*、generation-context.json | unknown | 当前 docs、R 的资料和其它 ZIP 均不能冒充原 member | no | no | no |
| glossary bytes from original ZIP | 原 invocation member 无已证 identity | G#formal/glossary.yaml | unknown | R 的 review-time glossary 为 `70be302647a76f00de9f9802001e9f106b24cae666906c3a6dbef7f5c57bdb39`，**不能证明**它等于缺失 ZIP 的 glossary | no | no | no |
| original protected input bytes（仓库存留） | 与 formal input 同一保存文件 | B/inputs/OLD_ID.article | `06554ff7462184a90b908d394519b5a5a5fc1a72d14177f61b9b583efcd15bab` | batch/M/H 的 immutable 链；不额外声称已重读 G#formal/inputs/OLD_ID.article | yes | yes | yes |
| 原 ZIP 中 input member 的逐字节相等证明 | 原 member 缺失 | G#formal/inputs/OLD_ID.article | unknown（仓库 input SHA 不能填作已重读 member SHA） | 只能依赖正式 immutable batch 身份和安装合同建立 narrow substitute；没有直接 ZIP equality proof | no | no | no |
| 原 invocation 的 editable predecessor baseline / 全部 revision context | 未证明完整原成员集合 | G#formal/protected-candidates/* 等 | unknown | 不能把当前重算 revision inputs 称作原 bytes | no | no | no |
| 持久化旧 protected mapping | H 的 OLD_ID document，R 冻结副本；old-representation.json | H；R#old-representation.json | H=`e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4`；member=`c0973ccef8f26c6d960af02ac904eb4aca703dba34d73c4b9fbb98ddea58246e` | M options.historical_documents；S/R inventory/question；source coordinates/input/restore 精确复算 | yes | yes | yes |
| historical candidate | 原 process candidate | B/candidates/OLD_ID.article | `9d978bca55c326a2c24d43684da62383523fe01901c4e22563c3512581dc904d` | Q14 + S + M predecessor + R；旧 raw 经保存 mapping exact restore | yes | yes | yes |
| historical validation | attempt 1 passed evidence | B/validation/OLD_ID.json | `1d43ad7bb60fb1a6c7875c3a76ee199210d5a8e069b7eb6fc0f63cb7bbfb13e5` | Q14/S/M/R refs；严格核对 locale/batch/unit/source/input/raw/candidate/attempt | yes | yes | yes |
| machine reconciliation | immutable replacement receipt | M/receipt.json | `8f159c38611cd9538c4c71976e66f8191e6fa6007ab18583e983f213b5875324` | Q23 representation_reconciliation；receipt identity 及 original_manifest/result/raw/predecessor/semantic refs | yes | yes | yes |
| materialized current raw | historical machine-only semantic projection | M/UNIT_ID.raw | `5ceea40a6ed7e8da4e3f0d1e2770f5301a4e0bb6b932bcbbf5b37254099b609e` | M current_raw；使用已批准 question 和 frozen event representation 重现；**不是 imported model output** | yes | yes | yes |
| current candidate | 与原 candidate bytes 完全相同 | M/UNIT_ID.candidate | `9d978bca55c326a2c24d43684da62383523fe01901c4e22563c3512581dc904d` | M + Q23 candidate_sha256 + REOPEN | yes | yes | yes |
| machine validation | reconciliation 专用 record，非新语言 validation | M/UNIT_ID.validation.json | `2fb866bc92d5ce3386da0f18830f162d69646ec9b137ea4f1f8b2b863e91c450` | M + Q23 validation_sha256；typed replacement/predecessor/raw/candidate/glossary bindings | yes | yes | yes |
| semantic reconciliation receipt | 已记录 whole-interval approval | S/receipt.json | `8b952e7d9c80f5283239f4910fa47f0f31f56f91f5f4ff47743a297da4e5d5c9` | M semantic_authority；自身 identity、exact decision/Bundle/QC/predecessor refs | yes | yes | yes |
| semantic review Bundle | 35-member immutable review transport | R | `36759d10e6cbcf8dcdb74a72a18483849e474b18cf439cecc0508f6144e834a3` | S/receipt.json Bundle SHA/identity；每个 inventory member hash/size | yes | yes | yes |
| semantic review response | 独立 Reviewer 原 response bytes | S/decision.json | `666e3cfd46f70ebc4f8495d3f73ffae7f9730ff541b49fd0957e5d3e164e8035` | S receipt evidence；strict decision 与 receipt 决定相同；不新增判断 | yes | yes | yes |
| original Snapshot | 选择 OLD_ID/031/attempt 1 的历史 Snapshot | Q14 | `234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619` | M snapshot + S/R + 原 QC A chain | yes | yes | yes |
| current predecessor Snapshot | 选择 explicit reconciliation 的 Snapshot | Q23 | `2fdde982c05eb53d54ff4c353780e07055b9e4ea19ae28105f52f4abdaa7c7d9` | REOPEN current_validation.snapshot；selected batch/attempt/candidate/validation/M refs | yes | yes | yes |
| current-validation reopen | 本轮唯一 language authority | REOPEN | `338c96b02ab4b1f793a2d08b56e98597c1666ef820fd7d76b4788d6474f7fcbd` | N selected previous_review_path/SHA；authority_source=current-validation；原 A/current validation assessment | yes | yes | yes |
| current source/input/representation | source 同上；current formal input 272 tokens；Unit/source/input 与 Q23/M current 一致 | N/inputs/UNIT_ID.article；_content/doc/tutorial/json.md | input=`2bb5c43ced6e3a7c3c8b94f6be127793c6c6063429cbbcb09435cd412df6e059`；source 同上 | N/M input hashes；当前 source parser 独立重建 Unit/input；完整 package/adapter 仍须 future consumer 检查 | yes | yes | yes |
| frozen reconciliation event representation | semantic 决定发生时的 current-side mapping | R#current-representation.json | `2e93513d388a5c0246e97c2db31658e9b8a340590697f966ed5b153b4ab09a64` | R inventory/question；只用于重现历史事件，不冒充 today-current descriptor | yes | yes | yes |
| pending revision batch | immutable 064 manifest | N/manifest.json | `280338157c956da37f1970f32f80b81ff404fcc1c2a8ec11720f68fb8844bb4b` | 原 batch；selected reopen/input exact refs；不是 legacy 授权落点 | yes | yes | yes |

### 此矩阵证明与不证明的内容

1. **bytes themselves still exist：** receipt、batch、formal input、raw、source、result、validation、candidate、frozen mapping、machine reconciliation、semantic review 三件套和 Snapshot 都可重新读取、精确重哈希。R 保存的是这些历史材料的另一份已正式冻结副本。它不保存原 Generation Bundle 或其 Generation manifest。
2. **receipt attests this existed at import time：** 原 ZIP hash、aggregate input identity、Generation manifest hash/size 和 output identity 是 receipt 原本记录的事实。hash 值存留不等于被 hash 的 bytes 存留；import success 也不表示每个旧 input/docs/glossary 当前仍可读。
3. 安装事实来自可信 immutable import record + exact response hash/size + 唯一 outputs path + 原 batch basename/attempt 安装合同 + passed validation/result + Snapshot/reconciliation 的交叉绑定。receipt **没有 install_path、session、generated_at 或逐 output UnitID 字段**。本设计不补造这些字段；OLD_ID 由正式 batch 与 exact basename 绑定，当前 UNIT_ID 由明确 replacement 绑定。
4. 旧 input/mapping/source→raw→candidate 与原 validation/attempt 都可由存留字节证明。`031` 的旧 ID 与本次 current ID 不同；这不是本轮新增 replacement。M 已记录的 `OLD_ID → UNIT_ID` 及 Q23 的 current input 与 064 相同，不能改写 031 或把其 raw 文件重命名为 current ID。
5. 原 invocation 的完整 Files、ExpectedOutputs 到 InstallPath 表、docs/context/glossary 内容、editable predecessor template 的完整输入闭包，均不能从 receipt hashes 逆推。在当前已知 evidence 下不可恢复；若将来找到原字节，可按后文重新比较。在此之前必须把这些信息标记 unknown，不能用今天 exporter 或其它 archive 填空。

## 核心判断与最小事实边界

选择 **A：SUFFICIENT_FOR_NARROW_LEGACY_ATTESTATION**。理由不是“曾成功 import，所以所有输入都正确”，而是本例要消费的历史对象仅为：**031 / attempt 1 / OLD_ID 的 exact imported response 及其已有 candidate lineage**。raw provenance 来自 trusted receipt；decoder 所需旧 protection 完整 bytes 已独立存留于 H/R，且能重现原 formal input 与 exact candidate。其后 whole-interval semantic transition 由已记录 S/M 证明，不需要读取原 invocation 的 docs、glossary 或 editable baseline 来猜 mapping。

这里批准的 substitute 明确比原 archive 弱：它不证明存留 batch/input 的 bytes 曾逐字节出现在原 ZIP，也不提供完整 invocation replay。它依赖这些 artifacts 原有的正式 immutable 身份，以及后来 S/M 对其 exact bytes 的正式冻结。新的 recorded authority 必须明确授权**这组 surviving facts 的限定消费**，不能将该弱化藏进旧 receipt 的解释。

缺失 ZIP 内容对该限定用途为何不是安全必需项：

| 缺失内容 | 当前 narrow consumption 的替代证明或排除边界 |
| --- | --- |
| full Files / original manifest bytes / aggregate identity 重算 | 不再判断旧 Bundle CURRENT、不 replay/import 031；只保留 receipt 中的 opaque identity；actual raw 的 provenance 单独 exact match |
| 原 docs/context/instructions | 不重新判断旧 Generation 是否遵守所有旧语言 instructions；原语言质量与 A 只由原独立 QC chain 承担；本轮 language authority 仍为现有 reopen |
| 原 invocation glossary | 不声称复现旧 invocation glossary world；历史 accepted-input reader 已有自己的 hash-bound archived glossary/QC world，仍按原 reader 验证；后续 current full glossary/Review/compatibility/active validation 独立重验，不能拿未知旧 glossary 豁免它们 |
| 原 editable predecessor template | 031 已有完成且正式接受的普通 Page response；存留 H mapping 足以 exact decode。若是必须读取缺失 comment/Surface/bridge codec 才能 decode 的其它案例，本合同拒绝，不由 raw/candidate 反推 codec |
| arbitrary other Unit 的 invocation inputs | 只授权列出的 OLD_ID → UNIT_ID；031 receipt 的其余15个 outputs 不因共享 invocation 获得 legacy exception |

未来 approved-interval consumer 的 historical fact footprint 必须限于 matrix 中 `NEEDED=yes` 的 facts，并逐项注明用途。如果它需要一个未存留的原 ZIP member、原 glossary identity 或其它新历史事实，则 **reject / INSUFFICIENT_EVIDENCE for that consumer**。不能因为本设计选择 A，就承诺任何未来 transport 一定可用。typed current finding nonoverlap、opaque parent/children inventory、active validation、token exact/unique 等都仍在下一层审定；任何 gate 不通过仍 blocked。

## 四层 authority separation

| 层 | 唯一职责 | 不获得的权限 |
| --- | --- | --- |
| Language authority | 原 `authority_source=current-validation` reopen；原 A、exact ordered affected set/finding、Snapshot 不变 | legacy 不产生 finding/rating、不放宽 locality、不重译 |
| Historical import provenance | 既有 initial-manifest.json 原字段，只证明当时正式 import 记录的 provider/model/task/batch/attempt/output/opaque input identities | 不扩大为全部旧 ZIP inputs 已知、不修改 receipt |
| Legacy provenance migration authority | 新 **recorded** authority 显式选择 exact receipt + surviving artifacts，允许指定 Unit 的 historical-response provenance substitute | 不是 reconstructed Bundle，不是 import/replay authority，不证明任意旧 Bundle CURRENT，不对整个 batch/locale 生效 |
| Approved-interval representation transport | 后续独立审定如何消费已批准 whole frozen interval，生成严格不可编辑 parent 的 current transport | legacy 不替它作 token mapping，不隐式启用 approved-interval-opaque，不取消当前 protection |

既有 semantic receipt 仍只授权 whole-interval correspondence。legacy 层没有新语义 authority，也不赋予 current `Marshal` atom 的 target coordinates。`approved-interval-opaque` **仍然需要**下一独立 authority contract；没有 machine reason 可以省略它。

## derived 与 recorded receipt 的选择

| 方案 | 优点 | fail-closed / audit 限制 |
| --- | --- | --- |
| 每次 Bundle build deterministic derive，无新 receipt | 字节 proof 可重算；不增加正式文件 | exporter 一旦发现缺 archive 就自动采用较弱 provenance，会把授权决定隐含在实现版本中；难以固定哪些 Unit/receipt/缺失事实获批准，也易被未来 locale 泛化 |
| explicit recorded receipt，consumer 每次仍重算 proof | 授权范围、信任起点、unknown fields、允许用途、依赖 identity 可独立审计；不会把工程 probe/缺失 archive 变成自动许可 | 多一次正式 record，需原子 write-once/CAS/unknown recovery；receipt 不能缓存 PASS，也不能补齐缺失事实 |

选择第二种。**可确定性 derive 的是 evidence graph；不可隐式 derive 的是允许较弱 graph 替代原 archive 的正式授权。** receipt 记录今天实际建立的 migration authority，不能声称今天重新发生 031 Generation/import。独立 QC 不由维护者 record 代替；新 receipt 不制造 model invocation、session 或生成时间。

## 未来 formal command contract（未实现，不可执行）

repository 当前有 `generation-bundle export/current-check/recover/archive-imported/import`；其中 `archive-imported` 专门接收真实原 ZIP。这是 historical Generation provenance 问题，应在 **generation-bundle family** 增加独立 `legacy-provenance` operation，不塞进 quality-check language reopen，不复用 archive-imported/recover 的名称或输入含义。

下列是未来设计字段，不是可粘贴执行命令或 `COMMAND_CONTRACT_VERIFIED=yes`。实现完成后必须重新通过 docs → implementation → direct CLI tests 的 first-real-use gate。

| 字段 | Future contract |
| --- | --- |
| COMMAND | `generation-bundle legacy-provenance`；默认 readonly inspection；显式 `--record` 才记录 |
| OPERATION | 检查并计划 exact single-Unit historical-response substitute；record 重新构造同一 proof 后 publish 新 authority；不修复旧 ZIP |
| INPUT_KIND | 显式 trusted import receipt + expected receipt SHA、single historical/current Unit、batch/attempt/task、original Bundle SHA、exact predecessor Snapshot/reconciliation/semantic refs，以及完整 registered/native package selector；不接收任意候选译文或 caller-authored token mapping |
| INPUT_PATH_CONTRACT | 所有 evidence/receipt/Snapshot refs 必须 repository-relative、regular、no symlink/no traversal；例如 `data/retranslation-runs/.../initial-manifest.json` 合法，`/home/user/repo/data/...` 即便在 repo 内也拒绝；identifiers 不是路径；archive path 由 receipt locale/hash 唯一派生，caller 不可选择 alternate fallback path |
| REQUIRED_FLAGS | 计划 `--locale`、`--content-package`、`--batch-id`、`--unit-id`（historical ID）、`--current-unit-id`、`--attempt`、`--task-kind`、`--import-receipt`、`--import-receipt-sha256`、`--original-bundle-sha256`、`--previous-snapshot-id`、`--previous-snapshot-sha256`；machine/semantic refs 必须由该 Snapshot 的显式 graph 选择；record 另须 exact `--expected-identity`，不能用 caller 值扩 scope |
| FORBIDDEN_FLAGS | `--all`、多 Unit、latest/auto-discovery authority、`--bundle` 替代 missing ZIP、`--apply`、force/overwrite、provider/model override、candidate replacement、target atom/occurrence 参数；没有为当前例外自动开全 locale 的模式 |
| STDOUT_KIND | readonly `{planned_receipt,planned_path,identity,recorded:false,mutation:"none"}`；record `{receipt_path,receipt_sha256,identity,recorded:true,mutation:"known"}`；字段契约实现前仍须固化 direct tests，不能把 plan 当 receipt |
| MUTATION_EXPECTATION | readonly=none；record=known，仅新增一个独立 immutable authority receipt；不写 batch/import receipt/raw/candidate/validation/Snapshot/reopen，不写 archive，不执行 downstream |
| IDEMPOTENCE | 同 exact evidence graph + policy/version 的 planned identity deterministic；record write-once，任何已存在 destination（即使相同 bytes）拒绝重复记录；只读可核对已存在 exact receipt，但不能报告本 invocation recorded=true |
| CURRENTNESS_CHECK | 不检查旧 Bundle CURRENT；每次重读 trusted receipt SHA/provider/model/scope、每个 surviving ref hash/size、raw/install-path、source/input/mapping/restore/candidate/validation/result/Snapshot/reconciliation/semantic chain 与无 pending/conflict；检查 canonical archive state；current consumer 另验 current package/source/input/glossary/Review/adapter/validator 和自身 authority |
| PRIOR_OR_CAS | `--expected-identity` exact planned proof，absence destination、原 receipt SHA、complete dependency inventory hashes、archive absence observation、无 pending guard；publication 前 fresh 重建和重读 scope/dependencies/absence；并发变化拒绝，不信先前 preflight PASS |
| SUCCESS_STOP | readonly 停止于审阅计划；record 停止于 receipt exact readback，下一独立阶段才可审议 approved-interval；不自动 export/current-check/Generation |
| FAILURE_STOP | 任一未知/hash/scope/identity/冲突/I/O 失败保持 blocked，保留旧全部 artifacts 和诊断，无部分授权、无 fallback、无自动 successor |
| RECOVERY_REPLAY | record failure 先读取 destination/staging/完整 before-after inventory，诊断 committed/none/unknown；unknown 禁止 replay/downstream；已发布 exact receipt 不重写、不用新 ID 规避重复，合法 no-write check 后才交回后续阶段 |

### record 落点与 identity

设计的 destination type 为独立 immutable JSON authority receipt，归属 **existing Generation provenance storage family**：

```text
data/generation-provenance-authorities/<locale>/<identity>.json
```

这是未来路径类型，**本轮不创建目录或 receipt/schema implementation**。不放在 `B/generation-imports` 冒充历史 import，不放在 `quality-check-reopens` 冒充语言授权，也不修改 M/S。实现时需将其纳入 canonical formal inventory/transport readers；不能把普通 docs/probe JSON 认作 authority。

receipt identity 应按版本化 canonical encoding（复用 repo existing hash/strict-JSON/reference primitives）覆盖：

- kind/version、migration purpose=`historical-response-provenance-only`、明确不提供 original-input-inventory/currentness/replay 的 claim set；
- locale/package identity、exact original batch/task/attempt/historical Unit/current Unit/replacement、trusted import path/SHA、原 Bundle SHA、opaque InputIdentity 和 GenerationManifest hash/size；
- selected output bundle path、由正式合同验证的 install path、raw hash/size、formal source/input、旧 mapping 和 exact restore lineage；
- 原 Snapshot/current selected Snapshot、manifest/result/validation/candidate、M/S/review Bundle/decision、全部所用 immutable dependencies path/hash/size 与对应 ZIP member inventory；
- allowed consumer kind/version 与 exact historical fact allowlist、missing facts/unknown values 列表、proof contract/implementation version identity。不能仅用 original Bundle SHA 当 receipt identity。

作用域为 **per-Unit / per-invocation**：031 attempt 1 的一个 OLD_ID 输出，经 M 明确链接本 UNIT_ID。receipt 内完整 import receipt 的16 output inventory 只用于完整性/冲突检查，不授权其它15 Unit。多 Unit 必须各自满足同一 gate、各自显式授权；不得把一个 Unit 的成功扩成 batch/global exception。未来 locale reuse 复用 verifier/合同，不能复用本例信任起点或 authority bytes。

identity 不包含“原生成时间”或编造 session。record 的真实操作时间/执行者若有审计字段，表示**migration record 事件**，不能进入 historical Generation provenance。为保持 deterministic planned identity，非 proof 的操作日志与 immutable proof identity 分离；正式 encoding/字段在未来实现任务中与 docs/tests 一致固化。

所有 surviving artifacts 原样引用，不机械复制整份 batch 或所有 bundle inputs。record 的 immutable dependencies 不包含持续变化的 current glossary head 作为永久 current 授权；其历史 hash-bound glossary world 原样保留，today-current 的检查属于每次 consumer invocation。implementation 演进必须明确 compatible reader contract，不能凭 docs drift 追认历史 Bundle CURRENT。

## fail-closed 条件

以下拒绝均发生在 publication/transport 前；失败不修改现有 evidence：

| 条件 | 必须行为 |
| --- | --- |
| canonical G 实际存在 | 禁止进入缺档 legacy 分支；normal archive 优先；按下一节 fresh 比对 legacy 依赖；不得用 legacy 屏蔽 corrupt/wrong-hash archive |
| import absent、SHA 不等 trusted expected SHA、strict shape/identity 错误 | reject；新 receipt 不可替代旧 receipt，不可重写/重新解释它 |
| invalid formal provider/model/batch prefix，或 session/generated_at 缺失却要求证明它们 | reject；不借今天默认模型补字段 |
| raw absent/hash/size/path/UTF-8/EOF mismatch | reject；只允许原 bytes，不能规范化后再算作 receipt output |
| duplicate output、task/batch/attempt/old Unit/install path mismatch | reject；不能只按 hash 找一个相似 output；不能把 materialized raw 当 model output |
| receipt 本身不记录 InstallPath | 必须同时通过 immutable input basename、exact attempt path rule、validation/result/Snapshot/ref 链；任一缺口 reject，不从 missing ExpectedOutputs 猜路径 |
| raw→candidate restore、result/validation、candidate/Snapshot、source/input/replacement mismatch | reject；不修改 validation/Snapshot；不复用历史 PASS 省略 byte proof |
| 原 archive SHA unknown、InputIdentity/GenerationManifest identity malformed | reject；opaque recorded identity 也必须完整合法 |
| missing frozen mapping、mapping 需从 raw/candidate/目标语言推断 | reject；不能因为一次 restore bytes 相同就批准非唯一 token mapping |
| conflicting artifact、两份 incompatible receipts、later attempt/current selection 冲突 | reject；不选 newest、较方便或字节相同的一支规避冲突；检查精确 invocation 命名空间，无 latest authority discovery |
| scope 超出 exact historical/current Unit 或 consumer allowlist | reject；即使同 batch/source path 也不授予额外权限 |
| consumer 请求 surviving evidence 未覆盖的历史事实 | reject；重新作 feasibility 判断，不能扩大旧 record 的 meaning |
| pending import/record transaction，读取不完整、symlink/nonregular、dependency drift 或 mutation=unknown | preserve/diagnose；禁止 record/replay/downstream |
| 原 QC/A、semantic approval 或当前 language authority 的独立 gate 不满足 | reject；legacy record 不给语言 PASS、不消除 current validation defect |

只读 feasibility probe 检查了单 Unit 正向 byte/lineage 可行性，**没有实现或运行上述未来 verifier**。后续工程必须以 focused negative fixtures 证明每个拒绝项，包括 absent/changed receipt、raw hash/size/path、provider/model、attempt/task、Snapshot/input/mapping 冲突、跨 Unit、missing fact、archive later appears 和 publication race；不能在本轮声称这些 production gates 已落地。

## exact archive 后来出现

每次 record、export/current-check、import/process/retry/Snapshot 的 provenance reconstruction 都先检查 canonical G；不能永久缓存 `archive_missing=true`。

1. G 存在时首先验证 exact SHA、regular path、strict ZIP inventory 与 receipt identity。wrong hash/corrupt/symlink 立即 fail closed，不退回 legacy。
2. 若已有 legacy receipt，fresh compare 它覆盖的所有历史 facts 与 actual archive：locale/batch/task/attempt、aggregate identity、GenerationManifest hash/size、exact selected ExpectedOutput/InstallPath、formal manifest/input/source refs、所声称的 surviving bytes、任何 archive 实际携带且与该 claim 有关的 decoder。legacy 未声明的原 docs/glossary 不反向假装以前已知。
3. 任何冲突均停止；不得删改 legacy receipt、覆盖原 archive 或选弱证据继续。即使 archive SHA 与旧 receipt 相同也必须检查 graph，不把 SHA 单独当 reader PASS。
4. 全部 match 后 consumer 可以恢复 normal archive-backed historical path，legacy record 仅保留作 migration audit。已经冻结的 derived transport 仍保留其 bytes；独立 current-check 必须重验更强证据，若 canonical graph 变化导致旧 transport stale，则通过正式刷新路径处理，不能 patch ZIP。
5. normal archive 的存在本身不代表它在今天 CURRENT，不允许以它自动重放原 Generation。也不绕过下一层 approved-interval、current validation 或 QC。

record publication 前检查缺档状态只能证明当时观察；若并发原 archive 恰在 publication 后出现，不删除新 receipt，下一 consumer 必须按上述 stronger-evidence rule 检查。cannot determine publication state → `mutation=unknown`，先 exact inventory diagnosis，禁止盲重放。

## 064 与 derived transport graph

legacy authority 不需要回写 031 或 064 manifest。未来可设计在**新 derived Generation Bundle**中存放显式 pinned `legacy-provenance` reference/proof，将新 recorded receipt 与全部 surviving refs/member inventories 纳入 Files、aggregate identity、canonical reconstruction 与 dependency guards；不自动扫描 latest receipt。

为了避免 reference 在 process 阶段丢失，未来必须证明整个传输闭包：

```text
explicit legacy receipt selection
→ new Bundle derived provenance metadata + frozen refs
→ independent current-check 重建同一 graph
→ normal import 只消费这份真实新 Bundle，原样封存实际新 invocation archive
→ process/retry 从实际 imported new archive 重建 historical decoder/provenance
→ validation/Snapshot 沿已有 explicit selection 重验相同闭包
```

若 exporter 需要新 selector/flag，该 contract 属于 modified formal CLI，必须实现后重新 first-real-use；本设计不向现有命令偷塞 flag。legacy receipt 本身不能传给普通 import，当成原 ZIP 或新的语言输出。current-check 不信 caller transport JSON 自称 authority，必须验证 recorded receipt 的 exact scope 与全部 surviving bytes。

此图只证明 legacy reference **可以规划为 transport dependency**，不证明后续 approved-interval layer 和所有064 Unit readiness 已成立。因此当前结论：

```text
SAME_BATCH_064_REFRESH_POSSIBLE=unknown
SUCCESSOR_REVISION_BATCH_REQUIRED=unknown
```

未来只有 graph 在 export/current-check/import/process/retry/Snapshot 全部贯通、approved-interval 独立 contract 确认无需 manifest delta、064 无正式 output/import 且完整 representation readiness PASS，才可决定 same-batch refresh。若任一 authority 必须写入 batch manifest，则原064 immutable，必须合法 successor batch；不得猜编号、改064或再造原 reopen。

## 工程 evidence、限制与唯一下一步

已读取：AGENTS、handoff/task/runbook/replacement/approved-interval 文档；`generation_import_archive.go`、`current_validation_predecessor.go`、`generation_bundle.go`；process/validation/accepted-frozen-input、Snapshot/review、reconciliation/semantic review readers 及相关 probe。现有 `ArchiveImportedGenerationBundle` 只能保存真实已有 ZIP；`verifyImportedRevisionInvocation`/`currentValidationPredecessorImport` 仍要求 original archive，本设计不改变实现。

新增 opt-in single-Unit engineering probe：

```sh
GO_TOUR_LEGACY_PROVENANCE_PROBE=1 go test ./internal/sitecontent \
  -run '^TestLegacyGenerationProvenanceFeasibilityReadonly$' -count=1 -v
```

最终 PASS（test 0.90s；package 输出 0.915s，不含编译），未调用 exporter/importer/recorder、active validation 或完整 lifecycle。检查了 trusted receipt exact SHA/字段/raw hash-size、R inventory/identity 与多份历史 exact 副本、manifest/result/validation/Snapshot/reopen bindings、H→saved input→raw→candidate、today formal Unit/input、frozen event semantic materialization→M raw/candidate。probe 只支持本次 feasibility，不授予 legacy capability 或当前语言资格。

初版 probe 误用即时 parse 的 descriptor 重现 historical semantic question，得到 `reviewed semantic question changed`。修正为使用 R 中已冻结的 event representation 后 exact reproduction PASS；today parser 的 Unit/input 另行检查。这是 probe 的历史/current world 选择错误，未发现需改 production reader 的新 failure，不记录 deferred issue，也不通过忽略 question equality 解决它。

formal mutation accounting scope 为完整 `data/**`、`locales/**`、`production/**`：before/after 均为48794个 regular files，完整 path set + SHA-256 inventory byte-exact 相同，三个 mutation counts 均为0。inventory 文件本身的 SHA 均为 `b495ed4921a38dbd839d39b51478b746f001125d45a8980934757053f5c91a20`。工程 probe 和本设计文档均不是正式 receipt。原 archive availability 直接复用维护者 `EXACT_ARCHIVE_MATCHES=0`，probe 只检查 canonical path 缺失；没有重复 exact archive hash search 或重建历史 ZIP。`git diff --check` PASS，另检查了本轮未跟踪 docs/test 的 whitespace（无错误）。

预期顺序仍为：

```text
legacy provenance authority
→ approved-interval authority
→ all 064 representation readiness
→ new/modified CLI first-real-use gate
→ Generation Bundle export/current-check
→ Generation
```

唯一下一阶段：**`legacy-generation-provenance-authority-contract-review`**，由维护者审定此 recorded authority 的 exact trust/claim/consumer 边界；不执行实现或 record。未来若授权工程，应先完成独立 legacy verifier/record/consumer contract 和 focused tests，任何首次真实使用都先执行 first-real-use gate；上述顺序不豁免新 legacy record 自身的 first-real-use。本轮停止在 legacy implementation/record、approved-interval implementation、Generation Bundle export 和 Generation 之前。

## 正式 implementation command contract

本节在实现之前固化合同，沿用上文批准的 family、content-addressed destination 和 per-Unit scope。设计阶段的未实现说明是历史记录，不代替本节。支持已有 semantic reconciliation 明确选择的普通 Page attempt 1（initial 或 revision）；依赖缺失 imported codec 的 restricted response、retry 和其它未经证明的 graph 均拒绝。没有新增语言判断、reopen 或 approved-interval transport。

| 字段 | 当前合同 |
| --- | --- |
| COMMAND | `generation-bundle legacy-provenance` |
| OPERATION | 默认 readonly inspection；`--record` 重新构建同一 narrow graph 并记录 immutable authority |
| INPUT_KIND | exact complete package + historical/current Unit + batch/task/attempt + trusted import path/SHA + original Bundle SHA + explicit current predecessor Snapshot ID/SHA；reconciliation/semantic refs 只沿 Snapshot graph 读取 |
| INPUT_PATH_CONTRACT | evidence refs 为 repository-relative regular no-symlink paths；`data/.../initial-manifest.json` 合法，absolute path、`..`、反斜杠、symlink 均拒绝；IDs 不是路径 |
| REQUIRED_FLAGS | `--locale --content-package --batch-id --unit-id --current-unit-id --attempt --task-kind --import-receipt --import-receipt-sha256 --original-bundle-sha256 --previous-snapshot-id --previous-snapshot-sha256`；每个 flag 只能出现一次；record 另须 `--expected-identity` |
| FORBIDDEN_FLAGS | positional、重复 flag、多 Unit、`--all`、`--apply`、`--bundle`、override provider/model、candidate/mapping/target-occurrence；inspection 不接受 expected-identity |
| STDOUT_KIND | JSON `{planned_receipt,planned_path,identity,recorded:false,mutation:"none"}`；record 返回同一 plan 并增加 `receipt_path/receipt_sha256`、`recorded:true,mutation:"known"`；publication/readback 不确定则返回 plan 与 `mutation:"unknown"` 并失败，禁止重放；已发布并 exact readback 后 cleanup 失败仍为 `recorded:true,mutation:"known"`，保留 guard，先诊断，禁止重放 |
| MUTATION_EXPECTATION | inspection=none；record 仅新增 `data/generation-provenance-authorities/<locale>/<identity>.json`，以及 publication 期间的独占 `.pending` staging guard；不写其它 formal artifacts |
| IDEMPOTENCE | inspection deterministic；authority_id=receipt identity（canonical JSON 将 identity/authority_id 清空后 SHA-256）；record duplicate ID 一律失败，不覆盖；相同 invocation/Unit 的冲突 record 也拒绝 |
| CURRENTNESS_CHECK | fresh import/raw/input/source/candidate/validation/result/Snapshot/reconciliation/semantic/QC graph；完整 package identity；dependency path/hash/size；canonical archive state；不判断旧 Bundle CURRENT、不授予 current language eligibility |
| PRIOR_OR_CAS | expected-identity + destination absent + invocation/Unit 独占 guard + complete fresh dependency graph；最终 link 前重查 archive absence/所有依赖与冲突 records；preflight 不是缓存 PASS |
| SUCCESS_STOP | inspection 后审阅 plan；record 后只核对 receipt，不执行 approved interval/export/Generation |
| FAILURE_STOP | preserve evidence；拒绝 hash/scope/provenance drift、unknown 或已出现 archive；不得自动 fallback、删改 receipt 或创建 successor |
| RECOVERY_REPLAY | `.pending` guard 或 mutation=unknown 必须先只读诊断 destination/staging/inventory；禁止盲重放或自动清理未知事务；已发布 receipt 不重写 |

独立 API `VerifyLegacyGenerationProvenance` 消费 explicit receipt path/SHA 和 exact requested options，不选择 latest，也不由 receipt 自行扩大 scope。每次重建同一 evidence graph；返回 typed provenance、immutable dependencies 以及 `legacy-recorded` 或 `normal-archive` provenance source。原 archive 后来出现时先验 exact SHA/ZIP/receipt，再 fresh compare manifest/input/output 与覆盖的全部 facts；冲突 fail closed，匹配优先 normal archive。record 看到任何 canonical archive 都拒绝 legacy record；不删除旧 legacy receipt、不覆盖 archive。archive 的缺失只记录为当时观察，不声称能机器证明永久丢失。

receipt schema 为 Go strict JSON `go-learning/legacy-generation-provenance/v1`，固定 capability=`narrow-per-unit-historical-response-provenance`，允许消费历史事实的 reader contract=`historical-response/v1`。scope、unknown facts、全部 binding 和 dependency inventory 均由 canonical builder 构造；没有 reconstructed Bundle/member inventory、target mapping、language approval 或 replay 字段。读取时拒绝未知字段、非 canonical receipt、错误 path/identity、扩大 capability 或另一个 Unit。后续 consumer 需要 allowlist 之外的历史事实时必须拒绝。

本轮实现结束的唯一下一阶段为 `legacy-generation-provenance-authority-first-real-use-gate`，此前不执行真实 record。`approved-interval-opaque` 仍是下一独立 authority layer，current-validation predecessor、普通 Bundle/import/process/retry/Snapshot 不获得 missing archive bypass。

### Implementation evidence 与当前停止点

实现提供 `PrepareLegacyGenerationProvenance`（inspection / explicit record）、`VerifyLegacyGenerationProvenance`（exact receipt reference + requested scope 的独立 reader）。proof algorithm 为 `ordinary-page-surviving-import/v1`，strict receipt 同时保留原 import 的 opaque identities 和显式 unknown facts。依赖包括完整 current package 的 source bytes、历史 QC Snapshot/candidate/validation/batch/input/raw/result/glossary、machine/semantic artifacts 与逐成员 inventory；这些 proof dependencies 不扩大 per-Unit capability。v1 不支持 nested reconciliation 的历史 QC 输入，遇到该形状明确拒绝，不能选择 latest 或猜 codec。

Focused tests：CLI flags/package ownership；synthetic missing-archive inspection/record/fresh reader、duplicate/scope/unknown target-mapping rejection；dependency drift/provider/model/task/attempt/install lineage；archive later appears/covered-fact conflict/final publication race；publication unknown guard；已发布但 cleanup 失败仍报告 known。真实 json 仅运行 opt-in readonly API regression，计划可重建，未 record。所有 synthetic writes 均在 TempDir，未对真实 batch、receipt、Snapshot/reopen 或 Production 写入。

当前 `current_validation_predecessor`、Generation Bundle、import/process/retry 和 Snapshot 未消费新 authority；没有 `approved-interval-opaque` 实现。真实 json/064 仍 blocked。下一阶段唯一为 `legacy-generation-provenance-authority-first-real-use-gate`；本次 targeted tests 不替代实现完成后的 docs → implementation → direct CLI tests 顺序重读，也不授权真实 record。
