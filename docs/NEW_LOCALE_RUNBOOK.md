# 新增 Locale 执行手册

本文档是新增一门社区语言的唯一高层入口。它把 locale 准备、翻译、语言表层审核、首次生产部署和上线验收串成一条发布路径，但不复制各阶段的正式细则。执行具体阶段时，必须进入本文引用的对应规范。

本文适用于新增 locale；已有 locale 的 TranslationUnit 修订仍从 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 开始，日常生产发布直接使用 [生产运维手册](PRODUCTION_RUNBOOK.md) 的维护部署流程。

## 总体顺序与四条边界

```text
locale / domain / CDN 决策
→ locale glossary
→ locale 配置与公共 UI catalog
→ 首页、导航、语言选择器与 article metadata
→ TranslationUnit 翻译与 automatic validation
→ Candidate Snapshot → Quality Check → Final Review
→ promotion
→ 离线生成并验证课程页 SEO metadata
→ 完整 projection / preview
→ Locale Surface Review
→ production publish
→ 首次 production 基础设施、部署 profile 与广告接入
→ 最终源站、公网和浏览器上线验收
```

必须区分四类工作：

1. **TranslationUnit 质量审核**只审核进入 translation workflow 的 Page 和 eligible Example candidate，规则见 [Translation Quality Review](TRANSLATION_QUALITY_REVIEW.md)。它保持逐 TranslationUnit、Quality Check 全 A、Final Review A-only 的既有 promotion gate。
2. **Locale Surface Review**审核 TranslationUnit 之外及组合后页面上的语言表层，规则见 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md)。它是独立的 locale release gate，不生成 TranslationUnit review evidence，也不允许替代或弱化 A-only gate。
3. **首次生产部署**为新 locale 建立 hostname、CDN、service、port、TLS、vhost、DNS/CDN、Playground Origin、部署 profile，以及对既有 AdSense 能力的 production 接入；它不是一次普通 release 切换。课程页手动广告、Auto Ads、Angular SPA 生命周期和局部布局保护均为共享实现，第三门及后续 locale 不重新开发广告功能。
4. **日常维护部署**只对已完成上述基线的 locale 执行 `scripts/deploy-production.sh <release-dir>`，不重新探测或设计服务器环境。

## 1. 冻结语言与生产身份

在创建翻译资产前记录并确认：

- 规范 locale、`html_lang`、本地显示名、英文名和域名使用的 lowercase language code；
- production hostname 与 CDN；除 zh-CN 既有站外，社区语言遵循 [LANGUAGES.md](../LANGUAGES.md) 的 `<language-code>-go-dev.shuijingwanwq.com` 与 Cloudflare 约定；
- 独立 data root、release/current/lock 路径、systemd service、未占用的 loopback port、public URL；
- 是否使用非中文共享静态资源基线；
- Playground 代理需要新增的精确 HTTPS Origin；
- 首页语言 registry 中的显示顺序与目标 URL。

这些决定应先形成明确记录，再修改 locale 资源或生产环境。不得根据其他 locale 的目录名、端口或译法自动推导新 locale。

## 2. 建立 locale 术语权威来源

先阅读 [术语治理政策](TRANSLATION_TERMINOLOGY.md) 和 [术语制定指南](TERMINOLOGY_GUIDE.md)，再建立 `locales/<locale>/glossary.yaml`。不得机器翻译 zh-CN、ja-JP 或其他 locale 的 glossary。

Glossary 同时承担两项正式职责：

- 它是 TranslationUnit 模型执行时与 manifest、全部 inputs 不可拆分的正式输入；
- 它是该 locale 全站的正式术语权威来源，公共 UI、首页、`/tour/`、`/tour/list`、导航、语言选择器、编辑器、runtime message、article metadata 和 SEO 可见文案均必须遵守。

