#!/usr/bin/env python3
"""Record the real UI in a disposable local Compose project; no operator .env."""
import base64
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
import urllib.request
import uuid

ROOT = Path(__file__).resolve().parents[2]
IMAGE = ('mcr.microsoft.com/playwright:v1.64.0-noble@sha256:'
         '06a9939e57531807f8d5fd76ce44b53165ffb7d7501d87ab10e285c20b1e971f')
project = 'filex-tour-' + uuid.uuid4().hex[:10]
server = project + '-browser'
env = dict(os.environ, POSTGRES_PASSWORD=secrets.token_hex(24),
           MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode(),
           FILEX_TOUR_PROJECT=project, FILEX_TOUR_PASSWORD=secrets.token_urlsafe(24),
           FILEX_TOUR_OWNER='designer@example.test', FILEX_TOUR_ADMIN='operator@example.test')
compose = ['docker', 'compose', '--env-file', '/dev/null', '--project-name', project,
           '-f', str(ROOT / 'deploy/compose.yaml')]


def run(args, **kwargs):
    return subprocess.run(args, cwd=ROOT, env=env, check=True, text=True, **kwargs)


def wait_http(url):
    for _ in range(90):
        try:
            with urllib.request.urlopen(url, timeout=2) as response:
                if response.status == 200:
                    return
        except OSError:
            pass
        time.sleep(1)
    raise RuntimeError('Local tour service did not become ready')


def main():
    raw = ROOT / '.cache/tour/raw'
    raw.mkdir(parents=True, exist_ok=True)
    # Refuse to connect to any existing service on the fixed dev port.
    import socket
    with socket.socket() as probe:
        if probe.connect_ex(('127.0.0.1', 8090)) == 0:
            raise RuntimeError('Port 8090 is occupied; stop its owner explicitly first')
    try:
        run(compose + ['up', '--detach', '--build', '--wait', '--wait-timeout', '120'])
        wait_http('http://127.0.0.1:8090/api/v1/status')
        for email, role in [(env['FILEX_TOUR_OWNER'], 'user'), (env['FILEX_TOUR_ADMIN'], 'admin')]:
            run(compose + ['exec', '-T', 'api', 'filex-cli', 'user', 'create', '--json'],
                input=json.dumps({'email': email, 'password': env['FILEX_TOUR_PASSWORD'], 'role': role}),
                stdout=subprocess.DEVNULL)
        run(['docker', 'run', '--detach', '--init', '--name', server, '--user', 'pwuser',
             '--read-only', '--tmpfs', '/tmp:rw,mode=1777', '--shm-size', '1g',
             '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges:true',
             '--env', 'HOME=/tmp', '--publish', '127.0.0.1::3000', '--mount',
             f'type=bind,src={ROOT}/node_modules,dst=/modules/node_modules,readonly',
             IMAGE, 'node', '/modules/node_modules/playwright/cli.js', 'run-server',
             '--host', '0.0.0.0', '--port', '3000'], stdout=subprocess.DEVNULL)
        port = subprocess.check_output(['docker', 'port', server, '3000'], text=True).strip().rsplit(':', 1)[1]
        wait_http(f'http://127.0.0.1:{port}/')
        env['FILEX_BROWSER_WS'] = f'ws://127.0.0.1:{port}/'
        run(['node', 'scripts/tour/capture.mjs'], timeout=1200)
    finally:
        # Cleanup is deliberately scoped to the random project created above.
        subprocess.run(['docker', 'rm', '--force', server], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        run(compose + ['down', '--volumes', '--remove-orphans', '--rmi', 'local'])


if __name__ == '__main__':
    main()
