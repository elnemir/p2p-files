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
        if path == '':
            # A new protocol profile extends the document without changing REST.
            # Once in the baseline it is compared recursively like any contract.
            additions -= {'x-webdav'}
        if additions and path not in ('/paths','/components/schemas','/components/securitySchemes','/channels','/operations'):
            result.extend(path+'/'+key+' added inside existing contract' for key in sorted(additions))
        return result
    return [] if before==after else [path+' changed']


def new_schema_findings(existing, added):
    """Allow new versioned documents, never a missing or replaced baseline file."""
    findings=[]
    ids={value.get('$id') for value in existing.values() if isinstance(value,dict) and value.get('$id')}
    for name,value in sorted(added.items()):
        if not isinstance(value,dict) or value.get('$schema')!='https://json-schema.org/draft/2020-12/schema':
            findings.append(name+': new schema must declare JSON Schema 2020-12')
            continue
        schema_id=value.get('$id')
        if not isinstance(schema_id,str) or not schema_id.startswith('https://') or schema_id in ids:
            findings.append(name+': new schema requires a unique HTTPS $id')
        else:ids.add(schema_id)
    return findings


def main():
    old_schemas=subprocess.check_output(['git','ls-tree','-r','--name-only',BASELINE,'contracts/schemas'],cwd=ROOT,text=True).splitlines()
    files=sorted({'contracts/openapi.json','contracts/asyncapi.json',*old_schemas,*[str(p.relative_to(ROOT)) for p in (ROOT/'contracts/schemas').glob('*.json')]})
    findings=[]
    existing={}
    added={}
    for path in files:
        if path not in old_schemas and path.startswith('contracts/schemas/'):
            added[path]=json.loads((ROOT/path).read_text())
            continue
        raw=subprocess.run(['git','show',BASELINE+':'+path],cwd=ROOT,capture_output=True,text=True)
        if raw.returncode:raise RuntimeError('Missing versioned baseline: '+path)
        before=json.loads(raw.stdout)
        existing[path]=before
        findings.extend(path+': '+d for d in differences(canonical(before),canonical(json.loads((ROOT/path).read_text()))))
    findings.extend(new_schema_findings(existing,added))
    out=ROOT/'.cache/evidence/compatibility.json';out.parent.mkdir(parents=True,exist_ok=True)
    out.write_text(json.dumps({'baseline':BASELINE,'files':files,'breakingChanges':findings},indent=2)+'\n')
    if findings:raise SystemExit('\n'.join(findings))
    print('PASS: OpenAPI, AsyncAPI and JSON Schemas preserve the v1 baseline')


if __name__=='__main__':main()
