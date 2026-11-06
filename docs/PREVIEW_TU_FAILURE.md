# Preview 发现 TU 缺陷的 machine reopen authority

本合同与 locale、content package、milestone 无关。它补齐独立 Integrated Surface PASS 后，正式 preview acceptance 发现可机械证明 TU 缺陷的授权入口；复用 `quality-check surface-reopen` 和既有 revision lifecycle，不创建第二套语言审核。旧 Surface PASS、Reviewer evidence、QC A、finalization 与 authorization 均 immutable。

## 支持范围与拒绝边界

当前仅支持 `missing-source-bound-link`：一个 registered Page 的 canonical source 实际渲染出 external HTTP(S) anchor，exact finalized target 本身没有该 anchor，现场 preview 也缺失，并且现场 HTTP 200 response 与当前完整 renderer/composition 的该 candidate 输出逐字节一致。source/target 使用 Markdown renderer 或 HTML parser 的实际 anchor 语义，闲置 reference definition、code span 不产生 expectation。reference、inline Markdown 和 HTML href 使用相同检查。href 必须在 coverage resolver 中保持原样；templated Page、其他 representation/check 或无法机械确定的语言问题 fail closed，不能填自由文本越过检查。

任意 locale 和 registered content package 的受支持 Page 都走同一入口。source path + kind + source SHA 必须唯一映射为 Snapshot Unit；route 仅用于核对 descriptor 和观察现场，不能选取近似 Unit 或 latest candidate。UI、navigation、runtime、coverage、renderer 等 failure 若 target TU 的链接仍完整，则不能得到此授权。未来 representation/check 必须增加其可重算 proof reader 和 focused tests，不能凭错误文本声明为 TU defect。

## Evidence 与 currentness

旧 verifier 只打印 `BrowserFailure`，没有可消费的结构化 immutable result；旧 Surface receipt 只能表达原独立 Reviewer 结论。因此增加独立 `go-learning/preview-tu-failure/v1` receipt，路径为 `data/preview-tu-failures/<locale>/<failure-id>.json`。它由 Go recorder 对单 route 重现同一 machine check 后原子、不可覆盖地保存，不导入聊天里的诊断文本，不重新运行完整 browser acceptance。

receipt 绑定 locale、package identity、route、exact finalized Snapshot/manifest/finalization、完整 selected source/candidate/validation/attempt identity、verifier/renderer/module contract paths + hashes、loopback preview URL、HTTP 200 的完整 response + SHA、精确 missing href set、自身 identity，以及当前 completed content/coverage/production-origin/resources 的重建 context hash。完整 source/target evidence 继续由既有 Snapshot reader 校验。合同中没有新增语言 rating。

现有 preview 没有正式 server build receipt/endpoint；这里的 context hash 是当前 repository 重建身份，不冒充 server build attestation。现场 response 必须 exact 等于当前 reconstruction。若已有 preview 是旧 build、response 不同、source/candidate/validation 漂移、package/coverage/contract/resource/context 变化，或原 preview 已停止，record / reopen / 首次 export 均 fail closed。不得将旧终端 FAILED 文本直接当授权。

## 唯一执行顺序

维护者 Local terminal 先对当前仍运行的 preview 记录机器证据：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check preview-failure \
  --content-package <package> --locale <locale> \
  --previous-snapshot-id <finalized-snapshot> --unit-id <unit-id> \
  --route <exact-route> --preview-url <loopback-origin/> --failure-id <new-id>
```

然后使用新 reopen ID 消费该 receipt，不能同时提供 `--finding`、`--surface-review-id`、`--integrated-surface` 或 historical replacement flags：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check surface-reopen \
  --content-package <package> --locale <locale> --reopen-id <new-id> \
  --previous-snapshot-id <finalized-snapshot> \
  --preview-failure data/preview-tu-failures/<locale>/<failure-id>.json \
  --unit-id <exact-unit-id>

go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --content-package <package> --locale <locale> --generator <provider> \
  --allow-reexport --previous-snapshot-id <finalized-snapshot> \
  --surface-reopen-id <new-id> --id <exact-unit-id>
```

本入口每份 receipt 精确授权一个 TU。reopen schema v1 additive 保存 `preview_failure` path/SHA，其他 Surface authority 字段为空；反馈来源是 `preview_acceptance`，不是 `surface_review`。`previous_rating=A` 仅描述 predecessor 原有判断；机器 finding 由 exact check/route/source/href 生成，不属于 Reviewer finding，也不产生新的 A/B/C/D。

记录、reopen 和首次 export 重算当前性；export 后原 receipt 是 invocation 的冻结输入，Generation Bundle 携带 reopen 与完整 machine receipt。revision / retry 复用已有冻结 evidence/currentness 机制，不要求旧 preview 永久存活，也不把 mutable coverage 永久绑定到历史 finalization。原 source/candidate/validation/Snapshot/receipt 仍须 exact；不允许借冻结语义发起新的 stale reopen/export。

之后仍必须执行 exact revision → validation → successor Snapshot → independent re-QC → A-only finalization → package closure → new independent Integrated Surface Review → activation → fresh preview acceptance。机器授权只允许进入 revision，不能替代任何后续 gate。

## 历史兼容

既有 FAILED Integrated Surface exact finding 路径及 historical direct Surface Markdown 路径不变。`--historical-documents` 只与 `--previous-reopen-id`、`--integrated-surface` 联用，用于从已有合法 reopen 经 frozen representation reconciliation 建立 successor UnitID 授权；它不创建新的 finding，不支持 preview failure，也不是绕过原 PASS finding=0 的入口。详见 [TranslationUnit replacement](TRANSLATION_UNIT_REPLACEMENT.md)。
