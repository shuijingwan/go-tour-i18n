# Versionless Locale / Site v2 Workflow Foundation

Site shell Homepage Page 在 raw source 不变时永久改变 canonical protected representation，必须使用独立的 [Shell representation successor authority](SHELL_REPRESENTATION_SUCCESSOR.md)。默认 readonly、exact plan SHA/CAS、immutable receipt/current pointer 与 pending recovery 边界，不由 semantic reconciliation 或 QC reopen 授予 canonical mutation。

shared structured Document 使用独立的 [Shared structured representation successor](SHARED_STRUCTURED_REPRESENTATION_SUCCESSOR.md)，按已注册 package/asset 的 keyed head选择；Homepage Page TU family保持原 scope。新表示必须重新进入既有 required language stages，不能继承旧表示的语言 PASS。

shared structured representation 的 dependency-only 演进使用其独立 [re-attestation authority](SHARED_STRUCTURED_REPRESENTATION_REATTESTATION.md)，保持 base topology/head immutable；不能借 Homepage re-attestation 或直接忽略 parser SHA 恢复 currentness。

Learn/Docs current统一51 Page；`docs-entry-v1`仅保留 migration历史。原49+2 finalized语言成果经 exact membership/currentness proof组合成同一 current closure input，不创建新 A或新 Snapshot。source governance、历史 descriptor、readonly preflight与既有 closure的 additive合同见[Learn/Docs membership](LEARN_DOCS_MEMBERSHIP.md)。

所有正式翻译单元的执行以[共享翻译单元生命周期与五角色协作](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)为共同 authority。Tour、Learn/Docs 和未来内容包共用 `internal/i18n` 的成熟状态机；本文件仅定义页面适配和内容包后续边界，不授权平行 Generation/recovery/retry/QC/finalization。

本文是 V2-D 的 executable contract。v1/v2/v3 是 implementation/content-package/source-contract milestone，不是 locale language governance namespace。**Content package 可以版本化；Locale language system 不版本化。** package 划分 source、coverage、completion、activation 与 reuse，不划分 glossary、locale identity、术语治理、Generation/Reviewer session 或 provenance。

## Full locale Preview runtime composition

### 全站共享项目 chrome

所有正式 Site 与 Tour 页面共用 `tour.SharedChromeModel`、project metadata、About destination、唯一 languageRegistry、current locale、labels/URLs/order、footer metadata 和键盘交互语义。`ApplySharedChrome` 提供宿主 adapter：Site 使用 native go.dev header/footer visual language，Tour 使用 native Tour compact header/footer。共享 authority/model/behavior，不要求两者 HTML/CSS 或像素布局相同。组件位于语言 renderer 之后，不进入 TranslationUnit；future content packages 自动继承同一 composition，不逐 route 接入。

「About Project」是 all-page navigation，exact 目标 `/translation/`，文案复用 `header.about_project`；语言 trigger/label 复用 `site.current_language`、`site.language_versions`。Site 桌面 About 位于 existing go.dev navigation 后，language selector 最右，字号/行高/纵向对齐跟随原导航。Site 手机端复用 existing drawer，第一 Escape 关闭语言 details 并聚焦 summary，第二 Escape 才关闭 drawer 并聚焦 toggle；同一被消费按键的 keyup 不再重复关闭外层。Tour 保持 header compact controls 和独立的 course navigation，不新增 Site 式 hamburger/drawer；native language details 的 Escape 关闭及 summary focus 语义相同，无关 Escape/Tab 保持宿主行为。所有语言项左对齐且 label 单行，菜单按内容扩宽并限制在 viewport 内；窄屏使用接近 viewport 的可用宽度，必要时在菜单内部滚动。

`internal/tour/languages.go` 的 `languageRegistry` 是唯一语言导航 authority；model 复用 `languagesFor`/`currentLanguage`，每个 selector 使用完整 registry 的 presentation order、既有 EnglishName/Autonym label convention、exact registered URLs 和唯一 current marker。官方 English 使用其注册的官方 Go Tour URL，不推导 hostname、不维护第二份 locale map、不硬编码数量。browser verifier 对代表性 Site families 与 Tour welcome/list/course 页面验证同一个 selector/footer contract，而不是跳到某个历史页面统计旧 class。

global ProjectFooter 每页 exactly once。Site 保留 upstream go.dev footer，项目区位于 primary footer navigation columns 与 upstream/legal bottom row 之间，使用 upstream Container/padding 并默认左对齐；Tour 保留 native project footer 的居中布局。共享 model/metadata 不强制跨 renderer alignment，不强制 project anchors underline。Tour 原 project footer 由同一内容组件替换，不额外叠加 footer landmark。桌面保留两个明确 rows，手机允许换行但不合并语义分组：row 1 为 project identity/unofficial identity · Home · GitHub；row 2 为 copyright | locale/site legal metadata。组件为 server-rendered 静态 HTML，无 JavaScript 仍可阅读。

footer 文案复用 `footer.unofficial`、`footer.home`、`footer.github`。GitHub、owner、ICP 事实复用 `tour.Project`，copyright year 在共享 metadata 中固定为 2026，不随当前时间自动变化。Home root 从 caller 已验证的 production/site identity origin 取得，不硬编码某个 locale 的 hostname。legal metadata 为独立 locale/site policy：当前 zh-CN 显示 Project 中的 exact ICP 文本与普通外链，其他 locale 不自动继承；不注册本地 legal fallback 或 redirect。

Docs delivery 复用 frozen `_content/doc/default.tmpl` 所指定、已存在于 `_content/site.tmpl` 的 `doclayout`，给所有 Docs families 恢复 `.Doc.Article`、单一 completed title、TOC 与 metadata 控制的结构。工程 adapter 不改精确绑定的 shell reconstruction/renderer 文件或正式 authority；Article width、正文和 code/pre 背景继续来自 frozen stylesheet，不添加单页 CSS。Learn/Homepage 的独立 native layout 不套 Docs wrapper。内容边界与下一轮正式扩展见 [navigation frontier 与项目页 audit](SITE_V2_CONTENT_FRONTIER_AUDIT.md)。

`preview --locale` 在原 exact-current Tour Surface A gate 和 Tour projection 后，通过 `internal/site.LoadContent` + `Content.Compose` 组合 runtime。Site 页面只消费 current completed package 的 reconstructed bytes；正式 metadata、HTML / Markdown、Learn template / localized YAML 分别解析后渲染。`/tour/**` 的 canonical 页面和必要静态资源、lesson、Playground HTTP endpoints 保留原 Tour handler；其他页面、alias、true 404、canonical 与 sitemap 共用 Site Resolver，不注册 Tour SPA 全路径兜底。coverage navigation 与直接 HTTP 请求分开：未覆盖目的地链接 go.dev，未覆盖本地请求返回 404。

`Resolver.ResolveLink` 对 URL parser 已验证、具有非空 fragment 的纯 `#fragment` 引用保留 fragment-only 形式，沿用 Go URL 的转义规则，不补当前 route 或选择 coverage/host。这同时保留 frozen runtime 对 raw href 的 attribute-selector contract；`#` 空 fragment 仍走原 URL resolution，不作为实质 target。带 path/query 的链接继续按相对路径、alias 和 coverage 正常解析，跨文档 query/fragment 不丢失；此规则不修改 frozen HTML/JS、语言数据或 canonical coverage。

完整 Preview 的无 production analytics 约定同样适用于 shared Site shell，见 [Surface Preview 合同](LOCALE_SURFACE_REVIEW.md)。frozen `_content/site.tmpl` 自带的 GTM bootstrap / noscript fallback 不受 `TOUR_ANALYTICS` 控制；Go HTML template 会丢弃源注释，因此 Preview `DeliveryResources(..., true)` 在语言渲染后按完整标准 bootstrap、GTM fallback iframe 和 loader preconnect 的角色识别并移除；保留源标记时还验证成对块，不接受未知块内容。相似 hostname 或包含额外功能代码的 script 不被移除，仍受 browser gate 约束；不按请求 hostname、POST 或 `ERR_ABORTED` 放行。此 adapter 不改 frozen source、语言 evidence、Production delivery、`TOUR_ANALYTICS` 注入合同或独立 advertising policy；browser verifier 继续拒绝所有非既定 `/socket` 探测的网络失败。

Preview sitemap 的 exact set 来自 current `locales/<locale>/content-scope.json`，runtime 先验证其 completion evidence。统一 membership 的三个 package 全部 current complete 时为 158 条（Tour 105、Learn/Docs 51、shell 2）；full Homepage/shared chrome adoption 不扩大 route coverage。旧 shell completion 在 adoption 后仅为 historical predecessor；缺新 Homepage A-only + 四份 structured + translation exact carry 时，runtime 在监听前 fail closed，不能发布简化 home.html。

### 共享 Playground UI bootstrap

共享 Playground 消息依赖：加载项目拥有的 `/js/playground.js` 的 Site 页面，均在该 script 之前由 delivery adapter 插入唯一 `site-playground-ui` bootstrap。消息来自 `tour.UIMessagesBootstrap`，直接复用 Tour 的 `jsI18nBootstrap`、完整 `jsI18nKeys` 和严格 `ui.Load`；按 renderer 输出的 exact `html lang` 选 catalog，不新增子集 schema、fallback 或语言值。现有 shared shell 有意全局加载 Playground，因此其静态页面也满足同一依赖；不加载该 runtime 的独立静态页面不增加状态。JSON 保留 canonical HTML-safe escaping；缺 catalog/key、冲突、重复、错误顺序或 stale bootstrap fail closed。Preview 与 Production delivery 使用同一路径，frozen source 和 Tour 原有 script bytes 保持不变。

### Site 下载 runtime dependency

HTML navigation 和 JS 内部 URL 都必须遵循 coverage authority。`DeliveryResources` 将 frozen `js/site.js`、`doc/download.js` 的 script 引用投影到同源 `/_/site-runtime/<source>`；此 allowlist 只提供已验证资源 inventory 的派生 JavaScript，不提供上游代理、页面或 canonical coverage。Preview 与 sealed release handler 共用 `projectRuntimeScript`：单引号、双引号和 template literal 中的 `/dl/?mode=json` 使用当前 Resolver 的 `/dl/` authority，未覆盖时使用 canonical go.dev，正式覆盖并提供该 endpoint 时允许本地解析。动态文件下载 URL 仍使用 canonical `https://go.dev/dl/`：仅覆盖 `/dl/` 不代表覆盖任意文件子路径。query 与动态文件名 suffix 保持原样，未知 script / authority fail closed。

