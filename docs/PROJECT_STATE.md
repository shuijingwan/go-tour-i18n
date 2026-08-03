# 项目状态

更新时间：2026-08-03

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
- zh-CN glossary 对强制链接 label 使用 protected token 确定性恢复，例如 `A Tour of Go` → `Go 语言之旅`、`previous` → `上一页`、`next` → `下一页`、`Run` → `运行`、`Format` → `格式化`；validator 仍执行防御性术语与禁用译法检查。
- 已支持 ready candidate 本地预览。运行 `go run ./cmd/tour-i18n preview -id welcome/1 -locale zh-CN` 会在 `/tmp` 创建临时 Tour 内容副本，只替换目标中文 Section，不修改仓库正式 `_content`；其他未翻译页面继续显示英文。

## zh-CN 翻译进度

- 正式发布投影共 103 页，另保留 2 条条件源页面审计记录。
- 当前完成 8 页：`welcome/1`、`welcome/2`、`welcome/3`、`welcome/4`、`welcome/5`、`basics/1`、`basics/2`、`basics/3`；其余页面为 pending。
- `basics/3` 为 `ready`、Attempts=1。所有页面的原始模型 attempt 均完整保留。
- 当前已验证 GLM-5.2 在前三个正式 Basics 技术页面中没有出现技术性误译或结构损坏：`basics/1` 可直接通过，`basics/2`、`basics/3` 经轻微人工润色后定稿。

本阶段 glossary 已沉淀：`package` → `包`、`import path` → `导入路径`、`package name` → `包名`、`import statement` → `导入语句`、`exported name` → `导出名`、`unexported name` → `未导出的名称`。

## 本日完成

- 将 `welcome/1` 正式发布投影切换为远程服务器执行语义，同时保持本地原始 Tour 与 `/socket` 行为不变。
- 完成 `basics/1`、`basics/2`、`basics/3` 中文翻译及定稿。
- 修复 legacy present 行内代码保护，保留所有模型原始 attempt。

## 后续策略

- 不再长期按课程顺序逐页人工推进。下一步先从不同章节制定约 5～7 个代表页面的校准清单，而不是直接翻译 `basics/4`。
- 代表页覆盖较长说明、练习、特殊 present 结构、方法与接口、泛型、并发、图片或其他少见结构。
- 代表页稳定后自动试跑 10 个普通 pending 页面；试跑稳定后再批量翻译剩余页面。
- 全量翻译不等于直接发布：失败页面进入重试或 blocked，通过页面仍需统一验证。
