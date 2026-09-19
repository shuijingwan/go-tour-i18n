# Glossary Review 规范

Glossary Review 是新 locale 在正式语言生成前对完整 `locales/<locale>/glossary.yaml` 执行的独立语言审核。它验证术语决策本身是否可作为全站 authority；TranslationUnit Quality Check 只验证 candidate 是否遵守 glossary，不能替代本阶段。

## 顺序与职责

新 locale 的正式顺序是：

```text
locale init
→ Generation session 制定完整 glossary
→ generation-independent Reviewer session 完整 Glossary Review
→ Local terminal 记录并检查 current passed receipt
→ Generation session 生成 UI / article metadata
→ TranslationUnit generation
→ TranslationUnit Quality Check / promotion / Course SEO
→ Locale Surface Review
```

Reviewer session 必须从未参与该 locale 的 language generation。同一个符合此条件的长期 Reviewer session 可以依次承担 Glossary Review、TranslationUnit Quality Check / re-QC 与 Locale Surface Review；三个 gate 的范围和 evidence 始终独立。Reviewer 发现问题时只返回 finding，不自行修改 glossary 或生成 replacement；Generation session 修订后，原 Reviewer session 可以重新完整审核当前 glossary。

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

## Receipt 与 current gate

审核完成后由 Local terminal 记录机器 receipt；reviewer 不填写 glossary SHA：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-review record \
  --locale <locale> \
  --review-id <review-id> \
  --reviewer <reviewer> \
  --decision passed

go run -mod=readonly ./cmd/tour-i18n glossary-review record \
  --locale <locale> \
  --review-id <review-id> \
  --reviewer <reviewer> \
  --decision failed \
  --finding '<specific finding>'
```

产物为 `data/glossary-reviews/<locale>/<review-id>.review.json`，绑定 `schema_version`、`evidence_type`、`locale`、`review_id`、`stage`、`decision`、`reviewer`、`rubric = glossary-review/v1`、规范 glossary path 和 CLI 从当前正式文件计算的 SHA-256。Receipt 不覆盖；新一轮审核使用新的 review-id。

检查 current gate：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-review check --locale <locale>
```

`check` 重新读取当前 glossary。missing、malformed、wrong locale、non-passed、schema/evidence/stage 或 rubric mismatch、glossary path/SHA mismatch、重复 current receipt，或正式 receipt 与 legacy coverage 同时成为 current authority，都 fail closed。Glossary 任意字节变化会使旧 receipt stale。聊天记录不是 evidence。

所有首次及后续正式 `retranslation export` 都调用同一 gate；没有 current coverage 时不会创建 batch 或其他 export artifact。Glossary Review PASS 只证明当前 glossary 已完成独立审核，不证明任何 UI、metadata、TranslationUnit candidate 或 Course SEO 已通过审核。

## 已上线 locale 的一次性 migration

本 gate 引入时，仓库从 `production/identity.json` 自动确认了 18 个 `production_state=live` locale，并逐个确认当前 glossary SHA 已存在于真实 `decision=passed` 的历史 Locale Surface Review A-gate。它们不重新执行 glossary-only review，也不把历史 Surface Review 冒充新的 Glossary Review。

固定 migration authority 是 `data/glossary-review-legacy-coverage.json`。其中 18 项是 2026-09-19 migration 时冻结的原始集合，不是 production 当前或未来 live locale 总数。每项精确绑定 locale、glossary path/SHA、历史 Surface Review review-id、gate path 与 gate 文件 SHA；未来通过正式 Glossary Review 上线的 locale 不加入该集合。

运行时对固定 JSON 本身执行 strict parse，并全局校验 schema/evidence/stage、重复 locale 与各 entry 的静态 identity 格式；只有当前被检查 locale 确实位于该集合时，才读取并验证它自己绑定的历史 gate。无关 legacy locale 的历史 gate 不参与当前 locale 的 check。Glossary 任意字节变化或本 locale 引用的历史 gate identity 变化都会使对应 legacy coverage 失效；之后必须执行正式 Glossary Review。

## 与后续 gate 的关系

Glossary Review 不替代 automatic validation、Candidate Snapshot、A-only TranslationUnit Quality Check、machine finalization、promotion、Course SEO schema v2、Locale Surface Review、preview machine acceptance、visual HUMAN gate 或 Production gate。

Glossary 变化后，既有 TranslationUnit Snapshot / QC carry-forward 仍按当前 identity 规则 stale；新的 Glossary Review PASS 不会恢复旧 A。最终 Locale Surface Review 仍完整检查 glossary 决策在 UI、metadata、TranslationUnit context、Course SEO 与其他 surfaces 中的实际一致性，并继续做完整 source ↔ target 语言质量审核。
