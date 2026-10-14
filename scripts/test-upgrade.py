#!/usr/bin/env python3
"""Replace a pinned baseline image in a disposable project and measure downtime."""
import base64
import copy
import hashlib
import http.client
import json
import os
from pathlib import Path
from license_fixture import new_fixture, activate
import secrets
import subprocess
import tarfile
import tempfile
import threading
import time
from urllib.parse import urlsplit
import uuid

ROOT = Path(__file__).resolve().parents[1]
BASELINE = '28ab4f36e15a0b5e052b23e0f058bdda7a1c9d12'
EXPECTED_MIGRATIONS = ['0017_native_webdav.sql', '0018_branding.sql', '0019_federated_auth.sql', '0020_offline_licensing.sql']


class Client:
    def __init__(self, port):
        self.port, self.cookie, self.csrf = port, '', ''

    def request(self, method, path, value=None, headers=None, after_read=None):
        values = {'Origin': 'http://localhost:8090', 'Cookie': self.cookie, 'X-CSRF-Token': self.csrf}
        if isinstance(value, dict):
            value = json.dumps(value).encode()
            values['Content-Type'] = 'application/json'
        values.update(headers or {})
        conn = http.client.HTTPConnection('127.0.0.1', self.port, timeout=5)
        try:
            conn.request(method, urlsplit(path).path, body=value, headers=values)
            response = conn.getresponse()
            body, result = response.read(), dict(response.getheaders())
            if 'Set-Cookie' in result:
                self.cookie = result['Set-Cookie'].split(';', 1)[0]
            if after_read is not None:
                after_read(response.status)
            return response.status, body, result
        finally:
            conn.close()

    def expect(self, method, path, status, value=None, headers=None, after_read=None):
        def completed(actual):
            if actual == status and after_read is not None:
                after_read()
        got, body, result = self.request(method, path, value, headers, completed)
        if got != status:
            raise RuntimeError(f'HTTP {method} probe expected {status}, got {got}')
        return body, result


def wait_for_database(read, expected, label, timeout=10):
    """Observe the server commit while the download connection remains open."""
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if read() == expected:
            return
        time.sleep(0.05)
    raise RuntimeError('Server commit was not observed: ' + label)


