# 语言、域名与 CDN 规划

`go-tour-i18n` 面向多语言扩展，第一阶段只交付简体中文 `zh-CN`。当前采用构建时单语言生成，不实现运行时语言切换；英文上游基准继续由 `_content/tour` 提供。

## 域名模式

以下均为规划模式，`<实际域名>` 不是已经部署的真实域名：

| 模式 | 语言与用途 |
| --- | --- |
| `go-tour.<实际域名>` | 默认简体中文 `zh-CN` 站 |
| `en.go-tour.<实际域名>` | 英文 `en` 站 |
| `ja.go-tour.<实际域名>` | 未来日语 `ja` 站 |
| `go-tour.<实际域名>/about/` | 项目介绍、语言选择、上游信息和 GitHub 入口 |

不创建 `zh.` 或 `zh-cn.` 子域。域名标签使用小写，而目录和 HTML 使用规范 locale/lang，例如目录 `zh-CN`、`en`、`ja`，HTML 使用 `lang="zh-CN"`。

## CDN 角色

- 默认 zh-CN 主站：EdgeOne
- 英文站：Cloudflare
- 未来日语站：Cloudflare

每种语言未来可以独立选择 CDN、部署区域和发布节奏。当前没有配置真实 DNS、证书、EdgeOne 或 Cloudflare，也没有发布英文、日文或中文版。

第一阶段不创建繁体中文。只有真实流量或用户需求支持时，才考虑未来的 `zh-Hant`，它不是近期交付计划。
