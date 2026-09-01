# 语言、域名、CDN 与共享静态资源

`go-tour-i18n` 采用构建时单语言站点：每个社区语言独立构建和部署，不实现运行时 locale switching。首页语言入口共享一份有序 registry，并链接到各语言的正式站点或官方来源。

新增语言统一从 [新增 Locale 执行手册](docs/NEW_LOCALE_RUNBOOK.md) 开始；不要只在 registry 中增加一行就视为完成。新 locale 必须先明确规范 locale、显示名称、domain/CDN、全站 glossary 决策和 production profile，并通过独立的 TranslationUnit Quality Review 与 Locale Surface Review。

语言 registry 的正式展示顺序按语言的英文名称字母顺序排列，不按加入项目的时间排列。当前顺序为 Chinese（`zh-CN`）→ English（`en`）→ French（`fr-FR`）→ German（`de-DE`）→ Japanese（`ja-JP`）→ Korean（`ko-KR`）。

## 语言站点与 CDN

| Locale | 显示名称 | 正式入口 | CDN | 说明 |
| --- | --- | --- | --- | --- |
| `zh-CN` | 简体中文 | <https://go-dev.shuijingwanwq.com/> | EdgeOne | 当前默认社区语言站；不创建 `zh.go-dev` 或 `zh-cn.go-dev` |
| `en` | English | <https://go.dev/tour/> | Go 官方提供 | 继续使用官方 A Tour of Go；当前不建设本项目的英文社区版本，也不规划 `en-go-dev.shuijingwanwq.com` |
| `fr-FR` | Français | <https://fr-go-dev.shuijingwanwq.com/> | Cloudflare Free | 已正式上线的法语社区语言站；域名 language code 为 `fr` |
| `de-DE` | Deutsch | <https://de-go-dev.shuijingwanwq.com/> | Cloudflare Free | 德语社区语言站；域名 language code 为 `de` |
| `ja-JP` | 日本語 | <https://ja-go-dev.shuijingwanwq.com/> | Cloudflare Free | 已正式上线的日语社区语言站 |
| `ko-KR` | 한국어 | <https://ko-go-dev.shuijingwanwq.com/> | Cloudflare Free | 已冻结生产身份、尚未首次上线的韩语社区语言站；域名 language code 为 `ko` |

## ko-KR production identity

ko-KR 使用规范 locale 与 HTML `lang` `ko-KR`，本地显示名为 `한국어`，英文名为 `Korean`，域名 language code 为 `ko`。production hostname 为 <https://ko-go-dev.shuijingwanwq.com/>，CDN 使用 Cloudflare Free；非中文共享静态资源使用 <https://assets-go-dev.shuijingwanwq.com/>。Playground 后续需要允许的精确 Origin 为 `https://ko-go-dev.shuijingwanwq.com`。

以下值是已冻结 production identity 的可读快照；权威、可执行来源为 `production/identity.json`。当前 `production_state` 使用现有首次上线生命周期值 `first-production`，只表示该 identity 将供未来首次生产编排器使用，不表示生产基础设施或部署已经完成。

| 项目 | 值 |
| --- | --- |
| data root | `/data/go-tour-ko-KR` |
| releases | `/data/go-tour-ko-KR/releases` |
| current | `/data/go-tour-ko-KR/current` |
| deploy lock | `/data/go-tour-ko-KR/.deploy.lock` |
| systemd service | `go-tour-ko-KR.service` |
| service user | `go-tour` |
| loopback port | `4003` |
| health URL | <http://127.0.0.1:4003/> |
| Nginx vhost | `/usr/local/nginx/conf/vhost/ko-go-dev.shuijingwanwq.com.conf` |
| TLS certificate | `/usr/local/nginx/conf/ssl/ko-go-dev.shuijingwanwq.com.crt` |
| TLS private key | `/usr/local/nginx/conf/ssl/ko-go-dev.shuijingwanwq.com.key` |

生产服务器已实际确认 `3999`、`4000`、`4001`、`4002` 正在监听，`4003` 当前未监听，因此正式采用 `4003`。本阶段不创建服务器文件，不修改 systemd、Nginx、Cloudflare 或 Playground，也不执行首次 production 部署。

## fr-FR production identity

fr-FR 使用规范 locale 与 HTML `lang` `fr-FR`，本地显示名为 `Français`，英文名为 `French`，域名 language code 为 `fr`。production hostname 为 <https://fr-go-dev.shuijingwanwq.com/>，CDN 使用 Cloudflare Free；非中文共享静态资源使用 <https://assets-go-dev.shuijingwanwq.com/>。Playground 精确 Origin 为 `https://fr-go-dev.shuijingwanwq.com`。

以下值是当前 production identity 的可读快照；权威、可执行来源为 `production/identity.json`：

| 项目 | 值 |
| --- | --- |
| data root | `/data/go-tour-fr-FR` |
| releases | `/data/go-tour-fr-FR/releases` |
| current | `/data/go-tour-fr-FR/current` |
| deploy lock | `/data/go-tour-fr-FR/.deploy.lock` |
| systemd service | `go-tour-fr-FR.service` |
| service user | `go-tour` |
| loopback port | `4002` |
| health URL | <http://127.0.0.1:4002/> |
| Nginx vhost | `/usr/local/nginx/conf/vhost/fr-go-dev.shuijingwanwq.com.conf` |
| TLS certificate | `/usr/local/nginx/conf/ssl/fr-go-dev.shuijingwanwq.com.crt` |
| TLS private key | `/usr/local/nginx/conf/ssl/fr-go-dev.shuijingwanwq.com.key` |

fr-FR 已于 2026-08-30 完成首次 production、Cloudflare、Playground、machine/browser 与广告轻量验收；最终 evidence 为 `data/locale-surface-reviews/fr-FR/20260830-first-production.md`。

## de-DE production identity

de-DE 使用规范 locale 与 HTML `lang` `de-DE`，本地显示名为 `Deutsch`，英文名为 `German`，域名 language code 为 `de`。production hostname 为 <https://de-go-dev.shuijingwanwq.com/>，CDN 使用 Cloudflare Free；非中文共享静态资源使用 <https://assets-go-dev.shuijingwanwq.com/>。Playground 已允许精确 Origin `https://de-go-dev.shuijingwanwq.com`。

以下值是当前 production identity 的可读快照；权威、可执行来源为 `production/identity.json`：

| 项目 | 值 |
| --- | --- |
| data root | `/data/go-tour-de-DE` |
| releases | `/data/go-tour-de-DE/releases` |
| current | `/data/go-tour-de-DE/current` |
| deploy lock | `/data/go-tour-de-DE/.deploy.lock` |
| systemd service | `go-tour-de-DE.service` |
| service user | `go-tour` |
| loopback port | `4001` |
| health URL | <http://127.0.0.1:4001/> |

de-DE 已完成首次 production；历史最终 evidence 为 `data/locale-surface-reviews/de-DE/20260829-first-production.md`。后续 release 使用日常维护部署流程。

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
