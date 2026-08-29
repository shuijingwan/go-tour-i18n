# de-DE Locale Surface Review — 2026-08-29 首次 production 工作记录

这是首次 production 前的 Surface Review 工作记录，不是最终上线 evidence。production 部署和公网验收完成后，必须在本文件补充真实 production evidence，才可将正式 decision 改为 `passed`。

## 审核身份

- locale：`de-DE`
- review_id：`20260829-first-production`
- reviewed commit：`e1045bff5b29b951034a28423f36339b85e44608` (`e1045bf`)
- upstream baseline：`golang/website` `master` 的 `b3fc6537086f09e88cb3c1ecd09bd47c31c54241`
- upstream commit time：`2026-08-26T21:55:26Z`
- intended production public identity：`https://de-go-dev.shuijingwanwq.com/`
- date：`2026-08-29`

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article：`7`

### Glossary identity

- path：`locales/de-DE/glossary.yaml`
- SHA-256：`e13b933c218596b3cf8b522d892bc7d08aba7d6008719e7d77234c958b4ad6d5`

### UI / metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`98a2ab79bd319da815b6e592b4e3131abbcd18356e771ab2fbbd3db2794a3588`

de-DE UI target：

- path：`internal/tour/ui/de-DE.json`
- SHA-256：`0a8f836c3f3402f873078d9b1c59bb630e1057d13d45fee997de6b475fa6f1bc`

Article metadata：

- path：`locales/de-DE/article-metadata.json`
- SHA-256：`e930a35d7beac1d1185104b011b514aeaabe058cba674633cf7fd5d93bcaa505`

Course metadata：

- path：`locales/de-DE/course-metadata.json`
- SHA-256：`66d4c384855131c3b3c9dca518617931382498de584a4892164f854dec97b811`

## A. Locale-level language quality review

结果：`passed`

本阶段独立于 TranslationUnit Quality Check 和 Final Review。对 TranslationUnit 之外的 de-DE locale-level 语言资产进行完整 source ↔ target 审核，并以 de-DE glossary 为正式术语基线。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/de-DE.json`
- `locales/de-DE/glossary.yaml`

检查全部 message key 的 source ↔ target 语义、`plain` / `rich` kind、占位符与 markup identity，以及 glossary mandatory / preferred / forbidden / keep 决策；覆盖首页、`/tour/`、`/tour/list`、导航与语言选择器、编辑器 Run / Format / Reset、runtime message、footer、共享 shell 和 SEO 可见 UI 文案。

最终公共 UI 统一采用正式 **Sie** 语体。此前 UI 与 article metadata 中的 du / Sie 不一致均已修正；没有遗留语言质量 blocker。

### Article metadata

完整对照当前 7 个正式 article 的英文根级 title / subtitle 与 de-DE article metadata。

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

### Course SEO metadata

按每个 Page 的完整英文 source、最终 canonical de-DE target 和 de-DE glossary 审核正式 course metadata。

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

首轮 Surface Review 发现并修复的 5 个 description，已由当前正式 `locales/de-DE/course-metadata.json` 和提交 `e1045bf` 确认：

- `basics/14`：未类型化数值常量的类型由常量精度决定；
- `flowcontrol/9`：`switch` 分支值既不必为常量，也不必为整数；
- `moretypes/5`：`Name:` 可按无固定顺序指定字段子集，`&` 指向完整 Struct 值；
- `moretypes/15`：`append` 及底层数组容量不足时的更大数组分配；
- `generics/1`：`Index` 在元素类型满足 `comparable` 时于 Slice 中查找值。

没有引用已删除的 `data/course-metadata-generation/` 临时中间产物。最终没有遗留 de-DE locale-level 语言质量 blocker。

## B. Rendered surface acceptance — complete preview

结果：`passed`

### Preview identity

完整 de-DE preview 使用 production-style complete locale preview：

- locale：`de-DE`
- ready：`122`
- pending：`0`
- blocked：`0`
- pages：`103`
- articles：`7`
- public canonical identity：`https://de-go-dev.shuijingwanwq.com/`

### Desktop acceptance

实际浏览器覆盖：

- 首页；
- `/tour/welcome/1`；
- `/tour/list`；
- 普通课程页；
- 编辑器课程页；
- header / footer、navigation、editor、Run / Format / Reset、language selector、theme。

结果：页面 shell、课程内容、编辑器、导航与主题显示正常，无未解决的 desktop rendered blocker。

### Mobile acceptance

实际 Firefox `375 CSS px` 覆盖：

- 首页、`/tour/list`、`/tour/welcome/1`；
- header、footer、menu、editor、pagination；
- light / dark theme；
- Run、Format、Reset。

本轮 preview 发现并完成以下共享 rendered 修复：

