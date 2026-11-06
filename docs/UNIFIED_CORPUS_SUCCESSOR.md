# Unified glossary corpus successor

`go-learning/unified-glossary-corpus-successor/v1` 是唯一 unified corpus 的下一 identity，不是独立 glossary、语言 namespace、Review、completion 或 activation。命令明确消费已经冻结的 Shell affected-scope artifact，绑定其 source authority 的 path/SHA/self identity；不执行第二套 Homepage/chrome parser、不重新提取 slots 或计算 carry。

preflight 从严格验证的 current corpus 开始，用 `ProjectShellSuccessorContributor` 对已冻结 representation 重投影，并要求与 affected scope 的 proposed contributors 完全相同。五个 additions 仍为235 contexts。旧 `data/site-shell/home.html` 按声明退休；另有独立消费者时 fail closed，不存在时记录 `retirement_not_present`。其余 contributor/context bytes 全部沿用，包括 translation body、Learn/Docs、Tour 与 discovery。当前 corpus 不做跨 contributor 文本去重，因此退休旧5 contexts 后净增230，而不是235。

artifact 包含 predecessor corpus path/file SHA/semantic identity、Shell source 与 affected-scope refs/identities、完整 additions/retirements、unchanged count/identity、完整 resulting contributors/contexts、next semantic identity/canonical file SHA 和 self identity；`state=pending`。严格 canonical JSON、exact-set、无歧义输入、path traversal/symlink 拒绝及 immutable no-overwrite 均保持。

## Preflight 与 apply

```sh
# 只读 preflight；不会保存 artifact、改 corpus 或启动语言工作：
go run -mod=readonly ./cmd/tour-i18n site-content corpus-successor \
  --input data/site-shell/successor/affected/zh-CN/b106897bf70ef8c8bd3d439bebca598ea911635364c8a1000a40b3ce83b599e2.json
```

**本轮未执行实际 apply。** 维护者之后显式加 `--apply --prior-sha256 <exact-predecessor-file-sha>` 才会发布。apply 重算 inputs/projection，使用现有 corpus lock 并在安装前再次检查 exact predecessor bytes；拒绝 preflight 后漂移。旧 bytes 按既有 canonical history 路径 `data/glossary-corpus-history/<sha>.json` immutable 归档，successor evidence 保存于 `data/unified-glossary-corpus-successors/<identity>.json`；任一同名不同 bytes 都拒绝覆盖。publication marker `data/.corpus-successor-pending` 使中断或安装结果不明时 reader 返回 `mutation=unknown`，不盲目重放。

旧 corpus 的退休 ZIP storage reference 保留在历史 bytes 中。内容寻址 archive 只承接同 SHA current corpus 的原 proof，验证自己的 archived bytes，不能读取 later current corpus 作为历史依赖；原 proof 不改写。next corpus 用原 proof 已验证的 exact Learn/Docs source membership 的 repo-local path/SHA，避免声称为新 corpus 制造过不存在的 ZIP proof。全部现有 Learn/Docs contributors 和 contexts 不变。

只有 current corpus bytes 精确等于 successor result、immutable evidence 能由 archived predecessor 与仍有效的 Shell inputs 重算时，`CheckUnifiedCorpus` 才采用已安装 successor。pending artifact 的存在不改变 active authority。旧 `language-authority` 不读取 pending sources；遇到已经安装的 successor 时拒绝回退成旧 global projection。原 corpus 的其他旧路径保持原语义。

apply 后，旧 `20261011-zh-CN-glossary-review-010` 仍为 immutable historical PASS，但其 `corpus_identity_sha256` 不等于 next identity，既有 Review reader 将其判为 NON_CURRENT。不会生成新 Review，也不会声称 Glossary PASS。随后必须依次完成 Unified Glossary Refresh、独立 Glossary Review、compatibility/currentness，才可进入 affected Homepage/shared language Generation。本合同没有修改 locale completion、package closure、activation、Surface 或 language results。
