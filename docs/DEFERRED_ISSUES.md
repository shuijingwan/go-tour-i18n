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
