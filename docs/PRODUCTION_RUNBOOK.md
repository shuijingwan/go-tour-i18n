# 生产运维手册

本文档记录当前已验证的 production 基线，并严格区分“新 locale 首次生产部署”和“已有 locale 日常维护部署”。项目进度和部署历史见 [PROJECT_STATE.md](PROJECT_STATE.md)；新增语言的完整阶段导航见 [新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)。

本文档中的标准终端步骤不绑定执行主体。维护者可以直接执行，也可以在明确要求且具备相应终端访问能力时交由工具执行；两种方式使用相同命令、前置条件、验收标准和停止规则。只有明确标为 **HUMAN GATE** 的 UI 操作必须由维护者完成。标准化流程应能在没有工具持续参与的情况下按文档独立完成。

## 已验证服务器基线

除非已验证基线实际发生变化，新 locale 首次部署直接复用以下事实，不重新探测或重新设计整台服务器：

| 项目 | 已验证基线 |
| --- | --- |
| 主机与登录 | 阿里云源站 `121.40.248.29`；部署脚本使用 SSH alias `aliyun`，远端生产运维账号为 root |
| Web stack | OneinStack，唯一正式维护目录 `/root/oneinstack` |
| Nginx | vhost 位于 `/usr/local/nginx/conf/vhost/`，证书/私钥位于 `/usr/local/nginx/conf/ssl/`；OneinStack executable 为 `/usr/local/nginx/sbin/nginx`，配置检查与重载使用 `/usr/local/nginx/sbin/nginx -t && service nginx reload` |
| TLS | Let's Encrypt，由 OneinStack 内部 acme.sh 与 Cloudflare DNS API（`cf` / `dns_cf`）管理，当前使用 `ec-256` |
| 应用数据 | 每个 locale 使用独立 `/data/go-tour[-<locale>]/`，包含 `releases/`、原子 `current` symlink 和 `.deploy.lock` |
| 进程管理 | systemd；service user 为 `go-tour`；每个 locale 使用独立 service 与 `127.0.0.1:<port>` |
| 入口链路 | CDN/DNS → HTTPS Nginx vhost → locale loopback service；非中文社区语言使用 Cloudflare Free |
| 执行链路 | 浏览器 Run / Format → `https://play.go-dev.shuijingwanwq.com:8443` → `play.golang.org`；生产源站不开放本地 `/socket` |
| 非中文静态资源 | `https://assets-go-dev.shuijingwanwq.com/` 的固定 allowlist；bundle 仍保留完整本地副本 |

基线复用不等于省略目标校验：首次部署仍须确认新 hostname、端口、路径和 service 没有冲突。只有只读检查显示基线已变化或新 locale 有明确不同需求时，才单独调查并更新本手册；不要把例行部署变成全服务器环境探测。

## 当前 production 语言站点

- `zh-CN`：<https://go-dev.shuijingwanwq.com/>，服务 `go-tour.service`，监听 `127.0.0.1:3999`，data root 为 `/data/go-tour/`。
- `ja-JP`：<https://ja-go-dev.shuijingwanwq.com/>，使用 Cloudflare Free；服务 `go-tour-ja-JP.service`，监听 `127.0.0.1:4000`，data root 为 `/data/go-tour-ja-JP/`，当前 release 为 `/data/go-tour-ja-JP/releases/20260824-ja-JP-164fecdd`。
- `de-DE`：<https://de-go-dev.shuijingwanwq.com/>，使用 Cloudflare Free；服务 `go-tour-de-DE.service`，监听 `127.0.0.1:4001`，data root 为 `/data/go-tour-de-DE/`。
- `fr-FR`：<https://fr-go-dev.shuijingwanwq.com/>，使用 Cloudflare Free；服务 `go-tour-fr-FR.service`，监听 `127.0.0.1:4002`，data root 为 `/data/go-tour-fr-FR/`。2026-08-30 已完成首次 production 与最终验收。

`zh-CN` 请求链路为 Cloudflare 权威 DNS → 腾讯云 EdgeOne → 源站 `121.40.248.29:443` → Nginx → `127.0.0.1:3999`。EdgeOne 到源站使用 HTTPS，回源 Host 为 `go-dev.shuijingwanwq.com`。

### Production CDN 缓存策略

当前 production hostname 统一采用约 1 个月的 CDN Edge Cache TTL，不主动把 Browser Cache TTL 强制设为 1 个月：

- `go-dev.shuijingwanwq.com`：EdgeOne 节点缓存 TTL 为 30 天，匹配整个 hostname，强制缓存关闭；已用首页 `/` 与课程页 `/tour/welcome/1` 验证公网 `MISS → HIT`。
- `ja-go-dev.shuijingwanwq.com`：Cloudflare Cache Rule 将整个 hostname 标记为 Eligible for cache，Edge Cache TTL 为 1 个月；该规则与 `assets-go-dev.shuijingwanwq.com` 共用。首页 `/` 与课程页 `/tour/welcome/1` 均已验证 `MISS → HIT`。
- `de-go-dev.shuijingwanwq.com`：Cloudflare Cache Rule 将整个 hostname 标记为 Eligible for cache，Edge Cache TTL 为 1 个月；首页 `/` 与课程页 `/tour/welcome/1` 均已验证 `MISS → HIT`。
- `assets-go-dev.shuijingwanwq.com`：Cloudflare Edge Cache TTL 为 1 个月，Browser Cache TTL 不主动覆盖；shared-assets 继续按部署脚本输出的实际 changed URLs 做精确 Custom Purge。

language production 使用固定 URL，因此 release 更新后不能等待约 1 个月自然过期。`zh-CN` release 激活后应对 EdgeOne 执行 `go-dev.shuijingwanwq.com` Hostname 缓存刷新；`ja-JP`、`de-DE` 与 `fr-FR` release 激活后应在 Cloudflare Custom Purge 中按 Hostname 刷新各自 production hostname。不得为刷新单一 language hostname 使用会影响同 zone 其他 hostname 的 Purge Everything。shared-assets 继续使用已有的 changed-URL 精确 purge 流程，不改为整 hostname purge。

