# ar Locale Surface Review — 2026-09-18 首次 production（r2）

## 审核身份

- locale：`ar`
- stage：`Locale-level language quality review (Stage A)`
- review id：`20260918-first-production-r2`
- reviewed repository HEAD baseline：`8a1ad30b3a17b34f9df87746acae61dd9e93126d`（`feat: 初始化阿拉伯语并支持 RTL`）
- reviewed working tree：包含本轮 Arabic RTL layout / splitter 修复、`welcome/1` 正式 TranslationUnit revision promotion，以及对应 schema v2 Course SEO refresh
- current review package：`/tmp/ar-surface-review.json`
- current package SHA-256：`8cd60683c260fb586fb81ab7898a45b2fd3149f1fd38507598944f1d213e6364`
- package size：`4257 lines`，`462293 bytes`
- reviewer：`ChatGPT GPT-5.6 Sol High`
- date：`2026-09-18`
- production state：`first-production`
- Stage A language quality review result：`passed`

## 当前正式输入 identity

- glossary：`locales/ar/glossary.yaml` — `348abf2c0915c6432b4c4f9d2a315178f0259ef2e82457c5b46255eded2eef46`
- English UI catalog：`internal/tour/ui/en.json` — `9a6e6877cd195144d6121f93cdabd387bdeb35c60a3f1afa9b8f417c7b6dabda`
- ar UI catalog：`internal/tour/ui/ar.json` — `3f12dce6fa2a60b76a329c78bf8cf4200866fd79b3b631513bf6f0be17cb5f79`
- article metadata：`locales/ar/article-metadata.json` — `860dd538c48dc66d088583322be2d05b0a04a95cd69536b2feda3f989635a65f`
- schema v2 course metadata：`locales/ar/course-metadata.json` — `d62d9db2b5ffd3c44a1ed72bdba626404e3761d9aad6672c2f924f8524495225`
- canonical English source descriptions：`data/course-seo/source-descriptions.json` — `1a46d981268ba75767986b3c212be06b6f678179f882efbc1813dedfad0a171d`
- canonical source-description review authority SHA-256：`9377e85b56bef45c5e1a1832186be7bc804816d0ac9149147db97267371b736f`
- catalog/source identity SHA-256：`9b6f9e5a5d698b4417aee5fc64239408461fb47876a69c3c803b3e51f25f285f`
- languages config SHA-256：`7496d738b046df47210083c62cd4eed6bc42778e509d8809ae28e8a4ee282fec`
- project config SHA-256：`78ae201af9207dbb74b266fb64614b82688195e8fff3b71f44b618b2a6ac23e3`
- SEO config SHA-256：`dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`
- production public identity SHA-256：`3f1f519bcf72365a86efe71bc31c618b0ef3ca9da2296b3d115ff11294b79980`
- production public identity：locale `ar`，hostname `ar-go-dev.shuijingwanwq.com`，URL `https://ar-go-dev.shuijingwanwq.com/`

## 上一轮 Preview HUMAN gate 失败与正式回流

上一轮 Stage A 已通过，并记录了旧 machine-readable A gate。随后完整 preview 的 automated acceptance 为 `PASS`，但维护者在 Preview HUMAN visual gate 中发现 Arabic desktop Tour 存在真实组合式阻塞问题，因此 HUMAN gate 判定 `failed`，没有进入 publish / Production。

人工发现的主要问题：

1. RTL desktop Tour 中 lesson pane 与 editor pane 实际完全重叠，lesson 正文被 editor 覆盖，另一半屏出现异常空白；
2. splitter 虽可见，但 `vertical-slide` 使用错误的 Angular directive 注册名，实际没有 link；
3. `welcome/1` 的 3 处 UI 方位说明仍按 LTR 物理方向表达，与已经镜像后的 Arabic RTL 界面冲突：logo、菜单和“下一页”箭头均会误导用户。

上述问题按正式流程分别回流到 repository-level shared RTL implementation 与 TranslationUnit workflow，没有直接改 canonical candidate 绕过质量 gate。

## Shared RTL implementation 修复

本轮 repository-level 修复涉及：

- `_content/tour/static/css/app.css` — `1ee1fe7ebde50711a0e0c43468fc079eedc5658f9685eebd5a336cca560d5a9d`
- `_content/tour/static/js/directives.js` — `2d1cc0a600ce3727ae0f9537f79b7500338688dae57737589f22d2bb3ebd0c95`
- `internal/tour/rtl_test.go` — `2cbef55b07a0bd99bea6bfad0d42cd7674a4623ead78ec867944977041b4be3d`
- `scripts/browser_acceptance.py` — `187529f5104afe70916596d6fb04af4274e2d9275e74641a0227941c84fe29f5`
- `scripts/verify-preview-browser-test.py` — `85c34bd5b0892916ea6f499c6192eba2dbda92ed03d6666093c10254091523c4`

