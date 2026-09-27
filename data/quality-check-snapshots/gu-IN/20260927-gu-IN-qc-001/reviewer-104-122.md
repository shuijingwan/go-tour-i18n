# gu-IN 首次独立 TranslationUnit QC — Example stable indexes 104–122

- locale: `gu-IN` (ગુજરાતી); independent Reviewer: original gu-IN ChatGPT GPT-5.6 Sol High session (no gu-IN Generation, revision or replacement).
- stage: `translation-quality/v1`; immutable full Snapshot: `20260927-gu-IN-qc-001`.
- working set: 19 complete Example Go files, stable indexes 104–122; batch `chatgpt-gu-IN-003`, attempt 1 for all.
- reviewer ZIP: `/tmp/gu-IN-20260927-gu-IN-qc-001-qc-104-122.zip`.
- reviewer ZIP SHA-256: `8002d817ad5aa71e0fad6fd30bd7dee8e9e7c227ab704d070051c0535a700ec6`.
- reviewer bundle identity: `d937281ab41ee2022320cc1ad674e1dcc4d8894a4a4d02c2e203aad2effd680b`.
- full Snapshot manifest SHA-256: `9e2439603ddb028435537e8ff5294709350b395f951df085805d9ccc4dfdd2a9`.
- full glossary SHA-256: `9e9698f4281fb2be23f5532d3e6804f85eec5dbc25759044ce8b233d7e85c377`.
- exact ZIP inventory: 48/48 declared members, hashes and sizes verified; 49 total entries with manifest; ZIP CRC PASS.
- repository `quality-check reviewer-bundle-check`: CURRENT for this ZIP, snapshot, kind and stable working set.
- source, candidate, protected generation input and validation files: all 19 SHA-256 matched the immutable Snapshot/batch manifest and per-file validation evidence; 19/19 `passed`.
- independent source/target structure comparison: 19/19 full files preserve exact line count, code, identifier/string content, build directives and code-side comment delimiters. All changed lines are eligible `//` teaching comments only.
- read complete English original, Gujarati candidate and official glossary for each of the 19 Example files; checked technical concepts, target-language clarity, glossary, comment-variable references, conditions and teaching accuracy.
- prior groups: machine-recorded Pages 1–103 A=97, B=3, C=3, D=0; these Page decisions were not reviewed again or modified.
- **independent Example rating result: A=17, B=2, C=0, D=0 (19/19).**
- This file is Reviewer Markdown evidence only; no `quality-check record` performed and no translation/replacement generated.

## 逐 Example 独立评级

| Stable index | Unit ID | Batch | Attempt | Automatic validation | QC rating | Reviewer note |
| ---: | --- | --- | ---: | --- | :---: | --- |
| 104 | `example:basics/numeric-constants.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Bit shift count and binary value explanation preserved. |
| 105 | `example:basics/type-inference.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | The one editable instructional comment is natural and complete. |
| 106 | `example:concurrency/channels.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Channel c send/receive and sum semantics preserved. |
| 107 | `example:concurrency/exercise-equivalent-binary-trees.go` | `chatgpt-gu-IN-003` | 1 | passed | **B** | Minor Gujarati redundancy and ambiguous locative around t1/t2; see finding EX-107. |
| 108 | `example:concurrency/exercise-web-crawler.go` | `chatgpt-gu-IN-003` | 1 | passed | **B** | Concurrent fetching task loses explicit plurality of URLs; see finding EX-108. |
| 109 | `example:concurrency/mutex-counter.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Mutex and one-goroutine-at-a-time explanation accurate; goroutine kept. |
| 110 | `example:flowcontrol/if-and-else.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Variable v scope restriction after if block expressed correctly. |
| 111 | `example:generics/index.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | comparable constraint, equality, slice and Index behavior accurate. |
| 112 | `example:generics/list.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Generic singly linked list holding arbitrary type values accurate. |
| 113 | `example:methods/exercise-reader.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | MyReader.Read signature and TODO accurate. |
| 114 | `example:methods/exercise-stringer.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | IPAddr.String method signature and TODO accurate. |
| 115 | `example:methods/interfaces-are-satisfied-implicitly.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Implicit interface implementation semantics accurate. |
| 116 | `example:methods/interfaces.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Vertex vs *Vertex interface implementation distinction accurate. |
| 117 | `example:moretypes/append.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | nil slice behavior, growth and multiple-element append accurate. |
| 118 | `example:moretypes/exercise-fibonacci-closure.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Function returning another function returning int explained accurately. |
| 119 | `example:moretypes/pointers.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Pointer referents and read/write/divide effects accurate. |
| 120 | `example:moretypes/slice-len-cap.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Slice zero length, re-extension and dropping first two elements accurate. |
| 121 | `example:moretypes/slices-of-slice.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Tic-tac-toe board and turn-taking comments accurate. |
| 122 | `example:moretypes/struct-literals.go` | `chatgpt-gu-IN-003` | 1 | passed | **A** | Struct literal types, implicit Y:0 and zero fields accurate. |

