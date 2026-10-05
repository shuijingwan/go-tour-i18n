# zh-SG TranslationUnit Quality Check — `qc-001` / Page 1–60

**Attachment / identity check：PASS**

- locale: `zh-SG`
- snapshot_id: `qc-001`
- working_set_kind: `page`
- stable indexes: `1–60`
- unit_count: `60`
- bundle identity: `09eb1a9df621d79793de1b064209697cd62f245bad135783a8ef1c42f4655e00`
- ZIP SHA-256: `72ecbd4a7833dcd2715889b24022baa6fca4b1b1c3c76cca0136527d81b3c368`
- manifest inventory/hash: PASS
- relevant input / validation / Snapshot identity: 60/60 PASS
- automatic validation evidence: 60/60 `passed`
- reviewed scope: **Page stable indexes 1–60，60/60**

本组正式 QC 结论：**FAILED（A-only gate 未满足）**。

| Index | unit_id | Rating | Finding |
|---:|---|:---:|---|
| 1 | `welcome/1` | A | — |
| 2 | `welcome/2` | A | — |
| 3 | `welcome/4` | A | — |
| 4 | `welcome/5` | A | — |
| 5 | `welcome/3` | A | — |
| 6 | `basics/1` | A | — |
| 7 | `basics/2` | A | — |
| 8 | `basics/3` | A | — |
| 9 | `basics/4` | A | — |
| 10 | `basics/5` | A | — |
| 11 | `basics/6` | A | — |
| 12 | `basics/7` | A | — |
| 13 | `basics/8` | A | — |
| 14 | `basics/9` | A | — |
| 15 | `basics/10` | A | — |
| 16 | `basics/11` | A | — |
| 17 | `basics/12` | A | — |
| 18 | `basics/13` | A | — |
| 19 | `basics/14` | A | — |
| 20 | `basics/15` | A | — |
| 21 | `basics/16` | A | — |
| 22 | `basics/17` | A | — |
| 23 | `flowcontrol/1` | A | — |
| 24 | `flowcontrol/2` | A | — |
| 25 | `flowcontrol/3` | A | — |
| 26 | `flowcontrol/4` | A | — |
| 27 | `flowcontrol/5` | A | — |
| 28 | `flowcontrol/6` | A | — |
| 29 | `flowcontrol/7` | A | — |
| 30 | `flowcontrol/8` | A | — |
| 31 | `flowcontrol/9` | A | — |
| 32 | `flowcontrol/10` | A | — |
| 33 | `flowcontrol/11` | A | — |
| 34 | `flowcontrol/12` | A | — |
| 35 | `flowcontrol/13` | A | — |
| 36 | `flowcontrol/14` | A | — |
| 37 | `moretypes/1` | **C** | 两条 preformatted teaching comment 将 “read/set `i` through the pointer `p`” 译成“通过指针读取 `i` `p`”和“通过指针设置 `i` `p`”。当前中文语序不成立，并丢失了 **`p` 是所经由的指针、`i` 是被读取/设置变量** 的技术关系。需修正这两条注释的语义关系，同时保持 `i`、`p` 代码标识符身份不变。 |
| 38 | `moretypes/2` | A | — |
| 39 | `moretypes/3` | A | — |
| 40 | `moretypes/4` | A | — |
| 41 | `moretypes/5` | A | — |
| 42 | `moretypes/6` | A | — |
| 43 | `moretypes/7` | **B** | `a[1:4]` 的说明把 source 的 “elements 1 through 3” 写成“`a` 的**第 1 到第 3 个元素**”。中文的“第 N 个元素”通常表达序数，会与 Go 的零基索引产生歧义：`a[1:4]` 实际选择的是**索引 1、2、3** 的元素。需明确这里描述的是索引范围，避免序数含义。 |
| 44 | `moretypes/8` | A | — |
| 45 | `moretypes/9` | A | — |
| 46 | `moretypes/10` | A | — |
| 47 | `moretypes/11` | A | — |
| 48 | `moretypes/12` | A | — |
| 49 | `moretypes/13` | A | — |
| 50 | `moretypes/14` | A | — |
| 51 | `moretypes/15` | A | — |
| 52 | `moretypes/16` | A | — |
| 53 | `moretypes/17` | A | — |
| 54 | `moretypes/18` | A | — |
| 55 | `moretypes/19` | A | — |
| 56 | `moretypes/20` | A | — |
| 57 | `moretypes/21` | A | — |
| 58 | `moretypes/22` | A | — |
| 59 | `moretypes/23` | A | — |
| 60 | `moretypes/24` | A | — |

**汇总：A=58，B=1，C=1，D=0。**

本次仅审核了 **Page stable indexes 1–60**。**未审核 index 61–122**，也未执行 record、finalize、promotion、preview、publish 或任何 Production 操作；未修改译文、未生成 replacement。

唯一下一步：将 `moretypes/1` 与 `moretypes/7` 的上述 findings 返回原 zh-SG Generation session，进入正式 revision 流程；本 Reviewer 在此停止。
