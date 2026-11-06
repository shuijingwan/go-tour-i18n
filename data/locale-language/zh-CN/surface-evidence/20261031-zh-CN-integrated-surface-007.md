# Independent Integrated Surface Review

- review_id: `20261031-zh-CN-integrated-surface-007`
- reviewer_session: `chatgpt-zh-CN-integrated-surface-reviewer`
- role: `Independent Integrated Surface Reviewer`
- provider: `chatgpt`
- model: `GPT-5.6 Sol High`
- formal_provenance: `gpt-5.6-sol-high`
- locale: `zh-CN`
- review_scope: current complete Integrated Surface; fresh judgment only for 13 `review_required` assets; 521 carried assets context-only

## Input gate

`INPUT_GATE=PASS`

Verified from the uploaded current Bundle:

- Bundle schema: `go-learning/integrated-surface-reviewer-bundle/v1`
- lifecycle: `go-learning/integrated-locale-surface/v1`
- locale: `zh-CN`
- ZIP SHA-256: `4361d098f2ad958d2753397a1422c607d15e549f87b71449a23bb72dc1d899ef`
- ZIP bytes: `34195038`
- manifest identity: `75316d0c695dd1e56fa201f297051b07ddd8fe8aeeaf6e269f296ba638aa7d91`
- scope SHA-256: `97513cc929644583f8fe52421ebad424fa07edafda5cffd763b4f412b9339323`
- scope identity: `00c6e1a3f910e76f7c72bfa318ba77c7ed5f3ea358b606d24ae0fffa9e406a85`
- `review_required`: 13
- `carried`: 521
- manifest inventory: 336 files; all declared byte sizes and SHA-256 values matched
- current glossary and Unified Glossary Review were present and matched their declared identities
- source/target payloads, representations, provenance, closures, integration context, runtime/public context, authority files, and result template were present

No repository, RDC, web, browser, preview, Production, or other external semantic input was used.

## Fresh review results

All 13 `review_required` assets were read as complete current source/target pairs and judged fresh. The 521 carried assets were used only where required for cross-surface/integration context and were not re-rated.

| # | Fresh asset | Language/source-target judgment | Integrated judgment |
|---|---|---|---|
| 1 | `_content/doc/database/cancel-operations.md` | PASS | PASS |
| 2 | `_content/doc/database/open-handle.md` | PASS | PASS |
| 3 | `_content/doc/database/querying.md` | PASS | PASS |
| 4 | `_content/doc/index.html` | PASS | PASS |
| 5 | `_content/doc/install.html` | PASS | PASS |
| 6 | `_content/doc/security/vuln/database.md` | PASS | PASS |
| 7 | `_content/doc/security/vuln/editor.md` | PASS | PASS |
| 8 | `_content/doc/tutorial/add-a-test.html` | PASS | PASS |
| 9 | `_content/doc/tutorial/greetings-multiple-people.html` | PASS | PASS |
| 10 | `_content/doc/tutorial/handle-errors.html` | PASS | PASS |
| 11 | `_content/doc/tutorial/random-greeting.html` | PASS | PASS |
| 12 | `_content/index.md` | PASS | **FAILED — surface-integration** |
| 13 | `data/site-shell/translation.html` | PASS | PASS |

No new TranslationUnit-language or structured-surface-language finding was identified in the fresh set. Current glossary usage, Homepage/translation terminology, Learn/Docs/Tour terminology, source-owned semantics, protected technical identities, and current visible Chinese were acceptable for this Integrated Surface review.

## Successor check of predecessor finding

Predecessor review: `20261030-zh-CN-integrated-surface-006`

Predecessor finding class: `surface-integration`

Predecessor exact defect: six root-relative Homepage/shared-navigation targets differed from already-covered local canonical routes only by one trailing slash, causing them not to resolve to those local canonical routes.

`PREDECESSOR_FINDING_RESOLVED=no`

The current engineering change in `integration/internal/site/runtime.go` partially changes the behavior but does not close the defect.

### Current runtime behavior

In `Resolver.Destination`, after explicit alias processing, the current implementation detects the alternate trailing-slash spelling and assigns that covered path to the local variable `route`:

- `/doc/install/` selects covered alternate `/doc/install`
- `/doc` selects covered alternate `/doc/`
- `/learn` selects covered alternate `/learn/`

This is appropriately bounded by exact local coverage and occurs only after alias handling.

However, the returned destination still uses the original parsed URL via `u.String()` rather than the normalized `route`. Consequently the selected alternate affects only the origin decision; it is not projected into the returned path.

For the six current instances, `Destination` therefore produces the local origin with the original non-canonical path:

