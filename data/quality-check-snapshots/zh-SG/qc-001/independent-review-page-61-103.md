# zh-SG TranslationUnit Quality Check — `qc-001` / Page 61–103

**Attachment / current identity check：PASS**

- locale: `zh-SG`
- snapshot_id: `qc-001`
- working_set_kind: `page`
- stable indexes: `61–103`
- unit_count: `43`
- bundle identity: `267e2059e2667655cc62058a0b33856c66f915b693052a5c308cb502c6b5efa5`
- ZIP SHA-256: `a7e3a678315a254117c1bdffce730a4154a1228f9f89dff0421afa3200612947`
- manifest inventory/hash: PASS，96/96 members exact-match
- relevant input / validation / Snapshot identity: 43/43 PASS
- automatic validation: 43/43 `passed`
- reviewed scope: **Page stable indexes 61–103，43/43**

本组正式 QC 结论：**FAILED（A-only gate 未满足）**。

| Index | unit_id | Rating | Finding |
|---:|---|:---:|---|
| 61 | `moretypes/25` | A | — |
| 62 | `moretypes/26` | A | — |
| 63 | `moretypes/27` | A | — |
| 64 | `methods/1` | A | — |
| 65 | `methods/2` | A | — |
| 66 | `methods/3` | **B** | “你只能为接收者类型与方法定义在同一个包中的情况声明方法”虽然能够恢复原意，但中文句法明显生硬，把“接收者类型必须与方法定义在同一 package”这一条件表达成了“为……情况声明方法”。需仅重组该条件句，使同包约束自然、明确，不改变技术含义。 |
| 67 | `methods/4` | **B** | “尝试从第 16 行函数声明中删除 `*`；该函数就是 `Scale`，并观察……”的中文组织不自然，“该函数就是 `Scale`”割裂了原文中“第 16 行 `Scale` 函数声明”这一整体修饰关系。需明确删除对象是第 16 行 `Scale` 函数声明中的 `*`，保持后续观察程序行为的指令不变。 |
| 68 | `methods/5` | A | — |
| 69 | `methods/6` | A | — |
| 70 | `methods/7` | A | — |
| 71 | `methods/8` | A | — |
| 72 | `methods/9` | **B** | 首个定义句仍保留 `_interface_type_`；按 Present authority，该形式用户可见为强调的 “interface type”，不是受保护代码身份。当前 glossary 将 `interface type` mandatory 定为“接口类型”，因此这里存在一处明确的 untranslated/m mandatory terminology inconsistency。需按当前 glossary 处理该可翻译强调术语，并保持 emphasis 结构。 |
| 73 | `methods/10` | A | — |
| 74 | `methods/11` | A | — |
| 75 | `methods/12` | A | — |
| 76 | `methods/13` | A | — |
| 77 | `methods/14` | A | — |
| 78 | `methods/15` | **B** | 首个定义句仍保留 `_type_assertion_`；实际可见文本为强调的 “type assertion”，属于可翻译自然语言，而 glossary mandatory 要求“类型断言”。同一 Unit 后文已经使用“类型断言”，因此这里还造成 Unit 内部术语不一致。需按 glossary 统一该定义处术语，同时保持 emphasis 结构。 |
| 79 | `methods/16` | **B** | 首个定义句仍保留 `_type_switch_`；实际可见文本为强调的 “type switch”，而 glossary mandatory 明确要求“类型选择”。后文已使用“类型选择”，因此首处定义形成明确的 untranslated terminology inconsistency。需按 glossary 统一该定义处术语，并保持 emphasis 结构。 |
| 80 | `methods/17` | A | — |
| 81 | `methods/18` | A | — |
| 82 | `methods/19` | A | — |
| 83 | `methods/20` | A | — |
| 84 | `methods/21` | A | — |
| 85 | `methods/22` | A | — |
| 86 | `methods/23` | A | — |
| 87 | `methods/24` | A | — |
| 88 | `methods/25` | A | — |
| 89 | `methods/26` | A | — |
| 90 | `generics/1` | A | — |
| 91 | `generics/2` | A | — |
| 92 | `generics/3` | A | — |
| 93 | `concurrency/1` | A | — |
| 94 | `concurrency/2` | A | — |
| 95 | `concurrency/3` | A | — |
| 96 | `concurrency/4` | A | — |
| 97 | `concurrency/5` | A | — |
| 98 | `concurrency/6` | A | — |
| 99 | `concurrency/7` | A | — |
| 100 | `concurrency/8` | A | — |
| 101 | `concurrency/9` | A | — |
| 102 | `concurrency/10` | A | — |
| 103 | `concurrency/11` | A | — |

**汇总：A=38，B=5，C=0，D=0。**

本次严格只审核了 **Page stable indexes 61–103，43/43**。**未重审 indexes 1–60，也未审核 Example indexes 104–122**；没有修改任何既有 QC 结论，没有生成 replacement，没有修改译文，也没有执行 record、finalize、promotion、preview、publish 或 Production。

停止于本组独立 Reviewer 结论。
