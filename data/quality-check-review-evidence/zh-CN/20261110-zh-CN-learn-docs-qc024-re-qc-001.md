本轮 `learn-docs-v1-qc-024` successor re-QC 已完成 **32/32 fresh Page TranslationUnit** 的独立审核。

**INPUT_GATE=PASS**：ZIP 为 `1,834,044` bytes，SHA-256、Bundle identity、manifest inventory 全部逐项匹配；Snapshot/predecessor、32 个 pending stable indexes、19 个 carry-forward Unit、相关 Generation input / validation / candidate identities 均一致。19 个 carry-forward Unit 未重新评级。

本轮结果为：

**A=9 / B=21 / C=2 / D=0**

其中两个 C 是 stable index **6** 与 **23**，均存在实质性 source-semantic / relation defect；其余 non-A 主要是可 bounded 修复的中英文标点、representation spacing、普通英文残留或局部自然度问题。

## 32 个 fresh Unit 正式结果

| Index | Unit ID | Rating | Finding |
|---:|---|:---:|---|
| 2 | `393fb67915a49876749590fd6480c9a3482597a9cebdd61e72c9a5f3d1449e2c` | **B** | **Category:** visible punctuation spacing. 完整 Page same-class scan 确认 1 处：测试函数说明中 `<code>Test</code> 开头）` 与句号之间残留可见空白，呈现为 `开头） 。`。**Scope:** 仅该 Page 中非代码自然语言的“可见文本后存在空白再接中文标点”predicate；本 Page 确认 1 处。保留代码、identifier、HTML 和测试语义。 |
| 3 | `69b4a569a5bf08fc858a3aee621392dcb718a11e39e1e53842b1dabf25de8a59` | **A** | none — 完整 Page source/target、模块调用关系、`replace`/`require`、目录结构及 protected code identity 均可接受，无 blocking finding。 |
| 5 | `704fb9e2968f11379508a1f22a9837e154dd94ef14e4f781ecd7d7ed701a13a3` | **B** | **Categories:** source-style punctuation residue / ordinary-English residue / technical-boundary. 完整 Page scan：① visible natural-language prose 中仍有 **7** 个 ASCII `--` 作为句间/列表标点；② `key/value 对` 为 1 处普通英文自然语言残留；③ `<code>:=</code>运算符` 为 1 处 technical identifier→中文边界粘连。**Scope:** 仅这三类 predicate 在当前 Page 内的全部实例；保留 `:=`、代码、链接和模块语义。 |
| 6 | `36346dcd0f2901eadeb8f4ec8c70dc4c0e709457e72ed30880c9e471e18c398a` | **C** | **Category 1: semantic-role mistranslation.** source `use the CLI for the DBMS itself` 被译为 `使用 CLI 自带的 DBMS`，把“DBMS 自身的 CLI”反转成“CLI 自带的 DBMS”。**Category 2: semantic attachment/naturalness.** source `an SQL query designed to return multiple rows` 当前为 `设计为返回 SQL 多行的查询`，把 `SQL` 错接到“多行”。**Category 3:** 完整 Page scan 发现 **18** 处可见空白后接中文标点，以及 **2** 个自然语言 ASCII `--` punctuation instances。**Scope:** 当前 Page 上述三类；不得修改命令、SQL、DBMS/API identity 或代码。 |
| 7 | `0ecf62b07a368ffb8b06193d58be2c23e812fcbb51bf4f5810c67ef31107d3ff` | **B** | **Categories:** formatting/natural-language terminology. 完整 Page scan 发现 **18** 处可见空白→中文标点；调试器链接因换行呈现为 `设置 调试器`；UTF-8 诊断段的 `替代字符 UTF-8 �` / `替代 UTF-8 字符` 对 source “UTF-8 replacement character” 语序不自然、概念边界不清。**Scope:** 当前 Page 的上述 whitespace predicate、该链接标签和该 replacement-character 说明；代码、rune/byte、`-fuzz` 等 technical identity 保持。 |
| 9 | `5b475205acbf5bd75bf05d04a110ff6d39bbc4e6b377943170aa2151ce86bed8` | **B** | **Category:** source-style ASCII dash punctuation. 完整 Page scan 确认 **3** 个 natural-language `--`：`例如，--rsc.io/quote--...` 两个边界及 `最新版本--v1.5.2` 一个边界。**Scope:** 仅 visible prose 中作为中文标点使用的 ASCII `--`；版本号、模块路径和代码保持。 |
| 11 | `1829c72166fe3d16154c3f36be4e436b51e8d5ff1725c426d2564bf6a4c44275` | **B** | **Category:** visible punctuation spacing. 完整 Page scan 确认 1 处：`` `Parse` 函数 。``。**Scope:** 当前 Page 非代码自然语言中空白后接中文句号的同类实例；本 Page=1。保留 `Parse`、漏洞 IDs、govulncheck 输出 labels。 |
| 14 | `22e2b081cf7eb38b8a4bba0b99a2157e02e96f415f81f81e561b4aa42f15e957` | **A** | none — 教程导航、技术名称、描述关系与链接语义完整准确，无 blocking finding。 |
| 15 | `b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c` | **B** | **Categories:** visible punctuation spacing / untranslated natural-language residue. 完整 Page scan 确认 **12** 处“非空字符 + whitespace + 中文标点”可见排版缺陷；另有 `和/或（and/or）`，中文含义已表达后仍保留普通英文 conjunction。**Scope:** 当前 Page 这两类 predicate；JSON、API、Go type、code identity 不变。 |
| 18 | `c3dfdb3d129df54ca09e79e21054c0afa5d3ec1bbd4e6cbeb227476bd8dcb0c3` | **B** | **Category:** visible punctuation spacing. 完整 Page same-class scan 确认 **11** 处自然语言文本在中文标点前保留可见 whitespace，例如 `模块的路径 。`、`依赖项 。`、`唱片集合都会丢失 ，`、`代码 。`。**Scope:** 完整 Page 中该 exact whitespace→Chinese-punctuation predicate；protected code comments、Gin/API/JSON identity 不在 scope。 |
| 19 | `b1d98230ae3df4fc789951b9434cebc92a097caf92e0493eb7d4e81f3135ea7e` | **B** | **Category:** technical identifier / Chinese prose boundary. 完整 Page scan 确认 1 处：`` `go work use [-r] [dir]`会为... ``。**Scope:** inline command closing boundary immediately touching Chinese lexical prose；当前 Page=1。保留命令字节、参数及 workspace semantics。 |
| 20 | `d799614a9f1efd848505c615a88f2fb0f092ac22f743b3da042e12de1127fd8b` | **B** | **Category:** spurious Markdown paragraph break. `有关 Context 的更多信息，请参阅` 与 `[Go 并发模式：Context](/blog/context)。` 之间存在空段落边界，把 canonical source 的单个完整句子拆成两个 paragraph，并使前一段句法未完成。完整 Page same-class scan确认 **1** 处。**Scope:** 仅此 paragraph-boundary predicate；`Context`/link identity 不变。 |
| 21 | `710755b7111a6d99c424ec6d345cbbe68672a3dab3ba998432cd28fd258b42ce` | **B** | **Categories:** localized comment naturalness / connection-description naturalness / punctuation spacing. `获取新唱片为客户端生成的 ID。` 的修饰关系含混；`用于保留的专用连接。` 对 source `For use with reserved connections` 表达重复生硬；完整 Page另确认 **2** 处 whitespace→中文标点。**Scope:** 仅上述 comment sentence、reserved-connection description 与同类 punctuation spacing；保留 `ID`、`Conn.ExecContext`、SQL/API identity。 |
| 22 | `d61aeba25412a5486456e3e5f2d3910e0bf27381f71b37eba29853d462174a1b` | **B** | **Categories:** source-style punctuation / visible punctuation spacing. 自然语言段落 `取消函数的执行 -- 包括数据库操作` 保留 1 个 ASCII `--`；另确认 1 处链接后 whitespace→中文逗号。代码注释中的 `enough`、`quantity`、`Commit` 属于当前 protected technical identities，本 finding **不要求翻译或删除它们**。**Scope:** 仅上述 2 个可见 prose/punctuation instances。 |
| 23 | `4708d52292821a7a838084e4407ab41f240bcfb3d77fc173497374f0e4788b70` | **C** | **Category 1: semantic-role mistranslation.** source `including ORM libraries for higher-level access to relational databases` 当前为 `包括用于 ORM 更高层次访问关系型数据库的库`，错误/含混地把 ORM 变成用途对象，而不是“提供更高层次访问的 ORM 库”。**Category 2:** `NoSQL数据存储` 在同一介绍段落形成 technical-name→中文边界粘连（该段确认 1 个明确 occurrence）；另有 1 处 whitespace→中文标点。**Scope:** 当前 Page 对应介绍句及同类明确 boundary/spacing；ORM/NoSQL technical identities 保持。 |
| 24 | `e1fb550acbf722fc3146f3acdf97ea9fc2a9bb5bfbef7742a40e0a02090d56c2` | **A** | none — 连接池、`SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxIdleTime`/`SetConnMaxLifetime` 等含义准确，自然度达到 A。 |
| 26 | `dd353848a883a0ce4efa4a6c72b49d08551f13701c8b98b35520092d45c6ebbe` | **B** | **Categories:** punctuation spacing / reserved-connection wording. 完整 Page scan 确认 **6** 处可见 whitespace→中文标点；`用于保留的专用连接` 对 `For use with reserved connections` 表达重复生硬。代码注释中的 protected `album` / `id` identity 不作为普通自然语言强制翻译。**Scope:** 当前 Page 6 个 spacing instances + reserved-connection description；API/SQL/code 不变。 |
| 28 | `ca5a4ed9af214836a5b317eb6961c46caa4daf529a5434b12251e58b94b36b16` | **A** | none — SQL injection 的参数化查询含义准确；代码块中的 protected security comment/code identity 不作为普通自然语言 reopen。 |
| 29 | `83847dd2c9d2c76a864241e435e527d39c8697366d30bed775f4a77d2da50d9a` | **B** | **Category:** visible punctuation spacing. 完整 Page scan 确认 1 处：模块设计实践 bullet 末尾 `新版本 \t。`。**Scope:** 仅非代码自然语言 whitespace→中文标点 predicate；当前 Page=1。 |
| 30 | `61c3a8166079f2411ded5981623c0e2c7945ee164f8ac11b17d4f956b214174b` | **B** | **Categories:** vendoring terminology / punctuation spacing. source translatable `vendoring`/`vendored dependencies` 当前分别呈现为 `[ vendor 机制]`、`已 vendor 的依赖项`，形成不自然的英文动词/混排；完整 Page另确认 **3** 处 visible whitespace→中文标点。**Scope:** 当前 Page 中自然语言 `vendoring/vendored` occurrences（不包括 `vendor/` 目录、`go mod vendor` 命令、`-mod=vendor`）及该 punctuation predicate。 |
| 32 | `7eb8c6d141170074062064dfc1367ae28002c16a1cd825ee5666fc73c0e3b8b5` | **B** | **Category:** source-style ASCII dash punctuation. `每种发布类型 --主版本、次版本、补丁版本或预发布版本-- 对...` 中 **2** 个 ASCII `--` 被直接用于中文插入语标点。完整 Page same-class scan=2。**Scope:** 仅该 natural-language punctuation construction；版本语义和 identifiers 保持。 |
| 33 | `82a1f9b16b20d60a538c1873daf9afb5c446aded1d2bff16e3b5650ed6bafb1e` | **B** | **Categories:** source-style punctuation / ordinary-English residue / punctuation spacing. 完整 Page scan：**4** 个 natural-language ASCII `--`（`test --`、`example --`、`--比如...--`）；1 个 `（and/or）` 英文 conjunction residual；**4** 处 visible whitespace→中文标点。**Scope:** 仅这三类 predicate；module path、fork、pull request、Go commands/identities 保持。 |
| 34 | `cb44a567ce61c076973f88f9dd2bbc98f7cc3bcff87106924270bdc9be6f78a6` | **A** | none — 模块源代码、VCS、许可证/仓库布局及发布关系准确；命令中的 `--all` 等 protected bytes 不构成自然语言 finding。 |
| 37 | `9386cbdb9abcc5044d6c66064f00ff3d2fcc864b8def7a63fdebf81e3a62980e` | **A** | none — semantic version、pseudo-version、major/minor/patch/pre-release 等概念准确，技术 identity 合理。 |
| 38 | `6ad2588c209947883baa68e81fcba2269136e3292b0f5aedd83b1923ad20192a` | **A** | none — 安全最佳实践、govulncheck、fuzzing、race detector、CI/CD 等含义完整且自然，无 blocking finding。 |
| 41 | `8042e585ed8b19f87a6b95c1e1a71bc43a9ab550e6308731005694a0dd6d061d` | **B** | **Category:** untranslated ordinary-language abbreviation. `用户取消进程（e.g，使用 Ctrl^C）` 中 `e.g` 为 source 自然语言举例缩写，不是 technical identity。完整 Page same-class scan确认 **1** 处。**Scope:** 仅 ordinary-language example abbreviation；`Ctrl^C`、OSS-Fuzz、Issue IDs 等保持。 |
| 42 | `950eb43dcbeff65a683ee58307184e4e75652f9ee881cd046f1c4cf7529498d0` | **B** | **Category:** untranslated ordinary-language abbreviations. 完整 Page scan确认 **3** 处：1 个 `e.g.` 和 2 个 `i.e.`（不可恢复 worker 示例、RPC 结果说明、ASCII 字节说明）。**Scope:** 仅作为普通说明语使用的 `e.g./i.e.`；worker/RPC/RNG/libFuzzer/API technical terms 保持。 |
| 44 | `78bb61869a1d012f449488964d8282dda51c0664f2cc8b65c00eec2054f1d166` | **B** | **Category:** untranslated ordinary-language residue. 完整 Page scan确认 **2** 处：`（e.g CRITICAL、HIGH、MEDIUM、LOW）` 中 `e.g`，以及 `或两者兼有（and/or）` 中重复英文 conjunction。**Scope:** 仅这两个 ordinary-language residues；PUBLIC/PRIVATE/URGENT、CVE/CVSS、邮件 subject/UI quoted strings 等 technical/required identities 保持。 |
| 46 | `970d9162f36489ed1cea186c70428d536fe1f9485b2e255537b735e3c275cb00` | **B** | **Category:** untranslated natural-language emphasis label. `**IMPORTANT**：` 为 source 普通强调词，在完整 zh-CN 正文中仍保持英文；complete Page same-class scan确认 **1** 处。**Scope:** 仅该 emphasis label；CVE ID、PUBLIC、issue tracker/link identities 不变。 |
| 47 | `a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f` | **A** | none — OSV schema、数据库 API、排除原因、`NOT_IMPORTABLE`、review status 等语义准确；technical schema/field identifiers 的原文保持合理。 |
| 49 | `ff777d1f41091a8b8269f4787a6aa78f2097263412b0e01ba0c7cdd73fcf6775` | **B** | **Category:** redundant untranslated natural-language label. `**注意（NOTE）：**` 已有完整中文“注意”，仍重复保留普通英文 `NOTE`。完整 Page same-class scan确认 **1** 处。**Scope:** 仅该 label residual；govulncheck、Go issue links、architecture identities 保持。 |
| 50 | `00a849ced4775e5dc75d72ecbdc81549135697ba84c51d26d05ac84d4989b866` | **A** | none — Docs hub 的导航、`GOPATH`/`bin` 边界、资源标题、技术名称与当前 source meaning 均达到 A。 |

