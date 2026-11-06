# Deferred Issues

本文件用于记录在开发、测试、部署或 production 验收过程中已经真实发现，但经明确决定暂缓处理的问题，确保问题不会因为超出当前任务 scope 而只留在聊天记录中。

## 记录边界

只有同时满足以下条件的问题才应登记：

- 问题已经在实际工作中被发现，有可描述的现象、失败结果或其他可靠 evidence；
- 用户已明确要求当前任务暂不处理，或明确确认延后处理。

普通 TODO、尚未验证的怀疑、未来优化建议和架构设想不属于 deferred issue。不要主动把所有顺手发现的改进机会写入本文件，也不要凭猜测补录历史问题；历史问题只有存在可靠 evidence 时才可登记。

本机制只负责避免已知问题遗忘，不改变 TranslationUnit、Quality Check、machine finalization、Locale Surface Review、production 或广告流程，也不替代这些流程原有的状态、gate 和 evidence。

## 维护规则

- ID 使用 `DI-YYYYMMDD-NNN`，日期取首次发现日期，序号为当日从 `001` 开始的未占用编号。ID 创建后保持不变。
- 新发现且明确暂缓的问题，先检查是否已有同一问题：没有则新增，有则补充本次发现阶段、上下文和 evidence，避免重复登记。
- `当前状态` 只能是 `open`、`resolved`、`accepted` 或 `obsolete`：
  - `open`：问题仍存在，等待后续处理或决定；
  - `resolved`：问题已经修复，并有验证或核销 evidence；
  - `accepted`：问题仍存在，但已明确接受其影响，不再计划修复；
  - `obsolete`：由于相关功能、环境或前提已消失，问题不再适用。
- 将条目从 `open` 改为其他状态时，不删除原始问题和暂缓原因；必须在“后续处理/核销证据”中记录决定或处理结果、日期，以及可复核的 commit、命令结果、日志、URL、文件路径或其他 evidence。
- evidence 尚不足以证明已修复、已接受或已失效时，状态保持 `open`。

## 条目模板

复制以下模板到“问题登记”末尾；同一条目有新上下文时直接更新原条目。

```markdown
### DI-YYYYMMDD-NNN：简短标题

- ID：`DI-YYYYMMDD-NNN`
- 发现日期：`YYYY-MM-DD`
- 发现阶段/场景：
- 问题描述：
- 暂缓原因：
- 当前状态：`open`
- 后续处理/核销证据：暂无；保持 open。
```

## 问题登记

### DI-20260901-001：术语政策对 glossary keep 实现状态的描述已过期

- ID：`DI-20260901-001`
- 发现日期：`2026-09-01`
- 发现阶段/场景：制定 ko-KR 正式 glossary 并核对 `docs/TRANSLATION_TERMINOLOGY.md`、`internal/i18n/glossary.go` 与 glossary tests。
- 问题描述：`docs/TRANSLATION_TERMINOLOGY.md` 第 9.1 节仍称 loader 不解析 YAML `keep`、实际 keep 仅来自硬编码保护；当前 `LoadGlossary` 已解析并校验 `keep`，`PromptRules` 也会把它作为“保持原样”规则注入模型 prompt，现有测试对此有覆盖。政策文档与真实实现不一致。
- 暂缓原因：本轮明确只制定 ko-KR glossary，不调整跨 locale 术语政策或实现状态文档。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-02 已更新 `docs/TRANSLATION_TERMINOLOGY.md` 第 2.3、9.1、9.2 节，使 policy、loader、prompt、protector 与 Example validator 的当前边界一致；由 `go test ./...` 与 `git diff --check` 验证。

### DI-20260909-001：共享 header 的两处小型垂直对齐观感问题

- ID：`DI-20260909-001`
- 发现日期：`2026-09-09`
- 发现阶段/场景：tr-TR 首次 production 前的 preview visual HUMAN gate；问题在较长的 Turkish header 文案下更明显。
- 问题描述：维护者实际观察到共享 header 中有两处小型垂直对齐观感问题；未造成遮挡、溢出或交互失效，preview automated acceptance 与 visual HUMAN gate 均已通过。
- 暂缓原因：维护者明确决定不阻塞 tr-TR 上线，不在本轮对 Turkish 做局部 CSS 特例；后续作为共享样式问题统一修复。
- 当前状态：`resolved`
- 后续处理/核销证据：原始 evidence 见 `data/locale-surface-reviews/tr-TR/20260909-first-production.md` 的 Visual HUMAN gate 记录。2026-09-10 commit `d018ee6babb56fc28b985802da104dac898b764b`（`fix: 修复多语言页面视觉对齐与输出换行`）将共享 top-bar/left/right 改为 flex + `align-items: center`，并在 `internal/tour/header_browser_test.go` 增加了与原问题精确对应的 `TestTourHeaderTitlesAreCenteredOnDesktopAndFitCommonMobileViewports`（包含 Go Turu）和 `TestHomepageTitlesAreGeometricallyCenteredOnDesktop`（包含 Go Turu Çok Dilli Çeviri Projesi）几何居中回归。2026-09-17，维护者在当前工作区重新执行 `GO_TOUR_RUN_BROWSER_TESTS=1 go test ./internal/tour -run '^(TestTourHeaderTitlesAreCenteredOnDesktopAndFitCommonMobileViewports|TestHomepageTitlesAreGeometricallyCenteredOnDesktop)$' -count=1`，结果 PASS；据此核销为 resolved。

### DI-20260916-001：Go upstream 的四个社区 Tour 链接未使用 canonical `/tour/`

- ID：`DI-20260916-001`
- 发现日期：`2026-09-16`
- 发现阶段/场景：共享课程 Header 语言导航与 publication policy 收尾；完整 `go test ./...` 的 `TestProjectedTourLinksFollowPublicationPolicy` 暴露正式 English `welcome.article` 中现有 Go local 链接仍使用无尾斜杠 `/tour`。
- 问题描述：当前 upstream `welcome.article` 中指向本项目的 French、German、Korean、Simplified Chinese 四个社区 Tour URL 分别使用 `https://fr-go-dev.shuijingwanwq.com/tour`、`https://de-go-dev.shuijingwanwq.com/tour`、`https://ko-go-dev.shuijingwanwq.com/tour`、`https://go-dev.shuijingwanwq.com/tour`；本站 canonical Tour 根入口实际为 `/tour/`，无尾斜杠形式会产生一次 `/tour` → `/tour/` 重定向。commit `58638ea` 已在 candidate validation 之后的 publication correction 层通过 audited source→canonical mapping 将正式 projection 规范化为 `/tour/`，未修改 TranslationUnit protected link target、upstream source 或 candidate identity。
- 暂缓原因：不为消除一次 upstream URL canonicalization 差异而直接修改冻结的 English source identity。维护者明确决定等后续再次向 Go 官方申请新增语言链接时，一并请求将现有四个社区链接更新为 canonical `/tour/`；在此之前继续使用本地 publication correction。
- 当前状态：`open`
- 后续处理/核销证据：当前本地修正见 commit `58638ea` 的 `internal/tourpolicy` publication correction 与 `internal/i18n` projection regression tests；待 upstream 链接改为 `/tour/` 并完成下一次 upstream sync 后，重新验证并评估删除本地 correction。当前保持 open。

### DI-20260916-002：未知或非 canonical Tour 路径在 Production 回退为 200 Tour shell

- ID：`DI-20260916-002`
- 发现日期：`2026-09-16`
- 发现阶段/场景：共享 Header HUMAN visual check 后进一步核对 Tour URL trailing-slash 与 canonical 行为，并分别对本地 server 和 `https://go-dev.shuijingwanwq.com` Production 进行 curl 验证。
- 问题描述：Production 对未精确命中正式 Tour route 的部分 `/tour/...` 路径仍会进入 SPA fallback 并返回 HTTP 200。例如 `/tour/list/` 返回 200 且 canonical 为 `/tour/`，`/tour/welcome/1/` 返回 200 且 canonical 为 `/tour/`，不存在的 `/tour/does-not-exist-20260916` 也返回 200 且 canonical 为 `/tour/`；相比之下正式 `/tour/list` 与 `/tour/welcome/1` 均返回 200 并具有各自正确 canonical。该现象与 Go upstream 已报告的 `golang/go#81482`（`x/website/tour: unknown paths return 200 instead of 404`）属于同类 Tour fallback 问题。
- 暂缓原因：维护者明确决定先等待 upstream Issue `golang/go#81482` 的调查和修复，随后按正式 upstream-sync 流程同步并重新验证本站行为，避免在 upstream 即将可能修改相同路由语义时提前维护一套 fork-specific routing implementation。若 upstream sync 后本站仍存在该问题，再实施最小本地修复。
- 当前状态：`open`
- 后续处理/核销证据：2026-09-16 公网验证结果为 `/tour` → 307 `/tour/`、`/tour/` → 200、`/tour/list` → 200 + canonical `/tour/list`、`/tour/list/` → 200 + canonical `/tour/`、`/tour/welcome/1` → 200 + canonical `/tour/welcome/1`、`/tour/welcome/1/` → 200 + canonical `/tour/`、`/tour/does-not-exist-20260916` → 200 + canonical `/tour/`。上游跟踪：https://github.com/golang/go/issues/81482 。当前保持 open，等待 upstream resolution 后同步复验。

### DI-20260916-003：mobile support copy toast 未按 visual viewport 水平居中

- ID：`DI-20260916-003`
- 发现日期：`2026-09-16`
- 发现阶段/场景：全量 browser regression；唯一失败为 `TestHomepageSupportCopyAndResponsiveLayoutInBrowser/ja-JP-mobile`。当时 working tree 与 clean HEAD 均以相同 evidence 失败，确认问题不是当日 Header 新改动引入。
- 问题描述：在 mobile `visualViewport` fixture 下，support copy toast 仍按 layout viewport 的 50% 定位，水平中心约为 242.5px；但 `visualViewport.offsetLeft=0`、`width=375`，其水平中心应为 187.5px，导致测试失败。
- 暂缓原因：维护者于 2026-09-16 明确决定本轮暂不处理，留到 2026-09-17 实施共享修复，不增加 ja-JP 特例。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-17 已完成共享修复：`_content/tour/static/js/support.js` 将 `visualViewport` 的 offset/size 同步至 CSS variables，并监听 `resize` 和 `scroll`；`_content/tour/static/css/app.css` 的 mobile toast 改用这些变量；`internal/tour/support_test.go` 将旧的“禁止 visualViewport compensation”静态断言更新为正向约束。验证结果：精确的 `ja-JP-mobile` subtest PASS；`GO_TOUR_RUN_BROWSER_TESTS=1 go test ./internal/tour -count=1` PASS（56.538s）；`go test ./...` PASS；`node --check _content/tour/static/js/support.js` PASS；`git diff --check` PASS。据此核销为 resolved。

### DI-20260919-001：A Tour of Go 的 PageUp/PageDown 导航依赖焦点位于课程容器

- ID：`DI-20260919-001`
- 发现日期：`2026-09-19`
- 发现阶段/场景：ar 首次 Production 前的 Preview HUMAN 验收结束后，为确认共享 RTL 修改是否影响既有 LTR locale，使用 fr-FR 做真实 LTR 回归时进一步比较 PageUp/PageDown 导航体验；随后在当前官方 `https://go.dev/tour/welcome/1` 上独立复现。
- 问题描述：A Tour of Go 的 PageUp/PageDown 导航事件当前绑定在 `#editor-container` 上。当焦点位于 CodeMirror editor 或课程区域内时，PageDown 可正常从 `/tour/welcome/1` 导航至 `/tour/welcome/2`；当焦点移动到 `#editor-container` 外的可聚焦 Header 控件时，例如官方站点的 `Toggle theme` 按钮或 `A Tour of Go` Logo，PageDown 不再发生导航，PageUp 存在同类行为。fr-FR 与 ar 在相同课程焦点状态下的连续 PageDown 压力测试均稳定通过，因此该现象不是 RTL 或 Arabic 特有回归，而是当前 upstream Tour 的通用焦点依赖。
- 暂缓原因：维护者明确决定本轮不为该 upstream 通用 UX 问题维护 fork-specific `directives.js` 修复，先等待 Go upstream 调查和处理；已提交 `golang/go#81596`。Arabic first-production 不因此阻塞。待 upstream 修复后按正式 upstream-sync 流程同步并重新验证；若长期未处理且实际用户影响需要本地解决，再评估最小 shared fix。
- 当前状态：`open`
- 后续处理/核销证据：上游跟踪：https://github.com/golang/go/issues/81596 。2026-09-19 对当前官方 `go.dev/tour/welcome/1` 的实测结果为：CodeMirror textarea 获得焦点时 PageDown 可从 `/tour/welcome/1` 导航到 `/tour/welcome/2`；`Toggle theme` 按钮或 `A Tour of Go` Logo 获得焦点时，PageDown 后 route 保持 `/tour/welcome/1`。当前保持 open，等待 upstream resolution 后同步复验。

