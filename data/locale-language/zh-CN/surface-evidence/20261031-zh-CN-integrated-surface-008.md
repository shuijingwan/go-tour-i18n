# Independent Integrated Locale Surface Review

- review_id: `20261031-zh-CN-integrated-surface-008`
- reviewer_session: `chatgpt-zh-CN-integrated-surface-reviewer`
- role: Independent Integrated Surface Reviewer
- provider: `chatgpt`
- model: `GPT-5.6 Sol High`
- formal_provenance: `gpt-5.6-sol-high`
- locale: `zh-CN`
- review_scope: full current Integrated Surface, fresh judgment for 13 `review_required` assets
- predecessor_review_id: `20261031-zh-CN-integrated-surface-007`

## Input gate

The review used only the uploaded hash-bound Reviewer Bundle.

- Bundle schema: `go-learning/integrated-surface-reviewer-bundle/v1`
- Bundle ZIP SHA-256: `4361d098f2ad958d2753397a1422c607d15e549f87b71449a23bb72dc1d899ef`
- Bundle bytes: `34195038`
- Manifest identity SHA-256: `75316d0c695dd1e56fa201f297051b07ddd8fe8aeeaf6e269f296ba638aa7d91`
- Scope SHA-256: `97513cc929644583f8fe52421ebad424fa07edafda5cffd763b4f412b9339323`
- Scope identity SHA-256: `00c6e1a3f910e76f7c72bfa318ba77c7ed5f3ea358b606d24ae0fffa9e406a85`
- `review_required`: 13
- `carried`: 521
- Manifest inventory: 336/336 declared files present with exact byte size and SHA-256
- `integration/internal/site/runtime.go`: present and hash-bound in the Bundle
- Canonical route, coverage, Homepage/shared structured composition, runtime/public, glossary, Unified Review, package closure, provenance, authority, and result-template inputs: present
- Repository/RDC/web semantic supplementation: none

`INPUT_GATE=PASS`

## Fresh review-required assets

All 13 current source/target payloads were read in full. The 521 carried assets were used only as integration/context and were not re-rated.

| # | Fresh asset | Fresh judgment |
|---|---|---|
| 1 | `_content/doc/database/cancel-operations.md` | PASS — no unresolved language/integration defect |
| 2 | `_content/doc/database/open-handle.md` | PASS — current exact revisions remain semantically and linguistically sound |
| 3 | `_content/doc/database/querying.md` | PASS — no unresolved language/integration defect |
| 4 | `_content/doc/index.html` | PASS — Docs hub language/navigation semantics are consistent |
| 5 | `_content/doc/install.html` | PASS — installation conditions, commands, paths, and platform semantics remain correct |
| 6 | `_content/doc/security/vuln/database.md` | PASS — security/schema terminology and exclusion semantics remain correct |
| 7 | `_content/doc/security/vuln/editor.md` | PASS — editor/vulnerability terminology and technical semantics remain correct |
| 8 | `_content/doc/tutorial/add-a-test.html` | PASS — current explanatory prose/code boundaries remain acceptable |
| 9 | `_content/doc/tutorial/greetings-multiple-people.html` | PASS — current terminology and explanatory prose remain acceptable |
| 10 | `_content/doc/tutorial/handle-errors.html` | PASS — current error/logging semantics and prose remain acceptable |
| 11 | `_content/doc/tutorial/random-greeting.html` | PASS — current random-selection semantics and prose remain acceptable |
| 12 | `_content/index.md` | Page source/target PASS; Integrated Homepage composition FAILED because mandatory shared shell context contains the exact structured-language finding recorded below |
| 13 | `data/site-shell/translation.html` | PASS — current project/About semantics, terminology, and integration context contain no separate blocker |

Current Unified Glossary usage, including `Go local` → `Go 本地化`, remains consistent across the reviewed fresh surfaces and supplied carried context. No TranslationUnit candidate is reopened by this review.

