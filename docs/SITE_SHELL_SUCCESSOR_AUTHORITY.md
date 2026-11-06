# Site v2 Shell successor language authority

后续新增 frozen Page package 与已接受 structured source 的正文扩展，使用[Source content adoption](SOURCE_CONTENT_ADOPTION.md)。本文件下述首轮 shell adoption 保持原始固定 scope；不能把它或 raw-unchanged structured-input-reopen 用来隐式采纳新的正文。

维护者于本轮正式决定：zh-CN 第一次 Site v2 release 的 `/` 必须使用冻结 go.dev 官方完整 Homepage，其 current English source 是 repo-local frozen `_content/index.md`。不先发布简化项目导航页。一个物理页面、一个完整正式 Homepage Page TranslationUnit，沿用 `go-learning/page-translation-unit/v1`；内部 slots 仅负责保护、还原与影响分析。

继续使用 `site-v2-shell`。foundation 建立当时确实为 pending：原 `go-learning/shell-source-successor/v1` 与 `go-learning/shell-successor-affected-scope/v1` 保持原 bytes/state/identity。当前治理通过 additive `go-learning/shell-current-adoption/v1` 绑定该冻结 foundation、原 global authority、新六份 source inventory、current package identity 与 reconstruction/runtime dependencies；`data/site-shell/successor/current.json` 原子指向 immutable adoption。原 `data/site-content-scope.json` 不重写，current reader 投影新 shell inventory，Learn/Docs 与 Tour package identity 不变。这不是第二个 locale namespace，也不是语言 acceptance。

## Source 与提取边界

| Source | 语言职责 | 正式表示 |
| --- | --- | --- |
| `_content/index.md` | Homepage metadata、正文、可访问性属性与安全教学注释 | 1 个完整 Page TU |
| `_content/site.tmpl` | shared chrome 可见文本、可访问性属性、输出到图片 alt 的 `$alt` 字面量 | site-level structured slots |
| `_content/menus.yaml` | header/footer labels、explanations、social accessibility names | site-level structured slots |
| `_content/resources.yaml` | resource title、description、iconName | site-level structured slots |
| `_content/testimonials.yaml` | testimonial role title、quote | site-level structured slots |
| `data/site-shell/translation.html` | `/translation/` 项目自有正文 | 原 structured 表示与语言结果 exact carry |

`data/site-shell/home.html` 仅 historical predecessor，不能成为本次 `/` 的 current publication source。旧 bytes、Generation、closure、Surface、activation evidence 不重写、不删除；historical closure reader 继续证明旧 judgment，但旧 completion 不满足新 source 的 current completion。Source adoption 本身不授权 activation 或上线。

模板先做 action/comment/quoted delimiter 词法隔离与 Go template 语法检查，不执行 template。named-template signatures 仅在语法检查的临时内存表示中归一化。原 function/name/control-flow、HTML/JS/CSS、URL、class、比较与 dispatch 字面量保持受保护代码；Material Icons ligature 不是语言。只有 frozen `$alt` 的可见 sink 字面量属于语言，其他赋值仍是代码。动态数据引用不会作为待翻译字符串重复投影。

Homepage 的 YAML title/summary 是可见 metadata，`template: true` 保持机器字段。源码 coordinates 保留原始 bytes，整个 Page 的 boundary/protected token 表示可逐字节还原 frozen source。textarea 中的代码、字符串与无法安全识别的注释保持 protected；可识别的自然教学注释复用现有 `SafeTeachingCommentBodies`，不扩大其保守规则。

YAML 使用标准库实现的 `internal/sitecontent/shellyaml` 严格冻结语法子集，输出带源坐标的 AST；覆盖本批实际使用的 nested mapping/sequence、flow mapping、quoted scalar 与 `|`/`|-` literal quote。它不是通用 YAML reader，额外语法必须显式扩展，不能近似解释。duplicate keys、alias/anchor、unknown fields/types、multiple documents、folded/multiline quoted scalar、unsupported escapes 与额外尾部均 fail closed。不新增 module dependency，不改正式 `go.mod`/`go.sum`，不做 tidy。package import-path display title、testimonial name/company 作为固定身份保留在 source 中，记录于 `locked_visible_identity_locators`，不成为译文生成槽位。quote 中的强调 markup 保持 protected。

## Exact carry 与 affected scope

scope 绑定 accepted predecessor completion 的 exact SHA、完整 shell/Learn-Docs closure、既有 independent Surface evidence、validated target SHA 及原 structured output。旧五槽位按原保护表示独立逐一映射，不拆开 bundled navigation block 后冒充已有逐 label 审核。仅规范化 source whitespace；不依据中文相似度、重复字词或包级 SHA 猜测 carry。每条 mapping 明示 old slot、exact-source candidates、唯一可接受 destination、carry/affected、原因和 evidence。source、role/context、encoding、protected structure 或 target 合法性无法证明均 affected；ambiguous destination 也不 carry。

