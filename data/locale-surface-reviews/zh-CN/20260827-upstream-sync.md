# zh-CN Locale Surface Review — 2026-08-27 upstream 同步

## 审核身份

- locale：`zh-CN`
- review_id：`20260827-upstream-sync`
- reviewed commit：`f9567f72de44a003b63acf329c51203db5f8f04f`
- upstream baseline：`b3fc6537086f09e88cb3c1ecd09bd47c31c54241`
- date：`2026-08-27`

### Glossary identity

- path：`locales/zh-CN/glossary.yaml`
- SHA-256：`0f8c5619ed8288ae9df2a380b2b98d34df1e05d36e846517de36f0b75f218e96`

### UI / Metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`98a2ab79bd319da815b6e592b4e3131abbcd18356e771ab2fbbd3db2794a3588`

zh-CN UI target：

- path：`internal/tour/ui/zh-CN.json`
- SHA-256：`1facaf66f4d626844ddd30d97d49fe8668a4ab431b2053c734c415774bad0956`

Article metadata：

- path：`locales/zh-CN/article-metadata.json`
- SHA-256：`b35eec52636238836fb9d904dbf03a627a2ccd95dc4279567e13001f32b836a6`

Course metadata：

- path：`locales/zh-CN/course-metadata.json`
- SHA-256：`8d5908ab51a7b3483552cabbcb9e7f39f6778007588ce065cc02b9aff872ef06`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 Final Review，对 TranslationUnit 之外的 zh-CN locale-level 可翻译资产进行了完整语言质量审核。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/zh-CN.json`
- `locales/zh-CN/glossary.yaml`

按全部 message key 检查：

- source ↔ target 语义完整性；
- `plain` / `rich` identity；
- glossary 的 mandatory、preferred、forbidden、keep；
- 普通英文残留；
- 首页文案；
- 导航与编辑器动作；
- runtime message；
- shell 可见文案。

第一轮审核发现并修复了包括以下问题：

- `A Tour of Go` 与 glossary mandatory `Go 语言之旅` 不一致；
- `eligible` 普通英文残留；
- 部分 source 含义被弱化；
- `tour` 相关动作与 glossary preferred 用词不一致。

随后执行 shell source identity 审计，又发现了绕过正式 UI catalog 的用户可见文案：

- 首页项目卡片硬编码 `A Tour of Go`；
- HTTPTransport 的 vet/build/通信错误英文；
- test result 的成功/失败英文。

这些文案均已进入正式 UI catalog / JavaScript bootstrap 本地化链。

最终：

`shell source identity closure = passed`

English、zh-CN、ja-JP UI catalog 的 message identity 与 kind 保持一致。

### Article metadata

完整对照当前 7 个正式 article 的英文根级 `title`、`subtitle` 与 zh-CN article metadata。

结果：

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

### Course SEO metadata

对当前全部正式课程 Page 逐页审核 description。

每一个 Page 均同时读取：

- 完整英文 Page TranslationUnit source；
- 当前正式 canonical zh-CN Page target；
- 完整 zh-CN glossary；
- 当前正式 course metadata description。

结果：

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

本次 upstream / canonical 更新后增量刷新的三个 Page：

- `flowcontrol/1`
- `methods/14`
- `methods/21`

也已独立完成语言审核并通过。

### 其他 locale-level surface

已完成 source identity 闭环：

- 首页 `/`
- `/tour/`
- `/tour/list`
- 课程页导航
- 编辑器
- runtime message
- language selector
- SEO shell

语言选择器中的 autonym、locale、URL 等属于稳定语言身份配置，不作为未翻译英文处理。

A 阶段最终没有遗留未解决的 locale-level 语言问题。

## B. Rendered surface acceptance

结果：`passed`

### Projection identity

完整 zh-CN projection：

- ready TranslationUnit：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible translated Example：`19`
- article：`7`

### Preview 架构

最终完整 locale preview 使用 production-style preview handler：

- HTTPTransport；
- 浏览器同源 `/_/compile`；
- 浏览器同源 `/_/fmt`；
- preview server-side proxy 转发到真实 Go Playground；
- `/socket` 禁用并返回 404；
- canonical 使用正式 locale public identity；
- 正式 robots / sitemap handler；
- 不注入 production analytics；
- 不注入 production AdSense。

单个 candidate 的：

`preview --locale <locale> --id <page-id>`

仍保持独立的本地 SocketTransport 开发预览语义。

### HTTP 与 SEO

代表路由检查：

- `/`
- `/tour/`
- `/tour/list`
- `/tour/welcome/1`
- `/tour/methods/14`
- `/tour/concurrency/11`
- `/robots.txt`
- `/sitemap.xml`

确认：

- `html lang=zh-CN`；
- 页面 title 正确；
- course description 正确；
- 首页 description 正确；
- canonical 正确；
- 无 localhost canonical 泄漏；
- 无其他 locale hostname 泄漏。

首页现在包含：

- 来自审核通过 UI catalog 的本地化 meta description；
- canonical `https://go-dev.shuijingwanwq.com/`。

