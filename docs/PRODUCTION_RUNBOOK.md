# 生产运维手册

本文档记录当前已验证的生产维护入口与操作边界。项目进度和部署历史见 [PROJECT_STATE.md](PROJECT_STATE.md)。

## 当前生产架构

- 正式站点：<https://go-dev.shuijingwanwq.com/>；A Tour of Go 使用 `/tour/` 路径。
- 请求链路：Cloudflare 权威 DNS → 腾讯云 EdgeOne → 源站 `121.40.248.29:443` → Nginx → `127.0.0.1:3999`。
- EdgeOne 到源站使用 HTTPS，回源 Host 为 `go-dev.shuijingwanwq.com`。
- Go 生产服务为 `go-tour.service`，监听 `127.0.0.1:3999`，release 根目录为 `/data/go-tour/`。

公网域名迁移不改变 `go-tour.service`、`/data/go-tour/`、仓库名或 Go module path。

## 生产统计配置

Go Tour 统一通过 `TOUR_ANALYTICS` 注入生产统计代码。生产启动路径最终将该环境变量作为 `analyticsHTML` 传给模板的 `{{.AnalyticsHTML}}`；首页 `/` 与 `/tour/...` 页面共用同一个入口。生产环境可在同一个变量中同时放置 Google Analytics 4 和百度统计两套完整 HTML / JavaScript 代码：

```text
TOUR_ANALYTICS='<Google Analytics HTML><Baidu Analytics HTML>'
```

本地开发默认不设置该变量，因此不会加载生产统计代码。实际统计代码以及具体统计 ID 不写入 Git 仓库；公开前端标识也不在本手册中固定记录。

Google AdSense 使用独立的 `TOUR_ADSENSE_CLIENT`。变量为空时，首页和课程页面完全不注入广告代码，自行部署默认关闭；生产环境显式设置后，服务端会校验 `ca-pub-...` 格式，并在每个完整 HTML 页面的 `<head>` 中生成一次 Auto Ads 站点代码，不插入手工广告位。当前从公开博客首页核验到的配置为：

```text
TOUR_ADSENSE_CLIENT='ca-pub-8392190980622725'
```

正式服务通过 systemd 环境文件读取统计配置：

```text
/etc/go-tour/go-tour.env
```

该文件权限应为 `600`，所有者为 `root:root`。`go-tour.service` 使用以下 drop-in 引入它：

```text
/etc/systemd/system/go-tour.service.d/analytics.conf
```

内容为：

```ini
[Service]
EnvironmentFile=/etc/go-tour/go-tour.env
```

新增或修改 systemd drop-in 后执行 `systemctl daemon-reload`；仅修改 `go-tour.env` 内容时，也必须重启 `go-tour.service`，使新进程重新读取统计环境变量。不要在 shell 历史、发布包或其他仓库文件中复制完整统计代码。

修改 `TOUR_ANALYTICS`、`TOUR_ADSENSE_CLIENT` 或其他影响 HTML shell 的内容后，EdgeOne 可能继续命中旧 HTML。若公网响应仍显示旧页面，应刷新 `go-dev.shuijingwanwq.com` 的 Hostname 级缓存；不要仅根据源站验证就判断公网已更新。公开的 `/socket` 既有安全原则保持不变：production 不注册或开放本地 Socket transport，普通请求和 WebSocket Upgrade 均应保持 404。

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
nginx -t && service nginx reload
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

本次 go-dev 使用 Let's Encrypt、HTTP → HTTPS、`ec-256` 和 Cloudflare DNS provider（`cf` / `dns_cf`），反向代理为 `http://127.0.0.1:3999`。OneinStack 内部使用 acme.sh；正常情况下优先通过 `vhost.sh` 流程创建或管理证书。与本项目相关的 acme.sh 证书管理记录中，当前只应保留 `go-dev.shuijingwanwq.com`；旧 `go-tour.shuijingwanwq.com` 的管理记录和残留目录已清理。

Cloudflare API Token 及其他密钥属于敏感凭据，不写入文档、不提交仓库，也不记录真实值。

## Production release 自动部署

先使用仓库现有的 `publish` 命令生成并验收 Linux/amd64、`zh-CN` production bundle。部署脚本不会自动构建 bundle，只接受一个已经生成的本地 release 目录：

```sh
scripts/deploy-production.sh \
  /tmp/go-tour-release-20260813-zh-CN-a4d4dca
```

本地目录名应遵循 `go-tour-release-YYYYMMDD-<locale>-<shortsha>` 约定，并且必须以 `go-tour-release-` 开头。脚本只删除这个固定前缀，并对剩余名称执行安全字符检查；因此上例对应的远端目录为：

```text
/data/go-tour/releases/20260813-zh-CN-a4d4dca
```

脚本固定使用 SSH 别名 `aliyun`。当前生产运维账号为 root；远端 `id -u` 不是 `0` 时会在上传前失败，不使用或依赖 `sudo`。部署过程如下：