## Surface 007 contradiction adjudication

The Surface 007 receipt/evidence remains immutable historical reviewer judgment. This review did not use the Surface 007 prose as authority; it directly inspected the current Bundle's hash-bound `integration/internal/site/runtime.go`.

### `Resolver.Destination` control flow

For a root-relative destination:

1. `url.Parse(destination)` sets `u.Path` to the parsed input path; query and fragment remain in `u.RawQuery` / `u.Fragment`.
2. `route := u.Path` initializes the local route from that parsed path.
3. Canonical suffix handling may remove `index.html`, `index.md`, `.html`, or `.md` only when the corresponding canonical route exists.
4. `aliasTarget(route)` is checked **before** the trailing-slash alternate branch. An explicit alias therefore retains precedence.
5. Only when no alias matched, the route is not exact local coverage, and the route is not `/`, the implementation computes exactly one alternate spelling by toggling one trailing slash.
6. The alternate is selected only when `r.routes[alternate]` is true — i.e. it is exact current local coverage, not merely a global canonical route.
7. **After** alias/alternate selection, the Bundle source executes `u.Path = route`.
8. It then executes `u.RawPath = ""`, preventing a stale escaped path from overriding the selected path serialization.
9. Origin selection uses the final `route`: default `https://go.dev`, replaced by `r.origin` when `r.routes[route]` is true.
10. Final `u.String()` serializes the updated `u.Path`, not the original path.
11. No code in this branch modifies `u.RawQuery` or `u.Fragment`, so query and fragment are preserved.
12. For a selected locally covered alternate, the returned URL therefore points directly to that existing local canonical route.

The relevant assignment `u.Path = route` is present **after** the trailing-slash alternate selection. Therefore the factual premise of Surface 007 — that the normalized route was not projected into the returned URL path — cannot be reproduced from this Bundle source.

`SURFACE_007_FINDING_REPRODUCED=no`

### Six routing instances

Bundle public identity for zh-CN supplies local origin `https://go-dev.shuijingwanwq.com`.

| Input | Selected route | Returned path | Origin | Fragment | Verdict |
|---|---|---|---|---|---|
| `/doc/install/` | `/doc/install` | `/doc/install` | local | none | PASS |
| `/doc` | `/doc/` | `/doc/` | local | none | PASS |
| `/learn#guided-learning-journeys` | `/learn/` | `/learn/` | local | `guided-learning-journeys` | PASS |
| `/learn#online-learning` | `/learn/` | `/learn/` | local | `online-learning` | PASS |
| `/learn#featured-books` | `/learn/` | `/learn/` | local | `featured-books` | PASS |
| `/learn#self-paced-labs` | `/learn/` | `/learn/` | local | `self-paced-labs` | PASS |

Thus the effective destinations are:

- `https://go-dev.shuijingwanwq.com/doc/install`
- `https://go-dev.shuijingwanwq.com/doc/`
- `https://go-dev.shuijingwanwq.com/learn/#guided-learning-journeys`
- `https://go-dev.shuijingwanwq.com/learn/#online-learning`
- `https://go-dev.shuijingwanwq.com/learn/#featured-books`
- `https://go-dev.shuijingwanwq.com/learn/#self-paced-labs`

All six reach existing current local canonical routes; all four Learn fragments are retained.

### Negative boundary checks

- Exact local route remains exact: PASS. The alternate branch runs only when the original route lacks exact local coverage.
- Explicit alias precedence: PASS. `aliasTarget(route)` is evaluated before slash-alternate selection.
- Global canonical but locale-uncovered alternate: PASS. Alternate localization requires `r.routes[alternate]`, not only canonical membership.
- Original and alternate both uncovered: PASS. `route` remains uncovered and the origin remains `https://go.dev`.
- Handler route expansion: none. `Destination` does not mutate `r.routes`; `Handler` continues to serve exact covered routes and explicit aliases.
- Canonical route inventory expansion: none.
- Sitemap expansion: none. `Sitemap` enumerates `r.routes`; navigation normalization creates no new route.
- Query preservation: PASS.
- Fragment preservation: PASS.
- General/multi-level path guessing: none. The additional normalization is exactly one trailing-slash toggle.

