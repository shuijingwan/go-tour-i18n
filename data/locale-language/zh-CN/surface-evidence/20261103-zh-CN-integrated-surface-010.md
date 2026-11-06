# Independent Integrated Locale Surface Review

- review_id: `20261103-zh-CN-integrated-surface-010`
- reviewer_session: `chatgpt-zh-CN-integrated-surface-reviewer`
- role: Independent Integrated Surface Reviewer
- provider: `chatgpt`
- model: `GPT-5.6 Sol High`
- formal_provenance: `gpt-5.6-sol-high`
- locale: `zh-CN`

## Input gate

This review used only the uploaded CURRENT Reviewer Bundle.

- Bundle ZIP SHA-256: `183968128c3df7a77cb92cfa8ba157cf8889612f2ecd9b59ebde90a461bc30cd`
- Bundle bytes: `34492834`
- Bundle manifest identity: `595db0315d270644fa68df2acbb9215444af9c7e1898a40fd83f3c2a1b01b5ba`
- Bundle schema: `go-learning/integrated-surface-reviewer-bundle/v1`
- Lifecycle/scope schema: `go-learning/integrated-locale-surface/v1`
- Scope SHA-256: `91136490e5ace23d1af9b82590f1930e9cf9b37569719d357d8b05b22ed18cea`
- Scope identity: `8db728da413788912ff8762b6ddace1e4e750df65bbed06381c3d6824869912d`
- `review_required`: 14
- `carried`: 520
- Manifest inventory: 337/337 declared files present with exact size and SHA-256
- Learn/Docs closure SHA-256: `464593c57c7d4b5490ca973f649cdf2c189040a124715940a7b794f94083ce9b`
- Site v2 shell closure SHA-256: `134a892e0fcf7edcf0b41fd557c623c427a4d99912e4d6296df3409f1092e35e`
- Current glossary / Unified Review / runtime-public / canonical / coverage / provenance / authority inputs: present
- Result template scope: exact-equal to Bundle `integrated-surface-scope.json`
- Semantic remote calls: 0
- Repository supplementation/mutation: none

`INPUT_GATE=PASS`

Surface009 was treated only as historical predecessor context. No Surface009 decision or finding was automatically carried into this judgment.

## Fresh review-required judgments

All 14 current canonical source/restored-target pairs were read in full. The 520 carried assets were used only for integration/context and were not re-rated as language assets.

| Fresh asset | Judgment | Notes |
|---|---|---|
| `_content/doc/database/cancel-operations.md` | **FAILED** | Exact TU finding recorded. |
| `_content/doc/database/open-handle.md` | **PASS** | Complete current Page acceptable; prior paragraph-break repair remains sound. |
| `_content/doc/database/querying.md` | **PASS** | Complete current Page acceptable. |
| `_content/doc/index.html` | **FAILED** | Exact TU boundary finding recorded. |
| `_content/doc/install.html` | **FAILED** | Exact Windows UI-label finding recorded. |
| `_content/doc/security/vuln/database.md` | **FAILED** | Exact TU boundary finding recorded. |
| `_content/doc/security/vuln/editor.md` | **PASS** | Complete current Page acceptable; protected command/UI identities remain semantically correct. |
| `_content/doc/tutorial/add-a-test.html` | **FAILED** | Exact punctuation finding recorded. |
| `_content/doc/tutorial/greetings-multiple-people.html` | **PASS** | Complete current Page acceptable. |
| `_content/doc/tutorial/handle-errors.html` | **PASS** | Complete current Page acceptable; no new blocker established. |
| `_content/doc/tutorial/random-greeting.html` | **PASS** | Complete current Page acceptable. |
| `_content/index.md` | **PASS (page target) / FAILED (composition)** | Homepage TU text is acceptable; mandatory shared composition exposes structured findings owned by `_content/site.tmpl` and `_content/testimonials.yaml`. |
| `_content/site.tmpl` | **FAILED** | Exact shared structured finding recorded; predecessor accessibility text itself is fixed. |
| `data/site-shell/translation.html` | **FAILED** | Exact structured-language finding recorded. |

## Current integration/context judgment

- Current Unified Glossary consistency, including `Go local` → `Go 本地化`: PASS
- Tour / Homepage / Learn / Docs cross-surface terminology: PASS except exact language findings below
- Current trailing-slash navigation normalization: PASS
- `/doc/install/` resolves to existing local `/doc/install`: PASS
- `/doc` resolves to existing local `/doc/`: PASS
- four `/learn#...` links resolve to existing local `/learn/` with fragments preserved: PASS
- exact local route behavior: PASS
- explicit alias precedence: PASS
- locale-uncovered/global-canonical fallback to `https://go.dev`: PASS
- Handler coverage expansion: none
- canonical inventory expansion: none
- sitemap expansion: none
- query/fragment preservation: PASS
- runtime rewrites `<html lang>` to active locale and canonical/`og:url` to local canonical URL: PASS
- ads/runtime/public composition: no separate engineering blocker found
- carried assets individually re-rated: no
- preview/browser/Production/HUMAN acceptance executed: no