def test_controlled_image_upgrade():
    project = 'filex-upgrade-' + uuid.uuid4().hex[:10]
    env = dict(os.environ, POSTGRES_PASSWORD=secrets.token_hex(24), MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode())
    license_fixture=new_fixture()
    env.update(license_fixture['env'])
    images = [project + '-baseline', project + '-candidate']
    with tempfile.TemporaryDirectory(prefix=project + '-') as directory:
        work = Path(directory)
        baseline = work / 'baseline'
        baseline.mkdir()
        archive = work / 'baseline.tar'
        with archive.open('wb') as output:
            subprocess.run(['git', 'archive', BASELINE, 'cmd', 'internal', 'go.mod', 'go.sum', 'deploy/Dockerfile.api'], cwd=ROOT, stdout=output, check=True)
        with tarfile.open(archive) as source:
            source.extractall(baseline, filter='data')
        # Resolve only disposable credentials, never the operator's .env file.
        config_cmd = ['docker', 'compose', '--env-file', '/dev/null', '--project-name', project, '-f', str(ROOT / 'deploy/compose.yaml'), 'config', '--format', 'json']
        config = json.loads(subprocess.check_output(config_cmd, cwd=ROOT, env=env, text=True))
        for name in ['api', 'worker']:
            config['services'][name].pop('build', None)
            config['services'][name]['image'] = images[0]
        for service in config['services'].values():
            service['restart'] = 'no'
        config['services']['traefik']['ports'] = ['127.0.0.1::8080']
        candidate = copy.deepcopy(config)
        for name in ['api', 'worker']:
            candidate['services'][name]['image'] = images[1]
        bad_key = copy.deepcopy(candidate)
        bad_key['services']['api']['environment']['MASTER_KEK'] = base64.b64encode(secrets.token_bytes(32)).decode()
        for name, data in [('baseline.json', config), ('candidate.json', candidate), ('bad-key.json', bad_key)]:
            path = work / name
            path.write_text(json.dumps(data))
            path.chmod(0o600)

        def dc(profile, *args, **kwargs):
            kwargs.setdefault('check', True)
            return subprocess.run(['docker', 'compose', '--project-name', project, '-f', str(work / (profile + '.json')), *args], cwd=work, env=env, text=True, **kwargs)

        def cli(profile, *args, value=None, ok=True, failure=None):
            result = dc(profile, 'run', '--rm', '--no-deps', '-T', '--entrypoint', 'filex-cli', 'api', *args, '--json', input=json.dumps(value) if value is not None else None, capture_output=True, check=False)
            if (result.returncode == 0) != ok:
                raise RuntimeError('CLI probe returned unexpected exit status: ' + ' '.join(args))
            if failure is not None and failure not in result.stderr:
                raise RuntimeError('CLI rejected the probe for an unexpected reason: ' + ' '.join(args))
            return json.loads(result.stdout) if ok else None

        def sql(statement):
            return dc('candidate', 'exec', '-T', 'postgres', 'psql', '-v', 'ON_ERROR_STOP=1', '-U', 'filex', '-d', 'filex', '-At', '-c', statement, capture_output=True).stdout.strip()

        def wait_failure(profile, reason):
            dc(profile, 'up', '--detach', '--no-deps', '--force-recreate', 'api')
            cid = dc(profile, 'ps', '--all', '--quiet', 'api', capture_output=True).stdout.strip()
            for _ in range(60):
                state = json.loads(subprocess.check_output(['docker', 'inspect', '--format', '{{json .State}}', cid], text=True))
                if state['Status'] == 'exited':
                    if state['ExitCode'] == 0:
                        raise RuntimeError('incompatible API exited successfully')
                    logs = subprocess.check_output(['docker', 'logs', cid], text=True, stderr=subprocess.STDOUT)
                    if reason not in logs:
                        raise RuntimeError('candidate failed for an unexpected startup reason')
                    return
                time.sleep(0.2)
            raise RuntimeError('incompatible candidate did not fail startup')

        samples, stop, sampler = [], threading.Event(), None
        try:
            for context, tag in [(baseline, images[0]), (ROOT, images[1])]:
                subprocess.run(['docker', 'build', *[arg for k,v in license_fixture['env'].items() for arg in ['--build-arg', k+'='+v]], '--file', str(context / 'deploy/Dockerfile.api'), '--tag', tag, str(context)], cwd=ROOT, check=True)
            dc('baseline', 'up', '--detach', '--build', '--wait', '--wait-timeout', '120')
            port = int(dc('baseline', 'port', 'traefik', '8080', capture_output=True).stdout.strip().rsplit(':', 1)[1])
            owner = Client(port)
            for _ in range(60):
                try:
                    if owner.request('GET', '/api/v1/status')[0] == 200:
                        break
                except (OSError, http.client.HTTPException):
                    pass
                time.sleep(0.5)
            else:
                raise RuntimeError('baseline API not ready')
            password = secrets.token_urlsafe(24)
            cli('baseline', 'user', 'create', value={'email': 'upgrade@example.test', 'password': password, 'role': 'admin'})
            login, _ = owner.expect('POST', '/api/v1/auth/login', 200, {'email': 'upgrade@example.test', 'password': password})
            owner.csrf = json.loads(login)['csrfToken']
            body, _ = owner.expect('POST', '/api/v1/transfers', 201)
            transfer = json.loads(body)['id']
            payload = b'Controlled upgrade retains immutable payload.\n' * 2048
            _, headers = owner.expect('POST', '/api/v1/transfers/' + transfer + '/uploads', 201, headers={'Tus-Resumable': '1.0.0', 'Upload-Length': str(len(payload)), 'Upload-Metadata': 'filename ' + base64.b64encode(b'upgrade.txt').decode()})
            upload = headers['Location']
            owner.expect('PATCH', upload, 204, payload, {'Tus-Resumable': '1.0.0', 'Upload-Offset': '0', 'Content-Type': 'application/offset+octet-stream'})
            owner.expect('POST', upload + '/finalize', 204)
            body, _ = owner.expect('GET', '/api/v1/transfers/' + transfer, 200)
            revision = json.loads(body)['revision']
            body, _ = owner.expect('POST', '/api/v1/transfers/' + transfer + '/publish', 200, {'expectedRevision': revision, 'ttlSeconds': 3600, 'commonLink': True})
            token = json.loads(body)['links'][0]['url'].split('#token=')[1]
            recipient = Client(port)
            recipient.expect('POST', '/api/v1/recipient/session', 200, {'token': token})
            refs, _ = recipient.expect('GET', '/api/v1/recipient/files', 200)
            lease, _ = recipient.expect('POST', '/api/v1/recipient/downloads', 201, {'fileRef': json.loads(refs)[0]['ref']})
            download = json.loads(lease)['url']
            first, _ = recipient.expect('GET', download, 206, headers={'Range': 'bytes=0-31'},
                after_read=lambda: wait_for_database(lambda: sql("SELECT count(*) FROM downloads WHERE coverage='[[0,32]]'::jsonb"), '1', 'initial Range coverage'))
            if first != payload[:32]:
                raise RuntimeError('baseline Range payload mismatch')
            before = cli('baseline', 'upgrade', 'check')
            preflight = cli('candidate', 'upgrade', 'check')
            if before['pending'] or not before['compatible'] or preflight != {'applied': before['applied'], 'pending': EXPECTED_MIGRATIONS, 'compatible': True}:
                raise RuntimeError('unexpected baseline/candidate migration plan')
            ledger_before = sql('SELECT name || chr(9) || sha256 FROM schema_migrations ORDER BY name')
            image_ids = [subprocess.check_output(['docker', 'image', 'inspect', '--format', '{{.Id}}', tag], text=True).strip() for tag in images]
            if image_ids[0] == image_ids[1]:
                raise RuntimeError('drill did not build distinct revisions')
            cid = dc('baseline', 'ps', '--quiet', 'api', capture_output=True).stdout.strip()
            if subprocess.check_output(['docker', 'inspect', '--format', '{{.Image}}', cid], text=True).strip() != image_ids[0]:
                raise RuntimeError('baseline container image mismatch')
            # No recovery job may race downtime preparation.
            dc('baseline', 'stop', 'worker')

            def sample():
                probe = Client(port)
                while not stop.is_set():
                    try:
                        code = probe.request('GET', '/api/v1/status')[0]
                    except (OSError, http.client.HTTPException):
                        code = 0
                    samples.append((time.monotonic(), code))
                    stop.wait(0.2)
            sampler = threading.Thread(target=sample, daemon=True)
            sampler.start()
            started = time.monotonic()
            prepared = cli('baseline', 'upgrade', 'prepare', value={'reason': 'Isolated controlled image upgrade drill'})
            if not prepared['drained'] or not prepared['requiresRestart'] or prepared['mode'] == 'open':
                raise RuntimeError('upgrade did not fence and drain')
            owner.expect('GET', '/api/v1/auth/me', 503)
            recipient.expect('GET', download, 503)
            dc('baseline', 'stop', 'api', 'worker', 'web')
            provider = sql("SELECT id FROM storage_providers WHERE purpose='backups' AND enabled ORDER BY id LIMIT 1")
            backup = cli('baseline', 'backup', 'create', value={'providerId': provider, 'reason': 'Verified pre-upgrade copy after drain'})
            verified = cli('baseline', 'backup', 'verify', value={'backupId': backup['id']})
            if not verified['verified']:
                raise RuntimeError('pre-upgrade backup not verified')
            # Corrupt only the isolated fixture's ledger, prove refusal, then
            # remove exactly this synthetic row. This is never a repair recipe.
            sql("INSERT INTO schema_migrations(name,sha256) VALUES('9999_upgrade_fixture.sql','fixture-only')")
            cli('candidate', 'upgrade', 'check', ok=False)
            wait_failure('candidate', 'database migration failed')
            if sql("SELECT mode FROM ops_control") == 'open':
                raise RuntimeError('failed migration reopened gate')
            sql("DELETE FROM schema_migrations WHERE name='9999_upgrade_fixture.sql' AND sha256='fixture-only'")
            wait_failure('bad-key', 'critical key check failed')
            if sql("SELECT mode FROM ops_control") == 'open':
                raise RuntimeError('failed key check reopened gate')
            dc('candidate', 'up', '--detach', '--no-deps', '--force-recreate', 'api', 'worker', 'web')
            activate(lambda *args, **kwargs: dc('candidate', *args, **kwargs),license_fixture)
            for _ in range(60):
                try:
                    if owner.request('GET', '/api/v1/auth/me')[0] == 200:
                        break
                except (OSError, http.client.HTTPException):
                    pass
                time.sleep(0.5)
            else:
                raise RuntimeError('candidate API/session not ready')
            cid = dc('candidate', 'ps', '--quiet', 'api', capture_output=True).stdout.strip()
            if subprocess.check_output(['docker', 'inspect', '--format', '{{.Image}}', cid], text=True).strip() != image_ids[1]:
                raise RuntimeError('candidate container image mismatch')
            owner.expect('GET', '/api/v1/transfers/' + transfer, 503)
            recipient.expect('GET', download, 503)
            after = cli('candidate', 'upgrade', 'check')
            if after != {'applied': before['applied'] + len(EXPECTED_MIGRATIONS), 'pending': [], 'compatible': True}:
                raise RuntimeError('candidate did not apply exactly the expected migrations')
            expected_ledger = ledger_before.splitlines() + [name + '\t' + hashlib.sha256((ROOT / 'internal/database/migrations' / name).read_bytes()).hexdigest() for name in EXPECTED_MIGRATIONS]
            if sql('SELECT name || chr(9) || sha256 FROM schema_migrations ORDER BY name').splitlines() != expected_ledger:
                raise RuntimeError('migration changed historical checksums or applied unexpected SQL')
            # The old binary must refuse the new schema. Rollback requires a
            # separately reviewed restore, never silently starting old code.
            cli('baseline', 'upgrade', 'check', ok=False)
            if sql("SELECT count(*) FROM files WHERE created_at IS NULL OR modified_at IS NULL OR dav_properties <> '[]'::jsonb OR staged_until IS NOT NULL") != '0':
                raise RuntimeError('migration metadata baseline is invalid')
            if sql("SELECT count(*) FROM dav_locks") != '0':
                raise RuntimeError('migration unexpectedly created locks')
            # Verification also enforces restore schema compatibility. The pre-DDL
            # backup remains a verified schema-16 recovery point, not a schema-19
            # import candidate. Do not weaken that guard to pass an upgrade drill.
            cli('candidate', 'backup', 'verify', value={'backupId': backup['id']}, ok=False, failure='"code":"command_failed"')
            post_backup = cli('candidate', 'backup', 'create', value={'providerId': provider, 'reason': 'Verify a fresh schema-19 recovery point'})
            verified_after = cli('candidate', 'backup', 'verify', value={'backupId': post_backup['id']})
            if not verified_after['verified']:
                raise RuntimeError('candidate backup verification failed')
            reconciliation = cli('candidate', 'recovery', 'reconcile', value={'reason': 'Review post-upgrade audit crypto storage'})
            state = cli('candidate', 'operations', 'status')
            job = next(j for j in state['jobs'] if j['id'] == reconciliation['id'])
            report = cli('candidate', 'recovery', 'report', value={'reconciliationId': job['result']['reconciliationId']})
            if not report['passed'] or report['findings']:
                raise RuntimeError('post-upgrade reconciliation has findings')
            cli('candidate', 'recovery', 'reopen', value={'reconciliationId': report['id'], 'hash': report['hash'], 'reason': 'Reviewed fixture capabilities and verified retained payload', 'acknowledgeCapabilities': True})
            owner.expect('GET', '/api/v1/status', 200)
            finished = time.monotonic()
            owner.expect('GET', '/api/v1/transfers/' + transfer, 200)
            # A long planned outage may expire the reservation. Reacquire the
            # remaining file explicitly; it must not have been consumed by drain.
            def completed():
                wait_for_database(lambda: sql("SELECT count(*) FROM transfers WHERE state='completed'"), '1', 'transfer completion')
            rest_status, rest, _ = recipient.request('GET', download, headers={'Range': 'bytes=32-'},
                after_read=lambda status: completed() if status == 206 else None)
            if rest_status == 404:
                refs, _ = recipient.expect('GET', '/api/v1/recipient/files', 200)
                lease, _ = recipient.expect('POST', '/api/v1/recipient/downloads', 201, {'fileRef': json.loads(refs)[0]['ref']})
                download = json.loads(lease)['url']
                rest, _ = recipient.expect('GET', download, 200, after_read=completed)
                actual = rest
            elif rest_status == 206:
                actual = first + rest
            else:
                raise RuntimeError('post-upgrade download failed: ' + str(rest_status))
            if hashlib.sha256(actual).digest() != hashlib.sha256(payload).digest():
                raise RuntimeError('payload changed during upgrade')
            # Content-Length can be read before finishDownload commits. Keeping
            # the connection open through the PG barrier avoids cancelling it;
            # after committed completion the unchanged contract requires 401.
            recipient.expect('GET', download, 401)
            if sql("SELECT count(*) FROM downloads WHERE state='completed'") != '1' or sql("SELECT count(*) FROM download_journal WHERE status='completed'") != '1':
                raise RuntimeError('one-time completion journal mismatch')
            cli('candidate', 'audit', 'verify')
            stop.set()
            sampler.join(timeout=6)
            if sampler.is_alive():
                raise RuntimeError('downtime sampler did not stop')
            unavailable = [(at, code) for at, code in samples if code != 200]
            if not unavailable:
                raise RuntimeError('no observed downtime samples')
            result = {'baselineCommit': BASELINE, 'candidateCommit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(), 'candidateDirty': bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True).strip()), 'baselineImageId': image_ids[0], 'candidateImageId': image_ids[1], 'migrationCount': after['applied'], 'schemaChanged': True, 'addedMigrations': EXPECTED_MIGRATIONS, 'downgradeRejected': True, 'maintenanceWindowSeconds': round(finished - started, 3), 'observedUnavailableSeconds': round(unavailable[-1][0] - unavailable[0][0], 3), 'sampleIntervalSeconds': 0.2, 'unavailableSamples': len(unavailable), 'backupVerified': True, 'oldSchemaBackupRejected': True, 'postMigrationBackupVerified': True, 'payloadVerified': True, 'startupFaultsRejected': ['unknown_migration', 'wrong_kek'], 'gateExplicitlyReopened': True}
            evidence = ROOT / '.cache/evidence/phase7-upgrade.json'
            evidence.parent.mkdir(parents=True, exist_ok=True)
            evidence.write_text(json.dumps(result, indent=2) + '\n')
            print('PASS: distinct baseline/candidate images, verified backup, drain, startup fault rejection, closed restart, explicit reopen, retained sessions/payload and one-time replay rejection.')
            print(json.dumps(result))
        finally:
            stop.set()
            if sampler:
                sampler.join(timeout=6)
            dc('candidate', 'down', '--volumes', '--remove-orphans')


if __name__ == '__main__':
    test_controlled_image_upgrade()
