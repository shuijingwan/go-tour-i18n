# Versionless Locale / Site v2 Workflow Foundation

本文是 V2-D 的 executable contract。v1/v2/v3 是 implementation/content-package/source-contract milestone，不是 locale language governance namespace。**Content package 可以版本化；Locale language system 不版本化。** package 划分 source、coverage、completion、activation 与 reuse，不划分 glossary、locale identity、术语治理、Generation/Reviewer session 或 provenance。

## Source 与统一 terminology authority

pre-V2-C1 正式 English authority 是主动冻结的 `golang/website@db076098077c07d3cef1b85a2cf56ff52777f587`，不是 latest upstream。campaign / Preview / Production 不被 deferred C1 阻塞，不增加联网 drift gate。四个 tracked issue 达到维护者预期且显式重开后，仅一次 C1 sync；此处不执行 sync。

`data/site-content-scope.json` 与 frozen `data/site-content-sources.zip` 继续管理 source/package closure。49 Page + 7 data = 56 localizable documents；6 redirect + 18 asset 不进入语言 TU。shell 两个正式 English source 是 `data/site-shell/home.html`、`translation.html`，local canonical `/`、`/translation/`；V1 homepage不能冒称 shell PASS。

唯一 locale glossary：`locales/<locale>/glossary.yaml`。mandatory/preferred/forbidden/keep 四类不变，禁止 Tour/Docs/Learn/v2/v3/core/extension/shadow glossary。

Unified Glossary Source Corpus schema：`go-learning/glossary-source-corpus/v1`，authority：`data/glossary-source-corpus.json`。contributors 同时是 shared structured consumer 的 source registry。机械 projection 使用各自正式 parser：103 Tour Page、19 standalone eligible Example（仅允许的自然注释）、113 English UI visible values、7 canonical article title/subtitle、103 canonical Course SEO descriptions、2 shell、7 Learn visible YAML、49 Learn/Docs Page。当前 403 contributors / 3,004 projected contexts / 373,363 UTF-8 context bytes；binary资产、machine-only代码不作为语言文本。protected technical context保留为只读术语判断材料。future surface必须 additive注册并由正式 parser投影，不能另建 glossary generation。

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

glossary改变之前归档原bytes；new full unified PASS之后，V2-B archived old/new → semantic delta → exact affected scope → immutable compatibility lineage。Docs-only term不会自动重翻旧Tour Page；shared core term只reopen真实affected TU/asset。unknown impact fail closed；历史A、Snapshot、glossary Review、Surface与provenance不改写。

## Unified Structured Assets

contract `go-learning/locale-structured-assets/v1` 使用corpus contributor的structured属性，当前11个complete-file assets：Tour UI、article metadata、2 shell、7 Learn YAML。new locale一次 composite invocation生成所有registered outputs；existing Tour locale机械验证旧completion/source/target/glossary lineage，current Tour UI/meta carry，只输出generation-required pending assets。

```sh
go run -mod=readonly ./cmd/tour-i18n site-content structured-plan --locale <locale>
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export --locale <locale> --task locale-assets --output /tmp/<locale>-structured.zip
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-check --bundle /tmp/<locale>-structured.zip
go run -mod=readonly ./cmd/tour-i18n site-content structured-import --bundle <repo-relative.zip> --input <repo-relative.json> --provider <chatgpt|codex> --model gpt-5.6-sol-high --session <generation-session> --generated-at <UTC>
```

模型只输出plan.Expected exact set，不重新输出carry。Tour UI/metadata使用原schema validator；shell/YAML输出完整slot target set，受parser重建，不允许任意目录绕过保护。composite evidence记录一个真实Generation session/provenance，subassets仍保持source/target/package/validator边界。structured asset不另建逐-TU QC；finding回原Generation → exact replacement → validation → affected Surface re-review。

Course SEO是Page-derived consumer，canonical English description + final ready Page + glossary决定scope；不强行并入initial structured generation。existing current Tour SEO carry；本轮没有正式Learn/Docs canonical SEO source authority，不生成虚假SEO contract。future canonical consumer接同一locale/provenance/compatibility framework。

## 完整 Page TU / internal slots / fixed batching

internal parser `go-learning/content-units/v2` 保留Markdown Goldmark AST、HTML tokenizer、窄Learn YAML schema；internal linguistic slots用于保护、精确impact、deterministic restoration与reconciliation，**不是formal TU**。当前56 documents有2,646 internal slots；min/median/p90/max = 2/84/269/691 UTF-8 bytes，<=20=383，punctuation-only=0；最大gomod-ref 207 slots。slot packing仅internal diagnostic，CLI summary明确标记diagnostic，不是模型调用plan。