## B/C/D 精确 findings（不生成 replacement）

### EX-107 — stable index 107 / `example:concurrency/exercise-equivalent-binary-trees.go` — B / minor

**原文证据：**

```go
// Same determines whether the trees
// t1 and t2 contain the same values.
```

**Gujarati 候选证据：**

```go
// Same તપાસે છે કે બંને ટ્રીમાં
// t1 અને t2 માં એકસરખાં મૂલ્યો છે કે નહીં.
```

**问题：**「બંને ટ્રીમાં」后紧接「t1 અને t2 માં」，对同一对树重复使用方位格「માં」，让 `t1`/`t2` 的修饰关系拗口、不自然。英文只是说明函数 Same 比较树 t1、t2 是否含有相同的值；当前句子虽能勉强理解，但达不到最终发布质量的教学注释流畅度。

**可操作 finding：** 仅润色 `Same` 的这两条允许翻译的教学注释，清楚地把 `t1`、`t2` 作为正在比较的两棵树，各自拥有相同值这一判断关系只表达一次；原样保留函数名、标识符、注释分隔符、完整 Go 代码和文件布局。

### EX-108 — stable index 108 / `example:concurrency/exercise-web-crawler.go` — B / minor

**原文证据：**

```go
// TODO: Fetch URLs in parallel.
```

**Gujarati 候选证据：**

```go
// TODO: URL ને સમાંતર રીતે મેળવો.
```

**问题：** 英文复数 `URLs` 明确要求对多个 URL 发起并行获取。Gujarati 注释单数宾语「URL ને」没有明确保留多个 URL 的对象范围，作为 crawler exercise 的核心 TODO，容易被读成将单个 URL 以并行方式获取。该句的 `સમાંતર રીતે` 已准确表达 parallel，并未发生 concurrency/parallelism 混用；问题仅限待获取对象的数量及教学指令的明确性。

**可操作 finding：** 仅在该条允许翻译的 TODO 注释中明确多个 URL 被并行获取，保持 `TODO:`、术语区分、Go 代码和其他受保护内容不变，不增加原文之外的任务要求。

## 完整 SHA-256 绑定清单

