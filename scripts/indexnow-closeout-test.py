#!/usr/bin/env python3
import importlib.util, pathlib, subprocess, sys, tempfile, unittest
from unittest import mock
ROOT=pathlib.Path(__file__).resolve().parent.parent
spec=importlib.util.spec_from_file_location("tested",ROOT/"scripts"/"indexnow-closeout.py"); CLOSEOUT=importlib.util.module_from_spec(spec); sys.modules[spec.name]=CLOSEOUT; spec.loader.exec_module(CLOSEOUT)
VHOST='''server { listen 80; server_name other.example; }
server {\n listen 443 ssl;\n server_name locale.example;\n location / { proxy_pass http://127.0.0.1:4000; }\n}\n'''
class TestProvision(unittest.TestCase):
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
  for value in ("zgocloud","ControlMaster","SOCKS","/usr/local/nginx/sbin/nginx","service nginx reload"): self.assertNotIn(value,source)
if __name__=="__main__": unittest.main()
