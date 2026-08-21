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
→ 生成 raw-responses/*
→ git commit
→ git push
```

本地终端：

```text
git pull
→ process
→ candidate
→ automatic validation
→ Quality Check
→ revision loop（如需要）
→ final validation
→ git commit
→ git push
```

ChatGPT：

```text
执行 Final Review
→ git commit
→ git push
```

本地终端：

```text
git pull
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

完成后，ChatGPT 将 `raw-responses/*` 提交并推送至 GitHub。本地终端先 `git pull` 获取返回内容，再执行 process。

## 4. 页面翻译

article → article

## 5. 示例翻译

go → txt → txt → go

## 6. 本地处理、质量检查、最终审核与提升

本地终端取得 ChatGPT 返回的 raw response 后，依次执行：

```text
process
→ candidate
→ automatic validation
```

执行 Quality Check，用于发现翻译质量问题。当前流程由 ChatGPT 作为质量检查执行者。Quality Check 可以多轮执行，但不生成正式 review evidence。

如需修订，由 ChatGPT 修改 raw response，不直接修改 candidate。修改后，本地终端重新 process 和重新 validation，再继续 Quality Check，直至得到最终 candidate。

最终 candidate 通过 final validation 后，本地终端提交并推送 candidate / validation 结果。随后由 ChatGPT 对最终 candidate 执行 Final Review，生成正式 review evidence，并将 review evidence 提交并推送至 GitHub。

本地终端再执行：

```text
git pull
→ promote
```

正式 review evidence 是 Final Review 的产物，也是 promote 前的审核依据。具体的 candidate、validation、review evidence 和 promotion 条件分别以对应规范为准。

## 7. Revision loop

Quality Check 发现需要修订的问题时，按以下顺序处理：

```text
修订 raw response
→ 重新 process
→ 重新 validation
→ 重新 Quality Check
```

Final Review 若要求修订，也回到同一 revision loop；新的最终 candidate 完成 final validation 后，必须重新执行 Final Review 并生成其正式 review evidence。

修订后的 raw response、重新生成的处理结果和最终 review evidence 仍按上述 GitHub 中转与角色交接流程提交、推送和拉取。

## 8. 不包含的内容

UI catalog 翻译是独立流程，不属于页面/示例翻译任务。

## 9. 故障排查

- token 遗失
- 验证器失败
- 重试