同一 derived `site.js` 还校验下载 DOM 的两种合法形状：`.js-latestGoVersion` 存在但三个平台 anchor 全部不存在时，跳过平台 href 赋值，保留 generic `.js-downloadBtn` 和独立的 `setVersionSpans()`；三个平台 anchor 齐全时执行原平台 URL 逻辑；部分平台节点缺失则抛出明确 runtime error，不作为 generic 页面静默接受。该 guard 仅在完整 frozen `setDownloadLinks` 函数结构唯一且精确匹配时插入，结构变化或重复定义使 runtime delivery fail closed；不按 locale、route 或 SHA 打补丁，不修改 frozen source。

该 adapter 不修改 frozen `_content`、TranslationUnit、shared assets 原始 bytes 或语言 evidence，不 patch 全局 fetch。`GET /dl/` 和 `GET /dl/?mode=json` 在未覆盖时仍返回真实 404；正常下载导航及 runtime fetch/file links 使用 canonical go.dev。跨域 JSON 使用浏览器默认 CORS fetch，不携带跨域 credentials；2026-10-19 在真实 locale Preview origin 上验证 `https://go.dev/dl/?mode=json` 返回 HTTP 200、`Access-Control-Allow-Origin: *`，Chrome `Response.type=cors` 且 JSON 可读。CORS/network 失败仍由原 browser acceptance fail closed，不设置 verifier whitelist。

