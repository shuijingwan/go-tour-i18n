# 生产运维手册

本文档记录当前已验证的生产维护入口与操作边界。项目进度和部署历史见 [PROJECT_STATE.md](PROJECT_STATE.md)。

## 当前生产架构

- 正式站点：<https://go-dev.shuijingwanwq.com/>；A Tour of Go 使用 `/tour/` 路径。
- 请求链路：Cloudflare 权威 DNS → 腾讯云 EdgeOne → 源站 `121.40.248.29:443` → Nginx → `127.0.0.1:3999`。
- EdgeOne 到源站使用 HTTPS，回源 Host 为 `go-dev.shuijingwanwq.com`。
- Go 生产服务为 `go-tour.service`，监听 `127.0.0.1:3999`，release 根目录为 `/data/go-tour/`。

公网域名迁移不改变 `go-tour.service`、`/data/go-tour/`、仓库名或 Go module path。

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

## 最小生产验收

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
