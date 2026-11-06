# Site v2 Preview browser coverage audit

本次审计以 2026-10-14 工作树为准，只审计并扩展 verifier；未运行正式 browser acceptance，也不构成 visual HUMAN PASS。未写语言、Generation / Reviewer evidence、closure、Surface receipt、activation 或 Production 工件。

修订前，verifier 对 103 个 Tour Page 做 raw HTTP shell 检查，浏览器访问首页、Tour 根入口、目录、welcome/1、basics/11、moretypes/1，并测试编辑器和 SPA。激活 shell 时，`/translation/` 只有 raw HTML 检查。Learn/Docs 只出现在 sitemap exact-set，页面可以完全未访问；因此旧 PASS 不能证明 v2 coverage。旧首页还使用历史 `data/site-shell/home.html` 的链接作为预期，与当前 source adoption 不符。

当前路由 authority 为 `sitecontent.LoadCurrent`（包含 shell successor adoption）和 locale `content-scope.json`；`site.Resolver` / `Content.Compose` 决定实际本地 canonical routes 和 auxiliary endpoints。新增只读 `preview-browser-plan --locale` 复用这些 authority 和 `renderPage`，以 source-rendered HTML 给出结构预期。它不重新记录 completion、不读写 receipt，也不把 source HTML 作为 locale runtime fallback。Preview 启动的正式 gate 保持原样。

当前 zh-CN 有 156 条 canonical routes：105 Tour routes、49 Learn/Docs Page、Homepage 和 translation。family 精确集合为 `/`、`/translation/`、`/tour/**`、`/learn/**`、`/doc/tutorial/**`、`/doc/database/**`、`/doc/modules/**`、`/doc/security/**`；没有激活泛 `/doc/`、`/play/` 或 `/doc/play/` family。

| Learn/Docs family | Page 数 | 含静态 pre 的 Page 数 |
| --- | ---: | ---: |
| `/learn/**` | 1 | 0 |
| `/doc/tutorial/**` | 18 | 16 |
| `/doc/database/**` | 9 | 7 |
| `/doc/modules/**` | 9 | 8 |
| `/doc/security/**` | 12 | 6 |

扩展后的 machine 检查逐一 GET 全部 51 个 Site Page，要求无重定向的 200、locale、production canonical、title、唯一 substantive main、唯一共享 header/footer、source-bound navigation/assets 和预期非空代码块；每个 family 另查一个未知路径必须直接 404。sitemap 仍要求完整激活集合完全一致，Tour 原有 raw checks 保留。

浏览器抽样按 `(family, source format, 是否有 pre, 是否有内嵌 executable)` 分桶，选每桶 route 字典序第一条，再加入所有 executable route。不按 locale 写死列表；Go plan 和 Python exact-set 检查共同拒绝漏页、未分类或歧义 family。

当前抽出的 12 页如下，每页检查 1280×800 和 375×812；其中 10 页属于 Learn/Docs：

```text
/
/translation/
/learn/
/doc/database/
/doc/database/cancel-operations
/doc/modules/developing
/doc/modules/gomod-ref
/doc/security/
/doc/security/best-practices
/doc/tutorial/
/doc/tutorial/add-a-test
/doc/tutorial/database-access
```

每个样本检查 browser DOM、main 可见文本/宽度、canonical、导航/资源、代码块、图片完成加载、页内 anchor 目标和 document overflow ≤2px。Learn sidebar anchor 需实际点击并到达有尺寸的 section，当前 shared shell 的 mobile drawer 需实际开合。原 Tour desktop/mobile pane 几何、RTL/LTR、Format、Reset model/view/localStorage、Run、SPA metadata convergence 和 `/socket` 404 均保留；语言选择器在 Tour 目录检查，当前 v2 Homepage 使用自身 renderer contract。

| 已知 review_required source 对应 route | 当前页面的执行边界 |
| --- | --- |
| `/doc/modules/release-workflow` | 静态命令/代码块；终端步骤不会在页面内执行 |
| `/doc/security/` | 文档导航和文字，无内嵌 runner，无 pre |
| `/doc/tutorial/compile-install` | 静态命令与输出示例，需要读者本地终端 |
| `/doc/tutorial/generics` | 静态代码；链接到外部 `https://go.dev/play/p/apNmfVwogK0`，不是本地 runner |
| `/` | 内嵌 textarea、Run、示例选择器及 output；唯一新增内嵌 executable |

`/doc/tutorial/web-service-gin` 另有外部 Google Cloud Shell walkthrough 链接。上述外链检查 navigation identity，不操作外部 Playground/Cloud Shell。49 个 Learn/Docs Page 均不将静态代码升级为执行控件。

Homepage 的 `index.js` 绑定 `playground`；默认 HTTPTransport，Run 先 POST `/_/fmt?backend=`，再 POST `/_/compile?backend=`，由现有 Tour HTTP proxy 转发 Playground。没有 local OS execution widget，也不使用 `/socket`。当前 Homepage 没有独立 Format / Reset 按钮；这两个操作仍在 Tour 检查。Homepage 的 8 个 toy 选项共享一个 runner；验收每个内嵌 runner，使用确定性短程序与错误程序，不逐一运行所有 toy。

新增执行检查点击真实 Run，等待 stdout 精确 marker、terminal system 状态、无 stderr，以及 compile response 无 Errors/VetErrors、Status=0 且 stdout 匹配；随后提交语法错误，要求页面显示 failure state。没有注入 transport 或伪造 API 成功响应。

全程启用 CDP Runtime / Log / Network：console error/assert、未捕获异常、弹窗、HTTP ≥400、loadingFailed、WebSocket 错误和意外 `/socket` 连接均失败。Site 页面还等待 document complete、全部已发请求完成和短暂 network quiet；包括外部 assets。唯一允许的 HTTP 错误是旧 Tour 主动验证的精确 `/socket` Fetch 404。导航重试不能清除已观察到的错误。

当前 source 接线存在一个必须在后续正式验收暴露的问题：renderer 将非 canonical scripts/assets 指向 `https://go.dev/`，Homepage `index.js` 初始化会请求当前 origin 的 `/doc/play/hello.go`，而 `Compose` / `tourAuxiliary` 没有注册该路由。定向 composition test 确认其 404，同时确认 compile/fmt 继续交给 Tour handler。外部脚本在浏览器实际加载的 bytes 未在本轮联网核验；verifier 不能据 source 推测运行成功，必须在正式阶段真实执行并 fail closed。本轮未修改该 runtime 接线。

定向测试限于 verifier-specific Python tests、preview plan/route-family 和 runtime-boundary Go tests、`git diff --check`。没有正式浏览器运行、截图视觉判断、Production、publish 或 activation。