### DI-20260919-002：hi-IN IndexNow closeout 受维护者本机 Mihomo DNS 状态影响

- ID：`DI-20260919-002`
- 发现日期：`2026-09-19`
- 发现阶段/场景：hi-IN 首次 Production 已完成 machine/browser acceptance、first-production finalize 并进入 `production_state=live` 后执行 Search-engine closeout；Google Search Console 与 Bing Webmaster Tools sitemap submission 已完成，随后运行 `scripts/indexnow-closeout.sh --locale hi-IN`。
- 问题描述：hi-IN IndexNow 首次执行已完成本地 key 生成与 Aliyun `PROVISIONING PASS`，但正式 Go bootstrap 在从维护者本机验证公网 root key 时以 `EOF` 失败；随后本机对 `https://hi-go-dev.shuijingwanwq.com/`、`/sitemap.xml` 和 `/<key>.txt` 的只读 curl 均返回 `curl (35) Send failure: Broken pipe` / HTTP `000`。同机对照中，`th-go-dev.shuijingwanwq.com` 在相同 Clash Verge Rev / Mihomo TUN 环境下可通过 fake-IP 正常完成 TLS 1.3 与 HTTP 200，而 `hi-go-dev.shuijingwanwq.com` 的 Mihomo service log 持续记录 `dns resolve failed: couldn't find ip`。与此同时，zgocloud 对 hi-IN 正式 hostname 可解析到 Cloudflare 公网地址并返回 HTTP 200，且该站此前正式 Production machine acceptance（含 sitemap 105/105）与 browser acceptance 均已 PASS，因此当前 evidence 不支持将问题归因于 hi-IN Production deployment 本身。
- 暂缓原因：维护者明确决定本轮不调整 Clash、本机 DNS/代理环境，也不修改现有 IndexNow/Production 部署方案；IndexNow 属于 first-production finalize 后的非阻塞 Search-engine closeout，Google 与 Bing 已完成，hi-IN 已正式 live，因此不为该问题阻塞上线或扩大实现 scope。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-19 稍后重新运行 `scripts/indexnow-closeout.sh --locale hi-IN`。现有 locale-specific key 被正确复用，Aliyun provisioning 幂等通过；第一次重试仅返回正式允许的 IndexNow probe `HTTP 202` pending，未执行 bulk submission。随后再次运行同一正式入口，继续复用同一 key，输出 `IndexNow bootstrap: PASS (locale=hi-IN sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=hi-IN)`。期间未修改 Clash/Mihomo、本机 DNS/代理环境、IndexNow network-runner 或 Production 部署方案，也未重新生成 key；因此此前问题按正式重试路径自行恢复，hi-IN Search-engine closeout 已完整完成，据此核销为 resolved。

### DI-20260921-001：桌面语言下拉中较长语言名称会换行

- ID：`DI-20260921-001`
- 发现日期：`2026-09-21`
- 发现阶段/场景：cs-CZ 首次 production 前的 preview visual HUMAN gate；维护者在真实桌面浏览器打开语言下拉列表时观察到 `Brazilian Portuguese — Português (Brasil)` 在当前菜单宽度下换为两行，同次 360 px mobile 检查中该条目保持一行。
- 问题描述：桌面语言选择器对较长语言名称允许换行，导致单个条目高度增加、列表视觉节奏不一致。当前 evidence 未显示文字截断、元素重叠、横向溢出或交互失效；正式 `scripts/verify-preview-browser.py http://127.0.0.1:41867/ cs-CZ` 同次输出 `PREVIEW SURFACE ACCEPTANCE: PASS`。该现象更适合作为共享 dropdown 宽度/换行策略的后续 UI polish 处理，具体实现方案待后续确认。
- 暂缓原因：维护者于 2026-09-21 明确决定当前可以暂缓修复，不阻塞 cs-CZ 上线；后续统一评估共享语言下拉的单行展示、可用宽度与响应式行为，不增加 cs-CZ 局部特例。
- 当前状态：`open`
- 后续处理/核销证据：原始 preview/HUMAN gate 记录见 `data/locale-surface-reviews/cs-CZ/20260921-stage-a-002.md`。2026-09-22 已决定通过现有共享 `app.css` 修复：已上线非中文站点的既有 production HTML 使用固定 shared-assets URL，待 shared-assets 正式更新后无需重新 publish 或 deploy locale；`zh-CN` 使用同源静态资源，本轮作为已确认例外暂不升级，待下次正常上游同步和部署时获得修复。本地 `TestTourHeaderTitlesAreCenteredOnDesktopAndFitCommonMobileViewports` targeted browser regression PASS，`git diff --check` PASS；shared-assets 正式部署及实际验收尚未完成，当前保持 open。

### DI-20260926-001：es-419 IndexNow 本地 locale 校验与后续公网密钥验证失败

- ID：`DI-20260926-001`
- 发现日期：`2026-09-26`
- 发现阶段/场景：es-419 完成 Stage A、Preview、首次 Production 验收及正式 `first-production finalize`，`production_state=live`；Google、Bing Sitemap 已由维护者提交。首次运行 `scripts/indexnow-closeout.sh --locale es-419` 因本地 locale 校验失败。2026-09-27 使用正式支持的 `--key-file` 显式密钥路径再次执行，进入后续公网验证阶段。
- 问题描述：首次命令返回 `indexnow closeout: FAILED: invalid locale for IndexNow local key store`。`scripts/indexnow-closeout.py` 的 `LOCALE_PATTERN` 只允许纯语言或两位大写字母地区代码，本地默认 key-store 因而不接受数字地区代码 `419`。2026-09-27 维护者在仓库外的 `go-tour-indexnow/es-419/` 建立有效密钥，使用 `scripts/indexnow-closeout.sh --locale es-419 --key-file <现有受保护密钥路径>` 绕过默认 key-store 命名限制，输出 `PROVISIONING PASS`，但正式 Go bootstrap 请求公网 `https://es-419-go-dev.shuijingwanwq.com/<key>.txt` 时返回 `EOF`，最终为 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`。因此已确认存在两个不同失败阶段：默认本地校验不兼容，以及显式密钥路径下的公网密钥读取失败；`EOF` 的根因尚未确定，不能仅据此认定 DNS、代理、服务器或 IndexNow API 故障。尚未取得 IndexNow submission PASS。
- 暂缓原因：维护者于 2026-09-27 明确决定暂时放弃重试，不调整本地代理/DNS、不修改脚本、不重复部署；现有密钥和已成功的服务端 provisioning 保留。es-419 正式 `live`，Google、Bing Sitemap 已提交；IndexNow 属于非阻塞 Search-engine closeout，不影响现有 Production gate。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-27 实际命令输出为 `PROVISIONING PASS` → `verify public IndexNow key: Get "https://es-419-go-dev.shuijingwanwq.com/<key>.txt": EOF` → `FAILED`；未发生成功提交。2026-09-28 维护者使用正式支持的显式 `--key-file /home/wangqiang/.local/share/go-tour-indexnow/es-419/d626992dae0956fb9caae5fcc2aa197654f727cc2c47eed9369932098fdc193f.txt` 再次执行，输出 `PROVISIONING PASS`；homepage probe 第一次返回 HTTP `202`，按正式 1 秒 backoff 重试后恢复，随后得到 `IndexNow bootstrap: PASS (locale=es-419 sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=es-419)`。因此 es-419 的 IndexNow Search-engine closeout 已完成，公网密钥读取与 submission failure 已恢复；但普通入口 `scripts/indexnow-closeout.sh --locale es-419` 仍会因 `LOCALE_PATTERN = ^[a-z]{2,3}(?:-[A-Z]{2})?$` 拒绝数字地区码 `419`，默认 local key-store 兼容性缺陷仍真实存在。2026-09-29 已将本地 locale 校验扩展为同时接受两位大写字母地区码与三位数字地区码（`^[a-z]{2,3}(?:-(?:[A-Z]{2}|[0-9]{3}))?$`），并新增针对 `es-419` 的生成/复用回归测试及 malformed locale 拒绝测试；`python3 scripts/indexnow-closeout-test.py` 14/14 PASS，`git diff --check` PASS。随后对维护者现有默认受保护 key-store 做只读等价验证路径：`default_key_file("es-419")` 返回 `reused`，目录唯一 entry，目录 mode `0700`、key file mode `0600`，未生成新 key、未执行远端 provisioning 或 IndexNow submission。因此默认 key-store 数字地区码兼容性缺陷已修复，本条目核销。

### DI-20260927-001：gu-IN IndexNow 公网密钥验证连续 EOF

- ID：`DI-20260927-001`
- 发现日期：`2026-09-27`
- 发现阶段/场景：gu-IN 已完成首次 Production 验收与 finalization，`production_state=live`；当前登记不重新核实各平台 Sitemap 提交状态。首次 IndexNow closeout 的公网密钥请求曾遇到 `EOF`；2026-09-27 明确重跑 `scripts/indexnow-closeout.sh --locale gu-IN` 再次复现。
- 问题描述：本次正式脚本输出 `IndexNow local key: reused` 和 `PROVISIONING PASS`，随后 Go bootstrap 对 `https://gu-go-dev.shuijingwanwq.com/<key>.txt` 的公网密钥 GET 返回 `EOF`，最终为 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`。密钥已复用，服务端 provisioning 已成功，但未取得公网密钥验证或 IndexNow submission PASS。重复 `EOF` 是真实结果，其根因尚未确定；不能仅据此判定为 DNS/代理、Nginx、CDN 或 IndexNow API 故障。
- 暂缓原因：维护者于 2026-09-27 明确决定暂时放弃，不继续重试或排查，不重新生成密钥、不重复修改服务器配置。gu-IN 仍为正式 `live`；IndexNow closeout 不属于 Production gate，原有 Sitemap 提交状态不因本次失败改变。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-27 实际重试的终态为 `IndexNow local key: reused` → `PROVISIONING PASS` → `verify public IndexNow key: Get "https://gu-go-dev.shuijingwanwq.com/<key>.txt": EOF` → `FAILED`。仅在维护者明确恢复时，先只读核对现有密钥对应的公网 URL、实际 HTTP/TLS/DNS 路径及 Production 公网验收结果，再依据实证决定是否使用现有正式 retry/恢复路径；mutation 状态不明确时不得盲目重复 provisioning 或生成新 key。后续维护者使用现有正式入口恢复，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`；probe 第一次返回 HTTP `202`，按正式 1 秒 backoff 使用同一 key 与 payload 重试后恢复，随后输出 `IndexNow bootstrap: PASS (locale=gu-IN sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=gu-IN)`。因此按 gu-IN 自身成功 evidence 核销为 resolved。

### DI-20260927-002：da-DK IndexNow 公网密钥验证 EOF，closeout 暂缓

- ID：`DI-20260927-002`
- 发现日期：`2026-09-27`
- 发现阶段/场景：da-DK 已完成 `qc-002` 122/122 A、首次 Surface Review `20260927-da-DK-stage-a-001` A gate、Preview 自动与人工验收、Production machine/browser acceptance 及 `first-production finalize`，正式 `production_state=live`。维护者确认站点地图已提交，随后执行 `scripts/indexnow-closeout.sh --locale da-DK`。
- 问题描述：本次 IndexNow 首次执行输出 `IndexNow local key: generated` 和 `PROVISIONING PASS`，表明 locale-specific 密钥已生成、服务端 provisioning 已通过。随后 Go bootstrap 对 `https://da-go-dev.shuijingwanwq.com/<key>.txt` 的公网 HTTPS GET 返回 `EOF`，以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。公网密钥验证未通过，没有取得 IndexNow submission PASS；单凭本次 `EOF` 不足以认定是维护者本机 DNS/代理、Cloudflare、源站或 IndexNow API 故障。此前正式 Production 公网与浏览器验收均已 PASS，本次 closeout 失败不改写这些历史验收。
- 暂缓原因：维护者于 2026-09-27 明确要求先记录 IndexNow 提交失败，暂不继续重试或排查；保留已生成的现有密钥与成功的 provisioning，不重新生成密钥、不重复部署或修改 production 配置。da-DK 保持 `live`，站点地图已由维护者提交；IndexNow 是非阻塞的上线后 Search-engine closeout。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-27 命令 `scripts/indexnow-closeout.sh --locale da-DK` 的实际终态为 `IndexNow local key: generated` → `PROVISIONING PASS` → `verify public IndexNow key: Get "https://da-go-dev.shuijingwanwq.com/<key>.txt": EOF` → `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`。2026-09-28 维护者按正式入口再次执行同一命令，现有 key 被正确复用，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`；homepage probe 首次返回 HTTP `202`，按正式 1 秒 backoff 重试后恢复，随后输出 `IndexNow bootstrap: PASS (locale=da-DK sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=da-DK)`。期间未重新生成 key、未修改 Production 配置，也未绕过正式 bounded retry，因此按 da-DK 自身成功 evidence 核销为 resolved。

### DI-20260928-001：sk-SK IndexNow probe 连续 HTTP 202，closeout 暂缓

- ID：`DI-20260928-001`
- 发现日期：`2026-09-28`
- 发现阶段/场景：sk-SK 已完成 `qc-002` 122/122 A、Locale Surface Review `20260928-sk-SK-stage-a-002` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，`production_state=live`；维护者确认 Google Search Console 与 Bing Webmaster Tools 的站点地图已提交。随后执行 `scripts/indexnow-closeout.sh --locale sk-SK`，并在首次失败后再次运行同一正式入口。
- 问题描述：首次执行输出 `IndexNow local key: generated` 和 `PROVISIONING PASS`，随后 homepage probe 连续三次返回 HTTP `202`（attempt 1/3、2/3 后按正式 1s/2s backoff 重试，第三次仍为 202），最终为 `IndexNow probe remained pending (HTTP 202) after 3 attempts`，以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束。第二次执行正确输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，仍以完全相同的三次 HTTP `202` pending 结果结束。两次运行都没有取得 probe HTTP 200，因此按正式 contract 均未进入 bulk submission；现有 key 已保留并被正确复用，服务端 provisioning 已成功。当前 evidence 只证明 IndexNow API probe 持续处于 pending，不能据此认定为密钥、Nginx/CDN、DNS、维护者网络或 Production deployment 故障。
- 暂缓原因：维护者于 2026-09-28 明确要求记录并暂缓处理，不继续立即重复 probe 或扩大调查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成密钥、不重复修改 production 配置。sk-SK 保持正式 `live`，Google/Bing Sitemap 已提交；IndexNow 是 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或 sitemap submission 结论。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-28 前两次正式命令 `scripts/indexnow-closeout.sh --locale sk-SK` 的实际终态分别为 `IndexNow local key: generated` / `reused` → `PROVISIONING PASS` → probe HTTP `202` 按 1s/2s bounded retry 共 3 attempts → `IndexNow probe remained pending (HTTP 202) after 3 attempts` → `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`，两次均未执行 bulk submission。稍后维护者再次运行同一正式入口，继续输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`；probe 第一次仍返回 HTTP `202`，按正式 1 秒 backoff 重试后恢复，随后输出 `IndexNow bootstrap: PASS (locale=sk-SK sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=sk-SK)`。期间未重新生成 key、未修改 Production 配置，也未绕过正式 bounded retry，因此按 sk-SK 自身成功 evidence 核销为 resolved。
### DI-20260928-002：ChatGPT TranslationUnit 重试未优先恢复可验证的部分生成成果

