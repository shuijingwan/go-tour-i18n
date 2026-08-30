# fr-FR Locale Surface Review — 2026-08-30 首次 production 最终 evidence

这是 fr-FR 首次 production 的最终 Locale Surface Review evidence。Locale-level language quality review、complete preview rendered acceptance、production machine acceptance 与 production browser acceptance 均已完成并通过。

## 审核身份

- locale：`fr-FR`
- review_id：`20260830-first-production`
- reviewed commit：`7adaa3c99efd221b9d79c4bd492a6c88d665348c` (`7adaa3c`)
- local production bundle：`/tmp/go-tour-release-20260830-fr-FR-bd8df0a`
- remote production release：`/data/go-tour-fr-FR/releases/20260830-fr-FR-bd8df0a`
- upstream baseline：`golang/website` `master` 的 `b3fc6537086f09e88cb3c1ecd09bd47c31c54241`
- upstream commit time：`2026-08-26T21:55:26Z`
- production public identity：`https://fr-go-dev.shuijingwanwq.com/`
- date：`2026-08-30`

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article：`7`

### Glossary identity

- path：`locales/fr-FR/glossary.yaml`
- SHA-256：`bd3c4f96e956dd13784f42238f36cd5b97255cd5eeb1827394f37a633f9bb7fc`

### UI / metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`52cab0eaf41fd205d7096665dcb7aff76e1538d8036dc49a09f5cfc89e61c35b`

fr-FR UI target：

- path：`internal/tour/ui/fr-FR.json`
- SHA-256：`db027c9448686cd1f596d39863ec275344d2a514a7cf7fc5b3e6eba2141b18c9`

Article metadata：

- path：`locales/fr-FR/article-metadata.json`
- SHA-256：`c6d75b58dee1ab308fe617c3d4df5fea557a42e733623ceb0b107de156f56b4e`

Course metadata：

- path：`locales/fr-FR/course-metadata.json`
- SHA-256：`017a6b9d98f1a97440bab00b70bb0cd1b56d34d75cd6c591ef90f3b0dc8e87bd`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 Final Review。对 TranslationUnit 之外的 fr-FR locale-level 语言资产进行了完整 source ↔ target 审核，并以 `locales/fr-FR/glossary.yaml` 为正式术语基线。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/fr-FR.json`
- `locales/fr-FR/glossary.yaml`

检查全部 message key 的 source ↔ target 语义、`plain` / `rich` kind、占位符与 markup identity，以及 glossary mandatory / preferred / forbidden / keep 决策；覆盖首页、`/tour/`、`/tour/list`、导航与语言选择器、编辑器动作和状态、runtime message、footer、共享 shell 与 SEO 可见 UI 文案。

首轮 Surface Review 发现并修复：

- `site.unofficial`：补回英文 source 中 multilingual translation project 的“traduction”语义；
- `footer.unofficial`：同样补回“traduction”语义。

随后完成共享 UI Surface 修复：

- `editor.on` / `editor.off` 正式进入英文 source UI catalog 和全部正式 locale catalog；
- fr-FR 状态文案为 `Activé` / `Désactivé`；
- 编辑器状态由 Angular + UI catalog 渲染，不再依赖共享 CSS 硬编码英文 `on` / `off`。

该修复属于共享 UI Surface，不属于 TranslationUnit revision。

### Article metadata

完整对照当前 7 个正式 article 的英文根级 title / subtitle 与 fr-FR article metadata。

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

### Course SEO metadata

按每个 Page 的完整英文 source、最终 canonical fr-FR target 和 fr-FR glossary 审核正式 course metadata。

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

首轮 Surface Review 修正了 `basics/12` description，当前正式文本为：

> Découvrez les valeurs zéro attribuées aux variables sans initialisation explicite : 0 pour les types numériques, false pour le type booléen et "" (la chaîne vide) pour les chaînes de caractères.

除此之外没有遗留 fr-FR locale-level language quality blocker。

## B. Rendered surface acceptance — complete preview

结果：`passed`

### Preview identity

完整 fr-FR production-style preview 使用：

```sh
go run -mod=readonly ./cmd/tour-i18n preview --locale fr-FR
```

- locale：`fr-FR`
- ready：`122`
- pending：`0`
- blocked：`0`
- pages：`103`
- articles：`7`
- public canonical identity：`https://fr-go-dev.shuijingwanwq.com/`
- reviewed commit：`7adaa3c99efd221b9d79c4bd492a6c88d665348c`

