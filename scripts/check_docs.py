#!/usr/bin/env python3
"""Validate local documentation links and requirement/test traceability."""
import json
import os
from pathlib import Path
import re
import subprocess
from urllib.parse import unquote

root = Path(__file__).resolve().parents[1]
errors = []
count = 0
for directory, dirs, files in os.walk(root):
    dirs[:] = [d for d in dirs if d not in {'.git', '.cache', 'node_modules', '.next', 'bin'}]
    for name in files:
        if not name.endswith('.md'): continue
        path = Path(directory)/name
        count += 1
        for target in re.findall(r'\]\(([^)]+)\)', path.read_text()):
            if re.match(r'[a-zA-Z]+:',target) or target.startswith('#'): continue
            target = unquote(target.split('#')[0].strip('<>'))
            if not (path.parent/target).exists(): errors.append(f'{path.relative_to(root)}: missing {target}')
ids = re.findall(r'\*\*((?:FR|SEC|OPS|NFR)-\d{3})\*\*', (root/'docs/requirements.md').read_text())
rows = json.loads((root/'contracts/traceability.json').read_text())
if len(ids) != len(set(ids)) or sorted(ids) != sorted(r['id'] for r in rows): errors.append('Requirement IDs and traceability do not match exactly')
for row in rows:
    if row['status'] not in {'planned','partial','verified'}: errors.append('Unknown status '+row['id'])
    for name in row['implementation']:
        if not (root/name).is_file(): errors.append('Missing implementation '+name)
    for test in row['tests']:
        p = root/test['path']
        pattern = (r'func\s+'+re.escape(test['name'])+r'\(') if p.suffix=='.go' else (r'''\btest\(\s*(['"])'''+re.escape(test['name'])+r'''\1\s*,''')
        if not p.is_file() or not re.search(pattern,p.read_text()): errors.append('Missing test '+test['id'])
subprocess.run(['python3','scripts/traceability.py','--check'],cwd=root,check=True)
if errors: raise SystemExit('\n'.join(errors))
print(f'{count} Markdown files checked; all {len(ids)} requirements traced; no broken local links.')
