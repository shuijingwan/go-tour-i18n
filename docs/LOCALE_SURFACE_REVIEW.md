# Locale Surface Review 规范

Locale Surface Review 是 TranslationUnit 之外的 locale-level 发布质量审核，由“Locale-level language quality review”和“Rendered surface acceptance”两个不可互相替代的阶段组成。前者完整比较可翻译资产的 source 与 target，后者在完整 preview 和 production 上检查组合后的页面。两阶段共同构成独立的 locale release gate。

## 与 Translation Quality Review 的边界

Translation Quality Review 继续只对 translation workflow 中的每个 TranslationUnit 执行 automatic validation、Candidate Snapshot、ChatGPT Quality Check、Final Review 和 promotion。现有全 A、逐 TranslationUnit、Final Review A-only 与 `approved` 规则完全不变。

Surface Review：

- 不给 TranslationUnit 评级；
- 不生成或替代 `review/*.json`；
- 不批准 promotion；
- 不允许用页面抽查替代完整 TranslationUnit Quality Check 或 Final Review；
- 在 promotion 后的完整 locale 组合产物上阻止不合格 release 进入 publish 或 production。

两个 gate 必须都通过。若 Surface Review 发现 candidate 本身的误译、漏译或术语问题，必须创建 revision batch，并重新走 automatic validation、Quality Check、Final Review 和 promotion；不得直接改 canonical candidate 来绕过 A-only gate。

完整 locale preview（`tour-i18n preview --locale <locale>`）是用于本阶段的 production-style Surface Review preview：它使用 HTTP Playground transport、正式 public canonical identity、robots 与 sitemap，但不注入 production analytics 或广告。带 `--id` 的 candidate preview 保持本地 SocketTransport 开发语义，不替代完整 locale preview。

完整 locale preview 在启动前会 fail closed：必须存在当前的 machine-readable A gate。它不解析人工 Markdown；缺失 gate 时先完成 A，gate 的正式输入发生变化时必须重新审核并重新记录。带 `--id` 的 candidate/development preview 不受此限制。

## 正式术语基线

`locales/<locale>/glossary.yaml` 是该 locale 全站的正式术语权威来源。审核 TranslationUnit 之外的公共 UI、首页、列表、metadata、runtime message 与 SEO 文案时，同样检查 glossary 的 `mandatory`、`preferred`、`forbidden` 和 `keep` 决策。

品牌或产品名称必须有 locale 显式决策。例如 `Go Playground` 可以由某 locale 翻译，也可以 keep；本规范不要求不同 locale 字面相同，但要求同一 locale 在正文、按钮附近说明、runtime message、首页和 metadata 中一致。未写入 glossary 的争议术语应先完成术语决策，不能由各页面临时决定。

Glossary 只回答正式术语选择，不能证明完整译文忠实、准确或自然，也不能替代 source ↔ target 比较。审核者必须同时读取 source、target 和当前 glossary。

## A. Locale-level language quality review

这是对 TranslationUnit 之外全部可翻译资产的完整语言质量审核。首次新增 locale 时必须逐项覆盖正式资产集合，不允许只浏览几个页面、只搜索英文残留或只依靠浏览器抽查。

### 正式审核输入

公共 UI 至少使用以下三项不可缺少的输入：

- `internal/tour/ui/en.json`：公共 UI 的英文 source 与 message identity；
- `internal/tour/ui/<locale>.json`：待审核的完整 target catalog；
- `locales/<locale>/glossary.yaml`：该 locale 的正式术语权威来源。

必须按 message key 完整比较 source 与 target，同时核对 `plain` / `rich` kind、占位符和 markup identity。Catalog loader 的结构校验不能替代语言质量判断。

其他 locale-level 资产也必须结合其对应英文或 source identity 审核：

- `locales/<locale>/article-metadata.json` 对照当前正式 article 集合及各 article 的英文根级 `title`、`subtitle`；
- `locales/<locale>/course-metadata.json` 按 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md) 对照当前 catalog 的每个完整 Page source、完整最终 canonical target 与当前 glossary，逐页审核 description；
- 首页、导航、语言选择器、`/tour/list` 和 runtime message 对照其英文 catalog key、模板或稳定项目配置中的 source identity；
- SEO 可见文案对照生成它的英文/source 字段，并核对 locale、hostname 与页面身份。

