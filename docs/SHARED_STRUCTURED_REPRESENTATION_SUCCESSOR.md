# Shared structured canonical representation successor

这是 raw source 不变时、已进入 Structured Generation 的 shared Document 的独立 source authority，schema 为 `go-learning/shared-structured-representation-successor/v1`，`locale_scope=all-locales`。它不提供翻译、语言 acceptance、Surface PASS、glossary/corpus successor 或 activation。Homepage 的 Page TU / teaching-body schema、pointer 与 [Shell representation successor](SHELL_REPRESENTATION_SUCCESSOR.md) 保持独立：其单一 Page head 不能无歧义表达多个 shared structured asset。

## 注册 owner 与 bounded proof

owner 复用 current structured corpus registry 的 `package + asset_id + Document.path`；v1 只支持已被 Shell adoption/foundation 绑定的 `site-v2-shell`、`ShellExtractionContract`、`kind=data`、非 Page Document。必须提供一个历史有效且实际输出了该 asset 的 StructuredGeneration Reference；其 source/contributor identity、完整 frozen Document 必须等于 exact predecessor。任意 repository/integration file、Homepage、wrong package/path/raw SHA、不同旧 representation 或未产生输出的 context 都不获权。Surface finding owner 仍是原 package + Document path，不另造 Surface 身份。

request schema 为 receipt schema 加 `/request`：`package`、`asset_id`、`document_path`、`raw_source_sha256`、`previous_representation_identity_sha256`、`consumed_generation` Reference、`parser_rule`、排序且非空无重复 `slot_ids`。previous identity 是完整 predecessor ShellLanguageSource 的 identity，不是 raw SHA。该历史 Generation 仅证明 representation 曾被消费，其 locale、target 和语言判断不参与 prospective classification。`parser_rule` 明确选择下面的一种 delta；不得混合两种 rule 或隐式采用其它 parser 变化。

rule `go-learning/source-role-slash-prose/v1` 只在显式选择的 source-owned HTML text/accessibility prose slots 中，对旧 `role=atom` slash token 使用既有 source-role protector policy。URL、带根/显式相对路径、文件格式、标准 Go package/API、protocol、code/HTML/template bindings 等已有 technical authority优先；slash spelling本身不构成 identity。不会使用目标语言、finding prose、词形白名单或 alphabetic/alphabetic 放行规则。只释放 fresh source-role proof 证明为普通 prose 的旧 slash atom，其他旧 bindings、slot coordinates/role/context/encoding、HTML/template/code/gaps 全部保持。每个请求 slot 必须有真实 delta，未知 slot、无 delta、额外 slot/asset 或未支持 encoding 拒绝。receipt 的非空 `released_atoms` 完整列出该 rule 的 exact binding set；不隐式采用 parser 对其它 slots/assets 的变化。

necessity proof 是旧 representation 强制保留的 atom 在同一 raw source 的 source-role policy 中属于 editable prose，以及完整 old/new topology 和 byte-identical source round-trip。它不依赖某门语言的替换，也不把 Surface free-form finding 当 mutation authorization。语言 revision authority 仍由原 recorded finding/lifecycle 提供。

显式 function/package/field/state 等 identity 位置中的未知 slash 拼写保持 ambiguous/fail-closed；不能因不在已知 namespace 中就当作 prose。receipt 同时给出 previous 和 `prospective_representation_identity_sha256`，两者分别绑定完整 old/new representation。

### Sentence-final period rule

`go-learning/source-sentence-final-period/v1` 是独立的、跨 locale 的 shared HTML sentence representation rule。legacy `ExtractShellLanguageSource` 与 source foundation 不变；只有正式 shared successor head 采用后，current Document 才包含新增 slots。它不直接改译文、raw template、renderer、href、identifier 或 template action。

`slot_ids` 在本 rule 中选择 **Old 中紧邻 inline link 之前的 visible-text prose prefix slot**，不是 link label，也不是不存在的旧 punctuation slot。必须同时满足：prefix 是 `<p>` 内独立可见自然语言，紧接 link start tag 之前仅有 whitespace，末字符是字母；link 有可见字母/数字 label；紧随 `</a>` 的完整 text node（除 whitespace）恰为单个 ASCII `.`；其后到所属 `</p>` 只能是按原嵌套顺序关闭的 enclosing tags 与 whitespace。prefix/link/period 中有 template action、隐藏/code/icon context、ellipsis、无 prose prefix、非句末后续内容、comment 分隔或非单个句号均不获权。identifier/domain/link label 与 href 本体不进入新 slot，也不会按标点拆分。