对 `Go Playground` 这类可能翻译、部分本地化或 keep 的名称，必须在该 locale 的 glossary 中形成显式决定。不同 locale 可以做不同决定，但同一 locale 不得在不同表层混用。此步骤只建立新 locale 的规则，不顺带修改 zh-CN 或 ja-JP 的现有译文。

## 3. 建立非 TranslationUnit 语言资产

以现有 locale 的**结构**为参考，不复制其语言内容：

- `locales/<locale>/locale.json`：locale 身份；
- `internal/tour/ui/<locale>.json`：完整公共 UI catalog，key 与 `plain` / `rich` kind 必须匹配英文 source `internal/tour/ui/en.json`，正式 locale 不使用英文 fallback；
- `locales/<locale>/article-metadata.json`：全部正式 article 的本地化 `title` 与 `subtitle`；
- `locales/<locale>/course-metadata.json`：全部正式课程页的目标语言 SEO description；其 schema、离线生成输入和 stale 规则见 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md)；
- 首页、导航、语言选择器与语言 registry 所需的 locale 条目。

UI catalog、首页和 metadata 不属于 TranslationUnit candidate、status、Quality Check、Final Review 或 promotion。它们必须在后续 Surface Review 中单独验收。

`status.tsv` 不是语言资产，也不得从其他 locale 复制。完成上述 locale 配置后、第一次进入 TranslationUnit retranslation export 前，必须以当前正式 TranslationUnit catalog 初始化该 locale 的统一状态表，并立即校验：

```sh
go run -mod=readonly ./cmd/tour-i18n status init --locale <locale>
go run -mod=readonly ./cmd/tour-i18n status check --locale <locale>
```

`status init` 只负责首次创建缺失的 `locales/<locale>/status.tsv`：按 Catalog Page 顺序、再按 eligible Example inventory 顺序写入当前 workflow 的全部 TranslationUnit，初始状态均为 `pending`。命令不写当前时间、不覆盖已有文件，也不承担已有状态的修复、同步或 source 更新迁移。只有 `status check` 通过后，才能执行首次 retranslation export。

## 4. 执行 TranslationUnit workflow

TranslationUnit 工作从 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 进入。正式翻译前还必须读取 [翻译任务规范](TRANSLATION_TASK_SPEC.md)、[Retranslation 执行手册](RETRANSLATION_RUNBOOK.md)、[Codex 翻译执行规范](CODEX_TRANSLATION.md)、当前 batch manifest、manifest 列出的全部 inputs，以及目标 locale glossary。

保持既有顺序：

```text
export
→ model translation
→ process
→ automatic validation
→ Candidate Snapshot
→ ChatGPT Quality Check（全 A）
→ Final Review（逐 unit A + approved）
→ promotion
→ ready
```

不要把 UI catalog 或 metadata 塞入 TranslationUnit batch；不要用 Surface Review 结论生成 review evidence；不要在 promotion 前用完整站点观感替代逐 TranslationUnit 审核。

## 5. 完整投影、预览与 Surface Review

只有 promotion 完成、全部 workflow TranslationUnit 为 canonical `ready`，并且 locale 配置、UI catalog 和 article metadata 完整后，才构建完整 projection 和 locale preview：

```sh
go run -mod=readonly ./cmd/tour-i18n build --locale <locale>
go run -mod=readonly ./cmd/tour-i18n preview --locale <locale>
```

执行 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 时，先以英文/source、目标资产和 glossary 为正式输入，完整审核 TranslationUnit 之外的 UI catalog、article metadata、首页及其他 locale-level 文案；不得用浏览器抽查替代。语言质量通过后，再在完整 preview 上验收首页、课程入口、列表、导航、语言选择器、编辑器、runtime message、SEO 以及桌面和移动端。

正式审核记录写入 `data/locale-surface-reviews/<locale>/<review-id>.md`。发现 TranslationUnit 内容问题时，回到新的 revision batch 和完整 A-only 审核链；发现表层资产问题时，修正对应 locale 资产并重新执行受影响的 Surface Review。语言质量审核或 preview acceptance 未通过，不得 publish production bundle。

