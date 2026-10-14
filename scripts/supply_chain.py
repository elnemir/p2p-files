#!/usr/bin/env python3
"""Produce reviewable source/image evidence; scanners never upload repository data."""
import argparse
import csv
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import uuid
from security_tools import ROOT, tool

OUT = ROOT / '.cache/evidence/supply-chain'
ENV = dict(os.environ, GOMODCACHE=str(ROOT/'.cache/gomod'), GOCACHE=str(ROOT/'.cache/go-build'))


def run(args, **kwargs):
    return subprocess.run([str(x) for x in args], cwd=ROOT, env=kwargs.pop('env', ENV), check=True, text=True, **kwargs)


def scan(trivy, target, name, *options):
    report = OUT / (name + '.json')
    run([trivy, target, '--quiet', '--cache-dir', ROOT/'.cache/trivy', '--scanners', 'vuln,license', '--format', 'json', '--output', report, *options])
    run([trivy, target, '--quiet', '--cache-dir', ROOT/'.cache/trivy', '--format', 'cyclonedx', '--output', OUT/(name+'.cdx.json'), *options])
    data = json.loads(report.read_text())
    findings = [{k:v.get(k) for k in ['VulnerabilityID','PkgName','InstalledVersion','FixedVersion','Severity']} for r in data.get('Results',[]) for v in r.get('Vulnerabilities',[])]
    high = [v for v in findings if v['Severity'] in ('HIGH','CRITICAL')]
    return {'name':name, 'findings':findings, 'highCritical':len(high)}


def source():
    gitleaks, trivy = tool('gitleaks'), tool('trivy')
    run([gitleaks, 'git', '--redact=100', '--report-format', 'json', '--report-path', OUT/'secrets-history.json', '--log-opts=--all'])
    # Include new nonignored files while excluding local credentials/caches and Git internals.
    names = run(['git','ls-files','--cached','--others','--exclude-standard','-z'], capture_output=True).stdout.split('\0')
    with tempfile.TemporaryDirectory(prefix='filex-source-') as directory:
        target = Path(directory)
        for name in sorted(set(names)-{''}):
            path = ROOT/name
            if not path.is_file() or path.is_symlink():
                continue
            dest = target/name
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(path,dest)
        run([gitleaks,'dir','--redact=100','--config',ROOT/'.gitleaks.toml','--report-format','json','--report-path',OUT/'secrets-worktree.json',target])
        return scan(trivy,'fs','source','--include-dev-deps',target)


def licenses():
    destination = ROOT/'.cache/security-tools/bin'
    destination.mkdir(parents=True,exist_ok=True)
    # Pin the module version; Go's checksum database verifies its module archive.
    run(['go','install','github.com/google/go-licenses/v2@v2.0.1'],env=dict(ENV,GOBIN=str(destination)))
    with (OUT/'go-licenses.csv').open('w') as output:
        run([destination/'go-licenses','report','./cmd/...','--ignore=filex.local/filex'],stdout=output)
    npm=[]
    for path, package in json.loads((ROOT/'package-lock.json').read_text())['packages'].items():
        if 'node_modules/' not in path or package.get('link'):
            continue
        npm.append({'path':path,'version':package['version'],'license':package.get('license','UNKNOWN'),'development':package.get('dev',False),'optional':package.get('optional',False),'integrity':package.get('integrity')})
    go=[dict(zip(['module','source','license'],r)) for r in csv.reader((OUT/'go-licenses.csv').open())]
    allowed={'MIT','Apache-2.0','ISC','BSD-2-Clause','BSD-3-Clause','0BSD','BlueOak-1.0.0','Python-2.0','CC-BY-4.0','CC0-1.0','(MIT OR CC0-1.0)'}
    review=[x for x in npm+go if x['license'] not in allowed]
    inventory={'npm':npm,'go':go,'bundledFont':{'path':'internal/reports/fonts/LICENSE','sha256':hashlib.sha256((ROOT/'internal/reports/fonts/LICENSE').read_bytes()).hexdigest()},'requiresOwnerReview':review,'policy':'Unapproved copyleft or unknown licenses block release; inventory is not approval.'}
    (OUT/'licenses.json').write_text(json.dumps(inventory,indent=2)+'\n')
    if any(x['license']=='UNKNOWN' for x in npm+go):
        raise RuntimeError('Incomplete license inventory')
    return {'npmPackages':len(npm),'goLibraries':len(go),'requiresOwnerReview':len(review)}


def images():
    trivy=tool('trivy')
    identifier=uuid.uuid4().hex[:10]
    targets=[]
    for name in ['api','web','postgres','redis','traefik','issuer','issuer-web']:
        tag=f'filex-scan-{identifier}-{name}'
        run(['docker','build','-f',f'deploy/Dockerfile.{name}','-t',tag,'.'])
        image=run(['docker','image','inspect','--format','{{.Id}}',tag],capture_output=True).stdout.strip()
        targets.append((name,image))
    results=[]
    for name, ref in targets:
        r=scan(trivy,'image',name,'--image-src','docker',ref)
        r['image']=ref
        results.append(r)
    return results


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('scope', choices=['source','images','licenses'])
    args=parser.parse_args()
    OUT.mkdir(parents=True,exist_ok=True)
    result={'source':source,'images':images,'licenses':licenses}[args.scope]()
    (OUT/(args.scope+'-summary.json')).write_text(json.dumps(result,indent=2)+'\n')
    if args.scope!='licenses' and any(r['highCritical'] for r in (result if isinstance(result,list) else [result])):
        raise RuntimeError('HIGH/CRITICAL vulnerabilities block acceptance; see redacted evidence')
    print(json.dumps(result if args.scope=='licenses' else [{'name':r['name'],'highCritical':r['highCritical'],'findings':len(r['findings'])} for r in (result if isinstance(result,list) else [result])]))


if __name__=='__main__':
    main()
