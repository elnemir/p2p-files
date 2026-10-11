#!/usr/bin/env python3
"""Conservative v1 compatibility gate against the last accepted contract baseline.

Existing operations and schemas are immutable apart from descriptive metadata.
New paths/components are allowed. Semantic changes require a reviewed new baseline
or a major API version, never an automatic scanner waiver.
"""
import json,subprocess
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
BASELINE='f68fc08bb795ae1903333336efe4620ce76cb4ec'
DOCUMENTARY={'description','summary','example','examples','externalDocs'}
NAMED_MAPS={'properties','patternProperties','$defs','definitions','schemas','paths','channels','operations','headers','responses','parameters','requestBodies','securitySchemes','links','messages','content'}


def canonical(value,parent=''):
    # Schema property names and literal enum/default values are contract data,
    # even when a field happens to be called description or examples.
    if parent in {'enum','const','default'}:return value
    if isinstance(value,dict):return {k:canonical(v,k) for k,v in value.items() if parent in NAMED_MAPS or k not in DOCUMENTARY}
    if isinstance(value,list):return [canonical(v,parent) for v in value]
    return value


def differences(before,after,path=''):
    if isinstance(before,dict) and isinstance(after,dict):
        result=[]
        for key,value in before.items():
            if key not in after:result.append(path+'/'+key+' removed')
            else:result.extend(differences(value,after[key],path+'/'+key))
        # Adding an operation/schema is compatible. Changing an existing schema
        # (including additional output properties) may break strict validators.
        additions=set(after)-set(before)
        if additions and path not in ('/paths','/components/schemas','/channels','/operations'):
            result.extend(path+'/'+key+' added inside existing contract' for key in sorted(additions))
        return result
    return [] if before==after else [path+' changed']


def main():
    old_schemas=subprocess.check_output(['git','ls-tree','-r','--name-only',BASELINE,'contracts/schemas'],cwd=ROOT,text=True).splitlines()
    files=sorted({'contracts/openapi.json','contracts/asyncapi.json',*old_schemas,*[str(p.relative_to(ROOT)) for p in (ROOT/'contracts/schemas').glob('*.json')]})
    findings=[]
    for path in files:
        raw=subprocess.run(['git','show',BASELINE+':'+path],cwd=ROOT,capture_output=True,text=True)
        if raw.returncode:raise RuntimeError('Missing versioned baseline: '+path)
        findings.extend(path+': '+d for d in differences(canonical(json.loads(raw.stdout)),canonical(json.loads((ROOT/path).read_text()))))
    out=ROOT/'.cache/evidence/compatibility.json';out.parent.mkdir(parents=True,exist_ok=True)
    out.write_text(json.dumps({'baseline':BASELINE,'files':files,'breakingChanges':findings},indent=2)+'\n')
    if findings:raise SystemExit('\n'.join(findings))
    print('PASS: OpenAPI, AsyncAPI and JSON Schemas preserve the v1 baseline')


if __name__=='__main__':main()
