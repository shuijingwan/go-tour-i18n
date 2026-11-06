# 确定性派生缓存

BeginMemoization 保持原来的 invocation-local L1，并为 Transform 增加可删除的 L2。
默认位置是 os.UserCacheDir()/go-tour-i18n/deterministic-v1。不新增 CLI flag，
不将缓存写入 data/、locales/、receipt 或 Bundle，也不改变正式输入及恢复合同。

## 身份与边界

缓存 key 包括内部版本、当前 executable 的完整 SHA-256、versioned operation 和
调用者提供的完整语义输入 bytes 的 SHA-256。executable 身份绑定编译后的 parser、
schema、adapter 与 validator 实现；operation 的输入必须另外覆盖该派生计算的全部数据
和选项。不能把 path、Unit ID、mtime、locale 或上次通过状态当作输入身份。
通过 /proc/self/exe 读取正在运行的 image，避免并发替换 executable 路径时误用新版本身份；
无法取得这个 exact image handle 的平台停用 L2，保留 L1 和原始计算。

当前共享派生层包括：

- 严格 JSON 的完整解析值：仅零值目标、没有 custom decoder、至少 4 KiB，且 JSON
  round-trip 完全保持类型值（包括 nil/empty 差别）时准入。
- Generation archive 的确定性 member index：key 已绑定刚刚读取的 archive SHA、大小
  和 decoder 限制；consumer 继续检查 import/provider/model/attempt、inventory 和原始输出。
- 已有 source-owned replacement representation reconciliation：输入包含完整历史
  Document 与当前 PageAdapter 数据，原有 round-trip 准入规则不变。
- Tour source Catalog 的完整数据图：每次调用 fresh 捕获 source tree、文章顺序、
  UPSTREAM_MANIFEST 和 parser 选项，核对当前 source/hash 后，复用对应的
  Pages/Conditional/Examples 派生结果。派生期间只读 owner 捕获的 bytes；返回前
  再次观察完整输入。预算超限、不支持的文件类型/路径、观察变化或 codec 不可用
  均回退原始 reader，不新增正式拒绝条件。这个组件不包含注册 package、QC、
  historical lineage 或 current authorization 的接受结论。
- Global scope 的完整基础图：fresh Tour routes、shell/Tour 完整 source inventory
  和 source SHA，加上 exact frozen snapshot SHA 与 inventory 选项，决定完整
  routes、source/dependency graph、package identities。返回前再次观察 live inputs。
  LoadCurrent 和 historical reader 仍独立比较声明的 Global、adoption、membership
  和 current pointers；这里不保存它们的接受判定。
- 历史 Page package 投影：完整 frozen Documents、当前 source-owned PageAdapter
  和 ordered descriptor 数据决定 historical descriptor 与 replacement bindings。
  只有 lossless 数据输入/输出准入，runtime readers/root、其他 package 与 QC finding
  均不进入派生结果。Shell foundation 等仍依赖 mutable authority 的路径继续 fresh
  重建，不通过这个组件绕过。冻结 reference 的读取/hash 与调用方的 Snapshot、QC、
  candidate、validation、glossary/currentness 检查继续执行。
- 整包基础 Document/protection graph：exact snapshot、Source inventory、package
  descriptor 与 fresh raw source → 完整 Documents/raw map。`CurrentPackageDocuments`
  统一消费这层；membership 的有序 predecessor 解析仍 fresh，adoption 与 structured
  successor 的原验证仍在外层。mutable authority 不进入结果。workflow generation/review
  的旧入口经 `sharedWorkflowDocuments` 复用该图及 formal Page projection；冻结身份的
  Page 注册/shell successor 仍走原 reader，不能声称所有 Page consumer 都已覆盖。
- Ordinary bridge 的 source-owned representation：完整 Page、transport mapping、raw、
  reviewed candidate 决定独立 ordinary template/ancestor。只准入 lossless codec；每次
  使用仍 exact restore，对未知 adapter 或不可用结果执行 canonical factory。不是
  Surface/QC authority、runtime decoder 或 bridge PASS 的缓存。当前 readonly bridge
  construction 另有单 phase source-proof owner，返回前关闭完整 freshness guard；record
  使用 eager 观察边界，保留两次 fresh construction 和最终 publication checks。
