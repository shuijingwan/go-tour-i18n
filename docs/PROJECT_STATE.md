# 项目状态

更新时间：2026-08-02

## 基线与架构

- 官方 upstream 基线为 `golang/website` 的 `master` 分支 commit `e11dacba76c5aae474746e9eedee19693f492803`；翻译运行时使用仓库内固定的最小 Tour 源码闭包，外部 checkout 仅用于同步与校验。
- 当前目录包含 101 个 standalone 页面和 2 个单独记录的 `#appengine:` 条件页面。
- 唯一维护 CLI 是 `cmd/tour-i18n`。
- 页面身份使用 `data/tour-pages.tsv` 中冻结的持久 `page_id`，不会以页面位置或临时语义 key 替代。
- 项目根 `.env` 的安全加载已完成：系统环境变量优先，`.env` 不覆盖已有值；`/.env` 被 Git 忽略，`.env.example` 可提交。

## 翻译执行与校验

- GLM-5.2 单页完整 `present.Section` 翻译执行器已完成；request、response 和 validation 按 locale、page_id、source hash 与 attempt 编号保留。
- 正式模式对一个 source 最多执行 3 次完整页面 attempt；失败后进入 `blocked`，blocked 页面不能继续正式重试。
- `translate run -dev` 仅用于单页开发校准：每条命令只执行 1 次 attempt，可从 pending 或 blocked 继续且不受正式三次上限约束；失败回到 pending，成功进入 ready。正式批量翻译不得使用 dev 模式。
- standalone 投影已修复：`welcome/1` 只保留 `your computer.` 分支，不含 `#appengine:` 或 remote-server 分支。candidate validator 会拒绝 standalone source/candidate 中的 appengine 内容，并继续校验 present 结构、directive、链接 target、行内代码和预格式化代码。
- zh-CN glossary 对强制链接 label 使用 protected token 确定性恢复，例如 `A Tour of Go` → `Go 语言之旅`、`previous` → `上一页`、`next` → `下一页`、`Run` → `运行`、`Format` → `格式化`；validator 仍执行防御性术语与禁用译法检查。
- 已支持 ready candidate 本地预览。运行 `go run ./cmd/tour-i18n preview -id welcome/1 -locale zh-CN` 会在 `/tmp` 创建临时 Tour 内容副本，只替换目标中文 Section，不修改仓库正式 `_content`；其他未翻译页面继续显示英文。

## 首个 zh-CN 页面

- `welcome/1` 当前状态为 `ready`，累计 attempt 数为 6。
- Candidate：`locales/zh-CN/candidates/welcome-1.article`
- 当前 source SHA-256：`3fbd64163f0301d60fcf1440c8aa65a79358e7028fec433aee49ae0c364d3034`
- 当前 source 的 `attempt-001` 至 `attempt-006` 完整保留在 `data/translation-runs/zh-CN/welcome/1/sources/3fbd64163f0301d60fcf1440c8aa65a79358e7028fec433aee49ae0c364d3034/`，作为开发审计记录；早期错误 source 的历史记录也保留在原路径。

当前下一步：评估 DeepSeek 新版本的翻译质量。
