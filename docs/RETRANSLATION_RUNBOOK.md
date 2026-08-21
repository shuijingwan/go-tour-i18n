# Retranslation 执行手册

## 1. 流程概览

本地导出批次
→ Git 提交并推送批次输入至 GitHub
→ ChatGPT 从 GitHub 读取 manifest.json、inputs 和 glossary
→ ChatGPT 生成 raw-responses 并提交返回 GitHub
→ 本地拉取 GitHub 返回内容
→ process
→ 验证
→ review
→ promote

## 2. 准备批次

说明：

- retranslation export
- manifest.json
- inputs/

## 3. ChatGPT 翻译流程

批次输入须先由本地 Git 提交并推送至 GitHub。ChatGPT 以 GitHub 中的批次内容作为输入，并将生成的 raw-responses 提交返回 GitHub；随后在本地拉取该返回内容，再执行 process。

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

## 4. 页面翻译

article → article

## 5. 示例翻译

go → txt → txt → go

## 6. 验证与提升

process：
原始响应 → candidate

review：
candidate 证据

promote：
规范 candidate + ready

## 7. 不包含的内容

UI catalog 翻译是独立流程，不属于页面/示例翻译任务。

## 8. 故障排查

- token 遗失
- 验证器失败
- 重试