- QC/Surface historical structured parser/protection graph：exact frozen Document、
  raw source 和已恢复 target bytes 决定完整 source/target Documents。frozen source/坐标
  与 protected targets 的验证在缓存外执行，graph 的 structure/count 接受比较也在缓存外。
  不将完整历史 QC/Surface verifier、Review、current scope 或 native adapter 的 PASS 存入 L2。

不缓存 authorization、currentness、QC carry 判断、record eligibility 或 PASS。
缓存里的 QC/manifest 解析值是正式 bytes 的数据表示，不是它们通过了当前 authority
检查的结论。source/corpus 的 fresh observation 与 authority 比较、scope、membership、glossary Review、
expected SHA、历史 lineage、最终 guards 和 record 的第二次 fresh construction 继续执行。

## 缓存故障

L2 只接受私有目录与普通文件；使用 os.Root 限制路径访问。每个派生结果至多
16 MiB，小型严格元数据头绑定 key、implementation、长度和完整 payload SHA-256。
读取失败、未知/旧版本、checksum 错误、截断、权限问题均视为 miss，调用原始计算。
无法发现 executable/cache 目录时也回退。错误计算结果不缓存。

写入使用私有随机临时文件、完整写入与 fsync、atomic rename，并尝试同步目录。
所有缓存写入错误均被忽略；并发 writer 可以重复计算同一个 key。未发布的临时文件
不会被读取。L1 仍有原有内存预算；缓存不保存对调用者可变 buffer 的引用。
磁盘写入还有 best-effort 准入预算（256 MiB / 512 entries），不引入 LRU 或跨命令锁；
并发写入可以重叠，满额或无法检查时直接放弃写入。

可以随时删除整个缓存目录。删除、不可写或损坏只会使计算变慢，不能提供或撤销正式
authority。缓存完整性校验不是正式 evidence 或签名，也不代替正式 source/hash 检查。

## 定向验证

TestPersistent* 覆盖 cold/warm、不同进程、删除、损坏、版本/实现不符、不可写、
并发写入、错误不缓存及 typed JSON 等价。TestRevisionBundlePersistentCacheFallback
使用隔离 revision fixture 验证 Bundle export/current-check 的 byte identity 和故障回退，
并验证 warm cache 后的 Snapshot/QC/candidate/validation/glossary drift 仍被拒绝。
Surface→ordinary bridge 与 current-validation bridge fixtures 也启用相同缓存层。
TestSourceAuthorityWorkflowProcesses 在独立进程中验证完整 Tour source Catalog
附加 registered fixture package 后的 revision export、Bundle export/current-check：
source 派生 cold 为一次，warm 为零；其余 authority checks 继续独立执行。
这类小型 fixture 的 timing 不能作为真实 current/historical authority graph 的性能达标证据。
TestGlobalAuthorityEnvironmentCrossProcess 和 TestHistoricalPageEnvironmentCrossProcess
分别验证两个完整数据组件 cold=1 / warm=0，TestSharedHeavyEnvironmentOwnership
防止 consumer 绕过对应的 observation/derivation owner。
TestPackageRepresentationEnvironmentCrossProcess 验证整包 graph cold=1 / warm=0，
并与原 canonical reader 的完整 Document/raw identity 对比；cache codec 不可用时重算。
TestWorkflowDocumentEnvironmentCrossProcess 与 TestBridgeRepresentationEnvironmentCrossProcess
验证新增数据节点 cold=1 / warm=0；current-source owner 的计数只表示同 phase 的 source
子图复用，不能当作整个 QC evaluator cold/warm 计数。完整 QC/current authorization、
same-unit/Surface bridge 的完整 proof 不进入 L2。TestQCSurfaceHistoryDerivationCrossProcess
验证历史 structured 数据图 cold=1 / warm=0、完整 graph 与输出等价；新增 invalidation/
codec 回退测试验证缓存不会替代 frozen/protected/structure 检查。当前必需重算的边界见
`docs/SHARED_HEAVY_ENVIRONMENT_AUDIT.md`。

TestSharedWorkflowEnvironmentPerformance 是显式选择 /tmp disposable root 的工程探针，
不新增正式命令。包含 setup 的 wall time <=540s 为目标，540–600s 为低余量通过，
>600s 为性能门禁失败。小型 synthetic fixture 通过不能代替 current package 的计时。
