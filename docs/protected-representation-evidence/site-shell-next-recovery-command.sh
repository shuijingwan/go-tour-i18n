#!/usr/bin/env bash
# Maintainer Local terminal only. This script performs formal recovery writes.
# No Generation, Reviewer record, target installation, closure or Surface work.
set -euo pipefail
cd /home/wangqiang/code/go-tour-i18n

recovery_log=$(mktemp -d /tmp/zh-CN-site-shell-recovery-20261018.XXXXXX)
recovery_stage=preconditions
recovery_result=FAILED
trap 'rc=$?; printf "===== FINAL =====\nRESULT=%s\nSTAGE=%s\nEVIDENCE=%s\nREMOTE_CALLS_USED=0\nSTOP=before-integrated-surface-scope\n" "$recovery_result" "$recovery_stage" "$recovery_log"; exit "$rc"' EXIT

test ! -e data/translation-unit-reconciliations/zh-CN/site-shell-qc001-machine-successors
test ! -e data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002

reconcile=(go run -mod=readonly ./cmd/tour-i18n quality-check representation-reconcile
  --content-package site-v2-shell --locale zh-CN
  --reconciliation-id site-shell-qc001-machine-successors
  --previous-snapshot-id site-v2-shell-qc-001
  --previous-snapshot-sha256 e73d2decb2bde2dccdaf8e330e5a67a507301ffc97f02a46a10de1b391220379
  --formal-impact docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json
  --formal-impact-sha256 8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e
  --historical-documents data/site-shell/successor/sources/30d7c2ef242aa2aa13e64aa5a8917baca377710cb0ef77a68dba3cfb27694272.json
  --historical-documents-sha256 3eba68d190fa9381b95ba5c439514bd70d1f38289849fe18954e242a4580f44c)
recovery_stage=representation-preflight
"${reconcile[@]}" > "$recovery_log/preflight.json" 2> "$recovery_log/preflight.stderr"
python3 - "$recovery_log/preflight.json" > "$recovery_log/preflight-assertions.log" 2>&1 <<'PY'
import json, sys
p = json.load(open(sys.argv[1]))
assert (p['ready'], p['excluded'], p['blocked'], p['applied']) == (1, 0, 0, False)
assert len(p['receipt']['units']) == 1
u = p['receipt']['units'][0]
assert u['replacement']['old_unit_id'] == '0cd6df6f3e0a9b8284ff503254f46315cbee756c7fc6c1f47a8861c0a6270456'
assert u['replacement']['current_unit_id'] == '112b210e73082124640e71c3f726b453e9b197edc0c9c80aa53a41dd950856d3'
assert u['candidate']['sha256'] == u['predecessor']['candidate_sha256'] == '52c02d58b168ce5a226a271e280d4b22f27795c3c7ee1251617c04a193e2648d'
PY

recovery_stage=representation-apply
"${reconcile[@]}" --apply > "$recovery_log/applied.json" 2> "$recovery_log/applied.stderr"
recovery_stage=successor-snapshot
go run -mod=readonly ./cmd/tour-i18n quality-check snapshot \
  --content-package site-v2-shell --locale zh-CN --snapshot-id site-v2-shell-qc-002 \
  > "$recovery_log/snapshot.log" 2>&1
recovery_stage=historical-a-carry
go run -mod=readonly ./cmd/tour-i18n quality-check scope \
  --content-package site-v2-shell --locale zh-CN --snapshot-id site-v2-shell-qc-002 \
  --previous-snapshot-id site-v2-shell-qc-001 --json \
  > "$recovery_log/scope.json" 2> "$recovery_log/scope.stderr"
python3 - "$recovery_log/scope.json" > "$recovery_log/scope-assertions.log" 2>&1 <<'PY'
import json, sys
s = json.load(open(sys.argv[1]))
assert (s['unit_count'], s['a_count'], s['carry_forward_count'], s['pending_count']) == (1, 1, 1, 0)
assert (s['b_count'], s['c_count'], s['d_count']) == (0, 0, 0)
assert s['ready_for_finalization'] is True
m = json.load(open('data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002/manifest.json'))
assert len(m['units']) == 1
u = m['units'][0]
assert u['unit_id'] == '112b210e73082124640e71c3f726b453e9b197edc0c9c80aa53a41dd950856d3'
assert (u['selected_batch_id'], u['attempt']) == ('chatgpt-zh-CN-044', 1)
assert u['representation_reconciliation']['path'] == 'data/translation-unit-reconciliations/zh-CN/site-shell-qc001-machine-successors/receipt.json'
PY

recovery_stage=successor-finalization
go run -mod=readonly ./cmd/tour-i18n quality-check finalize \
  --content-package site-v2-shell --locale zh-CN --snapshot-id site-v2-shell-qc-002 \
  --previous-snapshot-id site-v2-shell-qc-001 > "$recovery_log/finalization.log" 2>&1
recovery_stage=current-authorization-record
go run -mod=readonly ./cmd/tour-i18n quality-check reattest \
  --content-package site-v2-shell --locale zh-CN --snapshot-id site-v2-shell-qc-002 --apply \
  > "$recovery_log/reattest.json" 2> "$recovery_log/reattest.stderr"
recovery_stage=current-authorization-check
go run -mod=readonly ./cmd/tour-i18n quality-check authorization \
  --content-package site-v2-shell --locale zh-CN --snapshot-id site-v2-shell-qc-002 \
  > "$recovery_log/authorization.json" 2> "$recovery_log/authorization.stderr"
python3 - "$recovery_log/authorization.json" > "$recovery_log/authorization-assertions.log" 2>&1 <<'PY'
import json, sys
a = json.load(open(sys.argv[1]))
assert a['package'] == 'site-v2-shell'
assert (a['compatible_count'], a['affected_count'], a['stale_count'], a['unknown_count']) == (1, 0, 0, 0)
assert len(a['units']) == 1
assert a['units'][0]['accepted_snapshot_unit']['unit_id'] == '112b210e73082124640e71c3f726b453e9b197edc0c9c80aa53a41dd950856d3'
PY
recovery_stage=site-shell-successor-current-authorization-complete
recovery_result=PASS
