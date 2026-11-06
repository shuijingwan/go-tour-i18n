# Integrated Surface Independent Reviewer transport

本合同属于 `go-learning/integrated-locale-surface/v1`，复用 [Site v2 Workflow](SITE_V2_WORKFLOW.md#locale-plan--integrated-delta-aware-surface--activation) 的语言判断与 existing `IntegratedSurfaceReceipt`。Bundle 是确定性 transport，不是新 receipt、语言 authority、质量结论或 activation 授权。Generation / Reviewer 角色隔离、GPT-5.6 Sol High 独立 Reviewer、A-only、currentness、provenance、machine validation、Surface、Production 与 HUMAN gate 均不改变。

## 唯一输入及分工

Maintainer Local terminal 基于 locale + prospective closure(s) 或 canonical current scope 导出及 current-check。标准命令矩阵见 [workflow](SITE_V2_WORKFLOW.md#integrated-surface-reviewer-bundle)。完整 current 附件满足 semantic 输入要求，Independent Reviewer 的 semantic RDC calls = 0。不得以旧 `surface-review reviewer-bundle` 代替本 transport。

Reviewer 首先检查 manifest schema/kind/lifecycle、scope identity/SHA、全部 inventory SHA/size、fresh exact-set、carried counts、glossary/Unified Review、source/target 及 composition context。缺项、损坏、identity 不符时返回 `INPUT_GATE=FAILED`、`LANGUAGE_JUDGMENT=NOT_PERFORMED`、`FORMAL_SURFACE_FINDINGS=NONE`；这不是语言失败，不生成 language-failed receipt、reopen、revision authorization 或 replacement。修复输入后使用新的 invocation/review_id。

## Bundle 内容与阅读顺序

1. `manifest.json` 与 `integrated-surface-scope.json`：scope 是 current canonical bytes；manifest 绑定 lifecycle、locale、scope SHA/identity、closures、glossary/Review、runtime refs、fresh/carried counts 和完整文件 inventory。
2. `fresh_review_required`：每个 entry 精确对应 scope 的一个 `review_required` asset，包含 exact asset ID/package/state、完整 canonical source 与 restored target、各自文件 SHA、source representation 及适用的 Snapshot/candidate/validation/finalization 或 structured Generation provenance。不得只读 path/SHA 代替正文。
3. `context_only_carried`：prospective package 中已 carry 的完整 source/target，供跨页面、导航、Homepage 与 Learn/Docs 比较。scope 仍保留全部 carried inventory；该 subset 的长度不代替 scope 的 carried count，不把 carried rows 重新升级为 fresh review。
4. `integration/` 与 `evidence/`：当前完整 glossary、独立 Unified Review及其 immutable archive、existing coverage、canonical route exact set、runtime/public identity 与 public first-party composition source。存在 Tour completed coverage 时，`integration/context-only-tour.json` 复用既有完整 Tour source/target、UI、article/Course SEO 与 first-party runtime/template context，仅供 integration comparison。该 context 的 legacy kind 不改变本 Bundle 的 Integrated lifecycle，也不要求重新完成 legacy Stage A/B/C 或重新给 carried TU rating。
5. `authority/`：本合同与既有术语规范。正常 execution/handoff docs 不新增为语言 semantic dependency。原始模型 Bundle refs 保留为 provenance metadata；供语言判断的完整 source、restored target、representation 与 generation record 已在本 ZIP 内，不要求 Reviewer 经 RDC 读取模型 archive。
6. `independent-surface-result.template.json`：使用原 receipt schema 与精确 scope；空 decision、review_id、reviewer_session、Markdown path/SHA 必须由真实独立审核与落地过程填写。模板不预填 PASS，不虚构 invocation 或 generation。

注意 scope asset 的 `source_sha256` 可以是 package/source projection identity；payload `source.sha256` 则绑定完整 concrete source bytes。二者用途不同，不可手改 scope 使二者看似相同。source representation 与已验证 closure 证明它们的正式关系。

`integration/effective-content-scope.json` 是 current checker 应用 source adoption 后的完整 registry；存在 shell adoption 时，Bundle 同时包含 selector、immutable adoption 与 source foundation。它解释历史 `data/site-content-scope.json` 与当前 Homepage/shared-shell inventory 的关系。existing coverage 与 prospective closures/route set 分别呈现，不能把待 activation 的 coverage 冒充已 activation。

## 语言判断

逐一完整审核 fresh exact assets 的 source ↔ target 忠实性、中文或当前 locale 自然度、glossary、链接/markup/代码边界及实际组合一致性。`context-review-required` Homepage 必须同时读取完整 Homepage source/target、shared template、menus/resources/testimonials 等 shared structured context，以及 canonical/coverage/runtime/public/ads integration source。已有 Homepage Page A 仍是独立的 TU 判断；Integrated Surface 判断组合上下文，不伪造一次新的 A。

使用 carried context 检查跨 surface terminology、navigation、Homepage/shell 与当前/新增 Learn/Docs 的一致性，以及 coverage-aware routing、canonical/sitemap/public identity 和广告 integration 的实际 source composition。这里只审核输入所支持的语言与 integration contract，不声称已执行 browser、preview、Production 或 HUMAN acceptance。

carried 不重新逐项 rating。若组合检查确实发现 carried TU 缺陷，必须使用 scope 中原始 exact `candidate_artifacts`，按既有 `translation-unit-language` finding → shared reopen/revision/re-QC 路径处理；不得把输入失败伪装成语言 finding，也不得由 Reviewer 自行生成 replacement 后批准。

有真实未解决问题时，原 `decision=failed` / `findings` / `exact_findings` 合同照常执行；通过时 `decision=passed` 且 findings/exact_findings 为空。正式 Markdown evidence 必须来自真实 Independent Reviewer。Maintainer 落地后，`site-content surface-record --input ...` 继续重建 current scope，原样验证 session、scope、exact findings 与 Markdown SHA。Bundle-check PASS 从不代表语言 PASS。

## Mandatory shared composition finding owner

fresh `context-review-required` Page 的组合判断包含同一 prospective package closure 下的正式 shared structured assets。当前 shell representation 注册的 shared template、menus、resources、testimonials 是这一关系的成员；身份沿用原文件路径和 package，不把 finding 改挂到 Homepage TU，也不新增第二份 structured asset。

recorder 与后续 finding reader 使用同一 owner resolver：fresh Page 必须带 exact `candidate_artifacts` 并绑定该 prospective closure；被指向的 shared asset 必须已在该 scope 的 carried inventory 中，绑定相同 closure 和 exact target SHA，且不带 TU candidate artifacts/UnitID。closure 的既有 historical structured reader 必须证明 accepted target、继承或直接的 Generation owner、冻结 Document 和完整输出。只有 registered `ShellExtractionContract` 的 `data`/非 Page 表示才提供 shared structured owner；普通 structured prose、ordinary carried context、runtime source 或任意 Bundle 文件不因此获得 finding authority。此关系不新增 scope 字段、不改 fresh/carried counts、不重写旧 scope/receipt，也不把 carried 资产重新逐项 rating。

`structured-surface-language`（兼容 legacy empty class）finding 继续要求 artifacts 为空，owner 的 TU `candidate_artifacts` 为空。finding ID 保持实际 shared 文件路径；`structured-export --task revision` 在 current Plan 中按 owner package + Document path 选择唯一 asset，仅该 asset 的 `TargetPath` 进入 Expected，其余 context-only。歧义、duplicate/unknown finding、owner/target/closure 漂移、伪装成 structured 的 Homepage TU 一律拒绝。`translation-unit-language` 的 exact artifact/reopen/re-QC 与 `surface-integration` 的 bounded engineering 路径保持不变。

### Shared representation successor 的 exact revision authority

正式 shared structured representation successor 采用后，普通 current scope/closure gate 仍要求该 asset 的 fresh Generation；不能在 Generation 前要求先取得新的 Surface judgment。已记录的 FAILED judgment 保持 frozen，仅可通过以下 predecessor 证明继续授权 exact revision，不把旧 scope 宣称为 current。

此路径只向显式 `structured-surface-language`、空 artifacts、无 TU identity/candidate artifacts 的 exact mandatory shared structured owner 提供 representation 例外；每个 finding 必须唯一解析 owner。原 receipt identity/path、独立 reviewer session、Markdown SHA 和 finding 均按原 reader 验证；owner 继续来自同一 frozen closure/Generation 与 Homepage composition anchor，不产生第二套 asset identity。

同一合法 FAILED receipt 可以混合 TU、structured 与 integration findings。`representationRevisionScope` 只为需要 shared representation successor 的合法 structured owners 提供 exact predecessor proof，不因共存的 `translation-unit-language` / `surface-integration` finding 拒绝整个 judgment，也不将它们转换成 structured authority；它们的原 finding、artifacts 与各自 lifecycle 完整保留。普通 structured finding 不获 representation 例外，仍须通过完整 scope 重建中的原 current/compatible 校验；最终 structured selector 只选择 structured findings 对应的 exact outputs。malformed structured finding、unknown class/owner、错误 package/lineage/target 仍 fail closed；必须至少存在一个合法 shared representation owner，不能凭 TU-only、integration-only 或普通 structured finding 获得该例外。全 receipt/native TU validation、registered shared composition 与整个 frozen scope equality 均不省略。

reader 必须以当前注册的 package + asset ID + Document path，沿 exact current shared representation head 验证 frozen Old → New Document、raw source SHA、released slot/binding lineage 和当前 dependency re-attestation（若需要）。允许 Unit ID 随 protected topology 改变，不用文本匹配或旧 Unit ID 相等代替 lineage。历史 accepted target 必须仍等于同一 frozen Generation/closure 的 target bytes 和当前 predecessor localized target；successor 不授权 target mutation。

revision 专用 predecessor projection 复用完整当前 Page A-only、package/source/membership、registry、structured owner、glossary/Unified Review、retained coverage/Tour、runtime/routes 检查。仅 exact finding owner 可暂缓由上述 shared representation lineage 引起的 Generation eligibility；同 closure 的其它 structured assets 必须相对于该 closure 的历史 owner 仍 compatible。重建的整个 predecessor closure 和 scope 必须与 frozen judgment **完全相等**。任何其它漂移一律拒绝。该 projection 不作为 current closure/scope PASS 返回、不进入 current-eligibility cache，也不能用于 record、activation 或 carry 旧 A。

此扩展复用 `go-learning/integrated-locale-surface/v1` finding 与 existing shared representation successor / dependency re-attestation authority，不新增 receipt/schema/状态机。Homepage Page/TU successor、ordinary stale structured prose、无 shared successor 的 stale scope 均不能使用此路径。

完整 Page A-only proof 中的 sealed checkpoint 按 [历史 representation context contract](CURRENT_TRANSLATION_AUTHORIZATION.md#checkpoint-中的历史-representation-context) 验证当时的 pointer/immutable receipt 绑定。合法 re-attestation 替换 current pointer 不等于原历史 evidence 丢失；只有 immutable exact lineage 能证明原 pointer SHA 时才能只读消费，不能用当前 successor 冒充原 evidence，也不能省略当前 eligibility 检查。

exporter 通过上述完整 finding validation 后，`previous/` payload 从同一 exact reviewed closure 的 frozen Generation 解码。`TargetPath` 是 structured output 地址，accepted bytes 由 Generation/closure hash 绑定，不要求同名文件落地；若存在不同的物理覆盖 bytes 也必须拒绝。这里不能再要求 predecessor closure 按新 representation 具有 fresh Generation eligibility，否则会在合法 finding selection 后重新形成循环。historical decoder 只提供 predecessor bytes，不授予 current closure PASS。若工程修复改变 installed dependency re-attestation 已绑定的 implementation bytes，原 re-attestation 必须继续 stale；真实 export 前先按既有 dependency re-attestation 合同完成新审计，不能改写旧 receipt/head 或自动接受新 bytes。

global current Plan 可同时含其它 independently authorized pending work。revision 仍只将 finding owner 的唯一 `TargetPath` 纳入 Expected；其它 pending assets 仅在 transport 中作为 context，global state 与其既有 source/Generation obligations 不变。后续按正常 Generation → validation / 独立语言审核 → finalization / package closure → successor Integrated Surface 继续，历史 finding 不产生新 Reviewer A、Surface PASS 或语言 carry。

这是 `structured-export --task revision` 的 accepted finding currentness 扩展，属于 modified formal CLI contract；第一次真实 export 必须重新执行 docs → implementation → direct contract tests gate，使用新 output path。普通 `surface-record` / scope / closure gate 不变。

## Finding command contract / first-real-use

这是 `surface-record` accepted finding scope 及 structured revision owner/currentness 的 modified formal command contract。首次真实 record/export 前，必须按 [Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3)重新顺序核对 current docs、implementation、direct CLI/contract tests；工程完成本身不授权真实执行。原 Reviewer Markdown/result 保持原 bytes，不创建 normalized substitute input。

| 字段 | surface-record | structured-export revision |
| --- | --- | --- |
| COMMAND / OPERATION | `site-content surface-record` / record independent judgment | `site-content structured-export --task revision` / exact finding-bound transport export |
| INPUT_KIND | 原 Independent Reviewer result JSON，空 receipt identity，原 scope、decision、findings、Markdown path/SHA | 已正式 recorded 的 FAILED Integrated Surface receipt；不是未记录的 Reviewer result |
| REQUIRED_FLAGS / FORBIDDEN_FLAGS | 仅 `--input <result.json>`；不接受 locale、finding、output、apply/record flags | `--locale <locale> --task revision --finding <receipt.json> --output <new.zip>`；不接受 apply/record |
| INPUT_PATH_CONTRACT | input、Markdown 与 scope refs 均为 repository-relative safe paths | finding/output 均为 repository-relative safe paths |
| STDOUT_KIND | 成功为新 receipt repository-relative path 的 JSON string；失败非零 | 成功写指定 ZIP，无新 JSON result；失败非零 |
| MUTATION_EXPECTATION | known：仅 immutable Surface receipt；不改原输入/evidence/closure/语言 artifacts | known：仅指定 transport ZIP；不 record、不 Generation/import |
| IDEMPOTENCE | 同 path+exact bytes 允许，不同 bytes 拒绝覆盖；每次完整检查 | 同 path+exact bytes 允许，不同 bytes 拒绝覆盖 |
| CURRENTNESS_CHECK | 重建 exact current scope；统一 owner/class/duplicate/artifact 验证；独立 session 与 Markdown SHA | 重验 receipt/Markdown/session 与 exact current scope，或上文 shared representation successor 的完整 exact predecessor proof；current Plan 中唯一 exact owner；其余 transport currentness 见 [structured contract](STRUCTURED_GENERATION_TRANSPORT.md) |
| PRIOR_OR_CAS | 无 prior/CAS flag；destination absent 或 byte-identical | 无 formal state CAS；destination absent 或 byte-identical |
| SUCCESS_STOP | 仅 recorded 独立判断，不自动 revision/export/activation | 仅 exact revision transport；未 Generation/import/closure |
| FAILURE_STOP / RECOVERY | 保留原 result/Markdown 与全部旧 artifacts；先修 exact blocker，不改 finding ID/class 或输入 | 保留原 receipt/transport；不改 owner、不扩大 Expected、不 fallback 到 TU |

## 已记录 finding 的恢复路径

`structured-surface-language` finding（兼容 legacy empty class）只有在 finding artifacts 和 owner `candidate_artifacts` 均为空时，才可进入既有 structured revision；`translation-unit-language` finding 使用 exact TU artifacts 进入共享 reopen/revision/re-QC。`surface-integration` 的 runtime、routing 或 public composition 缺陷由 repository engineering 在 recorded finding 的 exact predicate 和 bounded scope 内修复；即使 fresh anchor 是 Homepage Page/TU，也不因此授权 structured replacement 或 TU 译文修改。不得改写 finding class、receipt、Markdown evidence 或旧语言 judgment 来适配 exporter。

工程修复通过 targeted tests 后，由 Maintainer Local terminal 使用既有入口重建 successor Integrated Surface scope，继续 `surface-reviewer-bundle` → `surface-reviewer-bundle-check` → 新 invocation/review_id 的独立 successor review。scope/Bundle 必须按当前 runtime/public composition 与仍 current 的 package closures 重建，不手改冻结的 scope；复审完整 fresh exact-set 及该 finding 的组合关系，carried 仍仅作 context。原 FAILED receipt/evidence 保持 immutable，只有真实独立复审后的新 receipt 才能提供后续 gate 所需的 Surface 判断；工程测试本身不关闭正式 finding、不创建语言 Generation/QC/reopen，也不替代 Surface、activation 或其他 gate。此恢复路径复用现有 lifecycle/CLI，不增加平行语言状态机。

## Language quality attention transport

当前 Integrated Surface Reviewer Bundle 在既有 file inventory 中包含 `machine/language-review-candidates.json`，合同以 [统一 language audit](LANGUAGE_DEFECT_EXHAUSTION.md#正式接入合同首次真实使用前仍须-command-contract-gate) 为准。fresh 与 carried 都须通过 current deterministic gate；原 scope/receipt/finding schema 和 fresh/carried 角色不变。候选按 exact owner/可用 TU identity 定位，只作 `review-candidate` attention，不能自动 FAILED 或生成 finding。完整 source/target、Tour context、glossary 与真实 composition 仍必须审核。新 validator 暴露历史确定 defect 时停止 carry，走既有 recovery；不修改旧 judgment。
