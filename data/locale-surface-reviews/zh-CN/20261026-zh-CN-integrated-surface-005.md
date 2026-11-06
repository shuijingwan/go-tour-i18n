本轮 `20261026-zh-CN-integrated-surface-005` 已完成完整 independent successor review。

**INPUT_GATE=PASS，但语言结论为 FAILED。** 13 个 fresh `review_required` asset 全部完成独立 judgment；521 个 carried asset 仅用于 integration context，没有重新评级。最终为 **A=6、B=7、C=0、D=0**。

发现的问题均为局部、可界定的发布质量缺陷，没有发现需要 C/D 的语义破坏。当前 `Go local → Go 本地化` 在 fresh `/translation/` 与 carried Tour context 之间保持一致，**不是本轮失败原因**。

### 13 个 fresh asset 结果

| # | Fresh asset | Rating |
|---|---|---:|
| 1 | `_content/doc/database/cancel-operations.md` | **A** |
| 2 | `_content/doc/database/open-handle.md` | **B** |
| 3 | `_content/doc/database/querying.md` | **A** |
| 4 | `_content/doc/index.html` | **B** |
| 5 | `_content/doc/install.html` | **A** |
| 6 | `_content/doc/security/vuln/database.md` | **A** |
| 7 | `_content/doc/security/vuln/editor.md` | **A** |
| 8 | `_content/doc/tutorial/add-a-test.html` | **B** |
| 9 | `_content/doc/tutorial/greetings-multiple-people.html` | **B** |
| 10 | `_content/doc/tutorial/handle-errors.html` | **B** |
| 11 | `_content/doc/tutorial/random-greeting.html` | **B** |
| 12 | `_content/index.md` | **A** |
| 13 | `data/site-shell/translation.html` | **B** |

下面是全部 blocking findings。每个 category 都已在对应完整 fresh asset 内完成 bounded same-class scan。

# Independent Integrated Surface Review

review_id: `20261026-zh-CN-integrated-surface-005`

locale: `zh-CN`

role: `Independent Integrated Surface Reviewer`

provider: `chatgpt`

model: `GPT-5.6 Sol High`

formal_provenance: `gpt-5.6-sol-high`

## Input Gate

BUNDLE_SCHEMA=`go-learning/integrated-surface-reviewer-bundle/v1`

BUNDLE_SHA256=`18761e9004908fd660fed28be1fb592d991a620d7d11f76a6bb8fc9ad26c9769`

BUNDLE_BYTES=`34200483`

SCOPE=`zh-CN-integrated-surface-scope-9ecc0537191d9964.json`

SCOPE_SHA256=`fbb83a4c30209d97c6a1bbc9b27d69c2f93797b5d710bd4cc16b89bd234e0a07`

SCOPE_IDENTITY=`9ecc0537191d99645599a27da929b675708f161f9bde7f761e98fa7ecc1656e2`

REVIEW_REQUIRED=`13`

CARRIED=`521`

CANONICAL_ROUTES=`158`

INPUT_GATE=`PASS`

All manifest inventory paths, byte sizes, and SHA-256 values matched the frozen Bundle.

---

## Fresh asset 1

ASSET=`_content/doc/database/cancel-operations.md`

UNIT_ID=`d799614a9f1efd848505c615a88f2fb0f092ac22f743b3da042e12de1127fd8b`

RATING=`A`

FINDING=`none`

The complete current source and target preserve cancellation, `context.Context`, timeout/deadline propagation, `QueryContext`, resource cleanup, and HTTP request context semantics. No fresh Integrated Surface blocker found.

---

## Fresh asset 2

ASSET=`_content/doc/database/open-handle.md`

UNIT_ID=`872da6c618956d7ff25dbe8db7d898808b6a78fc1b91244ffcda0f59ce2b1983`

RATING=`B`

DECISION=`rejected`

### Finding 2.1

Finding category:
`representation-protected-natural-language-residue`

Finding owner:
`translation-unit-language / representation`

Observed instances:

1. In the driver-specific connection section:
   `FormatDSN method`

2. In the `sql.OpenDB` section:
   `sql.OpenDB function`

Both appear verbatim in the current zh-CN visible target.

The natural-language descriptors `method` and `function` are not technical identifiers that should remain English. The current source representation protects the entire inline atoms `` `FormatDSN method` `` and `` `sql.OpenDB function` ``, which explains the machine-valid result but still creates a user-visible localization defect.

Same-class scan result:
Within this complete current Page, these are the two confirmed instances of this exact category.

Revision scope:
Only the visible natural-language descriptors contained in these two inline link labels, while preserving `FormatDSN`, `sql.OpenDB`, link destinations, Markdown binding, and technical identity.

Out of scope:
Other prose, API names, DSN semantics, driver behavior, links, code, and general stylistic rewriting.