`language_units` 分别列出 Homepage Page、四种 structured sources、translation body 与49个已接受 Learn/Docs Page。每个 source 的 `generation_required_slots` / `exact_carry_slots` 给出完整 exact ID set；Page 的 `formal_unit_id` 仍只有一个。Learn/Docs carry 保持原 physical source、Page ID、route、accepted target 与 closure/Surface evidence，不因 shared chrome 新增而使 Page 本身 stale。translation 正文 source/target 未变则 carry；shared chrome 是独立待完成语言 source，不要求重新生成正文。

## Foundation 时点的 terminology impact（历史事实）

提取器为五个新 source 生成 proposed contributor contexts，与当前 unified corpus 按 source 比较，记录新 source/context count、changed 与 retired predecessor source。projection 排除 template code 与机器 gaps；artifact 中的 proposed contributors **尚未注册到当前 corpus**，不替代当前 glossary Review。`terminology_impact.affected=true` 时唯一下一阶段先处理 corpus successor → Unified Glossary Refresh → Independent Glossary Review → compatibility，随后才可进入 affected language Generation。

foundation 当时未修改 corpus、glossary、candidates、Snapshot、QC、finalization、closure、completion、activation 或 Surface evidence。随后五份 source 已通过 corpus successor 进入 current unified corpus，zh-CN Unified Glossary Review 012 已覆盖其 corpus identity。本轮正式 source adoption 不重做该 glossary lifecycle。新 Homepage 仍须共享 Generation → restoration → validation → Snapshot → Independent QC → exact revision/re-QC → A-only finalization/current authorization；四份 structured 仍须真实 Generation、reconstruction/validation 与 Integrated Surface。旧 pending artifact 本身没有 activation 能力。

## Current adoption、六份原子 closure 与 JIT

`site-content shell-adopt --input <frozen-foundation.json>` 只读构造 adoption；`--apply --prior-sha256 <原 GlobalPath 文件 SHA>` 执行 exact CAS、immutable adoption save 和 atomic no-overwrite current pointer。`shell-adoption-check --input <adoption.json>` 重建 source/manifest/parser/corpus projection、package/dependency identity。未知 parser、source/target/structure drift 或不完整 inventory fail closed。旧 foundation 和全部历史 evidence immutable，不 mass migrate。

current shell closure 必须包含：`_content/index.md` 的 shared Page A-only finalization + site.tmpl/menus/resources/testimonials 的四份完整 validated structured bytes + exact current accepted translation.html carry。缺任何一份均不 complete；旧 home.html 不替代 Homepage。正常 closure 复用 structured baseline/delta primitive 和共享 Page finalization，不新增 QC 状态机。

当前唯一 JIT Generation 顺序：先 Homepage 的一份 shared TU Bundle；其结果正式导入、restore/validation PASS 后，才导出当时 current 的 structured assets Bundle（四份新 shell assets 与真实 affected Learn YAML，Expected 由机器确定）。不预导后者。随后 Homepage 独立 QC/A-only/current authorization，Learn closure 消费现有 qc-012，shell 六份原子 closure，再进入一次 Integrated Surface。现有 Learn/Docs 49 Page 不 reopen、regenerate、re-QC。

runtime `/` 消费 completed `_content/index.md`，所有正式 Site 页面共用 completed site.tmpl/menus；resources/testimonials 与 accepted Learn data 供完整 Homepage 模板消费。沿用冻结 upstream tmplfunc named-template semantics，不把模板或内部 slots 另造 TU。company grid 只读取 manifest-bound 的冻结 machine/brand metadata，不注册 solutions routes；未覆盖链接仍 go.dev，未覆盖 HTTP 仍 true 404。`/tour/**` 保持独立 renderer。Homepage TU 不重新 A/B/C/D；Surface 检查组合质量、changed shared structured，translation exact carry 不伪装新审核。

## 离线命令与 currentness

```sh
# 只读 preflight，完整输出 pending affected scope：
go run -mod=readonly ./cmd/tour-i18n site-content shell-successor --locale zh-CN
# 只追加 immutable source/affected artifacts；不启动任何语言或 runtime 操作：
go run -mod=readonly ./cmd/tour-i18n site-content shell-successor --locale zh-CN --apply
# 使用上一条打印的真实 affected_scope_path：
go run -mod=readonly ./cmd/tour-i18n site-content shell-successor-check --input <affected_scope_path>
```

artifacts 位于 `data/site-shell/successor/sources/<identity>.json` 与 `affected/<locale>/<identity>.json`。current-check 重算 extraction、source/manifest bytes、predecessor evidence、mapping、terminology 与完整 exact scope，并要求 strict canonical JSON 与 self-addressed path 完全一致。parser dependencies 显式绑定 repository path/SHA，不绑定 whole repository SHA，不读取外部 checkout 或旧 physical source ZIP。不依赖 renderer、`/dl` 或联网服务。