The predecessor routing finding is therefore resolved for this Bundle and is not copied into the current findings.

## Fresh Integrated Surface finding

### Finding 1 — structured-surface-language

**Exact asset:** `_content/site.tmpl`  
**Fresh integration relation:** mandatory shared structured composition of fresh Homepage `_content/index.md`  
**Locator:** screen-reader-only `#dropdown-description`

**Canonical source**

`Press Enter to activate/deactivate dropdown`

**Current zh-CN target**

`按 Enter 键可activate/deactivate下拉菜单`

**Defect**

`activate/deactivate` is ordinary natural-language action wording in an accessibility instruction, not a technical identifier. The current source representation protects that phrase as an atom, so the current restored zh-CN surface leaves the English action phrase verbatim inside otherwise Chinese screen-reader text. This is a user-facing accessibility localization defect even though the surrounding template asset is carried.

This review does **not** re-rate `_content/site.tmpl`. The finding arises from the mandatory shared structured context that must be composed with the fresh Homepage for Integrated Surface judgment.

**Exact predicate**

Within the mandatory Homepage shared structured composition, user-facing or accessibility natural-language text contains a protected ordinary-English lexical action phrase verbatim inside otherwise zh-CN prose.

**Bounded same-class scan**

Scope:

- `_content/site.tmpl`
- `_content/menus.yaml`
- `_content/resources.yaml`
- `_content/testimonials.yaml`

Result: exactly 1 confirmed instance — the locator above. Technical names, URLs, template tokens, `Enter`, Go/project names, and other intentionally preserved identities are excluded.

**Downstream boundary**

- finding class: `structured-surface-language`
- finding artifacts: empty
- do not modify TranslationUnit candidates
- do not modify glossary
- do not modify routing/canonical coverage
- do not generate replacement in Reviewer
- preserve template/HTML structure, identifiers, links, and menu semantics
- because the ordinary English phrase is protected in the current representation, downstream resolution must use the formal structured/representation authority path rather than treating it as an ordinary TranslationUnit rewrite

This defect was already present in the same hash-bound Bundle used by Surface 007 but was not detected by that historical review. Surface 007 remains immutable; the current fresh review records the independently detected defect rather than rewriting the historical receipt.

## Integration/context judgment

- Current zh-CN terminology and Unified Glossary consistency: PASS
- Tour / Homepage / Learn / Docs terminology relation: PASS
- Fresh source ↔ target semantic fidelity: PASS except the shared accessibility finding above
- Homepage/shared shell composition: FAILED only for Finding 1
- Learn / Docs integration: PASS
- Tour retained-context integration: PASS
- Local canonical routing: PASS
- Six predecessor routing instances: 6/6 PASS
- Query/fragment preservation: PASS
- Explicit alias precedence: PASS
- Legitimate uncovered `https://go.dev` fallback: PASS
- Coverage/canonical/sitemap non-expansion: PASS
- Ads/runtime/public composition: no additional unresolved defect found
- Carried assets individually re-rated: no
- Browser/preview/Production/HUMAN acceptance executed: no
- Semantic remote calls: 0
- Repository mutation: none

## Decision

`INPUT_GATE=PASS`

`LANGUAGE_JUDGMENT=PERFORMED`

The Surface 007 routing finding is not reproduced and is resolved by the actual Bundle runtime control flow. However, the fresh Integrated Surface review independently identifies one unresolved `structured-surface-language` defect in the mandatory Homepage shared composition.

`DECISION=failed`

`FINDING_COUNT=1`

`EXACT_FINDING_COUNT=1`

`STOP=before-surface-record`