如果无法确定某段 target 的 source identity，该项不能判定通过；必须先定位权威 source，不能凭目标页面猜测原意。

### 语言质量标准

对完整资产集合逐项检查：

- 忠实表达 source 的全部含义，无误译、漏译或语义弱化；
- Go 和项目相关技术含义准确；
- 遵守 glossary，且同一 locale 的术语和动作名称全站一致；
- 表达自然，符合目标语言的 UI、教学和 metadata 语境；
- 除 glossary `keep`、技术标识和其他明确保留项外，无未处理的英文残留；
- 不加入 source 没有支持的承诺、解释、品牌关系或其他无依据扩写；
- 富文本、变量、链接 label 和相邻文案组合后语义完整。

Glossary 一致但忠实度、准确性或自然度不合格时，语言质量审核仍然失败。

### A gate 记录与 freshness

人工 A 通过后，保留正式 Markdown evidence `data/locale-surface-reviews/<locale>/<review-id>.md`，然后由命令记录同一 review identity 的 machine-readable receipt：

```sh
go run -mod=readonly ./cmd/tour-i18n surface-review record-a \
  --locale <locale> \
  --review-id <review-id> \
  --reviewer <reviewer>
```

命令写入 `data/locale-surface-reviews/<locale>/<review-id>.a-gate.json`。其 schema 固定包含 `schema_version`、`locale`、`review_id`、`stage = locale-level-language-quality-review`、`decision = passed`、`reviewer` 和程序自动计算的 `inputs`；审核者不填写 SHA。当前实现的 inputs 覆盖英文与 locale UI catalog、locale glossary、article metadata、course metadata、完整当前 catalog/source identity，以及 `internal/tour/languages.go`、`project.go`、`seo.go` 和 `production/identity.json` 中会影响语言选择器、首页/导航项目文案和 runtime/SEO locale identity 的稳定 build-time 配置。

`preview --locale <locale>` 只接受 locale 正确、passed、schema/stage 完整且全部 inputs 与当前仓库一致的 gate。缺失、malformed、wrong-locale、non-passed 或任何 input 不一致均 fail closed；后者明确视为 stale language review evidence/gate。不要手写 JSON 或复用永久 `.passed` marker。

## B. Rendered surface acceptance

只有完整语言质量审核通过、已记录 current A gate，并且 TranslationUnit promotion、locale 配置、UI catalog 与 article metadata 已组成完整 projection 后，才在 preview 上执行本阶段；部署后再在 production 复核公网相关项目。浏览器检查用于发现组合、布局、交互、runtime 和部署问题，不能替代 A 阶段对全部可翻译资产的 source ↔ target 审核。

广告不属于 Locale-level language quality review，也不扩展为 TranslationUnit 或语言质量 gate。preview rendered surface acceptance 不要求真实 AdSense；首次 production 的最终 rendered surface acceptance 由 production browser automation 检查 loader、course-ad mount、请求机会、layout 与 SPA，filled/unfilled 均允许，再以极小 visual HUMAN gate 确认整体观感。具体边界见[生产运维手册](PRODUCTION_RUNBOOK.md)。本规范不重复广告实现、共享回归或广告失败隔离测试细节。

### Automated preview acceptance 与 visual HUMAN gate

完整 locale preview 必须监听带显式端口的本机 loopback HTTP origin。命令打印实际 URL 后运行正式机器入口：

```sh
go run -mod=readonly ./cmd/tour-i18n preview \
  --locale <locale> \
  --http 127.0.0.1:0

scripts/verify-preview-browser.py http://127.0.0.1:<port>/ <locale>
```

脚本从 `production/identity.json`、`data/tour-pages.tsv` 与 `internal/tour/languages.go` 读取权威 identity，fail closed 验证 preview loopback identity、正式 production canonical、关键 HTTP/SEO route、robots、完整 sitemap 及其全部本地映射、language selector、desktop/mobile rendered geometry、Run / Format / Reset、same-origin `/_/fmt` 与 `/_/compile`、SPA canonical/DOM 更新和 `/socket` 404。preview 不注入 production advertising 或 analytics；该入口不执行 AdSense、course-ad、production Playground Origin、CDN、TLS 或其他 production-only gate。

