# 共享翻译单元基础设施阶段交接

结果：PASS。阶段：共享 TranslationUnit 生命周期、五角色协作与执行效率优化。locale：`zh-CN`；内容包：`learn-docs-v1`。这是仓库工程与机器处理，不是正式语言生成或审核。

## 架构与行为

共享核心保留在 `internal/i18n`，通过 `TranslationUnitAdapter` 接入页面受保护输入、还原与结构/术语校验；Page JSON 的原 invocation 来源通过 `ArchivedGenerationAdapter` 解码验证。Tour 既有 present / Example 实现保持兼容，默认 Tour 调度不被其他表面的注册扩大。Learn/Docs 只保留源发现、完整页面表示、源坐标解析/保护/还原、结构校验、固定批次与路由上下文，内部槽位不升级为正式翻译单元。

复用既有 `retranslation export`、Generation Bundle current-check/import/recover、`process`、`retry`、最新结果选择、候选快照、独立质量审核、精确修订、再审核及 A-only finalization。内容包 closure 接收共享终结证据并检查当前候选一致；旧 Site CLI 不再新增独立审核/修订/终结记录，历史读取兼容。

未新增 schema 或 Site-specific recovery / partial acceptance receipt。只在既有 manifest 扩展可选 `generation_origin`，保留原完整 Bundle/result 的内容寻址归档，并在既有审核结果中添加可选 `reviewer_session`。原 JSON 不能由正常新 Generation Bundle receipt 如实冒充原模型输入，来源绑定解决的是表示适配与证据完整性，不新增生成 invocation。所有历史证据保持原布局。

传输导入检查安全、真实 provenance、identity、精确集合、文件格式与不可覆盖；逐单元语义校验由共享处理完成。29 通过、1 失败时保留29个成功候选及证据，仅失败页进入正常精确重试；重试成功得到30/30。批处理一次报告所有独立失败 ID/原因，不在第一个失败处停止。包级 identity 变化不自动使每页失效；沿用要求精确正式输入及候选不变、当前机器校验与当前完整术语审核通过。

机器调度：第一批30/30 → 第二批19/19 → 完整49页快照 → 第一批初审 → 第二批初审 → 集中 B/C/D 修订 → 精确再审核 → A-only。提前导出下一生成批次、建立不完整快照或首轮尚未全部完成时常规修订均受共享入口限制。合成回归同时验证48个未变 A 沿用、1个精确修订再审核；真实工作树没有执行质量审核。

五角色、中文优先、恢复/复用优先、唯一下一步、短机械自动完成、长低 reasoning 操作交 Local terminal、可预测 read/hash/check 合并、少轮询及完整 Bundle 下 Prompt 简化已进入[共同 authority](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)，并链接到 AGENTS 与全部正式执行入口。角色说明不新增机器状态或替代任何 gate。

## 当前真实状态与 mutation

第一批正式机器结果：[chatgpt-zh-CN-022/result.json](../data/retranslation-runs/zh-CN/chatgpt-zh-CN-022/result.json)：30个页面，还原通过30、校验通过30、失败0。

`manifest.json` SHA-256：`3800b790ac2a05198f8c1972a0f3c405ca4e62e3b47e568f195ab287eeeacd07`。

`result.json` SHA-256：`069b73cc57f8aec5a5e9ec2b6e3e04adbe9a6456fe6dcf7c9c9fc2b835f24239`。

共享进度只读核对：49页中30通过、19尚未生成、还原/校验失败0、过期0。只读生成计划仅包含固定第二批19页。第二批未导出新的共享生成批次、未开始模型生成；第一批独立质量审核尚未开始，没有创建 Learn/Docs 审核快照或审核结论。

`mutation=known`：新增完整原输入/结果的8份归档至 `data/generation-archives/zh-CN/`，以及正常共享批次022的 manifest、inputs、raw responses、30份还原候选、validation 与 result；未覆盖已有候选、未修改 zh-CN 措辞，没有新语言调用。staging 不再作为长期来源 authority；正式 manifest 绑定归档。

