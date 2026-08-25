# 课程页正式 SEO Metadata 规范

本文定义每个 locale 的课程页 SEO description 正式资产、生成输入、identity、stale 规则和发布质量边界。该资产不属于 TranslationUnit candidate、status、Quality Check、Final Review 或 promotion evidence。

正式资产固定为：

```text
locales/<locale>/course-metadata.json
```

## 生命周期与边界

正式顺序为：

```text
TranslationUnit promotion
→ offline AI metadata generation
→ metadata validation
→ projection / preview
→ Locale Surface Review
→ publish / prerender
→ production
```

AI 只能在 promotion 完成后作为离线维护步骤生成 metadata。禁止在浏览器 runtime、production HTTP 请求、服务启动、projection、publish 或 prerender 期间调用模型；publish 也不得自动生成、补齐或刷新 description。生成结果必须持久化到正式 locale 资产并进入 Git，随后所有构建和运行阶段只消费已经验证的确定性内容。

不得使用 `plainText`、正文机械截取、标题、其他语言或旧 description 作为缺失时的 fallback。正式 production 最终必须要求当前 catalog 全部课程页 metadata 完整；任一课程页缺失、额外或 stale 时必须 fail closed。基础设施分阶段接入期间，loader/validator 可以先存在而不改变当前 runtime，但最终发布 gate 不得保留例外。

## Schema version 1

顶层结构：

```json
{
  "schema_version": 1,
  "locale": "<locale>",
  "generator_contract": "course-seo-description-v1",
  "pages": []
}
```

每个 `pages` entry 包含：

```json
{
  "page_id": "welcome/4",
  "route": "/welcome/3",
  "description": "目标语言的单段纯文本摘要",
  "source_sha256": "<完整英文 Page TranslationUnit source SHA-256>",
  "target_sha256": "<完整最终 canonical locale Page target SHA-256>",
  "glossary_sha256": "<完整 locales/<locale>/glossary.yaml SHA-256>",
  "generation": {
    "provider": "<provider>",
    "model": "<model>",
    "prompt_version": "course-seo-description-v1",
    "generated_at": "<RFC3339 UTC>"
  }
}
```

`page_id` 是唯一正式主键，来自 committed catalog 的持久 Page identity。`route` 是必须与 catalog 当前映射一致的派生 identity，不得作为主键；例如 `page_id = welcome/4` 当前对应 `route = /welcome/3`，两者不能根据字符串相同来推导。

`provider`、`model` 和 `generated_at` 记录 provenance。`source_sha256`、`target_sha256`、`glossary_sha256`、`generator_contract` 和 `prompt_version` 共同决定结果是否仍然 current。

## 生成契约

每一页的正式 AI 生成输入不可缺少：

1. 完整英文 Page TranslationUnit source；
2. 完整最终 canonical locale Page target；
3. 完整 `locales/<locale>/glossary.yaml`。

模型不得只读取页面开头、渲染 HTML、纯文本截取或局部段落。description 必须忠实于完整页面，不得添加 source 没有支持的技术解释、保证、结论、品牌关系或其他信息。每个 locale 独立使用自己的 canonical target 与 glossary 生成目标语言 description。

本规范不改变 TranslationUnit 翻译、automatic validation、Quality Check、Final Review 或 promotion。metadata 生成失败只阻止后续 locale release，不得反向伪造 TranslationUnit review evidence。

### 离线组装

AI 只负责提供完整的 `page_id → description` 集合，严格输入格式为：

```json
{
  "pages": [
    {"page_id": "welcome/1", "description": "目标语言摘要"}
  ]
}
```

输入不得携带 route、hash、schema 或 generation provenance。使用离线命令组装正式文件：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata assemble \
  --locale <locale> \
  --descriptions <descriptions.json> \
  --provider <provider> \
  --model <model> \
  --generated-at <RFC3339-UTC> \
  --output <output>
