# 翻译质量实验

## 目的与边界

项目的自动 candidate validator 负责检查 present 解析、页面结构、链接、代码、directive、render 等安全性。validator 通过表示候选满足项目的结构与发布约束，但不等于中文译文一定达到最优质量。

2026-08-16～17 的实验关注三个问题：翻译输入方式是否影响可靠性、相同请求在不同运行间是否稳定，以及不同 Translation Engine 实际产出的最终中文质量。所有比较都以完整顶层 `present.Section` 为翻译单元，不采用句子级或多 text 槽位翻译。

## 2026-08-16～17：Default、minimal-v1 与 Static Context

### Default vs minimal-v1

实验固定使用 7 个代表页：

- `generics/1`
- `flowcontrol/8`
- `methods/16`
- `methods/20`
- `concurrency/7`
- `concurrency/11`
- `methods/24`

结构可靠性结果如下：

| 模式 | 首次 validator 通过 | 最终通过 |
| --- | ---: | ---: |
| Default | 7/7 | 7/7 |
| minimal-v1 | 5/7 | 7/7 |

早期单轮匿名质量比较中，Default 获得 4/7，minimal-v1 获得 3/7。这只是单次候选抽样；后续实验发现 GLM-5.2 对相同 Request 的输出存在运行间变化，因此不能把 4:3 解释为稳定的模式质量差异。

### Default protected-token 信息损失审计

对上述 7 页的审计结果为：

| 项目 | 结果 |
| --- | ---: |
| Pages | 7 |
| Protected tokens | 157 |
| Replaced source bytes | 1,206 |
| A machine structure | 761 bytes |
| B code / technical content | 419 bytes |
| C fixed natural language | 26 bytes |
| D hidden translatable English | 0 bytes |
| Hidden translatable English items | 0 |
| Hidden translatable English ratio | 0.00% |

审计没有发现 Default 隐藏需要翻译的英文自然语言。Default 与 minimal-v1 都向模型保留了完整页面级自然语言上下文；两者的差异主要是机器结构的抽象程度和静态代码的可见程度。因此，该实验不应被简单类比为“整篇翻译 vs 分段翻译”。

### 相同 Request 的运行时波动

重复实验固定使用以下 5 页：

- `generics/1`
- `flowcontrol/8`
- `methods/20`
- `concurrency/7`
- `methods/24`

5/5 页面都得到相同结论：

```text
REQUEST_IDENTICAL : true
RESPONSE_IDENTICAL: false
```

两次运行中的 source、model、system message bytes、user message bytes、thinking、do_sample、max_tokens 和实际 API payload 均相同，但 assistant content 不同。项目只记录“实际 API 请求未表现出字节级响应可复现性”，不猜测 GLM-5.2 服务端路由、部署或计算层原因。已观察到的运行间波动甚至可能改变 validator 的 pass / fail 结果。

### Static Context 30 次重复实验

实验由 5 页 × 2 种模式 × 3 次运行组成，共计划 30 次：

| 模式 | planned | API success | network failure | validator pass | validator fail | usable/planned |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Default | 15 | 14 | 1 | 12 | 2 | 12/15 |
| Static Context | 15 | 13 | 2 | 9 | 4 | 9/15 |

当前没有证据表明额外 static code context 带来结构可靠性优势。`methods/24` 的同一种 Default 模式在不同运行中既出现过盲评最好，也出现过盲评最差。现有证据更支持“GLM 运行间候选波动”是重要变量，而不是 Default 缺少自然语言上下文。

阶段结论：

- Default protected-token 继续作为当前正式翻译执行路径。
- minimal-v1 与 dev-static-context 继续保留为实验性质。
- 暂停围绕 protection policy 继续做细微优化。
- 下一步更值得评估 Translation Engine 本身。

## 2026-08-17：ChatGPT、Codex、GLM-5.2 最终译文匿名评审

### 实验目的与候选来源

本实验比较的是三种来源实际可得到的最终译文质量，不是 API benchmark。候选来源如下：

1. **GLM-5.2**：使用当前 canonical ready candidate，代表现有 Default 正式工作流最终留下的译文。部分历史 candidate 可能经历过人工润色，因此不能称为“纯原始 GLM response”。
2. **ChatGPT**：在隔离的新对话中读取完整可见英文 Section、术语规则和结构规则，不读取现有中文 candidate，生成独立候选。
3. **Codex Sol**：在全新 Codex 对话中读取英文 Section、glossary 和必要结构规则，生成前禁止读取现有中文 candidate，生成独立候选。

代表页为：