- ID：`DI-20260928-002`
- 发现日期：`2026-09-28`
- 发现阶段/场景：sl-SI 首次 TranslationUnit Generation 的 `chatgpt-sl-SI-001` 60 Page batch。首次 ChatGPT invocation 运行约二十多分钟后发生产品/消息流错误，维护者触发重试；重试输出显示重新按正式 Bundle 的全部 60 个 protected inputs 生成，而不是先恢复上一轮已经落地的部分成果。
- 问题描述：故障后正式 batch 仍为 60 Page、`status.tsv` 仍显示 122 pending，标准 hidden staging `/tmp/.go-tour-i18n-generation/sl-SI/chatgpt-sl-SI-001/` 为空；但 `/tmp/sl-SI-chatgpt-sl-SI-001-staging.tar.gz` 与对应 `.b64` 保留了上一轮中断的 transport artifact。容器整体因截断而失败（gzip `unexpected end of file`、base64 `invalid input`），但从 tar 中仍可确定性恢复 53/60 个完整成员，缺失仅 `moretypes-18.article` 至 `moretypes-24.article`。对这 53 个恢复文件的实际检查结果为：exact member subset 无 extra、非空、UTF-8 PASS、single-LF PASS，且与正式 `chatgpt-sl-SI-001` manifest inputs 对比的 protected-token multiset 53/53 PASS。现有 ChatGPT Generation retry/recovery 路径没有在重新生成前自动扫描并复用这类“容器整体损坏但完整成员仍可验证”的 partial transport/staging 成果，因此已完成且可证明可信的 TranslationUnit 会被重复生成。该问题不改变 formal import/process/automatic validation/QC gate；恢复结果在补齐完整 batch 后仍必须走现有 exact-set、import、process 与 validation。
- 暂缓原因：维护者明确决定先完成并提交本轮 5 门语言（塞尔维亚语、斯洛伐克语、克罗地亚语、斯洛文尼亚语、立陶宛语），之后再统一优化 Generation recovery；当前不在进行中的 locale lifecycle 中插入 tooling/schema 变更。后续优化应优先解决 retry 前的 recovery preflight，而不是通过固定小批 checkpoint 增加 RDC calls。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-29 在 5 门语言完成提交后新增 provider-neutral `generation-bundle recover`。命令先重验 CURRENT Generation Bundle、pending/provenance 与 formal batch state，再从可重复 `--source` 指定的 partial staging、单文件或 interrupted tar/tar.gz 中逐 Unit 读取候选；只有 filename 绑定正确且通过 UTF-8/no-BOM/single-LF、protected restore、glossary 与 machine candidate 检查的完整字节才写入一个新的非正式 recovery staging。损坏 archive 的 tail/integrity error 保留为 issue，但错误前完整且可独立验证的成员继续复用；同一 output 出现不同有效字节时 fail closed。命令明确返回 recovered/missing scope，Generation 只补生成 `missing` Unit，补齐 exact set 后仍走既有 import → process/retry → automatic validation，不新增 provenance/schema。新增 `TestGenerationRecoveryRecovers53Of60FromInterruptedTarGzip` 精确覆盖 60 Unit 中恢复 53、只补 7 后正常完成 60/60 import；另有多 source + invalid candidate、stale bundle fail-closed 回归测试。历史 sl-SI batch 已完成正式 import/QC/Production，因此没有为“验证恢复”倒退或重放该 live batch；原 `/tmp/sl-SI-chatgpt-sl-SI-001-staging.tar.gz` 的 53/60 历史实证继续作为本修复来源。targeted tests 与 `git diff --check` 均 PASS 后核销。

### DI-20260929-001：sl-SI IndexNow 公网密钥验证 EOF，closeout 暂缓

- ID：`DI-20260929-001`
- 发现日期：`2026-09-29`
- 发现阶段/场景：sl-SI 已完成 `qc-004` 122/122 A、Locale Surface Review `20260929-sl-SI-stage-a-002` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，`production_state=live`；维护者确认站点地图已经提交。随后执行 `scripts/indexnow-closeout.sh --locale sl-SI`。
- 问题描述：本次 IndexNow 首次执行输出 `IndexNow local key: generated` 和 `PROVISIONING PASS`，表明 locale-specific key 已生成且服务端 provisioning 已成功。随后正式 Go bootstrap 对 `https://sl-go-dev.shuijingwanwq.com/<key>.txt` 的公网 HTTPS GET 返回 `EOF`，命令以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。因此尚未取得公网密钥验证或 IndexNow submission PASS。当前 evidence 只证明 public key verification 阶段发生 EOF，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 sl-SI 正式 Production machine/browser acceptance 均已 PASS，本次失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-09-29 明确要求先记录失败，暂不继续重试或排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。sl-SI 保持正式 `live`，站点地图已提交；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或 sitemap submission 结论。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-29 实际命令 `scripts/indexnow-closeout.sh --locale sl-SI` 的终态为 `IndexNow local key: generated` → `PROVISIONING PASS` → `verify public IndexNow key: Get "https://sl-go-dev.shuijingwanwq.com/<key>.txt": EOF` → `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`。后续维护者使用现有正式入口恢复，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`；probe 第一次返回 HTTP `202`，按正式 1 秒 backoff 使用同一 key 与 payload 重试后恢复，随后输出 `IndexNow bootstrap: PASS (locale=sl-SI sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=sl-SI)`。因此按 sl-SI 自身成功 evidence 核销为 resolved。

### DI-20260929-002：et-EE IndexNow 公网密钥验证连续 EOF，closeout 暂缓

- ID：`DI-20260929-002`
- 发现日期：`2026-09-29`
- 发现阶段/场景：et-EE 已完成 Locale Surface Review `20260929-et-EE-stage-a-002` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，`production_state=live`。随后执行 `scripts/indexnow-closeout.sh --locale et-EE`，首次失败后又按同一正式入口重试一次。
- 问题描述：第一次执行输出 `IndexNow local key: generated` 与 `PROVISIONING PASS`，随后正式 Go bootstrap 对 `https://et-go-dev.shuijingwanwq.com/<key>.txt` 的公网 HTTPS GET 返回 `EOF`，以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。第二次执行正确复用同一 locale-specific key，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，但对同一公网 key URL 的验证再次返回 `EOF`，终态相同。两次均未取得 public key verification 或 IndexNow submission PASS。当前 evidence 只证明公网密钥验证阶段连续发生 EOF，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 et-EE 正式 Production machine/browser acceptance 均已 PASS，本次 closeout 失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-09-29 明确要求暂时停止 IndexNow 提交并先记录问题，不继续立即重试或扩大排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。et-EE 保持正式 `live`；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或其他已经完成的上线 gate。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-29 两次正式命令 `scripts/indexnow-closeout.sh --locale et-EE` 的终态分别为 `IndexNow local key: generated` / `reused` → `PROVISIONING PASS` → public key verification `EOF` → `FAILED`；后续两次恢复尝试均正确复用同一 key，但 probe 连续三次 HTTP `202` pending 后仍失败。2026-10-06 维护者再次使用同一正式入口与现有 key 恢复，输出 `IndexNow local key: reused` → `PROVISIONING PASS` → probe 首次 HTTP `202` 后按正式 bounded policy 复用同一 key/payload 重试 → `IndexNow bootstrap: PASS (locale=et-EE sitemap_urls=105 submitted_urls=105)` → `IndexNow closeout: PASS (locale=et-EE)`。期间未重新生成 key、未修改 Production 配置，据此核销为 resolved。

### DI-20260929-003：lv-LV IndexNow 公网密钥验证连续 EOF，closeout 暂缓

- ID：`DI-20260929-003`
- 发现日期：`2026-09-29`
- 发现阶段/场景：lv-LV 已完成 Locale Surface Review `20260929-initial-surface` PASS、首次 Production machine/browser acceptance 及正式 `first-production finalize`，`production_state=live`；维护者确认站点地图已经提交。随后连续两次执行 `scripts/indexnow-closeout.sh --locale lv-LV`。
- 问题描述：第一次执行输出 `IndexNow local key: generated` 与 `PROVISIONING PASS`，随后正式 Go bootstrap 对 `https://lv-go-dev.shuijingwanwq.com/892bc17d3ace7aa6da7671836a951635b40cb8a7f97e4250d4cc1b4e1d26ea3b.txt` 的公网 HTTPS GET 返回 `EOF`，以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。第二次执行正确复用同一 locale-specific key，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，但同一公网 key URL 再次返回 `EOF`，终态相同。两次均未取得 public key verification 或 IndexNow submission PASS。当前 evidence 只证明公网密钥验证阶段连续发生 EOF，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 lv-LV 正式 Production machine/browser acceptance 均已 PASS，本次 closeout 失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-09-29 明确要求先记录 IndexNow 失败，暂不继续重试或扩大排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。lv-LV 保持正式 `live`，站点地图已提交；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或 sitemap submission 结论。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-09-29 两次正式命令 `scripts/indexnow-closeout.sh --locale lv-LV` 的终态分别为 `IndexNow local key: generated` / `reused` → `PROVISIONING PASS` → `verify public IndexNow key: Get "https://lv-go-dev.shuijingwanwq.com/892bc17d3ace7aa6da7671836a951635b40cb8a7f97e4250d4cc1b4e1d26ea3b.txt": EOF` → `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`。后续维护者使用现有正式入口恢复，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，随后输出 `IndexNow bootstrap: PASS (locale=lv-LV sitemap_urls=105 submitted_urls=105)` 与 `IndexNow closeout: PASS (locale=lv-LV)`。因此按 lv-LV 自身成功 evidence 核销为 resolved。

### DI-20261003-001：de-CH IndexNow 公网密钥验证 EOF，closeout 暂缓