确认的根因为 RTL 下固定 LTR 物理 `left` geometry 使两个 pane 同时占据右半屏；同时 `vertical-slide` 的 dashed directive 注册名没有被 Angular 1.0.6 正确 link。

修复后采用逻辑侧契约：lesson=`inline-start`、editor=`inline-end`；LTR 仍为正文左/editor 右，RTL 为正文右/editor 左。splitter 根据 lesson 的逻辑宽度计算并使用 `verticalSlide` 注册名；CodeMirror、代码和 program output 保持 LTR isolation。Targeted browser regression 同时覆盖 pane viewport、重叠/异常空白、RTL/LTR 视觉顺序、可见 lesson text、editor 覆盖、divider 边界与 drag 初始化。

当前 deterministic Surface Review package 的 22 个 `other_surfaces` 中，仅 `tour-runtime/directives` identity 相比上一轮发生变化；其余 `21/22` 保持不变。`app.css` 不属于这 22 个 package entry，但独立 reviewer 已将当前 CSS 作为 shared implementation semantic context 重新读取并核对。

## `welcome/1` TranslationUnit 回流

Preview HUMAN gate 暴露的 `welcome/1` RTL 方位缺陷通过正式 TranslationUnit workflow 回流：

- `qc-003`：对旧 candidate 建立正式 `C` + finding revision evidence；
- `chatgpt-ar-007`：由 generation session 完成 revision；
- process：`restore_passed=1`，`validation_passed=1`；
- `qc-004`：其余 `121` 个有效 A 从 `qc-002` carry-forward，新的 `welcome/1` 由独立 reviewer 重新审核为 A；
- final scope：`A/B/C/D=122/0/0/0`，`pending=0`；
- machine finalization：完成；
- promotion：`APPLIED`，`changed=1`，`unchanged=121`；
- status：`122/122` ready。

当前 `welcome/1` target SHA-256：`dace99f1ef7cae1f97cca735ec065a68458c6ebe7198a8082f4d1e08372f4076`。

三处正式修复为：

- `جولة في Go`：`في أعلى يمين الصفحة`，与 RTL 页面右上角实际位置一致；
- table-of-contents menu：`في أعلى يسار الصفحة`，与 RTL 页面左上角实际位置一致；
- `.next-page`：`السهم الأيسر`，与 RTL 中左侧“下一页”控件一致。

独立 reviewer 本轮重新比较完整 English source、当前 promoted Arabic target、完整 glossary 与当前 shared navigation semantics，结论为 `passed`。未发现 revision 新引入的漏译、误译、技术回归、术语冲突或 Arabic 自然度问题。

## schema v2 Course SEO refresh 与复审

`welcome/1` target promotion 后，其旧 Course SEO `target_sha256` stale。只读比对确认 stale Page 精确只有 `welcome/1`；随后按 schema v2 正式 `course-metadata refresh` 执行精确 stale subset refresh：

- stale pages：`1` — `welcome/1`
- provider：`chatgpt`
- model：`gpt-5.6-sol-high`
- generated_at：`2026-09-18T14:56:26Z`

当前 canonical English description：

> Learn how to navigate A Tour of Go, run and edit example programs, format code, and use keyboard shortcuts while experimenting interactively.

当前 Arabic localized description：

> تعلّم التنقل في جولة في Go، وتشغيل برامج الأمثلة وتعديلها، وتنسيق الشيفرة، واستخدام اختصارات لوحة المفاتيح أثناء التجربة التفاعلية.

独立 reviewer 使用完整 English source、canonical English description、当前完整 Arabic target、完整 glossary、localized description 与 source/source-description/target/glossary identity 重新审核 `welcome/1`，结论为 `passed`。description 完整保留 navigation、运行/修改示例程序、格式化代码、键盘快捷键与交互实验语义；没有增加 RTL 方位等 canonical 未授权语义，也没有遗漏、unsupported expansion、keyword stuffing、generic/duplicate wording 或 route mismatch。

其余 `102` 个 Course SEO Page identity 未变化，沿用上一轮正式审核结论。

## r2 Stage A 增量复审范围与 carry-forward

当前 package identity 已由独立 reviewer 精确验证：

- SHA-256：`8cd60683c260fb586fb81ab7898a45b2fd3149f1fd38507598944f1d213e6364`
- size：`4257 lines`，`462293 bytes`
- coverage：UI `113`；articles `7`；Course SEO Pages `103`；TranslationUnits `122`；other surfaces `22`

本轮实际重新审核：

- TranslationUnit `welcome/1`；
- schema v2 Course SEO `welcome/1`；
- changed package other surface `tour-runtime/directives`；
- 与组合语义直接相关的当前 `tour-template/index`、`tour-partial/editor`、`tour-runtime/values`；
- 当前 `app.css` RTL pane / top-bar / code-direction implementation context。

经 current identity 核对后 carry-forward：

- UI catalog：`113/113` identity unchanged；
- article metadata：`7/7` identity unchanged；
- TranslationUnit：其余 `121/121` exact-identity unchanged；
- Course SEO：其余 `102/102` identity unchanged；
- package other surfaces：其余 `21/21` identity unchanged；
- glossary、canonical source descriptions、canonical source-review authority、catalog/source、languages、project、SEO 与 production public identity 均未变化。

