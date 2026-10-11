#!/usr/bin/env python3
"""Disposable Compose + real browser acceptance. Removes only its generated project."""
import base64,json,os,secrets,subprocess,time,urllib.request,uuid
from pathlib import Path
root=Path(__file__).resolve().parents[1]
project='filex-e2e-'+uuid.uuid4().hex[:10]
env=dict(os.environ,POSTGRES_PASSWORD=secrets.token_hex(24),MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode(),FILEX_E2E_EMAIL='browser@example.test',FILEX_E2E_PASSWORD=secrets.token_urlsafe(24),FILEX_E2E_ADMIN='ops-admin@example.test',FILEX_E2E_PROJECT=project,PLAYWRIGHT_BROWSERS_PATH=str(root/'.cache/browsers'))
cmd=['docker','compose','--project-name',project,'-f',str(root/'deploy/compose.yaml')]
def compose(*args,**kwargs):return subprocess.run(cmd+list(args),cwd=root,env=env,check=True,text=True,**kwargs)
(root/'.cache/evidence').mkdir(parents=True,exist_ok=True)
try:
 compose('up','--detach','--build','--wait','--wait-timeout','120')
 for _ in range(60):
  try:
   with urllib.request.urlopen('http://localhost:8090/api/v1/status',timeout=3) as r:
    if r.status==200:break
  except OSError:pass
  time.sleep(1)
 else:raise RuntimeError('API did not become ready')
 compose('exec','-T','api','filex-cli','user','create',input=json.dumps({'email':env['FILEX_E2E_EMAIL'],'password':env['FILEX_E2E_PASSWORD'],'role':'user'}),stdout=subprocess.DEVNULL)
 compose('exec','-T','api','filex-cli','user','create','--json',input=json.dumps({'email':env['FILEX_E2E_ADMIN'],'password':env['FILEX_E2E_PASSWORD'],'role':'admin'}),stdout=subprocess.DEVNULL)
 subprocess.run(['npx','playwright','test'],cwd=root,env=env,check=True,timeout=360)
 print('PASS: browser transfer/report/export, accessibility, responsive layouts, privacy, operational restore/restart/reconciliation and explicit reopen.')
finally:
 compose('down','--volumes','--remove-orphans')