The current Integrated Surface is nevertheless not language-clean because of the exact owner findings below.

## Exact finding 1 — `_content/doc/database/cancel-operations.md`

**Class:** `translation-unit-language`

**Summary:** Current Page contains 3 technical-identifier/Chinese-prose boundary defects plus 1 spurious Markdown paragraph break.

**Finding / exact predicate / bounded scope**

Fresh complete-Page review found two bounded defect categories. (1) technical-identifier/Chinese-prose boundary: exact observed boundaries are `[context.Context](...)来`, `` `Context`是一种 ``, and `[context.WithTimeout](...)或`; same-class scan of the complete current Page found exactly these 3 affected boundaries. (2) spurious Markdown paragraph break: the sentence ending `有关 `Context` 的更多信息，请参阅` is separated by a blank paragraph boundary from `[Go 并发模式：Context](/blog/context)。`, splitting one canonical continuous sentence and leaving the first paragraph syntactically incomplete; same-class scan found exactly 1 instance. Revision scope is only these 4 observed instances. Preserve `context.Context`, `Context`, `context.WithTimeout`, `context.WithDeadline`, link destinations, code blocks, API identities, cancellation semantics, and unrelated prose.
## Exact finding 2 — `_content/doc/index.html`

**Class:** `translation-unit-language`

**Summary:** Current Docs hub has 2 technical-identifier/Chinese-prose boundary defects in the GOPATH/bin installation paragraph.

**Finding / exact predicate / bounded scope**

Exact observed boundaries: `GOPATH</a>的` and `<code>bin</code>目录`. Exact predicate: within the complete current `_content/doc/index.html` visible natural-language prose, a Latin technical identifier rendered as an inline link/code span is directly adjacent to a Chinese lexical word without the locale's established separator. Bounded same-class scan found exactly 2 instances, both in the Tour local-installation paragraph. Revision scope is only those boundaries; preserve `GOPATH`, `bin`, hrefs, HTML structure, navigation semantics, and unrelated prose.
## Exact finding 3 — `_content/doc/install.html`

**Class:** `translation-unit-language`

**Summary:** Windows verification step contains one untranslated/inconsistent natural-language UI label: `Command Prompt`.

**Finding / exact predicate / bounded scope**

Exact location: Windows install verification step, current target `在出现的 Command Prompt 窗口中输入以下命令`. The same Page elsewhere consistently uses `命令提示符`, while canonical source says `In the Command Prompt window that appears`. The source representation protects `Prompt` as an atom, so the current visible residual may require formal representation successor authority before exact TU revision; that machine-resolution choice is downstream, not Reviewer work. Exact predicate: ordinary operating-system UI terminology remains English inside otherwise zh-CN natural-language instructions where the same Page already establishes a Chinese equivalent. Complete-Page same-class scan found exactly 1 instance. Preserve Windows, `cmd`, Enter, commands, paths, HTML structure, and installation order/semantics.
## Exact finding 4 — `_content/doc/security/vuln/database.md`

**Class:** `translation-unit-language`

**Summary:** Current vulnerability database Page has 2 technical-term/Chinese-prose boundary defects around `schema`/`Schema` links.

**Finding / exact predicate / bounded scope**

Exact instances: `[Open Source Vulnerability（OSV）schema](...)提供 Go 漏洞信息` and `见下文的[Schema](#schema)`. The finding does not require translating the technical term `schema`; it concerns the missing zh-CN prose boundary around the linked Latin term. Exact predicate: a link whose visible edge is a Latin technical term directly touches an adjacent Chinese lexical word. Complete-Page same-class scan found exactly 2 instances. Revision scope is boundary spacing only; preserve OSV, Schema/schema terminology choice, links, JSON/API identities, code blocks, security semantics, and unrelated prose.
## Exact finding 5 — `_content/doc/tutorial/add-a-test.html`

**Class:** `translation-unit-language`

**Summary:** Current tutorial Page retains one source-style ASCII double-hyphen as Chinese sentence punctuation.

**Finding / exact predicate / bounded scope**

Exact visible instance: `应该失败 -- <code>TestHelloEmpty</code> 则仍然会通过。` Canonical source uses `should fail -- ... still passes`; the double hyphen is ordinary source punctuation, not a protected technical identity, and is unnatural in otherwise zh-CN prose. Exact predicate: ASCII ` -- ` is used as an inter-clause sentence dash in localized natural-language prose outside literal/preformatted code. Complete-Page same-class scan found exactly 1 instance. Revision scope is that punctuation only; preserve test identifiers, code/output blocks, links, and test semantics.
## Exact finding 6 — `_content/site.tmpl`

**Class:** `structured-surface-language`

**Summary:** Current shared template has one OpenGraph-title English metadata residual and 2 Chinese sentence-ending ASCII-period defects.

**Finding / exact predicate / bounded scope**