1. `_content/index.md`: `/doc/install/` -> local origin + `/doc/install/`, not local canonical `/doc/install`
2. `_content/index.md`: `/learn#guided-learning-journeys` -> local origin + `/learn#guided-learning-journeys`, not `/learn/#guided-learning-journeys`
3. `_content/index.md`: `/learn#online-learning` -> local origin + `/learn#online-learning`, not `/learn/#online-learning`
4. `_content/index.md`: `/learn#featured-books` -> local origin + `/learn#featured-books`, not `/learn/#featured-books`
5. `_content/index.md`: `/learn#self-paced-labs` -> local origin + `/learn#self-paced-labs`, not `/learn/#self-paced-labs`
6. mandatory shared composition context `_content/menus.yaml`: `/doc` -> local origin + `/doc`, not local canonical `/doc/`

The four fragments are retained syntactically, but they remain attached to the uncovered `/learn` path rather than the covered `/learn/` route.

The current `Handler` serves only exact `r.routes[q.URL.Path]` paths and otherwise redirects only through the explicit alias table. The current redirect inventory contains no alias for `/doc/install/`, `/doc`, or `/learn`. Therefore the six returned local paths are not the covered canonical routes and are not recovered by Handler alias behavior.

### Bounded same-class scan

Exact predicate:

> Within the 13 fresh targets plus Homepage mandatory shared structured composition context, a root-relative navigation target is not an exact current local canonical route, its only trailing-slash alternate is an exact current local canonical route, and no explicit redirect alias owns the original target.

Bounded scan result: exactly **6** instances, all listed above.

No additional instance matching this predicate was found in the bounded scan of the 13 fresh targets plus `_content/menus.yaml`, `_content/resources.yaml`, and `_content/testimonials.yaml`.

### Preserved behavior checked

The current code correctly keeps the following constraints:

- explicit aliases are evaluated before trailing-slash alternate selection;
- exact covered routes remain exact;
- the alternate is selected only when that alternate is itself in local coverage;
- routes with neither exact nor alternate local coverage still retain the normal `https://go.dev` fallback decision;
- no new canonical route, HTTP route, sitemap entry, or coverage entry is introduced by the reviewed change;
- query/fragment bytes remain on the original parsed URL.

These preserved properties do not close the six affected navigation destinations because the normalized local route is not written into the returned URL path.

## Exact finding

### Finding 1

- class: `surface-integration`
- anchor fresh asset: `_content/index.md`
- mandatory shared context: `_content/menus.yaml`
- severity: blocking Integrated Surface defect
- affected instances: 6
- predecessor relation: same persisted finding class; current correction is incomplete

**Exact defect:** `Resolver.Destination` recognizes the locally covered trailing-slash alternate in its `route` variable, but returns `base + u.String()` without applying that normalized route to the URL path. The resulting Homepage/shared-navigation hrefs remain `/doc/install/`, `/doc`, and `/learn#...` on the local origin. Those paths are neither exact local coverage nor explicit aliases, so they do not reach the existing local canonical `/doc/install`, `/doc/`, and `/learn/` routes.

**Bounded resolution scope:** only the routing/public-composition behavior needed for the six enumerated destinations to resolve to their already-existing local canonical routes while preserving the four fragments. No TranslationUnit text, structured translation, glossary, canonical route inventory, coverage set, sitemap, unrelated redirect behavior, or carried language judgment is authorized by this finding.

## Integration/context judgment

- Chinese language quality across the 13 fresh assets: PASS
- current Unified Glossary consistency: PASS
- Tour/Learn/Docs/Homepage/translation terminology: PASS
- Homepage/shared-shell composition: FAILED only for the exact routing finding above
- local canonical routing: FAILED for the six bounded instances
- fragment preservation: fragment bytes are retained, but the four fragments remain attached to the uncovered `/learn` path; therefore the intended local canonical navigation is not satisfied
- explicit alias precedence: PASS
- non-covered-path fallback guard: PASS
- canonical route inventory / coverage / sitemap non-expansion: PASS
- ads integration and current `/translation/`/Homepage ad semantics: PASS
- carried assets re-rated: no

## Decision

`decision=failed`

`findings=1`

`exact_findings=1`

The predecessor Integrated Surface finding is **not resolved** by the current Bundle. This review does not generate replacement content or modify repository/runtime code.

## Reviewer boundaries

- independent review: yes
- replacement generated: no
- repository mutation: none
- semantic remote calls: 0
- carried assets re-rated: no
- preview/browser executed: no
- Production validation executed: no
- HUMAN acceptance performed: no
- stop point: `before-surface-record`
