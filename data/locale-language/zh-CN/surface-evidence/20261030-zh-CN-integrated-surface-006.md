# Independent Integrated Surface Review — 20261030-zh-CN-integrated-surface-006

- review_id: `20261030-zh-CN-integrated-surface-006`
- reviewer_session: `chatgpt-zh-CN-integrated-surface-reviewer`
- role: `Independent Integrated Surface Reviewer`
- provider: `chatgpt`
- model: `GPT-5.6 Sol High`
- formal_provenance: `gpt-5.6-sol-high`
- locale: `zh-CN`
- bundle_sha256: `7fa3ead53c59d9b6381d2eb9ca625667b897eed7fce18c7dd78f70b7d88980d2`
- scope_sha256: `97513cc929644583f8fe52421ebad424fa07edafda5cffd763b4f412b9339323`
- scope_identity: `00c6e1a3f910e76f7c72bfa318ba77c7ed5f3ea358b606d24ae0fffa9e406a85`
- input_gate: `PASS`
- review_required_count: `13`
- fresh_reviewed_count: `13`
- carried_count: `521`
- carried_rerated: `no`
- semantic_remote_calls: `0`
- repository_mutation: `none`

## Input gate

`manifest.json`、`integrated-surface-scope.json`、Bundle schema/kind/lifecycle/locale、scope SHA/identity、package closures、current glossary、Unified Review、runtime/public refs、336 个 inventory 文件的 size/SHA、13 个 `fresh_review_required` exact-set、521 carried count、source/target payload、representation/provenance、integration context、authority files 和 `independent-surface-result.template.json` 均完成核对。

结果：`INPUT_GATE=PASS`。Bundle ZIP 实际 SHA-256 为 `7fa3ead53c59d9b6381d2eb9ca625667b897eed7fce18c7dd78f70b7d88980d2`，scope SHA-256 为 `97513cc929644583f8fe52421ebad424fa07edafda5cffd763b4f412b9339323`，scope identity 为 `00c6e1a3f910e76f7c72bfa318ba77c7ed5f3ea358b606d24ae0fffa9e406a85`。

## Fresh review results

| # | fresh asset | judgment |
|---:|---|---|
| 1 | `_content/doc/database/cancel-operations.md` | PASS |
| 2 | `_content/doc/database/open-handle.md` | PASS |
| 3 | `_content/doc/database/querying.md` | PASS |
| 4 | `_content/doc/index.html` | PASS |
| 5 | `_content/doc/install.html` | PASS |
| 6 | `_content/doc/security/vuln/database.md` | PASS |
| 7 | `_content/doc/security/vuln/editor.md` | PASS |
| 8 | `_content/doc/tutorial/add-a-test.html` | PASS |
| 9 | `_content/doc/tutorial/greetings-multiple-people.html` | PASS |
| 10 | `_content/doc/tutorial/handle-errors.html` | PASS |
| 11 | `_content/doc/tutorial/random-greeting.html` | PASS |
| 12 | `_content/index.md` | **FAILED — Integrated Surface routing composition** |
| 13 | `data/site-shell/translation.html` | PASS |

前 11 个 Learn/Docs fresh Page 的完整 canonical source 与 current restored target 均完成 fresh comparison。此前 revision 涉及的 teaching comments、technical identity、Markdown paragraph boundary 和 inline technical wording 在 current candidates 中没有形成新的 unresolved language defect。`data/site-shell/translation.html` 的完整 structured source/target、`Go local → Go 本地化`、156-page coverage 陈述、workflow/project-status/ads prose 均与 Bundle authority 一致；其中 `issue` 作为 GitHub issue 语境中的技术名词不构成本轮 blocking language residue。

Homepage `_content/index.md` 的中文 Page TranslationUnit 本身没有发现新的语言质量 blocker；失败来自它与 current coverage-aware resolver/shared shell 的实际组合。

## Integration/context judgment

### Glossary and terminology

- current glossary SHA-256: `8274adcbf02829356d814a6bed6b77bf9e4508d850c0eed7efabd1fcbecbe54c`
- current Unified Review: `20261024-zh-CN-glossary-review-015`, passed
- `Go local → Go 本地化`: PASS
- Tour / Homepage / Learn / Docs / `/translation/` terminology consistency: PASS

