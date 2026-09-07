# nl-NL Locale Surface Review — 2026-09-07 首次 production Stage A 记录

这是 nl-NL 首次 production 的 Locale Surface Review 持续工作记录。本次仅记录已完成的 A（Locale-level language quality review）及其 current A gate；preview rendered acceptance、visual HUMAN gate、publish 和 production verification 均尚未执行。

## 审核身份与当前 working-tree / source identity

- locale：`nl-NL`
- review_id：`20260907-first-production`
- Stage A reviewed HEAD commit：`61d15a809b26aa1e9257aa42606ba945bfd2a469` (`61d15a8`)
- Stage A reviewed content basis：当前 working tree；本记录与 A gate 均绑定下列实际文件字节，不能仅以 HEAD 代替。
- Stage A reviewed working-tree status：`M internal/tour/ui/nl-NL.json`、`D locales/nl-NL/.locale-init-incomplete`、`D locales/nl-NL/course-metadata.todo.json`、`?? locales/nl-NL/course-metadata.json`
- date：`2026-09-07`
- configured production public identity：<https://nl-go-dev.shuijingwanwq.com/>

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article/lesson entries：`7`

`go run -mod=readonly ./cmd/tour-i18n status check --locale nl-NL` 返回 `status OK: 122 translation units for nl-NL (103 pages, 19 examples)`。

TranslationUnit 已完成 automatic validation、Quality Check、machine finalization 和 promotion；本记录不替代其既有审核证据。

### 当前 locale-level input identity

- current catalog / source lock：`data/tour-pages.tsv`，SHA-256 `50b6244ef1d8115b2332cd5fa20640ce12fef8db931a1e313b410ade586777b6`，当前 `103` 个 Page 的正式顺序与 source identity。
- locale config：`locales/nl-NL/locale.json`，SHA-256 `fb0d5dff1eba6a62a6aa887c2416fc8ef7aa8c351535b3f5e8e0fcf3036ee0e4`。
- glossary：`locales/nl-NL/glossary.yaml`，SHA-256 `d68f8e2c79b5c109767764b06b05f48bccf639e3d538c3cad54efaae2397b8f8`。
- English UI source：`internal/tour/ui/en.json`，SHA-256 `3a878119cf0d3414fcf6f4ab20abec6459727ad86060f50fd10b0391f46f2964`。
- nl-NL UI target：`internal/tour/ui/nl-NL.json`，SHA-256 `37d479a9d6eb42140a67b41af21c2d2341e1989ec21d59c07454963cf4888215`。
- article metadata：`locales/nl-NL/article-metadata.json`，SHA-256 `f9e72ec9afceec00d7fc45cf55baef3d41729e1eee4ab8d183947b0e61d17b75`。
- course metadata：`locales/nl-NL/course-metadata.json`，SHA-256 `ddbdeeb888f256f4f60d64dba72debf095a5c992ec315f96284cad515f60cf2a`。
- build-time language registry：`internal/tour/languages.go`，SHA-256 `7c7954e45b874c8e499f23c418295aa57415ae5474a8a162a99c733bb723b9e7`。
- stable project configuration：`internal/tour/project.go`，SHA-256 `47bba2660c2fea09af377a092498aedd868dca603f9a5f14ca1ffff7500a9d0b`。
- SEO origin behavior：`internal/tour/seo.go`，SHA-256 `dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`。

### Schema v2 production public identity

Surface Review A schema v2 只绑定解析后的目标 locale public identity projection：

- locale：`nl-NL`
- production_hostname：`nl-go-dev.shuijingwanwq.com`
- production_public_url：<https://nl-go-dev.shuijingwanwq.com/>

不将 production state、端口、service、路径、TLS、CDN 或 credential / secret 写入 A gate identity。

## A. Locale-level language quality review

结果：`passed`

ChatGPT GPT-5.6 Sol 对 TranslationUnit 之外的 nl-NL locale-level 语言资产完成完整 source ↔ target 审核，并以 `locales/nl-NL/glossary.yaml` 为正式术语基线。

### 公共 UI catalog

完整对照：

