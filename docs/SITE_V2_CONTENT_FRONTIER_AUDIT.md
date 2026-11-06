# Site v2 navigation frontier / project-page authority audit

2026-10-23 membership successor：current Learn/Docs 已统一为 `learn-docs-v1` 51 Page，`docs-entry-v1`仅作历史 migration identity，未来 locale不再执行独立两页阶段。当前 authority与49+2只读组合合同见[Learn/Docs membership](LEARN_DOCS_MEMBERSHIP.md)。下文的49页、独立 adoption及 pending glossary等状态均是注明日期的历史观察，不作为 current/future调度指令。

2026-10-23 Go local authority 补充：维护者已人工核实 [golang/go#81958](https://github.com/golang/go/issues/81958) 的正式申请及无广告 follow-up。Swahili、Kazakh、Persian、Amharic 为 **submitted / upstream issue open**，不是尚未提交，也不是 accepted/merged；French、German、Korean、Simplified Chinese 仍是四个已有链接，同一 issue 请求更正其 canonical `/tour/` URL。完整提交日期、2026-10-07 回复与冻结状态以 [Roadmap：Go local 申请状态](LOCALE_ROADMAP.md#go-local-申请状态)为准；下文 2026-10-20 的 application evidence missing 是当时的历史观察，已由本次补充 supersede，不能用于后续 source drafting 的 current 状态判断。本轮不修改任何项目页 draft 或语言 evidence。

2026-10-22 pre-release scope 决定：`/doc/`、`/doc/install`、`/translation/` 均必须在首次 zh-CN Site v2 Production 前完成，不能 deferred。新增 Docs source 属于独立 `docs-entry-v1` registered Page package；项目页 source successor 仍属于 `site-v2-shell`，通过 body reference 保留历史 source 和 accepted slots。current source pointers 见 `data/content-package-adoptions/current/` 与 `data/structured-source-successors/current/`，通用合同见 [Source content adoption](SOURCE_CONTENT_ADOPTION.md)。source/corpus authority 已发布；历史 reconciliation proof 与 current Review eligibility 已分离，只读 scope 已证明项目页 8 slots exact carry、22 slots pending。新 corpus 的 Unified Glossary Refresh / Independent Review 仍待正式语言角色完成，不代表 Generation ready。下文原 proposal / coverage / Preview 状态保留为当时审计，不代表新增页面已 Generation、QC、closure、activation 或可发布；learn-docs-v1 旧 49-Page finalization 不变。

本记录为 repository engineering 的只读 scope audit 与后续 machine authority proposal，不是 package adoption、语言生成、Surface 或 activation evidence。维护者本次 HUMAN GATE 已 FAILED；工程自动验收即使 PASS，也必须交回 HUMAN_GATE=PENDING_RECHECK。

2026-10-20 consolidation：本轮明确保持 `HUMAN_GATE=FAILED`，不启动 fresh Preview。`/doc/` 与 `/doc/install` 两项 frozen manifest 的 SHA/size/mode 与下文记录仍一致，`ADDITIONAL_FRONTIER_CANDIDATES=none`。所有 formal successors 完成后只安排一次 fresh Preview + automated acceptance + Maintainer HUMAN gate。

## Navigation frontier candidates

当前 `data/site-content-scope.json` 的 source_inventory 与 `locales/zh-CN/content-scope.json` 共同给出 learn-docs-v1 的 49 个正式 Page。下面是有限 frontier proposal，不扩大现有 authority。

| Canonical route | Source path | Reason / inbound source route | Prerequisite | Parent/hub | Current local coverage | Recommendation |
| --- | --- | --- | --- | --- | --- | --- |
| `/doc/`（主要导航入口写作 `/doc`） | `_content/doc/index.html` | frozen menus 的 Docs 主入口；49 Page 的共同父入口。frozen Getting Started 顺序是 `/doc/install` → `/doc/tutorial/getting-started.html` → `/doc/tutorial/create-module.html` | 否 | 是 | 否 | include-this-release |
| `/doc/install` | `_content/doc/install.html` | `/learn/`、`/doc/tutorial/`、getting-started、create-module、workspaces、generics、fuzz、database-access、govulncheck、govulncheck-ide、web-service-gin 与 security/best-practices 明确链接安装步骤 | 是 | 否 | 否 | include-this-release |

`pageRoute` 的现有 index contract 给 `_content/doc/index.html` 生成 `/doc/`，不能仅因菜单拼写 `/doc` 就擅自把 canonical 改成无尾斜线；未来 source adoption 应按既有 alias/redirect contract 提案并检查 `/doc` 入口。本轮二者仍未覆盖，正常导航 externalize，直接本地请求仍应 true 404。

source bytes 来自 `data/site-source-manifest.json` 的 frozen exact inventory：doc/index.html SHA256 `1e1ec2ae02d56739bc7491c64cc3d1d3e269423f22566d2a822b06c35c9470e3`；doc/install.html SHA256 `bbebc94da105910fb6f002a31a97f859c47202417cba4a0a60d6b1093e731c5c`。

已有 parent/hub `/learn/`、`/doc/tutorial/`、`/doc/database/`、`/doc/security/`、`/doc/security/fuzz/`、`/doc/security/vuln/` 均在当前 coverage；Modules 使用已覆盖的 `/doc/modules/developing` 入口，未发现 frozen modules/index 页面，不虚构 `/doc/modules/` hub。对当前 Page 中全部 `Navigation-prev` / `Navigation-next` source controls 的检查得到 14 条边，目标全部已覆盖，没有额外缺失序列步骤。普通 reference、API、release/blog、安装页的可选延伸内容不因被链接就纳入本次 release；本轮无其他 include proposal。

下一轮可提案仅含这两个 source 的增量 package/source authority，沿既有 TranslationUnit/structured Generation、独立 QC、currentness、finalization、closure、Surface 与 activation lifecycle 处理。本记录不执行其中任何步骤，不为新页生成译文。

## /translation/ 当前 source / target

- Canonical route：`/translation/`；source：`data/site-shell/translation.html`，source_inventory SHA256 `04ab70b5086539b36c55c504ee5cf302037b95c3abf4904824b079a8c316be6c`。
- 当前 active shell closure：`data/site-workflow/zh-CN/site-v2-shell/closure/1f5e9f0fb28b2098c1fff90778f652f9541cfc6ff4d8c75e72ec2093f70dc926.json`，该页 reconstructed target SHA256 `ad273e0c1bf67de01d26009580e3eebb106dd673c58c57088dd733537562b8ad`。
- target 是 `locales/zh-CN/structured/data/site-shell/translation.html` 的逻辑 structured target，当前不作为该路径的独立文件存在。既有 closure lineage 经 `41ec108c...` → `5e34b571...`，到 `data/locale-language/zh-CN/structured-generation/3aa00f3907d55604a78a411e88fe0fb478e232dd5f80cbfea66ca9e80c53bc78.json` 的 protected slots 重建，不能编辑虚构的 standalone target 或直接覆写 source。
- source 已称整个 Go Learning & Documentation Translations，并非只介绍 Tour；后续扩展应基于当前 source/target，避免把历史 Tour landing 的介绍误当现行 `/translation/`。

当前 source 与真实 Preview 正文的 exact project links **均为 4 条**：

| Exact source URL | Source label / current zh-CN label | Current purpose |
| --- | --- | --- |
| `https://github.com/shuijingwan/go-tour-i18n` | Source and development history / 源代码与开发历史 | 源码、开发历史 |
| `https://github.com/shuijingwan/go-tour-i18n/issues` | Feedback and corrections / 反馈与更正 | 反馈与更正 |
| `https://go.dev/` | Official Go website / Go 官方网站 | 官方网站 |
| `/` | Learning and documentation homepage / 学习与文档首页 | 当前 locale 首页，delivery 保留本地 root |

2026-10-20 历史观察：所述“8 个申请/相关链接”在上述页面 source/target authority 中不存在，不能按总页面 header/footer 中的重复 links 凑数或虚构申请 issue URL。2026-10-23 已补入的外部运营 authority 见上方 Roadmap 引用；它不自动修改这四条既有页面链接，也不把一个共享 issue 伪装成八个独立申请。

2026-10-20 targeted repo/history audit 补充：`internal/tourpolicy/policy.go` 核实四门 `go-local`，frozen `_content/tour/welcome.article` 的 Go local 列表提供下面四个 exact reference URLs。这是 frozen upstream 已列出状态，不是“申请已提交但未合并”，也不是今天 upstream HEAD observation。repo/docs/history 的 targeted 检查没有找到八项申请 issue/CL URL；不能把 publication policy、广告策略或 live 状态当作提交 evidence。

| Locale | Exact upstream-listed reference URL | Status |
| --- | --- | --- |
| zh-CN | `https://go-dev.shuijingwanwq.com/tour` | listed in frozen upstream source |
| fr-FR | `https://fr-go-dev.shuijingwanwq.com/tour` | listed in frozen upstream source |
| de-DE | `https://de-go-dev.shuijingwanwq.com/tour` | listed in frozen upstream source |
| ko-KR | `https://ko-go-dev.shuijingwanwq.com/tour` | listed in frozen upstream source |

2026-10-20 历史统计：`EXPECTED_APPLICATION_LOCALE_COUNT=8`；按 application/reference 的并集计 `APPLICATION_AUTHORITY_FOUND=4`、`APPLICATION_AUTHORITY_MISSING=4`；missing=`kk-KZ,fa-IR,sw-TZ,am-ET`。当时未找到 exact application URL；仅统计 issue/CL submission URLs 时 found=0。本次不改写当时观察，但该 missing 结论已由 2026-10-23 的 #81958 人工核实补齐：四门新增申请和四个已有链接 URL correction 共用一个 issue，不将 `open` 升级为 accepted/merged。

地区展示仅作事实分组：东亚（含东北亚）=`zh-CN,ko-KR`；欧洲=`fr-FR,de-DE`；中亚=`kk-KZ`；西亚=`fa-IR`；东非=`sw-TZ,am-ET`。申请状态独立取自上述 Roadmap 的已核实 evidence，不由地区、live 或 advertising policy 推导。PayPal 继续 deferred，不添加公开入口。

`/translation/` delivery presentation 已复用 `_content/tour/template/home.tmpl` 的九分区模型，保留 current go.dev header/upstream footer 与 ProjectFooter。现有五段 accepted 正文及链接不改写；分区标题只复用既有 UI catalog，coverage directory 从 validated Resolver exact routes 派生，未覆盖父目录不链接。新 `project_presentation` adapter/CSS 支持 source-owned rich sections，供后续正式 successor 承载 baseline、申请、地区、Independent QC、revision、closure、Surface、activation、Preview/HUMAN、Production 与长期维护说明；本轮未生成这些正式文案。

`PAYPAL_PUBLIC_URL_AUTHORITY=missing`。当前 `tour.ProjectSupport` 没有 PayPal public URL，现有 site identity/project metadata/source links 也无此 authority。不得猜 PayPal.me、账户、email 或其他财务标识；需要维护者后续提供公开且可发布的 exact URL。

## Coverage-derived directory proposal

唯一 current local set 来自 `locales/<locale>/content-scope.json` 的 complete package `local_canonical_routes`，需沿现有 runtime completion evidence validation 消费；`data/site-content-scope.json` 提供 source/route/family/package 映射。不能从文件存在、目录 prefix 或所有 locale 的 registry 推断完成状态，也不能在 transition 状态用 coverage 命令替代 raw authority 读取。

| Directory group | Current exact covered Page count | Group root covered |
| --- | --- | --- |
| `/` | 1 | 是 |
| `/translation/` | 1 | 是 |
| `/tour/` | 105 | 是 |
| `/learn/` | 1 | 是 |
| `/doc/tutorial/` | 18 | 是 |
| `/doc/database/` | 9 | 是 |
| `/doc/modules/` | 9 | **否**，必须以 exact children / developing 入口呈现 |
| `/doc/security/` | 12 | 是 |

目录应由 validated current route set 按 source family 分组，展示 exact covered children / counts，只有 root 已覆盖才使 group heading 指向该 root；父目录覆盖不意味着整个 prefix 已翻译。总计 156 条 canonical routes，其中 learn-docs-v1 为 49 Page。本轮不新增目录文案、UI keys 或页面内容。

## Upstream commit semantics

- `CONTENT_BASELINE_COMMIT=db076098077c07d3cef1b85a2cf56ff52777f587`，repository=`golang/website`；依据 `data/site-source-manifest.json`、`data/site-content-scope.json` 和 `_content/tour/site-metadata.json`。
- `data/terminology-discovery-sources.json.website.commit` 同样记录该 frozen commit。unified glossary corpus 的 contributor identity 是语义来源 identity，不是新的 upstream HEAD observation。
- `UPSTREAM_OBSERVED_COMMIT=missing`：上述 current machine authority 没有独立 repository-wide upstream observed HEAD 字段/证据；不能把 frozen commit 或其时间标成“当前上游最新”。本轮未联网查询 commit。
- 后续项目页可显示 repository-wide **content baseline**，范围涵盖 frozen website 内容；若要比较 observed upstream，先建立独立 exact observation authority，再走正式语言内容变更。

## Regional coverage structure proposal（不生成正式 replacement）

依据 `docs/LOCALE_ROADMAP.md`、唯一 languageRegistry 和 production identity，建议未来介绍分为：项目总体范围 → exact local coverage directory → 多区域语言选择策略 → verified application/related links → baseline/observed 差异 → public support（仅配置存在时）。地区不是新增 registry，也不改变 frozen commercial ranking。

可供后续正式文案审查的 locale grouping：

| Proposed geographic grouping | Existing locale facts / authority |
| --- | --- |
| 东亚（可在下层说明东北亚，避免作为两组重复计数） | `zh-CN`、`ja-JP`、`ko-KR` 已列入 roadmap、registry 和 production identity |
| 中亚 | `kk-KZ`；roadmap 明确已 live，并保留 official-link / ecosystem strategic override |
| 西亚 | `fa-IR`；roadmap 明确已 live，并保留同一 strategic override |
| 东非 | `sw-TZ`、`am-ET`；roadmap 明确已 live，并保留同一 strategic override |
| 欧洲等其他实际市场 | 如 `de-DE`、`fr-FR`，以既有 registry/production identity 为事实来源，不把 66 条 registry 等同于所有 Site packages 已本地化 |

这些是结构分组建议；repo 尚无独立地域 taxonomy。申请状态以 [Roadmap：Go local 申请状态](LOCALE_ROADMAP.md#go-local-申请状态)为准：四门新增申请已提交、共享 #81958 仍 open，另外四门为 existing links 的 URL correction；没有八个独立 application URLs，不能据此声称新增申请已获批或链接已收录。正式页面文案仍须后续 source/language lifecycle，本轮不生成 replacement。

上轮 Docs presentation 暴露的 empty ancestor crumb 已纳入 DI-20261019-008 的独立 breadcrumb correctness recovery：covered ancestor 使用 completed localized metadata，uncovered ancestor 使用经 frozen manifest 验证的 canonical source metadata 并 externalize，缺失/空 label fail closed；不为此新增父页译文或 coverage。工程自动验收不能代替 HUMAN GATE，DI-008 仍等待维护者复验。该修复不执行上述 navigation-frontier 或项目页内容扩展。
