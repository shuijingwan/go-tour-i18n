# 生产运维手册

本文档记录当前已验证的 production 基线，并严格区分“新 locale 首次生产部署”和“已有 locale 日常维护部署”。项目进度和部署历史见 [PROJECT_STATE.md](PROJECT_STATE.md)；新增语言的完整阶段导航见 [新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)。

本文档中的标准终端步骤不绑定执行主体。维护者可以直接执行，也可以在明确要求且具备相应终端访问能力时交由工具执行；两种方式使用相同命令、前置条件、验收标准和停止规则。只有明确标为 **HUMAN GATE** 的 UI 操作必须由维护者完成。标准化流程应能在没有工具持续参与的情况下按文档独立完成。

## 已验证服务器基线

除非已验证基线实际发生变化，新 locale 首次部署直接复用以下事实，不重新探测或重新设计整台服务器：

| 项目 | 已验证基线 |
| --- | --- |
| 主机与登录 | 阿里云源站 `121.40.248.29`；部署脚本使用 SSH alias `aliyun`，远端生产运维账号为 root |
| Web stack | OneinStack，唯一正式维护目录 `/root/oneinstack` |
| Nginx | vhost 位于 `/usr/local/nginx/conf/vhost/`，证书/私钥位于 `/usr/local/nginx/conf/ssl/`；配置检查与重载使用 `nginx -t && service nginx reload` |
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

`zh-CN` 请求链路为 Cloudflare 权威 DNS → 腾讯云 EdgeOne → 源站 `121.40.248.29:443` → Nginx → `127.0.0.1:3999`。EdgeOne 到源站使用 HTTPS，回源 Host 为 `go-dev.shuijingwanwq.com`。

公网域名迁移不改变 `go-tour.service`、`/data/go-tour/`、仓库名或 Go module path。

## 生产统计配置

Go Tour 统一通过 `TOUR_ANALYTICS` 注入生产统计代码。生产启动路径最终将该环境变量作为 `analyticsHTML` 传给模板的 `{{.AnalyticsHTML}}`；首页 `/` 与 `/tour/...` 页面共用同一个入口。生产环境可在同一个变量中同时放置 Google Analytics 4 和百度统计两套完整 HTML / JavaScript 代码：

```text
TOUR_ANALYTICS='<Google Analytics HTML><Baidu Analytics HTML>'
```

本地开发默认不设置该变量，因此不会加载生产统计代码。实际统计代码以及具体统计 ID 不写入 Git 仓库；公开前端标识也不在本手册中固定记录。

Google AdSense 使用独立的 `TOUR_ADSENSE_CLIENT`。服务端只在该变量为有效 `ca-pub-...` 值时，才在每个完整 HTML 页面的 `<head>` 中生成一次 Auto Ads 站点代码。课程页同时已有共享的手动 `course-ad` 实现：课程 editor partial 提供 mount 容器，Angular route view 在 link / `$destroy` 时调用其 mount / unmount，模板加载课程广告 CSS/JS；该 helper 创建 responsive AdSense `ins` 并请求广告。当前课程广告正在运行 ABCD 自然流量实验：A/B/C/D 各 25%，同一浏览器 tab session 通过 `sessionStorage` 保持稳定，存储不可用时回退为当前 window 内存；A/B/C 分别限制容器最大宽度为 336/468/728px，D 保持不限制宽度的 Responsive 对照组。四组均保持 `data-ad-format="auto"` 和 `data-full-width-responsive="true"`，不使用固定尺寸广告。局部 layout protection 只移除 AdSense/Funding Choices 曾写入编辑器祖先的两种高度覆盖，以保持课程高度和 footer 布局。课程页广告资源与 Auto Ads 一起构成最终广告形态，并非“只注入 Auto Ads、不插入手工广告位”。当前从公开博客首页核验到的配置为：

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

修改 `TOUR_ANALYTICS`、`TOUR_ADSENSE_CLIENT` 或其他影响 HTML shell 的内容后，EdgeOne 可能继续命中旧 HTML。若公网响应仍显示旧页面，应刷新对应 production hostname 的缓存；不要仅根据源站验证就判断公网已更新。公开的 `/socket` 既有安全原则保持不变：production 不注册或开放本地 Socket transport，普通请求和 WebSocket Upgrade 均应保持 404。