- ID：`DI-20261003-001`
- 发现日期：`2026-10-03`
- 发现阶段/场景：de-CH 已完成 Locale Surface Review `20261003-de-CH-stage-a-002` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，当前 `production_state=live`。随后执行 `scripts/indexnow-closeout.sh --locale de-CH`；本条不重新核实 Google Search Console / Bing Webmaster Tools 的 sitemap submission 状态。
- 问题描述：本次正式命令输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，说明现有 locale-specific key 被正确复用，服务端 provisioning 已成功。随后正式 Go bootstrap 对 `https://de-ch-go-dev.shuijingwanwq.com/<key>.txt` 的公网 HTTPS GET 返回 `EOF`，命令以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。因此尚未取得 public key verification 或 IndexNow submission PASS。当前 evidence 只证明公网密钥验证阶段发生 EOF，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 de-CH 正式 Production machine/browser acceptance 均已 PASS，本次 closeout 失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-10-03 明确要求先记录问题，暂不继续重试或排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。de-CH 保持正式 `live`；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或其他已完成上线 gate。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-10-03 正式命令 `scripts/indexnow-closeout.sh --locale de-CH` 的终态为 `IndexNow local key: reused` → `PROVISIONING PASS` → public key verification `EOF` → `FAILED`。2026-10-06 维护者再次使用同一正式入口与现有 key 恢复，输出 `IndexNow local key: reused` → `PROVISIONING PASS` → probe 首次 HTTP `202` 后按正式 bounded policy 复用同一 key/payload 重试 → `IndexNow bootstrap: PASS (locale=de-CH sitemap_urls=105 submitted_urls=105)` → `IndexNow closeout: PASS (locale=de-CH)`。期间未重新生成 key、未修改 Production 配置，据此核销为 resolved。
### DI-20261004-001：en-GB IndexNow 公网密钥验证 EOF，closeout 暂缓

- ID：`DI-20261004-001`
- 发现日期：`2026-10-04`
- 发现阶段/场景：en-GB 已完成 Locale Surface Review `20261004-en-GB-stage-a-001` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，当前 `production_state=live`；维护者确认 Google Search Console 与 Bing Webmaster Tools sitemap 已提交。随后执行 `scripts/indexnow-closeout.sh --locale en-GB`。
- 问题描述：本次正式命令输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，说明现有 locale-specific key 被正确复用，服务端 provisioning 已成功。随后正式 Go bootstrap 对 `https://en-gb-go-dev.shuijingwanwq.com/<key>.txt` 的公网 HTTPS GET 返回 `EOF`，命令以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。因此尚未取得 public key verification 或 IndexNow submission PASS。当前 evidence 只证明公网密钥验证阶段发生 EOF，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 en-GB 正式 Production machine/browser acceptance 均已 PASS，本次 closeout 失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-10-04 明确要求先记录问题，暂时无法提交，不继续重试或排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。en-GB 保持正式 `live`，Google/Bing sitemap 已提交；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或 sitemap submission 结论。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-10-04 正式命令 `scripts/indexnow-closeout.sh --locale en-GB` 的终态为 `IndexNow local key: reused` → `PROVISIONING PASS` → public key verification `EOF` → `FAILED`。2026-10-06 维护者再次使用同一正式入口与现有 key 恢复，输出 `IndexNow local key: reused` → `PROVISIONING PASS` → probe 首次 HTTP `202` 后按正式 bounded policy 复用同一 key/payload 重试 → `IndexNow bootstrap: PASS (locale=en-GB sitemap_urls=105 submitted_urls=105)` → `IndexNow closeout: PASS (locale=en-GB)`。期间未重新生成 key、未修改 Production 配置，据此核销为 resolved。
### DI-20261005-001：en-AU IndexNow 公网密钥验证连续 EOF，closeout 暂缓

- ID：`DI-20261005-001`
- 发现日期：`2026-10-05`
- 发现阶段/场景：en-AU 已完成 Locale Surface Review `20261005-en-AU-stage-a-001` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，当前 `production_state=live`；维护者已在 Google Search Console 与 Bing Webmaster Tools 提交 `https://en-au-go-dev.shuijingwanwq.com/sitemap.xml`。随后连续三次执行 `scripts/indexnow-closeout.sh --locale en-AU`。
- 问题描述：第一次执行输出 `IndexNow local key: generated` 与 `PROVISIONING PASS`，随后正式 Go bootstrap 对 `https://en-au-go-dev.shuijingwanwq.com/<key>.txt` 的公网 HTTPS GET 返回 `EOF`，以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。第二、第三次执行均正确复用同一 locale-specific key，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，但同一 public key verification 仍返回 `EOF`，终态相同。三次均未取得 public key verification、`IndexNow bootstrap: PASS` 或 `IndexNow closeout: PASS`。当前 evidence 只证明公网密钥验证阶段连续发生 EOF，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 en-AU 正式 Production machine/browser acceptance 均已 PASS，本次 closeout 失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-10-05 明确要求将当前状态记录在仓库文档中，后续有空时再继续提交 IndexNow，不继续立即重试或扩大排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。en-AU 保持正式 `live`，Google/Bing sitemap 已提交；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或 sitemap submission 结论。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-10-05 三次正式命令 `scripts/indexnow-closeout.sh --locale en-AU` 的实际终态分别为 `IndexNow local key: generated` / `reused` / `reused` → `PROVISIONING PASS` → public key verification `EOF` → `FAILED`。2026-10-06 维护者再次使用同一正式入口与现有 key 恢复，输出 `IndexNow local key: reused` → `PROVISIONING PASS` → probe 首次 HTTP `202` 后按正式 bounded policy 复用同一 key/payload 重试 → `IndexNow bootstrap: PASS (locale=en-AU sitemap_urls=105 submitted_urls=105)` → `IndexNow closeout: PASS (locale=en-AU)`。期间未重新生成 key、未修改 Production 配置，据此核销为 resolved。

### DI-20261005-002：he IndexNow 公网密钥验证 EOF / connection reset，closeout 暂缓

- ID：`DI-20261005-002`
- 发现日期：`2026-10-05`
- 发现阶段/场景：Hebrew（`he`）已完成 Locale Surface Review `20261005-he-stage-a-002` PASS、Preview 自动与人工验收、首次 Production machine/browser acceptance 及正式 `first-production finalize`，当前 `production_state=live`；维护者已在 Google Search Console 与 Bing Webmaster Tools 提交 `https://he-go-dev.shuijingwanwq.com/sitemap.xml`。随后连续两次执行 `scripts/indexnow-closeout.sh --locale he`。
- 问题描述：第一次执行输出 `IndexNow local key: generated` 与 `PROVISIONING PASS`，随后正式 Go bootstrap 在 public key verification 阶段对 Hebrew production hostname 的 key URL 返回 `EOF`，以 `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning` 结束（exit status 1）。第二次执行正确复用同一 locale-specific key，输出 `IndexNow local key: reused` 与 `PROVISIONING PASS`，随后同一 public key verification 返回 `read: connection reset by peer`，终态相同。两次均未取得 public key verification、`IndexNow bootstrap: PASS` 或 `IndexNow closeout: PASS`。当前 evidence 只证明公网密钥验证阶段发生两种 transport failure，不能据此进一步判定维护者本机网络、DNS/代理、Cloudflare、Nginx/源站或 IndexNow API 为根因；此前 Hebrew 正式 Production machine/browser acceptance 均已 PASS，本次 closeout 失败不改写这些验收结果。
- 暂缓原因：维护者于 2026-10-05 明确决定当前先记录状态，后续再提交 IndexNow，不继续立即重试或扩大排查；保留现有 locale-specific key 与已成功的 provisioning，不重新生成 key、不重复修改 Production 配置。Hebrew 保持正式 `live`，Google/Bing sitemap 已提交；IndexNow 属于 first-production finalize 之后的非阻塞 Search-engine closeout，不影响既有 Production、Surface Review 或 sitemap submission 结论。
- 当前状态：`resolved`
- 后续处理/核销证据：2026-10-05 两次正式命令 `scripts/indexnow-closeout.sh --locale he` 的实际终态分别为 `IndexNow local key: generated` / `reused` → `PROVISIONING PASS` → public key verification `EOF` / `read: connection reset by peer` → `indexnow closeout: FAILED: formal IndexNow bootstrap failed after provisioning`。2026-10-06 第一次恢复仍复用同一 key，并在 public key verification 再次返回 `EOF`。随后维护者再次使用同一正式入口与现有 key 执行，输出 `IndexNow local key: reused` → `PROVISIONING PASS` → probe 首次 HTTP `202` 后按正式 bounded policy 复用同一 key/payload 重试 → `IndexNow bootstrap: PASS (locale=he sitemap_urls=105 submitted_urls=105)` → `IndexNow closeout: PASS (locale=he)`。期间未重新生成 key、未修改 Production 配置，据此核销为 resolved。


### DI-20261019-001：zh-CN Site Homepage Preview 图片断言失败

