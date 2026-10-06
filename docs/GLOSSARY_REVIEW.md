# Glossary Review 规范

## Versionless locale authority（V2-D 收敛）

v1/v2/v3 是 implementation/content-package milestone，不是 locale language governance namespace。Content package 可以版本化；Locale language system 不版本化。全站只有 `locales/<locale>/glossary.yaml`，不得 Tour/Docs/v2 split。current executable migration、统一 corpus、一次完整 glossary generation/refresh、统一 structured assets、完整 Page TU 与 delta-aware integrated Surface contract 见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。本节是全站扩展入口；下文旧 Tour workflow/evidence 保持兼容，不授权重复初始化 locale 或改写历史证据。

已有 Tour-complete locale 首次扩展：一次 unified glossary refresh → 一次独立 full unified Review → compatibility → 一次 pending structured-assets Generation → 两批 Learn/Docs Page Generation / 两批 Page QC → 一次 integrated Surface Review。unchanged Tour/UI/meta/SEO carry；只有真实 affected scope reopen。新 locale 从零执行同一 locale campaign，glossary 与 structured-assets 各一次，Tour 和 Learn/Docs 各用其 surface-specific parser/TU batches；没有第二次“V2 locale init”。Course SEO 是 Page-derived consumer，按自己的 canonical source/ready target contract 单列，不强行并入 structured-assets；没有 Learn/Docs canonical SEO authority时不创建 SEO target。

新 unified receipt `go-learning/unified-glossary-review/v1` 与 old v1/Tour legacy coverage additive 共存。新 Generation 要求 current unified coverage；旧 completion仍按旧 evidence合法验证。unified current exact优先；多个 current unified receipts、glossary/corpus mismatch fail closed。旧 receipt immutable，不 retroactively upgrade corpus coverage。glossary bytes不变但 corpus变了仍需一次 full unified Review；不猜造 corpus compatibility shortcut。新 glossary byte change须 full unified PASS后才用 V2-B semantic delta/精确 lineage复用旧结果。

历史/current Tour completion 的 `RequireCurrentGlossaryReview` 优先接受唯一 current PASS unified review；well-formed FAILED 或 conflicting unified campaign receipts 不遮蔽仍 exact-valid 的旧 Tour formal/legacy proof。glossary bytes 真正改变后，旧 proof 不再匹配，不能 fallback carry。新工作使用 `RequireNewLanguageGlossaryReview` / `RequireUnifiedGlossaryReview`，仍严格拒绝 FAILED、conflicting 或 stale unified authority。malformed/schema/self-identity/unsafe-path/archived bundle 损坏属于证据完整性错误，两类 gate 均 fail closed，不以 legacy fallback 掩盖。


Glossary Review 是新 locale 在正式语言生成前对完整 `locales/<locale>/glossary.yaml` 执行的独立语言审核。它验证术语决策本身是否可作为全站 authority；TranslationUnit Quality Check 只验证 candidate 是否遵守 glossary，不能替代本阶段。

## 顺序与职责

新 locale 的正式顺序是：

```text
locale init
→ Generation session 制定完整 glossary
→ generation-independent Reviewer session 完整 Glossary Review
→ AI execution environment 记录并检查 current passed receipt
→ Generation session 一次 Unified Structured Assets（只输出 pending assets）
→ 同一 locale campaign 内 Tour TranslationUnits / Learn-Docs Pages generation
→ TranslationUnit Quality Check / promotion / Course SEO
→ Locale Surface Review
```

Reviewer session 必须从未参与该 locale 的 language generation。同一个符合此条件的长期 Reviewer session 可以依次承担 Glossary Review、TranslationUnit Quality Check / re-QC 与 Locale Surface Review；三个 gate 的范围和 evidence 始终独立。Reviewer 发现问题时只返回 finding，不自行修改 glossary 或生成 replacement；Generation session 修订后，原 Reviewer session 可以重新完整审核当前 glossary。