### robots.txt 与 sitemap.xml

确认：

- `/robots.txt` 返回 `text/plain`；
- robots 指向正式 zh-CN sitemap；
- `/sitemap.xml` 返回 XML；
- sitemap 共 `105` 个不重复 URL：
  - 首页；
  - `/tour/list`；
  - 103 个正式课程 Page；
- sitemap 使用正式 zh-CN public host。

### Desktop rendered acceptance

使用桌面 viewport 检查：

- 首页；
- `/tour/`；
- `/tour/list`；
- first Page；
- middle Page；
- last Page；
- 带编辑器 Page。

确认：

- 无页面级异常横向滚动；
- header / footer 布局正常；
- 首页项目卡片显示 `Go 语言之旅`；
- 可见 UI 本地化正确；
- 导航正常；
- Table of Contents 正常；
- feedback 正常；
- language selector 正常；
- theme control 正常。

### Mobile rendered acceptance

使用 `390×844` mobile viewport 检查：

- 首页；
- `/tour/`；
- `/tour/list`；
- 普通课程 Page；
- 带编辑器课程 Page。

确认：

- 无页面级异常横向滚动；
- header / footer 无重叠；
- editor 可见且可操作；
- 导航正常；
- language selector 正常；
- theme control 正常；
- 较长 zh-CN 文案可以正常换行。

### 导航与 SPA metadata

完成 first / middle / last Page 的代表性导航检查。

真实执行：

`methods/14 → methods/15`

SPA 切页后以下内容同步更新：

- 页面正文；
- `document.title`；
- meta description；
- canonical URL。

未发现上一页 identity 残留。

### Playground 真实交互

最终 production-style localhost preview 使用 HTTPTransport，并经 preview server-side proxy 访问真实 Go Playground。

浏览器实际请求：

- same-origin `POST /_/compile`
- same-origin `POST /_/fmt`

真实验收确认：

- 合法 Run 成功；
- 示例输出包含 `Hello, 世界`；
- 程序结束文案显示为中文；
- 制造真实编译错误后显示 `Go 构建失败。`；
- Format 将 `func main(){}` 格式化为 `func main() {}`；
- Reset 恢复当前 Page 的原始代码。

这些操作使用真实 Go Playground，不是 fake upstream。

### Runtime message 受控浏览器回归

另外使用真实 HTTPTransport shell + 本地受控 Playground response 的确定性 Chrome browser test，验证以下 zh-CN browser-visible runtime message：

- `Go vet 检查失败。`
- `Go 构建失败。`
- `与远程服务器通信时出错。`
- `1 个测试失败。`
- `2 个测试失败。`
- `所有测试均已通过。`

该受控 browser regression 与上面的真实 Playground Run / Format 验收相互独立。

### 广告边界

Preview rendered surface acceptance 不要求真实 AdSense。

本轮没有重复执行共享广告架构的完整 regression。

确认本次 UI/runtime/preview 修改没有造成明显的页面高度或 footer 布局异常。

## Production verification

状态：`pending`

与本次 Surface Review 对应的新 production release 尚未部署，因此 production verification 尚未执行。

仍待真实 production 验证：

- production release identity；
- 公网 HTTPS 页面；
- CDN / hostname；
- production canonical；
- production `robots.txt`；
- production sitemap；
- 真实 production Playground endpoint；
- 精确 production Playground Origin；
- desktop / mobile production smoke check；
- 轻量广告确认。

在这些检查真实执行之前，不记录 production verification 为 passed。

## Reviewer

- ChatGPT：Locale-level language quality review、course metadata 全量语言审核、审核结果分析
- Codex：本地工程修改、preview/browser 验收、确定性浏览器回归执行
- 王强：本地仓库执行、结果确认、正式发布与 production 操作

## Issues

当前唯一尚未完成的 release gate：

- production verification 尚未执行。

A 阶段没有遗留语言质量问题。

Preview acceptance 没有遗留阻塞问题。

## Decision

`decision = failed`

原因：

Locale-level language quality review 与 preview acceptance 均已通过，但 production verification 仍为 pending。

按照 Locale Surface Review 正式规则，在真实 production release 完成最终复核以前，整体 decision 不能写为 `passed`。