- ID：`DI-20261019-001`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`site-v2-playground-ui-bootstrap-correctness` 完成 19 个 targeted tests 后，对新完整 zh-CN Preview 执行 `scripts/verify-preview-browser.py http://127.0.0.1:42097/ zh-CN`。
- 问题描述：首次完整验收在 `stage=browser check=rendered/interaction`、Homepage `/`、desktop viewport `1280` 处返回 `Site content/assets/overflow`；实际 `width=1265`、`overflow=0`、`anchors=True`、`images=False`，终态 FAILED。相同页面已输出“程序已退出.”，原缺失 UI bootstrap 的 TypeError 不再出现。首次 evidence 只证明 main 图片断言为 false，尚未定位具体图片或原因；当时未据此判定翻译、资源、加载时序或 verifier 有错。
- 暂缓原因：用户明确要求本轮仅修复共享 Playground UI bootstrap，若完整验收出现另一独立 blocker，则停止并保留 Preview/logs；不扩展到图片问题，不改语言 artifacts、Surface、activation 或 Production。
- 当前状态：`resolved`
- 原始证据：Preview `http://127.0.0.1:42097/`，监听 PID `592566`；完整失败日志 `/tmp/go-tour-playground-bootstrap-acceptance.log`，bootstrap HTTP proof `/tmp/go-tour-playground-bootstrap-http.json`。
- 2026-10-19 图片诊断与修复：逐项采集全部 33 张 `main img` 的 URL、loading、完成状态、自然尺寸、视口位置与 Network 事件；唯一未加载者为 `/images/gophers/biplane.svg`，`loading=lazy`，top=2598、viewport height=800，`complete=false`、自然尺寸 0×0、`currentSrc` 为空且没有请求。滚入视口后请求 HTTP 200，`complete=true`、自然尺寸 190×150，33 张图片全部加载成功，无 `Network.loadingFailed`。根因为 verifier 在触发屏外 lazy 加载前就要求所有图片已加载，非资源交付或翻译缺陷。完整逐图与网络证据：`/tmp/go-tour-homepage-image-diagnosis.json`。
- 修复与 targeted tests：仅调整 `scripts/verify-preview-browser.py`，将未完成图片滚入视口并有界等待，检查 currentSrc 与非零自然宽高，恢复原滚动位置，继续检查所有网络事件；保留 source-bound assets、links 和布局断言。`PreviewImageAcceptanceTest` 的 4 个测试及 7 个直接相关既有测试共 11 PASS；真实 Chrome 用桌面 1280×800、手机 375×812 验证 eager/lazy 成功，以及观察后 404、HTTP 200 无效图片字节、连接失败仍 FAIL。既有 targeted tests 同时覆盖缺图、显式链接丢失、runtime/network 异常、pending resource、双视口与 mobile overflow。
- 后续处理/核销证据：本轮未改变 Go handler、运行时、HTML 或资源字节，因此复用原 Preview `http://127.0.0.1:42097/`。完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:42097/ zh-CN` 已通过 Homepage 桌面和手机分支，随后在 `/learn/` 的 `setDownloadLinks` 遇到独立 TypeError（见 `DI-20261019-002`），完整日志 `/tmp/go-tour-homepage-image-acceptance.log`。按本轮要求，完整 Preview 未 PASS 前不得核销，故仍为 open；保留 Preview，不继续处理独立 blocker。
- 2026-10-19 后续验收：下载 DOM adapter 修复后，新完整 Preview `http://127.0.0.1:45417/` 再次通过 Homepage 分支，随后在 `/learn/` 的 verifier 导航点击表达式遇到独立异常（`DI-20261019-003`），完整日志 `/tmp/go-tour-download-dom-acceptance.log`。整体 Preview 仍未 PASS，继续保持 open。
- 2026-10-19 Learn anchor verifier 修复后再次执行完整验收，在 `/learn/` 遇到 `jumplinks.js` 的独立 runtime 异常（`DI-20261019-004`），完整日志 `/tmp/go-tour-learn-anchor-acceptance.log`。整体 Preview 未 PASS，继续保持 open。
- 2026-10-19 fragment preservation 修复后，新 Preview `http://127.0.0.1:43499/` 已完成全部 Site browser samples（含 Homepage/Learn 的桌面与手机检查），随后在 language selector count 返回 expected=66、actual=0（`DI-20261019-005`）。日志 `/tmp/go-tour-fragment-acceptance.log`；整体 gate 仍 FAILED，保持 open。
- 2026-10-19 正式核销：Homepage 图片 desktop/mobile、lazy observation 与全部 Site browser samples 已通过，满足整体 gate 核销规则。新 full Preview `http://127.0.0.1:43811/`（监听 PID `610395`，go-run 父进程 `610330`，ready=122、pending=0、blocked=0）完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43811/ zh-CN` exit=0，终态 `PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-escape-focus-preview-20261019.log`，完整验收日志 `/tmp/go-tour-escape-focus-acceptance-20261019.log`。未执行 Surface、activation 或 Production。


### DI-20261019-002：Site Learn Preview 下载链接初始化抛出 TypeError

- ID：`DI-20261019-002`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`site-v2-homepage-image-acceptance-correctness` 的 11 个 targeted tests PASS 后，对现有完整 zh-CN Preview 执行完整 browser verifier。
- 问题描述：Homepage 桌面/手机分支通过后，在 `/learn/` 返回 `stage=browser errors`、`Runtime.exceptionThrown`、`Uncaught (in promise)`：`TypeError: Cannot set properties of null (setting 'href')`，调用帧 `setDownloadLinks`，URL `http://127.0.0.1:42097/_/site-runtime/js/site.js:239:29`。首次仅记录真实异常，未推断具体缺失元素或修改下载 runtime 修复。
- 暂缓原因：用户明确要求若完整验收出现另一独立 blocker，则停止并保留证据；本轮仅修复图片验收语义，不扩展调查该 runtime 路径。
- 当前状态：`resolved`
- 后续处理/核销证据：Preview `http://127.0.0.1:42097/`、PID `592566` 保留；完整失败日志 `/tmp/go-tour-homepage-image-acceptance.log`。完整 Preview gate 仍 FAILED，尚无核销证据；未修改翻译、Surface、activation、Production 或运行时字节。
- 2026-10-19 精确根因与修复：frozen Learn HTML 合法包含 `.js-latestGoVersion`、generic `.js-downloadBtn` 和 `.GoVersionSpan`，但没有 Win/Mac/Linux 三个平台 anchor；frozen `setDownloadLinks` 错把版本容器存在当作三个 anchor 存在的证明。旧 harness 对 `.js-download*` 自动制造 stub，掩盖了该形状。现从真实 frozen Homepage/Learn HTML 分别提取独立节点，缺失节点返回 null；`projectRuntimeScript` 对唯一且精确匹配的完整 frozen 函数插入 guard：零个平台节点时保留 generic 按钮并跳过平台赋值，三者齐全时保留原逻辑，部分节点缺失则抛出明确错误。独立 `setVersionSpans()`、下载 authority/query 和本地 `/dl/` 404 不变；不改 frozen source 或语言 artifacts。
- Targeted tests：`TestDownloadRuntimeProjection`、`TestDownloadRuntimeDOMShapes`、`TestDownloadRuntimeStructureDriftFailsClosed`、`TestDownloadRuntimeDeliveryAnd404`、`TestDownloadRuntimeResourcePolicy` 共 5 PASS，覆盖两个真实 DOM 形状、全部 6 种部分平台集合、无版本节点、精确 URL/版本 suffix、generic href/版本显示、源结构 drift 拒绝、Preview/sealed-handler 同策略与 true 404；`PreviewSiteV2Test.test_browser_errors_and_network_failures_are_not_silent`、`test_navigation_retry_cannot_erase_browser_errors` 共 2 PASS。`_content/js/site.js` 与 `_content/learn/index.md` 修复前后 SHA 相同。
- 新 Preview 验收证据：启动命令 `go run -mod=readonly ./cmd/tour-i18n preview --locale zh-CN --http 127.0.0.1:0`，URL `http://127.0.0.1:45417/`，ready=122、pending=0、blocked=0，启动日志 `/tmp/go-tour-download-dom-preview.log`。完整 verifier 已通过 Homepage 并进入 Learn 渲染/图片检查，原 `setDownloadLinks` 异常未再出现；随后在 Learn navigation anchor click 的 verifier 表达式遇到独立异常（`DI-20261019-003`）。完整日志 `/tmp/go-tour-download-dom-acceptance.log`，按用户要求立即停止并保留新 Preview。尚未完整越过 `/learn/`，不提前核销，状态仍为 open。
- 2026-10-19 后续验收：Learn anchor verifier 修复后复用 `http://127.0.0.1:45417/` 执行完整命令，终态为 `DI-20261019-004` 的 `jumplinks.js` runtime 异常，日志 `/tmp/go-tour-learn-anchor-acceptance.log`。用户本轮明确要求本项与 DI-001 均等待整体 Preview PASS，故仍为 open。
- 2026-10-19 后续验收：新 Preview `http://127.0.0.1:43499/` 的 Site browser samples（含真实 Homepage/Learn 下载 runtime 路径）已完成；完整 verifier 随后因 `DI-20261019-005` 失败。整体 Preview 未 PASS，仍为 open，日志 `/tmp/go-tour-fragment-acceptance.log`。
- 2026-10-19 正式核销：Homepage/Learn 下载 runtime 与版本显示路径均通过，无原 setDownloadLinks 异常，满足整体 gate 核销规则。新 full Preview `http://127.0.0.1:43811/`（监听 PID `610395`，go-run 父进程 `610330`，ready=122、pending=0、blocked=0）完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43811/ zh-CN` exit=0，终态 `PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-escape-focus-preview-20261019.log`，完整验收日志 `/tmp/go-tour-escape-focus-acceptance-20261019.log`。未执行 Surface、activation 或 Production。


### DI-20261019-003：Learn Preview 导航点击验收表达式查询到 null

- ID：`DI-20261019-003`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`site-v2-download-dom-shape-correctness` 的 7 个 targeted tests PASS 后，对新完整 zh-CN Preview 执行完整 browser verifier。
- 问题描述：`stage=browser check=rendered/interaction`，route=`/learn/`，具体 check=`Learn navigation anchor click`；verifier 的 JavaScript 表达式 `const a=document.querySelector('.LearnNav a[href^="#"]'); a.click(); ...` 抛出 `TypeError: Cannot read properties of null (reading 'click')`，source 为空，表达式 line=2、column=21。首次仅证明该点击表达式查询不到 anchor，当时未推断 renderer、selector、链接重写或语言数据的具体原因。
- 暂缓原因：用户明确要求若完整验收出现另一独立 blocker，则停止、保留新 Preview/log/evidence，不扩展范围。
- 当前状态：`resolved`
- 后续处理/核销证据：Preview `http://127.0.0.1:45417/` 保留，启动日志 `/tmp/go-tour-download-dom-preview.log`，完整失败日志 `/tmp/go-tour-download-dom-acceptance.log`。未继续诊断或修改导航、verifier、翻译、Surface、activation 或 Production；整体 Preview gate 仍 FAILED。
- 2026-10-19 精确修复：verifier 误把 raw href 以 `#` 开头当作 same-document fragment 的完整定义。现按浏览器 URL parser 解析 LearnNav 链接，仅选择 same origin、same pathname、decoded fragment 有实际内容的 anchor，执行实际 click 并检查 hash 确实变化为预期值、origin/path 保持一致、target 存在且具有非零宽高、可见样式与非空渲染文本。无合格链接、缺 target、阻止/错误点击、无实质渲染目标和 evaluation exception 均 fail closed；不改 rendered links、Site runtime、frozen source 或语言数据。
- Targeted tests：`PreviewLearnNavigationTest` 的 5 个测试覆盖 fragment-only、path-qualified、absolute same-page、URL/fragment 解析、跨 route/origin 排除、缺失/空/不可见目标、点击失败、JavaScript exception 与双视口强制执行；另运行直接相关的 4 个既有 browser error/navigation retry、双视口 image/layout/mobile/Playground orchestration tests，共 9 PASS。
- 本轮完整验收证据：verifier/tests/docs-only 修改未改变 Go Preview handler、HTML、runtime 或 resource bytes，因此复用 `http://127.0.0.1:45417/`。完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:45417/ zh-CN` 的终态为 `/learn/` 的 `jumplinks.js:45:34` runtime exception（`DI-20261019-004`），完整日志 `/tmp/go-tour-learn-anchor-acceptance.log`。原 `null.click()` 不再是本次终态，但该日志没有独立的 Learn click PASS 标记，且仍在同一路由失败；不据此提前核销，继续保持 open。按用户要求立即停止，保留 Preview，不调查新 runtime blocker。
- 2026-10-19 后续证据：新 Preview `http://127.0.0.1:43499/` 的完整 verifier 已正常返回整个 `site_acceptance(chrome)` callback（包括 Learn 双视口语义点击和最终 browser error 检查），随后才在 `preview_acceptance` 的 language selector count 断言失败（`DI-20261019-005`）。这次已证明真实 Learn 点击检查通过；按本轮用户要求，本项继续等待整体 Preview PASS，保持 open。日志 `/tmp/go-tour-fragment-acceptance.log`。
- 2026-10-19 正式核销：Learn 双视口语义 anchor click 及后续 browser error 检查通过，满足整体 gate 核销规则。新 full Preview `http://127.0.0.1:43811/`（监听 PID `610395`，go-run 父进程 `610330`，ready=122、pending=0、blocked=0）完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43811/ zh-CN` exit=0，终态 `PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-escape-focus-preview-20261019.log`，完整验收日志 `/tmp/go-tour-escape-focus-acceptance-20261019.log`。未执行 Surface、activation 或 Production。


### DI-20261019-004：Learn Preview jumplinks observer 对 null 读取 classList

- ID：`DI-20261019-004`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`preview-learn-navigation-anchor-semantics-correctness` 的 9 个 targeted verifier tests PASS 后，复用现有完整 zh-CN Preview 执行完整 browser verifier。
- 问题描述：`stage=browser errors`，check=`/learn/`，事件 `Runtime.exceptionThrown`，`Uncaught (in promise)`：`TypeError: Cannot read properties of null (reading 'classList')`，调用帧 `IntersectionObserver.callback`，runtime URL `http://127.0.0.1:45417/js/jumplinks.js:45:34`。首次只记录真实调用帧及异常，当时未推断具体缺失节点或 source/runtime 原因。
- 暂缓原因：用户明确要求若 Learn verifier 修复后的完整验收遇到另一独立 blocker，则保留 exact evidence 并停止，不扩展到 Site runtime。
- 当前状态：`resolved`
- 后续处理/核销证据：Preview `http://127.0.0.1:45417/` 保留，完整失败日志 `/tmp/go-tour-learn-anchor-acceptance.log`。未继续诊断或修改 runtime、frozen source、翻译、Surface、activation 或 Production；整体 Preview gate 仍 FAILED。
- 2026-10-19 精确根因：current shell renderer 的 href walk 调用 `Resolver.ResolveLink`，该函数将 `#section` 经 `base.ResolveReference` 补成当前路径，再由 `Destination` 应用 coverage。这样虽保持浏览器 URL 含义，却破坏了 frozen jumplinks 的 `[href='#${id}']` raw attribute contract。frozen Learn source 与 jumplinks 原本一致，非 upstream 或翻译缺陷。
- 最小修复：仅在共享 `ResolveLink` 的原 URL parsing/route-context validation 后，保留纯非空 fragment 的 fragment-only 形式，沿用 `url.URL.String()` 的现有转义；空 `#` 继续原解析。path/query、alias、external、相对跨文档与 uncovered externalization 不变；未修改 frozen renderer、Learn source 或 jumplinks。browser plan 的 required fragment target 同步保留该形式，source-bound link gate 仍生效。
- Targeted tests：5 个 Go tests（`TestResolveLinkFragmentPreservation`、`TestFrozenLearnFragmentDeliveryAndJumplinks`、`TestSiteRendererFormatsAndLinks`、`TestPreviewBrowserPlanSourceBoundLinks`、`TestSiteFoundationAliasesAndSitemap`）及 7 个直接相关 verifier tests PASS。真实 frozen Learn source 经 current shell 与 Preview/Production delivery 后，全部 LearnNav href 与 source 一致；直接执行未修改的 jumplinks，observer 可找到并激活每个对应 anchor，删除所需 anchor 的负例仍拒绝。编码/非 ASCII fragment、query、alias、覆盖/未覆盖路径与相对跨页链接有 focused regression；冻结 Learn/jumplinks 文件 SHA 保持不变。
- 新完整验收：启动 `go run -mod=readonly ./cmd/tour-i18n preview --locale zh-CN --http 127.0.0.1:0`，新 URL `http://127.0.0.1:43499/`，ready=122、pending=0、blocked=0；启动日志 `/tmp/go-tour-fragment-preview.log`。完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43499/ zh-CN` 已完成全部 Site browser samples（含 Homepage/Learn、两种 viewport、Learn click 与 browser/runtime/network 检查），之后在 language selector count 遇到 `DI-20261019-005`。调用顺序由 `scripts/browser_acceptance.py` 的 `preview_acceptance` 确认：先正常返回 Site callback，再执行该断言。完整日志 `/tmp/go-tour-fragment-acceptance.log`。停止并保留新 Preview；整体 gate 未 PASS，暂保持 open。
- 2026-10-19 正式核销：Learn 保留 fragment-only href，未修改的 jumplinks 在真实双视口交互中无异常，满足整体 gate 核销规则。新 full Preview `http://127.0.0.1:43811/`（监听 PID `610395`，go-run 父进程 `610330`，ready=122、pending=0、blocked=0）完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43811/ zh-CN` exit=0，终态 `PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-escape-focus-preview-20261019.log`，完整验收日志 `/tmp/go-tour-escape-focus-acceptance-20261019.log`。未执行 Surface、activation 或 Production。


