# Locale Surface Review Evidence: mr-IN

## Mechanical identity

- locale: `mr-IN`
- review-id: `20260925-mr-IN-registry-recheck-001`
- reviewer: `ChatGPT GPT-5.6 Sol High (independent)`
- date: `2026-09-25`
- reviewer bundle SHA-256: `f241f6f29274a7d9a3f0c66403cea685f9af1f570effb2433482d5765577d4a3`
- review package SHA-256: `02668e49b84e50bb751d22e8f17b984e325e1a14b10c419dfe54486802f4b133`
- coverage: pages=103, ui=113, articles=7, translation_units=122, other_surfaces=22
- production state at scaffold time: `live`
- public identity: `https://mr-go-dev.shuijingwanwq.com/` (`mr-go-dev.shuijingwanwq.com`)

## Reviewer conclusions

- attachment and CURRENT package: `passed`; manifest、六份 authority、surface-review.json 的声明 SHA-256 及 CRC 均通过；当前仓库重新导出的 Reviewer ZIP 与附件逐字节一致。
- history: `20260925-recheck.md` 与对应 schema v2 A gate 已核验；原审核 UI 113/113、article 7/7、Course SEO 103/103、TU context 122/122、other surfaces 22/22 全量 passed。
- exact-input comparison: 12 项正式输入中仅 languages_config_sha256 由 `4258ba6a1a19cb69365b156382e33884e42c4250db0629257f56148e5fd292d0` 变成 `091a58bc83203ca5e17be6a11d2db4c4066c96ee08c1dc2e2476a68e67947142`；其余 11 项一致，依日常维护受影响范围规则继承未变化部分的历史审核结论。
- independent registry review: 工作树差异新增 Bulgarian、Malayalam、Marathi 条目与对应 profile；现有 33 个 locale 与 URL 无重复，按 EnglishName 正确排序。Malayalam — മലയാളം 位于 Malay 后、Marathi 前；Marathi — मराठी 的 URL 与当前 mr-IN public identity 完全一致，唯一 Current。Marathi profile 为 Asia/Kolkata、स्थानिक वेळ、ltr；首页、导航选择器和 SEO origin 的组合逻辑已与 CURRENT package 核对，无新增问题。
- language quality review result: `passed`
- findings: `none`
- preview acceptance result: `passed (inherited from prior evidence; not rerun)`

## Final decision

- decision: `passed`
- blocking findings: `0`
- production: 原 `20260925-recheck.md` 中已完成的首次生产 finalization 保持不变；本轮仅恢复 CURRENT A gate，未执行新 preview、publish 或部署，也不新增首次生产 pending placeholder。
