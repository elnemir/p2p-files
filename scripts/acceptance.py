#!/usr/bin/env python3
"""Bind executed gates to source content and refuse incomplete release acceptance."""
import argparse,hashlib,json,os,subprocess,time
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
OUT=ROOT/'.cache/evidence/gates'
GATES={
 'check':['make','check'],
 'integration':['make','integration-docker'],
 'dependencies':['make','security'],
 'source':['python3','scripts/supply_chain.py','source'],
 'licenses':['python3','scripts/supply_chain.py','licenses'],
 'images':['python3','scripts/supply_chain.py','images'],
 'fuzz':['python3','scripts/test-fuzz.py'],
 'browser':['make','e2e'],
 'ingress':['make','ingress'],
 'upgrade':['make','upgrade-drill'],
 'performance':['python3','scripts/test-performance.py'],
 'compatibility':['python3','scripts/contract_compatibility.py'],
 'verifier':['python3','scripts/release_bundle.py'],
}
DECISIONS={'public-acme-dns','host-firewall-backups','real-smtp-s3','independent-key-checkpoint-escrow','target-load-dr','real-browser-assistive-technology','independent-security-review','third-party-licenses',*[f'R-{i:02}' for i in range(1,7)]}


def source_hash():
 names=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z'],cwd=ROOT).decode().split('\0')
 digest=hashlib.sha256()
 for name in sorted(set(names)-{''}):
  # Publishing measured results in the wiki must not invalidate binary evidence.
  # Markdown links/format are still checked by the check gate and CI on each push.
  if name.endswith('.md') and (name.startswith(('wiki/','docs/')) or name in ['README.md','CHANGELOG.md','MEMORY.md']):continue
  path=ROOT/name
  digest.update(name.encode()+b'\0')
  if path.is_symlink():digest.update(b'link:'+os.readlink(path).encode())
  elif path.is_file():digest.update(hashlib.sha256(path.read_bytes()).digest())
  else:digest.update(b'deleted')
 return digest.hexdigest()


def decision_gaps(document):
 rows=document.get('decisions',[])
 if document.get('version')!=1 or len(rows)!=len(DECISIONS) or {r.get('id') for r in rows}!=DECISIONS:return ['invalid or incomplete external decision inventory']
 return [r['id'] for r in rows if r.get('status')!='accepted' or not isinstance(r.get('reviewer'),str) or not r['reviewer'].strip() or not isinstance(r.get('evidence'),str) or not r['evidence'].strip()]


def run_gate(name):
 if name=='browser' and os.environ.get('FILEX_UPDATE_VISUALS')=='1':raise SystemExit('Review new baselines before recording the browser gate')
 OUT.mkdir(parents=True,exist_ok=True)
 before=source_hash();started=time.time()
 result=subprocess.run(GATES[name],cwd=ROOT)
 after=source_hash()
 record={'gate':name,'sourceSha256':before,'sourceUnchanged':before==after,'commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'startedAtUnix':started,'finishedAtUnix':time.time(),'exitCode':result.returncode,'profile':{k:v for k,v in os.environ.items() if k in ['FILEX_PERF_MIB','FILEX_PERF_ROWS','FILEX_FUZZ_SECONDS','FILEX_E2E_PROJECTS']},'passed':result.returncode==0 and before==after}
 (OUT/(name+'.json')).write_text(json.dumps(record,indent=2)+'\n')
 if not record['passed']:raise SystemExit('Gate failed or source changed during execution: '+name)


def gate_gaps(name,result,current,now):
 gaps=[]
 if result.get('gate')!=name or not result.get('passed') or not result.get('sourceUnchanged') or result.get('exitCode')!=0 or result.get('sourceSha256')!=current:gaps.append('failed/stale gate: '+name)
 age=now-result.get('finishedAtUnix',0)
 if age<0 or age>86400:gaps.append('gate outside 24-hour window: '+name)
 profile=result.get('profile',{})
 if name=='browser' and profile.get('FILEX_E2E_PROJECTS'):gaps.append('browser matrix was restricted')
 if name=='performance':
  if int(profile.get('FILEX_PERF_MIB','32'))<2048:gaps.append('multi-GB local performance profile has not been recorded')
  if int(profile.get('FILEX_PERF_ROWS','2000'))<20000:gaps.append('large export profile has not been recorded')
 return gaps


def readiness():
 current=source_hash();gaps=[]
 for name in GATES:
  path=OUT/(name+'.json')
  if not path.exists():gaps.append('missing gate: '+name);continue
  result=json.loads(path.read_text())
  gaps.extend(gate_gaps(name,result,current,time.time()))
 gaps.extend('external decision: '+x for x in decision_gaps(json.loads((ROOT/'contracts/release-acceptance.json').read_text())))
 rows=json.loads((ROOT/'contracts/traceability.json').read_text())
 if len(rows)!=32 or any(r['status']!='verified' for r in rows):gaps.append('all 32 MUST requirements must have verified release evidence')
 dirty=subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True)
 if dirty:gaps.append('working tree is not clean')
 OUT.mkdir(parents=True,exist_ok=True)
 (OUT/'readiness.json').write_text(json.dumps({'sourceSha256':current,'ready':not gaps,'blockers':gaps},indent=2)+'\n')
 if gaps:raise SystemExit('Release blocked:\n'+'\n'.join(gaps))
 print('PASS: release acceptance complete. This command does not deploy or publish.')


if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('action',choices=['run','readiness']);parser.add_argument('gate',nargs='?',choices=GATES)
 args=parser.parse_args()
 if args.action=='readiness':readiness()
 elif args.gate:run_gate(args.gate)
 else:parser.error('run requires a gate')