### Coverage, canonical, runtime and ads

- canonical routes: `158`
- local canonical `/learn/`: present
- local canonical `/doc/`: present
- local canonical `/doc/install`: present
- zh-CN Homepage `/` and `/translation/` advertising disabled by current `AdPolicy`: consistent with source-owned project statement
- zh-CN Tour advertising disabled by `GoLocal` policy: consistent
- documentation advertising eligibility: consistent with current policy
- no browser, preview, Production or HUMAN acceptance was performed or claimed

## Exact finding

### Finding 1 — covered local route falls back to official Go because equivalent slash form is not canonicalized

**Finding class:** `surface-integration`

**Fresh anchor:** `_content/index.md` (`context-review-required`)

**Defect:** current rendered Site v2 navigation can send users to `https://go.dev` for pages that are already present in the current local zh-CN canonical coverage. The current resolver normalizes `.html`, `.md`, `index.html`, and `index.md`, and applies explicit redirect aliases, but it does not normalize a route solely by adding/removing the trailing slash before the exact `r.routes[route]` coverage decision. The current redirect inventory contains no aliases for the affected forms.

**Observed instances in the fresh Homepage target:**

1. `href="/doc/install/"` — visible label `下载 Go`; current local canonical route is `/doc/install`. Current resolver therefore selects the official base and resolves this to `https://go.dev/doc/install/` instead of the localized page.
2. `href="/learn#guided-learning-journeys"` — current local canonical route is `/learn/`; resolves to `https://go.dev/learn#guided-learning-journeys`.
3. `href="/learn#online-learning"` — same route mismatch; resolves to official Go.
4. `href="/learn#featured-books"` — same route mismatch; resolves to official Go.
5. `href="/learn#self-paced-labs"` — same route mismatch; resolves to official Go.

**Same-class integration context instance:**

6. Mandatory Homepage shared context `_content/menus.yaml` contains the visible `Go 用户手册` child with `url: /doc`; current local canonical route is `/doc/`. Under the same resolver predicate this shared navigation destination resolves to `https://go.dev/doc` rather than the localized `/doc/` page. This carried structured asset is context-only and is **not re-rated**; the instance is recorded only because the fresh Homepage Integrated Surface composition is required to include the current shared menu.

**Exact predicate:** within the 13 fresh targets plus the four mandatory Homepage shared composition assets (`_content/site.tmpl`, `_content/menus.yaml`, `_content/resources.yaml`, `_content/testimonials.yaml`), a root-relative user-visible navigation destination is not itself in current local canonical coverage and has no explicit redirect alias, while adding or removing only its trailing slash yields a current local canonical route. Under current `Resolver.Destination`, that mismatch causes fallback to `https://go.dev` despite equivalent local coverage.

**Same-class scan result:** complete for the bounded scope above. Exactly 6 instances were found: 5 in fresh `_content/index.md` and 1 in carried-context `_content/menus.yaml`. No other root-relative destination in that bounded scope satisfies this predicate.

**Bounded resolution scope:** routing/canonical normalization or exact source/adoption/navigation correction sufficient to make these six destinations resolve to the already-covered local canonical routes. This finding does not authorize translation rewriting, unrelated link changes, route-set expansion, general navigation cleanup, or re-rating any carried asset.

**Preserve:** visible translated labels, source meaning, current `/learn/`, `/doc/`, `/doc/install` canonical identities, fragments, shared menu structure, other external fallback behavior, glossary decisions, protected technical identity, and existing carried language judgments.

## Carried context statement

All 521 carried assets remained context-only. No carried TranslationUnit was re-rated. The `_content/menus.yaml` instance above is evidence of a current composition relation, not a new carried language rating.

## Decision

`decision=failed`

The 13 fresh assets were all reviewed. Language fidelity is acceptable across the fresh exact-set, but the fresh Homepage Integrated Surface composition has one unresolved coverage-aware routing finding with six exact same-class instances. Therefore the Integrated Surface cannot pass until the surface finding is resolved and a successor independent review is completed.

- findings: `1`
- exact_findings: `1`
- unresolved integrated surface findings: `1`
- replacement generated: `no`
- repository mutation: `none`
- semantic remote calls: `0`
- preview/browser/Production/HUMAN acceptance: `not performed`
- stop: `before-surface-record`
