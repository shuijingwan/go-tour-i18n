# 正式阶段交接合同

本合同适用于 Generation role/session、Reviewer role/session 和具备仓库终端能力的当前 AI execution environment。它统一阶段完成、失败与人工 handoff 的最终回复；不新增持久化状态，不替代 machine gate，也不扩大 AI 权限。

## Reviewer Markdown evidence 格式

QC、re-QC 等新 Reviewer Markdown 必须先写入正式路径之外的临时 draft，再通过同一格式检查以原始字节、原子且不可覆盖地保存到正式目标：

```sh
go run -mod=readonly ./cmd/tour-i18n review-evidence save \
  --input /tmp/<review-draft>.md \
  --output <formal-review-evidence>.md
```

`surface-review evidence-scaffold` 已创建的正式待填写文件不另行复制或覆盖；Reviewer 完成填写后立即只读检查，`PASS` 后才执行正式 `surface-review record-a`：

```sh
go run -mod=readonly ./cmd/tour-i18n review-evidence check \
  --file data/locale-surface-reviews/<locale>/<review-id>.md
```

Glossary Review 如保存独立 Markdown，也使用 `review-evidence save`。两条命令只检查非空、UTF-8/BOM、LF、行尾空白和文件末尾换行格式，不修改 Markdown 内容、不作语言审核。格式失败时，只修复尚未被正式 receipt 绑定的当前 draft 或待审核文件；不得修改历史 evidence，不得自动重新审核。正式目标已存在时 `save` 必须失败，尤其不得覆盖已被 receipt 绑定 SHA-256 的 Markdown。

本文件是 Generation / Reviewer Bundle 的共享 authority。此 authority 修改后继续使用任何在途 ZIP 前，必须先执行该 bundle 对应的正式 current-check；只有检查真实判定 stale 时才按实际受影响范围重新导出。不得仅因本节新增格式规则而重译、重新审核、修改已有 A 结论或改写 finalization。

## 必填交接内容

需要跨会话、交给维护者 Local terminal、等待明确继续、进入 HUMAN gate，或因失败停止时，最终回复依次提供：

1. 当前事实：locale、stage、`PASS` / `FAILED`，batch / review / Snapshot ID，真实完成数量，以及正式 evidence path 或 machine output。
2. 唯一下一步：根据当前 machine state 和正式 scope 指定一个动作、执行者与停止条件。有多个恢复选择时先说明阻塞，不猜测选择。
3. 附件 handoff：具备权限且需要跨会话 ZIP 时，先真实导出并执行 current-check，再给出真实绝对路径、`sha256sum`、目标 session/role 和完整指令。禁止把占位路径、推测 hash 或未运行的检查写成事实。
4. Local terminal handoff：提供可直接粘贴的完整命令、前置 gate 与完成后的检查方法；不得在当前 shell 顶层设置 `set -e`、`set -u`、`set -o pipefail`，不得包含 `exit`。
5. 暂不能导出时：说明缺少终端、附件通道、current input 或其他真实原因，并给出正确的导出、current-check 与 SHA-256 命令，不能只写“请上传 ZIP”。
6. 失败交接：记录已完成事实、失败 stage、mutation 为 `none` / `known` / `unknown`、最小安全恢复路径。`mutation=unknown` 时禁止盲目重放写操作。

闭环内合法的短机械操作连续执行到真正需要附件上传、维护者决策、明确“继续”或 HUMAN gate。`promotion`、build、preview/browser verifier、publish、Production/deploy/verifier、search closeout、最终 commit/push 仍由维护者 Local terminal 执行。

## ZIP 交接模板

以下 `/tmp/<...>`、`<sha256>`、ID 与数量均为文档占位符，不是已生成附件：

```text
locale: <locale>
stage: <stage>
result: PASS
identity: batch=<batch-id> snapshot=<snapshot-id>
completed: <actual>/<scope>
evidence: <repository evidence path>
next: 把 <absolute-zip-path> 上传到独立 Reviewer session；完成该 ZIP 的一次独立 invocation 后停止。
bundle: <absolute-zip-path>
sha256: <sha256>
current-check: PASS
```

只有命令尚不能执行时才交付下列占位命令，并明确标注“尚未执行”：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle \
  --locale <locale> --snapshot-id <snapshot-id> \
  --previous-snapshot-id <previous-snapshot-id> \
  --start-index <stable-index> --limit 60 \
  --output /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle-check \
  --bundle /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
sha256sum /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
```

## 真实流程示例

以下路径和 hash 仍是明确的占位示例；实际回复必须替换为已核验事实。

### Initial Generation validation PASS 后等待继续

```text
locale: <locale>
stage: initial TranslationUnit batch validation
result: PASS
batch: <batch-id>
completed: 60/60 validation passed
evidence: data/retranslation-runs/<locale>/<batch-id>/result.json
next: 维护者确认“继续”后，由 Local terminal 导出下一批 Generation Bundle。
stop: 等待维护者明确“继续”；此前不启动下一次 model invocation。
```

### 每组 QC record 后导出下一组

当前 AI execution environment 先运行 `quality-check preflight`，确认 persisted predecessor 与 scope，再记录本组；只有 record 成功后才导出下一组，运行 `reviewer-bundle-check` 和 `sha256sum`。交接给出真实 ZIP 绝对路径、hash、stable indexes 和目标 Reviewer session。首次默认顺序固定为 Page `1–60`、Page `61–103`、Example `104–122`，每组使用新的用户请求/model invocation。

### Revision 后导出 pending Reviewer ZIP

新 Snapshot 建立后先执行：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check preflight \
  --locale <locale> --snapshot-id <new-snapshot-id> \
  --previous-snapshot-id <previous-snapshot-id>
```

只有 predecessor、carry-forward 与待审数量符合实际 revision 身份，才导出当前 pending Reviewer ZIP。交接停止于独立 Reviewer 的新 invocation，不自动把 replacement 评为 A。

### Surface Review PASS 后移交 preview

`record-a` 与 `check-a` 均通过后，AI 不启动 preview。交给维护者以下完整命令，并以 `PREVIEW SURFACE ACCEPTANCE: PASS` 为停止条件：

```sh
go run -mod=readonly ./cmd/tour-i18n preview --locale <locale> --http 127.0.0.1:0
# 另一个终端使用上条命令打印的真实端口：
scripts/verify-preview-browser.py http://127.0.0.1:<port>/ <locale>
```

### 额度耗尽恢复

```text
locale: <locale>
stage: <stage>
result: FAILED
completed: <actual>/<scope>
last current evidence: <path>
failure: generation quota exhausted before <not-started unit/batch>
mutation: none
next: 额度恢复后由原 Generation role 从 <exact batch/staging/evidence> 继续；先核对 raw responses、result.json、validation、Snapshot/QC，禁止重复 import 或覆盖已有输出。
stop: 等待额度，或等待维护者批准安全批次边界上的 provider/model 变更。
```

额度耗尽不降低模型、A-only QC、current/exact-set、provenance 或任何 Production gate。
