#!/usr/bin/env python3
"""Fetch official, hash-pinned scanner archives without running an install script."""
import hashlib
import io
import json
import os
import platform
from pathlib import Path
import subprocess
import sys
import tarfile
import urllib.request

ROOT = Path(__file__).resolve().parents[1]


def tool(name):
    spec = json.loads((ROOT / 'scripts/security-tools.json').read_text())[name]
    asset, digest = spec['platforms'][platform.system() + '-' + platform.machine()]
    directory = ROOT / '.cache/security-tools' / (name + '-' + spec['version'])
    directory.mkdir(parents=True, exist_ok=True)
    archive = directory / asset
    if not archive.exists():
        url = f"https://github.com/{spec['repository']}/releases/download/v{spec['version']}/{asset}"
        with urllib.request.urlopen(url, timeout=120) as response:
            data = response.read(200 * 1024 * 1024 + 1)
        if hashlib.sha256(data).hexdigest() != digest:
            raise RuntimeError(name + ' archive checksum mismatch')
        archive.write_bytes(data)
    data = archive.read_bytes()
    if hashlib.sha256(data).hexdigest() != digest:
        raise RuntimeError(name + ' cached archive checksum mismatch')
    with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as tar:
        member = tar.getmember(name)
        if not member.isfile() or member.size > 500 * 1024 * 1024:
            raise RuntimeError('invalid scanner archive member')
        binary = tar.extractfile(member).read()
    path = directory / name
    if not path.exists() or hashlib.sha256(path.read_bytes()).digest() != hashlib.sha256(binary).digest():
        temporary = directory / (name + '.tmp')
        temporary.write_bytes(binary)
        temporary.chmod(0o700)
        os.replace(temporary, path)
    return str(path)


if __name__ == '__main__':
    sys.exit(subprocess.run([tool(sys.argv[1]), *sys.argv[2:]], cwd=ROOT).returncode)
