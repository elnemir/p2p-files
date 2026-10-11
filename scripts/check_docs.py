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
links = {}
if (root/'docs').exists():
    errors.append('Project documentation belongs in wiki; remove the legacy docs directory')
for directory, dirs, files in os.walk(root):
    dirs[:] = [d for d in dirs if d not in {'.git', '.cache', 'node_modules', '.next', 'bin'}]
    for name in files:
        if not name.endswith('.md'): continue
        path = Path(directory)/name
        count += 1
        links[path] = set()
        for target in re.findall(r'\]\(([^)]+)\)', path.read_text()):
            if re.match(r'[a-zA-Z]+:',target) or target.startswith('#'): continue
            target = unquote(target.split('#')[0].strip('<>'))
            destination = (path.parent/target).resolve()
            if not destination.exists(): errors.append(f'{path.relative_to(root)}: missing {target}')
            if destination.suffix == '.md': links[path].add(destination)
reachable = set()
pending = [root/'wiki/index.md']
while pending:
    path = pending.pop()
    if path in reachable: continue
    reachable.add(path)
    pending.extend(links.get(path, set()) - reachable)
for path in sorted((root/'wiki').rglob('*.md')):
    if path not in reachable: errors.append(f'{path.relative_to(root)}: unreachable from wiki/index.md')
ids = re.findall(r'\*\*((?:FR|SEC|OPS|NFR)-\d{3})\*\*', (root/'wiki/requirements.md').read_text())
rows = json.loads((root/'contracts/traceability.json').read_text())
if len(ids) != len(set(ids)) or sorted(ids) != sorted(r['id'] for r in rows): errors.append('Requirement IDs and traceability do not match exactly')
test_ids = [test['id'] for row in rows for test in row['tests']]
if len(test_ids) != len(set(test_ids)): errors.append('Traceability test IDs must be unique')
for row in rows:
    if row['status'] not in {'planned','partial','verified'}: errors.append('Unknown status '+row['id'])
    for name in row['implementation']:
        if not (root/name).is_file(): errors.append('Missing implementation '+name)
    for test in row['tests']:
        p = root/test['path']
        if p.suffix == '.go': pattern = r'func\s+'+re.escape(test['name'])+r'\('
        elif p.suffix == '.py': pattern = r'def\s+'+re.escape(test['name'])+r'\('
        else: pattern = r'''\btest\(\s*(['"])'''+re.escape(test['name'])+r'''\1\s*,'''
        if not p.is_file() or not re.search(pattern,p.read_text()): errors.append('Missing test '+test['id'])
subprocess.run(['python3','scripts/traceability.py','--check'],cwd=root,check=True)
if errors: raise SystemExit('\n'.join(errors))
print(f'{count} Markdown files checked; all {len(ids)} requirements traced; no broken local links.')