## 需要 revision 的 Units

**B（21）**

`2,5,7,9,11,15,18,19,20,21,22,26,29,30,32,33,41,42,44,46,49`

**C（2）**

`6,23`

对应 Unit IDs：

- index 6 — `36346dcd0f2901eadeb8f4ec8c70dc4c0e709457e72ed30880c9e471e18c398a`
- index 23 — `4708d52292821a7a838084e4407ab41f240bcfb3d77fc173497374f0e4788b70`

**A（9）**

`3,14,24,28,34,37,38,47,50`

```text
===== FINAL =====
RESULT=PASS
STAGE=learn-docs-v1-qc024-successor-re-qc
REVIEW_ID=20261110-zh-CN-learn-docs-qc024-re-qc-001
LOCALE=zh-CN
SNAPSHOT_ID=learn-docs-v1-qc-024
PREVIOUS_SNAPSHOT_ID=learn-docs-v1-qc-023
CONTENT_PACKAGE=learn-docs-v1
BUNDLE_SHA256=8e724a8cffbe0c8ce969b330568914c450f5a322ef42426474e8d77b442b7e98
BUNDLE_IDENTITY=a0f2edf77e5feb80260c17ae8d1a5a0414cba415fa4cd3a1954248e782ee86b6
INPUT_GATE=PASS
FORMAL_UNITS=32
REVIEWED_UNITS=32
A=9
B=21
C=2
D=0
CARRY_FORWARD_UNITS=19
FRESH_JUDGMENT=yes
SAME_CLASS_SCAN=yes
REPLACEMENT_GENERATED=no
REPOSITORY_MUTATION=none
SEMANTIC_REMOTE_CALLS=0
NEXT=machine-record
STOP=32/32 fresh judgments complete; record ratings/findings before exact revision lifecycle
```

本轮 Reviewer 工作到 32/32 judgment 完成即停止；没有重新审核 19 个 carried A，也没有生成 replacement 或执行 repository mutation。
