# 项目状态

更新时间：2026-08-10

## 基线与架构

- 官方 upstream 基线为 `golang/website` 的 `master` 分支 commit `e11dacba76c5aae474746e9eedee19693f492803`；翻译运行时使用仓库内固定的最小 Tour 源码闭包，外部 checkout 仅用于同步与校验。
- 当前目录包含 103 个正式发布页面和 2 条单独保留的 `#appengine:` 条件源审计记录；两个条件 Section 去标记后同时投影为 `welcome/4`、`welcome/5`。
- 唯一维护 CLI 是 `cmd/tour-i18n`。
- 页面身份使用 `data/tour-pages.tsv` 中冻结的持久 `page_id`，不会以页面位置或临时语义 key 替代。
- 项目根 `.env` 的安全加载已完成：系统环境变量优先，`.env` 不覆盖已有值；`/.env` 被 Git 忽略，`.env.example` 可提交。

## 翻译执行与校验

- GLM-5.2 单页完整 `present.Section` 翻译执行器已完成；request、response 和 validation 按 locale、page_id、source hash 与 attempt 编号保留。
- 正式模式对一个 source 最多执行 3 次完整页面 attempt；失败后进入 `blocked`，blocked 页面不能继续正式重试。
- `translate run -dev` 仅用于单页开发校准：每条命令只执行 1 次 attempt，可从 pending 或 blocked 继续且不受正式三次上限约束；失败回到 pending，成功进入 ready。正式批量翻译不得使用 dev 模式。
- 正式发布投影中，`welcome/1` 使用去除条件标记后的 `a remote server.` 分支，本地原始 Tour 仍使用 `your computer.` 分支；`welcome/4`、`welcome/5` 使用去除 `#appengine:` 标记后的完整 Section。candidate validator 会拒绝 projected source/candidate 中残留的 appengine 标记，并继续校验 present 结构、directive、链接 target、行内代码和预格式化代码。
- legacy present 程序字体语法已按官方语义保护：例如 `` `package`rand` `` 作为一个完整代码单元，语义内容为 `package rand`，token 恢复后仍逐字保持原始 present 源写法。
- 固定 zh-CN 翻译提示词与 glossary 控制说明已统一为简体中文；protected token 必须完整、唯一，且不得修改、删除、复制或伪造。为适应目标语言自然语序可以调整 token 位置，但不得破坏链接、代码、directive、预格式化等结构关系；这是面向后续多语言扩展的结构安全原则，不是中文特例。
- protected token 已记录 kind 元数据。恢复前严格检查总数、未知 token、缺失和重复，不再要求全部 token 保持一条全局原始顺序；随后按模型响应中的实际 token 顺序确定性规范化独立 inline-code token 的外部边界。`` `package`rand` `` 等 legacy 单一代码 span 继续作为完整单元保护，不会被拆分。
- 预格式化 Go 教学代码已支持安全翻译自然语言行注释：所有非注释代码逐字保护，注释内引用的 Go 标识符单独保护；candidate validator 会独立检查注释结构、非注释字节及标识符的数量、大小写和顺序。
- candidate validator 基于 present 解析结果校验斜体、粗体和 program span 的类型与结构。inline-code 校验内容和数量（多重集合）及 program span 安全，但不再根据跨语言出现顺序判断技术语义；legacy program span 继续完整保护。跨语言技术语义是否正确不伪装成静态结构校验能力，仍须由提示词与人工审核把关。
- link target 继续严格校验内容、数量和自身顺序；preformatted block 继续严格校验内容、块顺序和代码安全；directive 继续严格校验内容、数量、自身顺序和 Section 归属。Section 拓扑、预格式化块与 directive 的 Section 归属均受校验；源中位于 Section 尾部的 directive 在候选中仍必须位于尾部。
- zh-CN glossary 对强制链接 label 使用 protected token 确定性恢复，例如 `A Tour of Go` → `Go 语言之旅`、`previous` → `上一页`、`next` → `下一页`、`Run` → `运行`、`Format` → `格式化`；validator 仍执行防御性术语与禁用译法检查。
- 已支持 ready candidate 本地预览。运行 `go run ./cmd/tour-i18n preview -id welcome/1 -locale zh-CN` 会在 `/tmp` 创建临时 Tour 内容副本，只替换目标中文 Section，不修改仓库正式 `_content`；其他未翻译页面继续显示英文。

### 2026-08-10 翻译输入架构实验

