# Surface revision transport capability

本合同只处理 TranslationUnit Page 的 `surface_review` revision transport，不提供语言判断、QC A、replacement、closure 或 Surface PASS。`CURRENT_INPUTS` 证明 exact inputs/current authority；`REVISION_TRANSPORT_CAPABLE` 另外证明 persisted finding 已由显式 machine region request 绑定到可执行的编辑表示。前者不能推出后者。没有 capability authority 的新 Surface Generation Bundle export/current-check fail closed；不等待 Generation 读完 Bundle 才发现无可编辑位置。

## 分域与 source-owned regions

FAILED Integrated Surface receipt、独立 Reviewer/evidence、exact candidate artifacts 与 shared reopen 继续决定语言 finding/Unit/predecessor 授权。historical decoder 只重建旧 raw；QC-only `representation-bridge` 与 current-validation-only comment transport 的权限保持独立。Surface capability builder 复用这些已验证的 source-owned mappings，**不复用它们的语言 finding authority**。

machine discovery 只提供以下坐标，不从 finding prose 推断语义：

- source parser 已拥有的非 teaching-comment ordinary Page slot；坐标由 canonical Unit/source SHA、slot index 和 kind 确定。技术 atoms、HTML/Markdown/link structure、soft-wrap boundaries 保持原 mapping。
- 已被 source parser 识别为独立 inline-code atom 中的 API 描述词：严格形如 `identity method`、`identity function` 或 `package-identity package`。identity 必须有 Go 标准命名空间，或该 atom 所在的已绑定 source link 的 API fragment；package 必须为标准 Go package。只打开尾部自然语言描述词，identity、空格前缀、code delimiter、link destination/绑定不变。不能打开代码块、literal、directive、任意 opaque atom 或未经 namespace/binding 证明的单词。

首版实现限于提供该 source-coordinate interface 的 registered Page adapters；其它 kind/adapter 没有此 machine proof 就失败，不自动套用 Page 或 current-validation 规则。discovery 的 source/target preview 只来自已验证 token/slot correspondence，供准备显式 request 对照，不能把 preview lexical search 当作 exporter 的 scope inference。

维护者提供显式 request，将每个 persisted finding SHA 映射到 discovery 提供的非空、无重复 region ID exact-set。request 是 representation scope 提案，不是 Reviewer translation。机器逐项核对 source ownership、Unit/predecessor/reopen/finding identity 和完整 raw→candidate proof；它不声称理解自然语言 predicate 或替代 Reviewer 对修订内容/完整性的判断。region mapping 不明确时停止，由工程准备明确提案，不能由 exporter 猜测或静默打开全部 Page。

ordinary representation 已能表达请求的 source prose 时保持原 raw/decoder，`kind=ordinary`，不制造新 bridge。restricted current-validation predecessor 则从原 import/archive 的普通 ancestor、精确 comment mapping 和 passed raw 组合 ordinary source mapping，再只开放 request 指定区域；未授权正文和所有 comments/code 保持 opaque。已有 inherited bridge 同样只提供 old response decoder。

需要新投影时，codec 从已验证 raw 的既有 source token/slot correspondence 组合；不调用 Protect(candidate)，不 reverse-protect restored candidate，不猜 translated token occurrence。普通 prose 按已有 token 边界分段；inline descriptor 只分解 source-owned atom 的已证明 descriptor payload。所有其它字节通过 opaque tokens 恢复。新增 token、跨 segment、外部文字、markup/code injection、scope superset 或 descriptor identity 改动均失败。

这是 **revision-only** response codec：canonical source、Page parser 输出、formal Unit ID、saved protected English input 和 batch manifest 均不变，codec 不安装为 canonical source。restore 后仍执行 active canonical validator、glossary/Review、独立 QC。若需要改变 canonical parser/protected input、formal Unit identity、技术身份或不支持的结构，则此 contract 拒绝，走现有 replacement/reconciliation successor。

## 正式 CLI contract

`quality-check surface-transport` 消费 locale、已经存在的 Surface revision batch ID，以及可选的 explicit region request；**batch ID 不是 manifest 文件路径**。package、predecessor 和 reopen 从该 batch 读取，不能另指定或猜 latest。与共享 CLI 一样，显式 `--content-package` 不能指定 partial/wrong package。

三种互斥操作：

