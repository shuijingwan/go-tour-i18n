# 项目状态

更新时间：2026-08-09

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

## zh-CN 翻译进度

- 正式发布投影共 103 页，另保留 2 条条件源页面审计记录。
- 当前正式页面 103 页，其中 ready 12 页、pending 91 页、blocked 0 页；新增完成 `methods/20`。
- `generics/1` 为 `ready`、Attempts=5。五次 attempt 均完整保留：依次暴露网络沙箱失败、token 重复、inline-code 边界失败、token 换序，最终第五次通过全部自动校验。
- `flowcontrol/8` 为 `ready`、Attempts=1。GLM-5.2 第一次翻译即通过 token、present 和结构校验；本页覆盖 legacy inline code、两个链接、预格式化代码、注意段和 `.play`。
- `methods/16` 为 `ready`、Attempts=2。attempt-001 因模型翻译教学代码注释而被旧版预格式化代码逐字校验拒绝；增强安全注释翻译与强调结构校验后，attempt-002 通过，人工润色、candidate validate 和浏览器预览均已完成。
- `methods/20` 为 `ready`、Attempts=1。attempt-001 的 GLM-5.2 API 调用成功、`finish_reason=stop`，并完整保留 16 个 protected token；旧版全局 token 顺序规则因 link target / `Sqrt`、`error` / 预格式化代码块、`fmt.Sprint(e)` / `Error` 三组自然中文语序调整而错误拒绝。分析确认技术含义与 present 结构均未改变；通用规则修复后对原 response 的确定性回放通过完整 candidate validation，未调用 attempt-002。经人工润色、candidate validate、本地 HTTP 预览和浏览器人工检查后进入 ready；attempt-001 原始失败 response/validation 历史证据继续保留。
- 当前已验证 GLM-5.2 在前三个正式 Basics 技术页面中没有出现技术性误译或结构损坏：`basics/1` 可直接通过，`basics/2`、`basics/3` 经轻微人工润色后定稿。
- `generics/1` 已完成人工润色和重新校验，并通过本地 HTTP 与浏览器预览。
- `flowcontrol/8` 自动校验通过后，人工发现并修正了“找到最接近 x 的 z”这一技术含义偏差；润色后的 candidate validate 通过，本地 HTTP 和浏览器预览正常。

本阶段 glossary 已新增 preferred 术语：`standard library` → `标准库`、`iteration` → `迭代`、`loop condition` → `循环条件`。新增 mandatory 术语：`type switch`、`type switches` → `类型选择`，`type assertion`、`type assertions` → `类型断言`，`interface value` → `接口值`，`interface type` → `接口类型`，`concrete type` → `具体类型`。`square root`、`Newton's method`、`derivative` 等单页术语暂不加入。

`generics/1` attempt-004 与 `methods/20` attempt-001 共同证明，全局 protected token 顺序要求会误伤正确的跨语言自然语序；本次调整是通用多语言恢复与候选校验修复，而非 `methods/20` 的单页特例。

## 本日完成

- 完成 `methods/20` 代表页的完整源与 present 结构分析。
- 完成 `methods/20` attempt-001：GLM-5.2 成功返回，但暴露 protected token 全局顺序会误判跨语言自然语序的问题。
- 完成通用 token 恢复与 candidate 校验规则修复，并以 `methods/20` 保存的真实 response 进行确定性回放且通过完整校验。
- 完成 `methods/20` 人工润色、candidate validate、本地 HTTP 预览和浏览器人工检查；页面最终为 ready / Attempts=1，未创建 attempt-002，attempt-001 的历史失败证据保留。
- 已同步 `TestCommittedStatus` 基线；`go test ./internal/i18n` 完整通过。

## 后续策略

- 不再长期按课程顺序逐页人工推进。代表性校准页面总序列已固定为：`generics/1`、`flowcontrol/8`、`methods/16`、`methods/20`、`concurrency/7`、`concurrency/11`、`methods/24`；前四页已 ready。
- 第 5 页为 `concurrency/7`：唯一包含 `.image` 的正式课程页面，覆盖 `.image` directive、图片与正文混排、非尾部 directive 的 Section 归属，以及 `javascript:click('.next-page')` 链接。
- 第 6 页为 `concurrency/11`：长篇多段说明与高密度链接组合，覆盖大量 link target、跨段链接、`slides` 强制 glossary label，以及无代码/directive 的长文本翻译。
- 第 7 页为 `methods/24`：API 说明型页面，覆盖较长静态 interface 代码块、多个 API 链接、强调段与尾部 `.play` 的组合。
- 完成上述三页后代表页总数达到 7 页；当前官方源没有 `.iframe` 或 `.code` 页面可供真实代表页验证，代表页校准阶段结束，随后自动试跑 10 个普通 pending 页面；试跑稳定后再批量翻译剩余页面。
- 全量翻译不等于直接发布：失败页面进入重试或 blocked，通过页面仍需统一验证。
