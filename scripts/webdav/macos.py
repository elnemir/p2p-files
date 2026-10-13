#!/usr/bin/env python3
"""Exercise the real macOS filesystem client with a disposable credential on stdin."""
import hashlib
import json
import os
from pathlib import Path
import pty
import select
import shutil
import subprocess
import sys
import tempfile
import time

config = json.load(sys.stdin)
mount = Path(tempfile.mkdtemp(prefix='filex-dav-mount-'))
child = None
try:
    pid, fd = pty.fork()
    if pid == 0:
        os.execv('/sbin/mount_webdav', ['mount_webdav', '-S', '-i', config['url'], str(mount)])
    child = pid
    transcript = ''
    sent_user = sent_password = False
    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        if select.select([fd], [], [], 0.2)[0]:
            try:
                text = os.read(fd, 4096).decode(errors='replace')
            except OSError:
                text = ''
            transcript += text
            if 'username' in transcript.lower() and not sent_user:
                os.write(fd, b'filex\n')
                sent_user = True
            if 'password' in transcript.lower() and not sent_password:
                os.write(fd, (config['password'] + '\n').encode())
                sent_password = True
        ended, status = os.waitpid(pid, os.WNOHANG)
        if ended:
            child = None
            if os.waitstatus_to_exitcode(status):
                raise RuntimeError('mount_webdav failed: ' + transcript.replace(config['password'], '[redacted]'))
            break
    else:
        raise RuntimeError('mount_webdav timed out: ' + transcript.replace(config['password'], '[redacted]'))
    fixture = mount / 'from-server.txt'
    if fixture.read_bytes() != b'server fixture for native read\n':
        raise RuntimeError('Native GET failed for server-created file')
    fixture.unlink()
    folder = mount / 'Нативный клиент'
    folder.mkdir()
    payload = ('native macOS client — FileX\n' * 190000).encode()
    source = folder / 'договор.txt'
    with source.open('wb') as stream:
        stream.write(payload)
        stream.flush()
        os.fsync(stream.fileno())
    if source.read_bytes() != payload:
        raise RuntimeError('Mounted read did not match uploaded bytes')
    replacement = b'atomic replacement from macOS\n'
    with source.open('wb') as stream:
        stream.write(replacement)
        stream.flush()
        os.fsync(stream.fileno())
    target = folder / 'готово.txt'
    source.rename(target)
    if target.read_bytes() != replacement:
        raise RuntimeError('Mounted replacement or rename changed bytes')
    target.unlink()
    folder.rmdir()
    print(json.dumps({'client': 'mount_webdav', 'platform': subprocess.check_output(['sw_vers', '-productVersion'], text=True).strip(), 'operations': ['mount', 'mkdir', 'upload', 'read', 'replace', 'rename', 'delete', 'rmdir'], 'uploadedBytes': len(payload), 'sha256': hashlib.sha256(payload).hexdigest(), 'status': 'passed'}))
finally:
    if child:
        os.kill(child, 15)
        os.waitpid(child, 0)
    result = subprocess.run(['/sbin/umount', str(mount)], capture_output=True)
    if os.path.ismount(mount):
        raise RuntimeError('Test mount remains mounted: ' + str(mount))
    shutil.rmtree(mount)