hostname purge 后观察到 `MISS → HIT` 是理想结果，但真实 CDN 可能在连续多次请求中仍返回 `MISS`，因此 language production 的 machine gate 不以固定次数内出现 `HIT` 或任何固定 cache status 时序作为通过条件。正式 verification 只记录 cache status，并确认请求进入预期的 cache eligibility 路径。

公网域名迁移不改变 `go-tour.service`、`/data/go-tour/`、仓库名或 Go module path。

## 生产统计配置

Go Tour 统一通过 `TOUR_ANALYTICS` 注入生产统计代码。生产启动路径最终将该环境变量作为 `analyticsHTML` 传给模板的 `{{.AnalyticsHTML}}`；首页 `/` 与 `/tour/...` 页面共用同一个入口。生产环境可在同一个变量中同时放置 Google Analytics 4 和百度统计两套完整 HTML / JavaScript 代码：

```text
TOUR_ANALYTICS='<Google Analytics HTML><Baidu Analytics HTML>'
```

本地开发默认不设置该变量，因此不会加载生产统计代码。实际统计代码以及具体统计 ID 不写入 Git 仓库；公开前端标识也不在本手册中固定记录。

Google AdSense 使用独立的 `TOUR_AD_HTML`。该变量包含 production AdSense HTML / loader；production runtime 将其视为受信任 HTML，注入每个完整页面的 HTML shell。实际 AdSense HTML、publisher ID 和完整变量值不写入 Git 仓库。

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

目标 production service 必须引用包含有效 `TOUR_AD_HTML` 的正确 `EnvironmentFile`；该引用可以直接写在 unit 本体中，也可以由既有 drop-in 引入，新 locale 不要求为了形式统一额外创建 drop-in。新增或修改 systemd drop-in 时执行 `systemctl daemon-reload`；仅修改 `EnvironmentFile` 内容时，无需因文件内容变化本身执行 `daemon-reload`，但必须重启相关 production service，使新进程重新读取统计和广告环境变量。修改 `TOUR_AD_HTML` 后同样必须重启相关 production service 才会读取新值。不要在 shell 历史、发布包或其他仓库文件中复制完整统计或广告 HTML。

修改 `TOUR_ANALYTICS`、`TOUR_AD_HTML` 或其他影响 HTML shell 的内容后，必须在相关 service restart 后按“Production CDN 缓存策略”刷新对应 language hostname；不要仅根据源站验证或 `deploy-production.sh` 的 public HTTP 200 就判断新 release 已在全部 CDN 边缘节点生效。公开的 `/socket` 既有安全原则保持不变：production 不注册或开放本地 Socket transport，普通请求和 WebSocket Upgrade 均应保持 404。

### 广告职责、首次接入与最终验收边界

Auto Ads、课程页手动广告、Angular SPA mount/unmount 生命周期、局部 AdSense layout protection，以及覆盖这些行为的 browser tests，都是项目共享实现。新增 locale 只接入和使用这套既有能力；第三门及后续 locale 不重新设计广告位、不重复共享架构验证，也不把广告纳入 TranslationUnit 或 Locale-level language quality review。

新 locale 必须在**首次 production release 激活前**完成以下 production 广告接入准备：

- 确认目标 locale 的 production service 引用包含有效 `TOUR_AD_HTML` 的正确 `EnvironmentFile`；该文件可由 unit 本体或既有 drop-in 引入，无需为新 locale 额外创建 drop-in；
- 完成 Auto Ads 所需的 production 配置；
- 准备并验收课程广告 CSS/JS 的 production asset 来源（zh-CN 为同源；非中文 locale 按共享 assets 策略）；
- 非中文 locale 使用 shared-assets 时，在首次上线前确认共享的 `course-ad.css` 与 `course-ad.js` 已部署，且已完成缓存验收；这项广告专项检查不能替代完整 11 文件 current-state freshness gate，后者按“非中文共享静态资源第一版”和 shared-assets 发布状态机执行。

此阶段只准备 production 配置和资源，不要求证明尚未激活的 production 进程已实际读取变量、正式 HTML 已生成 Auto Ads head code，或浏览器已产生真实广告请求；这些都属于激活后的最终 production acceptance。因此首次正式部署启动后即为最终“已启用广告”形态，不允许先上线无广告版本、再以第二次上线接入广告。

首次 production 激活后，只在同一次最终 production acceptance 中完成轻量广告确认，并记录在现有 `data/locale-surface-reviews/<locale>/<review-id>.md` 的 `production verification result`：

- 实际 production HTML 已加载或生成预期的 AdSense loader / Auto Ads head 配置；
- 课程页存在手动广告 mount；
- 浏览器存在真实广告请求机会；广告可为 filled 或 unfilled，不以填充为通过条件；
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
/usr/local/nginx/sbin/nginx -t && service nginx reload
```

当前环境中 `service nginx configtest` 不可用；`service nginx reload` 会正常转发到 `systemctl reload nginx.service`。

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

## 新 Locale 首次生产部署

首次部署是建立新 locale 的 production 基础设施，不得直接套用日常 release 切换。开始前必须已完成 TranslationUnit promotion、完整 projection、Surface Review 的语言质量与 preview acceptance、production publish，以及 shared-assets current-state freshness gate。

正式入口只有 release 目录：

```sh
scripts/first-production.sh \
  /tmp/go-tour-release-YYYYMMDD-<locale>-<shortsha>
