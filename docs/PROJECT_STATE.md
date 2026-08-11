# 项目状态

更新时间：2026-08-11

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

## zh-CN 课程正文完成状态

- 正式发布投影共 103 页，另保留 2 条条件源页面审计记录；当前正式状态为 `ready=103`、`pending=0`、`blocked=0`。
- 103 个正式发布页面均已完成翻译，课程正文阶段已经结束；第三批至后续各批的翻译、校准与修复均为已完成的历史过程，不再作为当前推进项。
- 发布前已完成 103 页全局译文质量审计，并完成必要的修订。全量导出与核对材料完整：100 个普通 Section、3 个特殊投影；英文源 103/103 成功导出且每页 SHA-256 与冻结状态源一致；zh-CN canonical candidate 103/103 成功导出且均与当前状态指向一致；缺失、重复、多余页面均为 0，`index.md` 共 103 条；导出前后 Git 状态一致。
- 特殊投影已纳入上述审计：`welcome/1` 使用 appengine remote 分支 `a remote server.`；`welcome/4`、`welcome/5` 使用完整 `#appengine:` 条件 Section 去前缀后的投影。
- 已形成的翻译执行结论继续有效：Protected Token 保护 payload 与结构角色，允许为目标语言自然语序调整位置但不得破坏 present 结构；静态校验持续覆盖链接、代码、directive、预格式化块及其拓扑关系。`flowcontrol/10`、`moretypes/1` 的校准和 raw-input/minimal-protect 实验均已形成结论，不是当前待办。

本阶段 glossary 已新增 preferred 术语：`standard library` → `标准库`、`iteration` → `迭代`、`loop condition` → `循环条件`。新增 mandatory 术语：`type switch`、`type switches` → `类型选择`，`type assertion`、`type assertions` → `类型断言`，`interface value` → `接口值`，`interface type` → `接口类型`，`concrete type` → `具体类型`。`square root`、`Newton's method`、`derivative` 等单页术语暂不加入。

`generics/1` attempt-004 与 `methods/20` attempt-001 共同证明，全局 protected token 顺序要求会误伤正确的跨语言自然语序；该调整是通用多语言恢复与候选校验修复，而非单页特例。

## 当前下一阶段：公共 UI 本地化与发布前 UI 验收

- 第一阶段仍只交付 zh-CN，但公共 UI 本地化须面向后续多语言扩展：不得将官方英文 UI 直接硬编码替换为中文；公共 Go 服务端、模板基础结构、CSS 和 JavaScript 应尽量共享，每种语言独立维护其公共 UI 文案，且 UI 文案继续与课程正文翻译分开维护。
- 当前继续采用构建时单语言生成，不实现运行时动态语言切换；不为未来需求过早引入数据库、Web 审校平台、第三方复杂 i18n 框架或其他重型基础设施。
- 下一步先对正式 Tour 运行时的公共 UI 可见英文文案完成一次完整只读审计，按 template、JavaScript、Go 服务端动态 UI、accessibility 文案、用户可见错误/状态信息及内部日志或开发者文本分类，明确清单和本地化边界后再决定 UI 修改范围；在此之前不直接开始大规模 UI 修改。
