# 2026-08-23 Translation Engine 质量实验 evidence

本目录保存 ChatGPT GPT-5.6 High、Codex GPT-5.6 Sol High 与 Codex GPT-5.6 Sol Extra High 在五个完整 Page TranslationUnit 上的最终 V2 质量实验 evidence。实验用于比较默认生产 Translation Engine，不属于 production retranslation batch，也没有进入 promotion。

五个 TranslationUnit：

- `methods/24`
- `concurrency/7`
- `concurrency/11`
- `generics/1`
- `flowcontrol/8`

`input-batch/` 是正式实验输入快照，包含 manifest 和五份 protected input。实验使用 manifest 对应的 `locales/zh-CN/glossary.yaml`；精确 Git 与 SHA-256 provenance 记录在 `provenance.json`。

第一次实验因遗漏正式 locale glossary 而整体作废。本目录只保存最终有效的 V2 evidence，不包含任何 V1 candidate 或评审结果。

最终 V2 的 15 份 candidate 全部通过 Engineering Gate（15/15）。每个 TranslationUnit 分别进行独立随机匿名排列，匿名候选、冻结评审 Prompt 和四份原始 reviewer 输出均保持原始字节；真实模型映射只由 `reveal-key.json` 提供。冻结 Prompt SHA-256 为：

```text
94a386eca3680a663409f3be4a7e770310189bb762733cd435ff40f7d91a2add
```

`aggregate.json` 由四份 raw review 与 reveal key 重新计算。GLM-5.3 对 `concurrency/11` Candidate C 的原始 reported total 为 95，而六项 component 合计为 96；raw review 保持不变，aggregate 按 component 算术和使用 96，原始 ranking 不变。

最终生产决策：默认 Translation Engine 使用 Codex GPT-5.6 Sol High；ChatGPT 统一承担 Quality Check；Extra High 不作为默认配置。

完整实验分析见 [`docs/TRANSLATION_QUALITY_EXPERIMENTS.md`](../../../../docs/TRANSLATION_QUALITY_EXPERIMENTS.md)。
