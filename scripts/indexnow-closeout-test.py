#!/usr/bin/env python3
import importlib.util, pathlib, stat, subprocess, sys, tempfile, unittest
from unittest import mock
ROOT=pathlib.Path(__file__).resolve().parent.parent
spec=importlib.util.spec_from_file_location("tested",ROOT/"scripts"/"indexnow-closeout.py"); CLOSEOUT=importlib.util.module_from_spec(spec); sys.modules[spec.name]=CLOSEOUT; spec.loader.exec_module(CLOSEOUT)
VHOST='''server { listen 80; server_name other.example; }
server {\n listen 443 ssl;\n server_name locale.example;\n location / { proxy_pass http://127.0.0.1:4000; }\n}\n'''
class TestProvision(unittest.TestCase):
 def test_default_key_first_run_generates_secure_key_and_second_run_reuses_it(self):
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory)/"store"
   path,action=CLOSEOUT.default_key_file("zz-ZZ",root)
   self.assertEqual(action,"generated"); key,raw=CLOSEOUT.read_key(path)
   self.assertEqual(len(key),64); self.assertRegex(key,r"^[a-f0-9]{64}$"); self.assertEqual(raw,key.encode()+b"\n")
   self.assertEqual(stat.S_IMODE(root.stat().st_mode),0o700); self.assertEqual(stat.S_IMODE(path.parent.stat().st_mode),0o700); self.assertEqual(stat.S_IMODE(path.stat().st_mode),0o600)
   second,action=CLOSEOUT.default_key_file("zz-ZZ",root)
   self.assertEqual((second,action),(path,"reused")); self.assertEqual(len(list(path.parent.iterdir())),1)
 def test_default_key_store_rejects_ambiguous_or_malformed_entries(self):
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory)/"store"; locale=root/"zz-ZZ"; locale.mkdir(parents=True)
   for key in ("a"*8,"b"*8): (locale/(key+".txt")).write_text(key+"\n")
   with self.assertRaises(CLOSEOUT.CloseoutError): CLOSEOUT.default_key_file("zz-ZZ",root)
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory)/"store"; locale=root/"zz-ZZ"; locale.mkdir(parents=True); (locale/"bad_key.txt").write_text("bad_key\n")
   with self.assertRaises(CLOSEOUT.CloseoutError): CLOSEOUT.default_key_file("zz-ZZ",root)
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory)/"store"; locale=root/"zz-ZZ"; locale.mkdir(parents=True); target=pathlib.Path(directory)/"target"; target.write_text("a"*8+"\n"); (locale/("a"*8+".txt")).symlink_to(target)
   with self.assertRaises(CLOSEOUT.CloseoutError): CLOSEOUT.default_key_file("zz-ZZ",root)
 def test_explicit_key_file_bypasses_default_store(self):
  with tempfile.TemporaryDirectory() as directory:
   key="a"*8; path=pathlib.Path(directory)/(key+".txt"); path.write_text(key+"\n")
   with mock.patch.object(CLOSEOUT,"default_key_file") as default:
    instance=CLOSEOUT.Closeout("zh-CN",path)
   default.assert_not_called(); self.assertEqual(instance.key_file,path); self.assertEqual(instance.key_source,"explicit")
 def test_key_contract_length_and_characters(self):
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory)
   for key in ("a"*8,"z"*128,"test-key"):
    path=root/(key+".txt"); path.write_text(key+"\n"); self.assertEqual(CLOSEOUT.read_key(path)[0],key)
   for key in ("a"*7,"a"*129,"bad_key","bad key","bad/key","bad.key"):
    path=root/(key.replace("/","-")+".txt"); path.write_text(key+"\n")
    with self.subTest(key=key), self.assertRaises(CLOSEOUT.CloseoutError): CLOSEOUT.read_key(path)
 def test_provisioning_ssh_has_bounded_total_timeout(self):
  instance=CLOSEOUT.Closeout.__new__(CLOSEOUT.Closeout)
  instance.key="test-key"; instance.key_bytes=b"test-key\n"
  instance.profile={"data_root":"/data/site","nginx_vhost_path":"/etc/nginx/site.conf","production_hostname":"locale.example","origin_ssh_alias":"aliyun"}
  instance.shared={"nginx_test_command":"nginx -t","nginx_reload_command":"service nginx reload"}
  with mock.patch.object(CLOSEOUT.subprocess,"run",return_value=mock.Mock(returncode=0)) as run:
   instance.provision()
  self.assertEqual(run.call_args.kwargs["timeout"],300)
 def provision(self,root,vhost,test_command,reload_command=None):
  return subprocess.run([sys.executable,"-c",CLOSEOUT.REMOTE_PROVISION,str(root),str(vhost),"locale.example","test-key.txt",test_command,reload_command or test_command],input=b"test-key\n",capture_output=True)
 def test_insert_unique_https_and_idempotence(self):
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory); vhost=root/"site.conf"; vhost.write_text(VHOST)
   self.assertEqual(self.provision(root,vhost,"/bin/true").returncode,0)
   self.assertEqual(vhost.read_text().count("location = /test-key.txt"),1)
   self.assertEqual(self.provision(root,vhost,"/bin/true").returncode,0)
   self.assertEqual(vhost.read_text().count("location = /test-key.txt"),1)
 def test_conflict_fails_closed(self):
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory); vhost=root/"site.conf"; original=VHOST.replace("location /","location = /test-key.txt { return 404; }\n location /"); vhost.write_text(original)
   self.assertNotEqual(self.provision(root,vhost,"/bin/true").returncode,0); self.assertEqual(vhost.read_text(),original)
 def test_test_or_reload_failure_restores(self):
  for test_command,reload_command in (("/bin/false","/bin/true"),("/bin/true","/bin/false")):
   with self.subTest(test_command=test_command,reload_command=reload_command), tempfile.TemporaryDirectory() as directory:
    root=pathlib.Path(directory); vhost=root/"site.conf"; vhost.write_text(VHOST)
    self.assertNotEqual(self.provision(root,vhost,test_command,reload_command).returncode,0)
    self.assertEqual(vhost.read_text(),VHOST); self.assertFalse((root/"verification"/"test-key.txt").exists())
 def test_no_dedicated_network_stack_or_hardcoded_commands(self):
  source=(ROOT/"scripts"/"indexnow-closeout.py").read_text()
  for value in ("zgocloud","ControlMaster","SOCKS","/usr/local/nginx/sbin/nginx","service nginx reload","nl-NL","it-IT"): self.assertNotIn(value,source)
if __name__=="__main__": unittest.main()
