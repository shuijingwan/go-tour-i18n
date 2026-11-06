# Structured Generation transport 合同

本合同规定 transport，不改变 [Structured Generation 语言语义 authority](STRUCTURED_GENERATION_SEMANTICS.md)、正式 source、glossary、Unified Review、Expected scope 或任何语言质量 gate。`go-learning/locale-structured-assets/v1` manifest additive 使用 `transport_profile=go-learning/structured-generation-projection/v1` 和 `validated_repository_inputs`；Bundle identity 绑定这些字段和完整 payload inventory，因此新旧 transport identity 可以不同，不能改写真实模型输出冒充新输入。

## Generation 自包含与 machine provenance 的边界

Generation 必须完整读取 Bundle manifest/Plan、当前完整 corpus/glossary/locale/Unified Review receipt、source/Document/slots/protected bindings、affected asset 的 previous accepted targets / baseline glossary、carried slots、source replacement/adoption/reopen/closure/Surface receipts 和正式语义/执行说明。只能输出 `Plan.Expected` exact set；carry/context-only 仍不进入 output scope。一个 source-owned replacement 不授权重译其它 carry assets。

只允许将 exporter 已完整验证的以下两类 archive ZIP 从 payload 投影为 exact repository-relative path + **原始 archive bytes SHA256**：

- `data/locale-language/<locale>/structured-bundles/<sha256>.zip`：历史 Generation transport，用于冻结 source/output/glossary/lineage 的 machine reconstruction。
- `data/unified-glossary-reviews/<locale>-bundles/<sha256>.zip`：历史完整 Unified Review transport，用于 machine Review/corpus/glossary/provenance 验证。

这不是按 `.zip` 后缀过滤，也不是任意 dependency omission。只投影经当前历史 Generation/Review reader 返回的 exact archive set；其它 payload 原样保留。Generation 不需要递归打开这些历史 archive，但 receipts/Plan 仍绑定其身份。当前完整语言输入仍在 ZIP 内；**offline machine currentness 不由 transport 单独证明**。

exporter 仍从 repository 读取、SHA/mode/type/currentness 校验这些 archive，验证历史 lineage 和 current authority，并在返回前复验 invocation dependency graph。`validated_repository_inputs` 必须排序、唯一、path/SHA exact，不能与 payload 重复。current-check 先执行原 transport structural/inventory/32 MiB gate，再独立重建当前 exporter 结果，比较完整 Plan、semantic payload、Expected 和 exact reference set；缺失、追加、改 SHA、错误 authority、source/Review drift 均 fail closed。historical generation reader 也重新绑定这些引用。引用不是 PASS cache，不跨 invocation 复用 disk observation。

旧 immutable transport 仍按原 schema 读取，不重写。合法且未超过原限制的旧 full-payload transport，只能在全部 semantic bytes/Plan/Expected 一致、每个被投影 archive 的原 inventory SHA 与 current exact reference 一致时，通过明确的 legacy→projection 比较；未知 profile / transition 拒绝。旧 oversized ZIP 仍在 32 MiB gate 失败，不能用 projection 绕过其输入检查。实际尚未 Generation 的失败 export 应保留，后续由维护者导出新的 transport。

## Formal command contract / first-real-use

