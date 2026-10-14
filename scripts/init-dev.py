#!/usr/bin/env python3
"""Create local-only bootstrap secrets without displaying or overwriting them."""
from license_fixture import new_fixture
import json
import base64
import os
from pathlib import Path
import secrets

root = Path(__file__).resolve().parents[1]
path = root / '.env'
try:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
except FileExistsError:
    raise SystemExit('.env already exists; preserved without changes.')
fixture=new_fixture()
private=root/'.cache/dev-license.json'
private.parent.mkdir(exist_ok=True)
private_fd=os.open(private,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
with os.fdopen(private_fd,'w') as keyfile:json.dump(fixture,keyfile)
with os.fdopen(fd, 'w') as f:
    f.write('POSTGRES_PASSWORD=' + secrets.token_hex(24) + '\n')
    f.write('MASTER_KEK=' + base64.b64encode(secrets.token_bytes(32)).decode() + '\n')
    for name,value in fixture['env'].items():f.write(name+"='"+value+"'\n")
print('Created local .env with mode 0600. Keep it out of source control.')