```

脚本从 `release.json.locale` 选择正式 identity，并要求其显式为 `production_state=first-production`；当前 `zh-CN`、`ja-JP`、`de-DE`、`fr-FR` 均为 `live`，不能用于 bootstrap。脚本不接受调用者重复传 hostname、port、service、data root、origin IP 或 zone。一个 run 为 aliyun 和 zgocloud 分别建立 invocation-scoped ControlMaster；正常、失败和 signal 退出均清理 control socket。状态 receipt 写在 release 同级的 `<release>.first-production-receipt.json`，只记录 run/stage/locale/hostname/release/time/result 等非 secret identity。若已完成 deploy 后本地连接中断，重跑同一命令会识别未完成 receipt，重新验证 `current`、service/source health 和全部关键 identity，再从后续幂等阶段继续；已完成 receipt 拒绝重复 bootstrap。

正式顺序固定为：

```text
全量无 mutation preflight
→ aliyun data root / systemd / DNS-01 TLS / Nginx bootstrap
→ ZgoCloud Playground Origin 精确幂等更新与接口/boundary 验收
→ scripts/deploy-production.sh <release-dir>
→ zgocloud --resolve direct-origin acceptance
→ Cloudflare proxied A record
→ zgocloud public machine acceptance
→ scripts/verify-production.sh <release-dir>
→ Chrome automated browser acceptance
→ 极小 HUMAN visual gate
→ evidence finalize
```

preflight 在任何 production mutation 前同时检查：正式 bundle 与 identity、TODO/unknown locale 间接由 publish/identity gate 拒绝、两台 SSH 和 root account、port/service/data-root/vhost/certificate 冲突、EnvironmentFile 与非空 `TOUR_AD_HTML`（不输出值）、Cloudflare secret 权限与变量、zone 唯一性、目标 DNS 无冲突、Playground 两个 location 的结构一致性，以及 shared-assets origin/public SHA-256 freshness。任一项失败时，不建立目录、unit、证书、vhost、DNS 或 Origin。

基础设施阶段复用已验证的 production service hardening 与 OneinStack Nginx/TLS 结构；所有 server name、proxy port、service、current、证书和私钥路径均从目标 locale 的正式 identity 渲染，不从 fr-FR 或 locale 名称复制值。service 从精确 `current` 启动，TLS 使用 Let's Encrypt/acme.sh `dns_cf`、`ec-256`，Nginx 只有精确 loopback proxy，不生成会截获 `/tour/static/` 的静态 location。证书通过 DNS-01 签发，不要求先建立 production A record；aliyun 的非交互 SSH 调用使用 `/usr/local/nginx/sbin/nginx -t`，通过后才 reload。已存在未知或不兼容配置时停止，不覆盖。失败边界保留为可审计、可 resume 的首次创建物：已经精确写入的 data root、unit 或 ACME/TLS 现场不做不确定删除，后续 preflight 必须重新逐项验证；本次新建 vhost 若 Nginx config test 或 reload 失败则移除并恢复原 Nginx 状态。任一失败均不会记录 infrastructure PASS，service enable/start/restart 失败也不能进入后续 stage。

Playground mutation 只接受当前已验证的“两处相同精确 Origin 正则 + 唯一 compile/fmt location”结构，保留全部既有 origin、避免重复，并在 `nginx -t`/reload 失败时恢复备份。随后验证新 origin 的 OPTIONS 204、POST 200、wrong Origin 403、GET 405。

FIRST_DEPLOYMENT 仍严格复用 `deploy-production.sh` 的 upload/current/health 状态机；编排器不实现第二套部署逻辑。源站连续健康后先从 zgocloud 用真实 hostname/SNI 和 `--resolve` 验收 HTTPS、HTTP redirect、关键 route、canonical、`html lang`、shared-assets 与 `/socket`。只有这一步通过才调用 Cloudflare API；已存在完全相同的 proxied A record 幂等通过，未知类型、IP、proxy 状态或重复记录一律拒绝。API failure 不回滚已健康的 origin。

Cloudflare Cache Rule 只做只读、低风险判断：能证明已启用的 cache-settings rule 以精确 host、可求值 wildcard 或 `ends_with` 匹配当前 hostname 时记录 `verified`；否则保留 `HUMAN_GATE`，不猜 rule ID、不修改 account-wide rule。

公网 DNS 生效后，zgocloud 完成关键 route、105 URL sitemap、canonical/locale、socket、cache header、shared asset 与 Playground 验收；随后仍调用现有 `verify-production.sh` 做正式 machine acceptance。首次编排器为该命令设置内部 `VERIFY_PRODUCTION_NETWORK_SSH=zgocloud`，verification 建立 invocation-scoped SSH/SOCKS ControlMaster，使全部 public curl 的 TCP/DNS 从 zgocloud 发出；远端 identity/source 检查仍直接访问 aliyun。普通维护者调用方式不变。`verify-production-browser.py` 使用仓库既有 `google-chrome`/DevTools 基线，自动覆盖 desktop/mobile、普通与 editor course、`/tour/moretypes/1`、语言列表、Run/Format/Reset、runtime output、Playground endpoint、canonical/lang/SEO、socket、shared assets、course-ad mount/loader/request opportunity 和 SPA 下一页。广告 filled/unfilled 均可通过。

全部自动验收 PASS 后，唯一人工 production gate 为：

1. Desktop 打开一个 editor 课程页，肉眼确认整体布局、editor 和广告区域无明显视觉异常。
2. Mobile 打开 `/tour/moretypes/1`，确认无非预期整页横向 overflow、广告/footer 无明显异常，并点击一次“下一页”确认 SPA 视觉正常。

人工记录只写 `passed` 或 `failed: <问题>`。人工不再重复 Run、Format、Reset、SEO、canonical、Network Origin 或 `/socket`。通过后，把 receipt、HUMAN visual 结果与 production URL 写回既有 Surface Review evidence，并把最终 `decision` 设为 `passed`。

成功摘要保持简洁，例如：

```text
[首次生产] 基础设施：PASS
[首次生产] Playground Origin：PASS
[首次生产] 源站验收：PASS
[首次生产] Cloudflare DNS：PASS
[首次生产] 公网验收：PASS
[首次生产] 浏览器验收：PASS
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