- `methods/24`：短页，但包含 image 包、接口、代码与 present 结构。
- `concurrency/7`：并发练习页，存在较明确的语义忠实度比较点。
- `concurrency/11`：长页，包含多个资料标题、codewalk、链接和整页文风一致性要求。

三方最终候选均通过同一个项目 candidate validator。结构兼容修正与中文语言质量分开记录，不使用结构传递问题给中文译文质量加分或减分。

`concurrency/7` 和 `concurrency/11` 中，ChatGPT 原始回复首行明确为 `*`，但复制到 Codex 验证步骤时曾变成 `-`。该差异不能归因于 ChatGPT 翻译模型本身，应视为人工复制或消息传递链路中的数据完整性风险，也是后续希望消除人工复制粘贴的重要原因之一。`methods/24` 的 ChatGPT 原始输出还存在 present 兼容性问题，例如全角冒号和链接 label 新增 inline-code；这些都可在不修改中文自然语言的前提下完成纯结构修正。

Codex 在 `concurrency/7`、`concurrency/11` 的首次实际 candidate 校验中均直接通过；`methods/24` 仅发生 present 结构兼容修正。这些结构现象不计入中文译文质量评分。

### 匿名评审方法

每页的三份候选使用系统随机源映射为候选甲、候选乙、候选丙，映射在评分完成前不提供给评审模型。四个独立评审模型为 ChatGPT、DeepSeek、GLM-5.2 和豆包；每个页面、每个评审模型都使用独立的新对话。

统一评分维度为：

| 维度 | 分值 |
| --- | ---: |
| 技术准确性 | 30 |
| 原文忠实度 | 20 |
| 中文自然度 | 20 |
| 教学表达 | 15 |
| 术语一致性 | 10 |
| 可读性 | 5 |
| 总分 | 100 |

不同模型的绝对打分尺度不同，因此最终分析优先看第一名票数、排名稳定性、具体扣分理由，以及是否存在真实技术错误，不简单平均不同 judge 的绝对分数。

### 12 次匿名评审完整排名

| 页面 | 评审模型 | 第 1 名 | 第 2 名 | 第 3 名 |
| --- | --- | --- | --- | --- |
| `methods/24` | ChatGPT | ChatGPT | Codex | GLM-5.2 |
| `methods/24` | DeepSeek | GLM-5.2 | ChatGPT | Codex |
| `methods/24` | GLM-5.2 | ChatGPT | GLM-5.2 | Codex |
| `methods/24` | 豆包 | ChatGPT | Codex | GLM-5.2 |
| `concurrency/7` | ChatGPT | ChatGPT | Codex | GLM-5.2 |
| `concurrency/7` | DeepSeek | ChatGPT | GLM-5.2 | Codex |
| `concurrency/7` | GLM-5.2 | Codex | ChatGPT | GLM-5.2 |
| `concurrency/7` | 豆包 | ChatGPT | GLM-5.2 | Codex |
| `concurrency/11` | ChatGPT | ChatGPT | Codex | GLM-5.2 |
| `concurrency/11` | DeepSeek | Codex | ChatGPT | GLM-5.2 |
| `concurrency/11` | GLM-5.2 | GLM-5.2 | Codex / ChatGPT（并列） | — |
| `concurrency/11` | 豆包 | ChatGPT | GLM-5.2 | Codex |

`concurrency/11` 的 GLM-5.2 judge 给 Codex 与 ChatGPT 同为 96 分，评审文字明确称两者并列；统计时两者均按第 2 名处理，不人为制造顺序差异。

### 第一名票数与平均名次

12 次全部 judge 的第一名票数：

- ChatGPT：8/12
- Codex：2/12
- GLM-5.2：2/12

排除 ChatGPT 自身作为 judge 的 3 次，只看 DeepSeek、GLM-5.2、豆包的 9 次外部评审：

- ChatGPT：5/9
- Codex：2/9
- GLM-5.2：2/9

平均名次如下；并列情形按评审给出的第 2 名计入：

| 范围 | ChatGPT | Codex | GLM-5.2 |
| --- | ---: | ---: | ---: |
| 全部 12 次 judge | 1.33 | 2.25 | 2.33 |
| 9 次外部 judge | 1.44 | 2.33 | 2.11 |

### 质量观察