每个选中的 prefix 只新增其对应的 `role=sentence-punctuation`、`encoding=html-text` slot，source 为 `.`、protected bindings 为空。坐标来自 HTML tokenizer，context 使用 source DOM paragraph occurrence；不硬编码 offset、locale 或 target 文本。所有 Old slots（含已有 accessibility slash-prose successor 的 source/protection/ID/coordinates）必须 byte-exact 保留；未选中的句号不变。选择无对应句号、已经 editable 的句号、未知/重复 prefix、额外 slot 或任何旧 binding release 均拒绝。可分次选择其它 prefix，不能因此重编号已接受的 slots。

该 rule 的 `released_atoms=[]`：它不伪造旧 protected atom 的 release。新增 topology 完整绑定在既有 receipt `old` / `new`、request exact prefix IDs 与 prospective identity 中，不新增 schema/receipt family 或平行状态机。historical reader 以冻结 Old 和同一 raw HTML syntax 重建 exact additions；current reader 还执行完整 source/consumed owner/head/dependency proof。frozen codec 只允许这些 source-proven punctuation frames 与原有完整 slot set，仍逐 slot 验证 source round-trip、encoding、protected tokens、ID 与完整 structure identity。

例如自然语言 prose + `<a href="/wiki/">Go Wiki</a>.`，或 prose + technical domain/path link label + `</a>.`，可把 link 外的最后一个句号作为独立 editable input；Generation 之后才能在保持 source/HTML/identifier 的前提下输出目标语言标点。旧表示下的语言 PASS 不因此自动覆盖新增 slots；仍须 exact consumed predecessor → 正式 successor → finding-bound revision/validation → successor Surface。

## Head、history 与 lifecycle

implementation dependency 演进使用独立的 [Shared dependency re-attestation](SHARED_STRUCTURED_REPRESENTATION_REATTESTATION.md)。其 bootstrap 不依赖 stale base 的 currentness PASS，不能改变本 receipt 的 source/topology 或覆盖本 family head。

每个 `identity([package,asset_id])` 只有一个 keyed head：`data/site-shell/shared-representation-successors/current/<key>.json`，内容严格为单一 `Reference`。receipt 位于同 root 的 `receipts/<identity>.json`。不扫描 receipts/latest；orphan inert，数组/duplicate JSON/矛盾 head 拒绝。每个 receipt 绑定 exact predecessor Reference、完整 old/new ShellLanguageSource、source package、registry/source authority、parser/contract SHA/mode；lineage 连续、无 cycle、depth ≤32。

没有 head 时 current reader保留 frozen foundation。采用后只替换相同 registered owner 的 current Document；使用现有 `Document.dependencies` 中单一 `kind=contract`、canonical immutable shared receipt path/SHA 绑定 authority，不扩展混合职责的 `units.go` Document 声明。frozen Unit 数组、raw source 与 legacy source frame 提供精确 source/slot/role/encoding 重建，receipt 保存完整 old/new 表示；Bundle 运输 head 和完整 receipt lineage。重复、未知 kind、非 canonical receipt path 或额外 dependency 均拒绝。historical Generation 依据其 hash-bound Document、receipt 和 frozen raw source重建，不读取今天的 head、不重判过去的语言或 prose分类。current head 必须另外通过当前 source/registry/dependency和完整 semantic delta proof；dependency drift不能静默 carry，现有 Homepage dependency re-attestation 不接受本 family，更不能授权 topology change。

historical Bundle membership 与 current parser dependency 是不同 authority。consumed Generation 的 source 消费证明复用 frozen transport/carry-forward、registry exact-set、完整 output restoration、旧 glossary/review 与真实 provenance 校验，并精确匹配 predecessor owner/Document。它不授予语言 PASS/carry，因此不重放无关 Page/Surface baseline eligibility；原完整历史语言 reader 仍额外执行全部 assessment/closure 检查。未运输的 parser 文件没有所谓 historical Bundle SHA，不能借其它 family 的 checksum 当成它的 archive member。新 receipt 的 `currentness_inputs` 独立 hash-bind 今天的 parser/contract bytes/mode，包括消费证明 validator。`sharedRepresentationParserPaths` 不是 Homepage re-attestation 的 drift allowance，也不允许 mixed module 的任意历史变化。shared codecs 通过独立适配入口消费 receipt dependency；Homepage codecs、Document 声明与其原 drift gate 保留已绑定的 exact bytes。公共 source-loading guard处理跨 family pending 状态，不修改 Homepage parser authority。