## A. Locale-level language quality review

- UI catalog：`113/113` — passed（identity unchanged carry-forward）
- article metadata：`7/7` — passed（identity unchanged carry-forward）
- schema v2 Course SEO：`103/103` — passed（`welcome/1` 重审 + `102` unchanged carry-forward）
- TranslationUnit context：`122/122` — passed（`welcome/1` 重审 + `121` exact-identity carry-forward）
- other first-party locale/runtime/template surfaces：`22/22` — passed（changed `directives.js` 重审 + `21` unchanged carry-forward；相关 shared context 额外复核）
- new TranslationUnit defect：`0`
- glossary defect：`0`
- UI defect：`0`
- article metadata defect：`0`
- Course SEO defect：`0`
- other-surface defect：`0`
- unresolved language blocker：`none`
- Stage A decision：`passed`

本轮 Surface Review 只使用 `passed/failed`，不以 TranslationUnit 的 A/B/C/D 替代 Stage A 判断。TranslationUnit QC、validation、Course SEO refresh 与 browser tests 均只作为各自 gate/identity/implementation evidence，不替代独立 Stage A 语言审核。

## B. Rendered surface acceptance

历史 Preview 记录：

- 修复前 automated preview acceptance：`passed`
- 修复前 Preview HUMAN visual gate：`failed`
- blocker：RTL desktop lesson/editor pane 完全重叠，以及 `welcome/1` 三处 RTL UI-direction 指引与实际界面冲突
- 回流：shared RTL implementation 已修复；`welcome/1` 已完成正式 TU revision / QC A-only / finalization / promotion；对应 Course SEO 已 refresh；Stage A r2 已重新通过

r2 Stage A gate 记录后第一次重新启动完整 locale preview，并执行正式 automated rendered acceptance：

- preview URL：`http://127.0.0.1:39385/`
- automated preview acceptance：`passed`（`PREVIEW SURFACE ACCEPTANCE: PASS`）
- Preview HUMAN visual gate：`failed`

该轮人工检查发现新的移动端视觉 blocker：Arabic RTL 的 editor 外层 `#right-side` 与 `#top-part` 均保持完整约 `375px` 宽，但 `#file-editor` / CodeMirror 实际只剩约 `143.39px`，代码被挤成极窄列并大量逐字换行，左侧留下大面积异常空白。该问题未被当时 automated acceptance 捕获，因此 machine PASS 不能替代 HUMAN visual gate。

进一步 live geometry 诊断确认：在 `375px` viewport 下，`#right-side=375px`、`#top-part=375px`、`#top-part > .relative-content=373px`，而 `#file-editor=143.39px`、`.CodeMirror=143.39px`。最终 root cause 为 mobile 基础规则要求 explorer controls `float:none`，但更高 specificity 且更靠后的 RTL rule 又把 syntax/imports controls 设为 `float:left`；浮动控件逸出 `#explorer` 正常流，随后 `#file-editor` 的 BFC 避让 float，导致 editor 被挤窄。

正式修复采用 shared responsive contract：RTL explorer 的 `float:left` 只在 `min-width: 601px` 的 desktop 生效；mobile 继续沿用原有 `float:none` 和“一行一个按钮”的布局，不保留 `#file-editor { width: 100% }` workaround。automated browser acceptance 同时新增 mobile explorer containment/float/逐行布局/editor geometry/viewport overflow/CodeMirror LTR 回归，因此该类 false PASS 现在会 fail closed。

修复后以当前 working tree 重新启动最终完整 locale preview：

- final preview URL：`http://127.0.0.1:33263/`
- current Stage A gate：`PASS`
- final automated preview acceptance：`passed`（`PREVIEW SURFACE ACCEPTANCE: PASS`）
- 375px RTL mobile：三个 explorer controls `float:none` 且逐行排列，`#explorer=88px`，editor parent=`373px`，`#file-editor=373px`，CodeMirror=`373px`，document overflow=`0`，CodeMirror direction=`ltr`
- desktop RTL：lesson/editor logical pane contract 与 splitter regression 继续通过
- Preview HUMAN visual gate：`passed`

维护者已在最终 preview 上人工确认桌面与移动端整体排版正常；移动端 explorer controls 恢复逐行排列，editor 完整占用可用宽度，未见新的明显视觉异常。至此 Preview rendered surface acceptance 已完成，允许进入 publish 前的 deterministic/Git/Production 准备阶段。

<!-- first-production-finalization:start -->
- production receipt identity: `locale=ar hostname=ar-go-dev.shuijingwanwq.com release=20260918T163333Z-ar-92e47622a371`
- production machine acceptance: `passed`
- production browser acceptance: `passed`
- unresolved production blocker: `none`
- overall final decision: `passed`
- decision: `passed`
<!-- first-production-finalization:end -->