原 `data/site-workflow/zh-CN/learn-docs-v1/generation/page-batch-1-recovery-1.json` 处理前后均为 SHA-256 `676b7d7a48657665778c8c6af8d6b0182ce3f2e68240fe55ad1060822b03ae42`，逐字节保持不变。未改写 Tour historical batch、raw responses、validation、Snapshot、QC、finalization、promotion、Surface 或 Production。未 stage、commit、push；既有大量合法 WIP 原位保留。

## 修改范围与验证

本轮修改 AGENTS、正式工作流/任务/重译/ChatGPT/Codex/Site/新增 locale/交接文档及本文；CLI 修改 `cmd/tour-i18n/main.go`、`site_content.go`。共享实现涉及 `catalog.go`、`example.go`、`status.go`、`translate_protect.go`、`validate.go`、`generation_bundle.go`、`retranslation_export/process/retry/latest/promote/review`、`candidate_snapshot.go`、`quality_check_bundle/results/preflight`、`target_successor.go`；新增 `surface_adapter.go`、`translation_progress.go`。页面适配涉及 `workflow.go`、`planner.go`、`package_closure.go`，新增 `translation_adapter.go`、`archived_translation.go`。对应夹具和定向测试同步修改，新增 `surface_adapter_test.go`、`shared_processing_test.go`、`shared_lifecycle_test.go`。整个 Git diff 包含以前的 WIP，不应误认为全部属于本轮。

定向测试通过：Tour 初次处理、失败精确重试、既有生成恢复（目录/文件/归档、冲突拒绝）、导入安全与语义分层、Tour 历史 A-only/全沿用证据、默认 Tour 调度兼容；页面29/1保留/恢复/精确重试、完整多失败集合、完整快照门、独立角色门、固定30/19审核选择、首轮集中修订门、精确再审核与共享 A-only；源坐标、标题后缀、链接绑定、教学注释、HTML/Markdown 结构和强调语法；相关 CLI 参数及失败摘要。普通混合评级快照的审核 A 沿用不要求该旧快照已经终结；零审核全沿用仍严格绑定前一不可变 finalization。

显式真实回归 `GO_TOUR_CURRENT_PAGE_REGRESSION=1` 在临时目录处理原30/recovery-8/2/1并得到30/30，真实仓库再以单次 `generation-land` 收口；未运行全仓库测试、浏览器回归、build、publish 或 Production。首次无批次目录、合成 revision 夹具漏填 provider，以及普通混合评级快照错误要求先终结的边界，均已修复并通过定向复测；没有放宽 A-only 或零审核沿用证据门，无未解决失败。最终 `git diff --check` 通过。

## 唯一下一步

执行者：Maintainer Local terminal。动作：为第二批19页导出共享完整 Generation Bundle，执行 current-check 并打印 SHA-256，交既有 `chatgpt-zh-CN-generation` 会话。下一正式语言步骤是第二批生成，不能开始第一批 QC。停止条件：完整19页 Bundle current-check 通过并完成附件交接；本段不启动模型或审核。如果固定批次023或目标文件已存在，命令立即停止，先检查真实状态，不盲目重复写入。

以下命令**尚未执行**；路径与批次由本轮实际状态确定，无 placeholder：

