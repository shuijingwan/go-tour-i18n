# Integrated Surface 定向性能记录（2026-10-18）

本轮仅执行 repository engineering，停止于 activation 之前；未修改语言工件、Reviewer evidence、Surface receipt 或 coverage。

## 实测结果与范围

相同 zh-CN successor 的原始 `surface-scope` CLI wall time 从维护者提供的 **444 秒**降至 **189.090 秒**（包含 `go run`）。输出 JSON 与优化前逐字节相同：

- scope identity：`bc856812ae5bb9d5c9f220ca2db2800f2dfb53f2fc9676383f7c3fe846371e37`
- carried：530；review_required：2。
- 待审资产：`_content/doc/security/fips140.md`、`_content/learn/books.yaml`。

带 CPU profiling 的同一 scope 重建从 457.748 秒降至 173.512 秒。主要阶段 wall time 如下；嵌套阶段不能直接相加。

| 阶段 | 优化前（秒） | 优化后（秒） |
| --- | ---: | ---: |
| currentInputs | 1.412 | 1.330 |
| prospective closures | 91.158 | 67.529 |
| Learn/Docs closure（含 Page finalization） | 81.154 | 57.169 |
| Learn/Docs 当前 Page finalization | 53.694 | 37.923 |
| Shell closure | 10.004 | 10.360 |
| Tour plan | 68.797 | 44.609 |
| Tour canonical QC 与 candidate binding | 60.602 | 33.131 |
| Learn/Docs structured carry set | 0.710 | 0.561 |
| Learn/Docs candidate binding | 222.792 | 11.665 |
| Shell candidate binding | 3.917 | 6.889 |
| 最终 dependency verification | 1.100 | 1.016 |

覆盖加载不足 1 毫秒，runtime references 不足 1 毫秒，最终 identity 约 6 毫秒。closureValidation 内 inventory catalog 共访问 8 次但只构建 1 次；Learn/Docs package catalog 访问 4 次，沿用该 inventory。两个 prospective closure 各验证 1 次；structured closure/generation 的现有 graph 节点也各构建 1 次。

## 原因与修复

原 CPU samples 中，machine reconciliation preparation 累计约 305 秒，replacement resolution 约 229 秒，其中完整 representation reconciliation 约 203 秒。逐 Unit candidate binding 各自准备整包 reconciliation，是最大重复来源。

修复仅增加两种 invocation-local reuse：

1. 批量 candidate 验证共享 native reconciliation preparation；每个 Unit 仍验证精确 source mapping、Snapshot membership 和 native machine proof。返回前重新检查引用 SHA、glossary、完整 Review 及 historical latest selection。
2. 同一 closureValidation 复用普通 Page resolver 的纯结果。key 包含 exact Reference 和完整 typed Catalog，保留 nil/empty 区别；私有副本隔离输入和输出。每次访问仍重新读取并验证 Reference，返回前仍执行 graph dependency guard。最多保留 8 个成功结果，超出后走原计算。

优化后两个 Learn/Docs representation reference 分别调用 6 次、20 次，各仅完整构建 1 次，其余复用。该复用不依赖通用 verified-byte cache 是否仍有容量。

Shell foundation reader 依赖当前 adoption/parser authority，继续执行原 reader。`page_replacement.go` 的原字节保持不变：该文件整体 SHA 属于 Shell adapter authority，修改其中辅助函数会正确触发 current authorization stale，不能用性能修复更新或绕过该 authority。

**绝对耗时尚未达到 60–120 秒目标。** 剩余时间集中在当前 Page/A-only 验证、历史完整性和两次 Tour canonical gate。进一步合并这些验证需要扩展完整 currentness dependency proof，不能只缓存返回的 PASS 或 finalization 指针。本轮低复杂度、低 correctness 风险范围内的高收益优化已完成，保留这些验证，不扩大缓存架构。

## 验证结果与未解决的失败

真实 fixture profiling、原始 CLI 及完整 scope 字节差分均 PASS。新增 candidate preparation reuse、typed resolution reuse（含容量耗尽、输出隔离和 nil/empty 区别）、既有 Reference/input guard、replacement fail-closed、Integrated Surface Reviewer Bundle/currentness 定向测试通过。

收尾选定的 9 项 unit tests 中，7 项通过；另两项既有 Tour checkpoint 测试在 fixture 初始化阶段失败，尚未执行 checkpoint 断言：

```text
TestTourCurrentCheckpointCarriesLegacyWithoutHistoricalBytes
tour_checkpoint_test.go:108: lstat /tmp/TestTourCurrentCheckpointCarriesLegacyWithoutHistoricalBytes2480073070/001/_content/doc: no such file or directory

TestTourCurrentCheckpointIndependentAuthoritiesFailClosed
tour_checkpoint_test.go:211: lstat /tmp/TestTourCurrentCheckpointIndependentAuthoritiesFailClosed708352027/001/_content/doc: no such file or directory
```

按本轮测试边界，仅保存该最小 failure evidence，未扩大调查或修改 fixture。未获得维护者的明确暂缓决定，因此未写入 Deferred Issues。测试集合未全绿，本轮不能作为完整验收 PASS。

## 精确复现与命令合同

以下命令只读重建 scope，stdout 是完整 `IntegratedSurfaceScope` JSON。重定向仅保存诊断副本，不记录 Surface decision，不激活 package。失败时保留错误并修复输入或工程问题；不重放 `surface-record`、不重写已有 receipt。

```sh
go run -mod=readonly ./cmd/tour-i18n \
  site-content surface-scope \
  --locale zh-CN \
  --generation data/site-workflow/zh-CN/learn-docs-v1/closure/67837cd680ff6109a5c77d2a948416ba950051b733f5a3e6c0349069211f4299.json \
  --generation data/site-workflow/zh-CN/site-v2-shell/closure/1f5e9f0fb28b2098c1fff90778f652f9541cfc6ff4d8c75e72ec2093f70dc926.json
```

显式 profiling test 先 before 后 after；after 要求保留 before 的 `/tmp` scope 文件：

```sh
GO_TOUR_INTEGRATED_SURFACE_PROFILE=before go test -mod=readonly ./internal/sitecontent -run '^TestIntegratedSurfacePerformanceReadonly$' -count=1 -v -timeout=15m
GO_TOUR_INTEGRATED_SURFACE_PROFILE=after go test -mod=readonly ./internal/sitecontent -run '^TestIntegratedSurfacePerformanceReadonly$' -count=1 -v -timeout=15m
```

测试对照上述已接受 receipt 的完整 scope，CPU profile、阶段计数和 JSON 差分副本仅写 `/tmp/go-tour-integrated-surface-*`。该 fixture 对应固定安全状态，后续正式状态变化时不得据此自动恢复或重新生成语言工件。
