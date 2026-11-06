# Learn/Docs complete-code teaching-comment 修复交接

locale：`zh-CN`。阶段：Learn/Docs complete-code teaching-comment parser/protector 修复与恢复。工程修复、49 Page 影响分析、corpus CAS 恢复、Reviewer Bundle 准备：PASS。语言恢复尚未执行，停止于独立 Reviewer handoff。本文是工程诊断与恢复规划，不是 Generation、Reviewer judgment、Snapshot 或 receipt。

## 根因与修复

四个直接 blocker（stable index 6/7/8/18）的 Completed code 实际都是 Markdown fenced code，内容包含完整 `package main` 文件。Goldmark 已正确提供 code block 坐标；没有模板/嵌入 source projection 漏接。共享 `analyzePreformattedGo` 的 `isSafeGoFragment` 原先只尝试补 `package p`、或补 package/function 的两种 fragment wrapper，完整文件因此重复 package，判为 Static。`SafeTeachingCommentSpans` 返回空，`teachingSpans` 无 slot 可加入；普通 Markdown prose parser 有意跳过 code block，`FormalPage` 最终把这些字节作为 boundary。这是安全 span 发现层的缺陷，不是 downstream restore 漏洞。

另外，旧 helper 在 identifier 两侧拆开自然语言 span，Page 将 identifier 留在两槽之间的 boundary，导致 comment 译文必须沿英文顺序分段；普通词形 identifier 无法完整表达中文语序。共享保守分类器也未识别 `Capture connection properties.`、`Associate message with name.`、`formats contains greeting formats.`、`Query albums by artist.` 中的明确教学词形。

最小修改：

- 共享 analyzer 增加原始完整文件的 Go parser 尝试，仍保留原有 fragment wrapper、scanner error 与 block-comment Static 拒绝；复用既有 Example machine-comment 分类，保护 Go/line/build/present/test-output/generated markers。
- 同一 analyzer 导出 complete teaching body、identifier source coordinates 与 code identity set；原 `SafeTeachingCommentSpans` 从该 helper 得出，不另造 Go comment parser。
- Page adapter 保持每个 comment body 一个 internal linguistic slot；源 identifier 作为 atom，其他 technical words 继续使用既有 `protectBlockMode`。delimiter、leading/trailing spacing、indentation、newline、完整 code 和 fenced/pre enclosing bytes 留在 boundary。
- 原 token exact-set、slot membership、boundary order、binding 与整体 Markdown/HTML structure 检查保持；补充 comment 内 exact identifier count、源 code identity 新增引用与未绑定 technical identity 拒绝。允许 identifier 在同一个 comment 内随目标语言重排，不允许跨 comment/code 移动。

没有开放整个 code block，没有修改语言政策，没有新增持久化 schema、正式 Reviewer unit、Site-only lifecycle、retry、Snapshot 或 QC。一个 physical Page 仍是一个 TranslationUnit。HTML plain pre 和 Markdown indented code 同样覆盖；HTML entity/markup、不认识的 Go fragment 和 block comments 继续保守关闭。

## 完整49 Page影响与复用规划

完整 old/new Unit ID、formal input SHA、source SHA、structure identity、slots、teaching spans、protected 数量和逐页状态见[影响报告](LEARN_DOCS_COMPLETE_CODE_IMPACT.md)。所有49 Page：**source bytes unchanged**。修复前后完整 Document（包括 protected raw/role/pair）分别保存在 `/tmp/go-tour-teaching-before.json`、`/tmp/go-tour-teaching-after.json`。

- changed=19；unchanged=30。
- 必须重新 Generation → machine validation → 独立 QC：`2 3 5 6 7 8 10 12 13 15 17 18 19 20 22 25 26 27 31`。
- unchanged A 可 carry：`4 14 16 21 23 28 34 45 48`（9）。旧 A 的20、27已 identity_changed，不能 carry。
- unchanged B/C 可保留真实 finding，待恢复后的集中 revision：`1 9 11 24 29 30 32 33 35 36 37 38 39 40 41 42 43 44 46 47 49`（21）。

30 个 unchanged candidate 的实际 SHA、validation SHA、原 raw response restore、当前 validator 全部通过；candidate 字节未改。19 个 changed 的旧 candidate/QC 保留为历史，不能静默绑定新 ID。第一固定工作集1–30有18个 affected Page，第二固定工作集31–49只有31；本轮因 Glossary gate stale，不导出 Page recovery batch/Bundle。

`learn-docs-v1-qc-001`、`learn-docs-v1-qc-002` 均 immutable。qc-002 的有效历史结果继续是 A=11、B=16、C=22、D=0、revision_required=38；本轮不修改这些结果，不创建 successor Snapshot。完整19个新 Page Generation/validation 完成后，才能以 qc-002 为 predecessor 建立完整49 Page successor，由共享机器 scope 计算9 A carry、21 unchanged B/C revision_required、19 identity_changed quality_check_required。19个新 identity 先完成独立 QC，再集中 exact revision 当前全部 B/C；数字是本轮精确恢复规划，不冒称已写入新 Snapshot。

## 027 / 028 currentness

两个原 ZIP 及 manifest/inputs 保留，current-check 都真实返回 stale：

```text
027: stale formal unit: unknown translation unit "a58b321bd863b55022f34c8b3f5b8fb48b6fa77ec9e8b252a88095df48a2d66f"
028: stale formal unit: unknown translation unit "2dc9a561d950be08d2963289b78129362fb12e4c57519637b3e1eea632057f46"
```

