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

状态：`passed`

### Production identity

最终验收 release：

- source commit：`916a23e45210f25e751b84b40b7c354d5a839dac`
- release：`/data/go-tour/releases/20260827-zh-CN-916a23e4`
- published_at：`2026-08-27T09:55:54Z`
- public URL：`https://go-dev.shuijingwanwq.com/`
- locale：`zh-CN`
- TranslationUnit：`122`
- Page：`103`
- eligible Example：`19`
- article：`7`
- execution transport：`http-playground-proxy`
- execution provider：`play.golang.org`
- local socket：disabled

部署后确认：

- `go-tour.service` 为 `active`；
- 源站 `http://127.0.0.1:3999/` 返回 200；
- deploy lock 不存在；
- 公网 `/`、`/tour/`、`/tour/list` 均返回 200。

### CDN 与公网 HTML

首次部署后检查发现 EdgeOne 首页仍命中旧缓存：

- `EO-Cache-Status: HIT`
- `Age` 为旧缓存年龄；
- 公网首页仍显示旧的 `A Tour of Go` 项目标题。

随后按 production runbook 对 `go-dev.shuijingwanwq.com` 执行整站缓存清除。

清除后确认：

- 首页重新回源并返回 `EO-Cache-Status: MISS`；
- 公网首页 title、description、canonical 与当前源站一致；
- 首页显示 `Go 语言之旅`；
- `/tour/methods/14` title、description、canonical 与当前 source 一致。

后续重新部署最终修复 release 后，再次确认公网首页已经包含当前语言版本入口。

### Production SEO 与 routing

真实公网确认：

- `/robots.txt` 正确指向 `https://go-dev.shuijingwanwq.com/sitemap.xml`；
- sitemap URL 总数：`105`；
- unique URL：`105`；
- wrong host：`0`；
- `/socket` 返回 404；
- canonical 使用正式 zh-CN hostname；
- 公网页面没有使用 localhost identity。

### Playground production verification

浏览器 production Run / Format 使用正式 Playground 链路。

命令行真实请求确认：

- compile endpoint：
  `https://play.go-dev.shuijingwanwq.com:8443/compile?backend=`
- fmt endpoint：
  `https://play.go-dev.shuijingwanwq.com:8443/fmt`
- Origin：
  `https://go-dev.shuijingwanwq.com`

真实 compile 成功并输出：

`Hello, 世界`

真实 Format 将：

`func main(){}`

格式化为：

`func main() {}`

浏览器中：

- Run 正常；
- Format 正常；
- Reset 正常；
- 制造真实语法错误后，Go 编译器返回原始英文技术错误；
- 页面本地化 runtime message 显示 `Go 构建失败。`。

第一次浏览器测试时曾看到英文 `Go build failed.`。检查确认 production source、catalog 与前端实现均为 zh-CN，本次现象来自部署前已经打开的旧浏览器页面状态。强制刷新 production 页面后重新测试，runtime message 正确显示 `Go 构建失败。`，因此没有遗留本地化缺陷。

### Desktop / mobile production smoke check

桌面端检查：

- `/`
- `/tour/`
- `/tour/list`
- `/tour/methods/14`

确认：

- 无重叠；
- 无异常截断；
- 无异常页面级横向滚动；
- 导航正常；
- language selector 正常；
- theme control 正常。

移动端使用约 `390×844` viewport 检查课程页：

- editor 可操作；
- Run 正常；
- Format 正常；
- Reset 正常；
- 导航、按钮、footer 无明显重叠或布局异常。

测试过程中出现过示例自身的：

`panic: interface conversion: interface {} is string, not float64`

该结果证明程序已经成功编译并运行；这是示例中的运行时类型断言 panic，不是 Playground 不支持 `any`，不属于发布缺陷。

### SPA production acceptance

真实点击：

`methods/14 → methods/15`

确认：

- SPA 下一页正常；
- 页面正文更新；
- title 更新；
- description 更新；
- canonical 更新；
- 没有上一页 metadata 残留。

### Language selector production issue and resolution

production smoke check 发现首页语言版本中的日本語链接原先指向：

`https://ja-go-dev.shuijingwanwq.com/tour/welcome/1`

按当前站点入口约定，应指向该 locale 的项目首页：

`https://ja-go-dev.shuijingwanwq.com/`

同时将 zh-CN registry 统一为：

`https://go-dev.shuijingwanwq.com/`

修复提交：

`916a23e45210f25e751b84b40b7c354d5a839dac`

修复后：

- registry test 通过；
- 新 zh-CN production release 已重新 publish / deploy；
- 源站首页日本語链接正确；
- EdgeOne 公网首页日本語链接也已确认正确。

该 production-only Surface Review 问题已解决，无遗留 blocker。

### Lightweight AdSense verification

production HTML 确认：

- AdSense loader 存在；
- course ad mount 存在。

真实浏览器检查进一步确认：

- `adsbygoogle.js` loader：`1`；
- course ad mount 正常；
- `ins.adsbygoogle` 中存在 `filled` 广告；
- 同时允许存在 `unfilled` 广告；
- 存在真实 Google AdSense / DoubleClick iframe；
- 存在多条真实广告网络请求；
- desktop course ad 实际获得有效尺寸；
- mobile viewport 下也产生真实响应式广告请求；
- 页面高度与 footer 无明显异常；
- SPA 下一页后继续存在新的广告请求机会。

本次只执行 locale production 所需的轻量广告确认，没有重复执行共享广告架构完整 regression。

### Production verification result

`passed`

最终 production release、CDN、公网 HTML、SEO、routing、Playground、runtime message、desktop/mobile、SPA、language selector 与轻量广告确认均已通过。

## Reviewer

- ChatGPT：Locale-level language quality review、course metadata 全量语言审核、审核结果分析
- Codex：本地工程修改、preview/browser 验收、确定性浏览器回归执行
- 王强：本地仓库执行、结果确认、正式发布与 production 操作

## Issues

当前没有未解决的 release blocker。

本次 production verification 中发现并已解决：

- EdgeOne 首页旧缓存：执行 hostname 整站缓存清除后恢复当前内容；
- language selector 日本語入口指向课程页：修复为 locale 项目首页并重新部署；
- 浏览器旧页面曾显示英文 `Go build failed.`：强制刷新当前 production release 后复测，正确显示 `Go 构建失败。`。

A 阶段、Preview acceptance 与 Production verification 均无遗留阻塞问题。

## Decision

`decision = passed`

Locale-level language quality review、Preview rendered surface acceptance 与最终 Production verification 均已通过。

本次 zh-CN upstream 同步后的正式 release 已完成最终 Locale Surface Review，可以进入日常维护状态。