- 已提交 `62b1b1a feat: 增加翻译输入实验与最小保护能力`，新增 `--raw-input`、`--minimal-protect` 和受控同进程开发重试 `--dev-attempts`。
- `--raw-input` 将完整 production 页面直接发送给 GLM-5.2，不执行 protected token/restore；`flowcontrol/6` 首次真实 raw-input 请求即通过统一 validator，usage 为 1,403 tokens。该页旧默认 protected-token attempts 1～3 的失败，属于完整 inline-code 结构为适应中文语序换位后被旧保护/恢复逻辑误判。
- `methods/24` 两次纯 raw-input 均真实破坏 present 结构：模型会为 `.play methods/images.go` 追加参数，并将链接普通标签中的 `image`、`image/color` 自行改为新的 inline code；完全 raw 当前不足以作为默认生产方案。
- `--minimal-protect` 当前只保护完整 `.play` directive。`methods/24` 的唯一 `.play` token 被模型原样且恰好一次保留，并精确 restore，directive 问题消失；但链接普通标签新增 inline code 仍存在。随后已增加该失败的针对性 retry feedback 与同进程 `--dev-attempts`，后续实验又出现 font span count mismatch，说明当前 minimal-protect 尚不足以稳定替代默认完整保护流程。
- 当前结论是：大量 protected token 并非越多越好，会增加提示上下文、restore 复杂度，并可能妨碍符合中文习惯的自然语序调整；完全 raw 又不能稳定保护 present 机器结构。长期方向应为“原始页面优先 + 少量真正高风险机器结构保护 + 严格统一 validator + 针对性 retry feedback”。目前暂停继续扩大 minimal-protect，不主动为每种潜在结构增加保护规则；成熟的默认 protected-token 流程继续作为正式翻译路径。
- 正式翻译状态与 candidate 已恢复至实验前状态；真实实验 attempts 审计继续保留在 `data/translation-runs`。

## zh-CN 翻译进度

- 正式发布投影共 103 页，另保留 2 条条件源页面审计记录。
- 当前正式页面 103 页：`ready=45`、`pending=58`、`blocked=0`。
- 第三批普通页面已完成，以下 10/10 页面均为 `ready`，canonical candidate 均已通过统一 `candidate validate`：`flowcontrol/7`、`flowcontrol/9`、`flowcontrol/10`、`flowcontrol/11`、`flowcontrol/12`、`flowcontrol/13`、`flowcontrol/14`、`moretypes/1`、`moretypes/2`、`moretypes/3`。
- 本轮首次翻译 Prompt 校准确认：Protected Token 不仅保护 payload，也具有结构角色。inline pair 可随中文语序整体移动，但必须继续作为行内结构；static preformatted token 代表独立 block；directive token 代表独立 directive 行；教学注释中的 identifier token 代表恢复后仍须词法独立的 Go 标识符。不恢复 protected token 的全局原始顺序，也不限制正常中文语序调整。
- static preformatted 输入边界的真实根因已修复：原 static span 把尾部 block separator 纳入 payload，整体 token 化后会将源中的 `TOKEN\n\nprose` 退化为 `TOKENprose`。现在仅调整 `protectedPreformattedStatic` 的翻译输入保护右边界，保留 source 中原有 separator 于 token 外；restore 仍逐字节 round-trip，`preformattedBlocks` 与 validator 的 Present block 语义不变。
- `protectedPreformattedIdentifier` 原本已存在，缺少的是首次 Prompt 的角色说明。现在明确要求其在教学注释中原样保留，恢复后仍是词法独立的 Go 标识符，且不得与自然语言字符拼接。
- retry feedback 仍接收 `[]string` validation failures。曾因固定 diagnostic suffix 含有 `directive`，使字符串匹配将 preformatted/font failure 误分为 directive；当前通过分类前剥离固定 suffix，并将 preformatted、font/emphasis 分类置于 generic directive 前修复。结构化 failure kind/code 仅是未来可选优化，不是当前待办。

本阶段 glossary 已新增 preferred 术语：`standard library` → `标准库`、`iteration` → `迭代`、`loop condition` → `循环条件`。新增 mandatory 术语：`type switch`、`type switches` → `类型选择`，`type assertion`、`type assertions` → `类型断言`，`interface value` → `接口值`，`interface type` → `接口类型`，`concrete type` → `具体类型`。`square root`、`Newton's method`、`derivative` 等单页术语暂不加入。

`generics/1` attempt-004 与 `methods/20` attempt-001 共同证明，全局 protected token 顺序要求会误伤正确的跨语言自然语序；本次调整是通用多语言恢复与候选校验修复，而非 `methods/20` 的单页特例。

## 本日完成

- 完成第三批 10 个普通 pending 页面的真实 GLM-5.2 翻译与审计，全部最终进入 ready。
- `flowcontrol/10` 在补充 token 结构角色说明后的干净首次回归一次通过。
- `moretypes/1` 依次完成 static block 输入边界、教学注释 identifier Prompt 角色说明校准；干净首次回归通过。最终候选的两条教学注释已人工润色为“通过指针 p 读取 i / 通过指针 p 设置 i”，并重新通过统一校验。
- 完成 retry feedback diagnostic suffix 误分类的局部修复与真实失败文本回归测试。
- 本轮 prompt、preformatted 输入表示、retry feedback 与全量测试均已验证通过。

## 后续策略

- 第三批已完成并稳定。提交本轮改动后，继续第四批普通 pending 页面。
- 继续使用默认 protected-token 翻译路径、现有首次 Prompt 结构角色说明、统一 candidate validation 与审计记录；仅在真实失败出现时进行最小、可回归验证的修复。
- 不把 `flowcontrol/10`、`moretypes/1`、raw-input 对比或 static-block 调试作为当前待办；这些校准结论和历史证据已保留。
- 全量翻译不等于直接发布：通过页面仍需统一验证与必要的人工审核，失败页面按既有重试或 blocked 流程处理。
