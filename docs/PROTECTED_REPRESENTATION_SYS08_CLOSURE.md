# SYS-08 protected representation 工程收口

当前工程结果 **PASS**：source-spelling false-machine 从16降至0，systemic defect 从1降至0，validator gap 保持0。沿用既有 taxonomy 和153 Page deterministic audit；没有重新发现或设计 SYS-01～07，没有重新处理 structured evidence drift。本报告不是正式 reconciliation、语言审核或新语言结论。

16个 exact source contexts（11页）全部在实际 registered corpus 中按原 source coordinates 修改成功。机器不决定这些文本应保持英文还是如何翻译；`可编辑文本` 和 `CMVP 验证证书` 仅是测试输入。完整 old contexts 保存在测试 fixture 和 audit 的 `source_spelling_false_machine_before_sys08`，当前冻结集合为空。

## 共享 classification policy

原 `uppercase / dot / slash / underscore / camel` → machine 的分支已从新 Page export 移除。顺序是 parser 明确结构/code/target → technical namespace/身份 authority →明确技术位置 →普通自然语言；只有明确 identity position 中的不确定技术 literal 才 fail closed。相邻上下文有长度上限；target-side structural parse 不执行该分类。

| Role | authority 和边界 | 验证 |
| --- | --- | --- |
| NATURAL_LANGUAGE | 没有 machine authority 的 prose，包括缩写、heading/discourse marker；不是英文例外白名单 | 原16处、7类正例，以及 OTHER、SomeHeading、a.b、x_y、time/space、file/package、key/value、斜杠、GET、PATH、-ing、普通数字 |
| TECHNICAL_IDENTIFIER | Go AST teaching identifiers；冻结标准库 namespace；技术位置中的 API/enum/environment identifier | identifier/package/API、state RUNNING、PUBLIC track、CUSTOM_ROOT、Go release、Certificate A1234 |
| CODE_OR_LITERAL | code span/opaque HTML/code framework；CLI option grammar；version grammar；明确 HTTP 位置的 method/status | Go keyword/string/command code、长短 flag、v1.2.3、HTTP GET/404；mutation 拒绝 |
| STRUCTURAL_TOKEN | 既有 parser 的 tag/attribute、delimiter、anchor、template、newlines 和 pair stack | 未放宽 token exact/unique、slot membership 或 pair/binding |
| URL_OR_TARGET | 原 link/HTML target binding；明确 URL；public suffix authority；路径/文件格式语法 | URL、域名、filename、path；URL/token mutation 拒绝 |
| KNOWN_TECHNICAL_ATOM | 共享术语规范§6的技术 identity；Go environment、Unicode/IETF、NIST programs/algorithm、CVE/CVSS/OSV namespace 等正向 authority | CLI/API/CPU/URL、FIPS/CVE、GO vulnerability ID 等仍保护；不根据大小写推导新成员 |
| AMBIGUOUS_FAIL_CLOSED | 已明确处于 identity position、但不能确认完整技术 literal 的安全 fallback | `function strange()=` 保留，普通缩写不进入该 fallback |

`semantic_go_packages.go` 冻结 Go 1.26 public standard-library index 的179个 package namespace facts，来源是本机 `go list std`；正常 workflow 不运行该命令或查询 GOROOT。不是全 Go API/keyword glossary keep 表。无 qualifier/delimiter 的少量真实 API 引用，通过 `binary.ReadVarint`、`binary.ReadUvarint`、`(*sql.Tx).Stmt`、`(*zip.Reader).Open` 的**编译器 symbol binding**导出 identity，而不是通过英文拼写猜测。普通英语 `Read`、`Open` 不因方法名而全局保护。其他品牌/教学概念仍遵守既有 glossary 与语言审核职责。

三个 registered families 共用其实际 adapter：Tour 的结构/Go identifier 路径未变；Learn/Docs 使用新 source role policy；Homepage 的 simple slots 和完整 inline runs 使用同一 policy。Homepage mixed run 中的 CLI 等 atom 也保留。candidate 校验先按 source coordinates 区分 structural/hidden 与 prose-machine inventory，再复用 frozen raw binder；不对中文/英文 target spelling 重新分类。hidden icon 的 `arrow_forward` 必须保留 enclosing DOM role，不能逐个 raw atom 脱离上下文判断；这一实际 regression 已由原 Homepage fixture 捕获并修复。P0 这样的合法 state identity 不会误匹配内部 opaque marker。完整 inline run 的语序、标点、nested/repeated/link/code binding 回归通过；没有重写 SYS-06 abstraction。

## corpus、impact 与剩余 formal unknown

实际 reader 与 identical-source projection 都为153页，按 `(source_path, route)` 对照的 Unit ID/SourceSHA exact-set 完全一致；3种表示未扩容。16/16 whole-Page mutation PASS。47个 soft-wrap Page 仍是 **translation-only-risk**，validator 不判断语言自然度。不存在剩余 SYS-08 representation blocker。

