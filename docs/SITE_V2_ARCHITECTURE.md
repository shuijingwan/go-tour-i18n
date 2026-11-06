# Site v2 Architecture / Content Scope authority

current Learn/Docs membership successor已把原49 Page与正式 adopted两页合并为同一个 `learn-docs-v1` 51-Page package。原 descriptor/adoption/语言 evidence仍供历史 reader验证；`docs-entry-v1`不再单独进入 current/future生命周期或 Surface planning。source/corpus与 lifecycle grouping的边界、49+2无新 A组合及 immutable closure扩展见[Learn/Docs membership authority](LEARN_DOCS_MEMBERSHIP.md)。

全项目执行共同遵守[共享翻译单元生命周期与五角色协作](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)：所有内容表面复用同一生命周期；页面差异限于适配器；传输安全与逐单元机器校验分层；先恢复、先复用，中文说明，交接只给一个执行者、动作和停止条件。本文中 Tour 的 present / Example 细节属于 Tour 适配边界，不另建 Learn/Docs 状态机。Learn/Docs 两批生成和机器校验全部完成后才初审，两批初审全部完成后集中修订。角色与工具分工不替代任何正式 machine authority 或语言质量 gate。

本文是 Site v2 架构与内容范围的正式 authority；V2-A 建立 contract、schema、离线 inventory/current-check 和可信 legacy bootstrap，不是翻译、runtime 切换或 Production 授权。单门 locale 的既有正式 gates 仍以各 Runbook 为准。

V2-D runtime / shell与Learn-Docs content-unit / transport / independent QC / atomic activation 的 executable contract 见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。V2-D 不生成 locale 翻译，不激活 incomplete package，不切换既有 Tour runtime。

## Versionless Locale Architecture

正式采用“版本无关的 Locale 持续演进模型”。v1/v2/v3 是 repository implementation / content-package / source-contract milestones，不是 locale language governance namespace。**Content package 可以版本化；Locale language system 不版本化。** package 是 source/completion/reuse boundary，不是 language-governance boundary。

每个 locale 永远只有同一个 locale identity、一份 `locales/<locale>/glossary.yaml`、一个长期 Generation session 和一个独立长期 Reviewer session。Tour / Learn / Docs / future registered surface 共享 terminology、provenance、compatibility、stale/carry 和 integrated Surface Review。新增 package 只处理新内容、shared surfaces 与真实 affected scope；exact source/protection/target/QC/glossary/public context unchanged 的历史成果继续 carry，不因 milestone、whole upstream commit 或无关 document 变化重做。

统一 source registry 是 `data/glossary-source-corpus.json` 的 contributors：由正式 Tour parsers、English UI、canonical article/SEO authority、Content Scope 注册的 shell/Page/Learn data 机械投影。structured-assets consumer 使用同一 contributor metadata，不维护另一份 surface 名单。当前正式 Learn/Docs Page TU 是完整物理 Page（51），internal slots 只负责保护/重建/影响分析；7 YAML 和 shell 属于 shared structured-assets。固定 Page stable index authority 在 `data/learn-docs-pages.json`。

