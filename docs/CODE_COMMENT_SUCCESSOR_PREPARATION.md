# Code-comment / Site v2 successor 工程交接

本轮停止于独立语言审核及正式 successors 之前。没有 Generation、Reviewer rating、语言 replacement、reopen、finalization、closure、Surface、activation、Preview、build、publish、Production、commit 或 push。`HUMAN_GATE=FAILED`，DI-20261019-008 仍 open。

共享持久合同与未来 gate activation 见 [代码注释完整性](CODE_COMMENT_INTEGRITY.md)。审计覆盖 103 Tour Page、19 eligible Example、49 Learn/Docs Page 的完整 canonical source 与可证明的 current accepted targets；每个 code block 都显式标记 extractor/unsupported，不把 unsupported 当作无注释。

## 已完成工程与证据

- Source inventory：432 code blocks；293 已提取 comment slots，其中 teaching 257、protected 36、unclassified 0。Tour Page 为 58/29/16/13，Tour Example 为 19/80/61/19，Learn/Docs 为 355/184/180/4（依次为 blocks/slots/teaching/protected）。另有 235 个未声明语言且不满足既有 Go grammar 的 block，显式 `UNSUPPORTED_COMMENT_LANGUAGE=unspecified`；其中 Tour Page 6、Learn/Docs 229。上述 slot counts 是已支持语法的实际提取计数，不声称这些未知语法潜在注释为零。
- 65 个 roadmap locale candidates 中，55 门 Tour 的 current accepted bytes 可证明；Learn/Docs 只有 zh-CN 有对应 current accepted package。成功 comparison 共 6,179 slots；非 English exact teaching carry 51；missing/protected mutation/structure mismatch 均为 0（限定成功读取且支持的语法）。无法证明 current target 的 locale 不计入 successful scan。
- currentness unavailable：`pt-BR,it-IT,nl-NL,tr-TR` 为共享 reader 的 complete A-only coverage failure；`ar,mr-IN,sl-SI` 为 `welcome/1` canonical mismatch；`he,pa-IN` 为 `basics/12` canonical mismatch；`ur-PK` 为 `moretypes/1` canonical mismatch。保留具体 reader failure，不使用旧 Snapshot 代替，不在本轮展开其他 lifecycle diagnosis。
- zh-CN Tour 的 16 Page teaching slots 与 61 Example teaching slots均无 exact carry；Learn/Docs 180 teaching slots 中 exact carry 39，分布于下面 8 个正式 Page。`vuln/database` 的 17 个 English schema comments 全部捕获。
- `/translation/` 复用 legacy 九分区 body model，保留 current shared header/footer，exact coverage directory 不制造父目录覆盖；不改五段 accepted 正文。Tour 手机端 globe 使用原 details summary，locale text/额外 arrow 隐藏，不增加 menu 或改变 JS keyboard/focus/scroll contract。
- `/doc/` 与 `/doc/install` 的 frozen source identity 与原 frontier audit 一致；没有新候选。application/reference authority 实际 4/8，详见 [frontier audit](SITE_V2_CONTENT_FRONTIER_AUDIT.md)；其余申请状态及 PayPal URL 不猜测。

本轮 `/tmp` evidence：`code-comment-source-inventory-20261020.json`、`code-comment-current-locale-audit-20261020.json`、`code-comment-zh-CN-summary-20261020.txt`、`code-comment-cross-locale-summary-20261020.txt`、`code-comment-preparation-authority-20261020.json`。JSON 包含 source/target exact identities、每 slot 状态、current authority、unsupported diagnostics 与 stable ordered review units。

## 后续五条 workstreams

正式 comment replacement 前，须先完成 source representation 的精确 activation migration：将已确认但旧表示仍 protected 的 teaching bodies 接入既有 Tour/Learn parser/protector/restore，保留 comment/code identity；按真实 protected-input / Unit identity / corpus delta 走既有 reconciliation、currentness 与 glossary authority 流程。特别是 JSON schema inventory 不代表 formal Page 已开放这些 slots，不能让 Generation 编辑 opaque boundary 或从旧 Bundle 生成 replacement。该前置工程不授权批量否定历史 A。

1. `site-v2-shell /translation/`：正式 successor content，依序 Generation → machine validation → Independent QC → exact revision/re-QC（如需）→ A-only finalization → successor closure。新的 baseline、地区、申请、维护投入等正文仍需该正式流程，工程不提供 replacement。
2. `zh-CN Tour`：当前已支持语法未发现 machine suspects，不 reopen 无变化 A。先解决六个 unspecified block 的 coverage limitation；只有后续 machine suspect 经独立 Reviewer 确认的 Page/Example 才进入 affected-only revision。
3. `zh-CN learn-docs-v1`：未来 Reviewer scope 仅以下 8 Page，按 stable Page order：`/doc/tutorial/add-a-test`、`/doc/tutorial/greetings-multiple-people`、`/doc/tutorial/handle-errors`、`/doc/tutorial/random-greeting`、`/doc/database/cancel-operations`、`/doc/database/open-handle`、`/doc/database/querying`、`/doc/security/vuln/database`。Human finding 强制保留最后一页。独立确认后才由原 Generation session 做 exact affected Page revision，不重审其余 41 Page。
4. Learn/Docs source expansion：仅 `/doc/` 与 `/doc/install`，先取得正式 source adoption / package authority，再走共享 Generation/QC lifecycle，不把目录链接当覆盖。
5. 其他 locale affected-only：按 roadmap 顺序，`id-ID`（methods/6、methods/7）、`nb-NO`（basics/11）、`da-DK`（basics/11）、`cs-CZ`（moretypes/1、methods/6、methods/7）、`pt-PT`（methods/6、methods/7）；均为 Tour Page，Example 无 machine suspects。共 5 个其他 locale、10 Page、12 exact carries。加上 zh-CN 为 6 个 affected locales、18 suspect Units、51 slots。仅 inventory/proposal，不启动 Reviewer lifecycle。10 门 currentness unavailable 不冒充已清查完成。

## 真实未解决 failure 与停止点

本轮新增 targeted Go tests、既有 Tour comment regressions，以及 6 项 focused Chrome presentation/keyboard tests通过。但直接相关既有 `TestCompleteCodeTeachingProtection` 在 Markdown fence、HTML pre、indented 三种 enclosing fixture 的 line 88 失败：fixture 预期额外 `newName` 被拒绝，实际 validator 返回 nil。原 protector 未被本轮修改；按 quota/test boundary 只保存最小真实 evidence，不扩大调查或修改正式 validator，也不推断是实现缺陷还是 fixture 与现行规则不一致。证据：`/tmp/code-comment-existing-protector-failure-20261020.txt`。没有将它标成已修复或写成推测 root cause，也没有把它记为维护者已批准的 Deferred Issue。

因此本轮不能宣称全项 PASS 或完整注释 coverage PASS。opt-in contract 已具备，但 `COMMENT_GATE_ACTIVE=no`；未知语法、currentness unavailable 和该既有 regression 都是显式限制。下一步由 Coordinator 消费本轮 evidence，先安排 exact protector failure 的工程处理，再调度上述 affected-only 语言工作；本轮不执行其中任何 formal mutation。

所有正式语言工作及 source expansion 完成后，只安排一次 fresh Preview → full automated acceptance → Maintainer HUMAN gate；本轮 `PREVIEW_STARTED=no`、`NEW_PREVIEW_REQUIRED=yes`。