旧 representation 下的语言 PASS 不能 carry 成新 representation 的判断。assessment 将 exact affected asset 置为 `generation-required`，不 carry 其旧 target slots；其他 shared assets 保持原 carry/context。后续复用既有 Generation → restoration/validation → package closure → independent successor Integrated Surface review，不改 target、不新增平行语言 lifecycle。source/corpus/glossary eligibility 仍独立；本 authority 不自动 refresh 它们。

## Formal Command Contract v3

formal command 的新 rule/input acceptance（包括 sentence-final period rule）同样使旧 verification 失效；首次真实使用前必须重新按 current docs → implementation → direct CLI tests 完成 [first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。本工程测试不授权任何真实 plan/inspection/apply。implementation/contract bytes 演进后，旧 representation/current re-attestation 不能直接视为新 topology 的 language authority，必须由后续正式阶段处理 currentness 与 successor lineage。所有 path 均为 repository-relative 普通文件，拒绝 absolute、traversal、symlink；SHA 指文件 exact bytes，不是 JSON identity。命令前缀为 `tour-i18n site-content`。

| 字段 | prospective | inspection | apply |
| --- | --- | --- | --- |
| COMMAND | `shared-structured-representation-successor` | 同左 | 同左 |
| OPERATION | readonly prospective proof | readonly reviewed-plan inspection | immutable receipt + keyed head publication |
| INPUT_KIND | request | plan | plan |
| INPUT_PATH_CONTRACT | repository-relative request | repository-relative plan | 相同 exact reviewed plan |
| REQUIRED_FLAGS | `--request`, `--request-sha256` | `--input`, `--input-sha256` | inspection flags + `--apply`, `--prior-identity`, `--prior-pointer-sha256` |
| FORBIDDEN_FLAGS | input/apply/CAS flags | request/apply/CAS flags | request flags |
| STDOUT_KIND | plain SharedRepresentationPlan (`schema=/plan`) | SharedRepresentationResult wrapper | 同 inspection |
| MUTATION_EXPECTATION | none | none | known；publication 不明为 unknown |
| IDEMPOTENCE | deterministic current proof | exact installed plan可返回 already_current | exact原 plan已 installed时 no-op；不造第二 receipt |
| CURRENTNESS_CHECK | fresh source/registry/consumed Generation/semantic/dependency/head guards | 重建 exact plan或验证 exact installed authority | 独立两次重建 + commit guards |
| PRIOR_OR_CAS | exact keyed head/absence observation | plan内 predecessor/head SHA | flags必须与plan exact match；absence=SHA256(empty) |
| SUCCESS_STOP | plan审阅；不自动落地 | inspection结束 | 发布结果后交还维护者，不进入语言阶段 |
| FAILURE_STOP | 无新 authority，停止 diagnosis | stale/错误 kind/hash/currentness即停止 | unknown保留 marker/lock，禁止 blind replay |
| RECOVERY_REPLAY | 无 mutation | 只检查原 exact plan | 原 head未变、无 pending/lock且exact orphan可经维护者明确授权重试；unknown先人工只读 diagnosis |

不接受 `--locale`、`--package`、`--source`、`--parser`、`--output`、positional args 或未知 flags；scope来自 exact request。request/plan/receipt不互换，不接受 normalized substitute input。

apply 使用 shared `data/.glossary-corpus.lock`。精确 plan SHA、prior representation identity 与 keyed head SHA全部 CAS。先 write-once content-addressed receipt并 fsync目录，再写/fsync head temp；pending marker `data/.shared-structured-representation-pending` 绑定 plan、head、prior SHA、result。再次 guards/CAS 后 atomic rename head、fsync并验证 installed bytes，最后清除 owned marker/lock。提交后 I/O/rename/durability不明返回 `mutation=unknown`，保留 marker/lock；无自动修复、覆盖、删除或 blind replay。其他 representation commands/current readers见 pending同样 fail closed。