Structured lifecycle 共用 historical/current 边界：historical completion 验证原冻结 source、旧 glossary/full Review、Bundle、provenance、output/validator 与 closure integrity；current eligibility 对 accepted asset 逐项判定 compatible/affected/stale/unknown。旧 completion 不因今天有 affected asset 而阻止规划。正常处理为 historical baseline → current per-asset assessment → exact affected Generation delta → merged current closure → delta-aware Integrated Surface；具体 executable contract 见 [Structured historical baseline 与当前精确 delta](SITE_V2_WORKFLOW.md#structured-historical-baseline-与当前精确-delta)。使用现有 structured plan/Bundle/generation/closure 的可选字段，保持旧 evidence immutable，不新增 QC 状态机、不 mass migrate，不伪造 Surface finding；未知 parser 或缺少 lifecycle validator 的 pending source foundation fail closed。

Terminology discovery 与 active Content Scope 分离。冻结的 go.dev / Go source 高价值术语输入使用独立 `data/terminology-discovery-sources.json` + `.zip` authority，不加入 `data/site-content-scope.json`、package coverage、routes、completion、runtime 或 structured assets，也不是 V2-C1 upstream sync。启用后 unified corpus 升为 `go-learning/glossary-source-corpus/v2`，每个 contributor 显式声明 `purpose=active-content|terminology-discovery`；discovery contributor 永远 `structured=false`，但与 active contributor 一起进入同一个 corpus identity 和同一次完整 Unified Glossary Review。未显式应用该 authority 时，当前 v1 corpus、现有 receipt 与 frozen baseline 保持不变。

新 unified Glossary Review 同时绑定完整 glossary bytes + current corpus identity；旧 Tour-only receipt/legacy coverage immutable，仍验证旧 Tour completion。exact unified current Review 优先，旧 receipts 不参与 unified ambiguity；新语言工作要求 unified coverage。具体 machine contract、CLI 与 migration 顺序见 [Site v2 Workflow](SITE_V2_WORKFLOW.md)。

## 1. 版本、命名与职责边界

canonical English public project name：**Go Learning & Documentation Translations**；中文维护语义：**Go 学习与文档多语言本地化项目**。这是 public / user-visible identity，不是技术身份迁移。V2-A 只冻结命名 contract，不修改现有 Tour 文案与其审核证据；后续正式 Site surfaces 才生成/审核新文案。

| Identity | 语义 | Coverage |
| --- | --- | --- |
| `site-v1` | 已正式上线的 A Tour of Go 成果 | `tour-v1` |
| `site-v2` | 统一 Go 学习与文档站点架构 | `tour-v1`、`site-v2-shell`、`learn-docs-v1` |
| `tour-v1` | 已有正式 Tour content package | `/tour/**` |
| `site-v2-shell` | 独立 localization / review 的站点 surfaces | homepage `/`、项目说明 `/translation/` |
| `learn-docs-v1` | Site v1 → v2 的增量 content package，不是项目版本 | Learn / Tutorial / Database / Modules / Security |

运行 Site v2 shell 不意味着 Learn / Docs 翻译完成。某 locale 可以有 Site v2 shell 和正式 `tour-v1`，其余目标直接链接 go.dev。没有跨 locale batch barrier。

保留 legacy-stable technical identity：`shuijingwan/go-tour-i18n`、Go module `github.com/shuijingwan/go-tour-i18n`、`cmd/tour-i18n`、`_content/tour/**`、Tour 专属 package/identifier、65 个 hostname、`production/identity.json`、`/data/go-tour*`、`go-tour*.service`、`go-tour-release-*`、ports、TLS、DNS、CDN、Playground Origin、IndexNow identity。不得为 public rename 迁移稳定 Production infrastructure。新 shared routing / site / homepage / docs / content scope 使用中性 namespace；V2-A 新模块是 `internal/sitecontent`，不做大规模 Tour namespace refactor。

本轮 repository coding 模型选择不改变 Translation Generation / Independent Reviewer 政策。现有正式 baseline 继续 GPT-5.6 Sol High。第一门 Site v2 正式翻译前，由维护者另行受控 benchmark GPT-5.6 Sol High、GPT-6.1 Sol High、GPT-6.1 Sol Medium，再明确决定；本架构不是模型切换授权。

## 2. Production identity 与 Content Scope 分离

`production/identity.json` 只回答 locale 在哪里部署、使用什么 machine identity。不得加入 coverage、translated route、package completion、glossary compatibility、stale 或 Site surface completion。

- 全局静态 authority：`data/site-content-scope.json`，schema `go-learning/site-content-scope/v1`。
- 每个 locale 独立 authority：`locales/<locale>/content-scope.json`，schema `go-learning/locale-content-scope/v1`。
- 离线 frozen source inventory snapshot：`data/site-content-sources.zip`。它不是 Generation/Reviewer bundle，不是 runtime content，也不是 upstream sync；保存可重算的 frozen bytes，避免 current-check 依赖仓库外 checkout/网络。初始化从 clean frozen checkout 只读生成，后续 current-check 只使用仓库内 snapshot。
- `data/glossary-compatibility.schema.json` 在 V2-A 是 design schema；V2-B 已正式化为 executable contract。实现与执行入口是 [Glossary Compatibility / Freshness Gates](GLOSSARY_COMPATIBILITY.md)。V2-A 历史不补造 compatibility PASS。

Global machine schema 定义 versions、packages、surfaces、source kind/path/origin/hash、canonical route、redirect alias/raw/resolved destination、asset/data dependency、package contract dependency、activation/publication semantics、upstream repository/commit、source/package/global SHA-256。不保存任何 locale 动态完成状态。`tour-v1` article 是多页面源，canonical routes 来自持久 Tour catalog，不是假定一个文件一个 URL。Tour inventory 是 physical source closure，不替代其现有 Page / Example catalog；既有 103 Page / 19 eligible Example 的 TranslationUnit membership仍由原 catalog 决定，不根据文件数量或 asset 存储分类扩展/删除旧 Unit。

per-locale state 是 **sparse completion authority**：只持久记录正式 `complete` package 的 package/scope identity、source authority、completed surfaces、local canonical routes/route families、completion evidence refs/digest、validated context identity。global package 在 locale file 中缺席，machine-readably 等价于 `incomplete`；禁止持久保存 incomplete record 或其 package/source identity。check/status 从 global package set + locale completion set 推导完整状态。未完成 shell / Learn-Docs source 变化、未来 global package additive 扩展均不使已有 Tour completion stale，也不要求修改全部 locale files；已完成 Tour 自身 identity 变化仍必须 stale。unknown package、伪造 completion fail closed；V2-A 历史只接受既有 `legacy-tour-closure/v1`。V2-D additive 接受 shell/Learn-Docs 的 `independent-package-closure/v1`，但必须重算完整 current source/parser、provenance、validation 与独立 A-only finalization；没有 evidence 的 complete 不合法。future unknown package 仍不允许 activation。

`tour-v1` public canonical projection 固定为 `/tour/`、`/tour/list` 与 current formal catalog 的 103 个 `/tour/<article>/<section>`，合计 105 routes，例如 `/tour/basics/1`、`/tour/welcome/1`。catalog 内部 `Page.Route`（如 `/basics/1`）不变，但不得作为 Site public canonical route。

不同 locale campaign 只写各自 state，不共同更新一份 mutable locale registry。Global scope 仅在正式 shared source/contract 变更时更新。包 identity 按本包逐文件 source/dependency/contract 重算，不以整个 upstream commit 作为唯一 freshness；全局 identity 不是跨 locale completion barrier。

严格 JSON parser 拒绝 unknown fields、duplicate members、trailing JSON；文件引用拒绝 traversal/symlink。current-check 重新从 snapshot 和实际 Tour source/catalog 生成 global authority，比较角色、routes、依赖、membership 和 deterministic identities；locale check 重新验证完成 closure。未知角色、重复 canonical、redirect 冒充 page、缺失/变化依赖、虚假完成或 stale identity 都 fail closed。SHA-256 是完整性/currentness，不是新的人工语言质量决定。

## 3. learn-docs-v1 exact frozen inventory

baseline 保持 `golang/website@db076098077c07d3cef1b85a2cf56ff52777f587`。V2-A 验证只读 checkout HEAD 与 clean working tree，不 fetch/pull/update，也不修改 runtime `_content` 或原 frozen baseline。

| 物理 source scope | files |
| --- | ---: |
| `_content/learn/**` | 8 |
| `_content/doc/tutorial/**` | 25 |
| `_content/doc/database/**` | 9 |
| `_content/doc/modules/**` | 16 |
| `_content/doc/security/**` | 22 |
| 合计 | **80** |

历史 foundation frozen tree 分类：**49 page + 7 data = 56 localizable sources，6 redirect-only，18 media/assets**。inventory 同时识别 frozen upstream `parseMeta` 的 YAML front matter 与 HTML-comment JSON metadata `<!--{ ... }-->`；JSON keys 按 upstream contract lower-case（`Redirect` → `redirect`），不能把 JSON redirect 误归类为 page。malformed/ambiguous metadata fail closed。

6 个 redirect-only source：`_content/doc/modules/pruning.md`，以及 `_content/doc/security/` 下的 `vulncheck.md`、`vuln/vulncheck.md`、`vulndb/index.md`、`vulndb/api.md`、`vulndb/policy.md`。`pruning.md` 的 JSON `Redirect` 对应 alias `/doc/modules/pruning`，raw / resolved destination 均为 `/doc/modules/managing-dependencies`；alias 不进入 canonical routes。所有 redirect 的 raw destination、解析结果及 frozen redirect implementation hash 均记录。Learn YAML 是 `/learn/` 数据，不拥有独立 route；redirect 不产生翻译 page；asset 不形成 TranslationUnit，但全部纳入 package source closure。scope 外直接图片依赖及 templates/route implementations 单独绑定，不增加 80-file 翻译 campaign 范围。

Security canonical route families 是 **`/doc/security/**`**。依据 frozen `internal/web/page.go` 的 source→page 路径映射、`internal/web/site.go` page lookup 与 `internal/redirect/redirect.go` 的旧 `/security/` handlers，legacy alias 指向 `/doc/security/`，不是反向。index source 对应目录 canonical route；`.md`/`.html` suffix 不成为页面 canonical URL。

redirect handlers 有精确与 subtree 匹配；subtree `Handler` 使用固定目的地，不自动拼接 suffix。例如 frozen `vulncheck.md` 的 raw `/security/vuln/vulncheck` 经 `/security/vuln/` subtree handler 到 `/doc/security/vuln/`。保留这种真实 legacy 行为，不擅自替 upstream 修 route，不把 stub 当 page。

后续 diff 按 path / role / route / dependencies / content SHA 区分 `unchanged`、`added`、`removed`、`content_changed`、`role_changed`、`route_changed`、`dependency_changed`。无法识别的 parser/frontmatter/映射与 identity move 是 `ambiguous/unknown`，停止并由维护者显式映射，不猜测 ID。V2-A 只提供 per-file diff primitive，不提供任意新 upstream commit 的自动 apply。

## 4. 65-locale legacy Tour bootstrap 与复用

`content-scope bootstrap-tour --all` 先 preflight 全部 locale，再只新建独立 state；已有完全相同文件可重复检查，已有不同文件拒绝覆盖。固定 bootstrap cohort 是当前 65 个 live community locale，与 current Tour registry 的 locale/public URL 一一匹配；官方 generic English `en` 不计入 community cohort。

完成 closure 不仅依赖 live：复用现有 Current Glossary Review（含既有固定 legacy coverage authority）、原始 passed Locale Surface Review A receipt 与对应 Markdown，读取真实 catalog/status，验证全部 canonical ready TranslationUnits、source/structure/术语、UI、article metadata、current Course SEO、其他正式 Tour surfaces。另选取既有 full A-only QC finalization，逐 Unit 比较其 Snapshot 的 candidate/source SHA 与当前全部 canonical bytes，调用现有只读 finalization verifier 验证实际 QC/carry-forward lineage；尤其不能只靠 Page SEO hash 推定 19 个 Example 已审核。使用与既有 CLI 相同的 committed catalog + source hydration，不重新生成历史 catalog identity。completion refs 绑定原 gate/Markdown、必要 registry compatibility baseline、locale identity/status、glossary、UI、metadata、全部 canonical candidates，以及真实 finalization、Snapshot manifests、QC results、selected candidates/validation evidence；validated context digest 同时绑定 read-only Surface package input、source/config 和该 locale 的 live deployment projection。

`legacy_surface_gate_state` 区分当时的 `current` 与 `historical-verified`，不能把旧 gate 伪称今日 current Surface A。没有新的 exact-current v4 checkpoint 时，原 historical carry 仍要求严格 glossary/config lineage 与完整原 context digest；已有新的完整 current v4 Surface A 时，可按文末组合独立 machine authority 建立 modern checkpoint，不恢复全部旧 mutable bytes。两条路径都不发新 A、不授予 Production/publication权限、不改写历史 completion。

既有 promotion 对文本 artifact 的合法 EOF 投影是“恰好一个 final LF”。Snapshot 保留原 raw candidate SHA；bootstrap 先验证实际 selected bytes 与原 SHA 完全一致，仅对该已有 EOF 规则的投影与 canonical bytes 比较，不容许任何其他字节差异，不改写 Snapshot。A-only finalization/current lineage 仍由原 verifier 检查。

这是可信 Site v1 成果的 deterministic migration，不是新 Reviewer 决策、QC generation、promotion、发布或 Production 测试。历史 Snapshot/QC/Surface/Production evidence 字节不变；不新造审核，不重写历史语言 provenance。现有 promotion 后的 ready canonical state + 验证过的既有 Surface A 语言证据与正式 export validator 是本次 closure；不虚构旧 Snapshot 中不存在的术语依赖，不要求对全部历史 QC 再跑一遍 lifecycle。

Bootstrap仅持久保存 `tour-v1=complete`；shell/Learn/Docs 未完成状态由缺席推导，不复制65份未完成 package。当前 Site v1项目页成果不冒称 Site v2 homepage完成。V2-A历史不绕过原 gates；V2-B新增 independent archive/compatibility/projection evidence向前扩展，仍 fail closed且不重写历史成果。

## 5. Homepage、translation、coverage-aware links

full frozen Homepage/shared chrome 现已由维护者决定作为本次 zh-CN Site v2 release 的 current source authority，见 [Shell current adoption authority](SITE_SHELL_SUCCESSOR_AUTHORITY.md)。foundation 建立时的 pending 状态保留原 evidence；additive CAS adoption 将 `_content/index.md` 的一个 shared Page TU、四份 structured source 与 translation exact carry 纳入 current shell package。旧 home.html 仅 historical-only，旧 completion 不满足 current gate；runtime 仅消费新的 completed bytes，共用 localized chrome，保留原 coverage/true 404/Tour renderer。adoption 不产生语言 acceptance、不改写旧 locale evidence、不授权 activation。

未来 Site v2 `/` 是整个 Go 学习与文档站首页，homepage display text 自身是必须 localization/review 的 surface，即使 destination 是官方 English。现有首页长项目说明迁至 **`/translation/`**，不要使用 upstream 已有 `/about/`。

`/translation/` 是全站翻译项目说明，包含 Tour、Learn、Docs、future formal surfaces、upstream authority、翻译 workflow、independent review、统一 glossary governance、stale/sync 方法、feedback/GitHub、development log、publication/advertising boundary；不是 Tour-only 页，也不绕过 upstream GoLocal agreement。

Coverage-aware resolution：

1. 当前 locale 对 canonical route 有正式 local coverage：链接当前 hostname 的同 canonical route。
2. 当前 locale 未完成目标：首页/导航直接链接 `https://go.dev` + canonical route（例如 `https://go.dev/doc/security/`）。
3. upstream 原本是 external destination：保留正式 external URL。

仅在 navigation destination resolution 中，既有 canonical 处理后未命中 explicit redirect alias、也没有 exact local coverage 的路径，可尝试只增加或删除一个 trailing slash；只有 alternate route exact 存在于当前 locale 的 local coverage 时，才链接该本地 canonical route，并保留 query/fragment。exact local route 保持原 identity，explicit alias 的既有 target/外链语义优先；global canonical inventory 中存在但 locale 未覆盖的 alternate 不授予 local coverage。此规则不注册 HTTP slash alias、不扩大 route set、不修改 Content Scope 或 sitemap canonical set，也不做其他 path guessing。

禁止 untranslated route 的 local fake page、local 200 shell、local 302 fallback。手工访问不存在的 local route 保持真实 HTTP 404；未来 404 可提供官方 English 链接，但不得 local canonical、不得进入 sitemap、不得冒称 coverage。本轮不修改现有 Tour rootHandler/runtime，以上是未来 Site v2 router 的验收 contract。

## 6. Sitemap、canonical 与 atomic activation

Site v2 runtime 启用后，sitemap 由 Content Scope authority 驱动，只纳入正式 shell `/`、`/translation/` 和该 locale 已 local-complete 的 canonical routes。两个 shell surface 必须先通过正式 completion gate；仅 global inventory 存在不代表每个 locale 可发布。fallback go.dev route、redirect alias、asset/data-only、真实不存在 route 均不得进 local sitemap。

canonical 指向该 locale 真正存在的 local canonical route；alias 不同时充当 canonical content URL；不存在 route 返回 404。不能为了 coverage 导航擅自修改现有 GoLocal Tour reachability/publication policy。

未来一门 locale 原则上一次连续完成完整 `learn-docs-v1`，内部允许合理 Generation/Review batch，但不得 tutorial/database/modules 分裂成长期多轮 campaign。Generation → independent review → automatic validation → machine gates 全包完成后，一次切换 local coverage，不提前暴露某个目录的半成品。不同 locale 独立 campaign/activation。

## 7. Unified glossary 与正式 compatibility evidence

每个 locale 永久只有 `locales/<locale>/glossary.yaml` 一份全站 glossary。Tour / Learn / Docs / future surface 共享；禁止 core/Tour/Docs/Learn glossary、surface extension 或 surface-owned 永久 terminology layer。

**Glossary Review currentness 不变**：full glossary 任意字节变化，旧 receipt stale；必须重新取得 generation-independent **full Glossary Review PASS**。impact analysis 不代替或缩小 full Review。

**Downstream compatibility 独立**：新 full Review PASS 后，V2-B 使用 old archived glossary → new archived glossary → normalized semantic delta → deterministic impact → compatibility lineage，不因 whole SHA 不同自动全量 reopen。没有可验证 evidence 时仍 fail closed。

Terminology discovery 的目的只是把高价值 Go/go.dev 术语更早暴露给同一份 versionless glossary，降低后续 package 增长造成 glossary byte churn 的预期，不缩小任何 Review gate。future content package 没有新增 governance-significant terminology 时，glossary refresh 可以合理得到 byte-identical 输出，但 corpus identity 变化仍必须执行完整 Unified Review；存在新高价值术语或 collision 时，由原 Generation session 刷新单一完整 glossary、独立 Reviewer 执行完整 Unified Review，再由 compatibility assessment 决定真实 affected prior scope。禁止采用“新增一个 Page 就机械新增一个 glossary term”的规则。

正式 schema `go-learning/glossary-compatibility/v1` 绑定 locale、old/new immutable bytes refs/SHA、新 full Review ref/file SHA、normalization/algorithm version、四类 normalized delta、exact source/candidate/UI/article/Course SEO/other context、完整互斥 affected/compatible scopes、prior lineage 与 self identity。真实 loader 的 mapping/list 转为 category/term/old/new，删除为 null，不保留从未正式使用的 design-only targets/context_rule shape。

V2-B machine validator 重建 glossary delta、当前 exact inventory、完整分类与 self identity，并验证每个 archive/Review/lineage ref 的原始 bytes。strict JSON 拒绝 unknown/duplicate/trailing；schema/version/path/hash/context mismatch、断链/分支/未知 parser 均 fail closed。Normalization 和 impact 算法版本、四类术语规则、Unicode boundary/protected structure 与精确消费端见正式 runbook；符合 JSON shape 本身不授予 compatibility PASS。

若新增 `SQL transaction` / `FIPS 140` 等 Docs-only terms，并通过真实 Tour source/candidate/UI/metadata/SEO/surface context 完整证明 Tour unaffected，可保留 Tour candidate/QC/SEO/UI/metadata/其他 surfaces。对 `interface`、`channel`、`function`、`module` 等 Tour 也会出现的术语，不能因 source 命中就机械全量 reopen：`tour-visible-glossary-impact/v2` 允许“新增 `preferred` 且历史 accepted target 已逐字包含当前首选译法”按 exact context carry；新增 `mandatory` 只允许 exact label pair carry。真正改变已有映射、`keep` 保护语义或无法证明 existing-target compliant 的 scope 仍 stale → reopen/revision/re-review；不得容许同一 locale 内 Tour 与 Docs 术语冲突。

历史 Snapshot 没有 entry-level glossary dependency，不能伪造当年的依赖记录。新的 compatibility evidence 重新分析可验证历史/当前 context，保留旧 receipt 与真实 full SHA，通过独立 lineage 接受 unaffected evidence，不修改旧 evidence。

## 8. whole-SHA/config freshness debt：V2-B scope

下表保留 V2-A 的 coupling 审计及 V2-B 处理边界；现行实现、CLI、档案路径、schema v4 与验证规则以 [正式 V2-B runbook](GLOSSARY_COMPATIBILITY.md) 为准：

| 当前绑定点 | 实际 code / coupling | V2-B 处理边界 |
| --- | --- | --- |
| Full Glossary Review / legacy coverage | `glossary_review.go` receipt 与 legacy entry 的完整 glossary SHA；`glossary_review_bundle.go` full byte/input identity | **保留 full currentness**，新 full Review 永远不可绕过 |
| Candidate Snapshot | `candidate_snapshot.go` manifest `glossary_sha256`；`retranslation_review.go` snapshot reader 对当前完整 bytes 比较 | immutable historical Snapshot + 独立 compatibility lineage |
| QC carry-forward / effective lineage | `quality_check_results.go` `BuildQualityCheckScope` glossary_changed 全量 reopen；effective results 仅 whole SHA 相同才承接 | affected exact-set；unaffected 有证明才继承 |
| QC reviewer bundle / reviewer identity | `quality_check_bundle.go` glossary bytes、snapshot/authority inputs、recomputed input identity | bundle 本身仍绑定真实 inputs；新增明确 compatibility context，不能改旧 bundle |
| QC finalization / promotion | `quality_check_finalization.go` 全 Snapshot/glossary/QC lineage 重新计算；`retranslation_promote.go`验证 finalization/latest candidates | lineage-aware current verifier；新工作仍 A-only，旧 finalization 不重写 |
| Revision/retry authorization | `retranslation_export.go` effective snapshot glossary equality、saved input stale provenance；`generation_bundle.go` full glossary/authority transport identity | exact affected stale/recovery，current Generation bundle 继续 full inputs |
| Locale-level generation / Course localization bundles | `locale_generation_bundle.go`/`course_localization_bundle.go` glossary file/input identity、`course_maintenance_bundle.go` rederived full formal input | 新 full input transport 不弱化；与历史成果复用分开 |
| Course SEO | `course_metadata.go` 每 page `glossary_sha256`；load/stale/refresh/revise 与 full current glossary 比较 | 按 page source/target/SEO context 精确 compatibility，不机械全课程 refresh |
| Locale Surface A receipt | `surface_review.go` `GlossarySHA256`、UI/full metadata/course metadata、catalog、project/SEO config、source-description authority | full language Review 与 surface-specific compatible downstream gates 分离 |
| Surface package / Reviewer Bundle | `surface_review_export.go` glossary full text、page glossary SHA、current metadata加载；`surface_review_bundle.go` rederived input/files identity | compatible artifact 读取与 surface projection；不假装旧 bundle current |
| project.go / seo.go whole-file | `surface_review.go` `ProjectConfigSHA256` / `SEOConfigSHA256` whole-file hashing | surface-specific public copy/route/SEO projection identity，Tour unaffected 不受无关新 Site 代码影响 |
| Registry / Production config | Surface schema v1 whole Production file；v2 target identity + registry baseline；v3 target/English/runtime projection + append-only registry compatibility，whole registry SHA 仍历史记录 | 复用已存在 projection/compatibility，不能退回全文件耦合或弱化 target identity |
| Ready candidate / downstream export/build | `projection.go` 与 `surface_review_export.go` 当前 glossary validator、Course SEO loader 的间接 freshness；`sitecontent` bootstrap/current context 也复用这些 gates | compatibility 必须贯通 consumer，不仅“优化 QC”一个点 |

V2-B 要覆盖上述 Snapshot reader、scope、effective QC、reviewer/finalization、promotion/revision authorization、Course SEO、Surface export/bundle/A gate 与消费者；保留 full Review 与新 Generation transport 的精确 currentness。config projection 的实质公开文案/identity 改变仍需独立 Surface Review，不以 namespace refactor 隐藏变更。所有历史 evidence 保持原始 bytes/provenance，新增 compatibility lineage 与 projections，不做 schema receipt retroactive rewrite。

shared-authority cleanup debt：`internal/sitecontent` 当前暂时复用 `internal/tour.FrozenUpstreamCommit`。在 V2-B freshness projection 完成前不为中性化常量改动 `internal/tour/project.go`，避免制造新的 whole-file Surface stale；后续共享 authority cleanup 不属于 V2-A repair。

## 9. Advertising / publication

`internal/tourpolicy` 是 publication 与 advertising classification 的共享正式 authority。两个维度彼此独立：`Publication` 继续决定 GoLocal publication / owner-content/navigation semantics；`Advertising` 单独决定广告资格。不得为了无广告把 locale 伪装成 `GoLocal`，也不得从 locale tag、语言名称、广告平台列表或网络查询动态推断 advertising classification。

维护者于 2026-10-07 正式冻结 `AdsUnsupported` explicit locale set：`sw-TZ`、`kk-KZ`、`fa-IR`、`am-ET`。四门的 publication 仍为 `Standard`。`GoLocal` 仍且仅为 `zh-CN`、`fr-FR`、`de-DE`、`ko-KR`。

| 显式 advertising policy | `/` | `/translation/` | `/tour/**` | 正式 Learn / Docs / 其他 Site v2 pages |
| --- | --- | --- | --- | --- |
| `AdsUnsupported`：`sw-TZ`、`kk-KZ`、`fa-IR`、`am-ET` | 无广告 | 无广告 | 无广告 | 全站无广告 |
| `GoLocal`：`zh-CN`、`fr-FR`、`de-DE`、`ko-KR` | 无广告 | 无广告 | 无广告 | 允许广告，仍满足现有 GoLocal agreement |
| `Standard` | 无广告 | 无广告 | 允许广告 | 允许广告，除非更高显式 policy 禁止 |

优先级：**AdsUnsupported > GoLocal > Standard**。Site v2 `AdPolicy` 与旧 Tour runtime 都投影同一 `internal/tourpolicy` advertising authority，不维护第二份 unsupported locale list。`/translation/` 和其他站内页面不得绕过 GoLocal Tour policy；`AdsUnsupported` 则在所有 current/future local surfaces 禁止广告。原“unsupported explicit locale set 尚未冻结”的 Production blocker 已解除；machine/browser/first-production gate 必须按该共享 authority 验证。

## 10. Upstream contract 与 deferred V2-C1

V2-C1 唯一一次正式 upstream source sync **deferred**。V2-C1 前，`golang/website@db076098077c07d3cef1b85a2cf56ff52777f587` 是本项目主动选择并冻结的正式 English source authority；architecture、runtime、localization campaign、independent review、package activation、Preview 与 Production 均允许基于该 baseline 推进。这不表示本地 checkout 等于今天官方最新 upstream，也不实时跟踪 master。本节是 frozen tracked upstream issue set 的 canonical authority：

- `golang/go#81336`
- `golang/go#81482`
- `golang/go#81596`
- `golang/go#81958`

正式 gate 必须同时满足：以上四个 issue 全部达到维护者预期完成状态 **并且** 维护者显式重新打开 V2-C1。部分完成不自动重开阶段。重开前不 fetch/pull/sync，也不为判断 master 漂移临时同步；不增加每次 Production 前联网 drift check，不把最新 upstream 状态作为 locale Production blocker。只按 frozen source/package identity 完成正式 gate。

```text
65 Tour locale live + IndexNow closeout complete
→ V2-A Architecture / Content Scope
→ V2-B compatibility / freshness gates
→ V2-D runtime / localization workflow foundation
→ 正式模型 benchmark
→ 可连续执行完整 learn-docs-v1 locale campaign
→ 按 frozen baseline 完成 Preview / Production
→ 4 个 issue 完成 + 维护者显式重开 V2-C1
→ V2-C1 唯一一次 upstream source sync
→ exact source/package/content-unit reconciliation
→ 已完成 / 已上线 / 在途 locale 仅处理真实 stale 范围
```

当前 pre-real-data 66-locale commercial execution order 已由唯一 planning authority [Locale 路线图](LOCALE_ROADMAP.md#5-跨项目-66-locale-广告收入潜力默认执行顺序) 冻结，Site v2 campaign 现在已有确定默认顺序。当前 Go source 仍为官方 generic `lang="en"`，仅在 commercial scheduling 层视为覆盖并跳过 rank 1 `en-US`，第一个实际商业 target 为 rank 2 `zh-CN`；不新增 `en-US` Go locale，不改变 current 65 Production identity、frozen English source authority 或 V2-C1 deferred 条件。V2-C1 后有足够真实运行数据时，仍可按路线图执行一次 empirical re-ranking；它与当前冻结执行顺序分离，此前不反复调整基础顺序。

V2-C1 后逐 package/file/route/dependency/content-unit 比较；unaffected translation/review/QC/SEO/Surface 继续复用，使用 V2-B glossary compatibility/freshness lineage，仅 affected scope reopen/revision/re-QC/Surface refresh。historical evidence 不改写，全量重译不是默认恢复方案，whole upstream commit 变化不自动使所有已上线 locale stale。route change 显式映射，ambiguous 停止人工确认。

Tour 的 persistent page identity、`present.Section` TranslationUnit 与 source-stale contract继续保留。Learn/Docs Markdown/HTML/YAML 使用 surface-specific parser/content-unit identity。V2-A snapshot 是 pre-V2-C1 正式冻结 baseline；基于它通过全部 gate 的 locale 不因 V2-C1 尚未执行而不合法。

## 11. V2-A deterministic commands 与停止点

```bash
go run -mod=readonly ./cmd/tour-i18n content-scope init --source-root "$HOME/code/go-website-upstream"
go run -mod=readonly ./cmd/tour-i18n content-scope inventory-check --source-root "$HOME/code/go-website-upstream"
go run -mod=readonly ./cmd/tour-i18n content-scope bootstrap-tour --all
go run -mod=readonly ./cmd/tour-i18n content-scope check --locale he
go run -mod=readonly ./cmd/tour-i18n content-scope check --all
go run -mod=readonly ./cmd/tour-i18n content-scope status --locale he
```

init 仅在 global/snapshot 都不存在时创建；bootstrap 不覆盖不同 state；check/status 不写文件、不依赖时间/网络/外部 checkout。inventory-check 是可选只读 frozen-checkout 对照，普通 current-check 不需要它。V2-A 完成即停止，不自动进入 V2-B/runtime/翻译/sync/Production。

本次未提交 implementation 的定向修复命令：`content-scope repair-uncommitted-v2-a --all`。先从既有 ZIP 重算 global，再 preflight 65 个真实 legacy closure；仅在全部输出仍为 Git untracked、原 evidence/context 完全不变时替换 global/locale scope。ZIP 不重写，tracked authority/历史审核文件不覆盖；这不是未来 upstream sync 或 activation workflow。

## Accepted TranslationUnit target successor authority

本节保留原 native/historical installation event 的 audit/recovery 合同；shared current TU checkpoint 与 current canonical binding 的正式能力见[历史完整性与当前授权](CURRENT_TRANSLATION_AUTHORIZATION.md)。historical finalization integrity 不要求 current glossary/config equality；current eligibility 独立要求原 exact A candidate、current source、full Unified Review、semantic delta 与 validator。已建立 checkpoint 的 current consumers 不再穿越旧 compatibility/mutable-state lineage。旧65-locale记录保持兼容，不做 mass migration，不改变 Production gates。

Course SEO 是独立 derived consumer，不扩展 TU successor。历史 Course SEO archive/recovery WIP 保留、停止扩展且不 apply；它不再是 Site v2 主路径。旧 closure、Surface gate、QC、Review、compatibility 均不修改。

合法 `legacy-tour-closure/v1` completion root（locale/package/source、complete、routes/surfaces、safe refs、EvidenceIdentity/ContextIdentity）加上 exact-current、passed 的 schema-v4 Surface A，可组成一次 legacy → modern current Tour checkpoint。checkpoint 不持久化新 schema，复用现有 gate，并独立要求 current full Unified Glossary Review、当前 glossary-bound A-only finalized TU Snapshot、canonical authorization 与 strict-current Course SEO。任一缺失/stale/冲突则不采用 checkpoint，保留原 fail-closed 路径。此时不再为当前语言质量恢复旧 mutable UI/SEO/registry/runtime bytes 或重建完整原 Surface package digest；历史记录仍是 completion trust root。

这是一次 baseline reset，不是每次变化必须 full Tour re-review。随后 Site v2 shell / Learn-Docs 仍走 package closure 与 Integrated Surface：unchanged Tour carry，new Generation/QC，changed/affected exact reopen；full machine closure 不等于 full human re-review。

历史 accepted-target transition 与 current glossary language authorization 是两层 authority：原 receipt/recovery 以其原 finalized Snapshot 的 archived glossary context 验证，后续 compatible refresh 不撤销历史事件；当前授权另要求完整 CURRENT Review、逐 TU compatibility 与绑定当前 glossary 的 finalized Snapshot。100% predecessor A carry-forward、0 新 Reviewer Unit 可用显式 `quality-check finalize --previous-snapshot-id` 完成，不伪造 current QC response，不改写历史 judgment。

历史 `legacy-tour-closure/v1`、compatibility、Snapshot、QC results、finalization、Surface/Production evidence 均 immutable；promotion 不重写旧 closure。Reviewer 的历史 A judgment 与 current canonical authorization 是不同事实。新增 additive successor contract 的严格 Go schema authority 为 `internal/i18n/target_successor.go`：

- `go-learning/native-promotion-receipt/v1` / `native-promotion-receipt` / `formal-promotion-apply`：未来正式 apply 的原始 machine evidence。先完整 staging，canonical/status 安装且 exact 输出复验成功后，exclusive link 发布到 `locales/<locale>/candidates/.promotion-<identity>.json`；发布前 staging JSON 不具有 current authority。
- `go-learning/historical-promotion-recovery/v1` / `historical-promotion-recovery` / `missing-original-promotion-receipt`：旧实现未保存原始 receipt 时的维护者显式历史恢复，位于 `data/promotion-recoveries/<locale>/<identity>.json`。它**不是原始 apply receipt**，不声称保存了过去事件；仅授权从 immutable baseline 到 finalized selected target 的 exact successor relation。

两者绑定 exact locale/Snapshot、Snapshot/finalization/QC refs、完整 Unit membership、source、glossary、selected batch/candidate、validation、输入/raw evidence 与 resulting status。`resulting_status_bytes` 保留完成/恢复时的 exact status 内容，验证其 SHA、exact Unit membership、source/target/attempt；native 还校验正式 promotion note/time 形状。它是 immutable 历史投影，不是 mutable `status.tsv` 当前 SHA 的永久 TU 授权条件。Recovery 的 `units` 只含真实 old→new changed Unit；native receipt 包含全部 changed/unchanged Unit（首次安装显式 `old_target_present=false`）。old baseline 必须由 closure SHA 与其绑定的历史 candidate 重建；native 保存 pre-apply old bytes/SHA。验证时重新执行 source/structure/parser、protected restore、validation、A-only/carry-forward 与 compatibility lineage，不允许 current bytes 自证。

`selected_candidate.sha256` 始终绑定原始 finalized candidate；`new_target_sha256` 绑定正式 promotion 的精确输出。Page 沿用既有、唯一的 deterministic EOF normalization，Example 不变；同时验证原始 candidate exact SHA 与 canonical exact 等于该投影，不允许其他 byte difference。self identity 是清空 `identity_sha256` 后 Go canonical JSON 的 SHA-256，plan identity 是 exact `units` canonical JSON SHA-256；文件本身 SHA 可独立重算。Recovery 不读取当前时间，同一组绑定输入（包括已有 recorded status bytes）产生相同 identity。读取拒绝 duplicate/unknown/null fields、traversal/symlink；保存使用完整临时文件 + exclusive link，no-overwrite。

旧 mutable `status.tsv` 不是 Reviewer judgment；historical recovery 单独绑定 exact resulting status，旧 status ref 仅在它确实属于绑定 closure 且当前仍为该 resulting projection 时接替。后续合法 status 更新不撤销 TU accepted-target relation，但不借此授权任意新 status 或其他消费者。

仅在 TU successor **historical event verification** 中，recorded non-TU mutable refs（UI、SEO/article、registry/config、runtime/template/partial 等）不再要求今日 filesystem 永久等于旧 SHA。artifact schema/identity 与完整 `proof_inputs` exact-set 不变；这些 refs 的历史 membership/SHA 从不可改写的 closure/compatibility inventory 交叉验证。closure 文件、TU source、Generation input/raw/manifest/result/candidate/validation、Snapshot/QC/finalization、archived glossary 与 immutable compatibility evidence 仍 exact；compatibility 的 strict schema/self identity/delta/classification/lineage 仍重算，只重新绑定 `tu:` context exact-set。old canonical 仍由 closure-bound Snapshot/candidate 与唯一 EOF normalization 重建，current canonical 仍 exact 等于正式 successor。TU successor 不授权 SEO、UI、Surface、runtime/config 或其他 derived consumer；各自 currentness 由独立 workflow 判断。

`.promotion-staging-transaction` 由 exclusive mkdir 建立，串行化每门 locale 的 apply。它与旧 `.promotion-staging-*` 残留共同构成 pending/unrecovered transaction guard：current successor verification、campaign-plan、promotion 与 historical recovery 均 fail closed，不自动清理或重放。mutation 开始后 panic/process interruption 保留 backups；rollback 无法完成时报告 `mutation=unknown`。成功收口或完整 rollback 后才清理本次 staging；不宣称跨路径 crash-atomic 或断电 durability。
