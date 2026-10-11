#!/usr/bin/env python3
"""Build an unpublished verifier review bundle with SHA-256 and build metadata."""
import hashlib,json,os,shutil,subprocess
from acceptance import source_hash
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
OUT=ROOT/'.cache/evidence/release'


def main():
 OUT.mkdir(parents=True,exist_ok=True)
 records=[]
 for system,arch in [('linux','amd64'),('linux','arm64'),('darwin','amd64'),('darwin','arm64'),('windows','amd64')]:
  name=f'filex-verify-{system}-{arch}'+('.exe' if system=='windows' else '')
  path=OUT/name
  env=dict(os.environ,GOOS=system,GOARCH=arch,CGO_ENABLED='0',GOMODCACHE=str(ROOT/'.cache/gomod'),GOCACHE=str(ROOT/'.cache/go-build'))
  subprocess.run(['go','build','-trimpath','-o',str(path),'./cmd/filex-verify'],cwd=ROOT,env=env,check=True)
  records.append({'name':name,'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'bytes':path.stat().st_size})
  metadata=subprocess.check_output(['go','version','-m',str(path)],cwd=ROOT,text=True)
  (OUT/(name+'.build.txt')).write_text(metadata)
 (OUT/'SHA256SUMS').write_text(''.join(r['sha256']+'  '+r['name']+'\n' for r in records))
 (OUT/'manifest.json').write_text(json.dumps({'status':'unpublished_review_bundle','sourceSha256':source_hash(),'commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'artifacts':records,'releaseRequires':'python3 scripts/acceptance.py readiness'},indent=2)+'\n')
 shutil.copyfile(ROOT/'docs/release-notes.md',OUT/'RELEASE-NOTES.md')
 print('PASS: five verifier targets built; checksums recorded. Nothing published.')


if __name__=='__main__':main()
