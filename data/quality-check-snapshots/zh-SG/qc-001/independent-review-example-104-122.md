# zh-SG TranslationUnit Quality Check — `qc-001` / Example 104–122

**Attachment / current identity check：PASS**

- locale: `zh-SG`
- snapshot_id: `qc-001`
- working_set_kind: `example`
- stable indexes: `104–122`
- unit_count: `19`
- bundle identity: `167c6d93acc2d8154afdb494453e23e523d1d5c89da5b9a937231925d297f9be`
- ZIP SHA-256: `b093deb00603da324693ee9fc0925275d7e2e31ff138c4ed59566fbae34d1519`
- manifest inventory/hash: **PASS，48/48 exact-match**
- input / validation / Snapshot identity: **19/19 PASS**
- automatic validation: **19/19 `passed`**
- current QC scope: indexes `104–122` 均为 `missing_quality_check / quality_check_required`
- reviewed scope: **Example stable indexes 104–122，19/19**

本组正式 QC 结论：**FAILED（A-only gate 未满足）**。

| Index | unit_id | Rating | Finding |
|---:|---|:---:|---|
| 104 | `example:basics/numeric-constants.go` | **B** | teaching comment `Create a huge number by shifting a 1 bit left 100 places.` 被译为“将一个 **1 位**向左移动 100 位……”。“一个 1 位”在简体中文中语法生硬且技术指向含混，容易把“值为 1 的 bit / 将 1 左移”误读成“一个一位的对象”。需准确表达 **将值为 1 的位/数值向左移动 100 位** 这一关系，同时保持代码和数值不变。 |
| 105 | `example:basics/type-inference.go` | A | — |
| 106 | `example:concurrency/channels.go` | A | — |
| 107 | `example:concurrency/exercise-equivalent-binary-trees.go` | A | — |
| 108 | `example:concurrency/exercise-web-crawler.go` | A | — |
| 109 | `example:concurrency/mutex-counter.go` | A | — |
| 110 | `example:flowcontrol/if-and-else.go` | A | — |
| 111 | `example:generics/index.go` | A | — |
| 112 | `example:generics/list.go` | A | — |
| 113 | `example:methods/exercise-reader.go` | A | — |
| 114 | `example:methods/exercise-stringer.go` | A | — |
| 115 | `example:methods/interfaces-are-satisfied-implicitly.go` | A | — |
| 116 | `example:methods/interfaces.go` | A | — |
| 117 | `example:moretypes/append.go` | A | — |
| 118 | `example:moretypes/exercise-fibonacci-closure.go` | A | — |
| 119 | `example:moretypes/pointers.go` | A | — |
| 120 | `example:moretypes/slice-len-cap.go` | A | — |
| 121 | `example:moretypes/slices-of-slice.go` | A | — |
| 122 | `example:moretypes/struct-literals.go` | A | — |

**汇总：A=18，B=1，C=0，D=0。**

本次严格只审核了 **Example stable indexes 104–122，19/19**。**未重审 Page indexes 1–103，也未修改其既有结论。**

未生成 replacement、未修改译文或仓库文件，也未执行 record、finalize、promotion、preview、publish 或 Production。停止于本组独立 Reviewer 结论。
