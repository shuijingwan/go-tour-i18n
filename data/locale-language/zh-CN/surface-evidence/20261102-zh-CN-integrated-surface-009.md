# Independent Integrated Locale Surface Review

- review_id: `20261102-zh-CN-integrated-surface-009`
- reviewer_session: `chatgpt-zh-CN-integrated-surface-reviewer`
- role: Independent Integrated Surface Reviewer
- provider: `chatgpt`
- model: `GPT-5.6 Sol High`
- formal_provenance: `gpt-5.6-sol-high`
- locale: `zh-CN`

## Input gate

- Bundle ZIP SHA-256: `8aadb6fefcbb5504fd176552c72e5e1dd421325b31255e961ee4b4b9b28ad8e9`
- Bundle bytes: `34490386`
- Bundle schema: `go-learning/integrated-surface-reviewer-bundle/v1`
- Lifecycle/scope schema: `go-learning/integrated-locale-surface/v1`
- Scope SHA-256: `236e7c167a0ccf3e3a45359a583b9a5e23331c6077dbf4504b1aa2a753558d97`
- Scope identity: `30e2595485f072fc2231aac479772a6b370a74066ef7457c570c66f1567697fc`
- Package closure 1 SHA-256: `464593c57c7d4b5490ca973f649cdf2c189040a124715940a7b794f94083ce9b`
- Package closure 2 SHA-256: `134a892e0fcf7edcf0b41fd557c623c427a4d99912e4d6296df3409f1092e35e`
- `review_required`: 14
- `carried`: 520
- Manifest inventory: 337/337 declared files present with exact byte size and SHA-256
- Fresh exact-set: 14/14 complete
- Runtime/public, canonical routes, coverage, glossary, Unified Review, structured/TU provenance and formal authority: present
- Reviewer independence: yes
- Semantic remote calls: 0
- Repository supplementation: none

`INPUT_GATE=PASS`

## Fresh review-required judgments

All 14 current canonical source and restored target payloads were read and judged fresh. The 520 carried assets were used only for integration/context and were not re-rated as language assets.

| Asset | Judgment | Notes |
|---|---|---|
| `_content/doc/database/cancel-operations.md` | **PASS** | No unresolved fresh language/integration defect. |
| `_content/doc/database/open-handle.md` | **PASS** | Current complete page remains semantically and linguistically acceptable. |
| `_content/doc/database/querying.md` | **PASS** | No unresolved fresh language/integration defect. |
| `_content/doc/index.html` | **FAILED** | TranslationUnit boundary finding recorded below. |
| `_content/doc/install.html` | **PASS** | Installation sequencing, platform conditions, commands and paths remain accurate. |
| `_content/doc/security/vuln/database.md` | **PASS** | Security/schema semantics and terminology remain acceptable. |
| `_content/doc/security/vuln/editor.md` | **PASS** | Editor/vulnerability semantics and technical identity remain acceptable. |
| `_content/doc/tutorial/add-a-test.html` | **PASS** | Current explanatory prose/code boundaries remain acceptable. |
| `_content/doc/tutorial/greetings-multiple-people.html` | **PASS** | Current terminology and explanatory prose remain acceptable. |
| `_content/doc/tutorial/handle-errors.html` | **PASS** | Current error/logging semantics and prose remain acceptable. |
| `_content/doc/tutorial/random-greeting.html` | **PASS** | Current random-selection semantics and prose remain acceptable. |
| `_content/index.md` | **PASS** | Homepage TranslationUnit itself has no separate fresh language defect; composition checked with shared shell/runtime context. |
| `_content/site.tmpl` | **FAILED** | Predecessor accessibility finding closed; separate punctuation/representation finding recorded below. |
| `data/site-shell/translation.html` | **FAILED** | Structured language boundary/residual finding recorded below. |

## Predecessor Surface008 finding closure

The prior `_content/site.tmpl` accessibility finding is closed in the current target:

- source: `Press Enter to activate/deactivate dropdown`
- current target: `按 Enter 键可激活或停用下拉菜单`

The current wording is complete, natural zh-CN, preserves `Enter`, and does not alter template semantics. This closure was independently verified from the current Bundle and was not carried automatically.

## Exact finding 1 — `_content/doc/index.html`

**Class:** `translation-unit-language`

**Category:** technical-identifier / Chinese-prose boundary

**Observed instances:**

1. `GOPATH</a>的`
2. `<code>bin</code>目录`

