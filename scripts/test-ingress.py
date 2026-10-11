#!/usr/bin/env python3
"""Exercise generated ingress through TLS in one disposable local Compose project."""
import base64
import http.client
import ipaddress
import json
import os
from pathlib import Path
import secrets
import socket
import ssl
import subprocess
import tempfile
import time
import uuid
import warnings

from production_config import configuration

ROOT = Path(__file__).resolve().parents[1]
PROJECT = 'filex-ingress-' + uuid.uuid4().hex[:10]
DOMAIN = 'files.example.test'
ENV = dict(os.environ, POSTGRES_PASSWORD=secrets.token_hex(24), MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode())


class LocalHTTPS(http.client.HTTPSConnection):
    def __init__(self, port, context):
        super().__init__(DOMAIN, port, context=context, timeout=5)

    def connect(self):
        # Connect only to loopback while verifying the configured DNS identity.
        stream = socket.create_connection(('127.0.0.1', self.port), self.timeout)
        self.sock = self._context.wrap_socket(stream, server_hostname=self.host)


def request(port, context, path, body=None, headers=None):
    conn = LocalHTTPS(port, context)
    values = {'Host': DOMAIN, **(headers or {})}
    if body is not None:
        values.update({'Content-Type': 'application/json', 'Origin': 'https://' + DOMAIN})
    try:
        conn.request('POST' if body is not None else 'GET', path, body=json.dumps(body) if body is not None else None, headers=values)
        response = conn.getresponse()
        return response.status, dict(response.getheaders()), response.read()
    finally:
        conn.close()


