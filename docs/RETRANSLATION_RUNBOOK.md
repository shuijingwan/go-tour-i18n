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
→ candidate / validation
→ git commit
→ git push
```

ChatGPT：

```text
执行 Translation Quality Review
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

## 6. 本地处理、审核与提升

本地终端取得 ChatGPT 返回的 raw response 后，依次执行：

```text
process
→ candidate / validation
→ git commit
→ git push
```

随后由 ChatGPT 执行 Translation Quality Review。review 完成后，ChatGPT 将 review evidence 提交并推送至 GitHub。

本地终端再执行：

```text
git pull
→ promote
```

具体的 candidate、validation、review evidence 和 promotion 条件分别以对应规范为准。

## 7. Review 修订闭环

当 review 为 C 或 D 时，不进行 promote。按以下顺序处理：

```text
修订 raw response
→ 重新 process
→ 重新 validation
→ 重新 review
```

修订后的 raw response、重新生成的处理结果和 review evidence 仍按上述 GitHub 中转与角色交接流程提交、推送和拉取。

## 8. 不包含的内容

UI catalog 翻译是独立流程，不属于页面/示例翻译任务。

## 9. 故障排查

- token 遗失
- 验证器失败
- 重试
