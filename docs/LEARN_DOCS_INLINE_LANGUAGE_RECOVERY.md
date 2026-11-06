# Learn/Docs inline language parser 修复交接

阶段：Learn/Docs HTML Page inline-language parser/protector 修复与恢复。

工程修复与完整 49 Page 影响分析：PASS。整体恢复：FAILED，存在真实 corpus authority blocker。逐 Page old/new Unit ID、input SHA、structure identity、internal slots、source SHA 和精确范围见 [影响分析](LEARN_DOCS_INLINE_LANGUAGE_IMPACT.md)。本文件是工程诊断/交接，不是语言 Generation、Snapshot、QC、receipt 或 finalization。

## 修复与共享机制

根因：`htmlBlocksMode` 将与 protected range 有任意 overlap 的 run 整体排除；包含 nested block 的非叶子 li 因而把 direct prose + inline code 放进 Page boundary。现在完整包含 inline protected element 的 run 可成为语言槽位，既有 `protectBlockMode` 继续产生 atom/open/close；被 opaque range 包含或截断的 run 不开放，opaque pre 内看似 paragraph 的 block 也不开放。未放宽 boundary、restore、binding、slot membership 或结构 validator，没有新增 schema、receipt 或生命周期。

两个真实共享缺口采用最小 extension：

- Generation Bundle 重建原先只核对保存的输入文件，没有确认当前 Catalog/protected input。现有共同重建路径现在先检查 current Unit/source identity，并重新构建完整 manifest 的 protected input SHA/token count。export/current-check/import/recover 共用该检查，不重写旧 ZIP。
- adapter 的 content-addressed ID 改变时，export scanner 将旧 ID 判成 unknown，阻断后续导出。现在只有同 kind/path/source SHA/bytes、合法 saved input path/hash 的记录可作为 superseded historical export 跳过；不进入新 ID 的 exported guard/candidate/QC，也不创建 alias。其他 unknown history 仍 fail closed。新 Snapshot scope 对 predecessor 同 index/path/kind 的不同 ID 报 identity_changed，不继承旧 rating/finding。

7 个 changed Page 使用新 ID 的既有 initial export → import/process/validation 路径，不伪造旧 C 对新 ID 的 revision 授权。原 findings 保留在 `learn-docs-v1-qc-001`。可信 42 candidate 的实际字节与最终 retry 输出还原、当前 validator 均已检查通过；仅 EOF framing 按现有 single-LF 合同处理。

后续机器条件满足后才可创建完整 `learn-docs-v1-qc-002`，以 `learn-docs-v1-qc-001` 为显式 predecessor。预期精确 scope：10 A carry-forward、7 identity_changed/review_required、32 unchanged B/C revision_required。该 Snapshot 尚未创建，7 个新候选尚未生成；不宣称已完成审核或 validation。

## 当前 machine state 与 Bundle

实测 `retranslation progress --content-package learn-docs-v1 --locale zh-CN`：49 total，42 passed，7 missing，restore_failed=0，validation_failed=0，stale=0。missing 指新 ID 尚无候选，旧 candidate 未删除。

| batch | current-check | 真实状态 |
| --- | --- | --- |
| chatgpt-zh-CN-024 | stale：old Unit ID 不在 current Catalog | 历史 export/ZIP 保留；维护者报告 Generation FAILED，0/23 outputs，mutation none；仓库无 raw-responses/candidates/validation/result |
| chatgpt-zh-CN-025 | CURRENT | 16 个 unchanged B/C revision 输入仍兼容；Generation 未开始，不强制重新导出 |
| chatgpt-zh-CN-026 | CURRENT | 新共享 initial batch，7 changed Page；仅 manifest/inputs，尚无语言输出 |

仅准备第一固定工作集 1–30 的 exact recovery Bundle；第二固定工作集 31–49 没有 parser affected Page，不创建空恢复 batch。

- path：`/tmp/zh-CN-learn-docs-v1-inline-recovery-pages-1-30-generation.zip`
- SHA-256：`1ec83fee472c35c3d48b1a5f2700aa9006ce1f1960a5cb5288c719a538a7e779`
- identity：`37418da21cc21c7c4b131153af25d6014811a62e6f1fd99ef8b9cbdc66a979e5`
- batch：`chatgpt-zh-CN-026`；task：`translation-unit-initial`；unit count：7
- stable indexes：`2,3,4,5,9,12,13`

历史 SHA-256 实测保持：

- 024 ZIP：`e75350bdcc797e17eefa116209800d80cc9d9c4024b8aa21e07290e944bb4743`
- 025 ZIP：`8284b6e030e3bb64d99b7ba581d27c5e901d9cee4a54012ca043efa9fb7b1c14`
- qc-001 manifest：`0cfa2bdfab907cdc5cf6f807b12c4ff870ce9028bf849d54d3ea1bd548386f5c`
- qc-001 QC results：`7e36685616e2f61bd574b4a94b08445035f751f7c68a5ab81059e2175879a6e0`

Formal evidence mutation=known：仅新增 026 manifest/inputs；旧 Snapshot/QC、024/025、retry、Generation origins/archive、corpus、Glossary Review 未修改。candidate mutation=none；译文措辞 mutation=none。未执行 Generation、Reviewer judgment、commit/push。

## 验证与唯一下一步

Focused tests 覆盖 direct prose + code/var/kbd/samp/link/link+code、多 inline elements、软换行、nested p/list、opaque pre/script/style/textarea（含内部似 HTML markup）、真实 frozen 7 Page 和完整 49 Page 影响比较。非法 code/href/tag/token/binding/block/list 改动必须拒绝。共享 focused tests 覆盖 current projection drift、historical export 保留、successor 不继承旧 A、deterministic import、sibling retry currentness、A carry 和真实 finding revision。以上 PASS。

`git diff --check`：PASS。全 worktree `git diff --stat` / `git status --short` 保存于 `/tmp/go-tour-inline-git-state.txt`，包含任务前的合法 dirty 变更，不能归为本轮改动。本轮实际修改 4 个 implementation 文件（linguistic.go、generation_bundle.go、retranslation_export.go、quality_check_results.go），新增 3 个 focused test 文件、2 个工程报告与 026 manifest/7 inputs。未运行全仓库、browser 或 Production 测试。

直接相关 `site-content corpus-check` 真实失败：

```text
tour-i18n: unified glossary corpus projection stale
exit status 1
```

language-span discovery 改变了统一 corpus 的真实 projection。保存该 failure，不修改 corpus 假装历史 Review current，不执行新 Glossary Review；维护者未明确暂缓，因此不写 Deferred Issue。026 TU current-check PASS 证明现有持久化输入/authority 与当前 Page projection 一致，不能替代 corpus-check 或宣称整体治理无阻塞。暂不交给模型开始生成。

唯一下一步：由 Codex 仓库工程恢复 parser 更新后的 unified corpus authority，准备合法的独立 Glossary Review 交接，停止于 Reviewer handoff；本轮不执行新的语言审核。corpus/Review CURRENT gate 收口后须重新 current-check 026，再由原 Generation session 处理这 7 Page。其余 42 不因 parser 改动重译，32 个 unchanged B/C 继续共享 revision。
