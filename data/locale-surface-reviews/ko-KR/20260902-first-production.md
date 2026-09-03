# ko-KR Locale Surface Review — 2026-09-02 首次 production；2026-09-03 maintenance 更新

本 evidence 同时保留 ko-KR 已完成的首次 production 验收，并记录其后发现的 `/tour/list` 缺陷、修复、preview re-acceptance 与已完成的 maintenance production verification。

## 审核身份与当前 implementation baseline

- locale：`ko-KR`
- review_id：`20260902-first-production`
- Stage A language-assets baseline commit：`f6dbe6772997291a3e37c062c083f936eb3c3338` (`f6dbe67`)
- 首次 production 前 preview reviewed commit：`2fc91938871fb079f7c9f5bfba0af59847566357` (`2fc9193`)
- 当前 maintenance reviewed implementation baseline：`8d61b465a2c06dc2ec7fc0b80d1d0c8a4c490a6f` (`8d61b46`)
- configured production public identity：`https://ko-go-dev.shuijingwanwq.com/`
- evidence update date：`2026-09-03`

在这份 maintenance evidence 于 2026-09-03 更新时，仓库的 `production/identity.json` 确实仍列 `ko-KR` 为 `production_state: first-production`。首次 production 的成功验收随后完成正式状态收尾；当前 identity 为 `production_state: live`。此变更不重写本记录的历史验收，也不授权再次 bootstrap。

### TranslationUnit / projection identity

- ready：`122`
- pending：`0`
- blocked：`0`
- Page：`103`
- eligible Example：`19`
- article/lesson entries：`7`
- modules：`5`

`go run -mod=readonly ./cmd/tour-i18n status check --locale ko-KR` 于本轮返回 `status OK: 122 translation units for ko-KR (103 pages, 19 examples)`。

本次没有 TranslationUnit 变更。因此不重新声明 TranslationUnit Quality Check 或 Final Review；已通过的 Stage A TranslationUnit 审核结论保持不变。

### 当前 locale-level asset identity

Glossary：

- path：`locales/ko-KR/glossary.yaml`
- SHA-256：`dcc0ad47b83965c6347daf9da6ecc9e7517136640d4ba9d1972743def7c97f39`

英文 UI source：

- path：`internal/tour/ui/en.json`
- SHA-256：`3a878119cf0d3414fcf6f4ab20abec6459727ad86060f50fd10b0391f46f2964`
- message count：`92`

ko-KR UI target：

- path：`internal/tour/ui/ko-KR.json`
- SHA-256：`f30df97c0d391dd3e50601abd68676c2e68c8837e3c29d3078ebda2da40a5949`
- message count：`92`
- catalog key identity：source/target 均为 `92`，无缺失或额外 message key。

Article metadata：

- path：`locales/ko-KR/article-metadata.json`
- SHA-256：`d3a2d54d3f23383b550e4b17e97c205a19750021154139e92f7ea6a6571552f8`

Course metadata：

- path：`locales/ko-KR/course-metadata.json`
- SHA-256：`5e69465edb29ff5ab55d58ee5e4803cc5bc39e453ece6501b6b3561f33abc782`

## A. Locale-level language quality review

### 首次 production 的历史结果

结果：`passed`

首次 production 前已完成 TranslationUnit 之外的完整 locale-level language review：公共 UI catalog `90/90`、article metadata `7/7`、course metadata `103/103`，均对照英文/source、ko-KR target 与 glossary，且无遗留 language blocker。

### 本次 `/tour/list` 新增 UI/SEO 文案的受影响范围复审

结果：`passed`（ChatGPT language review，A / passed，无 revision）

`04c7fe6` 新增的 source ↔ target 文案已逐 key 审核，未涉及 TranslationUnit：

| key | English source | ko-KR target |
| --- | --- | --- |
| `tour.list_title` | `Course directory — A Tour of Go` | `강의 목록 — Go 언어 투어` |
| `tour.list_description` | `Browse the modules and lessons in A Tour of Go, an interactive introduction to the Go programming language.` | `Go 프로그래밍 언어를 대화형으로 소개하는 Go 언어 투어의 모듈과 강의를 살펴보세요.` |

结论为 A / passed、无需 revision；与当前 glossary 一致。原 `90/90` 全 catalog review 是首次 production 的历史范围，本次新增两项已另行完成 source ↔ target review，因此当前 catalog identity 为 `92/92`。

## B. 首次 production 的历史验收

首次 production 此前已实际完成；production visual HUMAN gate 结果为 `passed`。这些是已完成的历史验收，不因本轮 maintenance browser-assertion 修复而撤销、重复或重新要求。

历史 complete preview rendered surface acceptance 已通过，原正式入口为：

```sh
scripts/verify-preview-browser.py http://127.0.0.1:43395/ ko-KR
```

历史 preview machine gate 覆盖 preview identity、HTTP / SEO routes、robots、sitemap、production canonical identity、language selector、desktop/mobile rendered geometry、Run / Format / Reset、same-origin Playground `fmt` / `compile`、SPA 与 `/socket` 边界。