```sh
(
  set -e
  cd /home/wangqiang/code/go-tour-i18n
  test ! -d data/retranslation-runs/zh-CN/chatgpt-zh-CN-023
  test ! -e .site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-export.json
  test ! -e .site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-generation.zip
  go run -mod=readonly ./cmd/tour-i18n retranslation export \
    --locale zh-CN --generator chatgpt --content-package learn-docs-v1 --json \
    > .site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-export.json
  rg --quiet '"batch_id": "chatgpt-zh-CN-023"' .site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-export.json
  rg --quiet '"unit_count": 19' .site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-export.json
  go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
    --locale zh-CN --batch-id chatgpt-zh-CN-023 --content-package learn-docs-v1 \
    --output /home/wangqiang/code/go-tour-i18n/.site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-generation.zip
  go run -mod=readonly ./cmd/tour-i18n generation-bundle current-check \
    --content-package learn-docs-v1 \
    --bundle /home/wangqiang/code/go-tour-i18n/.site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-generation.zip
  sha256sum /home/wangqiang/code/go-tour-i18n/.site-v2-staging/zh-CN-learn-docs-page-batch-2-shared-generation.zip
)
```

附件 Prompt 只需说明：原 Generation 角色；完整读取 current Bundle；精确第二批19个完整页面；沿用既有 provider/model；输出完整受保护单元到独立隐藏 staging；不审核、不改代码、不放宽保护；停止于输出落盘与共享导入/逐单元校验及必要精确重试。不得缩减 Bundle 或把机器通过写成 A。

## ChatGPT Project Instructions 同步建议

以下可整段复制；没有修改 ChatGPT UI：

```text
项目 Go Learning & Documentation Translations 使用仓库当前 authority。Tour、Site shell、Learn、Docs、未来内容包中的所有正式 TranslationUnit 共用一套共享生命周期：生成→传输/导入→逐单元还原/机器校验→失败精确重试→完整有效集合→候选快照→独立初审→精确修订→独立再审核→A-only终结→内容包/Surface/promotion。一个物理页面是一个正式页面翻译单元，内部槽位不是单元；仅解析、保护、还原、结构和来源上下文使用表面适配器。

固定五角色：会话协调负责状态、唯一下一步、短机器操作和交接，不做语言判断或改代码；Generation长期会话负责语言生成/重试/修订，不自审或改基础设施；独立Reviewer负责术语、逐单元及Surface审核，不生成replacement或改候选；Codex负责仓库工程、validator、CLI、测试/docs，未经明确授权不生成语言、不审核；Maintainer Local terminal负责长时间、低reasoning、批量确定性操作、最终机器验证/HUMAN gate和Git，节省calls与Codex额度。

中文优先：说明、状态、失败分析、docs、交接和提交说明默认中文；代码标识符、CLI、schema/field、路径、locale、ID/SHA、provider/model和canonical source保留原文。每次交接只给一个执行者、动作、停止条件；架构分歧先由维护者决策，不能提前实施某个方案。

恢复优先：先检查已有staging/Bundle/result/provenance/raw/validation/快照/审核/pending。可信单元不重做，仅处理missing、restore_failed、validation_failed或真实stale/affected；mutation unknown停止，不能盲目重放。复用优先：新增schema/receipt/状态机/loader前解释现有共享机制的真实缺口，先评估adapter、extension、helper、interface和既有evidence，不按目录复制生命周期。

Learn/Docs固定先第一批30页生成与机器校验，再第二批19页生成与机器校验，全部有效后才开始两批独立初审；两批首轮全部完成后集中汇总B/C/D并精确修订、独立再审核。机器通过不等于A；历史evidence不迁移或改写。

短且可以少量调用安全完成的机械操作连续自动执行；长测试、成组build/preview/deploy/verifier、批量checksum、最终Git优先交Local terminal，命令必须完整可复制、无placeholder，不要求用户推断变量。合并可预测read/hash/status/check，避免重复扫描Bundle和长进程轮询；简单阶段预计超过约5次calls时先考虑合并、正式CLI/helper或本地终端。

已有完整current Bundle时，Prompt仅强调角色、精确范围、任务、finding/特殊缺陷、禁止项和输出；仍完整读取Bundle、manifest、全部inputs和术语表。calls/额度优化不能降低模型政策、独立会话、完整输入、provenance、current/exact-set、自动校验、A-only、Production或HUMAN gate。代码默认只跑直接相关targeted tests与git diff --check，不自行扩大回归范围。
```
