# ar Locale Surface Review — 2026-09-18 首次 production

## 审核身份

- locale：`ar`
- stage：`Locale-level language quality review (Stage A)`
- reviewed repository HEAD：`8a1ad30b3a17b34f9df87746acae61dd9e93126d`（`feat: 初始化阿拉伯语并支持 RTL`）
- current review package：`/tmp/ar-surface-review.json`
- current package SHA-256：`50a6a0d83eb05df50e1d133d8d5c437abe8bfd34b8245d533ba5bab770a5535e`
- package size：`4257 lines`，`461591 bytes`
- reviewer：`ChatGPT GPT-5.6 Sol High`
- date：`2026-09-18`
- production state：`first-production`
- language quality review result：`passed`

## 当前正式输入 identity

- glossary：`locales/ar/glossary.yaml` — `348abf2c0915c6432b4c4f9d2a315178f0259ef2e82457c5b46255eded2eef46`
- English UI catalog：`internal/tour/ui/en.json` — `9a6e6877cd195144d6121f93cdabd387bdeb35c60a3f1afa9b8f417c7b6dabda`
- ar UI catalog：`internal/tour/ui/ar.json` — `3f12dce6fa2a60b76a329c78bf8cf4200866fd79b3b631513bf6f0be17cb5f79`
- article metadata：`locales/ar/article-metadata.json` — `860dd538c48dc66d088583322be2d05b0a04a95cd69536b2feda3f989635a65f`
- schema v2 course metadata：`locales/ar/course-metadata.json` — `22f1454d6dfa555b7df39aedb0aeed62c5ba19d6ebe878af06a1f8fde1d33845`
- canonical English source descriptions：`data/course-seo/source-descriptions.json` — `1a46d981268ba75767986b3c212be06b6f678179f882efbc1813dedfad0a171d`
- canonical source-description review：`canonical-en-002`，decision `passed`
- canonical source-description review authority SHA-256：`9377e85b56bef45c5e1a1832186be7bc804816d0ac9149147db97267371b736f`
- catalog/source identity SHA-256：`9b6f9e5a5d698b4417aee5fc64239408461fb47876a69c3c803b3e51f25f285f`
- languages config SHA-256：`7496d738b046df47210083c62cd4eed6bc42778e509d8809ae28e8a4ee282fec`
- project config SHA-256：`78ae201af9207dbb74b266fb64614b82688195e8fff3b71f44b618b2a6ac23e3`
- SEO config SHA-256：`dc09d40a9c4a0d3aa9f9699b85a684feb104cb3642b71bca1f08750220397a64`
- production public identity SHA-256：`3f1f519bcf72365a86efe71bc31c618b0ef3ca9da2296b3d115ff11294b79980`
- production public identity：locale `ar`，hostname `ar-go-dev.shuijingwanwq.com`，URL `https://ar-go-dev.shuijingwanwq.com/`

## 第一轮完整语言审核

第一轮使用当时的 deterministic Surface Review package 完成首次上线要求的完整语言审核。该旧 package 仅作为历史审核过程，不再作为当前 authority；当前 authority 是上方记录的 current package。

第一轮实际完整覆盖：

- UI catalog：`113/113`
- article metadata：`7/7`
- schema v2 Course SEO：`103/103`
- TranslationUnit context：`122/122`（103 Page + 19 Example；不重新执行 A/B/C/D 评级）
- other first-party locale/runtime/template surfaces：`22/22`

为避免把 exporter 的 TranslationUnit 数字 coverage 当作语言审核，独立 reviewer 还重新读取了 19/19 eligible Example 的当前正式 English source 与 Arabic ready candidate，并检查允许翻译的自然语言注释。因此 TranslationUnit context `122/122` 是实际语言复核 coverage，而不是复用此前 TranslationUnit QC 结论。

第一轮 Stage A 结论为 `failed`，只发现 4 个 schema v2 Course SEO localized-description defect：

1. `flowcontrol/2`：两个 optional statements 的阿拉伯语双数/并列结构不自然且有歧义，未清楚表达 initialization statement 与 post statement 均可省略。
2. `flowcontrol/3`：`تعليمي` 不能正确表达两个 `تعليمة`，破坏了 initialization/post statements 与 semicolons 的自然句法关系。
3. `methods/14`：`أنواع عشوائية` 把 arbitrary types 偏移成“随机类型”，与 canonical semantic scope 和当前 Arabic target 不一致。
4. `methods/26`：`أغلق الدرس` 对 close out 过度字面化，更像“关闭课程/页面”，不符合完成本课后选择下一步的语境。

第一轮没有发现：

- TranslationUnit defect；
- glossary defect；
- UI defect；
- article metadata defect；
- other-surface defect。

## Course SEO 缺陷修订

