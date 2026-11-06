# Site v2 release-path blocker remediation closure

baseline audit 与150-entrypoint inventory保持原样；本轮固定关闭原G01–G12，不重扫repository。当前覆盖以 [machine table](SITE_V2_RELEASE_PATH_REMEDIATION.json) 与 [inventory delta](RELEASE_EXECUTABLE_INVENTORY_DELTA.json) 为准。

这是code/targeted-fixture closure，未执行Production或formalPreview/browser/HUMAN acceptance，未改TranslationUnit/QC/Surface/activation/history。

| BLOCKER_ID | exact blocker | STATUS | closure |
| --- | --- | --- | --- |
| G01 | Release authority/runtime 未接 Site v2 | PASS | current LocalCoverageFiles → completed reconstruction → sealed Site export → existing publishBundle/runtime; single/batch/first/repeat never select a separate renderer. |
| G02 | 发布 Homepage 仍为旧 Tour 首页 | PASS | current Homepage/translation from completed frozen shell bytes; retired home.html rejected in input/path/checksum contract and never served. Tour library home remains an isolated compatibility component. |
| G03 | build→publish→deploy→runtime identity chain 未闭合 | PASS | reviewed build accepted with exact full build SHA; content copied without rerender; manifest records content/input-build/shared identities. Same exact-set/safe-path validator seals publish and verifies remote startup/PID/runtime, with predecessor CAS. |
| G04 | 每次部署 freshness 不满足 | PASS | every invocation checks current coverage/compiler/source/target and fresh shared export/origin/public. Historical PASS is immutable evidence; carry revalidates current runtime instead of returning because a file exists. |
| G05 | batch per-locale evidence 不完整 | PASS | all locales preflight first; each locale completes deploy/purge/machine/browser before the next mutation. Independent exact per-locale state+exclusive final result; FAILED/UNKNOWN/BLOCKED cannot be hidden by cached PASS. |
| G06 | assets-go-dev 全链未支持 current Site/zh-CN | PASS | locale-neutral current static inventory and delivery policy include Homepage/Learn/Docs/translation/Tour, zh-CN included. Export/validate/atomic origin/deployed proof/public CORS+hashes/consumer gate/browser share exact dependency and hostname. Raw identity legacy policy remains historical input; it no longer controls current consumers. |
| G07 | Production canonical/sitemap verifier 仍 Tour 固定 contract | PASS | current activated complete route set is the shared sitemap/canonical model. Tour canonical origin must equal sealed production identity; immutable Tour SEO/context source bytes remain unchanged. No fixed route count. |
| G08 | Production HTTP/browser/404 contract 与 Preview 分叉 | PASS | all origin/public/direct-origin routes, content identities, canonical/sitemap/404 use one sealed model. Production browser consumes carried Preview plan/checks plus Tour interactions and shared hostname gate; compatibility wrapper delegates current implementation. |
| G09 | Site Homepage runtime 辅助输入缺失 | PASS | Homepage selected toys are exact frozen auxiliary inputs, not canonical coverage. Preview serves frozen local resource bytes; Production uses the same logical resources through shared delivery. Queries/fragments preserved; no dependency on undeployed Production assets during Preview. |
| G10 | search closeout 不绑定 current activated sitemap | PASS | IndexNow requires current sealed release/shared/public proof and compares submission sitemap with exact complete activated canonical coverage before any POST. Probe comes from that set; sparse locale never submits uncovered root. Google/Bing/locale engine remain manual and use the same verified sitemap; no batch engine PASS claimed. |
| G11 | locale retry/uncertain CDN mutation 缺少持久化绑定 | PASS | durable exact-input attempts precede POST/apply; unknown stops. Successful locale mutation/result and successful shared purge chunks are carried, never replayed. Failed-before-mutation retries only incomplete scope. Shared origin attempt survives missing result; immutable receipts cannot be overwritten. |
| G12 | service/rollback 无完整 Site+assets predecessor contract | PASS | whole sealed predecessor/input CAS and identical shared dependency required for automatic locale rollback; asset origin uses atomic whole-tree exchange and whole-tree restoration. Unknown preserves evidence/locks; first recovery is narrow exact late-health proof only. Bare runbook service commands retired as independent release paths; approved env/unit updates use a new existing publish/maintenance input. |

历史Surface context故障由恢复原production.go/seo.go/shell_render.go的exact字节并把machinegate移至releaseadapter解决；gate未削弱，receipt未改写。完整current export已验证。Preview verifier source SHA保持baseline；其route/interactionplan为trustedcarry，Production只适配delivery/runtime-specific gates。

Standalone runbook daemon-reload/restart/nginx-reload入口退役；内部controlled操作仍由existingdeploy/first-production/closeouthelpers执行。approved unit/env更新需新的publish input并走普通maintenance；不同shareddependency或历史Tour-onlypredecessor不能自动rollback，unknown停止保留evidence。

Currentness/identity：activatedscope→sealedbuild/content→publish(includinginputbuildSHA)→exactbundleSHA→staging/startup/current/PID→origin/public HTTP model→carriedbrowserplan→search currentsitemap。Preview静态资源与Homepage toys从frozenlocalbytes提供；production全部locale（含zh-CN）使用同一sharedexactinventory/host。

验证列表见machineJSON TARGETED_TESTS；未运行go test ./...、full browser regression或任何production/network mutation。
