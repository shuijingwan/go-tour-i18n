# 课程页正式 SEO Metadata 规范

本文定义课程页 SEO description 的两阶段正式架构、identity、人工审核 authority、stale 传播和发布边界。相关资产不属于 TranslationUnit candidate、status、Quality Check、machine finalization 或 promotion evidence。

## 两阶段架构

正式数据流为：

```text
完整 English Page source
→ 每页一次 canonical English semantic extraction
→ ChatGPT 对全部 Page 做 source ↔ canonical description 人工审核
→ canonical source-description review gate
→ canonical English description 固定 semantic scope
→ 结合当前 Page 完整 source/ready target + 完整 locale glossary 忠实本地化
→ schema v2 localized description
→ metadata validation / build
→ 完整 source + canonical description + target + glossary + localized description 的 Locale Surface Review
→ preview / publish / production
```

第一阶段只做一次英文语义抽取；第二阶段忠实本地化同一语义范围，不重新摘要或选择重点。生成优化不减少最终 Locale Surface Review 的完整上下文。

AI 只能作为离线维护步骤产生 `page_id → description` 文本。浏览器 runtime、HTTP 请求、服务启动、projection、build、preview、publish、prerender 和 production 均不得调用模型，也不得自动生成、补齐或刷新 description。禁止 `plainText`、正文截取、标题、其他 locale 或旧 description fallback。

## Stage 1：canonical English source descriptions

正式 Git 资产固定为：

```text
data/course-seo/source-descriptions.json
```

schema 与 prompt contract 均为 `course-seo-source-description-v1`。顶层为：

```json
{
  "schema_version": 1,
  "generator_contract": "course-seo-source-description-v1",
  "pages": []
}
```

每个 Page 按正式 Catalog 顺序记录：

```json
{
  "page_id": "welcome/4",
  "route": "/welcome/3",
  "description": "Canonical English description.",
  "source_sha256": "<完整 English Page source SHA-256>",
  "generation": {
    "provider": "<provider>",
    "model": "<model>",
    "prompt_version": "course-seo-source-description-v1",
    "generated_at": "<RFC3339 UTC>"
  }
}
```

模型必须逐 Page 读取完整 English Page TranslationUnit source，只输出该页的 `page_id → description`。不得读取其他 Page 后跨页补充，也不得加入 source 不支持的信息。本命令只组装调用者已经生成的完整文本集合，不调用模型、不接受 route、hash、schema 或 provenance：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata source assemble \
  --descriptions <canonical-english-descriptions.json> \
  --provider <provider> \
  --model <model> \
  --generated-at <RFC3339-UTC> \
  --output data/course-seo/source-descriptions.json

