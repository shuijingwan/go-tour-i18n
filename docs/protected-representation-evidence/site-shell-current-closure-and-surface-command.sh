#!/usr/bin/env bash
# Maintainer Local terminal only; Codex must not execute this script.
# Create the current shell closure, check it, then compute Integrated Surface
# scope. No activation, Reviewer evidence, Generation or language writes.
set -euo pipefail
cd /home/wangqiang/code/go-tour-i18n

shell_run_dir=$(mktemp -d /tmp/zh-CN-shell-current-closure-surface.XXXXXX)
shell_stage=exact-inputs
shell_result=FAILED
shell_new_closure=none
shell_new_sha=none
shell_scope_sha=none
trap 'shell_rc=$?; printf "===== FINAL =====\nRESULT=%s\nSTAGE=%s\nCURRENT_SHELL_SNAPSHOT=site-v2-shell-qc-002\nNEW_SHELL_CLOSURE=%s\nNEW_SHELL_CLOSURE_SHA256=%s\nSURFACE_SCOPE_SHA256=%s\nEVIDENCE=%s\nREMOTE_CALLS_USED=0\nSTOP=before-integrated-surface-review\n" "$shell_result" "$shell_stage" "$shell_new_closure" "$shell_new_sha" "$shell_scope_sha" "$shell_run_dir"; exit "$shell_rc"' EXIT

# Bind exactly the confirmed inputs. Hashes check drift; all semantic/current
# validation still belongs to the existing Go readers below.
python3 -I - "$shell_run_dir" > "$shell_run_dir/inputs.log" 2>&1 <<'PY'
import hashlib, json, pathlib, sys
refs = {
 'data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002/manifest.json': '52890c232068bd911e1ae92208518ed7a0f1c35b63ccdceb8b3ee668ae85c47b',
 'data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002/finalization.json': 'fc5c325871ba9e29dc0b6e8693fbdb270038bc7d27a24f7a6a5cba641bd56ffb',
 'data/translation-authorizations/zh-CN/a03a8d8174fbafd8a01589a162af42f4c3adbb816f4131f2f2e0abbbe16d8ca2/site-v2-shell-qc-002/47870635bf0940d2c11faa24bc8ef2757878b8b70672264a5dee18b22504f946.json': 'a0fecc0a294d42c564087160fcd65447b2181240a6c859a46fab77ff429abcec',
 'data/site-workflow/zh-CN/site-v2-shell/closure/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json': 'dcb7fcafd1708c4198747356996b51776c2e1c874bd86ff7d0110c31135e8ae9',
 'data/site-workflow/zh-CN/learn-docs-v1/closure/fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9.json': '47588d761b615abb7f435b488c1058a83cc06e2b833d9118ccac5282a692e88e',
 'data/locale-language/zh-CN/structured-generation/3aa00f3907d55604a78a411e88fe0fb478e232dd5f80cbfea66ca9e80c53bc78.json': 'e4c7a8b4e9fa94cf06422d9c1a376777faf3a9e43a03f0694a90c358cebd0413',
 'data/locale-language/zh-CN/structured-generation/7caebba33580782e5837719efa23a8d1cf4d60f51de84fa3e5c56c8816d25cb5.json': '3bdea5f48d19a48c69964d515d87117a095015d9aa6a33e2ad4837b59a10c8c3',
 'data/locale-language/zh-CN/surface/20261014-zh-CN-integrated-surface-faa455c8-002.json': '13977766b2bcbd7f96f20151f824b60b8cc3df7c25a0604b1a19cd349d34361a',
}
for name, sha in refs.items():
 assert hashlib.sha256(pathlib.Path(name).read_bytes()).hexdigest() == sha, name
coverage = json.loads(pathlib.Path('locales/zh-CN/content-scope.json').read_bytes())
shell = [p for p in coverage['packages'] if p['package'] == 'site-v2-shell']
assert len(shell) == 1 and shell[0]['state'] == 'complete'
assert shell[0]['evidence'][0] == {'path': 'data/site-workflow/zh-CN/site-v2-shell/closure/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json', 'sha256': refs['data/site-workflow/zh-CN/site-v2-shell/closure/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json']}
pathlib.Path(sys.argv[1], 'inputs.json').write_text(json.dumps(refs, indent=2)+'\n')
PY