Reviewer 给出正式结论后，具备仓库终端能力的当前 AI execution environment 默认连续完成 reviewer bundle current-check、receipt record / check，以及 failed finding 回流后 glossary replacement 的必要落盘与再次导出。这里自动化的只是既有机械步骤；Reviewer 仍不得生成 replacement，Generation session 仍不得批准自己的 glossary，所有完整覆盖与 current gate 保持不变。locale init 与首次 bulk TranslationUnit ZIP handoff 仍由维护者 Local terminal 执行。

UI / article metadata 目前没有文件写入级 machine block，但正式生成必须发生在 Glossary Review PASS 后。若在途 locale 已提前生成这些资产，Glossary Review 直接 PASS 时可继续使用；若 glossary revision，Generation session 必须同步检查和修订受影响的 UI / metadata 后再继续。

本 gate 引入时的 `bn-BD` 属于上述在途状态：glossary、UI 与 article metadata 已由 Generation session 生成，但尚未执行 TranslationUnit export。不得为它补造 passed receipt；它是第一门必须完成正式 Glossary Review 的 locale。若当前 glossary PASS，既有 UI / metadata 可继续使用；若 glossary revision，必须先同步检查和修订受影响资产。

## 完整审核范围

不得抽样。Reviewer 必须完整读取当前 glossary 及制定术语所需的正式 English/source context，逐项审核 `mandatory`、`preferred`、`forbidden` 和 `keep`：

- 核心 Go / 编程术语在目标语言中准确、自然、稳定，并符合该语言技术社区通常表达；
- `mandatory` / `preferred` 分类合理；
- `forbidden` 只包含稳定错误表达，不误伤合理语境；
- `keep` 确实需要在可翻译自然语言中保持技术 identity；
- `A Tour of Go`、`Go Playground`、Run / Format / Reset 等全站高影响术语已有明确、一致的决定；
- 不机械翻译其他 locale glossary，不把 protector 已负责的技术 identity 复制成巨大 keep 表；
- 不新增 source 或项目语义没有依据的解释。

正式结论只有 `passed` / `failed`，不使用 TranslationUnit 的 A/B/C/D。`failed` 必须给出至少一条 finding；finding 回到 Generation session 产生 replacement。

Generation session 的完整输入默认由当前 AI execution environment 用 `generation-bundle locale-export --task glossary` 一次性导出；修订前用 `generation-bundle locale-check --bundle <zip>` 验证 current。产品界面需要人工附件上传时，只把实际 ZIP handoff 留给维护者。该 generation ZIP 与 Reviewer ZIP 不得混用：前者允许生成 glossary，后者明确只允许 findings / passed/failed。

独立 Reviewer 优先读取 deterministic、自包含 ZIP：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-review reviewer-bundle \
  --locale <locale> \
  --output /tmp/<locale>-glossary-reviewer.zip

go run -mod=readonly ./cmd/tour-i18n glossary-review reviewer-bundle-check \
  --locale <locale> \
  --bundle /tmp/<locale>-glossary-reviewer.zip
```

新 ZIP 包含完整 glossary、locale identity、完整 Unified Glossary Source Corpus、术语与审核 authority，以及 exact inventory/hash。Corpus 已机械覆盖 Tour、UI、metadata、canonical SEO、shell、Learn data 和 Learn/Docs Page；不再只使用 122 Tour TU。相同 working tree 重复导出必须 byte-stable；记录结论前必须运行 current-check，任何 glossary、source 或 authority 变化都使旧 ZIP fail closed。bundle 只是 transport，不是 receipt 或 review decision，Reviewer 不得在审核 session 生成 replacement。

## Receipt 与 current gate

审核完成后由当前 AI execution environment 默认记录机器 receipt；reviewer 不填写 glossary SHA：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-review record \
  --locale <locale> \
  --review-id <review-id> \
  --reviewer <reviewer> \
  --generation-session <generation-session> \
  --bundle /tmp/<locale>-glossary-reviewer.zip \
  --decision passed

go run -mod=readonly ./cmd/tour-i18n glossary-review record \
  --locale <locale> \
  --review-id <review-id> \
  --reviewer <reviewer> \
  --generation-session <generation-session> \
  --bundle /tmp/<locale>-glossary-reviewer.zip \
  --decision failed \
  --finding '<specific finding>'
```

