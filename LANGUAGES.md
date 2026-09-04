# 语言、域名、CDN 与共享静态资源

`go-tour-i18n` 采用构建时单语言站点：每个社区语言独立构建和部署，不实现运行时 locale switching。首页语言入口共享一份有序 registry，并链接到各语言的正式站点或官方来源。

新增语言统一从 [新增 Locale 执行手册](docs/NEW_LOCALE_RUNBOOK.md) 开始；不要只在 registry 中增加一行就视为完成。新 locale 必须先明确规范 locale、显示名称、domain/CDN、全站 glossary 决策和 production profile，并通过独立的 TranslationUnit Quality Review 与 Locale Surface Review。

语言 registry 的正式展示顺序按语言的英文名称字母顺序排列，不按加入项目的时间排列。当前顺序为 Chinese（`zh-CN`）→ English（`en`）→ French（`fr-FR`）→ German（`de-DE`）→ Japanese（`ja-JP`）→ Korean（`ko-KR`）→ Spanish（`es-ES`）。

## 语言站点与 CDN

| Locale | 显示名称 | 正式入口 | CDN | 说明 |
| --- | --- | --- | --- | --- |
| `zh-CN` | 简体中文 | <https://go-dev.shuijingwanwq.com/> | EdgeOne | 当前默认社区语言站；不创建 `zh.go-dev` 或 `zh-cn.go-dev` |
| `en` | English | <https://go.dev/tour/> | Go 官方提供 | 继续使用官方 A Tour of Go；当前不建设本项目的英文社区版本，也不规划 `en-go-dev.shuijingwanwq.com` |
| `fr-FR` | Français | <https://fr-go-dev.shuijingwanwq.com/> | Cloudflare Free | 法语社区语言站；域名 language code 为 `fr` |
| `de-DE` | Deutsch | <https://de-go-dev.shuijingwanwq.com/> | Cloudflare Free | 德语社区语言站；域名 language code 为 `de` |
| `ja-JP` | 日本語 | <https://ja-go-dev.shuijingwanwq.com/> | Cloudflare Free | 日语社区语言站 |
| `ko-KR` | 한국어 | <https://ko-go-dev.shuijingwanwq.com/> | Cloudflare Free | 韩语社区语言站；域名 language code 为 `ko` |
| `es-ES` | Español | <https://es-go-dev.shuijingwanwq.com/> | Cloudflare Free | 西班牙语社区语言站；域名 language code 为 `es` |

## Production identity 与历史 evidence

`production/identity.json` 是 hostname、CDN、service、路径、port 与 lifecycle 的唯一可执行 authority；查看当前 profile 使用 `python3 scripts/production-identity.py list`，仅查看 live profile 使用 `python3 scripts/production-identity.py list --state live`。本表不复制 lifecycle 或 server identity。

locale-specific 首次 production 历史 evidence 仍保留在：`data/locale-surface-reviews/de-DE/20260829-first-production.md`、`data/locale-surface-reviews/fr-FR/20260830-first-production.md`、`data/locale-surface-reviews/ko-KR/20260902-first-production.md` 与 `data/locale-surface-reviews/es-ES/20260904-first-production.md`。

后续所有非中文社区语言统一采用：

```text
https://<language-code>-go-dev.shuijingwanwq.com/
```

例如 `ko-go-dev.shuijingwanwq.com`、`de-go-dev.shuijingwanwq.com` 和 `fr-go-dev.shuijingwanwq.com`，统一使用 Cloudflare。域名标签使用小写；仓库目录和 HTML `lang` 使用项目确定的规范 locale，例如 `zh-CN`、`ja-JP`。

## 非中文共享静态资源第一版

所有非中文社区语言的 production 页面通过 Cloudflare 共享第一版明确列入 allowlist 的 locale-neutral 静态资源：

```text
https://assets-go-dev.shuijingwanwq.com/
```

当前代码 allowlist 包括 `app.css`、课程页 `course-ad.css` / `course-ad.js`、CodeMirror CSS、站点 Logo、32/512 PNG favicon、Go Logo、三个 theme icon 和 `app.css` 使用的 `gopher.png`，共 11 个文件。URL 保留原逻辑路径，例如 `https://assets-go-dev.shuijingwanwq.com/tour/static/css/app.css`。development/preview 始终保留完整本地副本；课程广告 CSS/JS 由所有 locale 使用固定 assets-go-dev URL，其余资源仍按 build-selected site 的既有策略解析。

第一版明确不拆分或共享 `/tour/script.js`：它继续包含 locale bootstrap、runtime 配置和现有上游 JavaScript 拼接链，并由每个 language origin 提供。Angular partial、`/tour/lesson/*`、`/tour/footer.html`、课程中的 `tree.png`、HTML、locale article/example、metadata、analytics 和 Playground endpoint 也不共享。课程广告的自有 CSS/JS 是 locale-neutral shared assets；Google 官方 `adsbygoogle.js` 仍直接从 Google 加载，不由 assets origin 代理或自托管。Inconsolata 继续使用现有 Google Fonts 外部资源。

第一版使用固定 URL，不使用 assets-release-id、content-hash URL、asset manifest version mapping 或独立 versioned assets release。共享资源以普通服务器静态目录作为 origin，经 Cloudflare 代理提供；不引入 R2、S3、Workers 或 Pages。language projection 和 production bundle 继续携带完整 `_content`，不会因为非中文 HTML 使用共享资源而裁剪本地副本。

`assets-go-dev.shuijingwanwq.com` 已正式部署并由 Cloudflare 代理。Cloudflare Edge Cache TTL 为 1 个月；Browser Cache TTL 不由项目主动覆盖，使用 Cloudflare/origin 默认或 Respect Existing Headers。formal allowlist 为 11 个文件，production origin 已部署当前 11 文件，公网 SHA-256 为 11/11 PASS。本次 changed URLs 为 `/SHA256SUMS` 与 `/tour/static/css/app.css`，Cloudflare Custom Purge 后均为 `MISS → HIT`；`/tour/script.js`、`/tour/static/img/tree.png` 与 `/tour/static/partials/editor.html` 均为 404。固定 URL 更新继续使用专用 `scripts/deploy-shared-assets.sh` 完整更新 origin tree，再由维护者在 Cloudflare Dashboard 按脚本输出 URL 执行 Custom Purge，最后完成 MISS → HIT → SHA-256。详细安全边界和 human gate 见 [生产运维手册](docs/PRODUCTION_RUNBOOK.md)。旧 `assets.go-dev.shuijingwanwq.com` 已废弃并清理，不提供兼容或迁移。

zh-CN 继续由 <https://go-dev.shuijingwanwq.com/> 同时提供 HTML 和自己的静态资源，并继续使用 EdgeOne。只有确认中文静态资源拆分有明确实际收益时，才考虑类似 `assets-cn.go-dev.shuijingwanwq.com` 的域名；当前不创建、不实现。

zh-CN 当前模式和 EdgeOne 配置保持不变；共享资源第一版也不以未来迁移 zh-CN 为实现前提。
