# de-DE Locale Surface Review — 2026-08-29 首次 production 最终 evidence

这是 de-DE 首次 production 的最终 Surface Review evidence。Locale-level language quality review、complete preview rendered acceptance、production machine acceptance、production browser acceptance 与 lightweight ads confirmation 均已完成并通过。

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

仓库已知的 4 个无关 stale upstream commit/time fixture expectation failures 不属于本轮 de-DE blocker：fixture 仍期待旧 baseline `645042eb`，而当前正式 FrozenUpstreamCommit 为 `b3fc6537086f09e88cb3c1ecd09bd47c31c54241`、commit time 为 `2026-08-26T21:55:26Z`。它们与 release / shared-assets SHA-256、cross-locale language selector 或本次 zh-CN / ja-JP 同步无关。

## Production verification

结果：`passed`

### Production identity

- locale：`de-DE`
- public origin：<https://de-go-dev.shuijingwanwq.com/>
- production release：`/data/go-tour-de-DE/releases/20260829-de-DE-8937fdc`
- local bundle：`/tmp/go-tour-release-20260829-de-DE-8937fdc`
- service：`go-tour-de-DE.service`
- loopback：`http://127.0.0.1:4001/`
- data root：`/data/go-tour-de-DE`
- CDN：Cloudflare Free
- shared assets：<https://assets-go-dev.shuijingwanwq.com/>
- Playground：`https://play.go-dev.shuijingwanwq.com:8443`
- allowed Origin：`https://de-go-dev.shuijingwanwq.com`

### First production deployment

- local release preflight：`passed`
- remote permissions / SHA-256 verification：`passed`
- `current` 已切换至 `/data/go-tour-de-DE/releases/20260829-de-DE-8937fdc`
- service health 最终连续 3 次：`active + HTTP 200`
- source deployment：`succeeded`
- public homepage：`HTTP 200`

首次 startup probe 的 `HTTP 000` 属于 service restart startup race；之后连续 3 次均为 HTTP 200，不构成 blocker。Cloudflare 对 `de-go-dev.shuijingwanwq.com` 执行 hostname purge，未使用 Purge Everything。

### Production machine acceptance

实际执行：

```sh
scripts/verify-production.sh \
  /tmp/go-tour-release-20260829-de-DE-8937fdc
```

- release identity：`PASS`
- remote identity：`PASS`
- source routes：`7/7 PASS`
- public routes：`7/7 PASS`
- HTML identity：`PASS`
- sitemap：`105/105`；host mismatch `0`；HTTP failure `0`；`PASS`
- socket boundary：`PASS`
- CDN `/`：`HIT -> HIT -> HIT PASS`
- CDN `/tour/welcome/1`：`HIT -> HIT -> HIT PASS`
- final：`PRODUCTION MACHINE ACCEPTANCE: PASS`

CDN gate 是 cache status observation，不强制 `MISS -> HIT`；三次 `HIT` 表示 cache 已 warm，属于正常 `PASS`。

### Production browser / Playground acceptance

实际 production browser acceptance 已覆盖首页、`/tour/`、`/tour/list`、desktop、mobile、navigation、language selector、Run、Format、Reset 及 SPA Weiter 下一页，结果均为 `passed`。runtime message 为 `Programm beendet.`。Network 确认 Playground endpoint 为 `https://play.go-dev.shuijingwanwq.com:8443`，Origin 为 `https://de-go-dev.shuijingwanwq.com`，未错误使用 `/socket`。

### Lightweight ads confirmation

共享广告实现未变；首次 de-DE production 按正式 lightweight 范围确认：production HTML / Auto Ads loader/config 符合预期、manual `course-ad` mount 存在、浏览器存在真实广告请求机会、course height / footer 无明显异常、SPA Weiter 下一页正常。filled 或 unfilled 均允许，不以 filled ad 为 gate。结果：`passed`。

### Supplemental / non-blocking cross-locale confirmation

本次 zh-CN 与 ja-JP 的额外同步仅作为 supplemental confirmation，不是 de-DE 首次 production gate，de-DE 的 `decision = passed` 不依赖它；未来新增 locale 也不要求旧 locale 在上线当天即时同步语言列表。

- zh-CN release：`/data/go-tour/releases/20260829-zh-CN-9c06f0f`；machine acceptance：`PASS`；sitemap `105/105`、host mismatch `0`、HTTP failure `0`；CDN `/` 为 `MISS -> MISS -> HIT PASS`，`/tour/welcome/1` 为 `MISS -> HIT -> HIT PASS`。
- ja-JP release：`/data/go-tour-ja-JP/releases/20260829-ja-JP-9c06f0f`；machine acceptance：`PASS`；sitemap `105/105`、host mismatch `0`、HTTP failure `0`；CDN `/` 为 `MISS -> HIT -> HIT PASS`，`/tour/welcome/1` 为 `MISS -> HIT -> HIT PASS`。
- 两个 existing locale 的真实浏览器均显示 `Deutsch`，并正确链接至 <https://de-go-dev.shuijingwanwq.com/>。

此确认不改变正式边界：existing locale language list 采用 eventual consistency，旧 locale 会在其下一次正常 upstream sync、UI 更新或日常 release 中自然获得最新 build-time registry。

## Reviewer

- TranslationUnit Quality Check / Final Review：ChatGPT GPT-5.6 Sol
- Locale-level language quality review：ChatGPT GPT-5.6 Sol
- Rendered browser acceptance：人工浏览器验收 + ChatGPT 分析

## Issues

unresolved language blocker：`none`

unresolved rendered blocker：`none`

unresolved production blocker：`none`

## Decision

`decision = passed`

- A language quality review：`passed`
- preview rendered acceptance：`passed`
- production machine acceptance：`passed`
- production browser acceptance：`passed`
- lightweight ads confirmation：`passed`

de-DE 已完成首次 production gate，进入 existing locale daily maintenance 状态。