Canonical source expresses the phrase as `GOPATH's <code>bin</code> directory.`

**Exact predicate:** in the complete current `_content/doc/index.html` visible natural-language prose, a Latin technical identifier rendered as an inline link or inline code is directly adjacent to a Chinese lexical word without the locale's established separator.

**Bounded same-class scan:** complete current Page; exactly 2 affected boundaries.

**Expected correction boundary:** only those two boundaries. Preserve `GOPATH`, `bin`, hrefs, HTML structure, navigation/route semantics and unrelated prose.

**Resolution owner:** current TranslationUnit candidate; formal TU reopen → exact revision → validation → independent re-QC.

## Exact finding 2 — `_content/site.tmpl`

**Class:** `structured-surface-language`

**Category:** Chinese sentence-ending punctuation at structured representation boundary

The predecessor accessibility wording is fixed, but two other user-visible Chinese sentences currently end with ASCII `.`:

1. `<i>此内容属于 <a href="/wiki/">Go Wiki</a>.</i>`
2. `<i>此文档的源文件可在以下位置找到： <a href="https://cs.opensource.google/go/x/tools/+/master:gopls/doc/">golang.org/x/tools/gopls/doc</a>.</i>`

**Exact predicate:** in the complete current `_content/site.tmpl` user-visible natural-language surface, an otherwise Chinese sentence terminates with ASCII `.` immediately after a link.

**Bounded same-class scan:** complete current structured asset; exactly 2 instances.

**Expected correction boundary:** sentence-ending punctuation only. Preserve link destinations/labels, template/HTML structure, identifiers, all other shell language and the corrected `按 Enter 键可激活或停用下拉菜单`.

**Resolution owner:** structured shared shell asset; if the current representation does not expose the punctuation, use the formal shared representation successor / structured-language path. No TranslationUnit rewrite is authorized.

## Exact finding 3 — `data/site-shell/translation.html`

**Class:** `structured-surface-language`

### Category A — technical-name / Chinese-prose boundary

Observed locations:

1. `探索<a href="/tour/">Go 语言之旅</a>中的...`
2. `而不是go.dev上的所有页面`
3. `已通过<a href="https://github.com/golang/go/issues/81958">golang/go#81958</a>提交`

**Exact predicate:** within this complete structured asset, a Latin-leading named/technical identity or pure-Latin domain/issue identity is directly concatenated with adjacent Chinese lexical prose without the locale's established separator.

**Bounded same-class scan:** complete current `data/site-shell/translation.html`; exactly these 3 locations.

### Category B — untranslated ordinary natural-language fragment

Observed instance:

- source: `The issue remains open`
- current target: `该 issue 仍处于开放状态`

Here `issue` is sentence-level natural-language wording, not a required protected technical identity.

**Exact predicate:** ordinary English lexical prose remains verbatim inside otherwise zh-CN natural-language text, excluding technical identities, URLs, locale IDs and project names.

**Bounded same-class scan:** complete current structured asset; exactly 1 instance.

**Expected correction boundary:** only the 3 boundary locations and the 1 ordinary-English residual. Preserve hrefs, `Go`, `Go 本地化`, `go.dev`, `golang/go#81958`, locale IDs, page-count claims, disclaimers, HTML structure and unrelated prose.

## Integration/context judgment

- Current glossary / Unified Review consistency: PASS
- `Go local` → `Go 本地化`: PASS
- Homepage/shared shell composition: FAILED only for the exact structured finding above
- Learn / Docs integration: FAILED only for `_content/doc/index.html` language finding above
- Tour integration: PASS
- Current `Resolver.Destination` trailing-slash normalization: PASS
- `/doc/install/` → local `/doc/install`: PASS
- `/doc` → local `/doc/`: PASS
- four `/learn#...` links → local `/learn/#...` with fragments preserved: PASS
- explicit alias precedence: PASS
- uncovered `https://go.dev` fallback: PASS
- canonical route inventory / Handler coverage / sitemap non-expansion: PASS
- ads/runtime/public composition: no separate unresolved blocker found
- carried assets individually re-rated: no
- preview/browser/Production/HUMAN acceptance executed: no
- repository mutation: none

## Decision

`DECISION=failed`

The Surface008 `activate/deactivate` finding is closed, but the fresh current Surface has 3 exact owner findings described above.

`FINDING_COUNT=3`

`EXACT_FINDING_COUNT=3`

`STOP=before-repository-record-or-revision`