def test_generated_tls_ingress():
    with tempfile.TemporaryDirectory(prefix=PROJECT + '-') as directory:
        work = Path(directory)
        # Determine a free private subnet without changing any existing network.
        names = subprocess.check_output(['docker', 'network', 'ls', '--quiet'], text=True).split()
        occupied = []
        if names:
            for n in json.loads(subprocess.check_output(['docker', 'network', 'inspect', *names], text=True)):
                for entry in (n.get('IPAM', {}).get('Config') or []):
                    if entry.get('Subnet'):
                        occupied.append(ipaddress.ip_network(entry['Subnet']))
        subnet = None
        for index in range(80, 240):
            candidate = ipaddress.ip_network('172.30.' + str(index) + '.0/29')
            if not any(candidate.overlaps(n) for n in occupied if n.version == 4):
                subnet = str(candidate)
                break
        if subnet is None:
            raise RuntimeError('no non-overlapping test subnet found')
        config = configuration(DOMAIN, 'operator@example.test', subnet, 'example.test/api@sha256:' + 'a' * 64, 'example.test/web@sha256:' + 'b' * 64)
        compose = config['compose.json']
        compose['name'] = PROJECT
        for name in ['api', 'worker', 'web']:
            service = compose['services'][name]
            service['image'] = PROJECT + ('-web' if name == 'web' else '-api')
            service['build'] = {'context': str(ROOT), 'dockerfile': 'deploy/Dockerfile.web' if name == 'web' else 'deploy/Dockerfile.api'}
            if name != 'web':
                service['environment']['APP_ENV'] = 'development'
        compose['services']['api']['environment'].update({'DEV_PAYLOAD_ROOT': '/data/payload', 'DEV_ARTIFACT_ROOT': '/data/artifacts', 'DEV_BACKUP_ROOT': '/data/backups'})
        for service in compose['services'].values():
            service['restart'] = 'no'
        proxy = compose['services']['traefik']
        proxy['ports'] = ['127.0.0.1::8080', '127.0.0.1::8443']
        proxy['volumes'] = [v for v in proxy['volumes'] if not v.startswith('./acme:')]
        proxy['volumes'].append('./certs:/etc/traefik/certs:ro')
        config['traefik-static.yaml'].pop('certificatesResolvers')
        for name in ['api', 'web']:
            config['routes.yaml']['http']['routers'][name]['tls'] = {}
        config['routes.yaml']['tls']['certificates'] = [{'certFile': '/etc/traefik/certs/server.crt', 'keyFile': '/etc/traefik/certs/server.key'}]
        certs = work / 'certs'
        certs.mkdir(mode=0o755)
        def openssl(*args):
            subprocess.run(['openssl', *args], cwd=certs, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        (certs / 'ca.cnf').write_text('[req]\ndistinguished_name=dn\nx509_extensions=ca\nprompt=no\n[dn]\nCN=FileX disposable CA\n[ca]\nbasicConstraints=critical,CA:TRUE,pathlen:0\nkeyUsage=critical,keyCertSign,cRLSign\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid:always\n')
        openssl('req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-keyout', 'ca.key', '-out', 'ca.crt', '-days', '1', '-config', 'ca.cnf')
        openssl('req', '-newkey', 'rsa:2048', '-nodes', '-keyout', 'server.key', '-out', 'server.csr', '-subj', '/CN=' + DOMAIN)
        (certs / 'extensions.cnf').write_text('subjectAltName=DNS:' + DOMAIN + '\nextendedKeyUsage=serverAuth\nbasicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n')
        openssl('x509', '-req', '-in', 'server.csr', '-CA', 'ca.crt', '-CAkey', 'ca.key', '-CAcreateserial', '-out', 'server.crt', '-days', '1', '-extfile', 'extensions.cnf')
        # Synthetic one-run server key must be readable by the non-root container.
        (certs / 'server.key').chmod(0o644)
        for name, value in config.items():
            (work / name).write_text(json.dumps(value, indent=2) + '\n')
        command = ['docker', 'compose', '--project-name', PROJECT, '-f', str(work / 'compose.json')]
        def dc(*args, **kwargs):
            return subprocess.run(command + list(args), env=ENV, cwd=work, check=True, text=True, **kwargs)
        def cli(*args, value=None):
            result = dc('exec', '-T', 'api', 'filex-cli', *args, '--json', input=json.dumps(value) if value is not None else None, capture_output=True)
            return json.loads(result.stdout)
        def port(inner):
            return int(dc('port', 'traefik', str(inner), capture_output=True).stdout.strip().rsplit(':', 1)[1])
        context = ssl.create_default_context(cafile=str(certs / 'ca.crt'))
        password = secrets.token_urlsafe(24)
        try:
            dc('up', '--detach', '--build', '--wait', '--wait-timeout', '120')
            https_port, http_port = port(8443), port(8080)
            last_probe = 'not attempted'
            for attempt in range(30):
                try:
                    code = request(https_port, context, '/api/v1/status')[0]
                    if code == 200:
                        break
                    last_probe = 'HTTP ' + str(code)
                except ssl.SSLCertVerificationError as error:
                    raise RuntimeError('test CA validation failed: ' + error.verify_message) from error
                except (OSError, http.client.HTTPException) as error:
                    last_probe = type(error).__name__ + ': ' + str(error)
                if attempt in (0, 10, 20):
                    print('TLS readiness: ' + last_probe, flush=True)
                time.sleep(1)
            else:
                raise RuntimeError('TLS ingress did not become ready: ' + last_probe)
            cli('user', 'create', value={'email': 'admin@example.test', 'password': password, 'role': 'admin'})
            active = cli('settings', 'show')
            active['value'].update(config['settings-ingress.json'])
            cli('settings', 'apply', value={'expectedRevision': active['revision'], 'value': active['value']})
            dc('restart', 'api', 'worker')
            for _ in range(30):
                try:
                    if cli('config', 'ingress-check')['ingressConfigured']:
                        break
                except subprocess.CalledProcessError:
                    pass
                time.sleep(1)
            else:
                raise RuntimeError('configured API did not restart')
            conn = http.client.HTTPConnection('127.0.0.1', http_port, timeout=5)
            conn.request('GET', '/', headers={'Host': DOMAIN})
            response = conn.getresponse()
            if response.status not in (301, 308) or response.getheader('Location') != 'https://' + DOMAIN + '/':
                raise RuntimeError('incorrect HTTP to HTTPS redirect')
            response.read()
            conn.close()
            try:
                request(https_port, ssl.create_default_context(), '/')
            except ssl.SSLCertVerificationError:
                pass
            else:
                raise RuntimeError('untrusted test CA accepted')
            tls12 = ssl.create_default_context(cafile=str(certs / 'ca.crt'))
            tls12.minimum_version = ssl.TLSVersion.TLSv1_2
            tls12.maximum_version = ssl.TLSVersion.TLSv1_2
            if request(https_port, tls12, '/')[0] != 200:
                raise RuntimeError('TLS 1.2 unavailable')
            legacy = ssl.create_default_context(cafile=str(certs / 'ca.crt'))
            with warnings.catch_warnings():
                warnings.simplefilter('ignore', DeprecationWarning)
                legacy.minimum_version = ssl.TLSVersion.TLSv1_1
                legacy.maximum_version = ssl.TLSVersion.TLSv1_1
            legacy.set_ciphers('ALL:@SECLEVEL=0')
            try:
                request(https_port, legacy, '/')
            except ssl.SSLError as error:
                if error.reason != 'TLSV1_ALERT_PROTOCOL_VERSION':
                    raise RuntimeError('client did not establish server TLS version rejection') from error
            else:
                raise RuntimeError('TLS 1.1 accepted')
            if request(https_port, context, '/', headers={'Host': 'unknown.example.test'})[0] != 404:
                raise RuntimeError('unknown virtual host accepted')
            for path in ['/', '/health', '/ready', '/metrics']:
                status, headers, _ = request(https_port, context, path)
                if status != (200 if path == '/' else 404):
                    raise RuntimeError('unexpected public route status: ' + path)
                if 'max-age=31536000' not in headers.get('Strict-Transport-Security', '') or headers.get('Referrer-Policy') != 'no-referrer':
                    raise RuntimeError('missing TLS privacy headers')
            credentials = {'email': 'admin@example.test', 'password': password}
            status, headers, _ = request(https_port, context, '/api/v1/auth/login', credentials, {'X-Forwarded-For': '203.0.113.222', 'X-Forwarded-Proto': 'http'})
            if status != 200 or 'Secure' not in headers.get('Set-Cookie', ''):
                raise RuntimeError('trusted HTTPS login/cookie failed: status=' + str(status) + ', secureCookie=' + str('Secure' in headers.get('Set-Cookie', '')))
            # A forged public header cannot move the rate-limit bucket to the spoofed IP.
            import hashlib
            forged_bucket = hashlib.sha256(b'ip:203.0.113.222').hexdigest()
            count = dc('exec', '-T', 'postgres', 'psql', '-U', 'filex', '-d', 'filex', '-At', '-c', "SELECT count(*) FROM auth_limits WHERE key='" + forged_bucket + "'", capture_output=True).stdout.strip()
            if count != '0':
                raise RuntimeError('client spoofed rate-limit identity through Traefik')
            active = cli('settings', 'show')
            active['value']['networkPolicy']['allowedClientCidrs'] = ['192.0.2.0/24']
            cli('settings', 'apply', value={'expectedRevision': active['revision'], 'value': active['value']})
            dc('restart', 'api', 'worker')
            for _ in range(30):
                status, _, _ = request(https_port, context, '/api/v1/auth/login', credentials, {'X-Forwarded-For': '192.0.2.5'})
                if status == 403:
                    break
                time.sleep(1)
            else:
                raise RuntimeError('client allowlist bypassed')
            cli('network', 'reset', value={'reason': 'Repair synthetic allowlist lockout'})
            dc('restart', 'api', 'worker')
            for _ in range(30):
                status, _, _ = request(https_port, context, '/api/v1/auth/login', credentials)
                if status == 200:
                    break
                time.sleep(1)
            else:
                raise RuntimeError('host reset did not repair access')
            state = cli('operations', 'status')
            if state['state']['mode'] == 'open':
                raise RuntimeError('network reset reopened data plane')
            print('PASS: generated TLS ingress, TLS 1.2 success/TLS 1.1 rejection, CA validation, redirect/HSTS, secure cookies, Host/private routes, proxy spoof rejection, CIDR denial, host reset and restart fencing. Public ACME was not contacted.')
        finally:
            dc('down', '--volumes', '--remove-orphans')


if __name__ == '__main__':
    test_generated_tls_ingress()
