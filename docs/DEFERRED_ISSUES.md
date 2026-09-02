# Deferred Issues

本文件用于记录在开发、测试、部署或 production 验收过程中已经真实发现，但经明确决定暂缓处理的问题，确保问题不会因为超出当前任务 scope 而只留在聊天记录中。

## 记录边界

只有同时满足以下条件的问题才应登记：

- 问题已经在实际工作中被发现，有可描述的现象、失败结果或其他可靠 evidence；
- 用户已明确要求当前任务暂不处理，或明确确认延后处理。

普通 TODO、尚未验证的怀疑、未来优化建议和架构设想不属于 deferred issue。不要主动把所有顺手发现的改进机会写入本文件，也不要凭猜测补录历史问题；历史问题只有存在可靠 evidence 时才可登记。

本机制只负责避免已知问题遗忘，不改变 TranslationUnit、Quality Check、Final Review、Locale Surface Review、production 或广告流程，也不替代这些流程原有的状态、gate 和 evidence。

## 维护规则

- ID 使用 `DI-YYYYMMDD-NNN`，日期取首次发现日期，序号为当日从 `001` 开始的未占用编号。ID 创建后保持不变。
- 新发现且明确暂缓的问题，先检查是否已有同一问题：没有则新增，有则补充本次发现阶段、上下文和 evidence，避免重复登记。
- `当前状态` 只能是 `open`、`resolved`、`accepted` 或 `obsolete`：
  - `open`：问题仍存在，等待后续处理或决定；
  - `resolved`：问题已经修复，并有验证或核销 evidence；
  - `accepted`：问题仍存在，但已明确接受其影响，不再计划修复；
  - `obsolete`：由于相关功能、环境或前提已消失，问题不再适用。
- 将条目从 `open` 改为其他状态时，不删除原始问题和暂缓原因；必须在“后续处理/核销证据”中记录决定或处理结果、日期，以及可复核的 commit、命令结果、日志、URL、文件路径或其他 evidence。
- evidence 尚不足以证明已修复、已接受或已失效时，状态保持 `open`。

## 条目模板

复制以下模板到“问题登记”末尾；同一条目有新上下文时直接更新原条目。

```markdown
### DI-YYYYMMDD-NNN：简短标题

- ID：`DI-YYYYMMDD-NNN`
- 发现日期：`YYYY-MM-DD`
- 发现阶段/场景：
- 问题描述：
- 暂缓原因：
- 当前状态：`open`
- 后续处理/核销证据：暂无；保持 open。
```

## 问题登记

### DI-20260901-001：术语政策对 glossary keep 实现状态的描述已过期

- ID：`DI-20260901-001`
- 发现日期：`2026-09-01`
- 发现阶段/场景：制定 ko-KR 正式 glossary 并核对 `docs/TRANSLATION_TERMINOLOGY.md`、`internal/i18n/glossary.go` 与 glossary tests。
- 问题描述：`docs/TRANSLATION_TERMINOLOGY.md` 第 9.1 节仍称 loader 不解析 YAML `keep`、实际 keep 仅来自硬编码保护；当前 `LoadGlossary` 已解析并校验 `keep`，`PromptRules` 也会把它作为“保持原样”规则注入模型 prompt，现有测试对此有覆盖。政策文档与真实实现不一致。
- 暂缓原因：本轮明确只制定 ko-KR glossary，不调整跨 locale 术语政策或实现状态文档。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-02 已更新 `docs/TRANSLATION_TERMINOLOGY.md` 第 2.3、9.1、9.2 节，使 policy、loader、prompt、protector 与 Example validator 的当前边界一致；由 `go test ./...` 与 `git diff --check` 验证。
