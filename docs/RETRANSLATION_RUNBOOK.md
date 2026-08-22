# Retranslation 执行手册

## 1. 流程概览

本地终端：

```text
export batch
→ git commit
→ git push
```

ChatGPT：

```text
从 GitHub 读取 batch 输入
→ 生成 raw-responses artifact（例如 zip）
→ 检查 raw-responses 文件格式
```

本地终端：

```text
导入 raw-responses artifact
→ process
→ candidate
→ automatic validation
→ Quality Check
→ revision batch（如需要）
→ 新 batch process
→ automatic validation
→ Quality Check
→ git commit
→ git push
```

ChatGPT：

```text
执行 Final Review
→ 生成 review evidence
```

本地终端：

```text
保存 review evidence
→ promote
```

## 2. 准备批次

说明：

- retranslation export
- manifest.json
- inputs/

## 3. ChatGPT 翻译流程

批次输入须先由本地 Git 提交并推送至 GitHub。ChatGPT 只以 GitHub 中的批次内容作为输入。

读取顺序：

1. manifest.json
2. inputs/*
3. locales/<locale>/glossary.yaml

输出目录：

data/retranslation-runs/<locale>/<batch-id>/raw-responses/

要求：

- 一个 unit 一个文件
- 不输出解释
- 不输出 Markdown code fence
- 不输出 JSON
- 保留 protected token
- protected token 不可展开：即使可从上下文推断真实内容，也不得恢复、重写、补充代码示例或添加原文不存在的技术说明；只翻译 token 前后的自然语言。
- 保持原始 present 结构：不得新增原文不存在的代码块、preformatted section、`.play` directive 或示例说明；不要根据教程上下文重新编写页面。

完成后，ChatGPT 交付 raw-responses artifact（例如 zip）。本地将 artifact 中的 `raw-responses/*` 导入上述目录，直接执行 retranslation process。

raw-responses 是 retranslation process 的输入产物，不要求将其作为流程必要步骤执行 `git add`、`git commit` 或 `git push`。Git 主要保存 batch 定义、manifest、candidate、validation evidence、review evidence 和 promotion 状态；raw-responses artifact 不作为默认 Git 上传流程。

导入前检查：

- `raw-responses/*.article` 是纯 article 文本；
- 第一层内容不是 JSON；
- 不包含 Markdown code fence；
- 不包含 GitHub API metadata；
- 文件可以直接作为 retranslation process 输入。

## ChatGPT 执行能力规则

本项目已经验证 ChatGPT 可以完成以下 retranslation 交付流程：

```text
inputs/
→ ChatGPT 翻译
→ raw-responses artifact
→ 本地导入 raw-responses/
→ 本地 process
→ validation
```

该能力属于项目固定执行模式。后续新的 retranslation batch 不需要依赖历史聊天记录重新判断该能力是否存在。

当用户明确要求：

- `执行。`
- `继续执行当前任务。`
- `执行 batch：<batch-id>`

表示：

- 已完成方案确认；
- 进入实际执行阶段；
- 按项目已验证模式执行。

执行阶段不要：

- 重新讨论已经验证的 ChatGPT 翻译与 raw-responses 交付能力；
- 建议改由 Codex 负责翻译；
- 重新设计 retranslation 架构；
- 重复验证已经验证过的流程。

只有实际执行失败时，才报告具体失败原因。

## 已验证 ChatGPT retranslation 能力

- `chatgpt-ja-JP-005` 已验证 20 Page TranslationUnit batch。
- ChatGPT 已验证可以读取 batch inputs 并生成 retranslation raw-responses，生成结果可进入后续 process 和 validation 流程。
- Page 与 Example 均可显式使用 20 TranslationUnit batch。
- 20 TranslationUnit 需要通过 `retranslation export --limit 20` 指定。
- 当前默认 export limit 保持代码实现中的默认值，不修改。

## 4. TranslationUnit 与 batch 粒度

翻译生成以 TranslationUnit 为最小单元。不同 TranslationUnit kind 的输入与输出如下：

Page：

```text
inputs/*.article
→ ChatGPT 翻译
→ raw-responses/*.article
```

Example：

```text
inputs/*.txt
→ ChatGPT 翻译
→ raw-responses/*.txt
```

每一个 TranslationUnit 独立翻译。多个 TranslationUnit 可以属于同一个 batch；全部 `raw-responses` 准备完成后，以 batch 为单位导入并执行：

```text
batch raw-responses artifact
→ 导入 data/retranslation-runs/<locale>/<batch-id>/raw-responses/
→ retranslation process
```

边界如下：

- TranslationUnit 是翻译与质量审核的最小单元；
- Batch 是一次翻译执行和结果归档容器，不作为 validation、review、promotion 状态判断的最小单位。相关状态判断均以 TranslationUnit 为准。

## 5. 页面翻译

article → article

## 6. 示例翻译

go → txt → txt → go

## 7. 本地处理、质量检查、最终审核与提升

本地终端接收 ChatGPT 返回的 raw-responses artifact 并导入后，依次执行：

```text
process
→ candidate
→ automatic validation
```

执行 Quality Check，用于发现翻译质量问题。当前流程由 ChatGPT 作为质量检查执行者。Quality Check 可以多轮执行，但不生成正式 review evidence。

如需修订，进入新的 revision batch，不在原 batch 内修改 raw response 或重新 process。revision batch 完成 process 和 automatic validation 后，再继续 Quality Check，直至得到最终 candidate。

最终 candidate 通过 automatic validation 后，本地终端提交并推送 candidate / validation 结果。随后由 ChatGPT 对最终 candidate 执行 Final Review，生成正式 review evidence；review evidence 由仓库维护流程保存。

本地终端再执行：

```text
git pull
→ promote
```

正式 review evidence 是 Final Review 的产物，也是 promote 前的审核依据。具体的 candidate、validation、review evidence 和 promotion 条件分别以对应规范为准。

`retranslation review record` 仅用于写入已经完成的 Final Review 结果，不执行翻译质量审核。

## 8. Revision batch

Quality Check 或 Final Review 发现翻译质量问题时，按以下顺序处理：

```text
创建新的 revision batch
→ re-export 需要修订的 TranslationUnit
→ ChatGPT 在新 batch 中生成修订后的 raw response
→ 新 batch process
→ automatic validation
→ Quality Check
```

已处理 batch 保持不可变，candidate 和 validation evidence 不直接覆盖。revision batch 是翻译质量优化的正常流程。新的最终 candidate 通过 automatic validation 后，必须重新执行 Final Review 并生成其正式 review evidence。

revision batch 的 raw response 仍按上述方式导入并 process；candidate、validation evidence、最终 review evidence 和 promotion 状态由仓库维护流程保存。

## 9. Retry 与 revision batch

retry 用于 `restore_failed`、`validation_failed` 等处理失败，不用于已经 validation 通过后的翻译质量修改。

revision batch 用于 Quality Check 或 Final Review 发现翻译质量问题的场景。

## 10. 不包含的内容

UI catalog 翻译是独立流程，不属于页面/示例翻译任务。

## 11. 故障排查

- token 遗失
- 验证器失败
- 重试