- `internal/tour/ui/en.json`
- `internal/tour/ui/nl-NL.json`
- `locales/nl-NL/glossary.yaml`

- reviewed：`92/92`
- passed：`92/92`
- failed：`0`

第一轮发现 `1` 个 UI language-quality issue：`site.official_version`。修订后，ChatGPT GPT-5.6 Sol 已对当前完整 UI catalog 进行第二轮独立复核并通过。

### Article metadata

完整对照当前 `7` 个正式 article 的英文根级 `title` / `subtitle` 与 nl-NL article metadata。

- reviewed：`7/7`
- passed：`7/7`
- failed：`0`

第一轮与第二轮均未发现 article metadata issue；本轮未修改 article metadata。

### Course SEO metadata

按每个 Page 的完整英文 source、最终 canonical nl-NL target 和 nl-NL glossary 审核正式 course metadata。

- reviewed：`103/103`
- passed：`103/103`
- failed：`0`

第一轮发现 `4` 个 Course Description language-quality issues：`welcome/1`、`welcome/2`、`moretypes/4` 与 `moretypes/21`。四项修订后，当前完整 `103/103` metadata 已由 ChatGPT GPT-5.6 Sol 第二轮独立复核通过。

### A 阶段结论

- 第一轮 issues：`5`（UI `1`，Course Description `4`）
- 修订后第二轮独立复核：`passed`
- unresolved language blocker：`none`
- Stage A result：`passed`

## B. Rendered surface acceptance

结果：`passed`

### Preview identity

- locale：`nl-NL`
- reviewed commit：`f9958e6e54618c8d42a615ba6ebb4b7b3e0de3ae` (`f9958e6`)
- preview loopback URL：<http://127.0.0.1:44231/>
- projection：`ready=122 pending=0 blocked=0 pages=103 articles=7`

### Automated preview rendered acceptance

实际执行：

```text
[preview-browser] preview identity: PASS
[preview-browser] SEO/routes: PASS
[preview-browser] desktop rendered surface: PASS
[preview-browser] editor Run / Format / Reset: PASS
[preview-browser] SPA: PASS
[preview-browser] mobile /tour/moretypes/1: PASS
PREVIEW SURFACE ACCEPTANCE: PASS
```

正式自动化验收覆盖 preview identity、HTTP / SEO routes、canonical、robots、sitemap、language selector、desktop rendered surface、编辑器 Run / Format / Reset、SPA navigation、same-origin `/_/fmt`、same-origin `/_/compile`、`/socket` HTTP `404` boundary 与 mobile rendered surface。

- preview automated rendered acceptance：`passed`

### Visual HUMAN gate

维护者已实际确认以下整体视觉观感：

- desktop `/`：正常
- desktop `/tour/list`：正常
- desktop `/tour/welcome/1`：正常
- mobile `/tour/moretypes/1`：正常
- 未发现明显视觉异常

Visual HUMAN gate 仅记录整体排版与视觉观感，不重复声称检查 automated acceptance 已覆盖的 canonical、sitemap、Run / Format / Reset、SPA、language selector URL、`/socket` 或 overflow。

- preview visual HUMAN gate：`passed`
- unresolved preview rendered blocker：`none`

## Production verification

- publish：`not executed`
- production deployment / machine acceptance / browser acceptance：`not executed`
- production visual HUMAN gate：`not executed`
- ads / production acceptance：`not executed`
- unresolved production blocker：`not assessed`

## Reviewer、issues 与决策

- Locale Surface Review A：ChatGPT GPT-5.6 Sol
- course metadata generation / regeneration：GPT-5.6 Sol High
- evidence 文件整理：Codex (GPT-5)
- unresolved language blocker：`none`
- Stage A decision：`passed`
- overall first-production final decision：`PENDING`

<!-- first-production-finalization:start -->
- production receipt identity: `PENDING`
- production machine acceptance: `PENDING`
- production browser acceptance: `PENDING`
- production visual HUMAN gate: `PENDING`
- unresolved production blocker: `PENDING`
- overall final decision: `PENDING`
- decision: `pending`
<!-- first-production-finalization:end -->
