# Structured Generation 语言语义 authority

本文件与 `TRANSLATION_TERMINOLOGY.md`、`TERMINOLOGY_GUIDE.md` 是 `structured-language/v1` 的语言语义依赖。修改语言生成约束时必须修改这些正式语义输入；AGENTS、交接、provider 执行手册与 Site workflow 中的执行者、工具调用、命令和运输说明不能覆盖本 contract。它仅拆分既有 structured language / execution authority，不新增语言质量 gate、状态机或另一套 glossary。

一个 locale 只有一份完整 glossary。Generation 完整读取 current Bundle 的 manifest、Plan、所有 source / Document / slots / protected bindings、当前完整 glossary、locale identity、Unified Glossary Source Corpus 与完整 CURRENT independent Unified Glossary Review。不得抽样、拼接旧输入或用聊天记忆替代 Bundle。mandatory / preferred / forbidden / keep 的语义以正式术语规范为准；原始 source 是内容语义 authority，glossary 是术语 authority。

只输出 `Plan.Expected` 精确集合。每份 structured source 输出完整 target slot exact set；slot 是重建坐标，不是额外 TU。输出保持 UTF-8、LF、恰好一个 EOF LF、strict JSON，无额外解释或 fence；`bundle_identity_sha256` 必须绑定实际模型输入 Bundle identity。不得输出 carry/context-only 资产，不得新增、缺少、重复或改名 target/slot。

若 `StructuredAssessment.structured_source_successor` 绑定正式 source-owned content expansion，`StructuredAsset.carried_slots` 是机器冻结的 accepted target，不能由 Generation 输出、重译或修改。该资产的输出 exact set 仅为完整 `Document.Units` 减去这些 carried IDs；机器在 import 合并后仍要求完整 slot set、protected structure 和 glossary validation。模型必须读取完整上下文，不能把旧槽位当缺失，也不能扩大新增 scope。没有该 exact authority 时仍要求原完整输出合同；source/CAS/currentness 见[Source content adoption](SOURCE_CONTENT_ADOPTION.md)。

目标语言必须忠实于 source，保持事实、技术语义、条件、身份与 source 提供的范围，不添加解释或保证。protected token exact membership、配对、链接/模板/代码/机器身份、非语言 bytes 与 required structure 必须按冻结表示保留；允许的自然语言语序、标点、teaching comment 翻译与 framing 由正式 source adapter 和当前 canonical validator 判断。Tour UI/article metadata 仍使用既有 schema validator；其他 structured assets 使用正式 parser 重建，不能用任意 JSON/文件绕过保护。

revision 保留独立 finding、当前精确 predecessor、原 source 与 glossary 绑定，仅处理 finding 授权的 exact scope；不得以重新翻译全部内容替代精确修订。Generation session 负责 replacement，独立 Reviewer session 不生成自己的 replacement 后再批准。模型真实 provenance 使用 `chatgpt|codex`、`gpt-5.6-sol-high`、原 Generation session 与真实 RFC3339 UTC generated_at；不能用导入/恢复时间替换生成时间，不能伪造 model invocation。

validation 不等于 Independent QC、A-only 或 Surface Review。structured assets 不另建逐 TU QC；后续 package closure、affected Surface re-review、Production 与 HUMAN gate 继续按现有 authority 执行。Homepage Page TU 与普通 TU 生命周期独立，不能用 structured evidence 替代其审核或 current authorization。

execution/tooling 文档仍随 Bundle 完整运输并受 transport inventory/hash 保护，但 `structured-language/v1` currentness 比较其余正式输入与上述语义依赖，不因执行说明变化强迫重新 Generation。语义 authority 任意 byte drift、source/Document/slot/protection/glossary/locale/Review/Expected drift 均 fail closed。legacy 未分类 Bundle 不能自动降级为新 profile：只接受实现中审计过的 exact legacy authority SHA → exact semantic authority SHA 迁移；未知历史版本必须停止并重新评估，不能把任意 docs change 当兼容。