formal Page contract：`go-learning/page-translation-unit/v1`；workflow evidence：`go-learning/package-workflow/v2`。49 physical Page = 49 formal Page TU；embedded CodeExample=0，7 data不进入content-QC。完整Page protected input以block-boundary和inline locked bindings还原；一个Page一个candidate、一个A/B/C/D rating。safe fenced/pre Go teaching-comment body属于Page；复用Tour conservative fragment analyzer，code/delimiter/identifier/layout/directive/unknown code受保护。Heading `{#create_folder}`、inline code、Markdown link destination及label-target binding完整保留；目标语言可合法移动完整inline bindings，不能移动Page结构边界。

stable append-only Page authority：`data/learn-docs-pages.json`，schema `go-learning/stable-page-index/v1`。首次按learn/tutorial/database/modules/security，surface内path lexical order冻结1–49。future append50–60进入batch2；delete tombstone不复用；move/rename必须explicit mapping，unknown/ambiguous不猜。index61 fail closed，交维护者决定batching authority，不自动第三批、不rebalance。

正式initial Generation与QC各两次独立invocation：

| batch | stable membership | current Page count | raw source/context bytes |
| --- | --- | ---: | ---: |
| page-batch-1 | 1–30 | 30 | 267,539 |
| page-batch-2 | 31–60 | 19 | 150,854 |

每组最多30 Page；通用60 defensive ceiling不授予合并两组。16 MiB text/context/output是transport hard safety limit，不是packing algorithm；超出停止、报告evidence，不自动拆成几十组。

```sh
go run -mod=readonly ./cmd/tour-i18n site-content page-plan --package learn-docs-v1
go run -mod=readonly ./cmd/tour-i18n site-content generation-plan --locale <locale> --package learn-docs-v1
go run -mod=readonly ./cmd/tour-i18n site-content generation-export --locale <locale> --package learn-docs-v1 --batch page-batch-1 --plan-batch 1 --output <repo-relative.zip>
go run -mod=readonly ./cmd/tour-i18n site-content generation-check --bundle <repo-relative.zip>
go run -mod=readonly ./cmd/tour-i18n site-content generation-import --bundle <repo-relative.zip> --input <repo-relative.json> --provider <chatgpt|codex> --model gpt-5.6-sol-high --session <generation-session> --generated-at <UTC>
go run -mod=readonly ./cmd/tour-i18n site-content review-plan --locale <locale> --package learn-docs-v1 --generation <artifact.json>
go run -mod=readonly ./cmd/tour-i18n site-content review-export --locale <locale> --package learn-docs-v1 --generation <artifact.json> --plan-batch 1 --output <repo-relative.zip>
```

batch2使用plan-batch2；revision `--task revision --finding <B/C/D review>`机械选exact Page；glossary-revision使用verified affected evidence。Surface finding 使用 `generation-export --task surface-revision --finding <failed Surface receipt>`，scope来自机器验证的 `exact_findings`，不是从自由文本猜ID。revision ZIP包含 exact prior candidates。new Generation永远绑定current full source/package/glossary/unifiedReview/parser/authority；historicalaccepted evidence只按selected Page/structure/target/glossary lineage比较，另一document变化不能抹掉旧A。

Reviewer只findings/rating，不能replacement；同locale Generation/Reviewer长期session完全独立，policy仍GPT-5.6 Sol High。normalized Reviewer Bundle每Page context一份、Rows为完整Page input/target/ref/provenance；不按slot重复Document。B/C/D → 原Generation exact Page revision → validation → independent re-QC；unaffected A carry。finalize要求49 current Page、A-only、pending0、unresolved0、完整真实provenance和exact lineage。

Tour成熟present.Section/standalone eligible .go流程不重写，仍103 Page+19 Example、60/43/19 initial batches与独立QC。site milestones不改变Tour semantic authority。

## Locale plan / integrated delta-aware Surface / activation

`site-content campaign-plan` 是共享versionless locale work plan；scope分类carry/generation-required/review-required/stale-reopen/retired/ambiguous基于真实source/target/QC/Surface/compatibility evidence。exact oldTour source/target/quality/publiccontext证明后carry，新Page/structured surfaces待生成；package数量或milestone不影响classification。

`go-learning/package-language-closure/v1` 原子绑定49 Page finalization + 7 structured data，或2 shell structured files。`site-content package-closure`用`--input <Page finalization>`和repeatable `--generation <structured evidence>`；shell不需要Page QC。partial file/subdirectory不能complete。

