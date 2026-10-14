#!/usr/bin/env python3
"""Activate only the local Compose development instance with its ephemeral key."""
import json
from pathlib import Path
import subprocess
from license_fixture import activate
root=Path(__file__).resolve().parents[1]
fixture=json.loads((root/'.cache/dev-license.json').read_text())
def compose(*args,**kwargs):
    return subprocess.run(['docker','compose','--env-file',str(root/'.env'),'-f',str(root/'deploy/compose.yaml'),*args],cwd=root,check=True,text=True,**kwargs)
activate(compose,fixture)
print('Local development license applied. This fixture is not a release license.')