### Browser acceptance coverage

实际人工浏览器验收覆盖：

- 首页 `/`；
- `/tour/`；
- `/tour/list`；
- 普通课程页；
- 带编辑器课程页；
- desktop；
- mobile `375 CSS px`；
- header / footer；
- navigation / pagination / TOC；
- language selector；
- editor；
- Run / Format / Reset；
- runtime message；
- theme；
- `robots.txt`；
- `sitemap.xml`；
- `html lang` / canonical / SEO wiring。

结果：`passed`。没有遗留 preview rendered blocker。

### Runtime / editor

- Run 实际执行成功，法语 runtime 可见文案正常；
- Format：`passed`；
- Reset：`passed`；
- editor 状态实际显示 `Coloration syntaxique Activé / Désactivé`；
- editor 状态实际显示 `Importations Activé / Désactivé`；
- CodeMirror 和既有 editor 行为未受共享状态本地化或移动布局修复影响。

### Shared Surface fixes

本次 preview 验收包含以下共享 Surface 修复：

1. CSS 硬编码英文 `on` / `off` 改为正式 UI catalog localization；
2. mobile lesson `pre` 改为保留原始文本、换行、连续空格和缩进语义的 `pre-wrap` 视觉自动换行，页面不再因长 `pre` 产生非预期横向 overflow；
3. 首页语言列表改为 registry 驱动的纵向 `English name — autonym` 文本链接展示。

首页实际语言列表保持 registry 顺序：

1. `Simplified Chinese — 简体中文`
2. `English`
3. `French — Français`
4. `German — Deutsch`
5. `Japanese — 日本語`

`French — Français` 为当前语言，保留 `aria-current="page"` 语义和加粗强调。English 继续指向官方 Tour，社区 locale 继续指向 registry 中各自既有 production URL。

### Layout / navigation / SEO

- mobile lesson `pre` 已视觉自动换行，lesson 和页面主容器无非预期横向 overflow；
- footer、header、navigation、pagination、TOC 和 theme 没有发现新的 layout blocker；
- sitemap 使用 `fr-go-dev.shuijingwanwq.com`；
- `html lang`、canonical 与 SEO wiring 使用 fr-FR production public identity；
- 浏览器控制台偶发 `inpage.js` / `contentScript.js` 错误已确认来自浏览器扩展注入脚本，不属于仓库、preview 或 release blocker。

### Automated verification

以下自动验证已完成并通过：

- `go test ./...`：`passed`；
- `git diff --check`：`passed`；
- fr-FR opt-in Chrome browser test：`passed`；
- `/tour/basics/11` mobile `375px`：`passed`；
- `/tour/moretypes/1` mobile `375px`：`passed`；
- desktop editor interaction：`passed`；
- homepage mobile `375px` browser test：`passed`。

## Production verification

结果：`passed`

### Release / deployment identity

- local bundle：`/tmp/go-tour-release-20260830-fr-FR-bd8df0a`；
- remote release：`/data/go-tour-fr-FR/releases/20260830-fr-FR-bd8df0a`；
- public URL：`https://fr-go-dev.shuijingwanwq.com/`；
- service：`go-tour-fr-FR.service`；
- loopback：`127.0.0.1:4002`。

首次 deployment 通过 `scripts/deploy-production.sh` 完成。远端 SHA-256 与 permissions validation passed；`current` 已切换到上述正式 release；service 为 `active`；localhost health 连续 3 次 HTTP 200；public acceptance HTTP 200。

