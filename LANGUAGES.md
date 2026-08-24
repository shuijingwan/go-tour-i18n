# 语言、域名、CDN 与共享静态资源

`go-tour-i18n` 采用构建时单语言站点：每个社区语言独立构建和部署，不实现运行时 locale switching。首页语言入口共享一份有序 registry，并链接到各语言的正式站点或官方来源。

## 语言站点与 CDN

| Locale | 显示名称 | 正式入口 | CDN | 说明 |
| --- | --- | --- | --- | --- |
| `zh-CN` | 简体中文 | <https://go-dev.shuijingwanwq.com/> | EdgeOne | 当前默认社区语言站；不创建 `zh.go-dev` 或 `zh-cn.go-dev` |
| `en` | English | <https://go.dev/tour/> | Go 官方提供 | 继续使用官方 A Tour of Go；当前不建设本项目的英文社区版本，也不规划 `en-go-dev.shuijingwanwq.com` |
| `ja-JP` | 日本語 | <https://ja-go-dev.shuijingwanwq.com/> | Cloudflare | 日语社区语言站 |

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

第一版只包括 `app.css`、CodeMirror CSS、站点 Logo、32/512 PNG favicon、Go Logo、三个 theme icon 和 `app.css` 使用的 `gopher.png`。URL 保留原逻辑路径，例如 `https://assets-go-dev.shuijingwanwq.com/tour/static/css/app.css`。development/preview 始终使用本地同源资源；zh-CN development 与 production 也始终使用本域资源。非中文 production 才解析到共享域名。

第一版明确不拆分或共享 `/tour/script.js`：它继续包含 locale bootstrap、runtime 配置和现有上游 JavaScript 拼接链，并由每个 language origin 提供。Angular partial、`/tour/lesson/*`、`/tour/footer.html`、课程中的 `tree.png`、HTML、locale article/example、metadata、analytics、ads 和 Playground endpoint 也不共享。Inconsolata 继续由 Google Fonts 外部提供，不在共享资源中自托管。

第一版使用固定 URL，不使用 assets-release-id、content-hash URL、asset manifest version mapping 或独立 versioned assets release。共享资源以普通服务器静态目录作为 origin，经 Cloudflare 代理提供；不引入 R2、S3、Workers 或 Pages。language projection 和 production bundle 继续携带完整 `_content`，不会因为非中文 HTML 使用共享资源而裁剪本地副本。

`assets-go-dev.shuijingwanwq.com` 已正式部署并由 Cloudflare 代理。Cloudflare Edge Cache TTL 为 1 个月；Browser Cache TTL 不由项目主动覆盖，使用 Cloudflare/origin 默认或 Respect Existing Headers。公网 9/9 allowlist 文件 SHA-256 已验证，且 Cloudflare 缓存已验证 MISS → HIT。旧 `assets.go-dev.shuijingwanwq.com` 未上线、已废弃，不提供兼容或迁移。固定 URL 下每次更新必须按“更新 origin → Cloudflare purge → 首次请求确认 MISS → 再次请求确认 HIT → 核对新资源内容”执行。

zh-CN 继续由 <https://go-dev.shuijingwanwq.com/> 同时提供 HTML 和自己的静态资源，并继续使用 EdgeOne。只有确认中文静态资源拆分有明确实际收益时，才考虑类似 `assets-cn.go-dev.shuijingwanwq.com` 的域名；当前不创建、不实现。

zh-CN 当前模式和 EdgeOne 配置保持不变；共享资源第一版也不以未来迁移 zh-CN 为实现前提。
