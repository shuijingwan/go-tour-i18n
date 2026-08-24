# 语言、域名、CDN 与共享静态资源规划

`go-tour-i18n` 采用构建时单语言站点：每个社区语言独立构建和部署，不实现运行时 locale switching。首页语言入口共享一份有序 registry，并链接到各语言的正式站点或官方来源。

## 语言站点与 CDN

| Locale | 显示名称 | 正式入口 | CDN | 说明 |
| --- | --- | --- | --- | --- |
| `zh-CN` | 简体中文 | <https://go-dev.shuijingwanwq.com/> | EdgeOne | 当前默认社区语言站；不创建 `zh.go-dev` 或 `zh-cn.go-dev` |
| `en` | English | <https://go.dev/tour/> | Go 官方提供 | 继续使用官方 A Tour of Go；当前不建设本项目的英文社区版本，也不规划 `en.go-dev.shuijingwanwq.com` |
| `ja-JP` | 日本語 | <https://ja.go-dev.shuijingwanwq.com/> | Cloudflare | 日语社区语言站 |

后续所有非中文社区语言统一采用：

```text
https://<language-code>.go-dev.shuijingwanwq.com/
```

例如 `ko.go-dev.shuijingwanwq.com`、`de.go-dev.shuijingwanwq.com` 和 `fr.go-dev.shuijingwanwq.com`，统一使用 Cloudflare。域名标签使用小写；仓库目录和 HTML `lang` 使用项目确定的规范 locale，例如 `zh-CN`、`ja-JP`。

## 共享静态资源规划

未来所有非中文社区语言可通过 Cloudflare 共享真正 locale-neutral 的静态资源：

```text
https://assets.go-dev.shuijingwanwq.com/
```

适用内容包括 CSS、公共 JavaScript、图片、Logo、图标、字体、CodeMirror 等第三方静态库，以及其他真正与语言无关的文件。`ja.go-dev.shuijingwanwq.com`、`ko.go-dev.shuijingwanwq.com`、`de.go-dev.shuijingwanwq.com` 等站点均可使用这一静态资源域名。

当前 `/tour/script.js` 包含 `window.__tourUIMessages`、`window.__tourModules` 等 locale-specific UI bootstrap，因此现阶段不是所有语言可直接共用的 locale-neutral 静态资源。未来如有明确收益，可另行评估拆分 locale bootstrap；本阶段不实施该重构。

zh-CN 继续由 <https://go-dev.shuijingwanwq.com/> 同时提供 HTML 和自己的静态资源，并继续使用 EdgeOne。只有确认中文静态资源拆分有明确实际收益时，才考虑类似 `assets-cn.go-dev.shuijingwanwq.com` 的域名；当前不创建、不实现。

本文件只记录正式规划。本阶段不修改 `/tour/static/`、`/images/`、CSS、JavaScript、图片、字体 URL，不修改构建、发布投影、部署、DNS、Cloudflare 或 EdgeOne 配置，也不创建 `assets.go-dev.shuijingwanwq.com` 的实际部署配置。
