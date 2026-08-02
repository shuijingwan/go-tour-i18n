# zh-CN 页面路径约定

未来候选和正式页面使用：

```text
locales/zh-CN/pages/<article-name>/<section-number>.article
```

例如：

- `locales/zh-CN/pages/welcome/1.article`
- `locales/zh-CN/pages/basics/1.article`
- `locales/zh-CN/pages/concurrency/10.article`

每个文件只包含一个完整顶层 `present.Section`，不得包含整份 Tour、多个课程页面、翻译服务响应包装或句子级 JSON。文件必须能通过项目的 present 解析与结构校验后才可进入 `ready`。

本目录当前没有真实课程页面。
