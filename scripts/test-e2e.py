#!/usr/bin/env python3
"""One disposable Compose project per browser; never share restored state across engines."""
import base64,json,os,secrets,subprocess,time,urllib.request,uuid
from pathlib import Path
from license_fixture import new_fixture, activate, renewal
root=Path(__file__).resolve().parents[1]
projects=os.environ.get('FILEX_E2E_PROJECTS','chromium,firefox,webkit,mobile-chromium,mobile-webkit').split(',')
allowed={'chromium','firefox','webkit','mobile-chromium','mobile-webkit'}
if not projects or not set(projects)<=allowed:raise ValueError('Unknown browser project')
# The same pinned Linux browsers/fonts run locally and on CI. No Docker socket
# is exposed to this container; only the host runner can restart its fixtures.
image='mcr.microsoft.com/playwright:v1.64.0-noble@sha256:06a9939e57531807f8d5fd76ce44b53165ffb7d7501d87ab10e285c20b1e971f'
server='filex-browsers-'+uuid.uuid4().hex[:10]
subprocess.run(['docker','run','--detach','--init','--name',server,'--user','pwuser','--read-only','--tmpfs','/tmp:rw,mode=1777','--shm-size','1g','--cap-drop','ALL','--security-opt','no-new-privileges:true','--env','HOME=/tmp','--publish','127.0.0.1::3000','--mount',f'type=bind,src={root}/node_modules,dst=/modules/node_modules,readonly',image,'node','/modules/node_modules/playwright/cli.js','run-server','--host','0.0.0.0','--port','3000'],check=True,stdout=subprocess.DEVNULL)
try:
 port=int(subprocess.check_output(['docker','port',server,'3000'],text=True).strip().rsplit(':',1)[1])
 for _ in range(60):
  try:
   with urllib.request.urlopen(f'http://127.0.0.1:{port}/',timeout=1) as response:
    if response.status==200:break
  except OSError:time.sleep(1)
 else:raise RuntimeError('Pinned browser server did not become ready')
 for browser in projects:
  project='filex-e2e-'+uuid.uuid4().hex[:10]
  output=root/'.cache/evidence/browser'/browser
  output.mkdir(parents=True,exist_ok=True)
  env=dict(os.environ,FILEX_BROWSER_WS=f'ws://127.0.0.1:{port}/',FILEX_BROWSER_PLATFORM='linux',POSTGRES_PASSWORD=secrets.token_hex(24),MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode(),FILEX_E2E_EMAIL='browser@example.test',FILEX_E2E_PASSWORD=secrets.token_urlsafe(24),FILEX_E2E_ADMIN='ops-admin@example.test',FILEX_E2E_PROJECT=project,FILEX_E2E_OUTPUT_DIR=str(output),PLAYWRIGHT_BROWSERS_PATH=str(root/'.cache/browsers'))
  license_fixture=new_fixture()
  env.update(license_fixture['env'])
  cmd=['docker','compose','--env-file','/dev/null','--project-name',project,'-f',str(root/'deploy/compose.yaml')]
  def compose(*args,**kwargs):return subprocess.run(cmd+list(args),cwd=root,env=env,check=True,text=True,**kwargs)
  try:
   compose('up','--detach','--build','--wait','--wait-timeout','120')
   for _ in range(60):
    try:
     with urllib.request.urlopen('http://localhost:8090/api/v1/status',timeout=3) as r:
      if r.status==200:break
    except OSError:pass
    time.sleep(1)
   else:raise RuntimeError('API did not become ready')
   previous_license=activate(compose,license_fixture)
   env['FILEX_E2E_LICENSE_RENEWAL']=renewal(license_fixture,previous_license)
   compose('exec','-T','api','filex-cli','user','create',input=json.dumps({'email':env['FILEX_E2E_EMAIL'],'password':env['FILEX_E2E_PASSWORD'],'role':'user'}),stdout=subprocess.DEVNULL)
   compose('exec','-T','api','filex-cli','user','create','--json',input=json.dumps({'email':env['FILEX_E2E_ADMIN'],'password':env['FILEX_E2E_PASSWORD'],'role':'admin'}),stdout=subprocess.DEVNULL)
   args=['npx','playwright','test','--project',browser]
   if os.environ.get('FILEX_UPDATE_VISUALS')=='1':args.append('--update-snapshots')
   subprocess.run(args,cwd=root,env=env,check=True,timeout=600)
   print(f'PASS: {browser} transfer/reports/restore, a11y, keyboard, responsive and visual acceptance.',flush=True)
  finally:
   compose('down','--volumes','--remove-orphans')
 subprocess.run(['python3','scripts/test-issuer-browser.py'],cwd=root,env=dict(os.environ,FILEX_BROWSER_WS=f'ws://127.0.0.1:{port}/',NEXT_TELEMETRY_DISABLED='1'),check=True,timeout=240)
 subprocess.run(['python3','scripts/test-issuer-browser.py'],cwd=root,env=dict(os.environ,FILEX_BROWSER_WS=f'ws://127.0.0.1:{port}/',NEXT_TELEMETRY_DISABLED='1',FILEX_SETUP_BROWSER='1'),check=True,timeout=240)
finally:
 subprocess.run(['docker','rm','--force',server],check=True,stdout=subprocess.DEVNULL)