上述 4 条 localized description 由与 reviewer 分离的 ChatGPT GPT-5.6 Sol High generation session 按当前 schema v2 Course SEO revision contract 重新生成，并由维护者本地终端使用正式 `course-metadata revise` 机械更新。

正式 revision subset：

- `flowcontrol/2`
- `flowcontrol/3`
- `methods/14`
- `methods/26`

当前完整 `course-metadata.json` 仍为 `103/103` Page。未受影响的 99 Page 保留原 generation provenance：`chatgpt / gpt-5.6-sol-high / 2026-09-18T13:02:48Z`；仅上述 4 个 revision Page 更新为 `chatgpt / gpt-5.6-sol-high / 2026-09-18T13:31:37Z`。四页绑定的 source、canonical source-description、最终 Arabic target 与 glossary identity 均保持 current，因此本次属于语言质量 revision，不是 identity refresh。

revision 后重新执行完整 build 与 `surface-review export`，生成当前 deterministic package：

- `/tmp/ar-surface-review.json`
- SHA-256：`50a6a0d83eb05df50e1d133d8d5c437abe8bfd34b8245d533ba5bab770a5535e`
- coverage：UI `113/113`；article metadata `7/7`；Course SEO `103/103`；TranslationUnit context `122/122`；other surfaces `22/22`

## 缺陷回流复审

独立 reviewer 重新读取 current package，并对 4 个发生变化的 Page 使用完整 English Page source、canonical English description、完整最终 Arabic target、完整 current glossary、修订后的 localized description、route 以及 source/source-description/target/glossary identity 完成 full-context 复审。

复审结果：

- `flowcontrol/2`：passed。修订后的 Arabic 清楚区分 `تعليمة التهيئة` 与 `التعليمة اللاحقة`，并通过 `كلتيهما` 明确两者都可省略，同时保留 `شرط الحلقة`；上一轮双数/并列歧义已解决。
- `flowcontrol/3`：passed。错误的 `تعليمي` 已消失；当前文案自然、准确地表达省略 initialization statement、post statement 以及与二者关联的 semicolons。
- `methods/14`：passed。`من أي نوع` 正确表达 arbitrary/any type，不再包含“随机类型”的错误含义；empty interface、等价的 `any` alias 与 general-purpose code 语义均完整保留。
- `methods/26`：passed。`اختتم الدرس` 自然表达完成/结束本课，不再字面化为“关闭课程”；返回 module list 与继续下一课两个选择均完整保留。

四页均未发现新的遗漏、unsupported expansion、技术误译、术语冲突、generic wording、duplicate wording、route mismatch 或 Page identity mismatch。

## 未修改范围 carry forward

与第一轮完整审核相比，当前 package 中 UI、article metadata、TranslationUnit、glossary、other surfaces 及其正式 identity 均没有发生新的变化。本次正式 `course-metadata revise` 只修改了明确的 4 Page subset；其余 99 个 Course SEO Page 的 description、identity 和原 generation provenance 均保持不变。

因此，第一轮已经完成完整语言审核且没有 finding 的以下范围继续适用于当前 package：

- UI catalog：`113/113`
- article metadata：`7/7`
- TranslationUnit context：`122/122`
- other first-party locale/runtime/template surfaces：`22/22`
- Course SEO 未变化 Page：`99/99`

当前 Course SEO 的完整有效结论由 `99` 个 unchanged Page carry forward 加本轮 `4` 个 revision Page 重新审核通过构成，最终仍覆盖 `103/103`。

## A. Locale-level language quality review

- UI catalog：`113/113` — passed
- article metadata：`7/7` — passed
- schema v2 Course SEO：`103/103` — passed
- TranslationUnit context：`122/122` — 未发现新的 TranslationUnit defect
- other first-party locale/runtime/template surfaces：`22/22` — passed
- unresolved language blocker：`none`
- language quality review result：`passed`

Stage A 当前已完成并通过。automatic validation、TranslationUnit QC、machine finalization、promotion、Course SEO assemble/revise 与 deterministic package export 均只作为各自 gate 或 identity evidence，不替代本次独立语言质量判断。

## B. Rendered surface acceptance

- automated preview acceptance：**尚未执行**
- Preview HUMAN visual gate：**尚未执行**
- Production verification：**尚未执行**

Stage A language-quality review 已完成并通过。Preview rendered surface acceptance 仍是后续独立 gate；本 evidence 不提前声明其结果，也不提前声明任何 Production machine/browser 结果。

<!-- first-production-finalization:start -->
- production receipt identity: `PENDING`
- production machine acceptance: `PENDING`
- production browser acceptance: `PENDING`
- unresolved production blocker: `PENDING`
- overall final decision: `PENDING`
- decision: `pending`
<!-- first-production-finalization:end -->