Canonical 分为不可混淆的三层：preview 原始 HTTP `GET /tour/` 返回的 shell 必须 self-canonical 为正式 production origin 下的 `/tour/`，`GET /tour/list` 同理使用 `/tour/list`，并且后者必须使用 locale UI catalog 中正式的 list title 与 description；preview 不加载 production prerender，因此原始 course route 返回 canonical 为 `/tour/` 的通用 Angular shell，不冒充 course-specific HTML。Angular 渲染后，`/tour/` 按唯一正式 route contract 跳转到 `/tour/welcome/1`，最终 pathname、`data-tour-rendered-route`、canonical、title 与正式 course description 必须精确对应该课程 route；`/tour/list` 必须保留一个、完整的 module/lesson directory，使用同一份 list title/description，且不重复正文。其他受检 course route 不允许意外跳转。Production raw course HTML 仍由独立 prerender gate 要求 course self-canonical 与正式 description；production raw `/tour/list` 同样必须由该 Chrome prerender 机制提供完整的 locale directory 正文、其自身 canonical、title 和 description，本 preview 规则不降低该 gate。

只有输出 `PREVIEW SURFACE ACCEPTANCE: PASS` 后才进入极小 **visual HUMAN gate**。人工只确认桌面与移动端整体排版观感正常，以及不存在自动 geometry 难以识别的视觉异常；不得再次人工检查 canonical、sitemap 数量、Run / Format / Reset、SPA、`/socket`、language selector URL 或脚本已覆盖的 desktop/mobile overflow。自动化不能替代 A 阶段语言质量审核，visual HUMAN gate 也不能替代自动验收。

### 1. 公共 UI 与页面 shell

- UI catalog 渲染时无英文 fallback、缺失文案、错误 rich markup 或占位符泄漏；
- header、footer、版权、项目链接、状态/提示、错误页及所有公共文案显示正确；
- 同一动作和概念在不同页面使用同一 glossary 决策；
- 文案没有被截断、遮挡、溢出，标点、空格、大小写与目标语言习惯一致。

### 2. 首页与课程入口

- `/` 的标题、项目介绍、状态、语言版本、链接说明、发布时间/开发态 metadata 和 footer 正确；
- `/tour/` 的首屏、课程标题、说明、开始/继续入口及特殊 welcome 页面正确；
- 首页与课程入口对项目身份、官方/社区属性和目标语言的表述一致。

### 3. `/tour/list`、导航与语言选择器

- `/tour/list` 的 locale-level heading、独立页面标题与 description、module/lesson 标题及说明、列表层级和链接目标正确；其 title/description 的正式 source 是英文 UI catalog `tour.list_title` / `tour.list_description`，不是 `course-metadata.json`；
- 上一页、下一页、目录/列表入口、页面计数和 footer 在首/中/末页面都正确；
- 语言选择器显示名称、顺序、当前语言状态与 URL 正确，不把其他语言的译文混入当前 locale；
- 键盘、鼠标和触摸操作的基本跳转可用。

language registry 是 build-time 输入，因此新增 locale 的首次 production Surface Review 只验收**新 locale → 已有 locale**：新站自身必须渲染当前正式 registry、正确标示 current language、链接已有 production locale，并保持 English 指向官方 Tour。**已有 locale → 新 locale** 采用 eventual consistency：旧 production release 无需在新 locale 上线当天立即显示反向链接，也不得因此单独重跑旧 locale 的 Quality Check、Final Review、Surface Review、publish、deploy、CDN purge 或 production final；旧 locale 会在下一次正常 release 时自然获得当前 registry。

### 4. 编辑器、动作与 runtime message

- 编辑器周边标签、帮助文案和按钮显示完整；
- Run、Format、Reset 的用词与 A 阶段审核通过的 target 一致；
- Run 返回的运行中、成功、程序退出、编译错误、网络错误等可见 runtime message 正确显示；
- Format 与 Reset 实际生效，状态反馈明确，不因文案长度破坏编辑器布局；
- preview 浏览器 Network 中 Run / Format 使用 same-origin `/_/compile` 与 `/_/fmt`，由 preview handler 代理到真实 Playground；production 浏览器继续使用正式 Playground endpoint，CORS Origin 为当前 locale 的精确 production origin；两者的 `/socket` 均不被启用。