go run -mod=readonly ./cmd/tour-i18n course-metadata source check
```

`assemble` 从当前 Catalog 自动派生顺序、route、source SHA 与 generation contract，在内存中通过正式 validator 后原子写入。当前能力刻意只提供完整 assemble；English source 改变时，先重新生成并审核完整 canonical asset，不把 partial 文件当正式资产。

### Strict validation

source-description validator 拒绝未知字段与额外 JSON value，并要求：

- Page set、顺序、`page_id`、route 与当前 Catalog 精确一致；
- `source_sha256` 与完整当前 English Page source 精确一致；
- schema、generator contract、prompt version 精确受支持；
- provider/model 非空，`generated_at` 为 RFC 3339 UTC；
- description trim 后不变、单行、plain text、无 control character、HTML、Markdown fence 或 URL；
- Unicode code point 长度为 30–200；
- exact duplicate 和移除 Unicode whitespace/punctuation/symbol、统一大小写后的 normalized duplicate 均为 0。

完整性不硬编码页数，而是与当前 Catalog exact-set 比较；当前正式 Catalog 因此必须得到 `103/103`。English Page source、route、Page set/order 或 contract 改变都会使旧 asset stale 并 fail closed。

## Canonical source-description 人工审核 authority

canonical English descriptions 是全部 schema v2 locale 的共同语义来源，machine validation 不能替代语言与语义审核。ChatGPT 必须对 Catalog 全部 Page 逐页读取完整 English source 与 canonical description，检查忠实度、完整性、技术准确性、unsupported expansion、generic/duplicate 表达和 route identity；不允许 sampling。

该审核不属于 TranslationUnit Quality Check，不使用 A/B/C/D，不生成 TranslationUnit QC/finalization evidence。人工 evidence 固定为：

```text
data/course-seo/source-description-reviews/<review-id>.md
```

Markdown prose 可自由记录逐页全量审核范围、日期与 issues，但必须且只能包含一个下列 machine-readable identity block；block 使用 strict JSON，未知字段、任意层级重复的 JSON object member name、额外 JSON value、缺字段、重复/缺失 marker 或 malformed JSON 均 fail closed：

```markdown
<!-- course-source-description-review:start -->
{
  "review_id": "<review-id>",
  "artifact_sha256": "<source-descriptions.json 原始字节 SHA-256>",
  "catalog_source_sha256": "<当前完整 English Page Catalog/source identity>",
  "source_description_schema_version": 1,
  "generator_contract": "course-seo-source-description-v1",
  "prompt_version": "course-seo-source-description-v1",
  "page_count": 103,
  "reviewer": "<reviewer>",
  "decision": "passed"
}
<!-- course-source-description-review:end -->
```

`review_id`、`reviewer` 必须分别与 `review-record --review-id`、`--reviewer` 精确一致；artifact、Catalog/source、schema、contract、prompt 和 Page count 必须与命令读取的当前正式输入精确一致。`decision` 只有精确为 `passed` 才能生成 passed gate；任一 Page 未通过时 evidence 必须记录 `failed`，且 `review-record` 必须拒绝生成 gate。

完成全量审核并明确通过后，才由维护者记录 machine-readable receipt：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata source review-record \
  --review-id <review-id> \
  --reviewer <reviewer>

go run -mod=readonly ./cmd/tour-i18n course-metadata source review-check
```

receipt 写入同目录的 `<review-id>.gate.json`。`review-record` 严格解析并核对上述 block 后，固定记录 `decision = passed`、reviewer、stage、程序计算的完整 source-description artifact SHA、English Page Catalog/source identity、schema、generator contract、prompt version，以及 Markdown evidence 原始字节的 `evidence_sha256`。Gate JSON 顶层及嵌套 `inputs` 同样拒绝重复的 object member name。不得仅凭 Markdown 非空生成 passed receipt。

`review-check` 对候选 current gate 重新读取 `<review-id>.md`，核对 `evidence_sha256`，再次严格解析 block，并验证 block、gate 与当前正式输入三者一致。当前 evidence 缺失、被修改、malformed、non-passed 或 stale 时均 fail closed；任何绑定输入改变后旧 gate stale，missing、unknown、malformed、non-passed 或没有 current gate 也均 fail closed。命令不会自行进行审核，也不会自动声称审核通过。本架构任务不生成实际 asset、人工 evidence 或 passed receipt。

## Locale course metadata

正式路径保持不变：

```text
locales/<locale>/course-metadata.json
```

### Schema v1：兼容边界

schema v1 的 `course-seo-description-v1` contract、字段、strict validation、stale 语义和 runtime 消费保持原样。它继续绑定完整 English Page source、完整 canonical locale target、完整 locale glossary 和 v1 provenance；旧 v1 asset 不读取或绑定新的 canonical source-description asset/review gate。

```json
{
  "schema_version": 1,
  "locale": "<locale>",
  "generator_contract": "course-seo-description-v1",
  "pages": [{
    "page_id": "welcome/4",
    "route": "/welcome/3",
    "description": "目标语言 description",
    "source_sha256": "<完整 English Page source SHA-256>",
    "target_sha256": "<完整 canonical locale Page target SHA-256>",
    "glossary_sha256": "<完整 glossary SHA-256>",
    "generation": {
      "provider": "<provider>",
      "model": "<model>",
      "prompt_version": "course-seo-description-v1",
      "generated_at": "<RFC3339 UTC>"
    }
  }]
}
```