本节只适用于已经完成首次生产基线、且 `scripts/deploy-production.sh` 已存在对应 profile 的 locale。日常部署不重新分析 OneinStack、Nginx、acme.sh、data root、systemd 或端口；除非预检显示配置漂移，否则使用已验证 profile 发布新 release。

### Production release 自动部署

先使用仓库现有的 `publish` 命令生成并验收 Linux/amd64 production bundle。publish 环境必须存在 `google-chrome`；构建会对当前 locale 的全部正式课程 URL 执行 headless Chrome prerender，并将 route-specific HTML 写入 `_content/tour/prerender/<lesson>/<page>.html`。Chrome 缺失、任一页面未完成渲染或产物缺少 route metadata、正文、完整示例源码时均 fail closed。部署脚本不会自动构建 bundle，只接受一个已经生成的本地 release 目录。调用方式对所有语言相同：

```sh
# zh-CN
scripts/deploy-production.sh \
  /tmp/go-tour-release-20260813-zh-CN-a4d4dca

# ja-JP
scripts/deploy-production.sh \
  /tmp/go-tour-release-20260824-ja-JP-<shortsha>
```

脚本严格读取 `release.json` 的 `locale` 作为唯一事实来源，不接受 `--locale`，也不根据目录名猜测语言。当前正式 identity 包含 `zh-CN`、`ja-JP`、`de-DE` 和 `fr-FR`；不支持、重复、冲突或 schema 不合法的 locale 会在 SSH、上传、远端加锁及任何生产修改之前 fail closed。下表只是便于阅读的当前快照，权威来源是 `production/identity.json`：

| locale | releases | current | deploy lock | systemd service | localhost health | public acceptance |
| --- | --- | --- | --- | --- | --- | --- |
| `zh-CN` | `/data/go-tour/releases` | `/data/go-tour/current` | `/data/go-tour/.deploy.lock` | `go-tour.service` | `http://127.0.0.1:3999/` | `https://go-dev.shuijingwanwq.com/` |
| `ja-JP` | `/data/go-tour-ja-JP/releases` | `/data/go-tour-ja-JP/current` | `/data/go-tour-ja-JP/.deploy.lock` | `go-tour-ja-JP.service` | `http://127.0.0.1:4000/` | `https://ja-go-dev.shuijingwanwq.com/` |
| `de-DE` | `/data/go-tour-de-DE/releases` | `/data/go-tour-de-DE/current` | `/data/go-tour-de-DE/.deploy.lock` | `go-tour-de-DE.service` | `http://127.0.0.1:4001/` | `https://de-go-dev.shuijingwanwq.com/` |
| `fr-FR` | `/data/go-tour-fr-FR/releases` | `/data/go-tour-fr-FR/current` | `/data/go-tour-fr-FR/.deploy.lock` | `go-tour-fr-FR.service` | `http://127.0.0.1:4002/` | `https://fr-go-dev.shuijingwanwq.com/` |

四个 profile 的 service user 均为 `go-tour`。

本地目录名应遵循 `go-tour-release-YYYYMMDD-<locale>-<shortsha>` 约定，并且必须以 `go-tour-release-` 开头。脚本只删除这个固定前缀，并对剩余名称执行安全字符检查；因此上例对应的远端目录为：

```text
/data/go-tour/releases/20260813-zh-CN-a4d4dca
```

脚本固定使用 SSH 别名 `aliyun`。当前生产运维账号为 root；远端 `id -u` 不是 `0` 时会在上传前失败，不使用或依赖 `sudo`。部署过程如下：

一次 `deploy-production.sh` 调用会为 aliyun 建立 invocation-scoped SSH ControlMaster，后续 preflight、rsync、remote validation 与 activation 复用同一连接；使用 BatchMode、连接/keepalive/retry 边界，并在主流程退出时清理 ControlMaster。SSH 中断后状态不确定时仍保留既有 lock/evidence，不能因 multiplex 自动盲目重试 mutation。

1. 本地严格检查 bundle 根结构、symlink、`bin/tour`、`release.json`、`site-metadata.json` 和 `SHA256SUMS`；manifest 必须满足 production 约束，其 locale 必须与由同一 `release.json` 选出的 profile 一致。
2. 远端在所选 profile 的 data root 中原子创建 `.deploy.lock` 防止并发部署，并验证 `current`、当前 release、目标名称和对应 systemd service。首次 deployment 仅在 `current` 完全不存在时允许继续；已有 locale deployment 则要求其为指向 release root 内当前 release 的合法 symlink。`current` 存在但不是 symlink、或指向 release root 外时均 fail closed。同名 release 已存在时拒绝覆盖；锁已存在表示可能有正在执行或上一次未完成的部署，脚本直接停止，不分析或自动删除该锁。
3. `rsync` 只上传到所选 profile 的 `releases/.<release>.staging-<token>`，不直接写最终 release 或 `current`，也不使用 `--delete` 覆盖 release。
4. 上传后无条件执行权限归一化：owner/group 为 `root:root`，所有目录为 `0755`，普通文件为 `0644`，`bin/tour` 为 `0755`。随后在远端重新验证 SHA-256，以及 `go-tour` 用户对二进制和必要内容的访问权限；production manifest 已在本地严格检查，第一版不在远端重复解析。
5. staging 在同一文件系统内原子重命名为最终 release；脚本创建临时 symlink 后以原子 `mv` 替换 profile 的 `current`，再 restart 对应 service。
6. 新版本只有在连续 3 次同时满足 profile service 为 `active`、profile localhost health URL 严格返回 HTTP 200 后才算健康。检查最多 12 轮，每轮间隔 3 秒；任何失败都会把连续计数归零，不能用瞬时一次 `active` 判断成功。
7. 只有在已有 locale 的 `current` 已明确切换到新 release 后，restart 失败或新版本健康检查明确失败，脚本才会自动回滚：`current` 原子切回旧 release、restart 服务，并使用相同的连续 3 次规则验证旧 release。失败的新 release 会保留用于诊断，不自动删除历史 release。首次 deployment 没有旧 release；若首次切换后的 health failure，脚本明确报告 `FIRST_DEPLOYMENT` failure、保留 `current`、lock 与现场并给出人工检查提示，不伪造 rollback。

