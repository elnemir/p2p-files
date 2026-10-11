#!/usr/bin/env python3
"""Render the requirement matrix from its versioned source of truth."""
import json
from pathlib import Path
import sys

root = Path(__file__).resolve().parents[1]
rows = json.loads((root/'contracts/traceability.json').read_text())
text = '''# Матрица requirement → implementation → test

Источник: [traceability.json](../contracts/traceability.json). Эта таблица генерируется `python3 scripts/traceability.py`; не редактировать вручную. `partial` означает компонентную реализацию, а не закрытое требование. `planned` означает отсутствие реализации. Факт запуска проверок отдельно записан в [development](../wiki/development.md).

| ID | Фаза | Статус | Реализация | Tests | Незакрытая часть |
|---|---|---|---|---|---|
'''
for row in rows:
    impl = ', '.join(f'[{p}](../{p})' for p in row['implementation']) or '—'
    tests = ', '.join(f"[{t['id']}](../{t['path']}) `{t['name']}`" for t in row['tests']) or 'Планируются'
    text += f"| {row['id']} | {row['phase']} | {row['status']} | {impl} | {tests} | {row['gap']} |\n"
out = root/'docs/test-matrix.md'
if '--check' in sys.argv:
    if not out.exists() or out.read_text() != text:
        raise SystemExit('Traceability matrix is out of date; run python3 scripts/traceability.py')
else:
    out.write_text(text)
