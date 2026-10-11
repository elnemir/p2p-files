#!/usr/bin/env python3
"""Run tests in a new disposable PostgreSQL container, then remove that container only."""
import os
import sys
from pathlib import Path
import secrets
import subprocess
import time
import uuid

root = Path(__file__).resolve().parents[1]
name = 'filex-test-' + uuid.uuid4().hex[:12]
password = secrets.token_hex(24)
env = dict(os.environ, POSTGRES_PASSWORD=password)
def docker(*args, **kwargs):
    return subprocess.run(['docker', *args], check=True, text=True, **kwargs)
started = False
try:
    docker('run','--detach','--rm','--name',name,'--label','com.filex.purpose=integration','--publish','127.0.0.1::5432','--env','POSTGRES_PASSWORD','--env','POSTGRES_DB=filex_test','postgres:18-alpine@sha256:77f585114c32fbca283dc835b0596f4e52b51b4c6662d7810b2f4084f60a1873',env=env,stdout=subprocess.DEVNULL)
    started = True
    for attempt in range(60):
        ready = subprocess.run(['docker','exec',name,'pg_isready','-U','postgres','-d','filex_test'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        if ready.returncode == 0: break
        time.sleep(1)
    else: raise RuntimeError('Test PostgreSQL did not become ready')
    port = docker('port',name,'5432/tcp',capture_output=True).stdout.strip().split(':')[-1]
    test_env = dict(os.environ, FILEX_TEST_DATABASE_URL=f'postgres://postgres:{password}@127.0.0.1:{port}/filex_test?sslmode=disable',GOMODCACHE=str(root/'.cache/gomod'),GOCACHE=str(root/'.cache/go-build'))
    pattern = sys.argv[1] if len(sys.argv) > 1 else 'TestIntegration'
    subprocess.run(['go','test','-race','-count=1','-run',pattern,'-v','./internal/...'],cwd=root,env=test_env,check=True)
finally:
    if started:
        docker('rm','--force',name,stdout=subprocess.DEVNULL)