027 的 Generation FAILED、0/22 outputs、mutation=none 事实不变；028尚未 Generation。不得修改旧 batch 使其重新 CURRENT。

## Unified Corpus authority

修复后第一次正式 `site-content corpus-check`：`unified glossary corpus projection stale`。已保存原 bytes：

`data/glossary-corpus-history/b6aa4bad4202c8728fd9abc0acb61d796b717ec558e3779682d02f151de1db47.json`

正式 CAS 的真实 prior file SHA：`b6aa4bad4202c8728fd9abc0acb61d796b717ec558e3779682d02f151de1db47`。先 dry-run、完整 old/new contributor 比较，再同一 prior SHA `--apply`；dry-run 与 apply JSON逐字段、逐字节一致。未手工覆盖 corpus，没有 compatibility shortcut。

- old corpus identity：`f4786502100b3ae6f34b539f70728f9a062e7c1982ee91111cf674c57aebd771`
- new corpus identity：`56fb2011ed379cb4b312ee04e861da9790f5dd6346594b768465fb56b24bd318`
- new corpus file SHA：`27825bc34eea7962ce331f1643bee99b3bed9dd2395739b8a49745ae250ec4e7`
- contributor=428；changed=19（全部 Learn/Docs active-content）；unchanged=409。Tour、shell、terminology-discovery、source registry、schema、contributor membership 均不变。精确比较：`/tmp/go-tour-teaching-corpus-impact.json`。
- 更新后 `site-content corpus-check`：PASS；`site-content terminology-check`：PASS，离线，25个 discovery contributor/8079 contexts 保持。
- glossary SHA不变：`5c6f5666eb17e0563221259a560605b878ee7a9faf46112bd62c0debac2b6f63`。
- `20261010-zh-CN-glossary-review-008` 原 passed receipt 保留，自然 stale；正式 `glossary-review check --coverage unified` 返回 missing/stale/ambiguous or non-passed。`retranslation progress` 同样被此正式 gate 阻断，未输出恢复计数或创建 artifact。

## 测试与证据完整性

核心 focused tests：PASS。包含 complete file/fragment、Markdown fenced/indented、plain HTML pre；Context/name/message/formats 同 comment 中文语序与 exact source identity；多个 teaching comments；unsafe/static/machine-semantic/block comments；identifier/string/numeric/code/operator/comment delimiter/comment line/newline/indentation/fence/pre 的修改拒绝；新增 source code identifier 或未绑定 technical identity 拒绝。

完整性 invariant：同一共享 analyzer 声明安全的 teaching body 必须落在正式 linguistic slot 中；原 natural span helper 的每个 span 同样存活。完整49 Page source-input roundtrip 逐字节相同。真实四页的 Capture/Get/albumsByArtist/Hard-code/addAlbum、Use f.Add、Initialize/SumInts/SumIntsOrFloats/SumNumbers、album/getAlbums/postAlbums/BindJSON/getAlbumByID 全部已进入正式 protected input，未再留在 boundary raw。共享 currentness、historical export、successor independent QC、A carry、corpus projection 与 terminology CAS 的直接相关测试通过。

额外运行的既有 `TestTerminologyCorpusChangeStalesOldUnifiedReview` 发现 scope 外真实 failure，未修复、未扩展调查：

```text
terminology_authority_test.go:508: unknown active-content corpus parser html-visible/v1
```

最小 evidence：该 fixture 第501行循环将已有 contributors 的 purpose 全部设为 active-content，而 fixture 已包含 discovery parser；第508行 strict validation 拒绝。没有将此失败归为 parser 修复验证通过，也没有写 Deferred Issue（维护者未明确暂缓）。真实 CLI corpus/terminology checks 和新 Bundle current-check 均通过。

重新核对1247个已有文件SHA：只有被授权CAS更新的当前 corpus 文件变化；两个旧 Snapshot、所有历史 reviews、024–028/其他 batch、raw/validation、canonical与batch candidates、glossary及原027/028 ZIP均未变化。证据：`/tmp/go-tour-teaching-evidence-check.json`。formal authority mutation=known（corpus CAS、新原bytes archive）；historical evidence mutation=none；candidate mutation=none；wording mutation=none。未执行 Generation、Reviewer judgment、review-009 record、commit/push。

## 唯一下一步：完整 Unified Glossary Reviewer handoff

- Bundle：`/tmp/zh-CN-20261010-glossary-review-009-reviewer.zip`
- SHA-256：`a71cbc40108de0cafb57d140324c1b75d89aa58f0e5faa692014ffd7f1cb7515`
- Bundle input identity：`9d60c21ecb88ba2f0308215e5a2c09bd5270967321da016fdc9e44e2beeea9d5`
- 正式 reviewer-bundle-check：CURRENT。
- 预留 review_id：`20261010-zh-CN-glossary-review-009`（未记录）。
- Reviewer：独立 `chatgpt-zh-CN-reviewer`，GPT-5.6 Sol + High。
- Generation session：`chatgpt-zh-CN-generation`；本轮不交给它执行。

把该 ZIP 上传到独立 Reviewer session，要求一次完整 Unified Glossary Review：完整读取ZIP中的manifest、全部正式authority、corpus与glossary，审核全部已有规则及所需source context，给出 passed/failed与全部blocking findings；不得生成replacement、修改glossary/candidate、执行Page QC或开始Page Generation。完成该独立Reviewer invocation后停止。Reviewer PASS并正式current-check/record/check后，才允许继续19个affected Page的共享恢复。