### DI-20261019-005：完整 Preview language selector count 断言失败

- ID：`DI-20261019-005`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`site-v2-same-document-fragment-preservation-correctness` 的 12 个 targeted tests PASS 后，对新的完整 zh-CN Preview 执行完整 browser verifier。
- 问题描述：Site browser callback 正常返回后，完整验收返回 `stage=browser check=rendered/interaction expected=PASS actual=language selector count: expected=66 actual=0`。当前只记录 count 断言的真实结果，不推断语言列表数据、renderer、selector 或覆盖策略的具体原因。
- 暂缓原因：用户明确要求若 `/learn/` 通过后出现另一独立 blocker，则停止并保留 exact evidence，不自动扩大范围。
- 当前状态：`resolved`
- 后续处理/核销证据：保留 Preview `http://127.0.0.1:43499/`；启动日志 `/tmp/go-tour-fragment-preview.log`，完整失败日志 `/tmp/go-tour-fragment-acceptance.log`。未继续调查或修改语言选择器、翻译、Surface、activation 或 Production；整体 gate 仍 FAILED。
- 2026-10-19 shared chrome 修复：按维护者产品 authority，新增唯一 `tour.SharedChromeModel` + shared template/CSS/runtime，Site/Tour canonical delivery 共同使用。About 与 language selector 接入全页 header；Site 手机端复用 existing drawer，Tour 原无 drawer 时由同一 adapter 提供唯一宿主；同一个 navigation node 随 viewport 移动，不渲染 hidden compatibility list。ProjectFooter 每页 exactly once，Site 放在 upstream primary/footer legal 两段之间，Tour 替换原项目 footer；保留两行语义、exact canonical Home/GitHub/copyright，zh-CN legal policy 使用现有 Project ICP metadata，非 zh-CN 不继承。registry、UI catalog、frozen templates 与正式语言 evidence 不变。
- Targeted evidence：共享 model、真实 Tour renderer、Site Homepage/Learn/Docs/Security/future-package fixture，以及既有 registry/header 和直接相关 fragment tests 共 7 个 Go tests PASS；shared verifier 正反例、keyboard-check orchestration、既有 browser error/Learn navigation/entrypoint contract 共 13 个 Python tests PASS。静态响应中完整 registry order/labels/URLs/current marker、footer exactly-once/位置/metadata 与 non-zh legal policy 均有覆盖；真实 desktop/mobile browser interaction 尚未执行，不能据此宣告视觉验收 PASS。
- 新 Preview gate：执行 `go run -mod=readonly ./cmd/tour-i18n preview --locale zh-CN --http 127.0.0.1:0` 后，在监听前复制临时 `_content/doc/go1.md` 时失败：`disk quota exceeded`（`DI-20261019-006`）。启动日志 `/tmp/go-tour-shared-chrome-preview.log`。新实例未 ready、没有 URL/PID，完整 verifier 未运行；未把旧 `43499` 当作新实现验收。按用户独立 blocker 停止规则不清理磁盘、不重试、不扩大诊断；本项及 DI-001..004 均仍 open，等待完整 Preview PASS。
- 2026-10-19 正式核销：代表性 Homepage/Learn/Docs/Security/Tour 页面的共享 About、registry exact set/order/labels/URLs/current、footer metadata 与两种视口键盘交互通过，满足整体 gate 核销规则。新 full Preview `http://127.0.0.1:43811/`（监听 PID `610395`，go-run 父进程 `610330`，ready=122、pending=0、blocked=0）完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43811/ zh-CN` exit=0，终态 `PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-escape-focus-preview-20261019.log`，完整验收日志 `/tmp/go-tour-escape-focus-acceptance-20261019.log`。未执行 Surface、activation 或 Production。


### DI-20261019-006：新完整 Preview 因 disk quota exceeded 无法启动

- ID：`DI-20261019-006`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`site-v2-global-shared-chrome-correctness`，focused code/tests 完成后启动新的完整 zh-CN Preview。
- 问题描述：真实命令返回 `tour-i18n: copy Tour content: Copy /tmp/go-tour-i18n-preview-zh-CN-2686773240/_content/doc/go1.md: write /tmp/go-tour-i18n-preview-zh-CN-2686773240/_content/doc/go1.md: disk quota exceeded`，exit status 1。失败发生在监听前，非 120 秒 timeout；没有新 ready URL/PID，不推断磁盘容量、inode、quota 或其他根因细节。
- 暂缓原因：用户明确要求遇到新的 independent blocker 立即停止，保留日志/evidence，不自动扩大范围；未获本轮磁盘清理授权。
- 当前状态：`resolved`
- 后续处理/核销证据：启动日志 `/tmp/go-tour-shared-chrome-preview.log`；旧 Preview `http://127.0.0.1:43499/` 保留但不是新 shared-chrome 的最终验收实例。未运行完整 verifier、Surface、activation 或 Production；需先处理环境 blocker，再启动新的完整 Preview 并执行 complete acceptance。
- 2026-10-19 环境恢复（维护者 terminal evidence）：停止 stale zh-CN Preview，删除 13 个 stale Preview roots，回收约 2.4 GB，`/tmp` 可用空间增至约 4 GB；fresh Preview `http://127.0.0.1:43869/` 已 ready，parent PID `607520`，日志 `/tmp/go-tour-shared-chrome-preview-retry-20261019.log`。本项状态更新为 `resolved`；仅核销环境阻塞，不代表整体 Preview PASS。新验收在语言菜单 Escape/focus 失败，另记 DI-20261019-007；本轮未重复清理。

### DI-20261019-007：真实 Preview shared language menu Escape/focus 失败