### 广告职责、首次接入与最终验收边界

Auto Ads、课程页手动广告、Angular SPA mount/unmount 生命周期、局部 AdSense layout protection，以及覆盖这些行为的 browser tests，都是项目共享实现。新增 locale 只接入和使用这套既有能力；第三门及后续 locale 不重新设计广告位、不重复共享架构验证，也不把广告纳入 TranslationUnit 或 Locale-level language quality review。

新 locale 必须在**首次 production release 激活前**完成以下 production 广告接入准备：

- 在目标 locale service 的 EnvironmentFile 中设置有效的 `TOUR_ADSENSE_CLIENT`，并完成对应 systemd drop-in 配置；
- 完成 Auto Ads 所需的 production 配置；
- 准备并验收课程广告 CSS/JS 的 production asset 来源（zh-CN 为同源；非中文 locale 按共享 assets 策略）；
- 非中文 locale 使用 shared-assets 时，在首次上线前确认共享的 `course-ad.css` 与 `course-ad.js` 已部署，且已完成缓存验收。

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

本次 go-dev 使用 Let's Encrypt、HTTP → HTTPS、`ec-256` 和 Cloudflare DNS provider（`cf` / `dns_cf`），反向代理为 `http://127.0.0.1:3999`。OneinStack 内部使用 acme.sh；正常情况下优先通过 `vhost.sh` 流程创建或管理证书。

acme.sh 记录必须与当前实际 HTTPS origin 一致，不再采用“本项目只应保留 go-dev 一条记录”的旧说明。当前项目至少包含 `go-dev.shuijingwanwq.com`、`ja-go-dev.shuijingwanwq.com` 和 `assets-go-dev.shuijingwanwq.com` 的有效证书用途；新增 locale 还应增加自己的 hostname 记录。旧 `go-tour.shuijingwanwq.com` 已不拥有源站 vhost，其旧管理记录和残留目录已清理。不要仅凭本文枚举删除证书；删除前必须核对当前 vhost 引用与 acme.sh 实际记录。

Cloudflare API Token 及其他密钥属于敏感凭据，不写入文档、不提交仓库，也不记录真实值。

## 新 Locale 首次生产部署