1. `--discover`：readonly source coordinate discovery；stdout 为 plan，`revision_transport_capable=false`。发现坐标不是 capability PASS，不可据此 handoff。
2. `--input <request.json>`：readonly capability preflight；request 路径可为绝对路径或相对当前目录的普通文件，拒绝 symlink/非法 JSON。输出 `current_inputs=true`、`revision_transport_capable=true` 才是 preflight PASS；无法证明即 nonzero。
3. `--input <request.json> --record`：重建两次相同 exact proof，fresh recheck 后 write-once 发布 `data/surface-revision-transports/<locale>/<reopen-id>.json`。必须是未产生 outputs/import/process/retry 的 batch；重复 ID（即使相同 bytes）拒绝。不能覆盖 batch、旧 reopen 或任何原 evidence。

request schema 是 `go-learning/surface-revision-transport/v1`：`locale`、`batch_id`、`manifest_sha256`、ordered `units[]`；每项含 `unit_id`、`finding_sha256`、非空 `regions[]`。按 manifest 完整 ordered Unit set，不能 partial、superset、跨 predecessor 或 mixed kind。discovery authority 中的 request metadata 和每项 binding 的 finding SHA 可用于准备 request；**选择 region 是显式工程/维护者 scope decision，discovery 不自动授权它**。

receipt 保存 request、exact reopen SHA、Snapshot candidate/validation/attempt、passed raw path/SHA、current adapter authority、private projection/codec、完整 current dependencies、frozen historical dependencies 与 identity。record 是新 immutable authority mutation，不是 language result。新/修改 formal CLI first-real-use gate 通过前禁止交接真实 `--record` 命令。

## Bundle 与 response consumers

新 Surface Bundle export 必须读取 exact reopen 对应的 capability receipt，并独立重建完整 proof；与 stored bytes 不同即 stale/tampered。不能 carry export PASS，current-check 重新执行同一 builder。Bundle 将 capability receipt 和每 Unit projection 放入 `formal/surface-transport-authority/`、`formal/surface-transport/`，进入 Files inventory/hash、aggregate identity；`formal/protected-candidates` 为可执行 baseline，`formal/historical-protected-candidates` 原样保留 passed raw，`formal/candidates` 保留 reviewed restored candidate。`generation-context.json` 分别给出 `current_inputs` 与 `revision_transport_capable`。

当前 entry 验证已记录的 exact reviewed Surface scope/artifact 与 reopen immutable integrity，另以 shared predecessor selector/current authorization 证明今天的 exact TU eligibility；current registered owner/membership 仍完整保留，不能用 49 predecessor 降低 51 current owner。它不重放旧 Site-wide lifecycle，也不把 49-A proof 宣称为整个 51-Page package 或 Surface PASS。source、saved/current protected input、glossary/full Review、active machine authority 均重验；readonly registration guard 跨 projection construction 保持，观察 current heads、目录新增和依赖变化，current Review selection 与全部 references 在返回前重读。Bundle serialization 后再次独立构造 capability proof。callback、目录存在或过去 CURRENT 不授权 capability。新 receipt 不赋予其它 Unit、structured finding、QC/Preview/current-validation 权限。

import 后的 response decoder 属于 historical representation domain：必须由该 batch 的实际 imported Generation archive 精确封存同一 capability bytes；重验 frozen refs、raw/candidate/validation/attempt，不能因已经 consumed 的旧 Surface review 再要求其 reviewed candidate 成为最新输出。process/Snapshot/current assessment 仍独立执行 fresh current source/glossary/validator guards。旧 archive 中没有此 authority 的 legacy historical decoding 不被改写。

Surface-bounded projection 不自动成为后续任意 QC finding 的 editor；不能表达的新 QC finding 继续 fail closed，要求合法 representation recovery，不能把旧 Surface scope 扩成全 Page。普通未受限 representation 的现有 QC 行为保持。

后续独立 QC B/C/D 可按 [same-Unit Surface→ordinary QC bridge](TRANSLATION_UNIT_REPLACEMENT.md#surface-bounded-predecessor--independent-ordinary-qc-bridge) 建立新的 machine construction receipt。原 Surface codec/raw/reopen 仅作为 frozen historical dependencies；新 ordinary template、独立 QC finding 和 write-once bridge 是不同 authority。没有 recorded/current bridge 的 ordinary QC handoff 仍失败，原 Surface transport 的区域和 decoder 不改变。

成功停点：capability proof/record 可 review；新的 canonical Generation ZIP handoff 前另按 export contract 验证。失败停点：保留所有旧 evidence、batch、transport；不自动 record、re-export、retry、reopen 或 Generation。本轮工程 proof 不授权真实 mutation。