- ID：`DI-20261019-007`
- 发现日期：`2026-10-19`
- 发现阶段/场景：`shared-language-menu-escape-focus-correctness`，完整 Preview verifier 对 `http://127.0.0.1:43869/` 执行 shared keyboard interaction。
- 当前状态：`resolved`
- 精确根因：首个失败面是 Site Homepage `/`，375×812（mobile media=true），Learn 同样受影响。语言菜单打开且焦点位于真实语言 anchor；Escape keyDown 后 `details.open=false`、summary focused、drawer active/aria-expanded=true；同一次 Escape keyUp 后，frozen `_content/js/site.js` 的 drawer 冒泡 keyup listener 再次执行，关闭 drawer（aria-expanded=false）并把焦点移到 `.js-headerMenuButton`。共享 capture keydown 确实执行；桌面及 Tour 的同项交互正常。无 CSP response header/meta/console violation，非 inline runtime delivery 问题。
- 修复：共享 runtime 只消费自己处理的 Escape 完整 keyDown/keyUp 周期，防止同一个物理按键同时关闭内外两层；未处理的 Escape 正常传播，保留 native details/summary、frozen Site runtime、普通 Tab。verifier 保留 exact summary focus 断言，并补充第一 Escape 保持 drawer 打开、第二 Escape 聚焦 toggle 的断言。
- Targeted evidence：实际 shared renderer + frozen Site drawer 的 real Chrome fixture 覆盖 Site/Tour、1280/375、首/中/末语言链接、两级 Escape、长按、菜单外焦点、无菜单 Escape 与普通 Tab。3 个真实 Chrome tests PASS；2 个直接相关 Go tests 与 5 个 shared verifier tests PASS，菜单不关闭/焦点未恢复仍 FAIL。诊断记录 `/tmp/go-tour-escape-diagnosis.json`，事件阶段记录 `/tmp/go-tour-escape-event-conflict.json`；修复前完整失败日志 `/tmp/go-tour-shared-chrome-acceptance-retry-20261019.log`。
- 核销规则：新 full Preview 的 complete verifier 必须真实通过该交互；不能仅凭 focused tests 核销。DI-001..005 仍等待整体完整 Preview PASS。
- 2026-10-19 正式核销：真实 Site/Tour 两视口语言菜单 Tab、Escape 关闭且 summary 精确聚焦通过；手机第一 Escape 保持 drawer 打开，第二 Escape 关闭且 toggle 精确聚焦。原失败未再出现。新 full Preview `http://127.0.0.1:43811/`（监听 PID `610395`，go-run 父进程 `610330`，ready=122、pending=0、blocked=0）完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43811/ zh-CN` exit=0，终态 `PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-escape-focus-preview-20261019.log`，完整验收日志 `/tmp/go-tour-escape-focus-acceptance-20261019.log`。未执行 Surface、activation 或 Production。

### DI-20261019-008：Site v2 visual HUMAN gate shared-chrome / Docs presentation defects

- ID：`DI-20261019-008`
- 发现日期：`2026-10-19`
- 发现阶段/场景：维护者对 automated PASS Preview `http://127.0.0.1:43811/` 人工验收，明确 `HUMAN GATE=FAILED`。
- 当前状态：`open`
- 已确认实际 findings：HG-001 Site header 字号/层级不协调；HG-002 长语言名换行；HG-003 About 错指 Homepage；HG-004 project footer 强制 underline；HG-005 footer 不应跨 host 强制 alignment；HG-006 getting-started 丢失 Docs shell presentation；HG-007 handle-errors code block 缺正常背景；HG-008 Tour 语言列表水平居中；HG-009 Security 样式不正确；HG-012 新 Site-style drawer 干扰 Tour-native course navigation。这些是本轮 engineering defects，不是翻译缺陷或放宽 verifier 的依据。
- 2026-10-20 consolidation：维护者再次明确 `HUMAN_GATE=FAILED`。本轮恢复 `/translation/` legacy presentation model、validated coverage directory 与未来 rich-section 承载；Tour 手机语言 trigger 保留 globe、隐藏 locale text/额外 arrow，桌面和 Site host 保持原控件。跨包/跨 locale code-comment audit 与 affected-only 计划见 [注释完整性合同](CODE_COMMENT_INTEGRITY.md)及本轮 `/tmp/code-comment-*-20261020` evidence。`vuln/database` 已纳入 machine suspect scope；没有翻译、rating、reopen 或修改任何正式语言产物。只有全部 formal successors 完成后的 fresh automated Preview 与 Maintainer HUMAN gate 明确 PASS 才能核销，当前仍 open；本轮不启动 Preview。
- 2026-10-20 source coverage closure：235 个原 unsupported block 中，4 个 Tour fragment 由关联 Example Go AST 证明无注释，29 个 go.mod 示例通过严格 grammar；新增 6 个 protected slots，未新增 teaching slots。仍有 202 个 ambiguous（Tour 2、Learn/Docs 200），完整 exact residual 见 `/tmp/code-comment-block-disposition-20261020.json`；source coverage 为 `PARTIAL_UNSUPPORTED`，gate 不 ready、不 active，不运行完整跨 locale audit。单独 zh-CN current-target 已知 slots 比较仍有 39 个 exact carry，均在既有 8 个 Learn/Docs suspect Page；这不证明未知块无漏译。future source-only `/doc/`、`/doc/install` 预检分别还有 1、7 个未知块，记录在 `/tmp/code-comment-frontier-source-preflight-20261020.json`，属于 future adoption blocker，不计入当前 49 Page。保留已通过 targeted tests 的分类，不放宽 parser、不修改正式语言产物；本 issue 仍 open。
- 2026-10-20 residual successor：原 230 个 proven block 的 disposition/language/proof 与原 299 个 slots 保留，202 个 residual 通过 source-owned transcript、严格 Go/template/module/workspace、JSON、受限 SQL 与 literal/notation grammar 闭合。current 为 432 blocks / 301 slots（teaching 259、protected 42、unclassified 0），current ambiguity 与两个 frontier ambiguity 均为 0；frontier 8 blocks 仍不属于 current package。新 zh-CN current audit 为 41 个 exact carry、9 个 Learn/Docs suspect Page，新增 `/doc/security/vuln/editor`；known database sample 保留。唯一一次跨 locale current audit 为 Tour 55 / Learn/Docs 1、6187 comparisons、53 non-English exact carry，missing/protected/structure mismatch 均 0；既有 10 个 current-target-unavailable locale 不作 lifecycle 诊断。最终 machine scope 见 `/tmp/code-comment-review-scope-20261020.json`，只提议 affected Units，不启动 Reviewer、不生成译文。source gate ready、active 仍 no；HUMAN gate 仍 FAILED，本 issue 仍 open。
- 原因与修复：共享 CSS/adapter 没有区分 host-native presentation；现共享 model/metadata/registry/键盘语义，Site 保持原 go.dev 导航与 footer container，Tour 保持 compact header、左对齐 dropdown 和居中项目 footer，不再建立额外 drawer。About 统一 `/translation/`，语言 label 单行且菜单有 viewport 边界，移除 shared underline override。Docs base shell 未使用 inherited doclayout，新增 generic delivery adapter 执行 frozen site.tmpl 自带 doclayout，恢复 `.Doc.Article` 与 metadata/code presentation，不改精确绑定的 shell_render.go 或 frozen source。
- Formal follow-up：HG-010 项目页覆盖目录可发现性、HG-011 `/doc/` 与 `/doc/install` 两个 source 强候选、HG-013 项目范围/地区策略/申请链接/PayPal/baseline 展示扩展，均仅做只读 audit/proposal，见 [content frontier audit](SITE_V2_CONTENT_FRONTIER_AUDIT.md)。未扩大 package authority，未生成 localized prose/UI strings 或修改 source/target artifacts；所述 8 个申请链接 authority 尚未在当前项目页找到，PayPal public URL missing，不猜测。
- 核销规则：focused tests 和 automated full Preview PASS 均不能核销本项；只有维护者下一次 HUMAN GATE 明确 PASS 才能 resolved。在此之前禁止 sealed build、publish、Production、activation、新 Surface 或 formal content generation。
- 本轮验证：7 个 focused Go tests（Site generic Docs / Homepage / shared families，Tour shared host/model/delivery/keyboard fixture）、7 个 Python shared-verifier/error tests、4 个 real Chrome keyboard/presentation tests，共 18 PASS。完整新 Preview `http://127.0.0.1:39333/`（监听服务 PID `627256`，go-run 父进程 `627206`；ready=122、pending=0、blocked=0）执行 `python3 scripts/verify-preview-browser.py http://127.0.0.1:39333/ zh-CN`，exit=0、`PREVIEW SURFACE ACCEPTANCE: PASS`。启动日志 `/tmp/go-tour-human-gate-correction-preview-20261019.log`，完整验收日志 `/tmp/go-tour-human-gate-correction-acceptance-20261019.log`。
- Focused 实页证据：7 个 Homepage/普通 Tutorial/含 code Tutorial/Security/Tour desktop/mobile presentation probes PASS，computed styles / menu geometry 为 `/tmp/go-tour-human-gate-visual-proof.json`，截图 `/tmp/go-tour-human-gate-*.png`。手机 Site drawer 的几何测量在 frozen `transform 100ms` transition 的 finished 后进行，不以刚切换 class 时仍在 viewport 外的过渡坐标判断最终 layout；产品与 verifier 未因此增加 sleep 或豁免。Docs 的 Article 宽度、pre 背景与 header 字号/行高、两类 footer alignment 均有实页 computed-style evidence。
- 人工复验注意：当前 shell `page` context 对没有 title context 的父节点仍提供空标题，恢复 frozen Docs breadcrumbs 后可见空的祖先 crumb（如 handle-errors 截图）；自动 gate 不等于该视觉细节已获人工认可。与缺失 Docs root 等内容前沿一起保留在本项后续人工复验范围，不新增未经审核的父页标题。当前终态仅为 `HUMAN_GATE=PENDING_RECHECK`，本项继续 `open`；未执行 sealed build、publish、Production、activation、Surface 或正式语言生成。
- 2026-10-19 breadcrumb correctness recovery：维护者确认 `39333` 的 getting-started / handle-errors 存在多个 ancestor empty visible labels，这是实际 rendered UI defect，不能仅作为 audit note。精确原因是 `shellFuncs.page(route)` 返回 `title=""` 的占位 context；`applyDocsPresentation` 恢复 frozen breadcrumb 模板后沿用了该 helper，既有 verifier 没有空标签 gate。
- 本轮工程修复：Docs adapter 将 `page` 绑定到 current completed ancestor metadata；目录拼写匹配 exact local canonical route。未覆盖祖先只读取 frozen manifest 中存在且 SHA/size/mode 验证通过的 source metadata，并由原 Resolver externalize；不注册新 route、不新增本地父页或 fallback title。Root 提供真实 frozen source context，由未修改模板执行既有 root-title exclusion，localized Homepage title 不改变此语义。Learn 直接调用同一 frozen breadcrumb 模板的既有路径复用该 context，不改正文或 header/navigation architecture。
- Fail-closed 与 verifier：空或缺失 metadata、covered-but-missing Page、source tamper 均拒绝；未来 `/doc/` complete 后自动以 completed metadata/canonical href 接管。所有实际 `.SiteBreadcrumb` items/anchors 必须有非空规范化文本和 href，ordered targets 必须符合 current Go plan/Resolver；真实 browser 还检查可见文本/geometry。浏览器采样包括 source-declared breadcrumbs，覆盖 getting-started / handle-errors；无 breadcrumb 声明的 Security 不被强制添加。8 个 Go tests + 6 个 verifier tests PASS，涵盖 Docs/code/Security presentation 与 source-bound link 保留。
- 状态仍为 `open`；只有本轮 fresh automated Preview PASS 且维护者 HUMAN GATE 明确 PASS 才能 resolved。本轮没有执行正式内容扩展、翻译生成、Surface、activation、sealed build、publish 或 Production。
- 本轮 fresh Preview：`http://127.0.0.1:43243/`，实际监听服务 PID `631643`，ready=122、pending=0、blocked=0；启动日志 `/tmp/go-tour-breadcrumb-preview-20261019.log`。完整命令 `python3 scripts/verify-preview-browser.py http://127.0.0.1:43243/ zh-CN` exit=1，日志 `/tmp/go-tour-breadcrumb-acceptance-20261019.log`。
- 本轮 exact terminal blocker：`stage=browser check=rendered/interaction`，`/doc/security/fuzz/technical`、desktop width=1280，`Site content/assets/overflow` state 为 width=1265、overflow=0、anchors=False、images=True。这只证明 main anchor/fragment predicate 为 false，日志未指出具体 anchor，不推测其原因。按本轮明确停止条件保留 Preview/log，不继续诊断或修改该路径，不通过减少采样或放宽 anchor gate 绕过。
- 验证范围：完整 verifier 在进入 browser callback 前已正常返回全部 Site raw machine checks，因此 getting-started/handle-errors 的真实 HTTP breadcrumb 非空文本、href 与 exact coverage-target 检查已通过；其两个 Chrome viewport checks 排在该失败面之后，尚未执行完，不能宣称完整 browser 或 HUMAN PASS。14 个 targeted tests PASS；DI-008 继续 `open`，HUMAN_GATE 仍等待后续工程阻塞解除后的复验。
- 2026-10-19 localized Markdown heading correctness：对 `43243` 的真实 Chrome 只读诊断确认，Technical 的两条 `#input-minimization` 链接保留，但“输入最小化”的 ID 为 `heading-5`；Vulnerability hub 的两条 `#feedback` 链接也保留，但“反馈”的 ID 为 `heading-3`。显式 ID 的 managing-dependencies 对照页正常。根因是当前 Goldmark 根据本地化 heading text 重新生成 ID；非 ASCII 字符被其既有算法忽略，造成 source-bound fragment 与 DOM target 脱离，不是译文链接丢失。诊断证据：`/tmp/go-tour-heading-diagnosis.json`。
- 本次 renderer 修复：从 frozen manifest 校验通过的 Markdown source 与模板 data，使用同一 Goldmark parser/options 取得 canonical heading IDs；核对 source/completed heading count、level、AST 顺序/位置后，仅在派生 HTML 上恢复 source IDs。显式 ID、重复 heading suffix、Unicode/punctuation/inline markup 均由现有 parser 决定；结构不符、缺 source/heading 或与 static ID 冲突时 fail closed。HTML page、raw HTML / shell IDs、正式 localized bytes、frozen shell/template 不作改写，breadcrumb 修复保留。
- 本次 machine gate：完整 verifier 在浏览器采样之前，对所有 activated Site Pages 验证 source-bound 与 rendered main 的同文档 fragment target；Markdown 统计按实际 main anchor occurrences 计数（重复链接分别计数），external/cross-document fragment 不计入。浏览器原 `anchors` predicate 未修改。13 个 Go targeted tests、16 个 verifier targeted tests PASS，覆盖 auto/explicit/duplicate heading、模板 source data、Unicode、结构 mismatch/source tamper、HTML/static IDs、Docs presentation、breadcrumb、source-bound links/assets 与 browser errors。
- 启动回归修正：首次新 Preview 在 Compose 阶段拒绝 version-numbers，日志 `/tmp/go-tour-heading-preview-20261019.log`，未进入 ready。新增 collision guard 原要求 ID 唯一，但 frozen source 本来同时具有 `<a id="in-development">` 和同名 auto heading。现用同一 Goldmark 渲染 source 得到 exact ID multiplicity，保留这种 canonical representation，同时拒绝新增冲突；加入真实 frozen version-numbers 与显式前置 anchor fixture，13 个 Go targeted tests 重跑 PASS。没有修改 source anchor、译文或既有 HTML ID。
- 结构检查校正：第二次启动在 manage-connections 的 heading AST sibling index 拒绝合法 completed bytes，日志 `/tmp/go-tour-heading-preview-retry-20261019.log`，同样未 ready。既有 `validatePageStructure/pageBlockShape` 明确允许授权 soft-wrap 造成 prose Paragraph/TextBlock 数量变化；heading 对应路径现与该规则一致，忽略 prose/inline 节点，保留 heading count、level、结构顺序及非 prose 容器位置。增加实际 completed manage-connections 与全套 finalized non-templated Markdown fixture：37 页 PASS；这不是放宽正式 Page validation，未修改任何 lifecycle/parser authority 或 candidate。
- 最终 targeted tests：15 个 Go tests、16 个 Python verifier tests PASS，日志 `/tmp/go-tour-heading-targeted-go-20261019.log`、`/tmp/go-tour-heading-targeted-python-20261019.log`。37 页是 immutable non-templated Markdown fixture 数量，不能冒充本轮完整 activated HTTP fragment scan 的终态统计。
- 最终 fresh Preview：`http://127.0.0.1:41961/`，实际服务 PID `637224`（go-run parent `637174`），ready=122、pending=0、blocked=0；日志 `/tmp/go-tour-heading-preview-final-20261019.log`。Technical 的真实 Chrome desktop（1280×800）与 mobile（emulated 375×812）均 `anchors=True`，可见“输入最小化”对应 `<h2 id="input-minimization">`，原有两条正文链接及动态 TOC/permalink 均能找到 target；证据 `/tmp/go-tour-heading-runtime-proof.json`。
- 完整验收仍 FAILED：完整执行 `python3 scripts/verify-preview-browser.py http://127.0.0.1:41961/ zh-CN`，exit=1，日志 `/tmp/go-tour-heading-acceptance-20261019.log`。机器阶段报告 `stage=Site fragment integrity check=/doc/tutorial/fuzz expected=DOM target for #top actual=missing id=top`。这是新增全页 gate 实际发现的下一项 fragment target blocker；按本轮停止条件不继续诊断其 source/runtime authority，不新增例外、不放宽 browser anchor predicate。当前全页 HTTP fragment scan 未完成，不能报告所有 current Markdown Pages PASS 或 broken targets=0。保留 fresh Preview/log，DI-008 继续 `open`，`HUMAN_GATE=PENDING_RECHECK`；未执行任何 formal content expansion 或 lifecycle mutation。
- 2026-10-19 special-top verifier correction：Coordinator readonly 全量 evidence 为 Pages checked=49、same-document fragment links=134、special top without ID=1、broken non-top=0；唯一特殊链接是 frozen fuzz tutorial 的 `[Back to top](#top)`。该项经确认为 verifier false positive：HTML 原生 top-of-document fragment 不要求 `id="top"`，并按 ASCII case-insensitive 识别。仅同步修正 Python machine gate 与浏览器 anchors predicate；普通 fragment 仍必须 exact-match DOM ID。未改 renderer、runtime、source、candidate，未启动新 Preview。
- 本次验证：12 个直接相关 verifier tests PASS，涵盖 `#top`/`#TOP`/mixed case/percent-encoded top、ordinary exact/missing/wrong-case target、top-like 非特例以及真实 fuzz link fixture；Node 执行 actual browser predicate，既有 broken-fragment negative fixture 继续拒绝。复用 `41961` 完整执行 verifier，raw HTTP machine gate 通过（其仅 `.md`、raw main anchors 的计数为 40 页/85 links、broken targets=0；不与 Coordinator 的 49/134 审计口径混用），日志 `/tmp/go-tour-special-top-acceptance-20261019.log`。
- 新的 exact terminal blocker：完整命令 exit=1，`stage=browser check=rendered/interaction`，`/doc/security/fuzz/technical` mobile width=375，`overflow=7`（现有上限 2），`anchors=True`、`images=True`。按维护者要求停止，不进一步诊断或修复布局、不弱化 overflow gate；Preview `http://127.0.0.1:41961/` 与失败日志保留。本次 `#top` false positive 已修复，但完整 Preview 尚未 PASS，DI-008 保持 `open`，`HUMAN_GATE=PENDING_RECHECK`。
- 2026-10-19 mobile Docs presentation correction：直接采用 Coordinator 已完成的 375px readonly audit（51 routes、5 overflow、0 errors），不重复根因诊断：Technical 7px 为 breadcrumb；major-version 55px、managing-source 54px、vuln/database 4px、workspaces 16px 涉及长 inline technical tokens / heading，managing-source 同时需要 responsive DocTable。
- 本轮工程改动：在既有 generic Docs adapter 中加载单份 project-owned `docs_mobile.css`，仅窄屏 media 生效。Breadcrumb flex wrap/min-width 允许所有项正常换行；article prose/list/heading 继承 `overflow-wrap:anywhere`，inline code 可换行；pre/code 显式保持 `white-space:pre` 与 normal word-break，沿用 frozen horizontal scroll。Docs table 在元素自身范围内滚动，保留 cells/structure，不隐藏或截断内容、不使用 body/html overflow masking。Preview 与 publication 共用这一 renderer adapter；未改 frozen CSS/template、source、正式候选或 sparse authority。
- Targeted 验证：8 个 Go renderer / breadcrumb / heading tests PASS；3 个真实 Chrome layout tests PASS，覆盖 breadcrumb labels/hrefs/current indication、inline module path、nested list、long heading、normal inline styling、pre exact text/局部滚动、wide table 36 cells/局部滚动及 1280px desktop 样式与几何尺寸不变。Go 日志 `/tmp/go-tour-mobile-docs-targeted-go-20261019.log`。DI-008 仍为 `open`，只能由完整 automated Preview PASS 加维护者 HUMAN GATE PASS 核销。
- 本轮唯一 fresh Preview：`http://127.0.0.1:41711/`，服务 PID `642795`（go-run parent `642744`），ready=122、pending=0、blocked=0；启动日志 `/tmp/go-tour-mobile-docs-preview-20261019.log`。对 current Go browser plan 的全部 51 个 Site routes 运行真实 Chrome 375×812 deterministic audit：`ROUTES_CHECKED=51 OVERFLOW_ROUTE_COUNT=0 ERROR_COUNT=0`，原 5 个 overflow route 均为 0；逐页证据 `/tmp/go-tour-mobile-docs-overflow-audit-20261019.json`，过程日志 `/tmp/go-tour-mobile-docs-overflow-audit-20261019.log`。
- 完整新 Preview 验收 PASS：执行 `python3 scripts/verify-preview-browser.py http://127.0.0.1:41711/ zh-CN`，exit=0、`PREVIEW SURFACE ACCEPTANCE: PASS`，日志 `/tmp/go-tour-mobile-docs-acceptance-20261019.log`。未修改 verifier threshold、未裁切 body/html、未丢弃 breadcrumb/table 内容，frozen CSS/template 与 sparse authority 保存哈希仍相同。此结果仅为 automated gate PASS；`HUMAN_GATE=PENDING_RECHECK`，DI-008 继续 `open`，保留新 Preview 供维护者人工复验，正式内容扩展与 Production 仍未执行。