finding-bound revision 的 owner eligibility 统一见 [Integrated Surface finding authority](INTEGRATED_SURFACE_REVIEW.md#mandatory-shared-composition-finding-owner)：mandatory shared composition 可以由其原 structured owner 精确寻址，不借用 Homepage TU owner；只有一个 matching package/Document asset 进入 Expected。此 accepted scope/currentness 修改同样触发下面的首次真实使用 gate。

mixed FAILED receipt 按 finding class 分流，见上述 authority 的 [exact revision contract](INTEGRATED_SURFACE_REVIEW.md#shared-representation-successor-的-exact-revision-authority)。每个 structured finding 选择一个 exact owner/output；多个合法 structured findings 可共同进入 Expected，TU/integration findings 保留在各自 lifecycle。representation predecessor proof 只给合法 shared successor owner 例外，其它 finding 不因共存而获得例外或被豁免 currentness；完整 receipt/native artifacts、scope 与 target proof 保持不变。

历史 FAILED structured finding 遇到正式 shared representation successor 时，使用该合同定义的 **exact predecessor proof**，而非删除全 scope currentness 检查。完整入口先验证原 receipt/Markdown/independent session，再验证 current head / dependency re-attestation、raw source、old/new representation lineage、同一 predecessor target，以及所有其它 scope authority；仅 exact structured finding owner 的 representation delta 可暂缓 Generation eligibility。普通 current scope 仍可因缺 fresh Generation 拒绝，历史 judgment 不改写、不 carry 旧 A。global Plan 的其它 independently authorized pending assets 不进入本次 revision Expected，也不被标记已完成。该 accepted currentness 扩展触发新的 first-real-use gate，不能复用旧 verification。

这是 modified transport/current-check contract。首次真实再使用前，必须按 [Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3) 顺序核对本合同、current implementation、direct CLI/contract tests；命令名不代替 operation/input-kind。

| 字段 | structured-export | structured-check |
| --- | --- | --- |
| COMMAND | `site-content structured-export` | `site-content structured-check` |
| OPERATION | export | transport current-check |
| INPUT_KIND | current locale Plan；initial 或显式 finding-bound revision | structured Generation transport ZIP；不是 Plan/receipt JSON |
| REQUIRED_FLAGS | `--locale <locale> --task initial --output <new.zip>`；revision 使用 `--task revision --finding <receipt>` | `--bundle <bundle.zip>` |
| FORBIDDEN_FLAGS | 不接受 apply/record/import flags；task/finding 必须匹配 | 不接受 locale/task/finding/output/apply/record |
| INPUT_PATH_CONTRACT | output/finding 是 repository-relative path | bundle 是 repository-relative path |
| STDOUT_KIND | 无新 JSON result；成功写入指定 transport | 成功无 stdout；失败非零并输出错误 |
| MUTATION_EXPECTATION | `known`，只写指定 transport，不写 formal language authority | `none` |
| IDEMPOTENCE | 相同 path+exact bytes 允许；不同 bytes 拒绝覆盖 | readonly；每次独立检查 |
| CURRENTNESS_CHECK | source/corpus/glossary/完整 Review、historical/current graph、exact dependency final guard；revision finding 必须 exact current scope 或合法 shared representation exact predecessor proof | transport size/shape/inventory 先行，再当前独立 exporter 重建；明确 execution-only exception 不放宽语义或 archive hashes |
| PRIOR_OR_CAS | 无 formal state CAS；destination absent 或 byte-identical | none |
| SUCCESS_STOP | 仅完成 transport export；未 Generation/import/closure | current-check 后停止；不是语言通过或 import |
| FAILURE_STOP | 保留旧 transport、全部 authority；不 import、不扩大 scope、不提高 limit | 保留失败 transport；不 fallback、不 import |
| RECOVERY / REPLAY | 检查 exact failure/current contract，再由维护者使用新 output path；不覆盖旧 bytes | 修复 engineering/authority 后重新 readonly check；不修 artifact bytes |

合法 path 示例：`evidence/locale-structured-v2.zip`；非法：`/home/user/repo/evidence/locale-structured-v2.zip`，即使 absolute path 位于仓库内。API engineering proof 可把返回 bytes 写入 `/tmp`，不是该 CLI 的 repository-relative landing contract。

同一只读 invocation 可以显式共享既有 `structuredSuccessorValidation` graph；每次访问仍读取 physical dependency、校验 exact bytes/mode/SHA，结束前 aggregate verify。没有 persistent/global/LRU cache，不改变 publication/CAS/default eager callers。

## Language policy/current validation

当前 Bundle 必须运输完整 locale metadata（含 versioned language policy）与 [language audit contract](LANGUAGE_DEFECT_EXHAUSTION.md#正式接入合同首次真实使用前仍须-command-contract-gate)。structured output/current Generation 验证完整重建后的 visible representation；只有 deterministic defect 阻断。review candidates 不构成 validation failure，而进入既有 Surface Reviewer transport。historical frozen decoder 不获得 current eligibility，不能以旧 A 或 frozen slot 边界豁免当前 gate。
