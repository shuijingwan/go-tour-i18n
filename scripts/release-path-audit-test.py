#!/usr/bin/env python3
"""Focused offline regressions and baseline-inventory delta closure.

No inventory discovery, network, Chrome, deployment, or formal artifact writes.
All external execution is mocked. --check validates the remediation delta.
"""
import contextlib
import hashlib
import importlib.util
import io
import json
import pathlib
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

sys.dont_write_bytecode=True
ROOT=pathlib.Path(__file__).resolve().parent.parent

def load(name,filename):
    spec=importlib.util.spec_from_file_location(name,ROOT/'scripts'/filename)
    value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value

M=load('release_test_maintenance','maintenance-production.py')
B=load('release_test_batch','maintenance-production-batch.py')
F=load('release_test_first','first-production.py')
S=load('release_test_shared','shared-assets-production.py')
R=load('release_test_top_batch','production-release-batch.py')
H=load('release_test_http','release_http.py')
C=load('release_test_cdn','production-cdn.py')
V=load('release_test_browser','verify-production-browser.py')

class ReleaseRegression(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.parent=pathlib.Path(self.tmp.name)
        self.identity=M.IDENTITY.load_identity(ROOT/'production/identity.json')
        self.profiles=[p for p in self.identity['locales'] if p['production_state']=='live']
        self.infos={}
        self.patches=[]
        for core in (M,B.CORE):
            for name,value in [('load',lambda path,current=True:self.infos[str(path.resolve())]),
                               ('shared_freshness',lambda info,shared:{'result':'PASS','shared_assets_identity':info['content']['shared_assets_identity']}),
                               ('prior_state',lambda profile,info:{'pointer':'/prior/'+profile['locale'],'input_identity':info['input_identity'] if self.has_deploy(info) else 'prior','mutation_status':'KNOWN'})]:
                patch=mock.patch.object(core.CONTRACT,name,side_effect=value);patch.start();self.patches.append(patch)
        self.capture=contextlib.redirect_stdout(io.StringIO());self.capture.__enter__()
    def tearDown(self):
        self.capture.__exit__(None,None,None)
        for p in reversed(self.patches):p.stop()
        self.tmp.cleanup()
    def has_deploy(self,info):
        path=pathlib.Path(info['path']+'.maintenance-production-state.json')
        return path.exists() and json.loads(path.read_text())['stages'].get('deploy',{}).get('result')=='PASS'
    def release(self,index=0):
        profile=self.profiles[index];path=self.parent/('go-tour-release-fixture-'+profile['locale']);path.mkdir()
        info={'result':'PASS','kind':'production','locale':profile['locale'],'origin':profile['production_public_url'].rstrip('/'),
              'input_identity':hashlib.sha256(profile['locale'].encode()).hexdigest(),
              'content':{'identity':'b'*64,'shared_assets_identity':'c'*64},'path':str(path)}
        self.infos[str(path)]=info;return path
    @staticmethod
    def commands(item,calls,fail=None):
        def command(stage,args,timeout):
            calls.append((item.locale,stage))
            if stage==fail:raise M.MaintenanceProductionError(stage,'PASS','fixture failed','inspect')
        return command
    def complete(self,item,calls):
        with mock.patch.object(item,'run_command',side_effect=self.commands(item,calls)):
            item.execute()

    def test_repeat_currentness_every_invocation_and_immutable_carry(self):
        path=self.release();item=M.Orchestrator(path);calls=[];self.complete(item,calls)
        self.assertEqual([stage for _,stage in calls],['preflight','deploy','purge','machine','browser'])
        original=item.result_path.read_bytes();state=item.receipt_path.read_bytes();calls.clear()
        carried=M.Orchestrator(path);self.complete(carried,calls)
        self.assertEqual([stage for _,stage in calls],['preflight','machine','browser'])
        self.assertEqual(original,item.result_path.read_bytes());self.assertEqual(state,item.receipt_path.read_bytes())
        self.assertEqual(item.receipt['INPUT_IDENTITY'],item.info['input_identity'])
        for key in ('PREFLIGHT','PRIOR_STATE','ACTION','RESULT','VERIFIER','MUTATION_STATUS'):self.assertIn(key,item.receipt)

    def test_stale_input_shared_and_passed_runtime_fail_closed(self):
        path=self.release();item=M.Orchestrator(path);calls=[];self.complete(item,calls)
        self.infos[str(path)]['input_identity']='d'*64
        with self.assertRaises(M.CONTRACT.ContractError):M.Orchestrator(path)
        self.infos[str(path)]['input_identity']=item.receipt['INPUT_IDENTITY']
        item=M.Orchestrator(path)
        with mock.patch.object(M.CONTRACT,'shared_freshness',side_effect=M.CONTRACT.ContractError('stale shared')):
            with self.assertRaises(M.CONTRACT.ContractError):item.execute()
        with mock.patch.object(M.CONTRACT,'prior_state',return_value={'input_identity':'old'}):
            with self.assertRaises(M.CONTRACT.ContractError):item.execute()

    def test_unknown_mutation_and_cdn_attempt_never_replay(self):
        path=self.release();item=M.Orchestrator(path);item.receipt['MUTATION_STATUS']='UNKNOWN';item.write_receipt('failed')
        with self.assertRaises(M.CONTRACT.ContractError):M.Orchestrator(path)
        item.receipt['MUTATION_STATUS']='KNOWN';item.receipt['cdn_attempt_started']=True;item.write_receipt('failed')
        with self.assertRaises(M.CONTRACT.ContractError):M.Orchestrator(path)

    def test_failed_before_mutation_can_retry(self):
        item=M.Orchestrator(self.release())
        with mock.patch.object(M.subprocess,'run',side_effect=OSError('cannot exec')):
            with self.assertRaises(M.MaintenanceProductionError):item.run_command('deploy',['fixture'],1)
        self.assertEqual(item.receipt['MUTATION_STATUS'],'NONE');item.fail(RuntimeError('before mutation'))
        resumed=M.Orchestrator(item.release_dir);calls=[];self.complete(resumed,calls)
        self.assertIn((item.locale,'deploy'),calls)

    def test_batch_failure_stops_before_later_locale_and_no_false_pass(self):
        paths=[self.release(i) for i in range(3)];batch=B.Batch(paths);calls=[]
        def run(item,stage,args,timeout):
            calls.append((item.locale,stage))
            if item.locale==batch.items[1].locale and stage=='machine':
                raise B.CORE.MaintenanceProductionError(stage,'PASS','fixture failure','inspect')
        with mock.patch.object(B.CORE.Orchestrator,'run_command',autospec=True,side_effect=run):
            with self.assertRaises(B.BatchError):batch.execute()
        self.assertEqual([r['RESULT'] for r in batch.rows],['SUCCEEDED','FAILED','BLOCKED'])
        self.assertNotIn((batch.items[2].locale,'deploy'),calls)
        self.assertTrue(batch.items[0].result_path.is_file())
        for row in batch.rows:
            for key in ('PREFLIGHT','INPUT_IDENTITY','PRIOR_STATE','ACTION','RESULT','VERIFIER','MUTATION_STATUS'):self.assertIn(key,row)
        original=batch.items[0].result_path.read_bytes();calls.clear()
        retry=B.Batch(paths)
        with mock.patch.object(B.CORE.Orchestrator,'run_command',autospec=True,side_effect=lambda i,s,a,t:calls.append((i.locale,s))):retry.execute()
        self.assertEqual([r['RESULT'] for r in retry.rows],['CARRIED','RETRIED','SUCCEEDED'])
        for locale in [batch.items[0].locale,batch.items[1].locale]:self.assertNotIn((locale,'deploy'),calls)
        self.assertEqual(original,batch.items[0].result_path.read_bytes())

    def test_single_and_batch_same_exact_contract(self):
        path=self.release();single=M.Orchestrator(path);batch=B.Batch([path]);calls=[]
        with mock.patch.object(single,'run_command',side_effect=self.commands(single,calls)):single.preflight()
        with mock.patch.object(B.CORE.Orchestrator,'run_command'):batch.preflight()
        self.assertEqual(single.info,batch.items[0].info)
        for key in ('INPUT_IDENTITY','CONTENT_IDENTITY','SHARED_ASSETS_IDENTITY','PRIOR_STATE','PREFLIGHT'):
            self.assertEqual(single.receipt[key],batch.items[0].receipt[key])

    def test_first_and_repeat_share_current_release_and_assets_authority(self):
        path=self.release();info=self.infos[str(path)]
        first=F.Orchestrator.__new__(F.Orchestrator);first.release_dir=path;first.info=info;first.shared=self.identity['shared'];first.receipt={}
        proof={'result':'PASS','shared_assets_identity':info['content']['shared_assets_identity']}
        with mock.patch.object(F.CONTRACT,'load',return_value=info) as loader,mock.patch.object(F.CONTRACT,'shared_freshness',return_value=proof) as freshness:
            first.shared_assets_freshness();loader.assert_called_once_with(path,current=True);freshness.assert_called_once_with(info,first.shared)
        self.assertEqual(first.receipt['SHARED_ASSETS_PROOF'],proof)
        with mock.patch.object(F.CONTRACT,'load',return_value={**info,'input_identity':'old'}):
            with self.assertRaises(F.CONTRACT.ContractError):first.shared_assets_freshness()

    def test_zh_cn_and_selected_all_modes_require_same_shared_workflow(self):
        selected=R.ReleaseBatch(self.parent,False,['zh-CN'])
        all_live=R.ReleaseBatch(self.parent,True,[])
        self.assertTrue(selected.needs_shared_assets);self.assertTrue(all_live.needs_shared_assets)
        self.assertIn('zh-CN',all_live.locales)
        source=(ROOT/'scripts/production-release-batch.py').read_text()
        self.assertIn('shared-assets-production.sh',source);self.assertIn('maintenance-production-batch.sh',source)

    def test_top_batch_current_failure_cannot_be_hidden_by_historical_pass(self):
        value=R.ReleaseBatch.__new__(R.ReleaseBatch);value.maintenance_current_result='FAILED';value.maintenance_status='COMPLETE'
        value._refresh_maintenance_status();self.assertEqual(value.maintenance_status,'FAILED')
        with self.assertRaises(R.ReleaseBatchError):value.print_summary([{'result':'BLOCKED','RESULT':'FAILED'}])

    def test_existing_verifier_wrapper_forwards_current_implementation(self):
        wrapper=(ROOT/'scripts/verify-production.sh').read_text()
        self.assertIn('exec python3 "$script_dir/verify-production.py" "$@"',wrapper)
        self.assertNotIn('EXPECTED_SITEMAP_URLS',wrapper)
        current=(ROOT/'scripts/verify-production.py').read_text()
        self.assertIn('HTTP.model(info,release)',current);self.assertIn('HTTP.public(info,release',current)
        first=(ROOT/'scripts/first-production.py').read_text()
        self.assertIn('"--direct-origin"',first);self.assertIn('"--release-dir", self.release_dir',first)

    def test_browser_uses_carried_preview_plan_and_shared_checks(self):
        info={'content':{'canonical_routes':['/','/tour/list'],'shared_assets':{'css/site.css':'h'},'asset_origin':'https://assets.example','asset_policy':'shared-cloudflare'},
              'browser_plan':{'locale':'zh-CN','canonical_routes':['/','/tour/list'],'pages':[]}}
        with (mock.patch.object(V.CORE,'acceptance') as tour,mock.patch.object(V.SITE,'site_browser_acceptance') as site,
                mock.patch.object(V.CORE.Chrome,'__init__',side_effect=lambda owner,**kw:setattr(owner,'events',[]),autospec=True),mock.patch.object(V.CORE.Chrome,'call'),
                mock.patch.object(V.CORE.Chrome,'close'),mock.patch.object(V.CORE.Chrome,'evaluate',return_value=True),
                mock.patch.object(V.SITE,'assert_browser_events'),mock.patch.object(V,'assert_asset_policy')):
            V.acceptance('https://locale.example/','zh-CN',{}, {},info=info)
            self.assertIs(tour.call_args.kwargs['site_plan'],info['browser_plan'])
            self.assertIs(site.call_args.args[3],info['browser_plan'])
        info['browser_plan']['canonical_routes']=['/tour/list']
        with self.assertRaises(V.BrowserFailure):V.acceptance('https://locale.example/','zh-CN',{}, {},info=info)

    def test_shared_historical_pass_still_validates_current_export(self):
        export=self.parent/'assets';export.mkdir();(export/'SHA256SUMS').write_text('a'*64+'  css/site.css\n')
        workflow=S.Workflow.__new__(S.Workflow);workflow.export_dir=export;workflow.needs_state_write=False;workflow.shared=self.identity['shared']
        workflow.receipt={'verification_result':'PASS','deployment_result':'NO_CHANGES','purge_result':'SKIPPED'};workflow.receipt_path=self.parent/'state'
        stages=[]
        def fail(stage,args,timeout):
            stages.append(stage);raise S.SharedAssetsProductionError(stage,'stale bytes')
        with mock.patch.object(workflow,'_run',side_effect=fail):
            with self.assertRaises(S.SharedAssetsProductionError):workflow.execute()
        self.assertEqual(stages,['current-export'])

    def test_shared_chunk_failure_retry_carries_successful_exact_urls(self):
        export=self.parent/'assets-chunks';export.mkdir();(export/'SHA256SUMS').write_text('fixture\n')
        paths=['css/chunk-%d.css'%i for i in range(C.SHARED_PURGE_BATCH_LIMIT+1)]
        receipt={'schema':C.SHARED_ASSETS_RECEIPT_V2,'export_dir':str(export),'manifest_sha256':hashlib.sha256((export/'SHA256SUMS').read_bytes()).hexdigest(),
                 'deployment_result':'DEPLOYED','production_base_url':self.identity['shared']['shared_assets_public_origin'],
                 'changed_paths':paths,'boundary_paths':C.SHARED_ASSETS_BOUNDARY_PATHS,'purge_result':'PENDING','verification_result':'PENDING'}
        state=pathlib.Path(str(export)+'.shared-assets-execution.json');state.write_text(json.dumps(receipt));calls=[]
        def remote(shared,config):
            calls.append(list(config['files']))
            if len(calls)==2:raise C.CDNError('definite failure')
        with mock.patch.object(C.subprocess,'run',return_value=subprocess.CompletedProcess([],0)),mock.patch.object(C,'_run_remote_config',side_effect=remote):
            with self.assertRaises(C.CDNError):C.shared_assets_coordinator('purge-shared-assets',state)
        partial=json.loads(state.read_text());self.assertEqual(partial['purge_result'],'PENDING');self.assertEqual([c['result'] for c in partial['purge_chunks']],['PASS','PENDING'])
        previous=list(calls[0]);calls.clear()
        with mock.patch.object(C.subprocess,'run',return_value=subprocess.CompletedProcess([],0)),mock.patch.object(C,'_run_remote_config',side_effect=lambda sh,cf:calls.append(list(cf['files']))):C.shared_assets_coordinator('purge-shared-assets',state)
        self.assertEqual(len(calls),1);self.assertNotEqual(calls[0],previous);self.assertEqual(json.loads(state.read_text())['purge_result'],'PASS')

    def test_locale_post_attempt_is_durable_and_unknown_cannot_replay(self):
        path=self.release();item=M.Orchestrator(path);item.receipt['stages']['deploy']={'result':'PASS','input_identity':item.info['input_identity']}
        item.receipt.update(ACTION='PURGE',MUTATION_STATUS='ATTEMPTED');item.write_receipt()
        fake_contract=type('Contract',(),{'load':staticmethod(lambda *a,**k:item.info),'prior_state':staticmethod(lambda *a,**k:{'input_identity':item.info['input_identity']})})
        real_factory=C.importlib.util.module_from_spec
        def factory(spec):return fake_contract if spec.name=='cdn_current_release_contract' else real_factory(spec)
        real_exec=C.importlib.machinery.SourceFileLoader.exec_module
        def execute(loader,module):
            if loader.name!='cdn_current_release_contract':return real_exec(loader,module)
        def remote(shared,config,before_request=None):
            before_request();self.assertTrue(pathlib.Path(str(item.receipt_path)+'.cdn-attempt').is_file())
            self.assertTrue(json.loads(item.receipt_path.read_text())['cdn_attempt_started'])
            raise C.MutationUncertain('lost response after POST')
        with mock.patch.object(C.importlib.util,'module_from_spec',side_effect=factory),mock.patch.object(C.importlib.machinery.SourceFileLoader,'exec_module',autospec=True,side_effect=execute),mock.patch.object(C,'_run_remote_config',side_effect=remote) as run:
            with self.assertRaises(C.MutationUncertain):C.coordinator('purge',item.locale,item.receipt_path)
            with self.assertRaises(C.MutationUncertain):C.coordinator('purge',item.locale,item.receipt_path)
            self.assertEqual(run.call_count,1)

    def test_prior_state_rejects_other_locale_and_binds_current_target(self):
        import os,shlex
        releases=self.parent/'prior-releases';target=releases/'previous';(target/'bin').mkdir(parents=True);(target/'_content/site').mkdir(parents=True)
        current=self.parent/'prior-current';current.symlink_to(target)
        profile={**self.profiles[0],'current':str(current),'releases_root':str(releases),'deployment_lock':str(self.parent/'prior-lock')}
        info={'locale':profile['locale'],'origin':profile['production_public_url'].rstrip('/')}
        (target/'release.json').write_text(json.dumps({'schema_version':3,'locale':profile['locale']}));(target/'SHA256SUMS').write_text('fixture identity\n')
        artifact={'locale':profile['locale'],'origin':info['origin'],'identity':'content','shared_assets_identity':'assets'}
        source=target/'_content/site/release-content.json';source.write_text(json.dumps(artifact))
        (target/'bin/tour').write_text('#!/bin/bash\npython3 -c \'import json,sys;print(json.dumps({"content":json.load(open(sys.argv[1]))}))\' '+shlex.quote(str(source))+'\n');os.chmod(target/'bin/tour',0o755)
        real_run=subprocess.run
        def run(command,**kwargs):
            args=command[command.index('--')+1:]
            prefix='systemctl(){ if [[ $2 == -p || $1 == show ]]; then printf "MainPID=4242";else printf active;fi; }\nreadlink(){ if [[ ${*: -1} == /proc/4242/exe ]];then printf \'%s\' '+shlex.quote(str(target/'bin/tour'))+';else command readlink "$@";fi; }\n'
            return real_run(['bash','-s','--',*args],input=prefix+kwargs['input'],capture_output=True,text=True,check=False)
        with mock.patch.object(C.subprocess,'run',side_effect=run):
            # M.CONTRACT.prior_state is mocked by setUp; use the independent
            # current adapter rather than weakening that unit fixture.
            contract=load('prior_target_contract','release_contract.py')
            state=contract.prior_state(profile,info);self.assertEqual(state['pointer'],str(target))
            artifact['locale']='different';source.write_text(json.dumps(artifact))
            with self.assertRaises(contract.ContractError):contract.prior_state(profile,info)


class AtomicAssetsRegression(unittest.TestCase):
    def run_fixture(self,fail=False):
        import grp,os,pwd,shutil
        with tempfile.TemporaryDirectory() as directory:
            www=pathlib.Path(directory);origin=www/'assets-go-dev.shuijingwanwq.com';staging=www/'.assets-go-dev.staging-fixture';backup=www/'assets-go-dev.shuijingwanwq.com.bak.fixture';lock=www/'.assets-go-dev.deploy.lock'
            def tree(p,data):
                p.mkdir();(p/'css').mkdir();(p/'css/site.css').write_text(data);(p/'SHA256SUMS').write_text(hashlib.sha256(data.encode()).hexdigest()+'  css/site.css\n')
            tree(origin,'old');tree(staging,'new');shutil.copytree(origin,backup);lock.mkdir()
            source=(ROOT/'scripts/deploy-shared-assets.sh').read_text().split("<<'REMOTE_UPDATE'\n",1)[1].split('\nREMOTE_UPDATE',1)[0]
            if fail:
                # Fault only after the atomic swap: the first validation fails,
                # while restoration must validate the complete original tree.
                source=source.replace("printf '[deploy-shared-assets:remote] production mutation starting\\n'", "saved_normalizer=$(declare -f normalize_and_validate);eval \"${saved_normalizer/normalize_and_validate/real_normalizer}\"\nnormalized_once=0\nnormalize_and_validate(){ if (( ! normalized_once )); then normalized_once=1;return 1;fi;real_normalizer \"$@\"; }\nprintf '[deploy-shared-assets:remote] production mutation starting\\n'")
            result=subprocess.run(['bash','-s','--',str(www),str(origin),str(lock),str(staging),str(backup),pwd.getpwuid(os.getuid()).pw_name,grp.getgrgid(os.getgid()).gr_name,'755','644'],input=source,text=True,capture_output=True,check=False)
            self.assertEqual(result.returncode,20 if fail else 0,result.stdout+result.stderr)
            self.assertEqual((origin/'css/site.css').read_text(),'old' if fail else 'new');self.assertEqual((backup/'css/site.css').read_text(),'old')
            self.assertFalse(lock.exists());self.assertFalse(staging.exists())
    def test_assets_apply_switches_complete_tree(self):self.run_fixture()
    def test_assets_failed_apply_restores_whole_predecessor(self):self.run_fixture(True)
    def test_public_font_cors_and_non_html_content_are_required(self):
        import shlex
        with tempfile.TemporaryDirectory() as directory:
            root=pathlib.Path(directory);asset=root/'font';asset.write_bytes(b'font-bytes');manifest=root/'SHA256SUMS';manifest.write_text(hashlib.sha256(b'font-bytes').hexdigest()+'  fonts/current.woff2\n')
            for headers,expected in [('Content-Type: font/woff2\r\nAccess-Control-Allow-Origin: *\r\n',0),('Content-Type: font/woff2\r\n',1),('Content-Type: text/html\r\nAccess-Control-Allow-Origin: *\r\n',1)]:
                shell='source '+shlex.quote(str(ROOT/'scripts/shared-assets-public-verification.sh'))+'\nSHARED_ASSETS_PUBLIC_BASE_URL=https://assets.example\nshared_assets_public_http_request(){ cp '+shlex.quote(str(asset))+' "$2";printf \'%s\' '+shlex.quote(headers)+' > "$3";SHARED_ASSETS_LAST_CURL_EXIT=0;SHARED_ASSETS_LAST_HTTP_STATUS=200; }\nshared_assets_verify_public_assets '+shlex.quote(str(manifest))+'\n'
                result=subprocess.run(['bash','-c',shell],capture_output=True,text=True,check=False)
                self.assertEqual(result.returncode,expected,result.stdout+result.stderr)

class LocaleRecoveryRegression(unittest.TestCase):
    def fixture(self,mismatch=False):
        import os
        with tempfile.TemporaryDirectory() as directory:
            root=pathlib.Path(directory);releases=root/'releases';releases.mkdir();old=releases/'old';staging=releases/'.new.staging-fixture';final=releases/'new';current=root/'current';lock=root/'.deploy.lock';assets=root/'assets-go-dev.shuijingwanwq.com'
            assets.mkdir();(assets/'file').write_text('asset');(assets/'SHA256SUMS').write_text(hashlib.sha256(b'asset').hexdigest()+'  file\n');shared_sha=hashlib.sha256((assets/'SHA256SUMS').read_bytes()).hexdigest()
            expected_assets='a'*64;expected_input='b'*64
            def tree(path,new):
                (path/'bin').mkdir(parents=True);body=json.dumps({'input_identity':expected_input if new else 'old-input','content':{'shared_assets_identity':'wrong' if mismatch and not new else expected_assets,'canonical_routes':['/tour/list']}})
                (path/'bin/tour').write_text("#!/bin/bash\npython3 - \"$0\" <<'PY_PROOF'\nimport hashlib,json,pathlib,sys\na=json.loads("+repr(body)+")\n"+("" if new else "a['input_identity']=hashlib.sha256((pathlib.Path(sys.argv[1]).parent.parent/'SHA256SUMS').read_bytes()).hexdigest()\n")+"print(json.dumps(a))\nPY_PROOF\n");os.chmod(path/'bin/tour',0o755)
                (path/'coverage').write_text('current complete Site routes' if new else 'previous complete Site routes')
                files=['bin/tour','coverage'];(path/'SHA256SUMS').write_text(''.join(hashlib.sha256((path/p).read_bytes()).hexdigest()+'  '+p+'\n' for p in files))
            tree(old,False);tree(staging,True);old_sha=hashlib.sha256((old/'SHA256SUMS').read_bytes()).hexdigest()
            current.symlink_to(old);lock.mkdir()
            source=(ROOT/'scripts/deploy-production.sh').read_text().split("<<'REMOTE_ACTIVATE'\n",1)[1].split('\nREMOTE_ACTIVATE',1)[0]
            # Fixture-only shell functions ensure no real service or HTTP call.
            prefix='''systemctl(){ if [[ $1 == restart ]]; then [[ $(readlink -f "$4/current") == "$4/releases/old" ]]; else printf active; fi; }
curl(){ printf 200; }
journalctl(){ :; }
sleep(){ :; }
'''.replace('$4/current',str(current)).replace('$4/releases/old',str(old))
            result=subprocess.run(['bash','-s','--',str(releases),str(current),str(lock),'fixture.service','http://127.0.0.1:3999/tour/list','3','0','EXISTING',str(old),str(staging),str(final),'fixture',expected_assets,expected_input,old_sha,str(assets),shared_sha],input=prefix+source,text=True,capture_output=True,check=False)
            self.assertEqual(result.returncode,21 if mismatch else 20,result.stdout+result.stderr)
            self.assertEqual(current.resolve(),final if mismatch else old)
            self.assertEqual(lock.exists(),mismatch);self.assertEqual((root/'.assets-go-dev.deploy.lock').exists(),mismatch)
            self.assertTrue((old/'coverage').is_file());self.assertTrue(final.is_dir())
    def test_locale_rollback_restores_whole_same_assets_predecessor(self):self.fixture()
    def test_different_assets_predecessor_stops_with_unknown_evidence(self):self.fixture(True)
    def test_first_recovery_never_removes_unhealthy_current_or_rolls_back_binary(self):
        source=(ROOT/'scripts/recover-first-production-health-failure.sh').read_text()
        self.assertIn('bin/tour" -verify-bundle',source);self.assertIn('/proc/$pid/exe',source);self.assertIn('consumer-token',source)
        self.assertNotIn('mv -T -- "$current"',source);self.assertNotIn('systemctl restart',source)

class HTTPRegression(unittest.TestCase):
    def setUp(self):
        self.m={'input_identity':'a','content_identity':'b','coverage_identity':'c','shared_assets_identity':'d','locale':'zz-ZZ',
                'origin':'https://locale.example','routes':['/','/doc/security/','/learn/','/tour/list','/translation/'],
                'families':['/','/doc/security/**','/learn/**','/tour/**','/translation/'],
                'asset_origin':'https://assets.example','shared_paths':['css/site.css'],'page_main_sha256':{}}
        self.pages={route:('<html lang="zz-ZZ"><head><link rel="canonical" href="'+self.m['origin']+route+'"><link rel="stylesheet" href="https://assets.example/css/site.css"></head><body><main>local '+route+'</main></body></html>').encode() for route in self.m['routes']}
        self.m['page_main_sha256']={r:H.main_digest(H.page(body)) for r,body in self.pages.items() if not r.startswith('/tour/')}
        self.sitemap=('<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">'+''.join('<url><loc>'+self.m['origin']+r+'</loc></url>' for r in self.m['routes'])+'</urlset>').encode()
    def request(self,url,headers=None,origin_ip=None):
        path=H.urllib.parse.urlsplit(url).path
        if path=='/site-release.json':return 200,json.dumps({'identity':'b','input_identity':'a','coverage_identity':'c','shared_assets_identity':'d'}).encode(),{}
        if path=='/sitemap.xml':return 200,self.sitemap,{}
        if path=='/robots.txt':return 200,b'Sitemap: https://locale.example/sitemap.xml',{}
        if path in self.pages:return 200,self.pages[path],{'cf-cache-status':'HIT'}
        return 404,b'missing',{}
    def test_origin_public_and_direct_share_exact_coverage_and_404(self):
        for kwargs in ({'base':'http://127.0.0.1:3999'},{'cache_header':'cf-cache-status'},{'origin_ip':'127.0.0.1'}):
            self.assertEqual(H.verify(self.m,request=self.request,**kwargs)['result'],'PASS')
    def test_false_404_stale_sitemap_wrong_renderer_and_assets_fail_closed(self):
        original=self.sitemap;self.sitemap=b'<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"/>'
        with self.assertRaises(H.HTTPError):H.verify(self.m,request=self.request)
        self.sitemap=original
        for replacement in (b'<html lang="zz-ZZ"><main>stale</main></html>',self.pages['/'].replace(b'https://assets.example',b'https://go.dev')):
            previous=self.pages['/'];self.pages['/']=replacement
            with self.assertRaises(H.HTTPError):H.verify(self.m,request=self.request)
            self.pages['/']=previous
        def false404(url,**kw):
            status,body,headers=self.request(url,**kw);return (200,b'Tour SPA',{}) if status==404 else (status,body,headers)
        with self.assertRaises(H.HTTPError):H.verify(self.m,request=false404)


def closure_check():
    baseline=json.loads((ROOT/'docs/RELEASE_EXECUTABLE_INVENTORY.json').read_text())
    report=json.loads((ROOT/'docs/SITE_V2_RELEASE_PATH_REMEDIATION.json').read_text())
    delta=json.loads((ROOT/'docs/RELEASE_EXECUTABLE_INVENTORY_DELTA.json').read_text())
    assert report['RESULT']=='PASS' and len(report['BLOCKERS'])==report['BLOCKER_COUNT']==12
    assert baseline['COUNT']==len(baseline['ENTRIES']) and delta['BASELINE_SHA256']==hashlib.sha256((ROOT/'docs/RELEASE_EXECUTABLE_INVENTORY.json').read_bytes()).hexdigest()
    assert [b['BLOCKER_ID'] for b in report['BLOCKERS']]==['G%02d'%i for i in range(1,13)]
    for b in report['BLOCKERS']:assert b['STATUS']=='PASS' and b['TARGETED_TEST'] and b['CLOSURE_EVIDENCE']
    for path,digest in delta['SOURCE_SHA256'].items():
        assert hashlib.sha256((ROOT/path).read_bytes()).hexdigest()==digest,path
    allowed={entry['ENTRYPOINT'] for entry in baseline['ENTRIES']}
    for update in delta['UPDATES']:assert update['ENTRYPOINT'] in allowed
    effective={entry['ENTRYPOINT']:dict(entry) for entry in baseline['ENTRIES']}
    for update in delta['UPDATES']:effective[update['ENTRYPOINT']].update(update)
    for added in delta['ADDITIONS']:
        assert added['ENTRYPOINT'] not in effective;effective[added['ENTRYPOINT']]=added
    for entry in effective.values():
        assert entry['STATUS'] in ('ACTIVE','LEGACY','DIAGNOSTIC_ONLY','TEST_ONLY','RETIRED')
        if entry['STATUS']=='RETIRED':assert not entry['CALLERS'] and not entry['CALLEES']
        for target in entry['CALLEES']:
            assert target in effective,(entry['ENTRYPOINT'],target)
            assert entry['ENTRYPOINT'] in effective[target]['CALLERS']
        for caller in entry['CALLERS']:
            if caller.startswith('doc:'):continue
            assert caller in effective and entry['ENTRYPOINT'] in effective[caller]['CALLEES']
    assert not [e['ENTRYPOINT'] for e in effective.values() if e['CLASSIFICATION']=='LEGACY_REFERENCED']
    assert all(row['RESULT']=='PASS' for row in report['MODES'])
    assert all(value=='PASS' for value in report['CONTRACT_RESULTS'].values())
    assert delta['FORMAL_ARTIFACT_MUTATION']=='none'
    for path,digest in delta['FORMAL_SOURCE_SHA256'].items():assert hashlib.sha256((ROOT/path).read_bytes()).hexdigest()==digest,path
    print('RELEASE PATH DELTA CHECK: PASS (baseline consumed; no inventory discovery)')

if __name__=='__main__':
    if sys.argv[1:]==['--check']:closure_check()
    else:unittest.main()