- ChatGPT 在 12 次综合评审中第一名票数最多、平均名次最好。
- 即使完全排除 ChatGPT 自己作为 judge，ChatGPT 仍获得 5/9 第一名，且外部平均名次最好；目前观察到的优势不能只用“ChatGPT 自评偏好”解释。
- 不同 judge 对字面忠实度、中文自然度和教学表达的权重明显不同。
- 三种来源总体都能生成高质量候选；多数差异集中于高质量译文之间的精细择优，而不是明显技术错误。
- 不能从 3 个页面直接推断现有 103 页都需要被替换，也不能把 94、95、96、97 等单次 judge 分数当成整个 103 页的绝对质量分。

## 2026-08-17 当时的决策与下一阶段

### 当时已决定

- 当时生产 zh-CN 继续保持 103/103 ready。
- 当时 canonical candidates 不因该次实验被批量覆盖，production release 不变。
- Default protected-token 是当时已经实现并验证的正式翻译输入路径。
- 不再优先投入 minimal-protect / Static Context 的细微调优。
- ChatGPT 是当时下一阶段最值得优先验证的高质量翻译来源，但尚未接入当时的正式执行路径。
- Codex 继续适合承担仓库读取、写入、结构适配、validator、render/test 等工程工作；本次实验也证明其自身翻译能力具有竞争力。
- GLM-5.2 现有 103 页并非质量失败，本实验不否定已经完成的翻译工作。

### 工作假设

以下内容是下一阶段的工作假设，不是已经确认的事实：如果后续更大范围实验能够证明 ChatGPT 重译可将现有最终译文从当前代表页中常见的约 94～95 分区间，稳定提升到约 96～97 或更高，或在多模型匿名评审中体现出稳定、实际可感知的质量提升；同时能够把 103 页的重译、结构处理和统一校验高度自动化，并将全流程工作量控制在大约 1～2 天量级，那么重新生成 zh-CN 103 页的 ChatGPT 高质量候选值得考虑。

94～95、96～97 只是当前代表页实验中的工作参考区间，不是对整个 103 页现有译文的全量量化评分。

### 下一阶段目标

当时的下一阶段先设计自动化 ChatGPT retranslation staging，不立即覆盖生产。理想流程为：

```text
完整英文 Section
→ ChatGPT 整页翻译
→ 原始响应以机器可读/字节安全方式直接保存
→ Codex / 本地工具进行必要结构适配
→ 同一 candidate validator
→ render / page validation
→ 新的 retranslation candidate staging
→ 与现有 canonical candidate 比较
→ 达到替换标准后才进入正式切换
```

关键原则：

- 尽量不让用户逐页复制粘贴。
- 避免 Markdown/UI 传递导致 `*` 变 `-` 等内容漂移。
- 保留原始 ChatGPT 输出和结构修正后的最终候选，便于审计。
- 新重译流程先写入独立 staging，不直接覆盖现有 103 个 ready candidate。
- 新旧译文必须使用相同 validator。
- 在自动校验和质量标准通过前，不改变 production。
- 不为这一阶段过早开发数据库、Web 审校平台或复杂队列系统。

该阶段只记录下一阶段需要研究“尽可能自动化的 ChatGPT 翻译接入方式”，具体如何调用 ChatGPT 当时留待后续单独设计。

## 相关公开记录

- [《A Tour of Go 中文翻译完成后，我为什么重新评估 minimal-protect 翻译模式》](https://www.shuijingwanwq.com/2026/08/16/26323/)
- [《A Tour of Go 翻译实验：更多原始上下文为什么没有更好？从 157 个保护标记到 30 次重复盲评》](https://www.shuijingwanwq.com/2026/08/17/26558/)

## 2026-08-23：Codex High 生产配置评估

本轮比较 ChatGPT High、Codex High 与 Codex Extra High。外部评审汇总为：

| Translation Engine | 平均分 | 平均名次 | 第一名 |
| --- | ---: | ---: | ---: |
| ChatGPT High | 96.67 | 1.73 | 5/15 |
| Codex High | 95.73 | 2.07 | 5/15 |
| Codex Extra High | 95.87 | 2.20 | 5/15 |

四评审汇总为：

| Translation Engine | 平均分 | 平均名次 | 第一名 |
| --- | ---: | ---: | ---: |
| ChatGPT High | 96.90 | 1.85 | 6/20 |
| Codex High | 96.50 | 1.85 | 9/20 |
| Codex Extra High | 96.15 | 2.30 | 5/20 |

结论：Codex High 已进入与 ChatGPT High 基本相同的质量梯队，Extra High 没有显示稳定收益。后续默认生产 Translation Engine 改为 **Codex GPT-5.6 Sol High**；ChatGPT 改为所有 TranslationUnit 的统一 Quality Check 执行者。此前 zh-CN 103 页由 ChatGPT 完成的事实属于历史结果，不因生产配置切换而改变。