### 2026-10-24：structured-plan 剩余阻塞（维护者明确暂缓诊断）

manifest 同 entry exact-path 去重及 invocation-local verifiedBytes 64→512 MiB 调整已完成，直接 focused tests 与前三轮 correctness regression PASS。唯一一次真实只读 `structured-plan --locale zh-CN` 运行 480.164928 秒后按上限中断，stdout=0 bytes，stderr 仅 `signal: interrupt`，没有 lifecycle mutation。

此前采样观察到 12 次 historical Global 构造；维护者明确要求本轮不处理 independent context multiplicity，后续单独诊断。尚未证明它是唯一剩余热点，不据此增加共享 cache 或放宽任何验证。保留当前已通过 tests 的修改，不重跑或自动进入下一轮性能工作。

Evidence：`/tmp/manifest-exact-path-dedupe-20261024/`；停止点：`before-structured-generation-export`。

### DI-20261027-001：future-content conformance fixture 缺 Surface transport authority

- ID：`DI-20261027-001`
- 发现日期：`2026-10-27`
- 发现阶段/场景：bounded shared-QC performance optimization 的指定 targeted regression。
- 问题描述：`TestFutureContentSharedLifecycleConformance` 在 `future_content_conformance_test.go:286` 的 synthetic Surface revision export 被 current capability gate 拒绝：`Surface revision transport capability authority required`；TempDir 缺少 `data/surface-revision-transports/zh-CN/future-surface-reopen.json`。撤销本轮唯一 performance optimization、确认两个 production 文件恢复原始 bytes 后，单独运行同名 test 仍于相同位置失败。该 evidence 仅证明隔离 fixture 尚未满足当前 transport authority 合同，不表示真实 batch 060 或现有 Reviewer Bundle 无效。
- 暂缓原因：维护者明确要求本轮只做 bounded performance optimization；targeted regression 失败即撤销优化并停止，不扩展新的 correctness/compatibility engineering。
- 当前状态：`open`
- 后续处理/核销证据：尚未修复。优化前真实 Bundle readonly current-check 为 CURRENT、identity 保持 `d1eba49f0f74c6f1ae40e235c63db82537aa78911ccb7a66c83b7fd7a27c2e53`。失败及回退后独立复验分别见 `/tmp/shared-quality-check-performance-20261027/shared-qc-regressions.log`、`/tmp/shared-quality-check-performance-20261027/unoptimized-future-conformance.log`；不弱化 gate、不建立 synthetic authority 来掩盖此失败。
- 2026-10-29 同类 fixture 补充：`TestSurfaceReplacementMixedAggregateRevisionAndRecovery` 在 `page_replacement_test.go:173` 缺 `data/surface-revision-transports/zh-CN/replacement-successor.json`，于同一 transport capability gate 失败。用 Go overlay 将本轮 Global/Historical Page 两个新 cache 入口改为原 canonical 计算后，仍在同一位置、同一缺失路径失败；未更改当前 worktree，未伪造 transport authority。证据：`/tmp/formal-command-shared-environment-convergence-20261029/environment-workflow-final.log`、`canonical-control-fixture.log`、`canonical-control-overlay.json`。该补充不表示本轮重新执行了原 `TestFutureContentSharedLifecycleConformance`，也不表示真实 batch 062 无效；fixture 合同修复继续独立处理。

- 2026-11-03 test-fixture migration 补充：指定 i18n recovery pattern 在完成 explicit locale policy 的临时 fixture 迁移后，`TestSurfaceReviewFindingCanExplicitlyReopenFinalizedAWithoutChangingHistory` 仍在 `quality_check_results_test.go` 的 Surface revision Bundle 构造被同一 capability gate 拒绝；缺少 `data/surface-revision-transports/zh-CN/surface-fix-001.json`。未生成 capability、未绕过 gate，未改历史 QC/finalization 或真实 locale authority；该独立 transport fixture 问题仍 `open`。证据：`/tmp/language-audit-systematic-migration-i8fm8vbd/recovery-pass5.log`。

### DI-20261030-001：qc-023 current authorization 的三个 current validation scope blocker

- ID：`DI-20261030-001`
- 发现日期：`2026-10-30`
- 发现阶段/场景：修复 composed sibling Reviewer evidence reader 后，唯一一次 readonly `quality-check authorization --content-package learn-docs-v1 --locale zh-CN --snapshot-id learn-docs-v1-qc-023` 真实验收。
- 问题描述：historical integrity 已能完整解析 51 个原 A，命令进入 current assessment 后仍以 `current authorization blocked: affected=3 stale=0 unknown=0` 退出；48 compatible。index 27 `_content/doc/database/querying.md` 与 index 47 `_content/doc/security/vuln/database.md` 报 `current_validation_failed: 两批页面生成及机器校验全部完成前禁止独立质量审核`；index 48 `_content/doc/security/vuln/editor.md` 报 `current_validation_failed: explicit/finalized package scope mismatch`。三者另有 `changed_source_term_absent`，不能把此次失败解释为已证明的 glossary 语义影响或要求重新语言审核。具体底层 scope 路由 root cause 尚未诊断。
- 暂缓原因：用户明确要求真实 authorization 若仍失败即停在新 blocker，不连续猜修。本轮只保留已通过 targeted correctness 的 sibling historical reader 修复，不追加 current validation 修复或重跑命令；不得继续 package closure。
- 当前状态：`open`
- 后续处理/核销证据：尚未修复。完整 JSON、stderr、184.379627281 秒的终态与三个 Unit 的 exact IDs/reasons 保存在 `/tmp/qc023-composed-historical-reviewer-evidence-20261030/real-authorization.stdout.log`、`real-authorization.stderr.log`、`real-authorization-result.json`、`new-blocker.json`。qc-023 finalization 保持原 artifact，不重写为后续 Snapshot 或新审核 authority。
- 2026-10-30 scope diagnosis 补充：bounded exact-path instrumentation 证明三个 active candidate validator 都 PASS；27/47 在 `verifyAcceptedFrozenInputWithMachines → inheritedBridgeOutput → walkBridgeResponseLineage → readQualityCheckSnapshotWithCompatibility` 把 qc-019 的 49 页与 current 51 页比较；48 在 `prepareRetranslationOutput → revisionGenerationInputsCore → SelectFinalizedTranslationPackage → selectTranslationPackageScope` 比较 current `95afb981…/51` 与 historical `bcd457bf…/49`。共同 root cause 是 frozen response reconstruction 的 world mismatch，非语言质量或 glossary impact。候选修复通过相关 targeted suite 和五 Unit exact-path controls，但唯一一次真实 authorization 在完整历史 proof 阶段以 `reconciliation machine proof mismatch: accepted response lacks exact historical Unit/source` 失败（112.224976806 秒、stdout 空）；新 eager same-ID 前提不能覆盖所有合法历史分支。按用户 stop-on-next-blocker 要求，没有追加猜修/重跑；仅撤销本轮候选自己的 Go hunks，逐字节恢复本轮开始时的生产文件，所有先前已 PASS 的修复保留。问题继续 open，不能进入 package closure。候选 diff、前后 bounded path、全部日志和下一精确步骤保存在 `/tmp/qc023-current-validation-scope-diagnosis-20261030/`。
- 2026-10-30 response branch 补充：overlay 已定位真实触发为 `chatgpt-zh-CN-041` 的 `_content/doc/modules/major-version.md`：Surface replacement 将旧 `81e2cb69…` 变为 `f9605236…`，随后 machine reconciliation 映射至 `7eb8c6d1…`；`learn-docs-v1-qc-009` 不包含后两个 ID。此前 eager caller 在 lineage selector 前要求所有 record 都有 same-ID ancestor，错误覆盖了应使用自身 frozen input/raw codec 的 replacement record。新候选仅在已选中 exact same-ID ancestor 且存在 inherited response lineage 时解析历史 scope；显式 current-validation transport 独立解析其 frozen scope，current validator 继续使用 current package。隔离 regression fixture 用旧 caller overlay 稳定复现原错误，用新 caller 通过；真实 recorded replacement→machine proof 与 qc-023 完整 51-Unit historical graph 只读测试 PASS，六 Unit（1/3/27/32/47/48）精确路径及相关负例 PASS。没有运行真实 authorization，也不据此宣称全体 current eligibility 已通过；保留候选，问题继续 `open`，等待下一轮单独授权 native verification。证据与精确续接点：`/tmp/qc023-reconciliation-response-branch-diagnosis-20261030/`。


### DI-20261103-001：Go Example 旧成功路径 fixture 与 current comment completeness gate 不一致

- ID：`DI-20261103-001`
- 发现日期：`2026-11-03`
- 发现阶段/场景：`multilingual-language-audit-systematic-test-fixture-migration` 指定的 i18n recovery targeted pattern；相关失败在上一轮 test isolation recovery 中亦已出现。
- 问题描述：为隔离 fixture 安装合法 explicit `language_audit` policy，并保持原 source/candidate/assertions 后，`TestValidateGoExampleCandidateAllowsNumericConstantsShiftVerbTranslation`、`TestValidateGoExampleCandidateCorpusBaseline`、`TestValidateTranslationUnitCandidateDispatchesExample` 仍返回 `code-comment gate: non-English teaching-comment exact source carry`。前两者定位 `example:basics/numeric-constants.go` 的 teaching comments（分别 comment 2/3 与 2/3/4）；最后一项定位 `example:demo/main.go` 的 comment 1。这不是 missing-policy cascading failure；尚未据此判定 production implementation bug 或真实语言修改范围。
- 暂缓原因：维护者明确要求本轮只迁移共享 fixture authority；policy migration 后仍存在的 exact-source-carry semantic failure 另行独立 diagnosis，不修改 production contract、不 mock validator、不修改真实翻译。
- 当前状态：`open`
- 后续处理/核销证据：原断言与候选保持，真实 source/glossary 只复制至 `t.TempDir()` 后调用同一 production validator。逐 test 的完整原始错误及 locator 见 `/tmp/language-audit-systematic-migration-i8fm8vbd/code-comment-blockers.log`，完整指定 pattern 结果见同目录 `recovery-pass5.log`；后续须独立判断测试意图与 current completeness contract，不能通过 bypass 将其改为 PASS。
