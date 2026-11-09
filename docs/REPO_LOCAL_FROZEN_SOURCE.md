# Repo-local frozen source authority

本合同向既有 Site v2 source contract 增加 storage migration，不修改语言生命周期、coverage、历史 evidence 或 package source identity。当前 frozen upstream 仍为 `golang/website@db076098077c07d3cef1b85a2cf56ff52777f587`，不执行 V2-C1。

`_content/**` 保存项目实际采用的 runtime source；`data/site-source-manifest.json` 是 deterministic provenance authority。`locales/<locale>/**` 继续保存翻译 overlay。upstream 文件存在不代表注册 TranslationUnit、完成 package、local coverage、canonical 或 sitemap。现有 Tour renderer、project-owned 文件及11个明确的已提交 runtime override原样保留，不能用 upstream copy 覆盖。

manifest逐文件记录 path、size、SHA-256、Git mode、local mode、role、upstream exact / project override / project-owned分类。override同时保存 original upstream SHA/blob与本项目 commit/blob provenance。binary按原始 bytes校验；不改LF/EOF、不转码。strict JSON、固定排序、canonical encoding/self identity、重复路径、traversal、symlink和未知角色均 fail closed。

## Repository-reproducible file mode

持久化 mode 使用 regular non-executable / executable 的 repository 安全类别，不把 Git 无法保存的 owner/group write observation 当作 source semantic identity。解释规则只投影 owner/group write bits：`0444/0464/0644/0664 → 0644`，`0555/0575/0755/0775 → 0755`。其它 read/execute pattern、world-write、setuid/setgid/sticky、symlink 和 non-regular path 均拒绝；不得简单 mask 掉全部 permission bits。read/execute bits 的改变仍 fail closed，non-executable authority 不能接受 0755。移除 read/execute 权限的 umask 不在兼容范围内。

新 manifest 的 `local_mode` 写 canonical class，已有文件不 chmod；`git_mode` 仍保留其原 Git provenance，不能拿 upstream Git mode 覆盖明确的本地 executable authority。旧 manifest/receipt 的 raw `local_mode`（例如 436/0664）、原 JSON bytes、self identity 和物理 SHA 原样验证及保留，仅在比较 filesystem currentness 时投影为相同类别。path、presence、size、SHA 和 regular/no-symlink 检查不变；这不是语言、QC 或 Production authority。

同一规则适用于 Shell/shared representation 的 `mode`，由 `internal/sourcestorage/manifest.go` 的统一解释器实现。新 representation inputs 写 canonical mode；旧 envelope 原始 identity 先验证，再对 typed mode 字段比较，不能归一化旧 JSON 或忽略 SHA。invocation-local mutation guard 继续捕获实际 mode，执行中 chmod 即使类别不变也停止；跨 checkout 的 owner/group write 变化不构成 semantic drift。解释器和合同实现的 SHA 仍由现有 dependency re-attestation boundary 绑定；修改后必须重新完成 first-real-use verification，并由维护者另行授权现有 family 的 successor publication，不自动沿用旧 checker PASS。

10个目录外 exact frozen renderer/route contract位于 `data/site-source-contracts/`，仅作 source contract reference，不编译或启动 upstream server，不接管 `/dl`、Playground、deployment或未知 routes。manifest声明的 upstream path不变，通过显式mapping找到本地文件。

## 离线迁移与检查

```sh
go run -mod=readonly ./cmd/tour-i18n site-content source-migrate --website-root "$HOME/code/go-website-upstream"
go run -mod=readonly ./cmd/tour-i18n site-content source-migrate --website-root "$HOME/code/go-website-upstream" --apply
go run -mod=readonly ./cmd/tour-i18n site-content source-check
go run -mod=readonly ./cmd/tour-i18n site-content source-retirement-check
```

迁移只读取 exact commit Git objects，先检查全部碰撞。same bytes安全保留；明确override必须与已提交HEAD bytes相同；未知差异停止。apply仅exclusive安装缺失文件，最后发布manifest；已有文件不覆盖。pending migration marker阻止current source reader。失败rollback仅移除本次新文件；无法完整rollback或进程中断则保留marker并报告mutation=unknown，不盲目重放。

