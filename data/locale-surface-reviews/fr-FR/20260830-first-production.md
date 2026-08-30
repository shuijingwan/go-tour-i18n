# fr-FR Locale Surface Review — 2026-08-30 首次 production 工作记录

这是 fr-FR 首次 production 的 Surface Review 工作记录，不是 production 完成后的最终 evidence。Locale-level language quality review 与 complete preview rendered acceptance 已完成并通过，允许进入 publish / first production；production verification 尚未执行，待真实首次 production 完成后在同一路径补写实际 evidence 和最终 decision。

## 审核身份

- locale：`fr-FR`
- review_id：`20260830-first-production`
- reviewed commit：`7adaa3c99efd221b9d79c4bd492a6c88d665348c` (`7adaa3c`)
- upstream baseline：`golang/website` `master` 的 `b3fc6537086f09e88cb3c1ecd09bd47c31c54241`
- upstream commit time：`2026-08-26T21:55:26Z`
- intended production public identity：`https://fr-go-dev.shuijingwanwq.com/`
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

结果：`pending`

production verification 尚未执行。本工作记录不声明或推断以下尚待真实首次 production 完成的项目：

- production release path；
- service 状态；
- TLS / vhost 实际结果；
- DNS / CDN 实际结果；
- Playground Origin 配置与验收结果；
- AdSense production 接入结果；
- production machine acceptance；
- production browser acceptance；
- lightweight ads confirmation。

完成真实首次 production 后，必须在本文件继续补写 production release identity、源站与公网 machine acceptance、真实浏览器与 Playground Network 验收、lightweight ads confirmation 和其他实际结果；全部必需阶段完成后，才可填写正式最终 decision。

## Reviewer

- TranslationUnit Quality Check / Final Review：ChatGPT GPT-5.6 Sol
- Locale-level language quality review：ChatGPT GPT-5.6 Sol
- Rendered browser acceptance：人工浏览器验收 + Codex GPT-5.6 Sol 分析
- 本工作记录整理：Codex GPT-5.6 Sol（轻度）

## Issues

unresolved language blocker：`none`

unresolved preview rendered blocker：`none`

production verification：`pending`

## Current gate status

- A language quality review：`passed`
- preview rendered acceptance：`passed`
- production verification：`pending`
- final decision：尚未给出

本记录当前仅确认 fr-FR 已通过进入 publish / first production 之前所需的 Surface Review 阶段；它不宣告 fr-FR 已完成首次 production，也不包含最终 decision。
