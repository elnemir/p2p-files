#!/usr/bin/env python3
"""Write a short-lived bootstrap capability to a new private directory."""
import argparse
import base64
from datetime import datetime, timedelta, timezone
import hashlib
from pathlib import Path
import os
import secrets


def create(directory, mode):
    directory.mkdir(mode=0o700, parents=False, exist_ok=False)
    token = secrets.token_urlsafe(32)
    prefix = 'FILEX_' if mode == 'product' else 'ISSUER_'
    expires = (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat(timespec='seconds').replace('+00:00', 'Z')
    env = prefix + 'SETUP_TOKEN_HASH=' + hashlib.sha256(token.encode()).hexdigest() + '\n' + prefix + 'SETUP_EXPIRES_AT=' + expires + '\n'
    if mode == 'issuer':
        env += 'ISSUER_CONFIG_KEY=' + base64.b64encode(secrets.token_bytes(32)).decode() + '\n'
    for name, value in [('bootstrap.env', env), ('token.txt', token + '\n')]:
        fd = os.open(directory / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as f:
            f.write(value)
    return expires


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['product', 'issuer'])
    parser.add_argument('directory', type=Path)
    args = parser.parse_args()
    expires = create(args.directory, args.mode)
    print(f'Created private bootstrap.env and token.txt; expires {expires}. No existing configuration changed.')