1. de-DE Run 按钮 `Ausführen` 因固定 `40px` 宽度裁切；共享修复改为 `min-width`。
2. 长标题 `Eine Tour durch Go` 在 mobile header 被裁切；共享 mobile flex/header 改为自适应布局。
3. `/tour/list` footer 首行被 `.container` 覆盖：mobile `.wrapper { top: 64px }` 覆盖了较早的 `.list-wrapper { top: auto }`。最终在 mobile 规则中恢复 `.list-wrapper { top: auto }`。
4. 课程分页 `< 1/5 >` 原来依赖固定高度、padding 和 line box；改为 flex 垂直居中，并保持约 `44px` 的最小触控目标。
5. mobile dark theme 中，分页与 editor 之间的 course-ad margin / transparent container 透出错误 body 背景。广告 margin 保持 `24px / 12px`，未修改广告 mount、loader、slot 或 SPA 生命周期；最终为正确的 mobile dark editor/container 区域提供背景，Firefox 实测复核通过。

### Browser regressions

新增的回归测试：

- `internal/tour/header_browser_test.go`
- `cmd/tour-i18n/footer_preview_browser_test.go`

Header regression 经 CDP `Emulation.setDeviceMetricsOverride` 强制真实 CSS viewport：`320`、`375`、`414`，并覆盖：

- `A Tour of Go`
- `Eine Tour durch Go`
- `Go 语言之旅`
- `Go のツアー`

测试断言标题不裁切、header 与页面无横向 overflow、课程内容不覆盖 header，以及 `window.innerWidth` 等于所请求的 viewport。

Footer complete-preview regression 使用 `BuildLocaleProjection`、`NewPreviewHandler` 和真实 `/tour/list`，覆盖 `320`、`375` CSS px、完整 footer 文案、container/footer overlap、`elementsFromPoint` stacking 与横向 overflow。没有保留已删除的旧 synthetic footer test。

## Runtime / interaction

在 production-style preview 实际检查：

- Run：`passed`；输出：`Hello, 世界`；runtime message：`Programm beendet.`
- Format：`passed`
- Reset：`passed`

在 mobile 上也分别实际执行一次 Run、Format、Reset，均为 `passed`。

## Language selector

实际顺序：

1. 简体中文
2. English
3. Deutsch
4. 日本語

`Deutsch` 是当前语言文本而非链接。其他实际 URL：

- 简体中文：`https://go-dev.shuijingwanwq.com/`
- English：`https://go.dev/tour/`
- 日本語：`https://ja-go-dev.shuijingwanwq.com/`

结果：`passed`。

## SEO / identity

首页实际 identity：

- `html lang`：`de-DE`
- title：`Yongye · Mehrsprachiges Übersetzungsprojekt zu „Eine Tour durch Go“`
- canonical：`https://de-go-dev.shuijingwanwq.com/`

`/tour/welcome/1` 实际 identity：

- title：`Hallo, 世界 — Willkommen! — Eine Tour durch Go`
- description：`Diese Einführung erklärt Aufbau, Navigation und interaktive Bedienung der Tour sowie das Ausführen, Bearbeiten und Formatieren der Go-Beispiele.`
- canonical：`https://de-go-dev.shuijingwanwq.com/tour/welcome/1`

`robots.txt`：

```text
User-agent: *
Allow: /

Sitemap: https://de-go-dev.shuijingwanwq.com/sitemap.xml
```

Sitemap：

- URL count：`105`
- 首页：`1`
- `/tour/list`：`1`
- Page：`103`
- wrong host count：`0`

结果：`passed`。

## Test evidence

以下针对性测试最终通过：

```sh
GO_TOUR_RUN_BROWSER_TESTS=1 \
go test ./internal/tour \
  -run '^TestTourHeaderTitlesFitCommonMobileViewports$' \
  -count=1

GO_TOUR_RUN_BROWSER_TESTS=1 \
go test ./cmd/tour-i18n \
  -run '^TestDEPreviewListFooterGeometryInBrowser$' \
  -count=1

go test ./cmd/tour-i18n \
  -run 'Test.*[Aa]sset' \
  -count=1

git diff --check
```

仓库已知的 4 个无关 upstream fixture 漂移测试失败不属于本轮 de-DE blocker。

## Production verification

状态：`pending / not yet executed`

尚未执行，因而不得视为 passed：

- production publish bundle deployment；
- de-DE 首次 production infrastructure activation；
- public HTTPS / CDN verification；
- production Playground Origin / Network verification；
- production Run / Format / Reset；
- production sitemap HTTP verification；
- lightweight AdSense confirmation。

未记录 release path、published_at、systemd 状态、TLS、CDN 或广告结果；这些均须由首次 production 后的真实证据补充。

## Reviewer

- TranslationUnit Quality Check / Final Review：ChatGPT GPT-5.6 Sol
- Locale-level language quality review：ChatGPT GPT-5.6 Sol
- Rendered browser acceptance：人工浏览器验收 + ChatGPT 分析

## Issues

当前没有 unresolved language 或 preview blocker。

唯一未完成项为首次 production verification；它尚未执行。

## Decision

`decision = failed`

A language quality review 与 preview acceptance 均为 `passed`，但首次 production verification 仍为 `pending`，因此 overall final release gate 尚未完成。该 `failed` 仅表示缺少首次 production evidence，并非存在未解决的语言或 preview defect。production 完成后将在本文件补充真实 evidence；只有全部通过才可改为 `decision = passed`。