脚本只区分三类主要结果：在 `current` 原子替换开始前，若脚本能够明确安全清理本次部署资源，会清理 staging/final、临时 symlink 和锁，`current` 保持不变；若远端状态与预期不一致，则保留现场并要求人工检查；新版本明确失败且旧版本恢复健康时会报告已回滚；激活 SSH 中断、current 切换状态无法确认、回滚失败或其他远端状态不确定时会保留 deployment lock 和现场，停止自动处理。遇到最后一种情况不要直接重复部署，应先按日志中所选 profile 的参数人工检查。以下为 `zh-CN` 示例；`ja-JP` 应替换为 `/data/go-tour-ja-JP/current`、`go-tour-ja-JP.service` 和 `http://127.0.0.1:4000/`，`de-DE` 应替换为 `/data/go-tour-de-DE/current`、`go-tour-de-DE.service` 和 `http://127.0.0.1:4001/`：

```sh
ssh aliyun 'readlink -f /data/go-tour/current'
ssh aliyun 'systemctl status go-tour.service --no-pager -l'
ssh aliyun 'curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3999/'
ssh aliyun 'journalctl -u go-tour.service -n 80 --no-pager'
```

localhost 连续健康后，脚本才检查对应 profile 的 public URL。正式域名异常属于 CDN、HTTPS、Nginx 或其他外部验收问题，不会自动回滚一个已经稳定健康的源站 release。脚本不调用 CDN API，也不自动清理缓存；language release 成功后必须按“Production CDN 缓存策略”主动刷新对应 hostname，并完成后续 CDN 验收。public HTTP 200 只证明公网入口可用，不能替代 hostname purge 或证明新 release 已在全部边缘节点生效。

`FIRST_DEPLOYMENT` 是例外：连续 localhost health 通过后脚本停止于源站 ready，不要求尚未启用 DNS 的 public URL，也不做无旧 release/cache 可刷新的 hostname purge。下一步必须先从外部主机使用 production hostname + `--resolve <hostname>:443:<origin-ip>` 完成 TLS/SNI、HTTP → HTTPS 和关键 route 的 direct-origin acceptance；通过后再创建/启用 `proxied=true` 的正式 DNS，并执行 public machine/browser acceptance。`EXISTING_DEPLOYMENT` 继续保持 `deploy → hostname purge → verify`。

### Production machine acceptance

已有 locale 日常部署的正式顺序固定为：

```text
scripts/deploy-production.sh <release-dir>
→ EdgeOne / Cloudflare hostname purge HUMAN GATE
→ scripts/verify-production.sh <release-dir>
→ browser acceptance HUMAN GATE
```

`scripts/verify-production.sh` 从 release 目录的 `release.json` 读取 locale，并以与部署脚本一致的 fail-closed profile 选择 releases/current/lock、service、loopback origin、production hostname 和 CDN header；当前支持 `zh-CN`、`ja-JP`、`de-DE`、`fr-FR`。调用者不得另外传 hostname、port、service 或 remote release name：

```sh
scripts/verify-production.sh \
  /tmp/go-tour-release-YYYYMMDD-<locale>-<shortsha>
```

脚本只执行只读、机器可确定的验收：本地 release identity；远端 `current` 精确指向目标 release、service active、deployment lock 不存在；7 条 source 与 public 关键路由严格 HTTP 200；首页和 welcome 页精确 `html lang` 与 canonical；公网 sitemap 恰好包含 105 个 HTTPS、正确 production hostname、无重复且逐 URL HTTP 200 的 URL；普通及 WebSocket Upgrade `/socket` 均为 404。hostname purge 后，对首页和 welcome 页各请求 3 次并记录 cache status：`ja-JP`、`de-DE`、`fr-FR` 读取 `CF-Cache-Status`，`zh-CN` 读取 `EO-Cache-Status`。每次都必须为 HTTP 200 且 header 存在；`MISS`、`HIT`、`EXPIRED`、`REVALIDATED`、`UPDATING`、`STALE` 均为允许的 observation，顺序不受限制，3 次全为 `MISS` 仍通过并注明 cache 尚未 warm。`BYPASS`、`DYNAMIC`、header 缺失或其他未允许状态表示请求没有进入预期 cache eligibility 路径，必须 fail closed。任一检查失败均输出 stage、URL/check、expected、actual 并以非零状态结束；全部通过时输出 `PRODUCTION MACHINE ACCEPTANCE: PASS`。

脚本不调用 EdgeOne 或 Cloudflare API，不搜索 token、不执行 purge、不修改 DNS/Cache Rule。hostname purge 是运行脚本前的 **HUMAN GATE**；machine verification 只观察 purge 后实际返回的三个 cache status，不要求第一次为 `MISS`、后续为 `HIT`，也不要求固定次数内出现 `HIT`。

curl machine acceptance 后可运行 `scripts/verify-production-browser.py <production-public-url> <locale>` 完成 rendered/interaction 自动验收；首次 production 已由 `first-production.sh` 自动调用。桌面/移动端、语言列表、Run / Format / Reset、runtime message、Playground endpoint、轻量广告 request opportunity、SPA 和长代码 overflow 均由 Chrome 检查。机器通过后只保留本手册“新 Locale 首次生产部署”定义的极小 visual HUMAN gate；不要人工重复机器项目。

## 非中文共享静态资源第一版

所有非中文社区语言的 production 页面使用：

