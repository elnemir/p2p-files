#!/usr/bin/env python3
"""Build and probe a disposable local Compose stack; never targets another project."""
from license_fixture import new_fixture
import base64
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
import urllib.error
import urllib.request
import uuid

root = Path(__file__).resolve().parents[1]
project = 'filex-smoke-' + uuid.uuid4().hex[:10]
env = dict(os.environ, POSTGRES_PASSWORD=secrets.token_hex(24), MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode())
env.update(new_fixture()['env'])
command = ['docker','compose','--project-name',project,'-f',str(root/'deploy/compose.yaml')]
def compose(*args, **kwargs):
    return subprocess.run(command+list(args),cwd=root,env=env,check=True,text=True,**kwargs)
def request(path):
    try:
        with urllib.request.urlopen('http://127.0.0.1:8090'+path,timeout=3) as r:
            return r.status, r.read(), r.headers
    except urllib.error.HTTPError as e:
        return e.code, e.read(), e.headers

try:
    compose('up','--detach','--build','--wait','--wait-timeout','90')
    for attempt in range(60):
        try:
            status, body, headers = request('/api/v1/status')
            if status == 200 and json.loads(body) == {'status':'operational'}: break
        except (OSError, json.JSONDecodeError): pass
        time.sleep(1)
    else: raise RuntimeError('Aggregate status failed')
    if headers.get('Cache-Control') != 'no-store': raise RuntimeError('Missing no-store')
    status, body, _ = request('/')
    if status != 200 or 'FileX' not in body.decode(): raise RuntimeError('Next.js page missing')
    for path in ['/health','/ready','/metrics']:
        status,_,_ = request(path)
        if status != 404: raise RuntimeError('Unexpected public route '+path)
    if request('/api/v1/transfers')[0] != 401: raise RuntimeError('Unauthenticated transfer access')
    compose('stop','postgres')
    status, body, _ = request('/api/v1/status')
    if status != 503 or json.loads(body) != {'status':'unavailable'}: raise RuntimeError('DB outage did not fail closed')
    print('PASS: Next.js page, API status/privacy, internal route isolation and PostgreSQL outage.')
finally:
    compose('down','--volumes','--remove-orphans')
