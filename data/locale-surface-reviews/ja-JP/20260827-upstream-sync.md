# ja-JP Locale Surface Review — 2026-08-27 upstream 同步

## 审核身份

- locale：`ja-JP`
- review_id：`20260827-upstream-sync`
- reviewed commit：`800a0e7330c380d43627ef4d00b647cd27930ffb`
- upstream baseline：`b3fc6537086f09e88cb3c1ecd09bd47c31c54241`
- date：`2026-08-27`

### Glossary identity

- path：`locales/ja-JP/glossary.yaml`
- SHA-256：`42d1cc8edcf027dc3a97b7775db1fc74781289b7fc8548508d3937b186bd1af4`

### UI / Metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`98a2ab79bd319da815b6e592b4e3131abbcd18356e771ab2fbbd3db2794a3588`

ja-JP UI target：

- path：`internal/tour/ui/ja-JP.json`
- SHA-256：`18e16b66cfa0f78092e3b9ff5a84fc86a7dd77f348c928b4dc6bec39ad65309e`

Article metadata：

- path：`locales/ja-JP/article-metadata.json`
- SHA-256：`f9c5a7f0cd2a164241e0133abdf77b4ad80fc991aed16fc78d6f1cfaa9a3cd5d`

Course metadata：

- path：`locales/ja-JP/course-metadata.json`
- SHA-256：`4614cf5de9c62b54e0eb6008eb82724da1fc2bafaa4343a140c297d3b0e1d523`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 Final Review，对 TranslationUnit 之外的 ja-JP locale-level 可翻译资产进行了 source ↔ target 语言质量审核。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/ja-JP.json`
- `locales/ja-JP/glossary.yaml`

检查范围包括：

- 全部 message key 的 source ↔ target 语义；
- glossary mandatory / preferred / forbidden / keep；
- 首页；
- `/tour/`；
- `/tour/list`；
- 导航与语言选择器；
- 编辑器 Run / Format / Reset；
- runtime message；
- footer 与共享 shell；
- SEO 可见 UI 文案。

结果没有遗留未解决的 ja-JP 语言问题。

`A Tour of Go` 按 ja-JP glossary mandatory 使用 `Go 言語ツアー`。旧 production 首页项目卡片仍显示 `A Tour of Go`，确认属于旧 release；当前共享模板已经通过 `tour.title` 使用正式 ja-JP UI catalog。

共享 CSS 生成的 `on` / `off` 按 `docs/PROJECT_STATE.md` 已有项目级主动保留决策处理，不属于 locale 未翻译残留，也不作为本次审核阻塞项。

### Article metadata

完整核对当前 7 个正式 article 的英文根级 title / subtitle 与 ja-JP article metadata。

结果：

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

本次 upstream 涉及的 `flowcontrol.article` 与 `methods.article` metadata 均保持准确。

### Course SEO metadata

本次 upstream / canonical identity 变化后实际 stale 的 Page：

- `flowcontrol/1`
- `methods/14`

已分别基于完整英文 Page source、promotion 后 ja-JP canonical target 与完整 ja-JP glossary，使用 Codex GPT-5.6 Sol High 刷新 description，并完成语言审核。

结果：

- stale reviewed：`2/2`
- passed：`2/2`
- failed：`0`

其余 101 个 Page metadata identity 未变化。

## B. Rendered surface acceptance

结果：`passed`

### Projection identity

完整 ja-JP preview：

- TranslationUnit ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible translated Example：`19`
- article：`7`
- preview：`http://127.0.0.1:4010/`
- public canonical identity：`https://ja-go-dev.shuijingwanwq.com/`

### Desktop acceptance

实际浏览器检查：

- `/`
- `/tour/` → `/tour/welcome/1`
- `/tour/list`
- `/tour/methods/14`
- `/tour/flowcontrol/1`

确认：

- 首页项目卡片显示 `Go 言語ツアー`；
- upstream baseline 已更新为 `b3fc6537…`；
- 首页、课程页、列表页日文显示正常；
- article title / subtitle 正常；
- header、footer、编辑器和导航无明显布局异常；
- 本轮两个 upstream 变化 Page 的新译文均正确进入 rendered surface。

### Playground interaction

在 production-style preview 实际检查：

- Run：`passed`
- Format：`passed`
- Reset：`passed`

`flowcontrol/1` Run 正常执行。

测试非法 Go 代码时，Format 正确返回：

`expected '}', found 'EOF'`

该结果属于预期的 Go formatter 语法错误，不是 preview 故障。

### SPA navigation

实际执行：

`/tour/flowcontrol/1 → /tour/flowcontrol/2`

确认：

- SPA 下一页正常；
- URL 更新为 `/tour/flowcontrol/2`；
- Page 从 `1/14` 更新为 `2/14`；
- 正文与编辑器内容同步切换；
- 未发现上一页内容残留。

### Mobile acceptance

使用 `393×852` viewport 检查：

- `/`
- `/tour/flowcontrol/1`
- `/tour/flowcontrol/2`

第一轮发现共享首页移动端两个布局问题：

1. 长 locale 标题允许换行后，固定 header 高度造成第二行超出背景；
2. header 和 Hero 多行标题行距过紧。

已在共享 `_content/tour/static/css/app.css` 修复：

- mobile `.site-header` 改为自适应高度并保留 `min-height: 48px`；
- mobile `.site-header .logo` 使用 `line-height: 1.25`；
- `.site-hero h1` 使用 `line-height: 1.25`。

修复 commit：

`800a0e7330c380d43627ef4d00b647cd27930ffb`

重建完整 preview 后复核：

- header 长标题完整显示；
- Hero 两行主标题行距自然；
- 无异常横向滚动；
- 编辑器与操作按钮正常；
- SPA 下一页正常；
- footer 正常。

课程页分页背景与页码存在轻微视觉贴近感，但无截断、遮挡、重叠或交互问题，判定为可接受的非阻塞视觉差异。

## Production verification

状态：`pending`

当前 production：

- public URL：`https://ja-go-dev.shuijingwanwq.com/`
- 当前已知旧 release：`/data/go-tour-ja-JP/releases/20260824-ja-JP-164fecdd`

本次 `800a0e7` 尚未 publish / deploy，因此 production 仍可能显示旧 upstream baseline 与旧首页共享模板。

待完成：

- publish 新 ja-JP release；
- 使用既有 locale 日常维护部署流程部署；
- 公网 `/`、`/tour/`、`/tour/list` 与本轮两个变更 Page smoke check；
- production Playground Run / Format / Reset；
- desktop / mobile 受影响范围复核；
- canonical / robots / sitemap / hostname 检查；
- 按现有共享广告实现进行轻量 production 广告确认，不重跑完整广告 regression。

## Reviewer

- TranslationUnit Quality Check / Final Review：ChatGPT GPT-5.6 Sol
- Locale-level language quality review：ChatGPT GPT-5.6 Sol
- Rendered browser acceptance：人工浏览器验收 + ChatGPT 分析

## Issues

当前无未解决的语言质量或 preview rendered 阻塞问题。

production verification 尚未执行。

## Decision

`decision = failed`

这是部署前的临时工作状态：A 阶段与 preview acceptance 均已 `passed`，仅因为 production verification 尚未执行，所以不能提前写最终 `passed`。production 复核完成后更新同一 evidence。