历史 production verification 已实际完成，且 production visual HUMAN gate 为 `passed`。本记录不以本轮本地检查改写其当时已通过的 production release 验收，也不虚构本轮未执行的 production command、release identity 或公网结果。

## C. 上线后 `/tour/list` 问题与 maintenance 修复

### 已发现的 production release 问题

上线后发现正在运行的旧 release 的 raw `/tour/list` HTML 缺少目录正文，且 title/description 错用泛化 Tour metadata。该问题属于 `/tour/list` 的 prerender / SEO 页面身份，不是 TranslationUnit 译文缺陷。

### 当前本地修复

- `04c7fe65d5b2172de8a75cece92d1f178eb33674` (`04c7fe6`，`2026-09-03`)：修复 `/tour/list` Chrome prerender，要求 raw HTML 含完整 locale directory 正文，并采用独立的 `tour.list_title` / `tour.list_description` SEO metadata。
- `8d61b465a2c06dc2ec7fc0b80d1d0c8a4c490a6f` (`8d61b46`，`2026-09-03`)：修正 browser acceptance 将 `7` article/lesson entries 与 `103` Page routes 混淆的语义；当前真实目录模型是 `5` modules、`7` article/lesson entries、`103` Page routes。

这两项修复未改变页面视觉，因此保留既有 production visual HUMAN gate `passed`，本轮不重新要求该 gate。

## D. 修复后的 preview machine re-acceptance

结果：`passed`

在 `8d61b46` 后实际执行：

```sh
scripts/verify-preview-browser.py http://127.0.0.1:42151/ ko-KR
```

结果：

```text
[preview-browser] preview identity: PASS
[preview-browser] SEO/routes: PASS
[preview-browser] desktop rendered surface: PASS
[preview-browser] editor Run / Format / Reset: PASS
[preview-browser] SPA: PASS
[preview-browser] mobile /tour/moretypes/1: PASS
PREVIEW SURFACE ACCEPTANCE: PASS
```

该 re-acceptance 覆盖修复后的 preview implementation；其后已由本节所记录的正式 maintenance production verification 完成 production 复核。

## E. `/tour/list` maintenance release production verification

结果：`passed`

### Release identity

- reviewed implementation/evidence commit：`4900995f926ce69fa23d1c1ef28005841969d5c2` (`4900995`)
- local maintenance bundle：`/tmp/go-tour-release-20260903-ko-KR-4900995f`
- remote/current production release：`/data/go-tour-ko-KR/releases/20260903-ko-KR-4900995f`
- published_at：`2026-09-03T02:42:12Z`
- production public URL：`https://ko-go-dev.shuijingwanwq.com/`
- production identity state：`live`

publish result：`locale=ko-KR ready=122 pending=0 blocked=0 pages=103 articles=7`。

### Maintenance deploy 与 CDN gate

已实际执行：

```sh
scripts/deploy-production.sh \
  /tmp/go-tour-release-20260903-ko-KR-4900995f
```

结果：local release preflight、remote SHA-256 与 permissions 均为 PASS；`current` 已原子切换至上述 remote release，service 为 `active`，localhost 连续 `3` 次 HTTP `200`，deployment completed，public acceptance HTTP `200`。

随后已执行 `ko-go-dev.shuijingwanwq.com` 的 Cloudflare hostname purge。

### Production machine acceptance

已实际执行：

```sh
scripts/verify-production.sh \
  /tmp/go-tour-release-20260903-ko-KR-4900995f
```

结果：

```text
[verify-production] release identity: PASS
[verify-production] remote identity: PASS
[verify-production] source routes: 7/7 PASS
[verify-production] public routes: 7/7 PASS
[verify-production] html identity: PASS
sitemap URLs: 105/105
host mismatch: 0
HTTP failure: 0
[verify-production] sitemap: 105/105 PASS
[verify-production] socket boundary: PASS
[verify-production] CDN /: MISS -> MISS -> HIT PASS
[verify-production] CDN /tour/welcome/1: MISS -> MISS -> HIT PASS

PRODUCTION MACHINE ACCEPTANCE: PASS
```

### Production browser acceptance 与 visual HUMAN gate

已实际执行：

```sh
scripts/verify-production-browser.py \
  https://ko-go-dev.shuijingwanwq.com/ \
  ko-KR
```

结果：

```text
[production-browser] desktop routes: PASS
[production-browser] mobile /tour/moretypes/1: PASS
[production-browser] Run / Format / Reset / SPA / ads: PASS
PRODUCTION BROWSER ACCEPTANCE: PASS
```

本次 browser-assertion / prerender / SEO 修复未改变页面视觉；不重新要求 visual HUMAN gate，沿用首次 production 已通过的 visual HUMAN gate：`passed`。

## Reviewer、issues 与结论

- evidence 更新：Codex GPT-5.6
- 新增 list UI/SEO language review：ChatGPT，A / passed，无 revision
- unresolved language blocker：`none`
- unresolved preview rendered blocker：`none`
- unresolved maintenance production blocker：`none`

首次 production 的历史 final decision：`passed`。

本次 `/tour/list` maintenance release production verification final decision：`passed`。