v1 模型生成输入仍是每页完整 English source、完整最终 canonical locale target 与完整 locale glossary；v1 validator 仍拒绝未知字段、错误 page set/route/source/target/glossary identity、错误 contract/provenance，以及不满足同一 30–200 code point、plain-text、无重复约束的 description。原 `course-metadata assemble` 默认仍生成 v1，原 `refresh` 对 v1 base 仍按旧 identity 精确刷新 stale subset；`revise` 也可在整个 v1 base identity current 且通过 strict validation 时只修订明确 subset，不改变 schema 或 v1 输入边界。

因此现有 live locale 不迁移、不重写、不重新生成，也不会仅因仓库出现或改变新的全局 asset/gate 而使 course metadata 或 Locale Surface Review A gate stale。以后迁移某个旧 locale 是独立任务。

### Schema v2：faithful localization

顶层为：

```json
{
  "schema_version": 2,
  "locale": "<locale>",
  "generator_contract": "course-seo-localization-v2",
  "pages": []
}
```

每页为：

```json
{
  "page_id": "welcome/4",
  "route": "/welcome/3",
  "description": "目标语言 description",
  "source_sha256": "<完整 English Page source SHA-256>",
  "source_description_sha256": "<canonical English description 精确文本 SHA-256>",
  "target_sha256": "<完整最终 canonical locale Page target SHA-256>",
  "glossary_sha256": "<完整 locale glossary 文件 SHA-256>",
  "generation": {
    "provider": "<provider>",
    "model": "<model>",
    "prompt_version": "course-seo-localization-v2",
    "generated_at": "<RFC3339 UTC>"
  }
}
```

schema v2 localization 允许并推荐向生成 session 提供当前 Page 的完整上下文：

- canonical English description；
- 完整 English Page source；
- 完整最终 ready canonical locale target；
- 完整 `locales/<locale>/glossary.yaml`；
- 目标 locale identity；
- `course-seo-localization-v2` contract/constraints。

canonical English description 是 localized description 唯一的 semantic-scope authority。完整 source 与 target 只用于核对技术语义、glossary/正文术语一致性、自然的目标语言表达和当前 Page 实际内容一致性，不是重新摘要或重新选择重点的 authority。即使上下文提供了更多细节，localized description 仍必须完整保留 canonical description 的关键语义，不得增加其未授权的语义、删除关键语义或 keyword stuffing。

同一 generation session 或 batch 可以处理多个 Page；每个 Page 必须独立使用自己的 canonical description 决定 semantic scope，不得从同 batch 的其他 Page 补充、混合或推断语义，也不得混淆 Page identity。

当 Generation provider 为 `chatgpt` 时，为避免普通 ChatGPT 通过 Remote Desktop Commander 分批重复读取 103 Page full context，首次 schema v2 localization 提供 deterministic transport：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle \
  --locale <locale> \
  --output /tmp/<locale>-course-seo-localization-generation.zip