`go-learning/integrated-locale-surface/v1`区分carried与review-required scope，绑定existing complete coverage、prospective closures、全部active routes、current unifiedglossaryReview、machine public/runtime identity、prior evidence与new/changed targets。unchanged旧Page不重新rating；carry不能伪装Reviewer重新审过。Reviewer审核new/changed语言、structured assets、cross-surface terminology、导航、homepage/translation、coverage/canonical/sitemap/publicruntime/广告integration。old TU被affected时仍必须正式revision/re-QC，Surface decision不能代替它。

```sh
go run -mod=readonly ./cmd/tour-i18n site-content surface-scope --locale <locale> --generation <package-closure.json>
go run -mod=readonly ./cmd/tour-i18n site-content surface-record --input <independent-surface-result.json>
go run -mod=readonly ./cmd/tour-i18n site-content activation-preflight --input <package-closure.json>
go run -mod=readonly ./cmd/tour-i18n site-content activation-apply --input <package-closure.json> --prior-sha256 <exact-prior>
```

independentSurface receipt immutable且绑定Markdown hash；activation需要 current完整integratedPASS，locale completion保存closure+Surface refs；以后newpackage扩展需要新的integratedgate，原exact package的历史语言judgment仍保留。每locale独立，无跨locale barrier；production/identity.json不保存language/content状态。

structured finding 使用 `site-content structured-export --task revision --finding <failed Surface receipt> --locale <locale> --output <path>`，只选择 exact finding 对应资产；旧 accepted structured evidence 与 replacement 通过该独立 receipt 形成授权 supersession，closure保留未受影响输出。未知/重复/out-of-scope finding、stale context、Markdown hash/session mismatch停止。新 locale 没有 Content Scope record 时只推导 incomplete；首次 activation以空 prior SHA atomic no-overwrite创建 sparse record，不执行第二次 locale initialization。

existing Tour locale core expansion理想新增invocations（无revision）：Unified Glossary Refresh1 Generation、Unified Review1 Review、pending Structured1 Generation、Learn/Docs Page2 Generation+2 QC Review、Integrated Surface1 Review，即Generation4 / Review4 / total8。existing current Course SEO carry；若真实derivedmetadata需要新生成，单独报告该invocation，不隐含在8内。新locale仍onecampaign，额外Tour TU和derivedstage明确单列。

## Runtime / C1 / safety

保留已验收neutral opt-in `internal/site`：coverage由current Content Scope证据驱动；local covered canonical链接当前origin，未覆盖navigation直接go.dev；直接localmissing请求真实404。external URL原样；Security canonical `/doc/security/**`，legacyalias保持frozen exact/subtree fixed-target语义，只有targetlocalcovered才redirect。sitemap/canonical只含complete真实localcanonical，排除alias/data/assets/fallback。旧65Tour runtime不切换。

Advertising优先级 AdsUnsupported > GoLocal > Standard；`internal/tourpolicy` 是共享 classification authority。维护者已于 2026-10-07 冻结 AdsUnsupported exact set：sw-TZ/kk-KZ/fa-IR/am-ET，四门 publication 仍为 Standard；GoLocal 仍为 zh-CN/fr-FR/de-DE/ko-KR。homepage/translation 永远无广告，GoLocal Tour 无广告但 Learn/Docs 可按正式 Site policy 显示广告，AdsUnsupported 全站所有 current/future surfaces 无广告。原 unsupported-set Production blocker 已解除；不猜语言 tag、不联网推断、不另建第二份名单。

futureC1：source/file/route/dependency/content-unit reconcile → rebuildonecorpus → onerefresh/review → compatibility → exactnew/stalegeneration/QC → integratedSurface → activation。unchangedcarry、removedretire、newgeneration、changedPage stale、ambiguousexplicitmapping；不以wholecommit invalidation65locale。internal slots保留精确diagnostics，formalPage是review/revision boundary。

新machine evidence strict JSON、hash/selfidentity可重算、UTF-8、exact-set、traversal/symlink拒绝、immutable/no-overwrite。静态`site-content language-authority --apply`仅机械创建共享corpus与stable-index；不生成localeevidence。所有语言成果/65locale/prodidentity保持不变，本轮停止V2-D，不benchmark、不语言generation、不C1、不preview/Production、不commit/push。

正式 source authority更新后，`language-authority --prior-sha256 <exact corpus file SHA> [--apply]` 对静态 corpus做 preflight/CAS refresh；先验证 append-only Page registry，再重建 source projection，不执行 source sync，不改locale evidence。YAML item URL只是restoration coordinate，不进入 terminology context identity；machine-only code/image变动只有在projected context真正改变时才影响corpus identity。