### DNS / TLS / Nginx

- Cloudflare Free 已启用，DNS 已公开解析到 Cloudflare；
- HTTP → HTTPS 返回 301；
- Let's Encrypt 证书 hostname 为 `fr-go-dev.shuijingwanwq.com`；
- fr-FR Nginx vhost 使用独立 port 80 server → 301 HTTPS，以及独立 443 server → `127.0.0.1:4002`；
- 自动生成且会截获静态资源的额外 location 已删除；
- `nginx -t` passed。

### Playground Origin

精确 allowlist Origin `https://fr-go-dev.shuijingwanwq.com` 已配置并实际验收：

- fr-FR `OPTIONS /compile`、`OPTIONS /fmt`：204；
- 既有 zh-CN、ja-JP、de-DE Origin：继续 204；
- wrong Origin：403；
- `GET /compile`、`GET /fmt`：405；
- `POST /compile`：200；
- `POST /fmt`：200。

### AdSense production preparation

- `/etc/go-tour/go-tour.env`：`root:root`、mode `600`；
- `TOUR_AD_HTML` 存在有效非空配置；
- `go-tour-fr-FR.service` 正确引用该 EnvironmentFile；
- shared course-ad assets 使用既有非中文 shared-assets production 基线；
- 首次启动即为正式 ads-enabled 形态。

### Production machine acceptance

执行：

```sh
scripts/verify-production.sh /tmp/go-tour-release-20260830-fr-FR-bd8df0a
```

实际结果：

- release identity：PASS；
- remote identity：PASS；
- source routes：7/7 PASS；
- public routes：7/7 PASS；
- html identity：PASS；
- sitemap URLs：105/105，host mismatch 0，HTTP failure 0；
- sitemap：105/105 PASS；
- socket boundary：PASS；
- CDN `/`：`MISS -> HIT -> HIT PASS`；
- CDN `/tour/welcome/1`：`MISS -> HIT -> HIT PASS`；
- `PRODUCTION MACHINE ACCEPTANCE: PASS`。

### Production browser acceptance

人工浏览器完成正式 production 验收，结果为 `passed`，没有发现 blocker。覆盖首页、`/tour/`、`/tour/list`、课程页、导航、语言选择器、editor、Run / Format / Reset、runtime、desktop/mobile 与长代码页面等首次 production Surface Review 必需范围。

Playground 实际功能正常；移动端页面布局正常；`/tour/moretypes/1` 长代码回归没有整页横向 overflow。

### Lightweight ads confirmation

实际 production 已出现真实 filled course ad，证明存在真实广告请求和填充机会。课程页面、footer 和 SPA 下一页未发现阻塞性布局或功能异常；filled / unfilled 不作为 gate。

发现一个非阻塞共享视觉 follow-up：课程广告区域左上方存在一个单独的 `=` 字符，广告尚未填充时尤其明显，填充后仍可见但影响较小。本次不修复，不作为 fr-FR production blocker；该问题属于 shared course-ad visual follow-up，不是 TranslationUnit、fr-FR 翻译或 locale-specific 缺陷。

## Reviewer

- TranslationUnit Quality Check / Final Review：ChatGPT GPT-5.6 Sol
- Locale-level language quality review：ChatGPT GPT-5.6 Sol
- Rendered browser acceptance：人工浏览器验收 + Codex GPT-5.6 Sol 分析
- 最终 evidence 整理：Codex GPT-5.6 Sol（轻度）

## Issues

unresolved language blocker：`none`

unresolved preview rendered blocker：`none`

unresolved production blocker：`none`

non-blocking follow-up：shared course-ad 区域左上方的单独 `=` 字符；不属于 TranslationUnit、fr-FR 翻译或 locale-specific 缺陷。

## Current gate status

- A language quality review：`passed`
- preview rendered acceptance：`passed`
- production verification：`passed`
- final decision：`passed`

`decision = passed`