现有 build/publish/Production adapter 已使用同一 activated coverage与completed payload，seal exact identity后进入既有runtime/部署路径。production、canonical/sitemap、browser plan、assetspolicy与searchcloseout共用currentcontract；详细machinecontract见 [Production runbook](PRODUCTION_RUNBOOK.md#current-site-release-contract)。这不授权实际publish、deploy或formalbrowseracceptance。

2026-10-14 baseline [release/deployment审计](SITE_V2_RELEASE_PATH_AUDIT.md)的FAILED evidence保持immutable。后续12项精确修复及baseline-inventory delta current-check分别记录于 `SITE_V2_RELEASE_PATH_REMEDIATION.json`、`RELEASE_EXECUTABLE_INVENTORY_DELTA.json`；current工程closure以该delta为准，不重扫baseline entrypoints。Preview browser audit为trustedcarry，未重新审计。

## Source 与统一 terminology authority

pre-V2-C1 正式 English authority 是主动冻结的 `golang/website@db076098077c07d3cef1b85a2cf56ff52777f587`，不是 latest upstream。campaign / Preview / Production 不被 deferred C1 阻塞，不增加联网 drift gate。四个 tracked issue 达到维护者预期且显式重开后，仅一次 C1 sync；此处不执行 sync。

原 `data/site-content-scope.json` 保留 historical package closure；历史 repo-local foundation 保留 49 Page + 7 data = 56 Learn/Docs localizable documents；current membership successor 另把两个已 adopted Docs entry Page 合并为 51 Page + 7 data。shell current adoption 投影 `_content/index.md` 的一个 Homepage Page TU、四份 shared structured source、既有 translation.html，canonical 仍仅 `/` 与 `/translation/`。home.html 仅 historical predecessor。详细 authority 与 CAS 见 [Shell current adoption](SITE_SHELL_SUCCESSOR_AUTHORITY.md#current-adoption六份原子-closure-与-jit)。

唯一 locale glossary：`locales/<locale>/glossary.yaml`。mandatory/preferred/forbidden/keep 四类不变，禁止 Tour/Docs/Learn/v2/v3/core/extension/shadow glossary。

Unified Glossary Source Corpus authority 是 `data/glossary-source-corpus.json`，contributors 同时是 shared structured source registry。foundation v1 曾投影 103 Tour Page、19 eligible Example、113 UI values、7 article headers、103 SEO descriptions、两个简化 shell source、7 Learn YAML、49 Learn/Docs Page（当时 403 contributors/3,004 contexts）。随后 v2 terminology discovery 与 shell corpus successor 已正式应用；当前 exact membership/count/identity 以 JSON 和完整 Unified Review 为准，包含 full Homepage 与四份 shared structured source，retire old home contributor。binary/machine-only代码不作为语言文本；protected technical context 保留只读。future surface additive 注册并使用正式 parser，不另建 glossary generation。

仓库已具备但当前尚未应用 terminology-discovery lifecycle。`site-content terminology-authority --website-root <clean-pinned-checkout> --go-source-archive <pinned-archive>` 默认只做离线 dry-run；显式 `--apply --prior-sha256 <current-corpus-file-sha256>` 才以 CAS 创建/刷新 `data/terminology-discovery-sources.json`、`.zip` 并把 unified corpus 升为 `go-learning/glossary-source-corpus/v2`。已有 discovery manifest 再刷新还必须提供 `--prior-terminology-sha256 <current-manifest-file-sha256>`。普通 `site-content terminology-check` 完全离线：未应用时验证合法 `not-applied` v1 状态；应用后从 snapshot 重建 projection 并对 manifest/parser/corpus drift fail closed。它不联网、不写 locale evidence，也不执行 V2-C1。

v2 contributor 显式使用 `purpose=active-content|terminology-discovery`。现有 Tour/UI/article/SEO/shell/Learn/Docs 全部是 active-content；冻结 website / Go source contributor 是 terminology-discovery、`structured=false`，不得形成 package、coverage、route、completion、runtime 或 locale structured output。raw manifest/snapshot identity 证明冻结输入 bytes；unified corpus identity只绑定 deterministic language projection，因此 machine-only raw bytes变化不会冒充 projected terminology变化。两类 purpose 仍共同进入唯一 corpus identity，不存在 discovery-only glossary、Review receipt 或 compatibility shortcut。

Corpus identity只绑定 deterministic contributor language/context projection；raw source file hashes另外负责机械 current-check，不把 image/code-only raw bytes变化等同于 terminology变化。`site-content corpus-check` 重建实际projection并比较；unknown parser/metadata/contributor、wrong hash、UTF-8、duplicate/unknown/trailing JSON、traversal/symlink fail closed。没有 AI术语抽取阶段。

## 一次 unified Glossary Generation / Review

新 locale：一次完整 Glossary Generation；existing locale：保留既有 glossary、一次 Unified Glossary Refresh。都输出完整 `glossary.yaml`，允许 byte-identical；不输出surface extension fragment。future sync跨Tour/Docs/future scope合成一次 corpus delta、最多一次 locale refresh。

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export --locale <locale> --task glossary --output /tmp/<locale>-glossary-generation.zip
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-check --bundle /tmp/<locale>-glossary-generation.zip
go run -mod=readonly ./cmd/tour-i18n glossary-review reviewer-bundle --locale <locale> --output /tmp/<locale>-glossary-reviewer.zip
go run -mod=readonly ./cmd/tour-i18n glossary-review reviewer-bundle-check --locale <locale> --bundle /tmp/<locale>-glossary-reviewer.zip
go run -mod=readonly ./cmd/tour-i18n glossary-review record --locale <locale> --review-id <id> --reviewer <independent-session> --generation-session <generation-session> --decision passed --bundle /tmp/<locale>-glossary-reviewer.zip
go run -mod=readonly ./cmd/tour-i18n glossary-review check --locale <locale> --coverage unified
```

transport `go-learning/unified-glossary-bundle/v1` 绑定完整 corpus、完整 glossary/skeleton、locale identity、terminology/review authority、exact outputs、input identity；Reviewer role没有replacement输出。新 receipt `go-learning/unified-glossary-review/v1` 保存完整 glossary path/SHA、corpus identity、archived Reviewer ZIP SHA/input identity、review ID、Reviewer/Generation session、decision/rubric/findings与self identity。ZIP与review evidence immutable、hash-bound。

old v1 receipt与fixed legacy coverage继续保持 Tour historical/current coverage。exact current unified Review优先；旧receipt不参与unified ambiguity。多个current unified receipts fail closed。无unified时旧Tour completion仍可合法验证；新语言工作要求current unified PASS。glossary任意byte change或corpus identity change均使旧unified Review stale；byte-identical refresh也不能拿Tour-only receipt冒充new corpus coverage。没有 corpus compatibility shortcut。

terminology-discovery authority 一旦在安全 workflow boundary 显式应用，哪怕 `glossary.yaml` bytes 最终不变，新 corpus identity 仍使旧 unified receipt 自然 stale，必须由独立 Reviewer 对单一完整 glossary 执行 full Unified Review。discovery corpus只降低未来 glossary churn 的预期，不降低审核范围或 A-only downstream gates。若 future package 没有 governance-significant terminology，refresh 可 byte-identical 但仍 full Review；若存在新术语/collision，则 Generation 刷新单一完整 glossary，full Review PASS 后才执行 compatibility assessment 并只 reopen真实 affected scope。不得按 Page 数量机械扩 glossary。

glossary改变之前归档原bytes；new full unified PASS之后，V2-B archived old/new → semantic delta → exact affected scope → immutable compatibility lineage。Docs-only term不会自动重翻旧Tour Page；shared core term只reopen真实affected TU/asset。unknown impact fail closed；历史A、Snapshot、glossary Review、Surface与provenance不改写。

## Unified Structured Assets

transport projection、32 MiB gate、exact archive references 与 export/check first-real-use 合同见 [Structured Generation transport](STRUCTURED_GENERATION_TRANSPORT.md)。该合同不改变语言语义、carry/Expected 或 lifecycle。

contract `go-learning/locale-structured-assets/v1` 使用 corpus contributor 的 structured 属性。adoption 后为 14 个完整资产：Tour UI、article metadata、四份 shell shared structured、translation body、7 Learn YAML；Homepage body 是独立的一个 shared Page TU，不混入 structured outputs。existing locale 沿用真实 accepted baseline、current source/target/glossary assessment；Expected 仅当前正式调度的 exact generation-required outputs。JIT 先 Homepage TU，再 current structured composite Bundle。

```sh
go run -mod=readonly ./cmd/tour-i18n site-content structured-plan --locale <locale>
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export --locale <locale> --task locale-assets --output /tmp/<locale>-structured.zip
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-check --bundle /tmp/<locale>-structured.zip
go run -mod=readonly ./cmd/tour-i18n site-content structured-import --bundle <repo-relative.zip> --input <repo-relative.json> --provider <chatgpt|codex> --model gpt-5.6-sol-high --session <generation-session> --generated-at <UTC>
```

模型只输出plan.Expected exact set，不重新输出carry。Tour UI/metadata使用原schema validator；shell/YAML输出完整slot target set，受parser重建，不允许任意目录绕过保护。composite evidence记录一个真实Generation session/provenance，subassets仍保持source/target/package/validator边界。structured asset不另建逐-TU QC；finding回原Generation → exact replacement → validation → affected Surface re-review。

### Published StructuredGeneration current machine check

`site-content structured-generation-check` 直接调用既有 `CheckStructuredGeneration(root, ref)`，验证一个已经 publication 的 StructuredGeneration 及其 current downstream authority。它不同于只接受 ZIP 的 `structured-check --bundle`，也不是 TU/Page `generation-check`。checker 的 currentness、historical integrity、timestamp recovery/ancestor bridge、source/representation/dependency 与 validation 语义不变。

| 字段 | 合同 |
| --- | --- |
| COMMAND | `site-content structured-generation-check` |
| OPERATION | current downstream validation of one published StructuredGeneration |
| INPUT_KIND | StructuredGeneration Reference：exact file path + SHA-256 |
| INPUT_PATH_CONTRACT | repository-relative regular file；`data/locale-language/<locale>/structured-generation/<identity>.json` 合法；absolute（即使 repo 内）、traversal、symlink 非法 |
| REQUIRED_FLAGS | `--input`、`--input-sha256` |
| FORBIDDEN_FLAGS | 不接受 bundle/output/provider/model/session/generated-at/apply/record/CAS/skip-currentness overrides 或 positional args |
| STDOUT_KIND | bounded deterministic `go-learning/structured-generation-check/v1` JSON：status=PASS、generation Reference、locale、plan identity、output_count、validated_file_count、mutation=none；不输出 reconstructed files |
| MUTATION_EXPECTATION | none；成功和失败都不创建 authority/receipt、archive、head、closure 或其它 artifact |
| IDEMPOTENCE | deterministic readonly；每次执行原 checker，不消费旧 PASS cache |
| CURRENTNESS_CHECK | exact `CheckStructuredGeneration` semantics；wrapper 不替代或跳过任何 reader/gate |
| PRIOR_OR_CAS | none |
| SUCCESS_STOP | machine validation only；before package closure，不自动进入 lifecycle |
| FAILURE_STOP | 保留 Generation/authority/outputs；失败非零、无 PASS stdout，不继续 lifecycle |
| RECOVERY_REPLAY | 修复实际 currentness/integrity blocker 后，重新核对 exact Reference 并 rerun check；不修改 evidence 迎合 checker |

普通合法 RFC3339 UTC Generation 支持原样检查。只有 checker 已成功验证 `timestamp_recovery` authority 时，summary 才输出该 Reference 与 `original_generated_at_status=unavailable`；不能凭空 `generated_at` 判定合法，也不把 unavailable 转成时间。普通 Generation 省略这两个 optional fields。

这是 new formal CLI。第一次真实使用前，Coordinator 必须按 [first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3) 重新核对 current canonical docs → implementation → direct CLI/contract tests。工程测试只在 isolated fixture 执行；工程完成停止于 `structured-generation-check-first-real-use-gate`，不代跑真实 check。

### Structured Generation stale-Bundle carry-forward

原 Generation `generated_at` 从未记录且无法可信恢复时，stale-Bundle recovery 和普通 import 均不能填假时间；只可走 [Structured timestamp-loss authority](STRUCTURED_TIMESTAMP_RECOVERY.md) 的专用 readonly preflight / explicit publication。该 metadata lineage 不改变语言输入、正常 RFC3339 UTC provenance 或任何 currentness/validation gate，不与 carry-forward 混合使用，不代表新 Generation 或语言通过。

若 raw source 不变，但 canonical parser 新增正式 localizable semantic inputs，不能 carry-forward。已完成 asset 只能通过 [Structured semantic-input-expansion reopen](STRUCTURED_INPUT_REOPEN.md) 的只读 preflight 和维护者显式 `--apply`，再调度完整 current asset Generation；无 valid receipt 仍保持 stale。

正常 structured import 仍要求 Bundle current-check 与 output 绑定实际输入 identity。首次模型输出已完成、但旧 Bundle 后来 stale 时，新 Bundle + 改写原 JSON identity 会伪造 model input；普通 TU recovery 不能表达 structured scope。这一证据缺口通过既有 `StructuredGeneration` 的可选 `carry_forward` successor edge 表达，不新增平行 Generation/retry/Review 状态机，也不作为语言 acceptance。

`site-content structured-recover` 默认只读 preflight；正常 preflight/apply 都由 Maintainer Local terminal 执行。必须提供原 Bundle/output 的可信 SHA 和完整原 provider/model/session/generated_at。实现验证旧 Bundle transport inventory、self identity、Plan self identity、StrictJSON 与原 output → old Bundle identity；重导并 current-check 当前 canonical Bundle，比较完整 Plan（包括 Expected、source/Document/slot/protection/target path、Review/glossary/corpus/assessment/finding），并逐字节比较全部非 execution transport members。Plan identity 不单独作为兼容证明。最后原 output 通过 current canonical reconstruction/schema/glossary validation；任何真实漂移、未知 legacy authority 或 validator failure 均 fail closed，需要真实 re-Generation 或进一步工程诊断，不能强行 carry。

`structured-language/v1` 的语义依赖是 `STRUCTURED_GENERATION_SEMANTICS.md`、`TRANSLATION_TERMINOLOGY.md` 与 `TERMINOLOGY_GUIDE.md`，连同 source、glossary、locale、corpus、完整 independent Unified Review、predecessor/finding 等正式输入决定 currentness。原 PackageAuthorities 完整文档继续随 Bundle 运输、受 inventory/hash 保护，但其 execution/tooling 说明不再是额外 language-semantic authority；语言规则必须在上述正式语义 contract 修改，不能通过执行手册覆盖。语义文件任意 byte drift 仍 stale。未分类 legacy Bundle 只接受实现中经审计的 exact legacy authority SHA → exact new semantic authority SHA 桥接；未知版本不能凭 `.md` 或 docs-only 声明自动兼容。

apply 将原 Bundle、原 output JSON 原始字节与 current checked Bundle 保存为既有 content-addressed immutable archives，再保存新的 StructuredGeneration successor。其 `bundle` 与 `output.bundle_identity_sha256` 仍指真实原模型输入；`generation` 保留原 provenance，绝不使用 recovery 时间；`carry_forward` 绑定原 output archive/current Bundle/authority edge 与 reconstructed target hashes。只有最后的 generation evidence 保存后才宣告 applied；partial content-addressed archives 不等于正式 import 成功，禁止覆盖历史 failure。closure/Surface 沿用既有 `--generation` 引用和 readers，重新验证原字节、冻结 lineage 与当前 source/semantic authority/glossary/validators。historical integrity 不要求今天 Plan 仍待生成，不改写旧 judgment；本 recovery 不执行 closure、Surface 或 activation。

```sh
go run -mod=readonly ./cmd/tour-i18n site-content structured-recover \
  --bundle <original-bundle-path> --input <original-output-path> \
  --bundle-sha256 <trusted-original-bundle-sha256> \
  --output-sha256 <trusted-original-output-sha256> \
  --provider <original-provider> --model <original-model> \
  --session <original-generation-session> --generated-at <original-generated-at>
# 同一可信输入 preflight PASS 后，由 Local terminal 加 --apply 保存 successor。
```

### Structured historical baseline 与当前精确 delta

**historical completion ≠ current eligibility**。旧 completion 先通过冻结输入下的 historical integrity：generation 与 Bundle hash/input identity、完整旧 Unified Review 及其 Reviewer archive、旧 glossary、真实 provenance/session、精确 source/parser/structure、outputs/reconstruction/validator，以及 closure 的完整 localized inventory/self identity。此步骤不要求旧 glossary 今天仍 compatible，不修改旧 evidence，不 mass migrate。

正常路径为 **historical baseline → per-asset current assessment → exact glossary-affected Generation delta → current closure → delta-aware Integrated Surface review**。`structured-plan` 对 completed package 逐 asset 绑定 baseline closure/generation/Surface、accepted target、旧/当前 glossary、当前完整 Unified Review、精确 source/input 与 assessment self identity；使用共享 `ParsedGlossaryDelta` / `ParsedGlossaryImpact` 和当前 reconstruction/validation，返回 compatible（carry）、affected（generation-required）、stale 或 unknown。stale/unknown 禁止 Bundle 导出与 closure；未知或尚只有 source foundation、缺 lifecycle reconstruction/validator 的 parser 不能成为 Generation 任务。

completed baseline 存在 affected 时，`Expected` 只包含其 exact affected outputs；其他未启动 package 的 initial outputs 保留 context-only，不混入本次 reopening invocation。导出的既有 Structured Bundle 包含 hash-bound baseline evidence/Generation archive、完整当前 glossary/Review、当前 source 与 predecessor slots/旧 glossary。current checker 必须重建相同 Bundle；import 仍绑定实际读取的 input identity、真实 Generation provenance、精确 outputs 和现有 validator。glossary delta 使用 assessment authority，不能伪造 Surface finding。

`package-closure` 根据 canonical assessment 选择 frozen structured baseline，再合并 current exact deltas；不要求每代 successor 平铺重传 ancestry。CLI、stdout、wire fields 与 baseline 语义统一见 [Package closure canonical command contract](#package-closure-canonical-command-contract)，affected/compatible Surface replacement 的窄授权见 [Structured Surface supersession owner resolution](#structured-surface-supersession-owner-resolution)。原 Page A-only finalization/current authorization 独立消费，structured glossary delta 不 reopen、regenerate 或重新 rating 无变化 TU。

Integrated Surface 根据同一 baseline/delta proof 处理范围：delta structured asset 进入 review-required（即使 bytes 恰好相同也要审核）；原 exact accepted target/source 及 prior Surface 支持的 unchanged structured assets、unchanged finalized Page 进入 carry。发生真实变化的 Page 仍消费共享 QC/authorization，不由 structured assessment 产生语言 A。global/Tour planner 不要求其他 package 的 historical structured completion 先通过 current closure；各 package 的 current eligibility 独立判定。

Course SEO是Page-derived consumer，canonical English description + final ready Page + glossary决定scope；不强行并入initial structured generation。existing current Tour SEO carry；本轮没有正式Learn/Docs canonical SEO source authority，不生成虚假SEO contract。future canonical consumer接同一locale/provenance/compatibility framework。

## Formal command contracts 与字段语义

以下集中说明既有 command/schema，不新增 lifecycle、schema、state 或 acceptance gate。命令名、flag 名和 Go 字段名不能代替 operation、INPUT_KIND 与 JSON wire fields。实现入口为 [`site_content.go`](../cmd/tour-i18n/site_content.go)、[`package_closure.go`](../internal/sitecontent/package_closure.go)、[`structured_lifecycle.go`](../internal/sitecontent/structured_lifecycle.go)、[`structured.go`](../internal/sitecontent/structured.go) 与 [`integrated_surface.go`](../internal/sitecontent/integrated_surface.go)。下列语法示例不构成执行授权；deterministic workflow 仍由 Maintainer Local terminal 执行。

### Package closure canonical command contract

当前含 shared Page 的 package 使用：

```text
site-content package-closure --locale <locale> --package <package> \
  --input <repository-relative shared Page finalization> \
  --generation <repository-relative StructuredGeneration> \
  [--generation <repository-relative StructuredGeneration>...]
```

| 字段 | 合同 |
| --- | --- |
| COMMAND / OPERATION | `site-content package-closure`；完整重建并发布 immutable package language closure |
| INPUT_KIND | `--input` 是 shared Page finalization；repeatable `--generation` 是已发布 StructuredGeneration References/current exact deltas，不是 arbitrary generation bundle、output JSON 或 finding input |
| INPUT_PATH_CONTRACT | 所有输入为 repository-relative regular file；例如 `data/quality-check-snapshots/<locale>/<snapshot>/finalization.json` 与 `data/locale-language/<locale>/structured-generation/<identity>.json`。absolute path（即便位于 repository 内）、traversal、symlink 均非法。CLI 从文件 path 计算 `{path, sha256}` Reference，不接收 caller 自造 Reference JSON 作为 flag 值 |
| REQUIRED_FLAGS / MODES | 当前 shared Page 模式指定 `--locale`、`--package`、`--input`；structured delta 使用 repeatable `--generation`。已有 composition mode 用有序 repeatable `--page-finalization` 代替 `--input`，两者互斥；TU-only、零新 structured delta 和 legacy historical readers 沿用各自既有 inventory gate，不能据本例伪造必需的 Generation |
| FORBIDDEN_FLAGS | 不接受 `--finding`、`--bundle`、`--output`、`--apply`、`--record`、provider/model/session、skip-currentness 或 prior/CAS override；finding authority 必须来自 StructuredGeneration 自身绑定的 Bundle |
| STDOUT_KIND | 成功 stdout 是 **JSON string containing the repository-relative newly/currently resolved closure path**，不是 result wrapper/object；没有顶层 `path`、`status` 或 `closure` 字段 |
| MUTATION_EXPECTATION / IDEMPOTENCE | 成功只发布 `data/site-workflow/<locale>/<package>/closure/<identity>.json`；exact replay 在完整验证后解析到同一 immutable artifact，不覆盖已有 bytes。返回 path 本身不证明本次新写了文件，mutation 按完整 inventory/provenance 判断 |
| CURRENTNESS_CHECK | 完整 Page A-only/current authorization、source/package/corpus/Unified Review、structured historical integrity/current reconstruction、exact deltas/inventory 及 dependency mutation 检查；保存后的 current-check 使用 closure 自己冻结的 baseline，见下文 |
| PRIOR_OR_CAS | 无 caller prior/CAS override；沿用 immutable/no-overwrite 与 invocation 内 dependency bytes/directory mutation 检查 |
| SUCCESS_STOP / FAILURE_STOP | 成功止于 `surface-scope` 之前；失败停止 downstream。publication、stdout capture 与 wrapper postcondition 分别诊断，不能凭 wrapper FAILED 推断无 mutation |
| RECOVERY_REPLAY | 保留原 artifact/终态/inventory；仅在 provenance 与 currentness 已确认后决定可否重试，不删除 closure 或盲重放。统一按 [Wrapper / verifier 与 mutation provenance](WORKFLOW_HANDOFF.md#wrapper--verifier-与-mutation-provenance) 处理 |

stdout 的形状示例（仅表示 JSON string 类型）：

```json
"data/site-workflow/<locale>/<package>/closure/<identity>.json"
```

Closure artifact 的 schema 为 `go-learning/package-language-closure/v1`。以下是实际 JSON field names；Go 的 `Structured` 字段在线上序列化为 `structured_generation`，不得写成 `structured`。

| JSON field | 含义 |
| --- | --- |
| `schema` | `go-learning/package-language-closure/v1` |
| `locale` / `package` | 此 closure 的 exact locale/package scope |
| `package_sha256` | 绑定的 package authority identity |
| `page_finalization` | shared Page finalization Reference；当前 shared Page 模式必需，已有 composition/historical mode 可省略 |
| `structured_generation` | StructuredGeneration Reference 数组；successor 运输 frozen baseline 上的 current exact delta set，历史 flat baseline 引用仍可按既有 reader 读取 |
| `structured_baseline_closure` | 可选的 exact frozen structured baseline Reference，提供 accepted owners/files；不是“上一 successor”关系字段 |
| `localized_files` | 完整 reconstructed localized inventory 的 `{path, sha256}` 数组，包含继承与 replacement outputs |
| `identity_sha256` | closure 的可重算 self identity，区别于整个 closure 文件原始 bytes 的 SHA-256 |
| `package_membership_successor` / `page_finalization_composition` | 既有可选 membership/composition fields；不以简化 field 清单拒绝合法 composition artifact，仍由 canonical reader 检查 |

#### Frozen structured baseline 与 successor delta

`structured_baseline_closure` 是 canonical closure reconstruction 选择并冻结的 **frozen structured baseline**。successor transports the current exact delta set over the frozen baseline; it does not serialize successor ancestry as baseline chaining. 因此 successor 不必把上一 successor closure 设成新的 baseline，也不能由 wrapper 根据“上一次输出”改写它。若 canonical assessment 实际选择了新的 frozen baseline，则绑定该 exact Reference；不能反过来推导所有 successor 永远共用同一个 baseline。

以下 zh-CN `site-v2-shell` 是 concrete example，不是跨 locale 的 permanent rule；简称表示对应 closure identity。本例选中的 frozen structured baseline 为 [1f5e9f…](../data/site-workflow/zh-CN/site-v2-shell/closure/1f5e9f0fb28b2098c1fff90778f652f9541cfc6ff4d8c75e72ec2093f70dc926.json)，两个 successors 如下：

| Closure | `structured_baseline_closure` | `structured_generation` |
| --- | --- | --- |
| [68cd5c…](../data/site-workflow/zh-CN/site-v2-shell/closure/68cd5c0142c1a73e06375d45dce5e45a2f8441f6d682bdf7db58b7b4336d4e42.json) | `1f5e9f…` | `6a30…`：translation.html ordinary affected delta |
| [a8bfb1…](../data/site-workflow/zh-CN/site-v2-shell/closure/a8bfb11fdbb8b02cb9133814893c8f8cc67cf869b3db05c41c108c2deb2e577a.json) | `1f5e9f…` | `6a30…` + `ef885…`：translation.html ordinary delta + site.tmpl Surface representation successor delta |

`a8bfb1…` 绑定 shared Page finalization `site-v2-shell-qc-004`，文件 SHA-256 为 `134a892e0fcf7edcf0b41fd557c623c427a4d99912e4d6296df3409f1092e35e`。它的 baseline 为 `1f5e9f…` 而非 `68cd5c…` 是合法结果；current exact deltas 共同重建当前完整 structured inventory，unaffected outputs 继承 exact bytes。已 hash-bound 的 baseline 递归提供 historical generations/owners/files，不要求在 delta 列表平铺 ancestry。

保存后的 `CheckPackageClosure` 固定读取该 closure 自己记录的 exact `structured_baseline_closure` 并重验整个合并，不信任旧 PASS；不把已 activation 的自身、前一个 successor 或后续 completion 自动选成 predecessor。missing/duplicate/extra/conflicting deltas、wrong package/baseline/glossary/input/provenance、非完整 inventory 仍 fail closed。

### Structured Surface supersession owner resolution

`affected` 并不自动意味着 glossary/source-only delta。唯一的 affected Surface replacement 路径是 current assessment 明确绑定 shared structured representation successor（`RepresentationSuccessor != nil`，reason 为 `explicit shared structured representation successor requires fresh language generation`）：唯一 replacement 的 current glossary 与 Unified Review 必须 exact，`structuredSupersedes` 必须通过合法 failed Integrated Surface structured-language finding 证明它 supersede 此 baseline 的 exact accepted Generation owner（含 inherited owner），且完整 current StructuredGeneration validation 必须 PASS，replacement target 必须 exact 对应该 affected asset。closure 消费该资产的 current reconstructed output，未受影响输出继续继承。该授权不是 glossary/source delta、generic stale recovery 或 `surface-integration` correction；没有明确 representation successor 的 affected asset 仍禁止使用 Surface finding，ordinary delta 继续要求 exact assessment authority。compatible 默认 carry，既有 `structuredSupersedes` exact Surface supersession 分支继续有效；compatible 任意 replacement 仍拒绝。

Integrated Surface scope 可以同时包含多个 package closures。`structuredSupersedes` 保持 exact finding/path/class、receipt/scope identity、Reviewer evidence hash 和 independent Reviewer session authority；所有 scope closure References 仍先验证，包括排列在匹配 closure 之后的 unrelated package。只有 finding owner 的 exact package closure（`closure.Package == finding owner.Package`）参与 direct/inherited Generation owner resolution；其它已验证 package closures 不参与 owner comparison，排列顺序不能改变合法 supersession judgment，unrelated package 中偶然相同的 parent Reference 也不能授权。inherited owner 通过 frozen baseline + `compositionStructuredOwner` 精确解析；sibling、arbitrary ancestor、wrong finding path/class、`surface-integration` finding、仅 target SHA 或 Generation identity 相同均不授予 supersession。该规则不放宽 `reviewedFindingAssets`、`structuredLanguageFinding` 或 `compositionStructuredOwner`。

涉及上述 acceptance/currentness 的 implementation 修改时，旧 first-real-use verification 不得 carry，第一次真实使用前须重新完成 docs → implementation → direct contract tests gate。本节只澄清既有字段与行为，不因文档整理增加 CLI gate 或变更 command semantics。

### Shared re-attestation plan field semantics

操作、输入路径、CAS/publication/recovery 合同仍以 [Shared representation re-attestation authority](SHARED_STRUCTURED_REPRESENTATION_REATTESTATION.md) 为准。本节只解释既有 plan 的 `authority` fields，不替代该 authority 或授权 re-attestation；receipt 本体中的同名字段没有 `authority.` 外层。

| Plan field | 精确 scope |
| --- | --- |
| `authority.changed_dependencies` | base shared representation successor 的 `old_currentness_inputs` → `current_currentness_inputs` 之间允许的 dependency drift；比较基准是 frozen base，不是“上一 re-attestation 的全部 implementation bytes” |
| `authority.validation_inputs` | shared re-attestation implementation/contract validation inputs，绑定 path/SHA/mode；例如 `internal/sitecontent/structured.go` 属于此 scope |

两组字段分别绑定 currentness dependency drift 与验证实现/合同。`structured.go` SHA 改变可以改变 re-attestation plan identity、planned receipt identity/path/SHA，而无需向 `changed_dependencies` 新增条目；`planned_receipt` 本身是 `{path, sha256}` Reference，不另含 identity 字段。不能用 `CHANGED_DEPENDENCY_COUNT` 推断所有 validation implementation bytes 是否变化，也不能因 count 不变便复用旧 plan 或 planned receipt SHA。两组输入仍各自按 canonical checker 重建/校验，未省略 currentness、identity、CAS 或任何正式 gate。

### Surface scope canonical command contract

```text
site-content surface-scope --locale <locale> \
  --generation <repository-relative package-closure> \
  [--generation <repository-relative package-closure>...]
```

| 字段 | 合同 |
| --- | --- |
| COMMAND / OPERATION | `site-content surface-scope`；只读重建 current Integrated Surface scope |
| INPUT_KIND | **PackageClosure References**；虽然 flag 名为 `--generation`，这里不是 StructuredGeneration，也不是 Generation Bundle、Surface receipt 或旧 scope JSON |
| INPUT_PATH_CONTRACT / FLAGS | `--locale` + repeatable `--generation <repository-relative package closure>`；CLI 计算文件 Reference。absolute（即使在 repository 内）、traversal、symlink 非法；不接受 `--input`、`--output`、`--apply`、`--finding` 或 positional args。stdout 重定向由调用者 shell 执行，不是 command mutation |
| STDOUT_KIND / SCHEMA | 完整 `IntegratedSurfaceScope` JSON object，当前 wire `schema` 为 **`go-learning/integrated-locale-surface/v1`**；没有独立的 `go-learning/integrated-locale-surface-scope/v1` schema，也没有 result wrapper |
| MUTATION_EXPECTATION / IDEMPOTENCE | `none`；每次重新验证 current closures/coverage、Unified Review 与 runtime/public context，不写 scope、receipt、closure 或 authority，不消费旧 PASS cache |
| PRIOR_OR_CAS | none；保持 reader 自身的 currentness 与 dependency mutation 检查 |
| SUCCESS_STOP / FAILURE_STOP | 成功停止在 Reviewer Bundle 之前；失败停止，不能把旧 scope 当 current 或继续 record/activation |

stdout 的 exact 顶层字段是 `schema`、`locale`、`glossary_sha256`、`unified_glossary_review`、`existing_coverage`、`package_closures`、`carried`、`review_required`、`runtime_public_context`、`canonical_routes`、`identity_sha256`。其中 `package_closures` 是 PackageClosure References；scope object 与带有嵌套 `scope` 的 Integrated Surface receipt 是不同 input kinds，不能只看共享的 schema string 推导 operation 或互换输入。

Shell/Python wrapper 仅按上述 canonical command/schema 验证字段；不自行推导 successor ancestry 或两个 re-attestation field scopes 的关系。典型误判、mutation 分类与 replay 停止边界统一见 [Wrapper / verifier 与 mutation provenance](WORKFLOW_HANDOFF.md#wrapper--verifier-与-mutation-provenance)。

## 完整 Page TU / internal slots / fixed batching

跨 Tour / Learn/Docs 的 source code-comment inventory、current accepted target comparison、English-base exception 与 future opt-in validator 见 [代码注释完整性合同](CODE_COMMENT_INTEGRITY.md)。该工程审计不改变以下正式 TU 边界或历史 lifecycle gates；unsupported grammar/currentness unavailable 必须显式报告，不能声称全量注释通过。

internal parser `go-learning/content-units/v2` 保留Markdown Goldmark AST、HTML tokenizer、窄Learn YAML schema；internal linguistic slots用于保护、精确impact、deterministic restoration与reconciliation，**不是formal TU**。当前56 documents有2,646 internal slots；min/median/p90/max = 2/84/269/691 UTF-8 bytes，<=20=383，punctuation-only=0；最大gomod-ref 207 slots。slot packing仅internal diagnostic，CLI summary明确标记diagnostic，不是模型调用plan。

正式页面表示保留 `go-learning/page-translation-unit/v1`；旧 `go-learning/package-workflow/v2` 仅用于历史证据读取与原 JSON 导入兼容，不继续复制审核或终结状态机。51 个物理页面就是51个正式页面翻译单元；embedded CodeExample=0，7 个结构化数据资产不进入逐单元质量审核。完整页面受保护输入通过块边界与行内绑定还原，一个页面一个候选及 A/B/C/D 评级。安全 Go 教学代码注释仍属页面；复用 Tour 保守片段分析器，代码身份、分隔符、布局和未知代码受保护。标题后缀、行内代码、Markdown 链接地址与标签绑定完整保留，可合法移动完整行内绑定，不能移动页面结构边界。

Page 重建逐 source coordinate 校验 protected token membership、pair/suffix 并恢复原始 bytes；内部 fragment 可为空或只剩标点，语言完整性由完整 Page 的独立 QC 判断。target 不重新参与 source teaching-comment eligibility 或 language-slot discovery；新增未保护 syntax/换行仍拒绝，Markdown heading/list/table/code 等 enclosing structure 仍验证，合法 soft-wrap token 的位置不绑定 prose paragraph segmentation/list tightness。Markdown emphasis delimiter 按 Goldmark 的实际匹配与 source binding 保护，内部自然语言不因 closing underscore 被误判为 machine atom。protector input/ID 真实变化时旧 bundle stale；只保留 exact input/ID 未变且重新验证通过的旧 target bytes，不改写旧 bundle/result 或 provenance。

stable append-only Page authority：`data/learn-docs-pages.json`，schema `go-learning/stable-page-index/v1`。首次按learn/tutorial/database/modules/security，surface内path lexical order冻结1–49。当前 /doc/ 与 /doc/install append 为50、51；future append52–60进入batch2；delete tombstone不复用；move/rename必须explicit mapping，unknown/ambiguous不猜。index61 fail closed，交维护者决定batching authority，不自动第三批、不rebalance。

正式initial Generation与QC各两次独立invocation：

| batch | stable membership | current Page count | raw source/context bytes |
| --- | --- | ---: | ---: |
| page-batch-1 | 1–30 | 30 | 267,539 |
| page-batch-2 | 31–60（当前31–51） | 21 | 以 current page-plan 输出为准 |

initial Generation / initial QC 每组最多30 Page；通用60 defensive ceiling不授予合并 initial 两组。两批初审完整结束后，正式 B/C/D 与 Surface reopen exact revision 可跨 initial membership，只受共享同 kind、最多60及正式 scope/identity/finding 等 gate 约束。successor re-QC 按 current `quality_check_required` pending、stable index顺序、同 kind、最多60选择，record语义一致，多组继续JIT。glossary-stale recovery保留既有固定分组；retry/recovery不变。transport safety与阶段调度分开，现有16 MiB text/context/output等transport hard safety limit不变；超出停止、报告evidence，不自动拆成几十组。

```sh
go run -mod=readonly ./cmd/tour-i18n site-content page-plan --package learn-docs-v1
go run -mod=readonly ./cmd/tour-i18n site-content generation-plan --locale <locale> --package learn-docs-v1
go run -mod=readonly ./cmd/tour-i18n retranslation progress --locale <locale> --content-package learn-docs-v1
go run -mod=readonly ./cmd/tour-i18n retranslation export --locale <locale> --generator chatgpt --content-package learn-docs-v1
# batch ID 使用真实 export 输出；后续 generation-bundle export/check/import 与 process/retry
# 均使用既有共享命令，并传 --content-package learn-docs-v1。
```

调度固定为第一批生成 → 精确重试至30/30 → 第二批生成 → 精确重试至21/21 → 完整51页候选快照 → 第一批初审 → 第二批初审 → 汇总全部 B/C/D → 共享精确修订 → 独立再审核。首轮未全部记录时禁止常规 revision export；真实恢复或当前性修复按既有授权路径处理。Surface finding 回共享精确修订机制，不恢复旧 Site revision 分支。原30/recovery-8/2/1完整输入与输出可由兼容适配器进入共享处理，不能用后来 subset Bundle 冒充原 invocation。新生成绑定当前完整输入与术语审核；历史接受按精确单元输入、候选及术语兼容证据判断，不因无关包级 SHA 变化抹掉旧 A。

Reviewer只findings/rating，不能replacement；同locale Generation/Reviewer长期session完全独立，policy仍GPT-5.6 Sol High。normalized Reviewer Bundle每Page context一份、Rows为完整Page input/target/ref/provenance；不按slot重复Document。B/C/D → 原Generation exact Page revision → validation → independent re-QC；unaffected A carry。finalize要求51 current Page、A-only、pending0、unresolved0、完整真实provenance和exact lineage。

Tour成熟present.Section/standalone eligible .go流程不重写，仍103 Page+19 Example、60/43/19 initial batches与独立QC。site milestones不改变Tour semantic authority。

## Locale plan / integrated delta-aware Surface / activation

### Integrated Surface Reviewer Bundle

mandatory shared structured composition 的正式 finding owner、`surface-record` 与 structured revision 的 current contract 见 [Integrated Surface finding authority](INTEGRATED_SURFACE_REVIEW.md#mandatory-shared-composition-finding-owner)。它复用 closure/Generation/representation 身份，不改变 fresh/carried 集合；首次真实使用 modified finding contract 前仍须完成 first-real-use gate。

标准顺序固定为：`package-closure` → `surface-scope` → `surface-reviewer-bundle` → `surface-reviewer-bundle-check` → Independent Reviewer → `surface-record` → `activation-preflight` / `activation-apply`。全部 deterministic 命令默认由 Maintainer Local terminal 执行；后续 locale 只替换 locale、closure/scope/output 等 runtime identities，不重复 CLI discovery。语义输入与 Reviewer result 详见 [Integrated Surface Reviewer contract](INTEGRATED_SURFACE_REVIEW.md)。旧 `surface-review reviewer-bundle` 继续服务 legacy Locale Surface Review，不能替代本 Integrated transport。

```bash
go run -mod=readonly ./cmd/tour-i18n site-content surface-scope --locale <locale> --generation <closure-a.json> --generation <closure-b.json> > <current-scope.json>
go run -mod=readonly ./cmd/tour-i18n site-content surface-reviewer-bundle --input <current-scope.json> --output <reviewer.zip>
go run -mod=readonly ./cmd/tour-i18n site-content surface-reviewer-bundle-check --bundle <reviewer.zip>
# CURRENT 后才上传 Independent Reviewer；也可按 locale + closures 直接导出：
go run -mod=readonly ./cmd/tour-i18n site-content surface-reviewer-bundle --locale <locale> --generation <closure-a.json> --generation <closure-b.json> --output <reviewer.zip>
# 真实审核后才落地 evidence/result，沿用原入口：
go run -mod=readonly ./cmd/tour-i18n site-content surface-record --input <independent-surface-result.json>
```

所有路径为 repository-relative；output immutable/no-overwrite。exporter 自己重建 current scope、closures 与 machine gates，不能信任任意输入 JSON。Bundle 自包含每个 fresh asset 的完整 source/restored target、适用的 Snapshot/Generation provenance、glossary/Review 与 integration context；carried 明确仅作 context，不扩大 fresh exact-set。current-check 严格验证 transport inventory、重建 current scope 并重导后 byte-compare，任何 semantic/candidate/closure/runtime 漂移 fail closed。实现复用一次 invocation 的 validation graph，不按每个 payload 重跑 full closure。

Reviewer 缺输入时是 `INPUT_GATE=FAILED`，不记录语言 failed receipt、不创建 reopen/revision，不改 candidate；修复 exporter 后用新的 Reviewer invocation/review_id。新 transport 不改 existing receipt/schema/record path，不改变语言 semantic authority，不重写原 Generation/Reviewer evidence。

`site-content campaign-plan` 是共享versionless locale work plan；scope分类carry/generation-required/review-required/stale-reopen/retired/ambiguous基于真实source/target/QC/Surface/compatibility evidence。exact oldTour source/target/quality/publiccontext证明后carry，新Page/structured surfaces待生成；package数量或milestone不影响classification。

`go-learning/package-language-closure/v1` 原子绑定51 Page finalization + 7 Learn data（迁移时可绑定经 current re-attestation 的49+2历史 finalization composition），或 current shell 的 Homepage shared Page finalization + 四份 shared structured + translation exact carry。输入、stdout 与 wire fields 见 [Package closure canonical command contract](#package-closure-canonical-command-contract)。旧无 Page 的 shell closure 仅历史验证。partial file/subdirectory 不能 complete。

`go-learning/integrated-locale-surface/v1`区分carried与review-required scope，绑定existing complete coverage、prospective closures、全部active routes、current unifiedglossaryReview、machine public/runtime identity、prior evidence与new/changed targets。unchanged旧Page不重新rating；carry不能伪装Reviewer重新审过。Reviewer审核new/changed语言、structured assets、cross-surface terminology、导航、homepage/translation、coverage/canonical/sitemap/publicruntime/广告integration。old TU被affected时仍必须正式revision/re-QC，Surface decision不能代替它。

`surface-scope` 的 `--generation` 实际消费 PackageClosure References，完整 stdout/mutation/停止点见 [Surface scope canonical command contract](#surface-scope-canonical-command-contract)；record 顺序见上方 Reviewer Bundle 入口，activation 输入与 CAS 见 [Activation canonical command contract](#activation-canonical-command-contract)。

independentSurface receipt immutable且绑定Markdown hash；activation需要 current完整integratedPASS，locale completion保存closure+Surface refs；以后newpackage扩展需要新的integratedgate，原exact package的历史语言judgment仍保留。每locale独立，无跨locale barrier；production/identity.json不保存language/content状态。

同一 PASS scope 中的多个 prospective closures 可分别按现有 CAS activation 依次完成，不逐 package 重做 Integrated Surface Review。首次 completion 的 currentness 只接受能够从该 receipt 的 `existing_coverage` 原 bytes 确定性重建的追加：原 completion 顺序与内容不变，每个新 completion 精确绑定该同一 PASS receipt 和 scope 内的对应 closure。原 Content Scope reference 的 SHA 必须等于 frozen baseline 的确定性编码 SHA；当前文件必须逐字节等于 baseline 加这些 completion 的正式编码。该证明仅恢复 coverage reference 的 predecessor/current 关系；完整 closure/QC、language、runtime/public context 与其余 scope 字段仍重验。consumed Surface reopen/replacement 与已冻结 Tour promotion recovery 的 historical closure reader 可通过该 lineage 读取 SHA 完全匹配的原 baseline bytes，保留原 ref/identity，不改写历史 authority，也不授权新/current reopen 或新 recovery creation。任何其他 coverage、证据或 context drift 都停止；scope 外新 package 仍须新的 integrated gate。

符合 class/owner gate 的 structured language finding 使用 `site-content structured-export --task revision --finding <failed Surface receipt> --locale <locale> --output <path>`，只选择 exact finding 对应资产；旧 accepted structured evidence 与 replacement 通过该独立 receipt 形成授权 supersession，closure保留未受影响输出。`surface-integration` runtime/public composition finding 按 [已记录 finding 的恢复路径](INTEGRATED_SURFACE_REVIEW.md#已记录-finding-的恢复路径)先做 bounded repository engineering，再重建 successor scope/Bundle 并独立复审，不调用 structured revision 或改写 TU 译文。未知/重复/out-of-scope finding、stale context、Markdown hash/session mismatch停止。新 locale 没有 Content Scope record 时只推导 incomplete；首次 activation以空 prior SHA atomic no-overwrite创建 sparse record，不执行第二次 locale initialization。

已有 completion 的 prospective replacement 只允许替换该 closure 的 exact `structured_baseline_closure`。旧 completion 的 package/source/routes、closure identity、历史 Page/structured integrity 与独立历史 Surface PASS 必须完全有效；旧 TU 不重做 current eligibility，当前资格由 prospective closure 的 exact package Snapshot/current authorization 验证。`existing_coverage` 排除了所有 prospective packages，因此冻结 Content Scope SHA 若绑定完整 predecessor coverage，必须从 retained coverage、sealed baseline closure、唯一独立历史 PASS 和原 completion 顺序重建完整原 bytes。歧义、scope 外 completion、非 baseline replacement、证据/hash/context drift 均停止。preflight 对同一 PASS 内的每个 prospective closure 验证当前语言 gates，原位替换 exact predecessor 或追加首次 completion；CAS 仍绑定实际完整 prior bytes，sequential replacement 不改任何历史 evidence 或 Surface judgment。

已安装且 current corpus bytes 精确匹配 sealed successor result 的 reader，可通过同一 exact reviewed activation lineage 读取其原 completion SHA；不能在后续合法 activation 后重新要求该 pending projection byte-current。pending corpus export/build/apply 仍要求真实 current completion，不使用此 historical reader。原 source/parser/affected scope、corpus delta、result identity 与 canonical bytes 全部继续重验。

Historical closure inventory 由该 closure 自身的 shared Page finalization、structured outputs 和冻结 source adoption 重建，不使用最终 coverage 的 inventory。复用跨 package 的旧 structured generation 时，只有原 Bundle 中 hash-bound adoption/foundation 明确授权、closure package identity 精确匹配、且 exact baseline 证明原 owner/target 的 retired source 才能排除；不能按 closure 列表任意裁剪 extra。保存后 current-check 的 baseline 选择统一见 [Frozen structured baseline 与 successor delta](#frozen-structured-baseline-与-successor-delta)。missing/extra、target/hash/epoch mismatch 保持 fail closed，所有原 evidence bytes 不变。

Historical Page finalization 必须按 closure 冻结的 package ID **和** authority SHA 解析 descriptor；不能用同名 current package 冒充。structured-only source successor 若只改变 enclosing package identity，只有重新验证原 source-adoption authority、证明正式 Page source/path/SHA 与 package contract 均未改变后，才可保留其原 descriptor 到 historical-only registry。membership successor 也必须保留完整 predecessor descriptor；两者都不改变 current scheduling 或历史语言 evidence。

package-closure 在昂贵 Page/current assessment 和 structured reconstruction 前，先只读检查 requested package、composition binding、finalization scope/schema/path 及 transitive historical closure 所需 descriptor 是否可精确解析。该 structural preflight 只提前拒绝不可能的输入，不是新 acceptance gate；成功后仍执行完整 corpus/Review、Page integrity/current validation、structured reconstruction、exact inventory 与结束前 dependency bytes/directory 检查。同一 invocation 可沿用已有 validation graph 的纯 representation projection 复用，包含 source-adopted Page 的不可变 wrapper；每次仍校验 reference SHA 与当前 bytes，current eligibility 不因 projection 命中而省略，不跨 invocation 保存结果。

existing Tour locale core expansion理想新增invocations（无revision）：Unified Glossary Refresh1 Generation、Unified Review1 Review、pending Structured1 Generation、Learn/Docs Page2 Generation+2 QC Review、Integrated Surface1 Review，即Generation4 / Review4 / total8。existing current Course SEO carry；若真实derivedmetadata需要新生成，单独报告该invocation，不隐含在8内。新locale仍onecampaign，额外Tour TU和derivedstage明确单列。

### Activation canonical command contract

本节是跨 locale 的 activation CLI / stdout / CAS / recovery 合同。实现 authority 为 [`site_content.go`](../cmd/tour-i18n/site_content.go)、[`activation.go`](../internal/sitecontent/activation.go) 和 [`activation_transition.go`](../internal/sitecontent/activation_transition.go)。命令由 Maintainer Local terminal 在仓库根目录执行；文档示例不构成 activation 执行授权。2026-10-18 本次文档交接停止于 `before-first-package-activation-preflight`，未执行下列 preflight 或 apply。

#### CLI、输入与 stdout

| 命令 | 允许的 flags | 输入 / 输出 |
| --- | --- | --- |
| `site-content activation-preflight` | 仅 `--input <repository-relative package closure>` | 只读；stdout 为一个完整 `ActivationPlan` JSON |
| `site-content activation-apply` | 仅 `--input <repository-relative package closure>`、`--prior-sha256 <exact preflight prior_sha256>` | CAS 提交；stdout 同样为一个完整 `ActivationPlan` JSON |

不传 `--locale`、`--package`、`--output` 或 `--apply`；locale/package 来自 closure，JSON 用 shell stdout 重定向保存。`--input` 必须指向 package closure artifact；不得传 Integrated Surface receipt、Surface scope、Reviewer Bundle 或之前保存的 ActivationPlan JSON。CLI 先调用 `ReferenceFile(root, input)` 取得 path/SHA，再将 closure Reference 交给 `ActivationPreflight` 或 `ApplyActivation`。

两个命令成功时的 stdout 都由 `printJSON(plan)` 输出，顶层 exact schema 如下；没有额外的 `path`、`status`、`identity` 包装，也不输出私有 prospective authority bytes：

```go
type ActivationPlan struct {
    Locale        string     `json:"locale"`
    Package       string     `json:"package"`
    PriorSHA      string     `json:"prior_sha256"`
    Completion    Completion `json:"completion"`
    ResultSHA     string     `json:"result_sha256"`
    AlreadyActive bool       `json:"already_active"`
}
```

`completion` 是完整 Completion 对象，包含 `package`、`state`、`package_identity_sha256`、`source_authority_sha256`、`completed_surfaces`、`local_canonical_routes`、`local_route_families`、`evidence_kind`、`evidence`、`evidence_identity_sha256`、`validated_context_sha256`、`legacy_surface_gate_state`；`evidence` 每项为 `{path, sha256}`。无需手工构造或修改这些字段。

- `LocalePath(locale)` 为 `locales/<locale>/content-scope.json`，它是 sparse locale authority。
- `prior_sha256` 是该文件当前**原始字节**的 SHA-256；文件不存在时为 JSON 空字符串 `""`，不是空文件的 SHA，也不是 `null`。shell 必须保留该空值，显式传 `--prior-sha256 ""`。
- `result_sha256` 是 prospective post-activation authority 原始字节的 SHA-256，不是 closure / Surface / ActivationPlan JSON 的 SHA。
- `already_active=true` 表示 exact Completion 已存在。此时 `result_sha256 == prior_sha256 == current authority SHA`；apply 仍先检查 exact prior，随后直接返回，不创建 lock、不重写文件。

#### Preflight、apply 与 Integrated Surface binding

`ActivationPreflight(root, closureRef)` 按以下路径验证：`CheckPackageClosure` → `LoadCurrent` → `packageCompletion`（内部再次检查 closure，并调用 `RequireIntegratedSurface(root, locale, closureRef)`）→ 读取 current sparse locale authority → 再读取 exact `LocalePath(locale)` bytes → `checkActivationPreflightState`，证明解码的 state 与捕获的 prior bytes 一致。随后仅在内存中原位替换 exact reviewed predecessor Completion，或追加首次 Completion；经 `ValidateLocale` 与 `Encode` 得到 prospective bytes / result SHA。exact already-active 分支直接返回原 bytes 的 SHA。**preflight 不修改仓库。**

Package completion 的 evidence 顺序固定为 **`[package closure, Integrated Surface receipt]`**，`evidence_kind=independent-package-closure/v1`。Surface receipt 由 `RequireIntegratedSurface` 根据 closure 自动解析并校验 current 完整 PASS、全部 prospective closure 与 retained coverage；调用者不以 Surface receipt 替代 closure input，也不手工拼接 evidence。

`ApplyActivation` 重新执行完整 `ActivationPreflight`，要求 `plan.PriorSHA == --prior-sha256`；already-active 才直接成功返回。其余情况以 exclusive create 创建 authority 同目录下的 `.site-activation.lock`，再次读取 current authority 并做 exact SHA / concurrent mutation 检查，然后写同目录 temp file、`fsync`、关闭文件：首次 sparse authority 通过 atomic hard-link/create，已有文件通过 rename 提交。只替换/追加目标 Completion，保留其他 completion；历史 closure、Surface receipt 与旧 completion 引用的 evidence 文件保持 byte-identical。

preflight 到 apply 之间 authority 若改变，必须 fail closed，不能重用旧 `prior_sha256` 强行重放。每个 package 都重新 preflight；前一包 apply 的 result SHA 是下一包实际 state 的 SHA，不能让同一队列的所有包共用第一包 prior。stdout 保存失败或 apply 终态不明也不能据此推断仓库未变。

#### 顺序与 2026-10-18 zh-CN 实例

Activation package order 来自 **current sparse/package authority 和 reviewed transition 合同**，不能按 package 名猜。replacement 保留现有 `packages[]` slot，首次 completion 才按提交顺序追加；这不规定永久的跨 locale package 名优先级。读取 raw/current authority 的数组及其 package/closure binding 来记录本次顺序，不用 `site-content coverage` 发现 transition 顺序：coverage 会验证旧 active completion 的 currentness，旧 authority stale 时会先 fail closed。首次 sparse authority 尚不存在时，依据 current package authority 与 reviewed prospective 集合明确本次追加计划，不把本例名称顺序当默认规则。

本次已确认的 `locales/zh-CN/content-scope.json` authority 顺序是 `tour-v1` → `learn-docs-v1` → `site-v2-shell`。Tour 为 retained carry，不是本次 prospective activation；本次按既有 authority slots 采用 `learn-docs-v1` → `site-v2-shell`。该解释由原位 replacement / reviewed baseline 重建合同支持，**不宣称另一顺序必然被实现拒绝，也不提升为跨 locale 永久规则**。

| Package | 当前旧 active closure | 本次 prospective successor closure |
| --- | --- | --- |
| `learn-docs-v1` | `data/site-workflow/zh-CN/learn-docs-v1/closure/20eb58e77e2f038cf7207384f36193b7e914f8ae5faebf41188788a73d514fbd.json` | `data/site-workflow/zh-CN/learn-docs-v1/closure/67837cd680ff6109a5c77d2a948416ba950051b733f5a3e6c0349069211f4299.json` |
| `site-v2-shell` | `data/site-workflow/zh-CN/site-v2-shell/closure/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json` | `data/site-workflow/zh-CN/site-v2-shell/closure/1f5e9f0fb28b2098c1fff90778f652f9541cfc6ff4d8c75e72ec2093f70dc926.json` |

两份旧 active completion 的第二项 evidence 均为 `data/locale-language/zh-CN/surface/20261014-zh-CN-integrated-surface-faa455c8-002.json`。本次 successor 共用的 current Integrated Surface receipt 是 `data/locale-language/zh-CN/surface/20261018-zh-CN-integrated-surface-004.json`。这些是本次已确认 state 的记录，不代表 activation 已执行；后续 locale 使用自己的 current authority 与 receipt。

已确认的错误输入示例（记录错误，不执行）：

```text
site-content activation-preflight --input data/locale-language/zh-CN/surface/20261018-zh-CN-integrated-surface-004.json
unknown JSON member scope
```

原因是 Surface receipt schema 不是 `PackageClosure`。本次 transition 中 coverage 的真实失败为：

```text
site-content coverage --locale zh-CN
current authorization stale: affected=0 stale=44 unknown=0
```

该失败不能代替 raw/current authority order discovery，也不授权刷新旧 authorization 或跳过 stale detection。prospective closure 的当前资格与 exact historical predecessor 的完整性由正式 preflight 分别验证。

#### Canonical shell 范例与跨 locale 复用

以下为进入 activation 阶段后由 Maintainer Local terminal 执行的完整 zh-CN 范例；本轮仅记录，不运行。全部变量值均为本次真实路径。每包单独执行 preflight、从其 stdout **按 string 原值**取 prior、apply，再比较 preflight/apply 的完整 JSON 与 current file exact SHA；任一步失败即停止，不继续下一包。诊断 JSON 留在 `/tmp`，不能作为下一次的 `--input`。Python 只解析 stdout / 计算文件 SHA，不替代 Go 的正式 validation。

```bash
(
  set -euo pipefail
  cd /home/wangqiang/code/go-tour-i18n

  activate_package() {
    local activation_locale="$1" activation_package="$2"
    local activation_closure="$3" activation_surface="$4"
    local activation_work activation_prior
    activation_work="$(mktemp -d /tmp/go-tour-activation.XXXXXX)"
    printf 'ACTIVATION_LOG_DIR=%s\n' "$activation_work"

    go run -mod=readonly ./cmd/tour-i18n site-content activation-preflight \
      --input "$activation_closure" > "$activation_work/preflight.json"

    activation_prior="$(python3 - "$activation_work/preflight.json" \
      "$activation_locale" "$activation_package" "$activation_closure" "$activation_surface" <<'PY'
import json, re, sys
from pathlib import Path
p = json.loads(Path(sys.argv[1]).read_text())
keys = {"locale", "package", "prior_sha256", "completion", "result_sha256", "already_active"}
if not isinstance(p, dict) or set(p) != keys:
    raise SystemExit("STOP: stdout is not the exact ActivationPlan schema")
if (p["locale"], p["package"]) != (sys.argv[2], sys.argv[3]):
    raise SystemExit("STOP: unexpected locale/package")
prior, result = p["prior_sha256"], p["result_sha256"]
if not isinstance(prior, str) or (prior != "" and not re.fullmatch(r"[0-9a-f]{64}", prior)):
    raise SystemExit("STOP: invalid/missing prior_sha256")
if not isinstance(result, str) or not re.fullmatch(r"[0-9a-f]{64}", result):
    raise SystemExit("STOP: invalid result_sha256")
if type(p["already_active"]) is not bool or (p["already_active"] and result != prior):
    raise SystemExit("STOP: invalid already_active state")
c = p["completion"]
if c["package"] != sys.argv[3] or c["state"] != "complete":
    raise SystemExit("STOP: unexpected completion")
if [ref["path"] for ref in c["evidence"]] != [sys.argv[4], sys.argv[5]]:
    raise SystemExit("STOP: unexpected closure/Integrated Surface binding")
sys.stdout.write(prior)
PY
    )"

    go run -mod=readonly ./cmd/tour-i18n site-content activation-apply \
      --input "$activation_closure" --prior-sha256 "$activation_prior" \
      > "$activation_work/apply.json"

    python3 - "$activation_work/preflight.json" "$activation_work/apply.json" <<'PY'
import hashlib, json, sys
from pathlib import Path
before, applied = [json.loads(Path(name).read_text()) for name in sys.argv[1:]]
if applied != before:
    raise SystemExit("STOP: apply plan differs; mutation=unknown; recovery diagnosis required")
authority = Path("locales") / applied["locale"] / "content-scope.json"
actual = hashlib.sha256(authority.read_bytes()).hexdigest()
if actual != applied["result_sha256"]:
    raise SystemExit("STOP: result/current SHA mismatch; mutation=unknown; recovery diagnosis required")
print(f'ACTIVATION_RESULT=PASS LOCALE={applied["locale"]} PACKAGE={applied["package"]} '
      f'RESULT_SHA256={actual} ALREADY_ACTIVE={str(applied["already_active"]).lower()}')
PY
  }

  activation_receipt='data/locale-language/zh-CN/surface/20261018-zh-CN-integrated-surface-004.json'
  activate_package zh-CN learn-docs-v1 \
    'data/site-workflow/zh-CN/learn-docs-v1/closure/67837cd680ff6109a5c77d2a948416ba950051b733f5a3e6c0349069211f4299.json' \
    "$activation_receipt"
  activate_package zh-CN site-v2-shell \
    'data/site-workflow/zh-CN/site-v2-shell/closure/1f5e9f0fb28b2098c1fff90778f652f9541cfc6ff4d8c75e72ec2093f70dc926.json' \
    "$activation_receipt"
)
```

跨 locale 复用时只替换 checkout 根路径，以及按该 locale current authority 确定的调用顺序、locale、package、closure path、expected Surface receipt path；CLI flags、stdout schema、prior 解析和 result SHA 检查保持不变。保留/增减本次真正 prospective 的调用，不给 retained carry 排 activation。每包的 prior 必须取自紧接其 apply 之前的本包 preflight，不能复制 zh-CN 的 SHA 或前一次日志中的 prior。shell options、变量和函数均局限于 subshell，失败不会关闭或污染维护者当前交互 shell。

#### Failure / recovery

| 状态 / 错误 | 唯一恢复动作 |
| --- | --- |
| `activation-preflight` FAILED | `MUTATION=none`；修复输入/currentness 后重新 preflight，不进入 apply |
| apply 返回 `activation prior state changed` | 停止；重新 preflight，不能盲重放旧 prior |
| `activation preflight state changed` / `activation concurrent mutation` | 停止；检查 current sparse authority，再重新 preflight |
| lock 已存在、I/O 失败、stdout 缺失、result SHA 不符或其他 `mutation=unknown` | 停止后续 package；保留现场与日志，先做 recovery diagnosis，不自动删 lock、重试 apply 或覆盖 authority |
| `already_active=true` 且 exact prior/result 检查通过 | 视为 idempotent current state；不得人为重写 authority |

## Runtime / C1 / safety

维护者已正式采用 [Shell current Homepage authority](SITE_SHELL_SUCCESSOR_AUTHORITY.md)：首次 zh-CN Site v2 release 直接使用 full frozen Homepage/shared chrome。foundation 原 pending 事实及 evidence 保持 immutable；source adoption 独立于 locale acceptance，不改 glossary/旧 completion。Homepage shared TU Bundle 是当前唯一下一份 JIT Bundle，机器校验通过后才导出 current structured Bundle。禁止用旧 home.html 或旧 completion 冒称 full Homepage complete。

保留已验收neutral opt-in `internal/site`：coverage由current Content Scope证据驱动；local covered canonical链接当前origin，未覆盖navigation直接go.dev；直接localmissing请求真实404。external URL原样；Security canonical `/doc/security/**`，legacyalias保持frozen exact/subtree fixed-target语义，只有targetlocalcovered才redirect。sitemap/canonical只含complete真实localcanonical，排除alias/data/assets/fallback。旧65Tour runtime不切换。

Advertising优先级 AdsUnsupported > GoLocal > Standard；`internal/tourpolicy` 是共享 classification authority。维护者已于 2026-10-07 冻结 AdsUnsupported exact set：sw-TZ/kk-KZ/fa-IR/am-ET，四门 publication 仍为 Standard；GoLocal 仍为 zh-CN/fr-FR/de-DE/ko-KR。homepage/translation 永远无广告，GoLocal Tour 无广告但 Learn/Docs 可按正式 Site policy 显示广告，AdsUnsupported 全站所有 current/future surfaces 无广告。原 unsupported-set Production blocker 已解除；不猜语言 tag、不联网推断、不另建第二份名单。

futureC1：source/file/route/dependency/content-unit reconcile → rebuildonecorpus → onerefresh/review → compatibility → exactnew/stalegeneration/QC → integratedSurface → activation。unchangedcarry、removedretire、newgeneration、changedPage stale、ambiguousexplicitmapping；不以wholecommit invalidation65locale。internal slots保留精确diagnostics，formalPage是review/revision boundary。

新machine evidence strict JSON、hash/selfidentity可重算、UTF-8、exact-set、traversal/symlink拒绝、immutable/no-overwrite。静态`site-content language-authority --apply`仅机械创建共享corpus与stable-index；不生成localeevidence。所有语言成果/65locale/prodidentity保持不变，本轮停止V2-D，不benchmark、不语言generation、不C1、不preview/Production、不commit/push。

正式 source authority更新后，`language-authority --prior-sha256 <exact corpus file SHA> [--apply]` 对静态 corpus做 preflight/CAS refresh；先验证 append-only Page registry，再重建 source projection，不执行 source sync，不改locale evidence。YAML item URL只是restoration coordinate，不进入 terminology context identity；machine-only code/image变动只有在projected context真正改变时才影响corpus identity。

## Post-promotion current authorization

shared TU 的正式 historical/current 分层、current re-attestation、immutable checkpoint、affected-only reopen 与 downstream steady-state 以[统一 contract](CURRENT_TRANSLATION_AUTHORIZATION.md)为准。registered package closure/runtime 默认使用同一 current authorization；新 package 增加不要求旧 package history replay。下文未建立 checkpoint 的 legacy Tour baseline reset / successor recovery 是兼容 migration 路径，不反向成为 future locale 的常规前置步骤。

Course SEO historical whole-file ref 不能借 TU successor carry。历史 archive/recovery WIP 保留、不扩展、不 apply，也不再是继续 Site v2 的前置条件。existing Tour 的合法 historical completion root + exact-current schema-v4 full Surface A + 独立 current Unified Glossary Review / finalized A-only TU / canonical authorization / strict Course SEO，组成只读 current checkpoint；不新增 closure/schema，不修改历史 evidence，不恢复旧 whole-file bytes。

Promotion 后历史 Reviewer A 不变；current target 必须另有 [accepted-target successor proof](SITE_V2_ARCHITECTURE.md#accepted-translationunit-target-successor-authority)。未来正式 `retranslation promote --apply` 必须指定 finalized Snapshot，自动保存 native receipt；旧实现缺 receipt 的历史只能由维护者显式执行 `retranslation recover-promotion --locale <locale> --snapshot-id <snapshot-id>` preflight，再明确 `--apply` 保存 historical recovery，不自动补写、不重跑 promotion。

`quality-check authorization --locale <locale> --snapshot-id <snapshot-id>` 分别输出 immutable Reviewer A 数量与 current canonical authorization；缺 receipt/recovery 时返回非零，不能把 pending current authorization 解读成 Reviewer 推翻旧 A。合法 successor 存在时 current QC/finalization 继续验证原 A-only/carry-forward，全套旧 evidence 不改写。

Native receipt 仅在 canonical/status 完整安装及 exact 复验后发布；recorded status bytes 是当时投影，不要求 mutable status 当前 SHA 永久不变。未收口 `.promotion-staging-*` 保留 backups 并阻止 current authorization、campaign-plan 与下一写操作；不得把 staging 中的 JSON 当作已成功 receipt，也不得盲目重放。

TU successor event 验证仍严格绑定 TU-critical historical evidence，但 recorded non-TU mutable context 不作为永久 current-SHA oracle；完整 proof_inputs 与 immutable historical membership 不改写，TU receipt 不授权 derived consumer。`campaign-plan` 只有完整 current checkpoint 成立才越过历史 mutable whole-file refs；否则维持原 fail-closed 路径。新的完整 Surface A 是一次 legacy baseline reset，后续 unchanged Tour carry、新 package/new Page 只审核真实 new/changed/affected scope，继续 package closure / Integrated Surface，不要求每次 full Tour 重审。Recovery 默认 dry-run、deterministic/content-addressed/no-overwrite，apply 仍是维护者 Local terminal 操作。

### Consumed structured source replacement

已被 corpus 消费的 repository-owned structured source head，重组/删除/替换正文时使用 [SOURCE_CONTENT_ADOPTION.md 的 replacement contract](SOURCE_CONTENT_ADOPTION.md#已消费-structured-source-的-replacement-successor)。默认 readonly preflight，显式 apply 才按 exact predecessor receipt identity 和 pointer SHA CAS；沿用同一 source receipts/current registry 与 corpus lock。append-only 与 unconsumed correction 不承载此语义。source authority 不产生语言接受，后续依次执行 corpus successor、glossary/full Unified Review、affected structured Generation/Review、current closure 与 Integrated Surface。历史 source/corpus/language proof 与 current eligibility 分开，pending/unknown publication 不盲重放。

## Current multilingual language validation

TU 与 structured current validation 统一消费 [versioned locale policy 和 language audit](LANGUAGE_DEFECT_EXHAUSTION.md#正式接入合同首次真实使用前仍须-command-contract-gate)。structured 检查完整重建表示（含 slot 外可见标点与 metadata），不以 frozen structure 或历史 accepted Generation 豁免。只有 deterministic defects block；完整 review candidates 进入既有 Reviewer transport。旧 receipt/closure/Snapshot 不重写；缺 policy 或 current gate 失败沿现有 currentness/recovery fail closed。新增 policy/schema、validator 与 transport acceptance 的第一次真实使用要求重新完成 first-real-use contract gate。
