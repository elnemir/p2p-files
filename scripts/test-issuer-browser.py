#!/usr/bin/env python3
"""Real issuer API/UI/signing against an explicitly synthetic TLS OIDC provider."""
import os,secrets,subprocess,time,uuid
from pathlib import Path
root=Path(__file__).resolve().parents[1]
name='filex-issuer-browser-'+uuid.uuid4().hex[:10]
password=secrets.token_hex(24)
work=root/'.cache/licensing-issuer-browser';work.mkdir(parents=True,exist_ok=True)
urlfile=work/'origin.txt';urlfile.unlink(missing_ok=True)
env=dict(os.environ,POSTGRES_PASSWORD=password,PLAYWRIGHT_BROWSERS_PATH=str(root/'.cache/browsers'),ISSUER_FIXTURE_URL_FILE=str(urlfile),GOMODCACHE=str(root/'.cache/gomod'),GOCACHE=str(root/'.cache/go-build'))
processes=[];logs=[]
try:
 subprocess.run(['docker','build','-f','deploy/Dockerfile.postgres','-t',name,'.'],cwd=root,check=True,stdout=subprocess.DEVNULL)
 subprocess.run(['docker','run','-d','--rm','--name',name,'-p','127.0.0.1::5432','-e','POSTGRES_PASSWORD',name],env=env,check=True,stdout=subprocess.DEVNULL)
 for _ in range(60):
  if subprocess.run(['docker','exec',name,'pg_isready','-U','postgres'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:break
  time.sleep(.2)
 else:raise RuntimeError('database readiness')
 port=subprocess.check_output(['docker','port',name,'5432'],text=True).strip().rsplit(':',1)[1];env['FILEX_TEST_DATABASE_URL']=f'postgres://postgres:{password}@127.0.0.1:{port}/postgres?sslmode=disable'
 subprocess.run(['go','build','-o',str(work/'fixture'),'./scripts/issuer-fixture'],cwd=root,env=env,check=True)
 commands=[(['npm','run','start','--workspace','@filex/issuer'],'web'),([str(work/'fixture')],'fixture')]
 if env.get('FILEX_SETUP_BROWSER')=='1':commands.insert(0,(['npm','run','start','--workspace','@filex/web'],'product-web'))
 for args,label in commands:
  log=(work/(label+'.log')).open('w');logs.append(log);processes.append(subprocess.Popen(args,cwd=root,env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True))
 for _ in range(100):
  if urlfile.exists():break
  if any(p.poll() is not None for p in processes):raise RuntimeError('fixture exited; see logs')
  time.sleep(.1)
 else:raise RuntimeError('fixture readiness')
 subprocess.run(['node','scripts/setup-browser.mjs' if env.get('FILEX_SETUP_BROWSER')=='1' else 'scripts/issuer-browser.mjs'],cwd=root,env=env,check=True,timeout=120)
finally:
 import signal
 for p in processes:
  if p.poll() is None:os.killpg(p.pid,signal.SIGTERM)
 for p in processes:
  try:p.wait(timeout=10)
  except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait()
 for log in logs:log.close()
 subprocess.run(['docker','rm','-f',name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