## 6. Publish 与首次生产部署

Surface Review 通过并完成其中所有修复后，使用 `assets-go-dev.shuijingwanwq.com` 的非中文 locale 必须在首次 production release 激活前通过 shared-assets current-state freshness gate：以当前仓库最终状态执行正式 `assets export` 与 `assets validate`，然后必须运行 [生产运维手册](PRODUCTION_RUNBOOK.md#shared-assets-production-发布状态机) 的 `scripts/deploy-shared-assets.sh`，由脚本正式比较当前 export 与 production origin 并生成 verification receipt。脚本输出 `NO CHANGES` 时，origin 已与当前 export 一致，不产生 changed URL 或 Custom Purge，直接运行 receipt 指定的 `scripts/verify-shared-assets-production.sh <receipt>`；脚本输出 changed URLs 时，先按既有状态机只对这些 URL 完成 Cloudflare Dashboard Custom Purge，再运行同一验证命令。验证脚本自动完成 changed URL 缓存验收（如有）、完整 11 个 allowlist 公网 SHA-256 与当前 export 的对照，以及 boundary 404；仅当最终为 11/11 一致且全部验证通过时，本 gate 才通过。不得以历史“11/11 已部署”记录、Git 历史、上次部署时间或人工判断文件是否变化替代此流程；receipt 只是本次 execution evidence，不能替代未来 locale 的 freshness gate。本 gate 只在首次 production 激活前检查当前最终状态，不要求每次 Surface Review 都部署 production。

随后按 [生产运维手册](PRODUCTION_RUNBOOK.md) 生成 Linux/amd64 production bundle，并核对 `release.json`、bundle 内 `site-metadata.json`、文件集合和 `SHA256SUMS`。`publish` 只生成 release，不创建 hostname、service、TLS、vhost 或部署脚本 profile。

随后执行该手册的“新 locale 首次生产部署”：先完成并记录 production profile、基础设施和 AdSense production 接入，再部署已验收 bundle。首次 deployment 的 `current` 可以尚不存在；脚本首次原子创建它后，如新 release 健康检查失败，没有旧 release 可回滚，必须保留现场并人工检查。已有 locale 的日常 deployment 继续要求 `current` 为指向 release root 内既有 release 的合法 symlink，并沿用既有 rollback 流程。首次激活的页面必须已是最终的启用广告形态，不安排“先无广告上线、再接广告”的两次发布。当前 `scripts/deploy-production.sh` 对 locale fail closed；新 locale 未经明确 profile 实现和验证前，不能假定通用命令已经支持它。

## 7. 正式上线验收与移交

首次上线至少完成三层验收：

- **源站层**：service active、loopback 连续健康、关键路由和静态资源正确、`/socket` 与保留路径符合 production 安全边界；
- **公网层**：HTTPS、CDN/DNS、首页、`/tour/`、`/tour/list`、课程页、静态资源、`robots.txt`、sitemap 全量 URL 与 canonical host；
- **真实浏览器层**：桌面与移动端页面、导航、语言选择器、Run / Format / Reset、runtime message，以及 Network 中真实 Playground endpoint 和允许的 Origin；并按生产运维手册对最终课程页做轻量广告确认。

首次 production 部署后只做这一次最终验收；不执行“无广告完整验收 → 开广告 → 再完整验收”的双重流程。最后重新执行 production 上的 rendered surface acceptance 关键项，在同一 Surface Review evidence 中记录 public URL、profile、证书/vhost 路径、当前 release、sitemap、Playground 与轻量广告验收结果，以及最终 `decision = passed | failed`。只有最终通过，该 locale 才从“首次部署”转入 [日常维护部署](PRODUCTION_RUNBOOK.md#已有-locale-日常维护部署)。