---

## Fresh asset 3

ASSET=`_content/doc/database/querying.md`

UNIT_ID=`21946bff61d8f7c52c444edcd122ea398ba80574a8095c37befa936541a9cd4c`

RATING=`A`

FINDING=`none`

The complete current target preserves single-row/multi-row query semantics, error handling, nullable values, conversion rules, reserved connections, multiple result sets, and current glossary usage. No fresh blocker found.

---

## Fresh asset 4

ASSET=`_content/doc/index.html`

UNIT_ID=`00a849ced4775e5dc75d72ecbdc81549135697ba84c51d26d05ac84d4989b866`

ROUTE=`/doc/`

RATING=`B`

DECISION=`rejected`

### Finding 4.1

Finding category:
`technical-identifier-chinese-prose-boundary`

Finding owner:
`translation-unit-language`

Observed instances:

In the installation/navigation prose describing where the `tour` binary is installed:

- `GOPATH</a>的`
- `<code>bin</code>目录`

Rendered, these produce the equivalent of:

`GOPATH的 bin目录中`

The technical identifiers are directly attached to surrounding Chinese words, unlike the otherwise established boundary treatment in the current locale.

Same-class scan result:
The complete current Page was scanned for this category. These two adjacent boundaries are the confirmed residual instances.

Exact predicate:
Within this Page's visible natural-language prose, a Latin technical identifier or inline-code identifier immediately touches an adjacent Chinese lexical word without the separator required by the locale's established technical prose convention.

Revision scope:
Only these two identifier/Chinese boundaries.

Out of scope:
`GOPATH`, `bin`, href values, command semantics, navigation structure, other titles/prose, and general polishing.

---

## Fresh asset 5

ASSET=`_content/doc/install.html`

UNIT_ID=`4b6d5c2294accb01dad0184b82adbf114a8a74186199fff44f115c880b2d96c7`

ROUTE=`/doc/install`

RATING=`A`

FINDING=`none`

Linux, macOS, and Windows installation sequencing, old-installation removal, archive/MSI behavior, PATH handling, commands, paths, and verification semantics remain accurate.

---

## Fresh asset 6

ASSET=`_content/doc/security/vuln/database.md`

UNIT_ID=`a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f`

RATING=`A`

FINDING=`none`

The complete security/vulnerability database Page preserves OSV schema semantics, identifiers, exclusion reasons including `NOT_IMPORTABLE`, URLs, JSON structure, and current terminology.

---

## Fresh asset 7

ASSET=`_content/doc/security/vuln/editor.md`

UNIT_ID=`8eb5e9ea7af00232ebd029ae5dd0682a2c956167f69d073d0bf16de21668e689`

RATING=`A`

FINDING=`none`

Editor vulnerability workflow, `govulncheck`, gopls/editor command identities, diagnostic behavior, code lens terminology, and security meaning remain accurate.

---

## Fresh asset 8

ASSET=`_content/doc/tutorial/add-a-test.html`

UNIT_ID=`393fb67915a49876749590fd6480c9a3482597a9cebdd61e72c9a5f3d1449e2c`

RATING=`B`

DECISION=`rejected`

### Finding 8.1

Finding category:
`technical-identifier-chinese-prose-boundary`

Finding owner:
`translation-unit-language`

Observed same-class locations include the visible prose surrounding:

- `Hello` → `函数`
- `go test` → `命令`
- `TestHelloName` → `调用`
- `Errorf` → `方法`
- `TestHelloEmpty` → `调用`
- `_test.go` → `结尾`
- `TestHelloName` → `测试函数`

Some identifiers occur more than once in the Page.

Exact predicate:
Within the complete Page's visible explanatory prose, excluding literal code/preformatted program bodies, a protected Go identifier, command, or filename suffix is directly concatenated with a following Chinese lexical word without the established separator.

Revision scope:
Complete same-class sweep of this Page under the exact predicate above.

Out of scope:
Go program bytes, identifiers themselves, test behavior, links, HTML structure, teaching content meaning, and unrelated punctuation/style.

---

## Fresh asset 9

ASSET=`_content/doc/tutorial/greetings-multiple-people.html`

UNIT_ID=`b49a0f38f79787a87a714c55b10e2987434ebdcd78c2c1cc0480e46523893493`

RATING=`B`

DECISION=`rejected`

### Finding 9.1

Finding category:
`technical-identifier-chinese-prose-boundary`

Finding owner:
`translation-unit-language`

Observed same-class locations include:

- `string` → `改为`
- `Hello` → `函数`
- `Hellos` → `函数`
- `for` → `循环`
- `range` → `会`

`Hellos` occurs at more than one relevant prose location.

Exact predicate:
Within this Page's visible explanatory prose outside program bodies, an inline Go type/identifier/keyword is immediately concatenated with an adjacent Chinese lexical word.

