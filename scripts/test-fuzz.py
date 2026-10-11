#!/usr/bin/env python3
"""Run bounded fuzz campaigns; seeds alone do not count as fuzz execution."""
import json,os,subprocess,time
from pathlib import Path
root=Path(__file__).resolve().parents[1]
out=root/'.cache/evidence/fuzz';out.mkdir(parents=True,exist_ok=True)
seconds=int(os.environ.get('FILEX_FUZZ_SECONDS','15'))
if seconds<5 or seconds>3600:raise ValueError('Fuzz duration outside 5..3600 seconds')
env=dict(os.environ,GOMODCACHE=str(root/'.cache/gomod'),GOCACHE=str(root/'.cache/go-build'))
results=[]
for package,target in [('domain','FuzzPathKey'),('cryptobox','FuzzPayloadAuthentication'),('httpapi','FuzzByteRangeBounds')]:
 started=time.monotonic()
 with (out/(target+'.log')).open('w') as log:
  subprocess.run(['go','test','./internal/'+package,'-run=^$','-fuzz=^'+target+'$','-fuzztime='+str(seconds)+'s','-parallel=2'],cwd=root,env=env,stdout=log,stderr=subprocess.STDOUT,check=True,timeout=seconds+120)
 results.append({'target':target,'seconds':time.monotonic()-started,'passed':True})
(out/'results.json').write_text(json.dumps(results,indent=2)+'\n')
print(json.dumps(results))