| Index | Source SHA-256 | Candidate SHA-256 | Validation SHA-256 |
| ---: | --- | --- | --- |
| 104 | `338906e7c0e0d5e81312c97b37638442b48a621d59e69215be78eed6bd003544` | `29f29ffad5d40d811844ce8e2df4c050f7e37b016354f86c8fbd7d5f28daaf60` | `3c28c30cb7840440eae9ba8560b9f743c39832acfca9cfc9593a14cc2a8d9062` |
| 105 | `fb638103bfaf5990d88398765cf801cee94cd8e199cace94740cebe562afc33d` | `a23dda66b9989ab06cfdcedc60ce9eafa8b7e9142e12d03393e9b450d0f4524f` | `6d3e2ef42b4bfee45f75d1a69e4c57448bfe74323d8b6354a61dd4f65d233315` |
| 106 | `8c3ce8583c8274e42ee0522989bf4cf2003b6f8045bcb6b16a6a79364f97aa70` | `a67c83965d2354aa4a5ea041bf89f2c769ac12891bcb1e992b488e897c6ddd62` | `57c97da3f9c27b743378ea742c7744c72d63d3a01a64c9a735a9c14b52708f09` |
| 107 | `184cbbd1e662b64de6e6928d4af6f2469c7bccdbc9a8b5f477c1b694b9ab02b1` | `ae5898972ba06ddc97a1d79d0fa261a74dbc1d4eb0b420946cf1530307bcea80` | `d93e870747f415740cc744d83227396ebd080584e36bae2a024a6e93683311b5` |
| 108 | `bd6226dcae3b663a3aa1cbf502d840357ac0efb6de46e7a66612ca2834cf5e37` | `7e15035119ad075f96032b8a773354b5371b931c2964a1fca8ead26076dde30b` | `aeb6e4a1a2080b676d1189a7a148e9dd0e56c60ad3f2d6039f458803c3426457` |
| 109 | `74488ecc347a123c8538069175ffeef5ea38577bb37b2c7a9ceb6bba47ab02b4` | `a2994511e28ccc8ab7b466b158500332eaf1ee54e9fda702f6887e7342771c20` | `6eb916ff2bda70ad880a932d10932e1c3b8670aa9da5764681deb63585505792` |
| 110 | `9d3030966f45fb90105c0325c9a090e8ac3fce2e7c4b10106d8f75caf0e536cf` | `bbc35bab8520fbe430e01011e04b881b8dab0e27c2869173fc20f71fe6d50d55` | `54624be492b448a70aab48793f91ab56c1f34630d9f51bd66961e19598780f64` |
| 111 | `398dd72a4b01efa279d571667bedbcccd1bbbceaeb77218635db48edf38fd983` | `a5369b767dd9cb28bcec79c163ed63570c5b6f9a8aaab3ff1dbde9981b4c29c0` | `e7b46ac9054c1c06189b6509494ad7f1c084c729f99417200b94d1467fcf9e34` |
| 112 | `ab72fc4fd321f507abde796abe0830876e0f8bcd8661d9ce4ac34c3da2f68060` | `faa87e394dfa6e34db846bf9f3ec0e325a66ac663bab93877d3b01f0d417906a` | `7681dce78add1cdcb5d3c8abf995b1e245a63ce05616b794489655ceb9b37560` |
| 113 | `7f5acb25434e41b1641fffd602725903cd2e217fb7f8809596c4369ac0900137` | `36f6f96087b159e152edcfd61bd630b632933a4e96709782dedf66bf11c75be2` | `5baa3ad66ca4e1d8e58445946d854b76cd3d581eb1d51f9d1036f87ec05f0c3e` |
| 114 | `b5bd782b7e0e7cd9541df782ccac827e8373a0867011f3fc998b8895fb470efc` | `d1b84eefdd07be279232e71cf604851114a436ce390c077c82c04062474fa6ee` | `1d3a961f54fc5949471310b9c5e623d5e5a12c87e3fa80c2d9a128a117062888` |
| 115 | `44e0699af42c651abaa67a36b4c48143d33a8651adb0a6bb8e285b65ec058dc2` | `7e2a6b54b40b6ae278d995a20e8f0c4f8bae97bd42840b7e1770a9fbc8020b56` | `600b284f9485ae25aa9fc9e8ab56e2afbdade59581fd317b21435fdf1fd80e9a` |
| 116 | `2530ef039212284377a46fe1063db6206d8db3a42a8138c6eed44275bb7eb10b` | `affe0109947f183158a515c1f37808dfd9bc7da291da022d6109a6b7bcc37c85` | `aeba27102d7edda265ca4f575d69486982198c08d5fc09c70a96bdc20111c355` |
| 117 | `655ee5085fec61bcb38c9f029741bb1a06a414f94b8b36ed0362055b0d6b10bb` | `1299606d61966072989ad9fb05438cac4bfdf333d6360831ffacf6a82d669c15` | `c38ed0fbc0a3975603e9dff2caef93b6683b1146bd576b5d4e7fc4e7107d8018` |
| 118 | `a04ba02819628af3c95c045330a5ce733f9f182e755e6ba74e78c0aea602530c` | `597343d5d1dfad77ba14f36364621ea1fcb88806fcc8a9f002faad313e4b4b81` | `156342929be97c8668e3a26f93539f2753e96d64b1a0521beb33b77ecf611035` |
| 119 | `bbc4a25d868979e959aa383f0de7894135e7f8ae4b2d766e653a033178a9a378` | `fc473e2f2763289f97430787d52f8fe7bc6604a81f5fa6b7667f5803d17b556e` | `3de7132c5a5d4a2ba22ba5b8fa95342ea3b325ab9163fb8bc845d79e469a56be` |
| 120 | `b2e5f7a96c8a04998c150ebe6ce344bc198db8b18f4826e6a85ff75ae3033f98` | `9f752763f2b8c57178f5b2d1dfa31ccd0dacf8ac03acd23e718ca850b2f9b145` | `e9ef68dea06e28418144c99f3af7b667f92e124bd0cf8d0379563f851bfa00d1` |
| 121 | `545cde9f9bc3ec940adbdcac0a801f5e834b6ee3cd5bf1b780f6f81d556562b3` | `d7e888c37f74e6da9f8bc23aedda6a34fbd7bea7cb761d407e1b6f9ece0a7779` | `6373c95f2ebe4c6327543f330dc2c4281c242f3adfba35be5e94488c4ece7ff1` |
| 122 | `c54fd1b039d00a926550fd74ac26c6ed6388c8ca6fe043e64ebdaad921ba2761` | `c6ed2ff28f63692b80e8bdf4f9651fb2a8ce9bb245da9567fcdeffeaad818a9f` | `826cd0d6feef8924f9f1abfa95a143529866d197383a00d41104b13073f0bb3c` |

## 本轮边界与交接

- Example initial QC 19/19 complete; 17 A and two B. Both B require original Generation role to revise and independent re-QC on a new full Snapshot after all three initial groups are machine-recorded.
- No re-review of Pages 1–103; their six previous B/C decisions remain unchanged. No `quality-check record`, no `finalize`, no revision, no generation.
- Next action / executor: original gu-IN Generation session or maintainer Local terminal must recheck this original ZIP is CURRENT and atomically record exactly 19 Example ratings into `20260927-gu-IN-qc-001`. Before the first record, check `quality-check preflight`: existing result count 103 (Page A97/B3/C3/D0), pending 25 (six previous non-A Pages plus 19 Example).
- After recording expected full initial QC totals: A=114, B=5, C=3, D=0; pending 8, all revision_required. Then hand off to original Generation role to coordinate a concentrated, strictly scoped Page+Example revision wave without changing historical lineage.
- STOP: original Generation session formal record and verification; Reviewer role waits for later independent re-QC of the new, CURRENT full Snapshot.