```text
https://assets-go-dev.shuijingwanwq.com/
```

zh-CN development 和 production 的既有站点静态资源继续使用 language origin；所有 locale 的 development/preview 也保留完整本地副本。课程页广告自有 CSS/JS 也通过模板 `asset` helper 选择：zh-CN 使用当前 release 的同源路径，非中文 production locale 可使用下列 shared-assets 路径。当前共享清单为：

```text
tour/static/css/app.css
tour/static/go-dev/course-ad.css
tour/static/go-dev/course-ad.js
tour/static/lib/codemirror/lib/codemirror.css
tour/static/img/gopher.png
images/site-logo.png
images/site-logo-32.png
images/go-logo-white.svg
images/icons/brightness_6_gm_grey_24dp.svg
images/icons/brightness_2_gm_grey_24dp.svg
images/icons/light_mode_gm_grey_24dp.svg
```

使用以下命令把清单导出为可直接作为普通服务器静态目录的 origin tree：

```sh
go run -mod=readonly ./cmd/tour-i18n assets export \
  --output /tmp/go-tour-shared-assets

go run -mod=readonly ./cmd/tour-i18n assets validate \
  --input /tmp/go-tour-shared-assets
```

目标目录必须不存在。命令通过同级 staging 目录构建，逐字节复制固定 allowlist，生成 `SHA256SUMS`，验证文件集合与校验和后再原子重命名为目标目录。`SHA256SUMS` 只用于部署前后完整性验证，不参与 URL、cache key 或版本选择。不要把完整 `_content` 同步到 assets origin。

共享资产固定使用原逻辑路径，不使用 assets-release-id、content-hash URL、query version、独立 versioned assets release，也不升级 language `release.json`。`/tour/script.js` 明确不拆分、不共享，继续由 language origin 动态拼接并提供。Angular partial、`tree.png`、lesson/footer、HTML、locale 内容、metadata、analytics 和 Playground endpoint 均继续由 language origin 提供；课程页广告自有 CSS/JS 同样由 `asset` helper 按 locale 选择，zh-CN 不依赖 assets origin。未来真实广告所需的 Google 官方脚本仍须直接从 Google 加载，不得代理到 assets origin。Google Fonts 继续使用现有外部 Inconsolata CSS。所有 language bundle 仍保留完整本地静态资源副本，供 projection、preview、rollback 和 CDN 故障排查使用。

`assets-go-dev.shuijingwanwq.com` 已正式部署，Cloudflare 已代理；源站为 `121.40.248.29`，origin root 为 `/data/wwwroot/assets-go-dev.shuijingwanwq.com`，Nginx vhost 为 `/usr/local/nginx/conf/vhost/assets-go-dev.shuijingwanwq.com.conf`。TLS 使用 Let's Encrypt / acme.sh / `dns_cf`，证书和私钥分别位于 `/usr/local/nginx/conf/ssl/assets-go-dev.shuijingwanwq.com.crt` 与 `/usr/local/nginx/conf/ssl/assets-go-dev.shuijingwanwq.com.key`；HTTP 80 永久跳转 HTTPS。

Cloudflare Edge Cache TTL 为 1 个月。项目不主动覆盖 Browser Cache TTL，不给这些固定 URL 设置 `immutable` 或一年浏览器缓存；使用 Cloudflare/origin 默认或 Respect Existing Headers。当前正式 allowlist 是完整 11 文件，不只 `course-ad.css` 与 `course-ad.js`；后两者已包含课程页 AdSense integration。文档中“11/11 已正式部署”记录的是一次历史 production acceptance，不是当前仓库状态的永久保证。判断当前 production 是否最新，必须以当前仓库执行正式 `assets export` / `assets validate` 所代表的完整 allowlist 内容为准，并运行下述 `scripts/deploy-shared-assets.sh` 由脚本比较 current export 与 production origin；不得依赖历史记录、Git 历史、上次部署时间或人工判断文件是否变化。任一 allowlist 文件都属于此完整比较范围，包括但不限于 `app.css`、`course-ad.css` 与 `course-ad.js`。脚本输出 changed URLs 时，必须走既有 changed URL Custom Purge 与 public validation；输出 `NO CHANGES` 时不执行 Custom Purge，但仍须完成完整 11/11 公网 SHA-256 与当前 export 的对照。不引入 assets version、query version 或 content-hash URL，也不改变 shared-assets 架构。历史首次 11 文件 production acceptance 中，对实际变化的 `SHA256SUMS`、`course-ad.css`、`course-ad.js` 已完成 Custom Purge 后的 `MISS → HIT` 验收，公网内容 SHA-256 为 11/11 一致，三个非 allowlist boundary 路径继续返回 404。

### Shared-assets production 发布状态机

以下阶段 A、B、C、E、F、G 是标准终端步骤；维护者可以直接执行，也可以委托具备终端访问能力的工具执行。阶段 D 是唯一的人工 UI 门。确定 current-state freshness 时，必须依次完成阶段 A、运行阶段 B 的 `scripts/deploy-shared-assets.sh` 比较 current export 与 production origin，并完成阶段 F 的完整 11/11 公网校验；只有脚本输出 changed URLs 时才执行阶段 C、D、E 的精确 Custom Purge 与缓存验收。任一阶段不满足验收条件时停止，不跳过、不把后续阶段的结果用于掩盖当前失败。

#### 阶段 A：本地生成

目标目录必须不存在。在仓库根目录执行：

```sh
go run -mod=readonly ./cmd/tour-i18n assets export \
  --output /tmp/go-tour-shared-assets

go run -mod=readonly ./cmd/tour-i18n assets validate \
  --input /tmp/go-tour-shared-assets
```

核对导出恰好包含 `SHA256SUMS` 和上述 11 个 allowlist 文件，不包含完整 `_content`、symlink 或其他文件，并执行：