新产物为 `data/unified-glossary-reviews/<locale>/<review-id>.review.json`，绑定完整 glossary path/SHA、corpus identity、Reviewer Bundle/input identity、独立 session、decision/findings、rubric 和可重算 self identity。Receipt/Reviewer ZIP/glossary archive 均不覆盖；新一轮审核使用新的 review-id。旧 `data/glossary-reviews/**` v1 receipt 与 fixed legacy coverage 继续只作为可信历史/Tour coverage，不改写、不冒称统一 corpus coverage。

检查 current gate：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-review check --locale <locale> --coverage unified
```

unified `check` 同时重新读取当前完整 glossary 和 corpus identity。missing、malformed、wrong locale、non-passed、rubric/path/hash/input mismatch、多个 current unified receipts 均 fail closed。exact current unified receipt 优先，旧 Tour-only receipt 不参与 ambiguity。Glossary byte change 或 corpus identity change 都使旧 unified receipt stale。默认 `--coverage tour` 保留旧 Tour coverage 检查；聊天记录不是 evidence。

所有首次及后续正式 `retranslation export` 都调用同一 gate；没有 current coverage 时不会创建 batch 或其他 export artifact。Glossary Review PASS 只证明当前 glossary 已完成独立审核，不证明任何 UI、metadata、TranslationUnit candidate 或 Course SEO 已通过审核。

## 已上线 locale 的一次性 migration

本 gate 引入时，仓库从 `production/identity.json` 自动确认了 18 个 `production_state=live` locale，并逐个确认当前 glossary SHA 已存在于真实 `decision=passed` 的历史 Locale Surface Review A-gate。它们不重新执行 glossary-only review，也不把历史 Surface Review 冒充新的 Glossary Review。

固定 migration authority 是 `data/glossary-review-legacy-coverage.json`。其中 18 项是 2026-09-19 migration 时冻结的原始集合，不是 production 当前或未来 live locale 总数。每项精确绑定 locale、glossary path/SHA、历史 Surface Review review-id、gate path 与 gate 文件 SHA；未来通过正式 Glossary Review 上线的 locale 不加入该集合。

运行时对固定 JSON 本身执行 strict parse，并全局校验 schema/evidence/stage、重复 locale 与各 entry 的静态 identity 格式；只有当前被检查 locale 确实位于该集合时，才读取并验证它自己绑定的历史 gate。无关 legacy locale 的历史 gate 不参与当前 locale 的 check。Glossary 任意字节变化或本 locale 引用的历史 gate identity 变化都会使对应 legacy coverage 失效；之后必须执行正式 Glossary Review。

## 与后续 gate 的关系

Glossary Review 不替代 automatic validation、Candidate Snapshot、A-only TranslationUnit Quality Check、machine finalization、promotion、Course SEO schema v2、Locale Surface Review、preview machine acceptance、visual HUMAN gate 或 Production gate。

Glossary 变化前必须按 [Glossary Compatibility](GLOSSARY_COMPATIBILITY.md) 归档当前 exact bytes。任意 byte change 后仍须完整独立 Glossary Review CURRENT PASS；该 PASS 本身不会恢复旧 A。之后只有可验证的逐 scope compatibility lineage 才允许未受影响 A carry-forward，受影响或无法证明的 scope 仍 stale。最终 Locale Surface Review 的完整语言质量范围不变。

若 glossary 变化发生在 candidate generation 之后，current PASS 后按 [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md#glossary-induced-protected-input-stale-recovery) 比较 current protected input 与 latest saved input，只重新生成真实 drift Unit。恢复后的 Snapshot 仍完整；旧 A 只有 exact identity + verified compatibility lineage 才继承，其他 Unit 必须正式 QC。
