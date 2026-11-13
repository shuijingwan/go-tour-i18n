# Generation Bundle export 的 immediate CURRENT 选项

本页是 `generation-bundle export --current-check` 的 canonical additive CLI 合同。
未带该选项的 export，以及 standalone `generation-bundle current-check`，继续遵循
[Generation Bundle transport](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)；
package selector、retry、approved selection、canonical encoding 和 identity 均不变。

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
  --content-package <registered-package> --locale <locale> --batch-id <existing-batch-id> \
  --current-check --output /tmp/<unused-generation-name>.zip
```

retry 仍只追加 `--unit-id <retry-unit-id>`；不改变 next attempt 或 exact output scope。
native/default Tour 可按原合同省略 selector；registered package 必须显式延续同一 selector。

- **OPERATION / INPUT_KIND**：对已存在的 initial/revision/retry batch 构造 canonical
  Generation ZIP，在同一 invocation 调用原 current verifier，再发布被验证的 exact bytes。
  `--current-check` 是 boolean option，默认 false；`--current-check=false` 等同于旧 export。
  `--current-check true` 的游离参数以及无效 boolean 值拒绝。其它 flags 沿用原合同。
- **PATH_CONTRACT**：`--batch-id` 不是 manifest path。`--output` 可绝对或相对当前工作目录，
  parent 必须已存在，目标必须不存在；普通文件、目录、symlink 或并发创建的目标均不覆盖。
- **MUTATION / PRIOR / CAS / IDEMPOTENCE**：repository mutation=none，只通过现有临时文件
  加 no-replace publication 发布一个外部 ZIP；没有 receipt、batch mutation 或 prior identity。
  同一输入生成相同 bytes，重复使用同一 output path 拒绝，已有目标保持原样。
- **CURRENTNESS**：export 和原 verifier 使用同一 command-owned Catalog、closureValidation
  与 invocation-scoped memoization。只复用已有 exact-input representation reconstruction，
  不省略 fresh dependency、package、source、validator、retry result/validation 或 tamper checks。
  CURRENT 对应本 invocation 最终验证边界及随后发布的 exact ZIP，不是长期 currentness receipt。
  后续时间点使用旧 ZIP 仍须 standalone current-check；import 的现有 currentness verification 不变。
- **SUCCESS / STDOUT_KIND**：exit 0；先输出原 `Generation bundle exported:` summary，随后输出
  原格式 `Generation bundle: CURRENT (locale=... batch=... task=... attempt=... identity=...)`。
  没有新增 JSON schema 或 ZIP member。publication 未完成前不输出 CURRENT。
- **FAILURE_STOP**：export、verification 或 publication 任一步失败即返回 error，不继续
  Generation/import/process。verification 失败不发布 ZIP。I/O/concurrent failure 后先只读检查
  output，已有 bytes 不删除、不覆盖、不盲重试；不自动 recovery。

该选项可替代“刚 export 后立即在另一个进程 current-check”的重复等待，不能替代后续
currentness、restore/validation、独立 QC 或其它 gate。standalone 的性能基线仍使用独立
invocation；combined 测量使用一个新 invocation，不能把上一次的 PASS 或 Catalog 带入。

这是 modified formal CLI；首次真实使用前由 Maintainer Local 按
[Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3)
核对本页、parser/dispatch/verifier/publisher 和 direct CLI tests。工程验收不授权新的正式 export。