### 5. SEO 与 production 可见身份

- HTML `<title>`、description、可见 heading、Open Graph/其他已实现的 SEO 字段与审核通过的 target 及当前页面一致；
- canonical/public host、`html lang`、首页和课程 URL 正确；`/`、`/tour/`、`/tour/list` 与每个 `/tour/<article>/<n>` 课程页均使用当前 locale production origin 下的自身正式 URL，不合并页面身份；
- `robots.txt` 指向当前 locale 的 sitemap；sitemap 包含当前正式集合，所有 URL 使用正确 host、无重复并返回预期状态；
- 页面不泄露 development、其他 locale hostname 或旧域名。

### 6. 桌面与移动端基本检查

- 至少在一个桌面 viewport 和一个移动 viewport 检查 `/`、`/tour/`、`/tour/list`、一个普通课程页和一个带编辑器的课程页；
- header、导航、语言选择器、正文、代码编辑器、按钮和 footer 可见且无重叠；
- 长译文可换行，横向滚动只出现在预期代码区域；
- 移动端长代码 / 长 `pre` 布局的固定回归页面为 `/tour/moretypes/1`，各 locale 只需替换 production hostname；已验证示例为 <https://ja-go-dev.shuijingwanwq.com/tour/moretypes/1>，用于确认长代码课程页不会横向撑破整个页面；
- 移动端能完成基本导航，并真实执行一次 Run、Format 和 Reset；
- 主题切换及已有的明暗模式图标/可见文本不因 locale 失效。

## 审核时点

每次新增 locale、公共 UI/首页/list/metadata/SEO 文案变更、glossary 决策影响表层、runtime message 变更，或上游同步改变可见页面组合时，都必须重审受影响范围。首次上线必须执行完整清单；日常维护可以按变更影响重审，但不得省略所有共享 shell 和导航的一致性检查。

## Evidence 与最终结论

每次正式 Surface Review 的最终 evidence 固定写入：

```text
data/locale-surface-reviews/<locale>/<review-id>.md
```

`review-id` 必须在该 locale 内唯一且可稳定引用，建议使用 `YYYYMMDD-<purpose-or-release>`。Evidence 是人工审核记录，不新增 JSON schema，也不冒充自动 validator 或 TranslationUnit review evidence。Markdown 至少记录：

- `locale`；
- `reviewed commit/release`：被审核 commit，以及 preview 或 production release identity；
- `glossary identity`：glossary path 与 SHA-256；
- `UI / metadata identity`：英文和目标 UI catalog、article metadata 及其他被审核 locale-level source/target 的 path 与 commit 或 SHA-256；
- `language quality review result`：完整范围、结果与问题；
- `preview acceptance result`：preview identity、环境、覆盖范围、结果与问题；
- `production verification result`：production release/URL、环境、覆盖范围、结果与问题；首次 production 同时记录按生产运维手册完成的轻量广告确认；
- `reviewer`；
- `date`；
- `decision = passed | failed`；
- `issues`。

首次上线时，A 阶段与 preview acceptance 都通过后才可进入 publish 和首次部署；此时可以先维护同一路径的工作记录。production 复核完成后，必须在该 evidence 中写入真实 production 结果和最终 `decision`，才能宣告正式上线。最终 evidence 不得把尚未执行的 production 复核写成通过。

Surface Review 只使用 `passed` 或 `failed`，不采用 TranslationUnit 的 A/B/C/D。任一必需阶段存在未解决阻塞项时，最终 decision 必须为 `failed`。

## 缺陷回流

- TranslationUnit candidate 缺陷：回到 revision batch，完成全套 A-only 链后重新 projection 和 Surface Review。
- glossary 决策缺失或冲突：先更新该 locale glossary，再同步受影响表层并重审。
- UI、首页、list、metadata 或 SEO 缺陷：修改相应 locale 资产，重建完整 preview，重审受影响范围。
- production-only 的 CDN、TLS、Origin、缓存或响应问题：按 [生产运维手册](PRODUCTION_RUNBOOK.md) 修复并在公网复核，不改写 TranslationUnit 审核结果。

只有 A 阶段与 preview acceptance 均通过时，首次 release 才能进入 publish / production；只有 production 复核也通过、最终 evidence 为 `decision = passed` 时，才能宣告 locale 正式上线。