shell_stage=current-shell-package-closure
# No structured replacement exists in this tranche. The canonical builder
# inherits the complete verified structured ancestry from current coverage
# (41ec is HISTORICAL), and binds the new Homepage finalization independently.
# Do not flatten/re-submit historical Generations as current delta inputs.
go run -mod=readonly ./cmd/tour-i18n site-content package-closure \
  --locale zh-CN --package site-v2-shell \
  --input data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002/finalization.json \
  > "$shell_run_dir/package-closure-result.json" 2> "$shell_run_dir/package-closure.stderr"

shell_stage=new-closure-binding
python3 -I - "$shell_run_dir" > "$shell_run_dir/closure-binding.log" 2>&1 <<'PY'
import hashlib, json, pathlib, re, sys
d = pathlib.Path(sys.argv[1])
name = json.loads((d/'package-closure-result.json').read_bytes())
assert isinstance(name, str) and re.fullmatch(r'data/site-workflow/zh-CN/site-v2-shell/closure/[0-9a-f]{64}\.json', name)
assert not name.endswith('/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json')
b = pathlib.Path(name).read_bytes(); c = json.loads(b)
assert c['schema'] == 'go-learning/package-language-closure/v1'
assert c['locale'] == 'zh-CN' and c['package'] == 'site-v2-shell'
assert c['identity_sha256'] == pathlib.Path(name).stem
assert c['page_finalization'] == {'path': 'data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002/finalization.json', 'sha256': 'fc5c325871ba9e29dc0b6e8693fbdb270038bc7d27a24f7a6a5cba641bd56ffb'}
(d/'new-closure.path').write_text(name+'\n')
(d/'new-closure.sha256').write_text(hashlib.sha256(b).hexdigest()+'\n')
PY
shell_new_closure=$(cat "$shell_run_dir/new-closure.path")
shell_new_sha=$(cat "$shell_run_dir/new-closure.sha256")

shell_stage=exact-closure-current-validation
# Existing canonical internal API, via opt-in read-only test transport.
GO_TOUR_CURRENT_SHELL_CLOSURE="$shell_new_closure" \
GO_TOUR_CURRENT_SHELL_CLOSURE_SHA256="$shell_new_sha" \
GO_TOUR_CURRENT_SHELL_CLOSURE_REPORT="$shell_run_dir/closure-current.json" \
go test -mod=readonly ./internal/sitecontent \
  -run '^TestMaintainerCurrentShellClosureReadonly$' -count=1 -v \
  > "$shell_run_dir/closure-current-test.log" 2>&1
python3 -I - "$shell_run_dir/closure-current.json" "$shell_new_closure" "$shell_new_sha" > "$shell_run_dir/closure-current-assertions.log" 2>&1 <<'PY'
import json, sys
r = json.load(open(sys.argv[1]))
assert r['result'] == 'PASS' and r['closure'] == {'path':sys.argv[2], 'sha256':sys.argv[3]}
assert r['localized_bytes_unchanged'] is True and r['translation_carry'] == 'PASS'
PY
printf 'SHELL_CLOSURE_CURRENT=PASS\nNEW_SHELL_CLOSURE=%s\nSHA256=%s\n' "$shell_new_closure" "$shell_new_sha"

shell_stage=integrated-surface-scope
go run -mod=readonly ./cmd/tour-i18n site-content surface-scope \
  --locale zh-CN \
  --generation data/site-workflow/zh-CN/learn-docs-v1/closure/fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9.json \
  --generation "$shell_new_closure" \
  > "$shell_run_dir/surface-scope.json" 2> "$shell_run_dir/surface-scope.stderr"
shell_scope_sha=$(sha256sum "$shell_run_dir/surface-scope.json")
shell_scope_sha=${shell_scope_sha%% *}
shell_stage=current-shell-closure-and-surface-scope-complete
shell_result=PASS