1. 本地严格检查 bundle 根结构、symlink、`bin/tour`、`release.json`、`site-metadata.json` 和 `SHA256SUMS`；manifest 必须满足当前 `zh-CN` production 约束。
2. 远端通过原子创建 `/data/go-tour/.deploy.lock` 防止并发部署，并验证 `current`、当前 release、目标名称和 `go-tour.service`。同名 release 已存在时拒绝覆盖；锁已存在表示可能有正在执行或上一次未完成的部署，脚本直接停止，不分析或自动删除该锁。
3. `rsync` 只上传到 `/data/go-tour/releases/.<release>.staging-<token>`，不直接写最终 release 或 `current`，也不使用 `--delete` 覆盖 release。
4. 上传后无条件执行权限归一化：owner/group 为 `root:root`，所有目录为 `0755`，普通文件为 `0644`，`bin/tour` 为 `0755`。随后在远端重新验证 SHA-256，以及 `go-tour` 用户对二进制和必要内容的访问权限；production manifest 已在本地严格检查，第一版不在远端重复解析。
5. staging 在同一文件系统内原子重命名为最终 release；脚本创建临时 symlink 后以原子 `mv` 替换 `/data/go-tour/current`，再 restart `go-tour.service`。
6. 新版本只有在连续 3 次同时满足 `go-tour.service` 为 `active`、`http://127.0.0.1:3999/` 严格返回 HTTP 200 后才算健康。检查最多 12 轮，每轮间隔 3 秒；任何失败都会把连续计数归零，不能用瞬时一次 `active` 判断成功。
7. 只有在 `current` 已明确切换到新 release 后，restart 失败或新版本健康检查明确失败，脚本才会自动回滚：`current` 原子切回旧 release、restart 服务，并使用相同的连续 3 次规则验证旧 release。失败的新 release 会保留用于诊断，不自动删除历史 release。

脚本只区分三类主要结果：在 `current` 原子替换开始前，若脚本能够明确安全清理本次部署资源，会清理 staging/final、临时 symlink 和锁，`current` 保持不变；若远端状态与预期不一致，则保留现场并要求人工检查；新版本明确失败且旧版本恢复健康时会报告已回滚；激活 SSH 中断、current 切换状态无法确认、回滚失败或其他远端状态不确定时会保留 deployment lock 和现场，停止自动处理。遇到最后一种情况不要直接重复部署，应先人工检查：

```sh
ssh aliyun 'readlink -f /data/go-tour/current'
ssh aliyun 'systemctl status go-tour.service --no-pager -l'
ssh aliyun 'curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3999/'
ssh aliyun 'journalctl -u go-tour.service -n 80 --no-pager'
```

localhost 连续健康后，脚本才检查 <https://go-dev.shuijingwanwq.com/>。正式域名异常属于 CDN、HTTPS、Nginx 或其他外部验收问题，不会自动回滚一个已经稳定健康的源站 release。脚本不调用 EdgeOne API，也不自动清理缓存；若 HTML 或静态资源仍显示旧版本，应检查 `EO-Cache-Status`、`Age` 并按需人工刷新 EdgeOne Hostname 缓存。

## 非中文共享静态资源第一版

所有非中文社区语言的 production 页面计划使用：

```text
https://assets.go-dev.shuijingwanwq.com/
```

zh-CN development 和 production 继续使用 language origin 的本地静态资源；所有 locale 的 development/preview 也始终使用本地资源，不依赖公网 CDN。第一版共享清单只有：

```text
tour/static/css/app.css
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
```

目标目录必须不存在。命令通过同级 staging 目录构建，逐字节复制固定 allowlist，生成 `SHA256SUMS`，验证文件集合与校验和后再原子重命名为目标目录。`SHA256SUMS` 只用于部署前后完整性验证，不参与 URL、cache key 或版本选择。不要把完整 `_content` 同步到 assets origin。

第一版固定使用原逻辑路径，不使用 assets-release-id、content-hash URL、独立 versioned assets release，也不升级 language `release.json`。`/tour/script.js` 明确不拆分、不共享，继续由 language origin 动态拼接并提供。Angular partial、`tree.png`、lesson/footer、HTML、locale 内容、metadata、analytics/ads 和 Playground endpoint 均继续由 language origin 提供；Google Fonts 继续使用现有外部 Inconsolata CSS。所有 language bundle 仍保持完整自包含，并保留全部本地静态资源作为 preview、rollback 和 CDN 故障排查能力。

Cloudflare 后续计划使用 28 天 Edge Cache TTL。项目不主动覆盖 Browser Cache TTL，不给这些固定 URL 设置 `immutable` 或一年浏览器缓存；使用 Cloudflare/origin 默认行为或 Respect Existing Headers。每次共享文件发生变化后必须按顺序执行：

1. 更新普通服务器上的 assets origin 文件；
2. purge Cloudflare 对应缓存；
3. 请求资源并确认首次为 MISS；
4. 再次请求并确认 HIT；
5. 对照 `SHA256SUMS` 核对公网资源内容。

当前仓库只具备导出与引用能力；尚未创建 assets DNS、TLS、Nginx origin、Cloudflare Cache Rule 或实际部署。不要把以上规划描述为已上线状态，也不要在 zh-CN 的现有 EdgeOne发布流程中加入 assets域名依赖。

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
