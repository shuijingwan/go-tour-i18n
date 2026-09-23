# ms-MY Locale Surface Review Stage A — 2026-09-23（首次审核）

- locale: `ms-MY`（Bahasa Melayu / Malay）
- review_id: `20260923-initial-stage-a`
- reviewer: `chatgpt-gpt-5.6-sol-high-independent-ms-MY`
- reviewed commit: `b1a93659458b8caedfd77ff62bb763872175d651`
- review ZIP SHA-256: `18e83c65bcdec533ea843339674c34b58fc0cddfd5db4e796d53a4f91bdec118`
- surface-review.json SHA-256: `ca591313d465be020123f410583f9d110f6cbf5eab5d9e068bf487787422b125`
- glossary: `locales/ms-MY/glossary.yaml` SHA-256 `5c0812ed978f2c877553d890fa368234c99d2591e0f7d87429821401f5965ee5`
- UI English SHA-256: `9a6e6877cd195144d6121f93cdabd387bdeb35c60a3f1afa9b8f417c7b6dabda`
- UI ms-MY SHA-256: `2e58121877dd93c7f1d7820035ba8004999755e837980cab30acf97170d9cd7d`
- article metadata SHA-256: `ca33075e0907e98b8b90f2ad3e5d92ac15258e3735d323527ac7760b25f7beb7`
- Course SEO metadata SHA-256: `064f0e3e8e4983f1263314ca29f0b30a17845f79e2a241ccf947f475ee5133d2`
- Production state (review point): `first-production`
- Production hostname: `ms-go-dev.shuijingwanwq.com`
- Production URL: `https://ms-go-dev.shuijingwanwq.com/`
- CDN: Cloudflare
- language quality review decision: **failed**
- A gate: **not recorded**

## 完整 Stage A coverage

- UI catalog: **113/113**，含 plain/rich、动作名称、占位符与模板实际显示场景；发现一组首页支持文案阻断问题。
- Article metadata: **7/7**，逐 article title/subtitle 对照；无新增阻断问题。
- Course SEO schema v2: **103/103**，逐页对照 canonical English description、完整 English source、ready Malay Page、正式 glossary 与 localized description；发现 `methods/26` 一项阻断问题。
- TranslationUnit context: **122/122** ready，使用已独立通过的全部 TU QC 与正式 package 中的 103 Page 全文核查组合语义；不重新执行 TU A/B/C/D QC，未发现新的 TU defect。
- Other surfaces: **22/22**，涵盖语言注册与 profile、首页、Tour shell、导航、runtime JavaScript、模板与 partial、SEO、公网身份；首页的 UI 文案与当前语言身份组合存在问题。
- Glossary: 检查完整 mandatory、preferred、forbidden、keep 及 Go Playground、Go identifiers、官方与社区项目身份；未发现新的术语决策阻断问题。
- Reviewer ZIP 成员及 manifest SHA-256 全部匹配；两次正式 current working-tree deterministic export 均与已上传 ZIP 逐字节一致。

## Blocking findings（交回原 Generation session / 必要时仓库维护者）

### F1 — Course SEO localized description — `methods/26`

- category: `course-seo-localized-description`
- English canonical: `Close out the lesson and decide whether to revisit the module directory or continue directly into the next lesson.`
- Current ms-MY description: `Tutup pelajaran ini dan tentukan sama ada mahu kembali melihat direktori modul atau meneruskan terus ke pelajaran berikutnya.`
- Page English: `You finished this lesson!`；Page ms-MY: `Anda telah menamatkan pelajaran ini!`
- Problem: `Tutup pelajaran ini` 在面向读者的搜索描述中容易产生关闭课程的动作义，不够自然，也与已经完成课程的 Page 语境不一致。
- Requirement: 由原 Generation session 只修订本页 schema v2 localized description；自然表达结束已完成的课程，并完整保留返回模块目录或直接进入下一课的选择，不增删 canonical semantic scope。按当前正式 course-metadata revise lifecycle 更新后交原 Reviewer re-review。

### F2 — UI/Home combined meaning and public language identity — `support.translation_title` / `support.intro`

- category: `ui-home-public-context`
- English source title: `Support the English version`
- Current ms-MY title: `Sokong versi bahasa Inggeris`
- English source intro: `You can also voluntarily support the continued maintenance of the English version. Your support helps this language version continue to be maintained and developed.`
- Current ms-MY intro: `Anda juga boleh menyokong penyelenggaraan berterusan versi bahasa Inggeris secara sukarela. Sokongan anda membantu memastikan versi bahasa ini terus diselenggara dan dibangunkan.`
- Actual home context: `_content/tour/template/home.tmpl` unconditionally renders both UI keys next to the build-selected `Bahasa Melayu` identity; `supportForLocale` creates locale-specific reference `go-dev-ms-MY`.
- Problem: visible English-versus-current-language sponsorship scope is contradictory/ambiguous on the Malay site. Both English source and target have the issue; it cannot be corrected by translating the same English source into a different claim.
- Requirement: obtain the real intended support audience from the project authority; coordinate any needed canonical English UI source adjustment and ms-MY UI replacement (and shared impact where applicable), making the supported language/version unambiguous without suggesting official Go endorsement or changing payment details. Return updated CURRENT Reviewer Bundle to the original independent Reviewer.

## 后续阶段

- Preview/browser acceptance: 未执行；A gate 未通过，不应启动完整 locale preview。
- Production verification: 未执行；当前仍为 first-production。
- record-a: 未执行；不得把 FAILED 记为 passed receipt。
- Replacement generation: 本 Reviewer 未参与。