```

命令只读当前正式输入，不调用模型、不生成 localized description、不修改 `course-metadata.json`，也不新增 gate。它要求 current canonical source-description review authority、完整 ready Page set、当前 glossary 与 locale identity 均可验证，然后生成 deterministic ZIP：`manifest.json` 绑定所有成员 SHA-256；`course-seo-localization.json` 逐 Page 包含 canonical description、完整 English source、完整 ready target 与对应 identity；`formal/source-descriptions.json` 保留 canonical asset 原始正式字节，`formal/` 还包含完整 glossary 和 `locale.json`；`authority/` 包含当前 `AGENTS.md`、ChatGPT generation 规范与本规范。输入变化后必须重新导出，不得复用旧 ZIP。

当 ZIP 来自未变化的正式 working tree 且 manifest/hash 完整时，ChatGPT Generation session 对附件的完整读取即满足这批首次 localization context 的传输要求，不再通过 Remote Desktop Commander 重复读取相同 Page/source/target/glossary。该 ZIP 是 ChatGPT transport optimization，只改变传输；canonical description 仍是唯一 semantic-scope authority，103/103 Page coverage、正式 provenance、assemble validator 和后续 Locale Surface Review 均不变。

普通 ChatGPT 与 Codex **GPT-5.6 Sol + High** 都是 schema v2 localized description 的正式生成环境，实际使用该 locale 开始正式 generation 前选定并原则上持续使用的 provider。两者都必须遵守相同的 `course-seo-localization-v2` semantic-scope 与上下文使用边界。ChatGPT 使用 `provider=chatgpt`、`model=gpt-5.6-sol-high`；Codex 使用 `provider=codex`、`model=gpt-5.6-sol-high`，并都使用真实 RFC 3339 UTC `generated_at`。Codex 已直接工作在当前 repository 中，必须完整读取当前 canonical source descriptions、完整 English Page source、完整 ready target、完整 locale glossary、locale identity 与 current authority，不需要生成或读取上述 ChatGPT localization ZIP。生成环境只产出 description 输入，不得直接修改正式 `course-metadata.json`；正式资产仍只能由下述 assemble/refresh/revise CLI 机械生成。

生成 session 与最终独立 ChatGPT Locale Surface Review session 必须分离，包括语言质量 revision：发现缺陷的审核 session 不得同时生成替换译文并批准自己的输出。这一独立性要求不要求 Course SEO generation session 是从未读取过 Page source/target 的新 conversation；同一 locale generation session 可在 TranslationUnit generation/revision 后继续执行 Course SEO localization/refresh/revision。本项不改变 schema、canonical source-description review authority 或 Locale Surface Review gate。

本地化可调整语序、句法、必要形态和 glossary 术语，但必须保持 canonical description 的 semantic scope：不删除关键语义、不增加信息、不 keyword stuffing、不根据 Page body 重新选重点。`target_sha256` 仍由工具读取完整 ready canonical target 自动计算，只负责正式资产的 identity/freshness；target body 作为生成上下文的作用与该 hash 绑定职责互不替代。

v2 同样使用唯一 strict loader：拒绝未知字段与额外 JSON value，要求 exact Page set、Catalog order、route、source/source-description/target/glossary identity、受支持 contract/provenance，以及相同的 description 文本安全、长度和 duplicate 约束。

上述变更只调整离线 generation context，不把额外上下文写入 schema 或 stale identity。`schema_version`、generator/prompt contract、四类 hash 与 provenance 均不变，因此不会仅因本规则调整而使现有 schema v2 asset stale。

### Assemble、refresh 与 revise

首次生成 schema v2：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata assemble \
  --schema-version 2 \
  --locale <locale> \
  --descriptions <localized-descriptions.json> \
  --provider <provider> \
  --model <model> \
  --generated-at <RFC3339-UTC> \
  --output <output>
```

增量维护 schema v2：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata refresh \
  --schema-version 2 \
  --locale <locale> \
  --descriptions <stale-localized-descriptions.json> \
  --provider <provider> \
  --model <model> \
  --generated-at <RFC3339-UTC> \
  --output <output>
```

当 Locale Surface Review 判定某些 localized description 语言质量不合格、但整个正式 metadata asset 的 identity 仍 current 时，只修订明确 subset：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata revise \
  --schema-version 2 \
  --locale <locale> \
  --descriptions <revised-localized-descriptions.json> \
  --provider <provider> \
  --model <model> \
  --generated-at <RFC3339-UTC> \
  --output <output>
```

schema v2 的三个入口都要求 current canonical source-description asset 与 current passed source review gate。模型输出严格仍为：

```json
{"pages":[{"page_id":"welcome/1","description":"目标语言 description"}]}
```