Fresh complete-asset review independently confirms the prior `activate/deactivate` accessibility defect is closed (`按 Enter 键可激活或停用下拉菜单`). Separate current defects remain. (1) metadata residual: `<meta name="og:title" ...>` still emits the English suffix ` - The Go Programming Language` for non-root shell-rendered local routes, while the adjacent document `<title>` is localized to ` - Go 编程语言`; same-class scan of user-facing/social metadata found exactly 1 untranslated natural-language metadata expression. (2) sentence punctuation: the two conditional user-visible Chinese notes end with ASCII period after a link: `此内容属于 <a ...>Go Wiki</a>.` and `此文档的源文件可在以下位置找到： <a ...>golang.org/x/tools/gopls/doc</a>.`; same-class scan found exactly 2 instances. The runtime correctly rewrites `<html lang>` to the active locale and canonical/og:url to the current canonical URL, so those are not findings. Correction boundary is only the metadata natural-language residual and the two sentence-ending punctuation instances. Preserve template/HTML control flow, links, identifiers, canonical/runtime behavior, and the already-correct accessibility instruction. If current structured representation does not expose the metadata/punctuation, downstream must use the formal shared representation successor path.
## Exact finding 7 — `data/site-shell/translation.html`

**Class:** `structured-surface-language`

**Summary:** Current project/About asset has 3 mixed technical-name boundary locations, one ordinary-English `issue` residual, and source ampersands retained in 2 Chinese headings.

**Finding / exact predicate / bounded scope**

Category A, technical-name/Chinese-prose boundary: 3 exact locations contain 5 affected boundary edges: `探索<a ...>Go 语言之旅</a>...` (Chinese→Latin edge), `不是go.dev上的所有页面` (Chinese→Latin and Latin→Chinese edges), and `已通过<a ...>golang/go#81958</a>提交` (Chinese→Latin and Latin→Chinese edges). Category B, untranslated ordinary natural-language fragment: exact instance `该 issue 仍处于开放状态`; here `issue` is sentence-level ordinary prose referring to the already identified GitHub issue, not a required technical identity. Category C, source punctuation retained in localized natural-language headings: exact instances `Go 学习 & 文档多语言本地化项目` and `支持 & 运营成本`; the source representation protects the `&amp;` atoms, so a formal structured representation successor may be required before correction. Complete-asset scans found exactly these 3 boundary locations / 5 boundary edges, 1 ordinary-English residual, and 2 protected ampersand heading instances. Preserve all hrefs, `Go`, `Go 本地化`, `go.dev`, `golang/go#81958`, locale IDs, page-count claims, disclaimers, HTML structure, and unrelated wording.
## Exact finding 8 — `_content/testimonials.yaml`

**Class:** `structured-surface-language`

**Summary:** Fresh Homepage composition exposes one carried shared-testimonial punctuation defect: `Kubernetes.` ends a Chinese quotation with an ASCII period.

**Finding / exact predicate / bounded scope**

This does not re-rate the carried structured asset. The fresh Homepage `_content/index.md` renders current `_content/testimonials.yaml` through `data "/testimonials.yaml"`, making the defect part of the current Integrated Surface. Exact observed instance is the Curve testimonial ending `例如 Kubernetes.”`; the localized Chinese sentence uses ASCII `.` immediately before the Chinese closing quotation mark. Exact predicate: within the mandatory Homepage shared testimonials composition, a localized Chinese quotation terminates with ASCII sentence period after a Latin proper name. Bounded same-class scan of the complete current `_content/testimonials.yaml` target found exactly 1 instance. Correction boundary is punctuation only; preserve `Kubernetes`, testimonial attribution/content, YAML structure, markup, and unrelated testimonials. Owner remains the registered shared structured path `_content/testimonials.yaml`; no Homepage TU or carried TU is re-rated.


## Same-class completeness summary

- `technical-identifier/technical-term ↔ Chinese prose boundary`: bounded complete scans were performed within each affected TU/structured asset; counts are specified per finding.
- `spurious Markdown paragraph break`: complete `_content/doc/database/cancel-operations.md` scan found exactly 1.
- `ordinary-English natural-language residual`: complete bounded scans found exactly the `Command Prompt`, `issue`, and OpenGraph title residuals described above; technical identities and accepted proper names were excluded.
- `localized punctuation`: complete bounded scans found exactly the ` -- ` tutorial instance, 2 site-template ASCII sentence periods, 2 protected ampersand headings in the project/About asset, and 1 Homepage testimonial ASCII period described above.
- No finding authorizes whole-page rewriting, unrelated stylistic cleanup, glossary changes, runtime changes, or carried TU re-rating.

## Decision

`INPUT_GATE=PASS`

`FRESH_REVIEWED=14/14`

`CARRIED_CONTEXT=520`

`DECISION=failed`

`FINDING_COUNT=8`

`EXACT_FINDING_COUNT=8`

`REPOSITORY_MUTATION=none`

`STOP=before-surface-record`