```sh
cd /tmp/go-tour-shared-assets
find . -type f -printf '%P\n' | sort
test "$(find . -type f ! -name SHA256SUMS | wc -l)" -eq 11
sha256sum -c --strict SHA256SUMS
```

只有文件集合正确且 11/11 校验通过，才能进入阶段 B。

#### 阶段 B：production origin 更新

production 目标是完整的 shared-assets export tree：

```text
/data/wwwroot/assets-go-dev.shuijingwanwq.com
```

正式部署接口为：

```sh
scripts/deploy-shared-assets.sh \
  /tmp/go-tour-shared-assets
```

脚本不运行 assets export；它只接受阶段 A 已生成的目录。脚本先调用仓库的 `assets validate` 复用唯一 Go allowlist，要求输入逐字节等于当前仓库正式资源，并验证顶层真实目录、无 symlink/unsupported entry、`SHA256SUMS` 的安全路径、文件集合与 11/11 SHA-256。用户自行制作一份 checksum 文件不能绕过正式 allowlist 与仓库 source 校验。

固定 production profile 为 SSH alias `aliyun`、origin `/data/wwwroot/assets-go-dev.shuijingwanwq.com`、lock `/data/wwwroot/.assets-go-dev.deploy.lock`。脚本使用唯一 token 在 `/data/wwwroot/` 建立非公开 `.assets-go-dev.staging-*`，上传后在远端重新校验文件集合、SHA-256 和权限；它从当前 origin 读取并要求统一的 owner/group、目录 mode 与普通文件 mode，不修改 `/data/wwwroot/` 中其他站点的权限。

如果 staging 与 origin 逐文件相同，脚本输出 `NO CHANGES`，清理 staging/lock，不创建 backup，也不产生 purge URL；这证明 origin 已与当前 export 一致，但不替代阶段 F 的完整 11/11 公网 SHA-256 对照。如果有变化，脚本在第一次修改 origin 前创建并验证完整非公开备份 `/data/wwwroot/assets-go-dev.shuijingwanwq.com.bak.<token>`，随后才在服务器端把完整 staging tree 以受限 `rsync --delete` 同步进固定 origin。delete 只作用于该精确 origin 内，确保新增、修改、删除后 production 文件集合与正式 export 完全一致；历史 backup 不自动删除。

preflight、lock、upload、staging validation 或 backup 阶段失败时，origin 不变，脚本只在能确认安全时清理本次 staging/lock。production mutation 开始后若同步或严格验证失败，脚本使用刚创建的完整 backup 恢复并重新验证；回滚明确成功时报告部署失败但旧内容已恢复。如果回滚失败、SSH 中断或状态无法确认，脚本保留 lock、staging、backup 与现场，输出只读人工检查命令，禁止直接自动重试。INT、TERM、HUP 在 mutation 前按安全边界清理，mutation 后保留 evidence。

脚本成功后生成与当前正式 export 的绝对路径和 `SHA256SUMS` identity 绑定的 machine-readable verification receipt，并打印 `verification receipt: <path>` 与唯一后续命令 `scripts/verify-shared-assets-production.sh <receipt>`。receipt 还包含 deployment result（`NO_CHANGES` 或 `DEPLOYED`）、实际 changed logical paths、固定 production base URL 和固定 boundary 路径；不包含 secret。它按稳定顺序输出 added、modified、deleted 的实际固定 URL（`SHA256SUMS` 如发生变化也属于 changed URL）。`DEPLOYED` 时脚本仍在 Cloudflare HUMAN GATE 前结束，不调用 Cloudflare API/CLI，也不在 purge 前执行公网 MISS/HIT 验收；`NO_CHANGES` 时也生成 receipt，直接执行其验证命令。

receipt verification 的全部 shared-assets 公网请求使用正式 `production/identity.json` 中 `shared.zgocloud_ssh_alias` 作为 network runner。验证脚本为单次 invocation 建立一个 SSH ControlMaster 与 SOCKS tunnel，所有 `curl` 通过 `--socks5-hostname` 复用该 tunnel，使 DNS 与 TCP 均从 zgocloud 网络出口发起；runner 无法建立即 fail closed，不回退到维护者本机网络，正常、失败和 signal 退出均清理该连接。Cloudflare 短时 HTTP `522` / `525` 仅在单个逻辑请求内最多重试三次；其他 HTTP 状态、内容 SHA-256、cache semantics、boundary 与 receipt identity 错误均立即 fail closed。该网络瞬态容忍不降低完整 11/11 freshness gate。

`deploy-shared-assets.sh` 已通过本地 mock 自动化测试，并已完成首次真实 11 文件 production deployment 验证。`SHA256SUMS`、`course-ad.css` 与 `course-ad.js` 的实际 changed URLs 已完成精确 Custom Purge 与 `MISS → HIT` 验收；公网 allowlist SHA-256 为 11/11 一致，三个非 allowlist boundary 路径继续返回 404。后续发布仍以脚本输出的实际 changed URLs 为唯一 purge 清单；如果真实权限或工具基线与预检不符，立即停止，不绕过检查。

origin 更新成功后，脚本已经执行文件集合、SHA-256、与 staging 一致性、无 symlink/unsupported entry 和权限验证。需要人工复核时可执行以下只读命令：

```sh
ssh aliyun '
  set -eu
  cd /data/wwwroot/assets-go-dev.shuijingwanwq.com
  find . -type f -printf "%P\n" | sort
  test "$(find . -type f ! -name SHA256SUMS | wc -l)" -eq 11
  sha256sum -c --strict SHA256SUMS
  find . -printf "%u:%g %m %P\n" | sort
'
```

文件集合必须只有 `SHA256SUMS` 与 11 个 allowlist 文件，SHA-256 必须为 11/11，权限必须符合脚本从部署前 origin 读取并保持的模型，并且不得存在额外可公开文件。任一条件不满足都停止，不进入缓存刷新。