Revision scope:
Complete current Page, exact predicate only.

Out of scope:
Identifier spelling, example code, map/range semantics, links, and general rewriting.

### Finding 9.2

Finding category:
`untranslated-natural-language-fragment`

Finding owner:
`translation-unit-language`

Observed instance:
The closing summary currently says:

`使用映射表示 name/value 对`

Canonical source meaning is ordinary natural-language `name/value pairs`; `name/value` here is not a required technical identifier or glossary keep term.

Same-class scan result:
No second unambiguous instance of this exact ordinary-English-fragment category was found in the complete current Page. Code-variable references such as `names` / `messages` were not broadened into this finding.

Revision scope:
Exact closing-summary fragment only.

Out of scope:
Code variables, map terminology, examples, identifiers, and other English technical identities.

---

## Fresh asset 10

ASSET=`_content/doc/tutorial/handle-errors.html`

UNIT_ID=`66264496c6d846fefd6419e37c58df366152af07523cdd1f81088c8839f0e5ec`

RATING=`B`

DECISION=`rejected`

### Finding 10.1

Finding category:
`technical-identifier-chinese-prose-boundary`

Finding owner:
`translation-unit-language`

Observed same-class locations include:

- `string` → `和`
- `errors.New` → `函数`
- `log` → `包`
- `Fatal` → `函数`

Some occur more than once.

Exact predicate:
Within this Page's visible prose outside program bodies, an inline Go/API identifier is immediately concatenated with an adjacent Chinese lexical word.

Revision scope:
Complete Page under this exact predicate.

Out of scope:
API spelling, links, code, error-handling behavior, logging semantics, and general language polishing.

### Finding 10.2

Finding category:
`representation-protected-natural-language-residue`

Finding owner:
`translation-unit-language / representation`

Observed instance:

`使用标准库 <code>log package</code> 中的函数`

The current source representation protects the complete atom `<code>log package</code>`. `log` is technical identity; `package` is ordinary natural-language description and remains visibly untranslated in zh-CN.

Same-class scan result:
This is the sole confirmed protected identifier-plus-English-descriptor instance of this category in the current Page.

Revision scope:
This exact visible atom only, while preserving the `log` technical identity and surrounding source semantics.

Out of scope:
Other `log` references, `Fatal`, program code, links, or general prose.

---

## Fresh asset 11

ASSET=`_content/doc/tutorial/random-greeting.html`

UNIT_ID=`0895c1f2bf020716e82386c738f1b63ff49194250cf4cf254f9cf346a453fc10`

RATING=`B`

DECISION=`rejected`

### Finding 11.1

Finding category:
`technical-identifier-chinese-prose-boundary`

Finding owner:
`translation-unit-language`

Observed instances:

- `randomFormat` → `以`
- `math/rand` → `包`
- `name` → `值`

Same-class scan result:
These are the confirmed current Page instances under this exact category.

Revision scope:
Only the Page's visible explanatory prose satisfying the predicate: inline technical identifier immediately followed by a Chinese lexical word without separator.

Out of scope:
Identifiers, imports, random-selection behavior, example code, links, and unrelated wording.

---

## Fresh asset 12

ASSET=`_content/index.md`

UNIT_ID=`f4365d05fcff295676c0964f81058e6485bb0361f8440cbf243220a6acc1e97d`

PACKAGE=`site-v2-shell`

RATING=`A`

FINDING=`none`

This is an Integrated Surface judgment of the fresh `context-review-required` asset and its composition, not a replacement for its existing TU QC authority.

The complete Homepage source/target was checked together with the supplied shared shell/menu/resources/testimonials, canonical-route, existing-coverage, runtime/public and Tour context. The current textarea teaching comments remain correct. No fresh Homepage/shell integration contradiction, unsupported claim, navigation mismatch, or glossary conflict was found.

---

## Fresh asset 13

ASSET=`data/site-shell/translation.html`

PACKAGE=`site-v2-shell`

RATING=`B`

DECISION=`rejected`

### Finding 13.1

Finding category:
`technical-name-chinese-prose-boundary`

Finding owner:
`structured-surface-language`

Observed locations:

1. Intro:
   `探索<a ...>Go 语言之旅</a>中的...`

2. Coverage paragraph:
   `不是go.dev上的所有页面`

3. Go local status paragraph:
   `已通过<a ...>golang/go#81958</a>提交`

These produce visibly crowded boundaries around a named Tour link, domain identity, and issue identity.

Same-class scan result:
Within the complete current structured asset, these three visible locations are the confirmed instances under the exact predicate.

Exact predicate:
A Latin/domain/project/issue identity, or a link whose visible label begins/ends in such identity, directly touches adjacent Chinese lexical prose without the locale's established separator.

Revision scope:
These three locations in `data/site-shell/translation.html`.