本轮39个 input/Unit identity 变化，与原39个 exposure 合并为**45个 unique Units**，不是78个，也不是45个语言 defect。所有 exact IDs/routes、old/current/Snapshot IDs 和 source SHA 位于 [FORMAL_IMPACT](PROTECTED_REPRESENTATION_FORMAL_IMPACT.json)。multi-label counts 可重叠：

- representation_reconciliation_required=45：这些旧 formal identities 下次正式消费新表示时必须显式 mapping；不授权 bulk reopen、carry A/B 或自动 Generation。
- reattest_only=0；unaffected=0：该45个集合相对其冻结正式表示确实存在 input/identity drift，不能仅改 validator receipt 当作同一个 Unit。
- true_language_defect=1：只有既有 fips140 qc-014 B finding；没有工程创建的新语言 finding。
- unknown=3：已实际观察到的 current predecessor baseline 复用尚未证明；不是语言 rating，也不是把新表示本身误判为 defect。未扩大为全部 formal migration readiness 审计。

三处 unknown：

| Source / current Unit | 明确 blocker |
| --- | --- |
| `_content/doc/modules/managing-dependencies.md` / `82a1f9b16b20d60a538c1873daf9afb5c446aded1d2bff16e3b5650ed6bafb1e` | 新 protected HTTP `404` 在旧 raw 中是 prose |
| `_content/doc/security/vuln/database.md` / `a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f` | 新 protected package identities `A/B/A` 在旧 raw 中是 prose |
| `_content/doc/tutorial/json.md` / `b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c` | 新 protected API `Marshal` 在旧 raw 中是 prose |

现有 rebaser 对它们返回 `replacement introduces protection outside historical source atoms`；没有从译文猜 token mapping、放宽 gate 或自动生成新 raw。要消费这3个 successor baseline，必须先取得明确的 migration proof/合法恢复路径；本轮没有解决或假称可执行该 formal recovery。Homepage 的机器 impact 已依据 exact source/input drift 归为 reconciliation required；没有声称其 accepted candidate 的复用已验证，也没有把候选语言质量作 unknown/A/B 判断。

## 兼容与 fips140 /046 handoff

旧 protected response 恢复只消费 persisted inventory，新 classifier 不重算它。旧 `ExtractShellLanguageSource` / structured foundations 明确保留 legacy source policy；新 Page reader 选择 semantic policy。无 schema/PB 新版本，无 raw/candidate/Snapshot bytes 重解释。prior 49 raw/candidate、38 byte-exact rebase PASS 作为既有 authority 保留，没有重复全历史扫描。本轮只读取11个 affected selections：旧 raw、candidate SHA、validation、Snapshot identity 全部 PASS；8个 current byte-exact rebase PASS，3个明确 fail closed 如上。旧 source-slot fixture 仍拒绝删除旧 locked token。

fips140 四项均 PASS：reference binding、editable label、整页 representation、canonical URL exact unchanged。fixture 输出 `[CMVP 验证证书][]` 及匹配 definition。`NOTE`、`U.S` 也能合法修改。机器逻辑不含该中文译法。

当前 fips140 successor ID：`8e2b3ec2b00595cb7e8ea8587ea6002107a0e616c18f49fc46562702af10f1c3`。真实 qc-014 selected raw → current baseline → restore 的 reviewed bytes 精确相等。现有 QC-B reconciliation bridge 的 synthetic 正负回归 PASS，包括错 rating/Snapshot/Unit/finding/混入 Surface authority 拒绝。ordinary initial、B/C/D、Preview revision、retry 的既有定向测试 PASS；未重写 bridge。

只读工程 assembly 保存于 [preflight](protected-representation-evidence/qc014-readonly-preflight.json)，绑定真实 Snapshot/results SHA、selected candidate/validation、完整 frozen representation package 与单 Unit necessity mapping。49个 source Documents 是已有冻结 fixture 的工程副本；不是扫描49份历史 language outputs。没有调用 Record/apply/export。preflight PASS 只覆盖 mapping 与 byte-exact baseline；正式 CLI 仍须重新执行 complete-QC/currentness/unconsumed-output gates。

046 manifest SHA 保持 `a7d005c29e630e58fdbdc0c7d6de9f0e2f5c2c33f1f4e39b51cb4b4a2143df95`，input 未改、0 output，无 import/process。唯一下一步是 Coordinator 将 [exact canonical command](protected-representation-evidence/qc014-next-canonical-command.sh) 交 Maintainer Local terminal，仅为 fips140 建立 QC-B representation reconciliation。该脚本只包装原 CLI 并携带原 exact finding；本轮只做 `bash -n`，**没有执行**。

```sh
bash docs/protected-representation-evidence/qc014-next-canonical-command.sh
```

后续链保持：immutable046 → representation reconciliation → successor representation → new revision batch → **同一 qc-014 B finding** → Generation → validation → independent re-QC。该命令不 export revision、不处理另外44个 Units，也不改变 qc-014。当前停点是 **before-formal-representation-reconciliation**。

## 定向测试与 timing