调用者不得填写 hash、route、schema 或 provenance。输入顶层只能有 `pages`，每项只能有 `page_id` 与 `description`；unknown field、duplicate `page_id`、unknown `page_id`、malformed/额外 JSON value 和空 revision subset 均 fail closed。工具从 Catalog、canonical source descriptions、ready canonical target 和 glossary 自动派生全部 identity，正式验证后原子写入。`assemble` 要求完整 Page set；`refresh` 从 base schema 推断版本，并在显式 `--schema-version 2` 时核对一致，只接受精确 stale subset。

refresh 对 non-stale entry 原样保留 description 与真实 generation provenance；只对 stale entry 写入新 description、当前 identity 和本轮 provenance。catalog Page set/order 与 base 不一致时 fail closed，不把 partial output 当正式资产。v1 的原命令保持可用：`assemble` 未指定 schema 时仍为 v1，`refresh` 未指定时沿用 base schema。

`revise` 与 `refresh` 的语义不可互换：identity stale 时必须使用 `refresh`；只有人工语言审核判定 description 需要改写、且 base 全部 identity current 时才使用 `revise`。`revise` 在任何 Page identity stale、base malformed/incomplete 或 schema v2 source-description/review authority non-current 时整体 fail closed，不顺带刷新 identity。它只替换输入中所选 Page 的 description，按当前正式输入重新派生该 Page identity，并写入本轮真实 generation provenance；未选 Page 的 description、identity 与原 generation provenance 原样保留。命令沿用 base schema，显式 `--schema-version` 只做一致性核对，不自动迁移 v1/v2；最终完整 asset 必须通过同一 strict validator。

schema v2 局部 revision 的每个被选 Page 使用与首次生成相同的当前 Page 上下文，并可额外提供 current localized description 与独立 reviewer 的 finding/defect description。current description 和 finding 只用于定位并修复语言质量问题；它们不是 semantic-scope authority，reviewer finding 也不得授权增加 canonical description 之外的语义。明确 subset 可在同一 generation session/batch 处理，但每页仍必须保持独立 semantic scope 和 Page identity。生成 session 只写上述 strict description 输入，正式 JSON 仍只由 `course-metadata revise` 机械更新并记录真实 provenance；发现缺陷的 reviewer session 不得生成 replacement 后再审核自己的修复，之后仍由独立 Locale Surface Review session 重审受影响范围。

## v2 stale graph

v2 只有全部 identity 当前时才合法：

- English Page source 改变 → canonical source description stale → source review gate stale → dependent v2 metadata 不得 current；
- canonical English description 改变 → 对应 `source_description_sha256` 改变 → 对应 locale Page stale；
- canonical locale target 改变 → 对应 `target_sha256` stale；
- locale glossary 任意字节改变 → 该 locale 全部 `glossary_sha256` stale；
- Page set/order、route 或 catalog identity 不一致 → fail closed；
- schema、generator contract 或 prompt version 不受支持 → fail closed。

source review gate 过期时不能执行 v2 assemble/refresh/revise，且任何 v2 loader/consumer 都会 fail closed。工具不伪造 generation provenance。

## Locale Surface Review 与 runtime

schema v2 审核包按 Catalog 顺序为每页包含：完整 English Page source、canonical English description、完整最终 canonical locale target、完整 glossary、localized description，以及 source/source-description/target/glossary identity。审核者必须判断本地化是否忠实于 canonical description、canonical description 是否与完整 source 对齐、localized description 是否与实际 target 一致、术语是否正确、表达是否自然，以及是否存在 unsupported expansion、generic/duplicate 或 route 错配。

schema v2 的 Locale Surface Review A freshness 额外绑定 canonical source-description artifact 与当前 source-review authority；任一改变都会 stale。schema v1 的 Surface Review input/freshness 不绑定这些新全局输入。

所有正式消费者继续只通过 strict `LoadCourseMetadata` 取得相同 runtime semantic surface：course route → 目标语言 description。projection、preview、publish、prerender 和 production 不需要知道模型过程；v1/v2 malformed、incomplete 或 stale 均 fail closed，不新增 English、正文截取或 runtime fallback。
