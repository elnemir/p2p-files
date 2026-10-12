#!/usr/bin/env python3
"""Exercise the actual GNOME GVfs DAV backend inside a disposable D-Bus session."""
import hashlib
import json
import os
from pathlib import Path
import socket
import socketserver
import subprocess
import sys
import tempfile
import threading
import urllib.parse

config = json.load(sys.stdin)
port = urllib.parse.urlsplit(config['url']).port
# Docker Desktop reaches the host through its bridge; GVfs and the HTTP Host
# header still use the exact loopback origin configured by the test server.
class Bridge(socketserver.BaseRequestHandler):
    def handle(self):
        peer = socket.create_connection(('host.docker.internal', port), 10)
        def copy(source, target):
            try:
                while data := source.recv(65536):
                    target.sendall(data)
            except OSError:
                pass
            finally:
                try:
                    target.shutdown(socket.SHUT_WR)
                except OSError:
                    pass
        worker = threading.Thread(target=copy, args=(self.request, peer))
        worker.start()
        copy(peer, self.request)
        worker.join()
        peer.close()
class Server(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True
server = Server(('127.0.0.1', port), Bridge)
threading.Thread(target=server.serve_forever, daemon=True).start()
url = config['url'].replace('http://', 'dav://filex@', 1)
def run(*args, stdin=None):
    result = subprocess.run(['gio', *args], input=stdin, capture_output=True, timeout=60)
    if result.returncode:
        output = result.stderr.decode(errors='replace').replace(config['password'], '[redacted]')
        raise RuntimeError('gio ' + args[0] + ': ' + output)
    return result.stdout
mounted = False
try:
    run('mount', url, stdin=(config['password'] + '\n').encode())
    mounted = True
    if run('cat', url + 'from-server.txt') != b'server fixture for native read\n':
        raise RuntimeError('Native GET failed for server-created file')
    run('remove', url + 'from-server.txt')
    folder = url + urllib.parse.quote('Нативный клиент')
    run('mkdir', folder)
    target = folder + '/' + urllib.parse.quote('договор.txt')
    with tempfile.TemporaryDirectory() as temporary:
        source = Path(temporary) / 'sample.txt'
        payload = ('native Linux GVfs — FileX\n' * 200000).encode()
        source.write_bytes(payload)
        run('copy', str(source), target)
        if run('cat', target) != payload:
            raise RuntimeError('GVfs read hash mismatch')
        source.write_bytes(b'Linux replacement\n')
        run('copy', str(source), target)
        renamed = folder + '/' + urllib.parse.quote('готово.txt')
        run('move', target, renamed)
        if run('cat', renamed) != source.read_bytes():
            raise RuntimeError('GVfs replacement or rename mismatch')
        run('remove', renamed)
    run('remove', folder)
    print(json.dumps({'client': 'GNOME GVfs (gio)', 'glibVersion': run('version').decode().strip(), 'gvfsVersion': subprocess.check_output(['dpkg-query', '-W', '-f=${Version}', 'gvfs-backends'], text=True), 'operations': ['mount', 'mkdir', 'upload', 'read', 'replace', 'rename', 'delete', 'rmdir'], 'uploadedBytes': len(payload), 'sha256': hashlib.sha256(payload).hexdigest(), 'status': 'passed'}))
finally:
    if mounted:
        run('mount', '--unmount', url)
    server.shutdown()
