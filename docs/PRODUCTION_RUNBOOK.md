# 生产运维手册

本文档记录当前已验证的 production 基线，并严格区分“新 locale 首次生产部署”和“已有 locale 日常维护部署”。项目进度和部署历史见 [PROJECT_STATE.md](PROJECT_STATE.md)；新增语言的完整阶段导航见 [新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)。

**Site v2 release/deployment 审计结果：FAILED（2026-10-14）。** 下文现有入口仍是 ACTIVE 的 Tour production 路径；不能据其历史 PASS 或共享 helper 推定 current activated Site v2 Homepage / translation / Learn / Docs、assets-go-dev、batch/repeat、verifier 和 search closeout 已通过。[完整 executable inventory](RELEASE_EXECUTABLE_INVENTORY.json) 与[跨入口审计](SITE_V2_RELEASE_PATH_AUDIT.md)记录当前缺口、逐模式检查和修复合同。Site v2 发布前必须闭合其中阻塞项；本次审计不授权任何 Production mutation，也未改写历史 evidence。

本文档中的标准 Production 终端步骤由维护者 Local terminal 执行，并使用相同命令、前置条件、验收标准和停止规则。AI execution environment 准备完整命令并直接消费维护者回传的合法终态，remote 默认 0；真实 failure diagnosis 与明确 remote 请求按[Call-Zero-First](WORKFLOW_HANDOFF.md#call-zero-first-交接合同)处理。语言闭环与 Production 的 deterministic repository / shell workflow 均默认 Local terminal，不按耗时授权 AI remote mutation。只有明确标为 **HUMAN GATE** 的 UI 操作仍必须由维护者人工完成；其他步骤虽是 deterministic，也不因此改由 AI 自动运行。

`preview` browser verifier、shared-assets Production、`publish` / prerender、`first-production`、maintenance Production 与 IndexNow closeout 等 deterministic workflow 与所有短步骤同样默认由维护者本地终端执行，与预计耗时无关。ChatGPT / Remote Desktop Commander 提供一条完整可粘贴命令并读取终态，不自行运行，也不得仅为等待任务完成持续轮询。该协作边界不省略任何脚本内部 gate、receipt、bounded retry 或 HUMAN gate；按正式发布顺序需要的最终 Git commit / push 同样由维护者执行。

正式脚本执行失败必须返回非零 exit status，不得关闭、替换或持续污染维护者的交互 shell；本文提供给维护者直接复制的命令不得在当前交互 shell 顶层执行 `set -e`、`set -u`、`set -o pipefail`、`set -euo pipefail` 等会影响后续交互行为的 shell options，也不得包含 `exit`、`logout`、`kill $$`。如需 fail-fast 或临时 shell options，必须放入独立 subshell（例如 `( set -euo pipefail; ... )`）或独立脚本进程，使失败只结束子进程。此规则不禁止脚本自身正常结束，也不禁止 shell wrapper 在自身进程中以正常 `exec` 调用正式子程序。正式脚本应作为独立程序执行，不要求维护者 `source` 一个可能调用 `exit` 的脚本。

## Current Site release contract

当前 build、publish、single/repeat/first/batch/all-live/selected deployment 共用 `internal/sitecontent.LocalCoverage`、completed payload 和同一 sealed release contract。只发布 activated complete local routes；Homepage、translation、Tour、Learn、Docs 按实际 completion 进入，未覆盖请求是真 404，正文导航可以指向 go.dev。历史 `home.html` 不进入任何 current release。canonical、sitemap、browser plan 和 search submission 来自同一 coverage；固定历史 page count 仅可作为历史记录。

`build` 在现有 `_content` projection 中导出 `site/release-content.json`、rendered Site pages、structured bytes、Homepage toys 和 carried browser plan。`build.json`、exact-set `SHA256SUMS` 绑定输入；`release-contract check --current` 验 coverage、source/compiler inputs、public target 与共享 assets 的 exact inventory/bytes。消费已经验收的 build 时，publish 必须使用其完整 `SHA256SUMS` identity，直接复制 sealed payload：

```sh
go run -mod=readonly ./cmd/tour-i18n release-contract check --input <build-dir> --current --json
go run -mod=readonly ./cmd/tour-i18n publish --locale <locale> --output <release-dir> \
  --published-at <RFC3339-UTC> --input-build <build-dir> --build-identity <input_identity>
```

`release.json` schema 3 记录 Site/build content identity、完整 input build SHA 与 shared-assets dependency。bundle 的 `SHA256SUMS` 是 deployment input identity；远端 staging/startup 由同一 `bin/tour -verify-bundle` 检查 exact set、路径、checksum、compiled locale 和完整 contract。`/site-release.json`、current pointer、MainPID executable、origin/public HTTP 与 browser 都绑定这个 exact input。输入 stale、未知 mutation、CAS predecessor 不一致均停止，不 fallback 到旧 bundle。不同生命周期只增加基础设施 gate，不改变 content authority。工程回归不等于实际部署或正式 Preview/browser/HUMAN acceptance。

## 已验证服务器基线

除非已验证基线实际发生变化，新 locale 首次部署直接复用以下事实，不重新探测或重新设计整台服务器：

| 项目 | 已验证基线 |
| --- | --- |
| 主机与登录 | 阿里云源站 `121.40.248.29`；部署脚本使用 SSH alias `aliyun`，远端生产运维账号为 root |
| Web stack | OneinStack，唯一正式维护目录 `/root/oneinstack` |
| Nginx | aliyun 与 zgocloud 的 OneinStack executable 均为 `/usr/local/nginx/sbin/nginx`；vhost 位于 `/usr/local/nginx/conf/vhost/`，证书/私钥位于 `/usr/local/nginx/conf/ssl/`；配置检查与重载使用 `/usr/local/nginx/sbin/nginx -t && service nginx reload` |
| Python | aliyun 的 system/default `/usr/bin/python3` 为 Python 3.6.8；Alibaba Cloud Linux 3 的 dnf/yum 使用 `/usr/libexec/platform-python`（Python 3.6）。当前不升级或替换 system/platform Python，所有 aliyun production remote helper 必须兼容此实际基线。未来可单独评估升级日常 `/usr/bin/python3`，但不得直接替换 `platform-python`。zgocloud 当前为 Python 3.10.12。 |
| TLS | Let's Encrypt，由 OneinStack 内部 acme.sh 与 Cloudflare DNS API（`cf` / `dns_cf`）管理，当前使用 `ec-256` |
| 应用数据 | 每个 locale 使用独立 `/data/go-tour[-<locale>]/`，包含 `releases/`、原子 `current` symlink 和 `.deploy.lock` |
| 进程管理 | systemd；service user 为 `go-tour`；每个 locale 使用独立 service 与 `127.0.0.1:<port>` |
| 入口链路 | CDN/DNS → HTTPS Nginx vhost → locale loopback service；非中文社区语言使用 Cloudflare Free |
| 执行链路 | 浏览器 Run / Format → `https://play.go-dev.shuijingwanwq.com:8443` → `play.golang.org`；生产源站不开放本地 `/socket` |
| 非中文静态资源 | `https://assets-go-dev.shuijingwanwq.com/` 的固定 allowlist；bundle 仍保留完整本地副本 |

基线复用不等于省略目标校验：首次部署仍须确认新 hostname、端口、路径和 service 没有冲突。只有只读检查显示基线已变化或新 locale 有明确不同需求时，才单独调查并更新本手册；不要把例行部署变成全服务器环境探测。

## 当前 production profile

`production/identity.json` 是当前 production locale、hostname、CDN、service、port、路径与 lifecycle 的唯一 machine-readable authority；本手册不维护第二份完整列表。维护者可使用以下只读命令查看全部或仅 live profile（依次输出 locale、state、public URL、CDN、service）：

```sh
python3 scripts/production-identity.py list
python3 scripts/production-identity.py list --state live
```

本手册保留架构差异：zh-CN 经 EdgeOne，community profile 使用 Cloudflare Free；每个 profile 都经 Nginx 转发到独立 loopback service。

`zh-CN` 请求链路为 Cloudflare 权威 DNS → 腾讯云 EdgeOne → 源站 `121.40.248.29:443` → Nginx → `127.0.0.1:3999`。EdgeOne 到源站使用 HTTPS，回源 Host 为 `go-dev.shuijingwanwq.com`。

### Production CDN 缓存策略

当前 production hostname 统一采用约 1 个月的 CDN Edge Cache TTL，不主动把 Browser Cache TTL 强制设为 1 个月。Cloudflare 正式共享 Cache Rule 为 `ends_with(http.host, "-go-dev.shuijingwanwq.com")`，将匹配 hostname 标记为 Eligible for cache，Edge Cache TTL 为 1 month；`assets-go-dev`、现有非中文语言站及未来非中文 `xx-go-dev` locale 自动复用，不逐 hostname 修改 Cache Rule：

- `go-dev.shuijingwanwq.com`：EdgeOne 节点缓存 TTL 为 30 天，匹配整个 hostname，强制缓存关闭；已用首页 `/` 与课程页 `/tour/welcome/1` 验证公网 `MISS → HIT`。
- 非中文 `*-go-dev.shuijingwanwq.com`：复用上述共享 Cloudflare Cache Rule；首页 `/` 与课程页 `/tour/welcome/1` 均按正式 machine gate 验收。
- `assets-go-dev.shuijingwanwq.com`：复用上述共享 Cloudflare Cache Rule；shared-assets 按正式 receipt 的实际 changed paths 自动做精确 URL purge。

截至 2026-09-13，Cloudflare community locale 已启用 Smart Tiered Cache，Zone 的 Tiered Cache Topology 为 `Active`；这是当前已验证的 Production baseline。其目的在于减少不同 edge 的 cold MISS 各自直接建立到中国内地阿里云源站的回源连接，改善 Cloudflare → 阿里云内地源站路径的稳定性。当前使用 Smart Tiered Cache，不配置 Alibaba Cloud 的 cloud region hint，也不使用 Enterprise-only custom topology；这不改变既有共享 Cache Rule、language hostname purge、shared-assets changed-URL 精确 purge 或 verifier gate。

Smart Tiered Cache 不是 cache warm-up 保证，也不能替代正式 machine/browser acceptance。language hostname purge 后仍须通过当前正式 verifier gate；等待一段时间本身不证明缓存已经 `HIT`。2026-09-13 启用后，针对此前失败的 Angular partial fresh MISS 只读复测为 10/10 HTTP 200、单次约 0.3–0.4 秒，后续正式 production maintenance 的全部 locale 最终通过 machine/browser acceptance；这些是当次 Production evidence，不构成永久的延迟、命中率或可用性保证。

language production 使用固定 URL，因此 release 更新后不能等待约 1 个月自然过期。zh-CN profile 激活后应对 EdgeOne 执行其 hostname 缓存刷新；Cloudflare community profile 激活后应按其 formal production hostname 执行 Cloudflare Custom Purge。不得为刷新单一 language hostname 使用会影响同 zone 其他 hostname 的 Purge Everything。shared-assets 继续使用 automatic changed-URL 精确 purge 流程，不改为整 hostname purge。

hostname purge 后观察到 `MISS → HIT` 是理想结果，但真实 CDN 可能在连续多次请求中仍返回 `MISS`，因此 language production 的 machine gate 不以固定次数内出现 `HIT` 或任何固定 cache status 时序作为通过条件。真实公网 `CF-Cache-Status` 是唯一正式 cache eligibility machine gate：`MISS`、`HIT`、`EXPIRED`、`REVALIDATED`、`UPDATING`、`STALE` 通过；`DYNAMIC`、`BYPASS`、header 缺失及未知值 fail closed。

公网域名迁移不改变 `go-tour.service`、`/data/go-tour/`、仓库名或 Go module path。

## 生产统计配置

Go Tour 统一通过 `TOUR_ANALYTICS` 注入生产统计代码。生产启动路径最终将该环境变量作为 `analyticsHTML` 传给模板的 `{{.AnalyticsHTML}}`；首页 `/` 与 `/tour/...` 页面共用同一个入口。生产环境可在同一个变量中同时放置 Google Analytics 4 和百度统计两套完整 HTML / JavaScript 代码：

```text
TOUR_ANALYTICS='<Google Analytics HTML><Baidu Analytics HTML>'
```

本地开发默认不设置该变量，因此不会加载生产统计代码。实际统计代码以及具体统计 ID 不写入 Git 仓库；公开前端标识也不在本手册中固定记录。

Google AdSense 使用独立的 `TOUR_AD_HTML`。该变量包含 production AdSense HTML / loader；只有当前 shared advertising authority 判定 Tour 广告启用的 locale 才会读取并注入该受信任 HTML。`GoLocal` 与 `AdsUnsupported` runtime 即使环境文件中仍残留该变量，也必须忽略并不注入。实际 AdSense HTML、publisher ID 和完整变量值不写入 Git 仓库。

当前课程页共享广告架构保持不变：课程 editor partial 提供手动 `course-ad` mount 容器，Angular route view 在 link / `$destroy` 时调用其 mount / unmount，模板加载课程广告 CSS/JS；该 helper 创建 responsive AdSense `ins` 并请求广告。当前课程广告正在运行 ABCD 自然流量实验：A/B/C/D 各 25%，同一浏览器 tab session 通过 `sessionStorage` 保持稳定，存储不可用时回退为当前 window 内存；A/B/C 分别限制容器最大宽度为 336/468/728px，D 保持不限制宽度的 Responsive 对照组。四组均保持 `data-ad-format="auto"` 和 `data-full-width-responsive="true"`，不使用固定尺寸广告。局部 layout protection 只移除 AdSense/Funding Choices 曾写入编辑器祖先的两种高度覆盖，以保持课程高度和 footer 布局。课程页广告资源与 Auto Ads 一起构成最终广告形态，并非“只注入 Auto Ads、不插入手工广告位”。本手册不重新设计这套共享广告架构。

正式服务通过 systemd `EnvironmentFile` 读取统计和广告配置：

```text
/etc/go-tour/go-tour.env
```

该文件权限应为 `600`，所有者为 `root:root`。以下是 `go-tour.service` 的既有历史 drop-in 配置示例，不是所有 locale 的统一强制架构：

```text
/etc/systemd/system/go-tour.service.d/analytics.conf
```

内容为：

```ini
[Service]
EnvironmentFile=/etc/go-tour/go-tour.env
```

目标 production service 必须引用正确的 `EnvironmentFile`；该引用可以直接写在 unit 本体中，也可以由既有 drop-in 引入，新 locale 不要求为了形式统一额外创建 drop-in。`Advertising=standard` 且 Tour 广告启用的 locale 必须在该文件中提供有效、非空 `TOUR_AD_HTML`；`GoLocal` / `AdsUnsupported` 不要求非空广告变量，并且 runtime 必须忽略可能残留的值。新增或修改已批准的 unit/drop-in、EnvironmentFile 或 `TOUR_AD_HTML` 后，必须生成新的 immutable publish bundle并执行普通 maintenance；既有 deploy helper 在 exact current/predecessor/assets gates后负责 daemon-reload、current switch和目标service restart，再执行 hostname purge与machine/browser。可复用 unchanged reviewed build，不重复语言审核。不得把裸 `systemctl daemon-reload`/restart作为独立 Site release入口，也不得重用旧完整PASS来跳过环境更新；新publish timestamp/input identity区分这次runtime更新。不要在 shell 历史、发布包或其他仓库文件中复制完整统计或广告 HTML。

修改 `TOUR_ANALYTICS`、`TOUR_AD_HTML` 或其他影响 HTML shell 的内容后，必须在相关 service restart 后按“Production CDN 缓存策略”刷新对应 language hostname；不要仅根据源站验证或 `deploy-production.sh` 的 public HTTP 200 就判断新 release 已在全部 CDN 边缘节点生效。公开的 `/socket` 既有安全原则保持不变：production 不注册或开放本地 Socket transport，普通请求和 WebSocket Upgrade 均应保持 404。

### 广告职责、首次接入与最终验收边界

publication 与 advertising 是独立 policy 维度，共享唯一 authority `internal/tourpolicy`。`Publication` 中 `GoLocal` 仍且仅为 `zh-CN`、`fr-FR`、`de-DE`、`ko-KR`，继续控制既有 owner-content/navigation agreement；不得为了禁用广告把其他 locale 改成 GoLocal。`Advertising` 的维护者冻结优先级为 **AdsUnsupported > GoLocal > Standard**：

- `AdsUnsupported`：`sw-TZ`、`kk-KZ`、`fa-IR`、`am-ET`。四门 publication 仍为 `Standard`，但整个站点所有 current/future surfaces 均不得显示广告；
- `GoLocal`：homepage、`/translation/`、Tour 无广告；Learn/Docs 可按正式 Site policy 显示广告；
- `Standard`：homepage、`/translation/` 无广告；Tour 与正式 Learn/Docs 可显示广告。

该 exact set 不从 locale tag、语言名称、Google/其他广告平台列表或联网查询动态推断；也不写入 `production/identity.json`。Auto Ads、课程页手动广告、Angular SPA mount/unmount 生命周期、局部 AdSense layout protection，以及覆盖这些行为的 browser tests，都是项目共享实现。新增 locale 不重新设计共享架构，也不把广告纳入 TranslationUnit 或 Locale-level language quality review。

Tour 广告启用的 `Standard` locale 必须在**首次 production release 激活前**完成以下 production 广告接入准备：

- 确认目标 locale 的 production service 引用正确 `EnvironmentFile`，并包含有效、非空 `TOUR_AD_HTML`；该文件可由 unit 本体或既有 drop-in 引入，无需为新 locale 额外创建 drop-in；
- 完成 Auto Ads 所需的 production 配置；
- 准备并验收课程广告 CSS/JS 的 production asset 来源（所有 locale 使用 sealed shared-assets policy；zh-CN 的 Tour 广告仍由 GoLocal policy 禁用）；
- 非中文广告启用 locale 使用 shared-assets 时，在首次上线前确认共享的 `course-ad.css` 与 `course-ad.js` 已部署，且已完成缓存验收；这项广告专项检查不能替代完整 shared-assets current-state freshness gate。

`GoLocal` 与 `AdsUnsupported` locale 不接入 Tour 广告 runtime；first-production preflight 仍要求正式 `EnvironmentFile` 本身存在、权限正确，但不得因 `TOUR_AD_HTML` 缺失/为空而失败。即使环境文件残留非空 `TOUR_AD_HTML`，runtime 也必须证明不注入 loader/HTML，不加载课程广告 CSS/JS，不初始化 mount/slot，也不产生本项目广告 request opportunity。`AdsUnsupported` 还要求 Site v2 Learn/Docs/future surfaces 同样保持无广告。

首次 production 激活后，只在同一次最终 production acceptance 中按 shared advertising policy 完成轻量广告确认，并记录在现有 `data/locale-surface-reviews/<locale>/<review-id>.md` 的 `production verification result`：

- `Standard` 且 Tour 广告启用：实际 production HTML 已加载或生成预期的 AdSense loader / Auto Ads head 配置，课程页存在手动广告 mount，浏览器存在真实广告请求机会；广告可为 filled 或 unfilled，不以填充为通过条件；
- `GoLocal`：Tour 不存在 AdSense loader、已初始化的手动广告 mount/slot、`course-ad.js/css` request 或本项目广告 request opportunity；Learn/Docs 按正式 Site policy 单独判定；
- `AdsUnsupported`：整个站点所有已覆盖 surface 均不存在广告 loader、已初始化 mount/slot、广告 helper request 或本项目广告 request opportunity；
- 课程高度与 footer 没有明显布局异常；
- SPA 跳到下一页正常。

此处不要求每个 locale 为广告专项重跑 observer 累积、完整 browser regression、广告失败隔离，或 Run / Format / Reset 等共享广告架构测试；这些属于共享实现的变更验证。新 locale 原有的基础浏览器与 Playground 验收仍按首次部署清单执行，但不以它们替代或扩大上述五项广告确认。也不把“无广告完整验收 → 开广告 → 再完整验收”作为首次上线流程。

## 域名与 EdgeOne

- `go-dev.shuijingwanwq.com` 使用 Cloudflare CNAME → `go-dev.shuijingwanwq.com.eo.dnse2.com` → EdgeOne。
- 旧域名 `go-tour.shuijingwanwq.com` 仅作为兼容入口保留；仍保留 Cloudflare CNAME、EdgeOne 域名和 HTTPS 证书，正常请求由 EdgeOne 直接永久 301 到新域名同路径并保留查询参数。
- 旧域名不再拥有独立的源站 Nginx 虚拟主机；若发生回源，回源 Host 使用 `go-dev.shuijingwanwq.com`。

## Nginx

正式配置与证书：

```text
/usr/local/nginx/conf/vhost/go-dev.shuijingwanwq.com.conf
/usr/local/nginx/conf/ssl/go-dev.shuijingwanwq.com.crt
/usr/local/nginx/conf/ssl/go-dev.shuijingwanwq.com.key
```

反向代理目标为 `http://127.0.0.1:3999`。旧 go-tour 虚拟主机、旧 SSL 文件和 Nginx 备份配置已从源站清理。

检查并重载配置：

```sh
# current Site release 不以裸 nginx reload 作为独立入口。
# 配置检查/reload由绑定sealed input的first-production或IndexNow helper执行。
```

历史裸 Nginx reload 命令退役为独立 release path；表中的 test/reload executable值仍供正式helper读取。需要调整已有locale vhost时先停止供维护者制定明确的配置migration/recovery，不用裸reload冒充完整Site release。当前环境中 `service nginx configtest` 不可用；`service nginx reload` 会正常转发到 `systemctl reload nginx.service`。

使用 OneinStack 新建 Tour 反向代理虚拟主机后，应检查自动生成的额外 `location`。不能让 Nginx 本地静态文件规则截获 `/tour/static/` 请求，否则会造成 404。修改后必须验证真实静态资源返回 200。不要根据本手册伪造或恢复具体 `location` 内容。

## OneinStack 与 HTTPS 证书

当前唯一正式维护目录为 `/root/oneinstack`。凡 OneinStack 已提供脚本或内置流程的操作，优先使用 OneinStack；只有脚本无法满足需求时，才直接调用底层工具或手工处理。

新增反向代理 HTTPS 虚拟主机时，已验证的入口为：

```sh
cd /root/oneinstack
./vhost.sh --proxy --dnsapi
```

本次 go-dev 使用 Let's Encrypt、HTTP → HTTPS、`ec-256` 和 Cloudflare DNS provider（`cf` / `dns_cf`），反向代理为 `http://127.0.0.1:3999`。OneinStack 内部使用 acme.sh；正常情况下优先通过 `vhost.sh` 流程创建或管理证书。

acme.sh 记录必须与当前实际 HTTPS origin 一致，不再采用“本项目只应保留 go-dev 一条记录”的旧说明。当前项目至少包含 `go-dev.shuijingwanwq.com`、`ja-go-dev.shuijingwanwq.com` 和 `assets-go-dev.shuijingwanwq.com` 的有效证书用途；新增 locale 还应增加自己的 hostname 记录。旧 `go-tour.shuijingwanwq.com` 已不拥有源站 vhost，其旧管理记录和残留目录已清理。不要仅凭本文枚举删除证书；删除前必须核对当前 vhost 引用与 acme.sh 实际记录。

Cloudflare API Token 及其他密钥属于敏感凭据，不写入文档、不提交仓库，也不记录真实值。

## 正式 production identity 与 secret 边界

所有 language production 的非 secret 身份统一保存在 [`production/identity.json`](../production/identity.json)。`scripts/production-identity.py validate` 对 schema、路径边界、URL/port 对应关系以及 hostname、port、service、data root、vhost、证书等跨 locale 冲突做严格校验；`deploy-production.sh`、`verify-production.sh` 与首次生产编排器均消费这一个来源，不再各自维护 profile case。JSON 是 Go 可直接解码的稳定格式；shell 通过 `scripts/production-identity.sh` 的有序无 `eval` 接口读取。

新 locale 在这里取得正式、唯一 identity 前，所有生产命令都 fail closed。不得从 locale 字符串推导 hostname、port、service 或路径，也不得静默 fallback 到其他 locale。`production_state` 是首次生产的显式生命周期门：编排器只接受 `first-production`；已经上线的 locale 必须为 `live`，即使 `current` 或历史 receipt 缺失也不得再次 bootstrap。首次生产及 evidence 完成后必须把该字段改为 `live`；日常 deploy/verify 接受已建立的正式 identity，不借此字段猜测远端状态。该文件只保存 identity，不保存 credential、广告 HTML、analytics HTML 或私钥。

Cloudflare API Token 的正式来源为 aliyun 上的 `/etc/go-tour/cloudflare.env`，必须是 `root:root`、mode `0600` 的普通文件，并定义非空 `CF_Token`。管理员只需一次性 provision；不要在命令行参数、shell history、日志、receipt 或 Git 中写入 token。可先安全建立空文件，再用 root editor 填写：

```sh
ssh aliyun 'install -o root -g root -m 0600 /dev/null /etc/go-tour/cloudflare.env'
ssh -t aliyun 'editor /etc/go-tour/cloudflare.env'
```

编排器只检查文件 identity 和变量是否存在，不输出变量值。Cloudflare zone ID 不落库：每次用正式 zone name 查询，且结果必须唯一。

EdgeOne 的正式非 secret authority 同样只来自 `production/identity.json`：`shared.edgeone_zone_name` 与 `shared.edgeone_secret_file`。当前 secret 文件为 aliyun 上的 `/etc/go-tour/edgeone.env`，必须是 `root:root`、mode `0600` 的普通文件，并且恰好定义非空 `TENCENTCLOUD_SECRET_ID` 与 `TENCENTCLOUD_SECRET_KEY`。代码和日志不得读取回、打印或持久化这两个值；ZoneId 不写死，每次由 zone name 精确解析。真实 mutation 前可执行只读检查：

```sh
scripts/verify-edgeone-authority.sh zh-CN
```

该命令只检查 secret 文件安全性，并调用 `DescribeZones`、`DescribeAccelerationDomains` 证明 exact enabled zone 与 exact online hostname ownership；不会调用 `CreatePurgeTask`、刷新缓存或修改 DNS。成功输出 `EDGEONE AUTHORITY PREFLIGHT: PASS`。

EdgeOne client 使用 Python stdlib 实现腾讯云 API 3.0 `TC3-HMAC-SHA256`，固定 endpoint `https://teo.tencentcloudapi.com`、service `teo`、Version `2022-09-01`，不依赖 tccli 或腾讯云 SDK。`DescribeZones` 必须只返回一个 `ZoneName` exact、非空 `ZoneId`、`ActiveStatus=active`、`Paused=false` 且 `LockStatus=enable` 的 Zone。`Type=partial`（CNAME 接入）要求 `CnameStatus=finished`，并接受表示 NS 尚未切换的 `Status=pending`；`Type=full` 要求 `Status=active`，不以 `CnameStatus` 为 gate；其它 Type fail closed。随后以动态 ZoneId 调用 `DescribeAccelerationDomains`，必须只返回一个 exact hostname、exact ZoneId 且 `DomainStatus=online` 的域名。0 个、多个、mismatch 或未知状态均 fail closed。`CreatePurgeTask` 必须返回非空 JobId 和空 FailedList；按 exact JobId 调用 `DescribePurgeTasks` 时，创建后暂时返回 0 个 task 按既有次数和间隔 bounded polling，耗尽仍不可见则失败，多个 task 或不一致响应仍立即 fail closed。唯一 task 只接受 `processing → success`，`failed`、`timeout`、`canceled`、未知状态、identity mismatch 或 polling exhausted 均失败。mutation response 未知时，以 start timestamp 前后受限窗口、exact ZoneId、`purge_host` 与 exact target 查询；唯一任务继续 polling，多个匹配立即失败，始终无法归因则失败且不盲目重发。

## 新 Locale 首次生产部署

首次部署是建立新 locale 的 production 基础设施，不得直接套用日常 release 切换。开始前必须已完成 TranslationUnit promotion、完整 projection、Surface Review 的语言质量与 preview acceptance、production publish，以及 shared-assets current-state freshness gate。

正式入口只有 release 目录：

```sh
scripts/first-production.sh \
  /tmp/go-tour-release-YYYYMMDD-<locale>-<shortsha>
```

脚本从 `release.json.locale` 选择正式 identity，并要求其显式为 `production_state=first-production`；当前 lifecycle 必须以 `production/identity.json` 为准，`live` profile 不能用于 bootstrap。脚本不接受调用者重复传 hostname、port、service、data root、origin IP 或 zone。一个 run 为 aliyun 和 zgocloud 分别建立 invocation-scoped ControlMaster；这些 master 不设置定时 idle `ControlPersist` 截止时间，只由当前编排器在正常、失败和 INT/TERM/HUP 退出时显式关闭并清理 control socket。当前 mutable execution state 写在 `<release>.first-production-state.json`，完整成功另存 exclusive `<release>.first-production-result-<input_identity>.json`。历史 receipt 保持 immutable；新 state 绑定 input/content/shared-assets identity、prior state 和 mutation status。ATTEMPTED/UNKNOWN 不允许 replay。若已完成 deploy 后本地连接中断，重跑同一命令会识别未完成 receipt，重新验证 `current`、service/source health 和全部关键 identity，再从后续幂等阶段继续；已完成 receipt 拒绝重复 bootstrap。

正式顺序固定为：

```text
全量无 mutation preflight
→ aliyun data root / systemd / DNS-01 TLS / Nginx bootstrap
→ ZgoCloud Playground Origin 精确幂等更新与接口/boundary 验收
→ scripts/deploy-production.sh <release-dir>
→ zgocloud --resolve direct-origin acceptance
→ Cloudflare proxied A record
→ zgocloud minimal public readiness
→ scripts/verify-production.sh <release-dir>
→ Chrome automated browser acceptance
→ passed receipt + READY FOR FINALIZATION
→ 维护者显式执行 first-production finalize
```

### FIRST_DEPLOYMENT health failure recovery

首次部署在 `current` 首次指向新 release 后若 service 无法连续达到 active + HTTP 200，不存在旧 release 可以回滚。`deploy-production.sh` 会保留失败 release、`current` 和 `.deploy.lock` 作为 evidence，且不会继续 DNS stage。不得直接重跑 deploy 或手工删除 lock。若 service 仍不健康，修复代码后必须重新 `publish` 生成新的不可变 release；不得覆盖、删除或重用失败 release。若 health timeout 后同一 service 自行恢复，则也不得把当时失败直接解释为部署通过，必须由下述正式 recovery 入口验证 current、完整 release checksum、service MainPID 对应的精确 binary 与连续 active + HTTP 200，再机械记录 deploy PASS。

唯一正式恢复入口是（参数必须是保留现场对应的**旧**失败 release 本地目录）：

```sh
scripts/recover-first-production-health-failure.sh \
  /tmp/go-tour-release-YYYYMMDD-<locale>-<failed-shortsha>
```

该命令只消费 failed current execution state 与 exact sealed input，并要求 failure stage 是 deploy、前置三个阶段完整 PASS、无任何 post-deploy stage。它证明 whole bundle、current、MainPID、连续 health、shared manifest 和属于该 deployment 的 consumer lease；只有同一 release 已恢复健康才释放已证明的两把锁并记录 known deploy PASS，随后对同一 input resume。仍不健康、历史 Tour-only predecessor、checksum/dependency 不一致或 mutation unknown 均保留 current、锁和 evidence，停止供维护者制定明确 recovery；不删除 current，不恢复部分目录或 binary。

preflight 在任何 production mutation 前同时检查：正式 bundle 与 identity、**唯一** current Surface Review A gate 所指向的 Markdown evidence identity，以及完整唯一且未改写的 first-production finalization placeholder；多个 current A gate 不猜测“最新”而是 fail closed。TODO/unknown locale 间接由 publish/identity gate 拒绝、两台 SSH 和 root account、port/service/data-root/vhost/certificate 冲突、EnvironmentFile、shared advertising policy（仅 Tour 广告启用的 `Standard` locale 要求非空 `TOUR_AD_HTML`，不输出值）、Cloudflare secret 权限与变量、zone 唯一性、目标 DNS 无冲突、Playground 两个 location 的结构一致性，以及 shared-assets origin/public SHA-256 freshness。placeholder 语义由 `tour-i18n first-production evidence-preflight` 统一实现，编排器不自行解析 Markdown；finalize 还会验证其传入的 review-id 对应 gate 本身仍为 current。它重新 export/validate 当前仓库、对照 aliyun origin，并复用正式 shared-assets public verification core（zgocloud runner、HTTP 522/525 和 curl exit 28 的 bounded retry、完整 current inventory SHA-256 与 boundary 404），不信任历史 receipt。任一项失败时，不建立目录、unit、证书、vhost、DNS 或 Origin。

Cloudflare control-plane 专用网络通道也在 preflight 建立：调用机先以 invocation-scoped SSH `-D 127.0.0.1:<local-port>` 连接 zgocloud，再以 SSH `-R 127.0.0.1:<aliyun-port>:127.0.0.1:<local-port>` 连接 aliyun；两个 listener 都只绑定 loopback，且 `ExitOnForwardFailure=yes`/`GatewayPorts=no`。因此 aliyun 的 curl 经 `--socks5-hostname 127.0.0.1:<aliyun-port>` 从 zgocloud 出口完成 DNS 和 TCP，但 HTTPS/TLS 与 `Authorization: Bearer $CF_Token` 仍只在 aliyun 的 curl 进程和 Cloudflare 之间建立，调用机与 zgocloud 都看不到 token 明文。建链失败立即停止，绝不回退 aliyun 直连；正常、失败、INT、TERM、HUP 都关闭两个 ControlMaster 和转发 socket。该 endpoint、端口、token 和 tunnel 细节不写入 receipt 或日志。

基础设施阶段复用已验证的 production service hardening 与 OneinStack Nginx/TLS 结构；所有 server name、proxy port、service、current、证书和私钥路径均从目标 locale 的正式 identity 渲染，不从 fr-FR 或 locale 名称复制值。service 从精确 `current` 启动，TLS 使用 Let's Encrypt/acme.sh `dns_cf`、`ec-256`，Nginx 只有精确 loopback proxy，不生成会截获 `/tour/static/` 的静态 location。证书通过 DNS-01 签发，不要求先建立 production A record；aliyun 的非交互 SSH 调用使用 `/usr/local/nginx/sbin/nginx -t`，通过后才 reload。已存在未知或不兼容配置时停止，不覆盖。失败边界保留为可审计、可 resume 的首次创建物：已经精确写入的 data root、unit 或 ACME/TLS 现场不做不确定删除，后续 preflight 必须重新逐项验证；本次新建 vhost 若 Nginx config test 或 reload 失败则移除并恢复原 Nginx 状态。任一失败均不会记录 infrastructure PASS，service enable/start/restart 失败也不能进入后续 stage。

不假设 acme.sh 支持任何代理环境变量。若需要签证书，preflight 读取实际已安装的 acme.sh 源码，要求其存在 PATH-resolved `curl` 调用且不以绝对 `/usr/bin/curl` 或 `/bin/curl` 绕过；否则在 mutation 前 fail closed。签发时仅为该 acme.sh 子进程创建 mode `0700` 的临时 `curl` wrapper，wrapper 向实际 curl 加入同一 `--socks5-hostname` 参数，随后删除。这样 `dns_cf` 的 Cloudflare 请求和 ACME HTTPS 均走 zgocloud 出口，CF token 仍只从 aliyun 的 root-only secret file 读取。仓库测试无法替代已安装 acme.sh 的真实执行路径；维护者可在不签发、不联网的前提下运行以下最小只读检查（仅显示匹配行，不显示 secret）：

```sh
ssh aliyun 'set -eu; a=/root/.acme.sh/acme.sh; test -x "$a" || a=/root/oneinstack/acme.sh/acme.sh; test -x "$a"; grep -nE "(^|[^[:alnum:]_])curl([^[:alnum:]_]|$)|/(usr/)?bin/curl([^[:alnum:]_]|$)" "$a"'
```

Playground mutation 只接受当前已验证的“两处相同精确 Origin 正则 + 唯一 compile/fmt location”结构，保留全部既有 origin、避免重复，并在 `nginx -t`/reload 失败时恢复备份。随后验证新 origin 的 OPTIONS 204、POST 200、wrong Origin 403、GET 405。

FIRST_DEPLOYMENT 仍严格复用 `deploy-production.sh` 的 upload/current/health 状态机；编排器不实现第二套部署逻辑。源站连续健康后先从 zgocloud 用真实 hostname/SNI 和 `--resolve` 验收 HTTPS、HTTP redirect、关键 route、canonical、`html lang`、shared-assets 与 `/socket`。只有这一步通过才调用 Cloudflare API；已存在完全相同的 proxied A record 幂等通过，未知类型、IP、proxy 状态或重复记录一律拒绝。Cloudflare 的只读 zone/DNS GET 对明确瞬态 curl transport exit `6`、`7`、`16`、`28`、`35` 最多重试 3 次，并使用 1s/2s backoff；每次最终仍要求 curl exit `0`、HTTP 200、Cloudflare `success=true` 以及原有唯一 identity 语义，其他 transport 或 HTTP/API semantic failure 均 fail closed。DNS POST 从不因 transport timeout/失败盲重发：每次 mutation 前先重查 identity，只有 `absent` 才 POST；POST 结果未知后必须重查，`exact` 即成功、`conflict` 立即停止、仍为 `absent` 才可在最多 3 次 mutation 内再次 POST。POST 成功后同样重查，并最终要求唯一 exact proxied A record；尝试耗尽或网络持续失败均 fail closed。API failure 不回滚已健康的 origin。

first-production 不查询或解析 Cloudflare Cache Rule expression，也不修改 account-wide rule；Cache Rule 的实际 eligibility 只由随后从 zgocloud 发起的真实公网 `CF-Cache-Status` machine gate 验证。

Cloudflare DNS 创建后，first-production 只从 zgocloud 对新 hostname 做 bounded public readiness poll：每轮请求sealed bundle 中的representative covered routes，两者均须为 HTTP 200 且具有目标 locale 的 `html lang`；连续三轮成功才进入 verifier。readiness 仅等待 DNS/CDN 稳定生效，不承担第二套 production acceptance；它通过后立即且仅一次调用 `scripts/verify-production.sh <release-dir>`。该脚本是 current routes、canonical 与 `html lang`、activated sitemap exact URL set、普通/Upgrade `/socket` boundary、CDN cache header/eligibility 等通用 machine gate 的唯一正式实现；本机只执行 release identity 与 aliyun remote/source gate，脚本本体以共享 HTTP model 直接在正式 shared `zgocloud` runner 执行全部 current canonical routes 与 sitemap/404 checks，不通过维护者本机↔zgocloud 的 SOCKS 转发整个清单。runner 不可用立即 fail closed；curl exit `6`、`7`、`16`、`28`、`35`、`97` 与 HTTP `522`/`525` 只在单一逻辑请求内最多尝试 5 次，单次 15 秒，耗尽后立即停止（尤其不再继续未验证的 sitemap URL），不能将未检查 URL 记为 PASS。Playground Origin 已在其专属阶段完成接口/boundary 验收，shared-assets 已在 preflight 完成 current-state freshness/public verification；二者不在 readiness 重复请求。`verify-production-browser.py` 随后使用仓库既有 `google-chrome`/DevTools 基线，自动覆盖 desktop/mobile、普通与 editor course、`/tour/moretypes/1`、语言列表、Run/Format/Reset、runtime output、Playground endpoint、canonical/lang/SEO、socket、shared assets、course-ad mount/loader/request opportunity 和 SPA 下一页。first-production 会自动将本次 preflight 已建立的 invocation-scoped zgocloud SOCKS endpoint 显式传给 Chrome；该 tunnel 不依赖 `public-machine` 或其独立 verifier 持续产生 multiplex traffic 来续命，browser stage 要求 listener 仍可达，绝不读取或回退到维护者本机默认代理/网络，也不要求维护者修改本机代理设置。该 tunnel 仍随 invocation cleanup。广告 filled/unfilled 均可通过。

first-production 的前置 public readiness 每个逻辑请求保持最多 3 次，但单次上限为 25 秒，以覆盖已观察到约 21.8 秒才成功的 cold MISS；外层 readiness 次数和 stage timeout 继续限制总预算。它仍只是 DNS/CDN readiness，不替代随后复用的正式 machine/browser correctness gate。

首次 production finalize 后的 Search-engine closeout 不属于 first-production lifecycle 或第三种 deployment。维护者完成 Google、Bing 和适用 locale-specific Dashboard sitemap submission 后，使用 `scripts/indexnow-closeout.sh --locale <locale> --release-dir <current-release-dir>` 作为唯一正式 IndexNow 入口。首次运行会在仓库外的 `${XDG_DATA_HOME:-$HOME/.local/share}/go-tour-indexnow/<locale>/` 生成并保存 locale-specific key，后续重跑只复用同一个有效 key；store/locale directory 为 `0700`，key file 为 `0600`。多个 candidate、symlink、非 regular file、非法 key 或任何造成 identity 不明确的额外条目均 fail closed。已有受保护 key 时可显式传入 `--key-file /secure/path/<key>.txt`，且不会创建默认 key。它从 production identity 取得 Aliyun origin、data root、Nginx vhost 和正式 Nginx command，provision root key/exact location 并 test/reload；config-test 或 reload 明确失败时恢复本轮 vhost/key 修改。该简单单维护者流程不声称 vhost mutation 是 atomic 或 crash-safe transaction。公网验证与 IndexNow submission 由本机直接运行的 Go primitive 执行；没有 failure evidence 不引入代理。probe 只有明确 HTTP 202 才以相同已 provisioned key 和 byte-identical 单 URL payload 最多执行 3 次，backoff 为 1 秒、2 秒；不重新 provisioning，不重试 transport failure 或其他 HTTP semantic failure，耗尽仍 non-zero。bulk HTTP 202 不自动重试。若未来需要境外公网 runner，复用 first-production/verify-production 已验证的 zgocloud direct-runner，不创建本机→SOCKS 的专用架构。IndexNow submission 是第三方 closeout，不修改 production lifecycle，submission failure 不回滚已通过 provisioning 的 key/vhost；明确失败后重跑继续使用同一 key。成功后不周期性重复全站 bootstrap。

编排器对 preflight、infrastructure、Playground Origin、deploy、direct-origin、Cloudflare DNS、public-machine 与 browser 输出 PASS/FAILED 的 wall-clock duration；receipt 仍只保存既有 identity/stage 结果。resume 仅复用 receipt 中已完成的 immutable bootstrap decision（例如同 release 的 unit/vhost baseline），但每次仍重新执行完整 preflight，并重新验证 current、service/source health、DNS identity、shared-assets public freshness 与 CDN/public state；这些 mutable state 不得由历史 PASS 代替。

运营注意：首次 Production 公网验收若没有紧急需求，尽量避开北京时间晚间的跨境网络高峰。发生上述典型 transient failure 时，保留失败现场并在网络条件改善后 resume；不要在没有新的 deployment evidence 时重新 deploy，也不要将网络失败解释为已检查 URL 的 PASS。

`first-production.sh` 在全部 machine/browser 验收 PASS 后写入 passed receipt，并正常停止于 `READY FOR FINALIZATION`；它不会自动改变 lifecycle。脚本打印下面唯一下一条正式命令，其中 release 与 review-id 来自本次已验证 identity。维护者显式执行它；finalizer 不读取 stdin，也不要求新的 Production visual confirmation：

```sh
go run -mod=readonly ./cmd/tour-i18n first-production finalize \
  --release-dir /tmp/go-tour-release-YYYYMMDD-<locale>-<shortsha> \
  --review-id YYYYMMDD-first-production
```

finalizer 从 release/receipt 和正式 identity 获取 locale、hostname、release，拒绝调用者重复输入 production identity。它要求 receipt schema 正确、`result=passed`、`public-machine` 和 `browser` 均为 PASS，且当前 Locale Surface Review A gate 仍有效；然后替换 evidence 中唯一且完整的 finalization placeholder，记录 receipt identity、machine/browser passed、无 blocker 和最终 decision；再以 candidate identity 中目标 locale 已为 `live` 的状态，从首页 language registry 同步生成 `README.md` 与 `README.en.md` 的 community live locale block。旧的未完成 placeholder 若仍包含 Production visual `PENDING` 字段可安全收口，但新 evidence 不再创建该字段。两份 README 都是 derived projection，`production/identity.json` 仍是唯一 Production machine authority；registry URL 与 identity public URL 漂移、缺失或不唯一 join 都会 fail closed。evidence、两份 README 与 identity 原子提交并在最终 identity validation 失败时一并恢复。已 `live` locale 必须拒绝重复 finalize。

成功摘要保持简洁，例如：

```text
[首次生产] 基础设施：PASS
[首次生产] Playground Origin：PASS
[首次生产] 源站验收：PASS
[首次生产] Cloudflare DNS：PASS
[首次生产] 公网验收：PASS
[首次生产] 浏览器验收：PASS
[首次生产] READY FOR FINALIZATION
receipt: /tmp/go-tour-release-....first-production-state.json
next: go run -mod=readonly ./cmd/tour-i18n first-production finalize --release-dir ... --review-id ...
```

失败固定给出 stage、expected、actual 与下一步，不输出 secret，例如：

```text
[首次生产] FAILED
stage: playground-origin
expected: command exit 0
actual: expected HTTP 204, got 403
下一步：按 stage/evidence 检查后重试
```

### Existing locale language-list consistency

首页 language registry 在 build 时写入每个 locale release。新增 locale 的首次 production gate 只验证**新 locale → 已有 locale**：新站必须显示当前正式 registry，current language 不为链接，已有 locale 与官方 English Tour 链接正确。**已有 locale → 新 locale** 采用 eventual consistency：已运行的旧 locale release 不需要在新语言上线当天立即获得新条目；它们在下一次正常 upstream sync、UI 更新或日常 release 的 publish/deploy 中自然获得当前 registry。不得只为反向链接即时同步而重新设计 runtime registry，或批量要求旧 locale 重新 Quality Check、Final Review、Locale Surface Review、publish、deploy、CDN purge 或 production final。

首次上线记录至少包含：上述冻结值、vhost/证书路径、CDN 类型、当前 release、localhost 与 public 结果、sitemap 汇总、Playground Origin、轻量广告确认和浏览器结果。全部通过后，该 locale 才进入日常维护状态。

## 已有 Locale 日常维护部署

本节只适用于已经完成首次生产基线、且正式 identity 明确为 `production_state=live` 的 locale。日常部署不重新分析 OneinStack、Nginx、acme.sh、data root、systemd 或端口；除非预检显示配置漂移，否则使用已验证 profile 发布新 release。`first-production` locale 必须使用首次生产流程，maintenance 入口会 fail closed，绝不能以日常部署 bootstrap。

### Production release 自动部署

使用上述 existing build/publish contract。Tour 的 prerender routes 从 parsed lesson/catalog authority 派生，Site pages 在同一个 sealed payload 中；两个 Chrome worker、逐 route bounded retry、rendered metadata/editor/content checks 保持。不得用旧 Tour count 或一套 batch-only renderer。

普通多 locale maintenance 的顶层入口：

```sh
scripts/production-release-batch.sh --all-live
scripts/production-release-batch.sh --locale de-DE --locale fr-FR
```

`--all-live` 动态读取唯一 production identity；selected 保持参数顺序。先导出/validate current shared assets，再由共同 `publishBundle` 串行构建所有 locale；全部本地 publish PASS 才允许共享 assets mutation，随后全部 locale preflight PASS，再逐 locale `deploy → purge → machine → browser → immutable result`。第一项失败立即停止后续 locale；已有成功 result 保留，失败/blocked/unknown 不计 PASS。

顶层 post-publish state 绑定 HEAD、production identity、ordered locale set、exact releases 与 shared export hashes。恢复入口不扫描临时目录，也不重新 publish：

```sh
scripts/production-release-batch.sh --resume <post-publish-state.json>
scripts/production-release-batch.sh --resume-maintenance <post-publish-state.json>
```

恢复仍检查所有 current sealed input/coverage/asset dependencies。已经成功的 deploy/purge 不重放；已完成 locale 只验证当前 runtime/public/browser，再显示 CARRIED。未知 mutation 停止，不把更换 HEAD/目录、旧 receipt 或旧首次上线 bundle 当作恢复依据。

底层 `deploy-production.sh <release-dir>` 复用 Go current validator 与 current shared export/origin/public proof，再验证 exact prior pointer/payload/PID。仅向 profile staging 上传，远端 exact binary validator PASS 后，CAS whole-tree rename + current symlink switch + service restart；health 使用本 bundle 的 covered route。activation 持有 shared-assets consumer lease，防止中途替换 dependency。自动 rollback 仅接受 exact predecessor 的完整 sealed Site bundle且 shared dependency 完全相同；whole-tree pointer、routes、sitemap/canonical、assets dependency 和 verifier plan 一起恢复。旧 Tour-only predecessor 或不同 assets identity 不具备自动 rollback eligibility。失败/未知保留锁和现场，停止恢复；不只恢复 binary。

### Maintenance production orchestration 与 machine acceptance

现有 single 与 batch wrappers 转发共同 maintenance implementation：

```sh
scripts/maintenance-production.sh <release-dir>
scripts/maintenance-production-batch.sh <release-dir>...
scripts/verify-production.sh <release-dir>
scripts/verify-production-browser.py <production-public-url> <locale> --release-dir <release-dir>
```

每次重新检查 current activated coverage、exact input build/bundle、target、shared export/origin/public currentness 和 predecessor；历史完整 PASS 仍不能替代本次 current proof。每个 locale state `<release>.maintenance-production-state.json` 独立保存 PREFLIGHT、INPUT_IDENTITY、PRIOR_STATE、ACTION、RESULT、VERIFIER、MUTATION_STATUS，全部阶段成功后 exclusive 写 `<release>.maintenance-production-result-<input_identity>.json`。历史 `.maintenance-production-receipt.json` 不改写或升级为 current proof。

batch 先全体 preflight，然后逐 locale 完成 mutation 与 acceptance；某 locale FAILED/UNKNOWN 后，剩余是 BLOCKED，总结果非 PASS。retry 只处理未完成阶段，成功的 deploy/purge 不重复 apply/POST；CARRIED 仍做只读 current machine/browser verification。CDN POST 前 durable ATTEMPTED，已尝试而无 durable terminal result 时禁止 replay。failed-before-process/mutation 可安全重试，CAS predecessor 保持。

origin machine verifier 证明 exact pointer、bundle、compiled locale 与 MainPID；origin HTTP/public HTTP/direct-origin 使用同一 sealed exact route set，校验 localized Site main content、lang、canonical、sitemap、robots、asset hostname/CSP、public HTTPS/hostname/cache eligibility、family missing routes/socket true 404。没有 current local coverage 的 canonical 永不进入 sitemap。curl 继续 bounded transient retry，确定性 semantic failure 不重试为 PASS。

Production browser 直接消费 bundle 中 carried Preview plan 与 shared checks，保留 Tour Run/Format/Reset、语言列表、SPA、socket、广告 policy；同时验证 Homepage interaction、Learn/Docs/translation、canonical、shared asset hostname、console/network。它不维护独立 Site route arrays。本工程 remediation 不重新审计 Preview verifier，也不执行正式 browser acceptance。

## Current shared assets / assets-go-dev

全部 production locales，包括 zh-CN，使用同一 `internal/assets.Paths(root)` exact global static inventory与 `shared-cloudflare` delivery policy。静态 namespace覆盖当前 Homepage/Learn/Docs/translation/Tour 的 CSS、JS、font、image；HTML、locale payload、Tour partial/script.js、lesson/Playground/toy endpoint保持 locale origin。冻结 language renderer之后的共享 delivery adapter采用同一 policy，历史 Surface inputs不改写。

`production/identity.json.shared_assets_policy` 保留原部署事实供历史 language evidence reader消费；current consumer须读取sealed `release-content.json.asset_policy/shared_assets_identity`，不得再依此历史字段跳过 zh-CN 或重复部署 freshness。正式hostname仍为 `https://assets-go-dev.shuijingwanwq.com`，固定logical URL，不建立平行 versioned origin。

### Shared-assets production 发布状态机

维护者在已授权发布阶段执行：

```sh
go run -mod=readonly ./cmd/tour-i18n assets export --output <new-export-dir>
go run -mod=readonly ./cmd/tour-i18n assets validate --input <new-export-dir>
scripts/shared-assets-production.sh <new-export-dir>
```

single 与 production-release batch使用同一global workflow。validate要求完整 current exact inventory/bytes、安全 paths、无symlink/unsupportedentry；数量从currentinventory派生。origin先锁、完整staging、校验和CAS原始backup，再以Linux `renameat2(RENAME_EXCHANGE)`原子交换整个目录，禁止in-place rsync fallback。失败恢复整个同identity backup；unknown保留origin/staging/backup/锁，停止。静态HTTP请求可继续读取完整old或newtree，localeactivation持有consumerlease绑定准确dependency。

origin mutation前exclusive持久化 `.shared-assets-attempt.json`，无durable result时拒绝retry。成功的`.verification-receipt.json`保持immutable；purge/verification的mutable state为`.shared-assets-execution.json`。NO_CHANGES不purge，DEPLOYED只purge receipt的exact changedURLs。changed URLs 按 API 上限分块；每块独立绑定manifest/URL identity与durable attempt/result，retry carry已成功块，任何partial/unknown不能整体PASS。ATTEMPTED/unknown POST不重试；purge PASS后只重试verification。PUBLIC验证完整currentexport SHA、font CORS、非HTML MIME、boundary true404与changedURL cache semantics。CSS/JS/image/font logical paths均保持一致；CSP必须允许共同assetorigin。

每次locale first/repeat/batch部署都重新验证currentexport、exact origin manifest与完整publicbytes。历史assetsPASS只作evidence，不是未来currentnessoracle；stale/shareddependency不匹配时localeProduction failclosed。此工程轮不调用上述productionworkflow，不修改CDN/DNS/origin/service。

## Shared-assets 历史 evidence（只读保留）

下列历史观测与原执行记录原样保留，不属于 current command/policy；其中旧数量、非中文限制与旧操作建议不得用于本轮 Site release。当前 contract 以本页 Current shared assets 一节为准。

此前首次课程广告资源部署的 changed paths 包括：

```text
tour/static/go-dev/course-ad.css
tour/static/go-dev/course-ad.js
```

如果脚本输出其他 changed path，wrapper 会把对应 URL 一并放入同一个正式 API request；删除路径的旧 URL 也必须刷新。`production-cdn.py purge-shared-assets --receipt <receipt>` 是 wrapper 使用的底层接口，只接受 v2 `PENDING` receipt，不接受 URL 或 hostname 参数。Cloudflare Dashboard 不再是本状态机的一部分。


#### 已知网络现象（2026-09-02）

ko-KR 首次 production 前的 shared-assets current-state freshness verification 发现公网波动。维护者本机曾随机收到 HTTP `525`，因此正式 verifier 已固定使用 zgocloud network runner；但 zgocloud 路径仍观察到随机 HTTP `522`、`525`，以及无 HTTP response 的 curl exit `28`（15 秒、0 bytes received）timeout。

对照只读测试中，zgocloud → Cloudflare public、zgocloud → Aliyun direct origin、Aliyun localhost → Nginx 各为 `20/20` HTTP 200；Nginx active、assets vhost TLS 与 direct-origin certificate verification 正常，未见持续性 Nginx/TLS 服务故障 evidence。50 次强制 Cloudflare MISS probe 有 `47` 次 HTTP 200 + `CF-Cache-Status: MISS`，均精确对应 Aliyun Nginx access log 的 `47` 个 HTTP 200；其余 `3` 次 curl HTTP `000` 未进入该 access log。30 次保留 stderr 的 transport probe 有 `18` 次 HTTP 200/MISS、`12` 次上述 timeout。

这表明 zgocloud → Cloudflare / Cloudflare 回源链路存在间歇性网络波动，但不足以推定 Nginx/TLS 持续故障、真实海外用户固定失败率或共享资产架构需要重设计。shared-assets 的长 Edge Cache TTL 会减少正常用户接触回源链路的机会，但不消除此现象。

当前将其作为已知 production 网络风险：保留 zgocloud runner 与 HTTP `522` / `525`、curl exit `28` 的三次 bounded retry，不修改 Cloudflare、Nginx/TLS 或服务器网络参数，也不放宽 receipt、14/14 SHA-256、cache 或 boundary gate；不因主站 verifier 有该分类而机械加入 curl exit `97`。若真实用户错误、production acceptance failure 或监控 evidence 显示持续/扩大，再作为独立基础设施问题调查。已知网络现象不能替代正式 shared-assets freshness verification `PASS`。


Google 官方 `adsbygoogle.js` 继续直接从 Google 域名加载，不下载、代理、镜像或 self-host 到 assets origin。

旧 `assets.go-dev.shuijingwanwq.com` 已废弃并清理，不提供兼容或迁移。不要在 zh-CN 的现有 EdgeOne 发布流程中加入 assets 域名依赖。


## ja-JP production 验收状态（历史 Site v1 evidence）

ja-JP 当前 release `/data/go-tour-ja-JP/releases/20260824-ja-JP-164fecdd` 已完成源站和 Cloudflare Free 公网验收。sitemap 的 105 个 URL 已全部验证，结果为 105/105、host mismatch=0、HTTP failure=0；`robots.txt` 已确认指向 <https://ja-go-dev.shuijingwanwq.com/sitemap.xml>。

Playground 代理当前允许以下两个正式 Origin：

```text
https://go-dev.shuijingwanwq.com
https://ja-go-dev.shuijingwanwq.com
```

ja-JP 的 Playground compile、fmt 和浏览器实际运行均已通过。非中文共享静态资源由 <https://assets-go-dev.shuijingwanwq.com/> 提供。

## 最小生产验收

统计配置变更建议按“源站 → 公网 → 浏览器真实上报”三层验收：

1. 在源站本机检查 `http://127.0.0.1:3999/` 和 `http://127.0.0.1:3999/tour/welcome/1`，确认 Google Analytics 与百度统计 ID 在 HTML 中各出现预期次数。不要把真实 ID 或完整代码写入命令示例、日志或 Git。
2. 检查公网 `/` 和 `/tour/welcome/1` 的 HTML 及响应头。若 EdgeOne 返回 `EO-Cache-Status: HIT` 和较旧的 `Age`，先刷新 Hostname 缓存；刷新后应重新回源并看到 `EO-Cache-Status: MISS`，再确认 HTML 已包含两套统计代码。
3. 在浏览器 Network 面板确认 Google Analytics 请求发往 `analytics.google.com/g/collect`，请求包含正确的 GA4 `tid` 和当前页面地址，响应为 HTTP 204；确认百度统计的 `hm.baidu` 相关请求响应为 HTTP 200。这两类请求分别证明统计代码已实际加载并完成真实上报。

```sh
curl -I https://go-dev.shuijingwanwq.com/tour/welcome/1
curl -I https://go-dev.shuijingwanwq.com/tour/static/css/app.css
curl -I https://go-tour.shuijingwanwq.com/tour/welcome/1
```

预期结果分别为新页面 200、静态资源 200，以及旧域名 301 并指向新域名同路径。必要时可在服务器本机绕过 EdgeOne 验证 Nginx 新虚拟主机：

```sh
curl -I \
  --resolve go-dev.shuijingwanwq.com:443:127.0.0.1 \
  https://go-dev.shuijingwanwq.com/tour/welcome/1
```

源站直连验证优先在服务器本机执行。本地开发电脑可能配置网络代理；若 `--resolve` 响应仍出现 `eo-log-uuid` 或 `eo-cache-status`，不能据此认定已经绕过 EdgeOne。

## 代码运行

生产 Go 示例通过远程 `go.dev` Playground 链路运行。域名迁移后已验证成功；偶发单次超时不属于当前域名迁移阻塞问题，若以后频繁出现再单独排查。
