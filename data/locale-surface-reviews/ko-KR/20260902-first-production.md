# ko-KR Locale Surface Review — 2026-09-02 首次 production 工作记录

这是 ko-KR 首次 production 的进行中 Locale Surface Review 工作记录，不是最终 production evidence。Locale-level language quality review、complete preview rendered surface acceptance 与 visual HUMAN gate 已完成；production verification 尚未执行。

## 审核身份

- locale：`ko-KR`
- review_id：`20260902-first-production`
- reviewed commit：`2fc91938871fb079f7c9f5bfba0af59847566357` (`2fc9193`)
- Stage A language-assets baseline commit：`f6dbe6772997291a3e37c062c083f936eb3c3338` (`f6dbe67`)
- production identity state：`first-production`
- configured production public identity：`https://ko-go-dev.shuijingwanwq.com/`
- date：`2026-09-02`

从 Stage A 基线至 reviewed commit，以下 Stage A 语言资产的 Git diff 为零；本次重新计算的 SHA-256 亦全部与 Stage A evidence 一致。

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article：`7`

`go run -mod=readonly ./cmd/tour-i18n status check --locale ko-KR` 已确认：`122` TranslationUnit（`103` pages，`19` examples）。

### Glossary identity

- path：`locales/ko-KR/glossary.yaml`
- SHA-256：`dcc0ad47b83965c6347daf9da6ecc9e7517136640d4ba9d1972743def7c97f39`

### UI / metadata identity

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`52cab0eaf41fd205d7096665dcb7aff76e1538d8036dc49a09f5cfc89e61c35b`

ko-KR UI target：

- path：`internal/tour/ui/ko-KR.json`
- SHA-256：`987f4362508d2ea5df69e3d183af59e5d247e3ce324cf884eb4ce35757cc6695`

Article metadata：

- path：`locales/ko-KR/article-metadata.json`
- SHA-256：`d3a2d54d3f23383b550e4b17e97c205a19750021154139e92f7ea6a6571552f8`

Course metadata：

- path：`locales/ko-KR/course-metadata.json`
- SHA-256：`5e69465edb29ff5ab55d58ee5e4803cc5bc39e453ece6501b6b3561f33abc782`

## A. Locale-level language quality review

结果：`passed`

本阶段已独立完成，未重新执行 TranslationUnit Quality Check 或 Final Review，也未重新审核已通过的 ko-KR Stage A TranslationUnit 语言内容。

- 公共 UI catalog：完整审核 `90/90`，对照英文 source、ko-KR target 与 glossary；
- article metadata：完整审核 `7/7`；
- course metadata：完整审核 `103/103`；
- 无遗留语言质量 blocker。

## B. Rendered surface acceptance — complete preview

结果：`passed`

### Preview identity

完整 production-style ko-KR preview 使用：

```sh
go run -mod=readonly ./cmd/tour-i18n preview \
  --locale ko-KR \
  --http 127.0.0.1:0
```

- preview loopback URL：`http://127.0.0.1:43395/`
- locale：`ko-KR`
- public canonical identity：`https://ko-go-dev.shuijingwanwq.com/`
- reviewed commit：`2fc91938871fb079f7c9f5bfba0af59847566357`

### Automated preview acceptance

正式入口：

```sh
scripts/verify-preview-browser.py http://127.0.0.1:43395/ ko-KR
```

结果：`PREVIEW SURFACE ACCEPTANCE: PASS`

机器 gate 已通过并覆盖 preview identity、HTTP / SEO routes、robots、sitemap、production canonical identity、language selector、desktop/mobile rendered geometry、Run / Format / Reset、same-origin Playground `fmt` / `compile`、SPA 与 `/socket` 边界等正式定义范围。

### visual HUMAN gate

结果：`passed`

该极小人工 gate 只确认整体桌面与移动排版观感正常；未重复机器 gate 已覆盖的 canonical、sitemap、language selector URL、Run / Format / Reset、SPA、`/socket` 或 desktop/mobile geometry 检查。

## Production verification

结果：**尚未执行**。

尚无 production release path、bundle、deployment、公网 URL 验收或 production browser acceptance 记录。本工作记录不声明 DNS、TLS、CDN、Playground Origin、AdSense 或任何 production-only gate 已通过；这些项目须在首次 production 部署后按正式生产运维流程实际验收并回填同一 evidence。

## Reviewer

- 工作记录整理：Codex GPT-5.6
- 本记录保留已完成的 Stage A、automated preview acceptance 与 visual HUMAN gate 的结果；未将其扩展为 production verification。

## Issues

unresolved language blocker：`none`

unresolved preview rendered blocker：`none`

unresolved production blocker：尚未判定（production verification 尚未执行）。

## Current gate status

- A language quality review：`passed`
- complete preview rendered acceptance：`passed`
- visual HUMAN gate：`passed`
- production verification：尚未执行
- final decision：未记录

依据 `docs/LOCALE_SURFACE_REVIEW.md` 的 Evidence 与最终结论规定，首次上线可在 A 阶段和 preview acceptance 通过后维护同一路径的工作记录；production 复核完成后，才在该 evidence 写入真实 production 结果和最终 `decision = passed | failed`。因此本工作记录不使用非正式的 decision 值，也不提前写入 final decision。