所有下列命令 PASS；没有 `go test ./...`、全 package suite、browser 或真实 formal lifecycle。synthetic bridge 只写 `t.TempDir()`。corpus reader 8.03s、projection4.30s、真实 QC mapping/predecessor probe16.19s；最后两组命令有重叠，不能作为 isolated benchmark。此前串行 post-policy corpus reader6.23s、projection2.95s；最终这两种 inventory exact 相同。11 selection 兼容测试2.19s；initial/revision/Preview/retry0.784s。没有分钟级 classification scan。没有重做全面性能审计。

```sh
GOCACHE=/tmp/go-tour-sys08-gocache \
GO_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-sys08-current.json \
GO_TOUR_SYS08_HISTORICAL_PROBE=/tmp/go-tour-sys08-historical-probe.json \
go test -mod=readonly ./internal/sitecontent \
  -run '^(TestRegisteredPageRepresentationAudit|TestSemanticRoleNaturalLanguage|TestSemanticRoleMachineControls|TestSemanticRoleShellPage|TestSemanticRoleHistoricalSelections|TestHomepageInlineRepresentation|TestReconstructSourceProtectionStrict|TestReconstructTargetLexicalAtoms|TestRepresentationKnownBlockers|TestStructuralBindingFrozenFIPS140|TestStructuralBindingAdapterAuthorityDependency|TestQualityCheckRepresentationReconciliation)$' -count=1 -v

GOCACHE=/tmp/go-tour-sys08-gocache \
GO_TOUR_SYS08_QC_PREFLIGHT=docs/protected-representation-evidence \
GO_TOUR_REPRESENTATION_BEFORE=/tmp/go-tour-representation-before.json \
GO_TOUR_REPRESENTATION_AUDIT=/tmp/go-tour-sys08-projection.json \
go test -mod=readonly ./internal/sitecontent \
  -run '^(TestSemanticRoleQCReconciliationPreflight|TestRegisteredPageRepresentationAudit)$' -count=1 -v

GOCACHE=/tmp/go-tour-sys08-gocache go test -mod=readonly ./internal/sitecontent \
  -run '^TestSemanticRoleShellPage$' -count=1 -v

GOCACHE=/tmp/go-tour-sys08-gocache go test -mod=readonly ./internal/i18n \
  -run '^(TestInitialGenerationBundleDoesNotRequireCandidates|TestRevisionGenerationBundleOrdinaryRatings|TestPreviewRevisionGenerationBundleInfersFrozenPackageScope|TestRevisionGenerationBundleProtectedOutputAndRetry)$' -count=1 -v

bash -n docs/protected-representation-evidence/qc014-next-canonical-command.sh
git diff --check
```

日志：`/tmp/go-tour-sys08-final-tests.log`、`/tmp/go-tour-sys08-projection-and-preflight.log`、`/tmp/go-tour-sys08-shell-test.log`、`/tmp/go-tour-sys08-lifecycle-tests.log`。最初一次 native audit 被 Go cache read-only sandbox 拒绝，原命令获准重跑；标准库 index 读取也在沙箱不完整后获准重跑。后续使用 `/tmp` cache。这些 setup failures 未被计为代码 PASS；修复中的真实负例/hidden-icon regression 均已在最终代码下关闭。

## 本轮 exact files

共23个本轮文件。其它既有 WIP 未整理；完整 worktree diff/status 不是本轮独占变化。没有 commit、reset、clean、stash、checkout 或 add。

```text
internal/sitecontent/linguistic.go
internal/sitecontent/semantic_role.go
internal/sitecontent/semantic_go_packages.go
internal/sitecontent/semantic_role_test.go
internal/sitecontent/semantic_reconciliation_probe_test.go
internal/sitecontent/testdata/semantic-role-natural-language.json
internal/sitecontent/shell_language_source.go
internal/sitecontent/shell_inline.go
internal/sitecontent/teaching.go
internal/sitecontent/translation_adapter.go
internal/sitecontent/shell_reconstruction.go
internal/sitecontent/structural_binding_test.go
internal/sitecontent/representation_audit_test.go
docs/PROTECTED_REPRESENTATION_MATRIX.md
docs/PROTECTED_REPRESENTATION_CORPUS_AUDIT.json
docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json
docs/PROTECTED_REPRESENTATION_AUDIT.md
docs/PROTECTED_REPRESENTATION_CLOSURE.md
docs/PROTECTED_REPRESENTATION_SYS08_CLOSURE.md
docs/protected-representation-evidence/qc014-historical-representations.json
docs/protected-representation-evidence/qc014-representation-necessity.json
docs/protected-representation-evidence/qc014-readonly-preflight.json
docs/protected-representation-evidence/qc014-next-canonical-command.sh
```

最终 `git diff --check=PASS`。完整既有 worktree 的 tracked stat 为 `139 files changed, 72302 insertions(+), 16156 deletions(-)`；status 为534行（含既有 untracked WIP），不能当作本轮独占统计。完整输出保存于 `/tmp/go-tour-sys08-diff-stat.txt`、`/tmp/go-tour-sys08-status.txt`。本轮 exact file set 仍为上列23个。`FORMAL_ARTIFACT_MUTATION=none`，`CODE_MUTATION=known`。