```

工具从当前 catalog、完整 glossary、正式 ready status 和通过 candidate validation 的 canonical Page target 自动生成全部 identity 与 provenance 字段，按 catalog Page 顺序固定缩进和结尾换行。
组装结果先在内存中通过同一正式 validator，再原子写入 output；输入集合不完整、identity stale 或 description 不合法时不留下半成品。
该命令不调用模型，也不从页面内容生成或补齐 description。

## Strict validation

唯一 loader/validator 必须严格解析 JSON，拒绝未知字段和额外 JSON value，并验证：

- `schema_version`、`locale`、`generator_contract` 与受支持版本完全一致；
- page set 与当前 catalog Page set 精确相等，无缺失、额外或重复 `page_id`；
- route 无重复，且每个 `page_id` 的 route 等于 catalog 当前 route；
- `source_sha256` 等于 catalog 当前 Page source identity；
- `target_sha256` 等于通过正式 ready status、canonical path 与 candidate validation 后加载的当前 locale 完整 canonical Page target 的实际原始字节 SHA-256；
- 每条 `glossary_sha256` 等于当前完整 glossary 文件的实际字节 SHA-256；
- `prompt_version` 受支持，generation provider/model 非空，时间为 RFC 3339 UTC；
- description Unicode trim 后非空，没有首尾空白，是不含换行或控制字符的单段纯文本；
- description 不含 HTML tag、Markdown code fence 或 URL；
- 第一版长度按 Unicode code point 计算，硬范围为 30–200；
- 不存在字节完全相同的 description；
- 不存在移除 Unicode whitespace/punctuation/symbol 并统一大小写后相同的 description。

长期完整性规则是 metadata page set 与 catalog Page set 精确相等，不能只硬编码页数。当前 production catalog 为 103 页，因此对当前正式资产执行该规则必须明确得到 `103/103`；未来 catalog 经正式流程增加或删除页面时，exact-set gate 会立即使旧 metadata 失效。

自动 validator 只负责可确定的结构、identity、staleness 和最低文本安全约束。它不能证明 description 忠实、自然、技术准确或没有无依据扩写。

## Stale 规则

一条 metadata 只有以下 identity 全部匹配时才合法：

- `page_id`；
- `route`；
- `source_sha256`；
- `target_sha256`；
- `glossary_sha256`；
- `generator_contract` 与 `prompt_version`。

任一 identity 改变都视为 stale：target、source 或 route 改变使对应 page stale；catalog 新增或删除页面使 exact-set validation 失败；generator contract 或 prompt version 升级必须通过修改受支持版本显式失效。第一版中 glossary 任意字节改变都会使该 locale 每一条记录的 `glossary_sha256` 失配，因此整个 locale 全量 stale，不做术语影响范围猜测。

允许离线工具只重新生成 stale page，但写入并准备发布的正式文件始终必须是完整集合，不能把 partial metadata 当作正式资产。

## Locale Surface Review

`course-metadata.json` 属于 Locale Surface Review 的正式 locale-level 输入。审核者必须对当前 catalog 每一页同时读取完整英文 Page source、完整最终 canonical target、当前 glossary 和 description，逐页检查：

- 忠实覆盖页面主题且没有无依据扩写；
- 技术含义准确并遵守 glossary；
- 目标语言表达自然、完整，适合搜索摘要语境；
- 不是正文截断、代码片段或其他页面的泛化重复摘要；
- `page_id`、route、locale 与实际 rendered page 一致。

Surface Review evidence 应记录 catalog、course metadata、glossary 和 target identity，以及完整 page count 和 validation 结果。自动 validation 通过不能替代逐页语言质量审核；任一页未通过时不得 publish。

## 第一阶段实现状态

第一阶段只提供 schema、严格 loader/validator、自动测试和本规范。它不生成真实 description，不把 metadata gate 接入 projection/publish，不改变 prerender、production runtime、`window.__tourSEO` 或当前浏览器 metadata 行为。后续阶段只有在所有正式 locale 准备好完整且审核通过的资产后，才可移除现有机械摘要逻辑并启用 production fail-closed gate。