Out of scope:
href values, `Go 语言之旅`, `go.dev`, `golang/go#81958` identities, route structure, project claims, other prose, and unrelated formatting.

### Finding 13.2

Finding category:
`untranslated-natural-language-fragment`

Finding owner:
`structured-surface-language`

Observed instance:

`该 issue 仍处于开放状态`

Canonical source:
`The issue remains open`

Here `issue` is ordinary natural-language prose in this sentence, not a protected identifier or proper name.

Same-class scan result:
The complete structured asset was checked for ordinary-English natural-language residue. This is the confirmed residual instance. Technical/project identities and URLs were excluded.

Revision scope:
This exact ordinary-language fragment only.

Out of scope:
`golang/go#81958`, GitHub URLs, project identity, other technical English tokens, and general rewriting.

---

## Cross-surface checks

CURRENT_UNIFIED_GLOSSARY=`PASS`

GLOSSARY_SHA256=`8274adcbf02829356d814a6bed6b77bf9e4508d850c0eed7efabd1fcbecbe54c`

GO_LOCAL_RULE=`Go local -> Go 本地化`

GO_LOCAL_CROSS_SURFACE_CONSISTENCY=`PASS`

The fresh `/translation/` asset and supplied carried Tour context use `Go 本地化` consistently for the canonical `Go local` concept. Ordinary `local` / `locale` / `localization` contexts were not incorrectly folded into that rule.

No carried asset was independently re-rated.

No fresh/carried combination exposed an additional carried-TU language defect requiring reopening.

---

## Same-class scan summary

1. `representation-protected-natural-language-residue`
   - affected fresh assets: `_content/doc/database/open-handle.md`, `_content/doc/tutorial/handle-errors.html`
   - bounded to each complete affected Page
   - confirmed visible instances fully enumerated above

2. `technical-identifier/name-chinese-prose-boundary`
   - affected fresh assets:
     - `_content/doc/index.html`
     - `_content/doc/tutorial/add-a-test.html`
     - `_content/doc/tutorial/greetings-multiple-people.html`
     - `_content/doc/tutorial/handle-errors.html`
     - `_content/doc/tutorial/random-greeting.html`
     - `data/site-shell/translation.html`
   - complete same-class scan performed within each affected current asset
   - where instances are numerous, the exact predicate and bounded Page/asset scope above define the revision boundary

3. `untranslated-natural-language-fragment`
   - `_content/doc/tutorial/greetings-multiple-people.html`: `name/value`
   - `data/site-shell/translation.html`: `issue`
   - same-class scan did not identify another unambiguous ordinary-language residual in those complete assets

No finding authorizes unrelated style cleanup or whole-Page rewriting.

===== FINAL =====
RESULT=FAILED
DECISION=failed
REVIEW_ID=20261026-zh-CN-integrated-surface-005
LOCALE=zh-CN
ROLE=Independent Integrated Surface Reviewer
PROVIDER=chatgpt
MODEL=GPT-5.6 Sol High
FORMAL_PROVENANCE=gpt-5.6-sol-high

BUNDLE_SHA256=18761e9004908fd660fed28be1fb592d991a620d7d11f76a6bb8fc9ad26c9769
BUNDLE_BYTES=34200483
SCOPE_SHA256=fbb83a4c30209d97c6a1bbc9b27d69c2f93797b5d710bd4cc16b89bd234e0a07
SCOPE_IDENTITY=9ecc0537191d99645599a27da929b675708f161f9bde7f761e98fa7ecc1656e2

INPUT_GATE=PASS
LANGUAGE_JUDGMENT=PERFORMED
REVIEW_REQUIRED_COUNT=13
REVIEWED_COUNT=13
CARRIED_CONTEXT_COUNT=521

A=6
B=7
C=0
D=0

FINDINGS=10
FINDING_CATEGORIES=representation-protected-natural-language-residue,technical-identifier-chinese-prose-boundary,untranslated-natural-language-fragment
NON_A_FRESH_ASSETS=7
UNRESOLVED_COUNT=7

GO_LOCAL_SURFACE_CONSISTENCY=PASS
INDEPENDENT_REVIEW=yes
REPLACEMENT_GENERATED=no
REPOSITORY_MUTATION=none
REMOTE_CALLS_USED=0

UNRESOLVED_BLOCKER=7 fresh review_required assets contain bounded language/surface defects requiring formal exact revision lifecycle
STOP=before-surface-record

因此本轮不能执行 `surface-record`。**唯一下一步**是由 Coordinator / Maintainer 保存这份 Reviewer evidence，并让正式 machine lifecycle 按这 7 个 non-A fresh assets 的 finding owner 分流到对应的 exact revision / representation-successor（如需要）→ validation → independent re-QC；521 个 carried asset 不应重做。