#### 阶段 C：确定 purge URL

部署脚本比较更新前 origin 与已验证 staging，只列出内容实际新增、修改或删除的固定 URL；不要默认刷新全部 11 个 URL，也不要默认使用 Purge Everything。此前首次课程广告资源部署的 changed URLs 包括：

```text
https://assets-go-dev.shuijingwanwq.com/tour/static/go-dev/course-ad.css
https://assets-go-dev.shuijingwanwq.com/tour/static/go-dev/course-ad.js
```

如果脚本输出其他变化 URL，把它们一并加入清单；删除路径的旧 URL 也必须刷新。脚本输出完整清单后到达阶段 D 并结束，不继续公网缓存验收。

#### 阶段 D：HUMAN GATE — Cloudflare Dashboard Custom Purge

维护者在 Cloudflare Dashboard 中选择 `assets-go-dev.shuijingwanwq.com` 所属 zone，进入缓存管理的 Purge Cache / Custom Purge 功能，按 URL 刷新阶段 C 的精确清单。Cloudflare UI 文案可能变化，这里只定义操作目标，不把 UI 文案当作程序接口。

当前 shared-assets production 发布不要求自动化 Cloudflare purge。不得为此搜索本地或服务器上的 Cloudflare Token、要求维护者提供 API Token、把凭据写入仓库或 shell history、猜测既有凭据入口、调用 Cloudflare API、使用 Wrangler 自动刷新或安装新的 Cloudflare CLI。未来如需自动化，应作为独立受控改进处理。

Dashboard purge 是正常 human gate，不是 deployment failure，也不是缺少 API 权限错误。维护者明确确认 Custom Purge 已完成后，运行 deployment 输出的唯一后续命令 `scripts/verify-shared-assets-production.sh <receipt>`；确认前不得把 MISS/HIT 不符合预期解释为新资源发布失败。

#### 阶段 E：公网缓存验收

由 receipt 指定的 `scripts/verify-shared-assets-production.sh <receipt>` 自动对阶段 C 的每个实际 changed URL 连续请求两次、要求 HTTP 200，并按当前基线验证 `CF-Cache-Status: MISS → HIT`，逐 URL 输出明确结果。Cloudflare 实际状态不符合基线时脚本 fail closed；不要手工复制 URL 或猜测、伪造 MISS/HIT 结论。`NO_CHANGES` 没有 changed URL，脚本明确输出 `SKIP CACHE PURGE VERIFICATION: NO CHANGES`，不要求 MISS → HIT。

#### 阶段 F：完整性验收

同一验证脚本自动请求正式 11 个 allowlist URL，要求 HTTP 成功，并逐一将公网内容 SHA-256 与 receipt 绑定的当前 export `SHA256SUMS` 对照。必须达到 11/11 内容一致；只验证本次 purge 的文件不能替代完整 allowlist 验收。receipt 与当前 export identity 不符、export 不再通过正式 `assets validate`，或任一公网 SHA-256 不符时，脚本 fail closed，必须重新走正式 shared-assets 流程。receipt 可以保留作为本次 execution evidence，但不能作为未来 current-state freshness gate 的替代。

#### 阶段 G：边界验收

同一验证脚本还自动确认以下固定非 allowlist 路径全部返回 HTTP 404：

```text
/tour/script.js
/tour/static/img/tree.png
/tour/static/partials/editor.html
```

任一路径意外返回共享内容，shared-assets 发布验收失败。固定 URL、Cloudflare Edge Cache TTL 与 Browser Cache TTL 策略保持不变；不引入 hash filename、version directory、query version、loader、manifest 或 assets release ID。

#### 已知网络现象（2026-09-02）

ko-KR 首次 production 前的 shared-assets current-state freshness verification 发现公网波动。维护者本机曾随机收到 HTTP `525`，因此正式 verifier 已固定使用 zgocloud network runner；但 zgocloud 路径仍观察到随机 HTTP `522`、`525`，以及无 HTTP response 的 curl exit `28`（15 秒、0 bytes received）timeout。

对照只读测试中，zgocloud → Cloudflare public、zgocloud → Aliyun direct origin、Aliyun localhost → Nginx 各为 `20/20` HTTP 200；Nginx active、assets vhost TLS 与 direct-origin certificate verification 正常，未见持续性 Nginx/TLS 服务故障 evidence。50 次强制 Cloudflare MISS probe 有 `47` 次 HTTP 200 + `CF-Cache-Status: MISS`，均精确对应 Aliyun Nginx access log 的 `47` 个 HTTP 200；其余 `3` 次 curl HTTP `000` 未进入该 access log。30 次保留 stderr 的 transport probe 有 `18` 次 HTTP 200/MISS、`12` 次上述 timeout。

这表明 zgocloud → Cloudflare / Cloudflare 回源链路存在间歇性网络波动，但不足以推定 Nginx/TLS 持续故障、真实海外用户固定失败率或共享资产架构需要重设计。shared-assets 的长 Edge Cache TTL 会减少正常用户接触回源链路的机会，但不消除此现象。当前将其作为已知 production 网络风险：保留 zgocloud runner 与仅 HTTP `522` / `525` 的三次 bounded retry，不修改 Cloudflare、Nginx/TLS 或服务器网络参数，也不放宽 receipt、11/11 SHA-256、cache 或 boundary gate；若真实用户错误、production acceptance failure 或监控 evidence 显示持续/扩大，再作为独立基础设施问题调查。已知网络现象不能替代正式 shared-assets freshness verification `PASS`。


Google 官方 `adsbygoogle.js` 继续直接从 Google 域名加载，不下载、代理、镜像或 self-host 到 assets origin。

旧 `assets.go-dev.shuijingwanwq.com` 已废弃并清理，不提供兼容或迁移。不要在 zh-CN 的现有 EdgeOne 发布流程中加入 assets 域名依赖。

## ja-JP production 验收状态

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