current-check、PackageDocuments与RedirectContracts使用repo-local manifest及实际采用的source，不依赖仓库外checkout。完整adoption checker检查全部文件；package freshness只消费registered closure，不绑定整个mirror或manifestaggregate identity。无关mirror bytes漂移由完整integrity checker报告，不自动撤销未消费它们的package语言成果。

旧80-file inventory、直接dependency和3个contract组成原100-entry registered subset。manifest绑定其原path集合与原ZIPdigest；reader从本地exact bytes在内存重建原deterministic encoding，并要求SHA完全一致。这是storage equivalence proof，不是新source或Reviewer PASS；原global JSON、package identities、65 locale sparse completion和历史receipts均保持byte-exact。旧fixture未迁移时保留原reader；已采用contract却缺manifest不能fallback。

## ZIP退休边界

current与archived glossary corpus的`source_files`继续保留原`data/site-content-sources.zip` path/SHA，不能改写历史bytes。新增`go-learning/source-storage-compatibility/v1`只证明physical storage承接，不证明语言质量、coverage、completion或Reviewer PASS。

proof位于`data/source-storage-compatibility/<old-zip-sha256>/<proof-identity>.json`。它绑定原ZIP path/SHA、实际消费它的current/history corpus文件path/SHA及semantic identity、每个corpus的exact required entry set、逐entry old/new path/SHA/role、current manifest文件SHA及self identity和`byte-exact`结果。entry集合严格等于corpus requirements的union；不消费的其他ZIP entries不成为glossary dependency。路径只接受直接、无歧义的registered upstream-exact Page/data映射。严格JSON、canonical encoding、self-address、排序、重复、traversal、symlink、未知role及任何不等字节均fail closed。

`site-content source-storage-compatibility [--apply]`只从仍存在且SHA完全一致的旧ZIP建立proof；不从current source自行声称历史等价。相同输入产生相同proof，apply只写immutable新artifact。机器storage reader位于`internal/sourcestorage`，供两个corpus consumer共享。

ZIP仍存在时，原physical SHA gate优先；所有已有proof还必须与旧ZIP中的逐entry bytes一致。旧manifest proof不能授权退休。ZIP不存在时，只有exact匹配实际corpus path/SHA、semantic identity及完整entry set的strict current proof才能解析；缺proof、manifest stale、bytes改变或多个匹配都拒绝。禁止按同名文件fallback。archived corpus验证自己的binding及原有source refs，不依赖later current corpus文件bytes。

`source-retirement-check`检查完整repo-local adoption、原registered subset/global identity、所有current/history corpus的current proof和current projection；未覆盖返回`ZIP_RETIREMENT_BLOCKED`。它不删除ZIP。真正删除前必须先通过隔离ZIP-absent测试：current/history loader、逐context/contributor identity、Unified Glossary Review、package/global identity及65 sparse completion均不变；删除后再运行同一focused verification。proof不会触发glossary refresh或新的Review，corpus的semantic identity算法不变。其他source references仍使用原physical hash gate。

退休资格检查与退休后的 representation currentness 分开：checker 使用既有 source-level global 重建及 current source adoption/membership 校验，corpus projection 也依赖 source authority，不先套用 downstream Shell representation envelope。因此，即使 Shell historical base 冻结 ZIP `absent=true`，仍可在 physical ZIP 存在时检查其 exact SHA、完整 required entry set、current manifest/self identity 和 current/history corpus proof；ZIP 不存在时仍须同一 exact proof。corrupt ZIP 不得被 proof 替代，missing/stale/ambiguous proof、source/corpus drift 均 fail closed。普通 `LoadCurrent` 与 Shell currentness 继续严格验证 path/mode/presence，退休 checker 的 `READY` 不表示 representation、语言或上线 gate 已通过，也不授权删除或 publication。此 dependency/currentness 边界变更使旧 formal CLI verification 失效；维护者须在首次真实使用前依次复核 current docs、implementation、direct CLI contract tests 并完成 Formal Command Contract v3，之后才能另行执行 readonly retirement check。

future sync仍为upstream workspace → exact diff →维护者批准→deterministic repo-local更新→manifest/current-check→source/route/dependency/content-unit reconciliation。whole upstream commit变化不等于所有package/locale stale。当前migration contract只支持既有frozenbaseline；未来V2-C1需显式successor，不用本命令覆盖新commit。