首次部署是建立新 locale 的 production 基础设施和部署 profile，不得直接套用日常 release 切换。开始前应已完成 TranslationUnit promotion、完整 projection、[Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 的完整语言质量审核与 preview acceptance，以及 production publish；production 部署后的 rendered surface 复核和最终 evidence decision 仍属于上线必经步骤。开始前冻结以下值：

```text
locale
hostname / public URL / CDN
data root / releases / current / deploy lock
systemd service / service user
loopback port / localhost health URL
Nginx vhost / TLS certificate paths
Playground allowed Origin
AdSense service environment / Auto Ads 与课程广告资产来源
```

按以下顺序执行：

1. **Hostname 与 CDN 决策**：按 [LANGUAGES.md](../LANGUAGES.md) 确认 hostname。非中文社区语言使用 Cloudflare Free，不引入 EdgeOne + Cloudflare 双层代理；确认共享 assets 策略。
2. **Service、port 与 data root**：为新 locale 选择未占用的 loopback port，建立独立 `/data/go-tour-<locale>/releases`、`current` 和 `.deploy.lock` 边界；创建独立 systemd service，service user 保持 `go-tour`。service 必须从 `current` 下的 release 启动，不能绑定临时上传目录。
3. **TLS 与 vhost**：在 `/root/oneinstack` 使用 `./vhost.sh --proxy --dnsapi` 创建 HTTPS reverse-proxy vhost，使用 Let's Encrypt、`ec-256` 和 `dns_cf`，反向代理到精确 loopback port。检查自动生成的 `location`，不得截获 `/tour/static/`；使用 `nginx -t && service nginx reload`。
4. **DNS / CDN**：创建新 hostname 的 DNS 并启用约定 CDN。等待公开解析生效后分别核对 HTTP → HTTPS、证书 hostname、源站 Host 和 CDN 响应；凭据不进入仓库或命令记录。
5. **Playground Origin**：把新站的精确 `https://<hostname>` 加入 ZgoCloud Playground 代理 allowlist，保持错误 Origin 403、OPTIONS/POST 方法边界和既有 origin 不受影响。未完成此项时 Run / Format 的页面渲染成功不算上线成功。
6. **AdSense production 接入**：在首个 release 激活前，完成本手册“广告职责、首次接入与最终验收边界”所列的 service 环境、Auto Ads 与课程广告资源准备；不得把它留到 production 上线后再补做。
7. **部署脚本 profile**：为新 locale 明确增加并测试 `scripts/deploy-production.sh` 的 fail-closed profile，包括全部路径、service、health URL 和 public URL。该步骤属于首次接入所需的代码能力变更；在 profile 合入前脚本应继续拒绝该 locale，不能用目录名猜测或临时绕过白名单。
8. **首个 release 激活**：部署已验收的 Linux/amd64 bundle，验证权限、SHA-256、`current`、service restart 和连续 localhost health。首次没有可回滚旧 release 时，必须事先定义人工恢复路径，不得声称自动回滚已覆盖。
9. **SEO 与公网验收**：检查 `/`、`/tour/`、`/tour/list`、全部 sitemap URL、`robots.txt`、canonical host、`html lang`、静态资源、`/socket` 404 与保留路径；确认 sitemap 无错误 host、重复或 HTTP failure。
10. **真实浏览器与最终广告验收**：桌面和移动端复核导航、语言选择器、编辑器、Run / Format / Reset 与 runtime message；从 Network 确认实际 Playground endpoint 与 CORS Origin。同时执行上述五项轻量广告确认。最后在 production 重新执行 rendered surface acceptance 关键项，并完成 `data/locale-surface-reviews/<locale>/<review-id>.md` 的 production 结果与最终 decision；不另行执行无广告版本的完整验收。

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

脚本严格读取 `release.json` 的 `locale` 作为唯一事实来源，不接受 `--locale`，也不根据目录名猜测语言。当前 production 白名单仅包含 `zh-CN` 和 `ja-JP`；不支持的 locale 会在 SSH、上传、远端加锁及任何生产修改之前 fail closed。所选 profile 如下：

| locale | releases | current | deploy lock | systemd service | localhost health | public acceptance |
| --- | --- | --- | --- | --- | --- | --- |
| `zh-CN` | `/data/go-tour/releases` | `/data/go-tour/current` | `/data/go-tour/.deploy.lock` | `go-tour.service` | `http://127.0.0.1:3999/` | `https://go-dev.shuijingwanwq.com/` |
| `ja-JP` | `/data/go-tour-ja-JP/releases` | `/data/go-tour-ja-JP/current` | `/data/go-tour-ja-JP/.deploy.lock` | `go-tour-ja-JP.service` | `http://127.0.0.1:4000/` | `https://ja-go-dev.shuijingwanwq.com/` |

两个 profile 的 service user 均为 `go-tour`。

本地目录名应遵循 `go-tour-release-YYYYMMDD-<locale>-<shortsha>` 约定，并且必须以 `go-tour-release-` 开头。脚本只删除这个固定前缀，并对剩余名称执行安全字符检查；因此上例对应的远端目录为：

```text
/data/go-tour/releases/20260813-zh-CN-a4d4dca
```

脚本固定使用 SSH 别名 `aliyun`。当前生产运维账号为 root；远端 `id -u` 不是 `0` 时会在上传前失败，不使用或依赖 `sudo`。部署过程如下：

1. 本地严格检查 bundle 根结构、symlink、`bin/tour`、`release.json`、`site-metadata.json` 和 `SHA256SUMS`；manifest 必须满足 production 约束，其 locale 必须与由同一 `release.json` 选出的 profile 一致。
2. 远端在所选 profile 的 data root 中原子创建 `.deploy.lock` 防止并发部署，并验证 `current`、当前 release、目标名称和对应 systemd service。同名 release 已存在时拒绝覆盖；锁已存在表示可能有正在执行或上一次未完成的部署，脚本直接停止，不分析或自动删除该锁。
3. `rsync` 只上传到所选 profile 的 `releases/.<release>.staging-<token>`，不直接写最终 release 或 `current`，也不使用 `--delete` 覆盖 release。
4. 上传后无条件执行权限归一化：owner/group 为 `root:root`，所有目录为 `0755`，普通文件为 `0644`，`bin/tour` 为 `0755`。随后在远端重新验证 SHA-256，以及 `go-tour` 用户对二进制和必要内容的访问权限；production manifest 已在本地严格检查，第一版不在远端重复解析。
5. staging 在同一文件系统内原子重命名为最终 release；脚本创建临时 symlink 后以原子 `mv` 替换 profile 的 `current`，再 restart 对应 service。
6. 新版本只有在连续 3 次同时满足 profile service 为 `active`、profile localhost health URL 严格返回 HTTP 200 后才算健康。检查最多 12 轮，每轮间隔 3 秒；任何失败都会把连续计数归零，不能用瞬时一次 `active` 判断成功。
7. 只有在 `current` 已明确切换到新 release 后，restart 失败或新版本健康检查明确失败，脚本才会自动回滚：`current` 原子切回旧 release、restart 服务，并使用相同的连续 3 次规则验证旧 release。失败的新 release 会保留用于诊断，不自动删除历史 release。

脚本只区分三类主要结果：在 `current` 原子替换开始前，若脚本能够明确安全清理本次部署资源，会清理 staging/final、临时 symlink 和锁，`current` 保持不变；若远端状态与预期不一致，则保留现场并要求人工检查；新版本明确失败且旧版本恢复健康时会报告已回滚；激活 SSH 中断、current 切换状态无法确认、回滚失败或其他远端状态不确定时会保留 deployment lock 和现场，停止自动处理。遇到最后一种情况不要直接重复部署，应先按日志中所选 profile 的参数人工检查。以下为 `zh-CN` 示例；`ja-JP` 应替换为 `/data/go-tour-ja-JP/current`、`go-tour-ja-JP.service` 和 `http://127.0.0.1:4000/`：

```sh
ssh aliyun 'readlink -f /data/go-tour/current'
ssh aliyun 'systemctl status go-tour.service --no-pager -l'
ssh aliyun 'curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3999/'
ssh aliyun 'journalctl -u go-tour.service -n 80 --no-pager'
```

localhost 连续健康后，脚本才检查对应 profile 的 public URL。正式域名异常属于 CDN、HTTPS、Nginx 或其他外部验收问题，不会自动回滚一个已经稳定健康的源站 release。脚本不调用 CDN API，也不自动清理缓存；若 HTML 或静态资源仍显示旧版本，应根据该语言站点实际使用的 CDN / reverse proxy 检查缓存状态并按需人工刷新。

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
```

目标目录必须不存在。命令通过同级 staging 目录构建，逐字节复制固定 allowlist，生成 `SHA256SUMS`，验证文件集合与校验和后再原子重命名为目标目录。`SHA256SUMS` 只用于部署前后完整性验证，不参与 URL、cache key 或版本选择。不要把完整 `_content` 同步到 assets origin。

共享资产固定使用原逻辑路径，不使用 assets-release-id、content-hash URL、query version、独立 versioned assets release，也不升级 language `release.json`。`/tour/script.js` 明确不拆分、不共享，继续由 language origin 动态拼接并提供。Angular partial、`tree.png`、lesson/footer、HTML、locale 内容、metadata、analytics 和 Playground endpoint 均继续由 language origin 提供；课程页广告自有 CSS/JS 同样由 `asset` helper 按 locale 选择，zh-CN 不依赖 assets origin。未来真实广告所需的 Google 官方脚本仍须直接从 Google 加载，不得代理到 assets origin。Google Fonts 继续使用现有外部 Inconsolata CSS。所有 language bundle 仍保留完整本地静态资源副本，供 projection、preview、rollback 和 CDN 故障排查使用。

`assets-go-dev.shuijingwanwq.com` 已正式部署，Cloudflare 已代理；源站为 `121.40.248.29`，origin root 为 `/data/wwwroot/assets-go-dev.shuijingwanwq.com`，Nginx vhost 为 `/usr/local/nginx/conf/vhost/assets-go-dev.shuijingwanwq.com.conf`。TLS 使用 Let's Encrypt / acme.sh / `dns_cf`，证书和私钥分别位于 `/usr/local/nginx/conf/ssl/assets-go-dev.shuijingwanwq.com.crt` 与 `/usr/local/nginx/conf/ssl/assets-go-dev.shuijingwanwq.com.key`；HTTP 80 永久跳转 HTTPS。

Cloudflare Edge Cache TTL 为 1 个月。项目不主动覆盖 Browser Cache TTL，不给这些固定 URL 设置 `immutable` 或一年浏览器缓存；使用 Cloudflare/origin 默认或 Respect Existing Headers。当前代码 allowlist 为 11 个文件，`course-ad.css` 与 `course-ad.js` 已包含真实课程页 AdSense integration；production assets origin 仍是扩展前的 9 文件，历史公网验收为 9/9。两个 course-ad 文件尚未完成首次 production origin 部署，首次 11/11 production 验收也尚未完成，不能描述为真实广告已经通过 assets production 上线。

### Shared-assets production 发布状态机

以下阶段 A、B、C、E、F、G 是标准终端步骤；维护者可以直接执行，也可以委托具备终端访问能力的工具执行。阶段 D 是唯一的人工 UI 门。任一阶段不满足验收条件时停止，不跳过、不把后续阶段的结果用于掩盖当前失败。

#### 阶段 A：本地生成

目标目录必须不存在。在仓库根目录执行：

```sh
go run -mod=readonly ./cmd/tour-i18n assets export \
  --output /tmp/go-tour-shared-assets
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

如果 staging 与 origin 逐文件相同，脚本输出 `NO CHANGES`，清理 staging/lock，不创建 backup，也不产生 purge URL。如果有变化，脚本在第一次修改 origin 前创建并验证完整非公开备份 `/data/wwwroot/assets-go-dev.shuijingwanwq.com.bak.<token>`，随后才在服务器端把完整 staging tree 以受限 `rsync --delete` 同步进固定 origin。delete 只作用于该精确 origin 内，确保新增、修改、删除后 production 文件集合与正式 export 完全一致；历史 backup 不自动删除。

preflight、lock、upload、staging validation 或 backup 阶段失败时，origin 不变，脚本只在能确认安全时清理本次 staging/lock。production mutation 开始后若同步或严格验证失败，脚本使用刚创建的完整 backup 恢复并重新验证；回滚明确成功时报告部署失败但旧内容已恢复。如果回滚失败、SSH 中断或状态无法确认，脚本保留 lock、staging、backup 与现场，输出只读人工检查命令，禁止直接自动重试。INT、TERM、HUP 在 mutation 前按安全边界清理，mutation 后保留 evidence。

脚本成功只表示 origin deployment 完成。它按稳定顺序输出 added、modified、deleted 的实际固定 URL（`SHA256SUMS` 如发生变化也属于 changed URL），然后在 Cloudflare HUMAN GATE 结束；不调用 Cloudflare API/CLI，也不在 purge 前执行公网 MISS/HIT 验收。

`deploy-shared-assets.sh` 已通过本地 mock 自动化测试，但尚未完成第一次真实 production deployment 验证。当前 production origin 仍是历史 9 文件，两个 course-ad 文件仍未部署，11/11 production 验收仍待完成。第一次受控运行必须严格观察脚本输出和远端状态；如果真实权限或工具基线与预检不符，立即停止，不绕过检查。

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

部署脚本比较更新前 origin 与已验证 staging，只列出内容实际新增、修改或删除的固定 URL；不要默认刷新全部 11 个 URL，也不要默认使用 Purge Everything。首次部署课程广告资源时至少包括：

```text
https://assets-go-dev.shuijingwanwq.com/tour/static/go-dev/course-ad.css
https://assets-go-dev.shuijingwanwq.com/tour/static/go-dev/course-ad.js
```

如果脚本输出其他变化 URL，把它们一并加入清单；删除路径的旧 URL 也必须刷新。脚本输出完整清单后到达阶段 D 并结束，不继续公网缓存验收。

#### 阶段 D：HUMAN GATE — Cloudflare Dashboard Custom Purge

维护者在 Cloudflare Dashboard 中选择 `assets-go-dev.shuijingwanwq.com` 所属 zone，进入缓存管理的 Purge Cache / Custom Purge 功能，按 URL 刷新阶段 C 的精确清单。Cloudflare UI 文案可能变化，这里只定义操作目标，不把 UI 文案当作程序接口。

当前 shared-assets production 发布不要求自动化 Cloudflare purge。不得为此搜索本地或服务器上的 Cloudflare Token、要求维护者提供 API Token、把凭据写入仓库或 shell history、猜测既有凭据入口、调用 Cloudflare API、使用 Wrangler 自动刷新或安装新的 Cloudflare CLI。未来如需自动化，应作为独立受控改进处理。

Dashboard purge 是正常 human gate，不是 deployment failure，也不是缺少 API 权限错误。维护者明确确认 Custom Purge 已完成后，才能继续阶段 E；确认前不得把 MISS/HIT 不符合预期解释为新资源发布失败。

#### 阶段 E：公网缓存验收

对阶段 C 的每个 URL 连续请求两次并保留响应头。第一次应符合刷新后首次回源预期，例如 `CF-Cache-Status: MISS`；第二次应符合缓存命中预期，例如 `CF-Cache-Status: HIT`。如果实际响应头名称或 Cloudflare 行为与已验证基线不同，记录真实响应并停止判断，不猜测或伪造 MISS/HIT 结论。

当前两个首次部署 URL 的验收命令如下；如果阶段 C 还列出其他变化 URL，把它们追加到 `urls`：

```sh
urls=(
  'https://assets-go-dev.shuijingwanwq.com/tour/static/go-dev/course-ad.css'
  'https://assets-go-dev.shuijingwanwq.com/tour/static/go-dev/course-ad.js'
)
for url in "${urls[@]}"; do
  curl -fsS -o /dev/null -D - "$url"
  curl -fsS -o /dev/null -D - "$url"
done
```

#### 阶段 F：完整性验收

逐一请求正式 11 个 allowlist URL，确认 HTTP 正常，并把公网响应内容的 SHA-256 与阶段 A 的 `SHA256SUMS` 对照。必须达到 11/11 内容一致；只验证本次 purge 的文件不能替代完整 allowlist 验收。

```sh
public_assets=$(mktemp -d)
while read -r checksum logical_path; do
  mkdir -p "$public_assets/$(dirname "$logical_path")"
  curl -fsS \
    "https://assets-go-dev.shuijingwanwq.com/$logical_path" \
    -o "$public_assets/$logical_path"
done < /tmp/go-tour-shared-assets/SHA256SUMS
cp /tmp/go-tour-shared-assets/SHA256SUMS "$public_assets/SHA256SUMS"
cd "$public_assets"
sha256sum -c --strict SHA256SUMS
```

验收结束后可删除该 `mktemp` 输出目录；不要把它误作下一次正式 export source。

#### 阶段 G：边界验收

以下路径不属于 shared allowlist，必须继续返回 HTTP 404：

```text
/tour/script.js
/tour/static/img/tree.png
/tour/static/partials/editor.html
```

任一路径意外返回共享内容，shared-assets 发布验收失败。固定 URL、Cloudflare Edge Cache TTL 与 Browser Cache TTL 策略保持不变；不引入 hash filename、version directory、query version、loader、manifest 或 assets release ID。

```sh
for path in \
  /tour/script.js \
  /tour/static/img/tree.png \
  /tour/static/partials/editor.html
do
  curl -sS -o /dev/null -w "$path %{http_code}\n" \
    "https://assets-go-dev.shuijingwanwq.com$path"
done
```

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
