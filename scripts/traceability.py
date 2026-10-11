#!/usr/bin/env python3
"""Render the requirement matrix from its versioned source of truth."""
import json
from pathlib import Path
import sys

root = Path(__file__).resolve().parents[1]
rows = json.loads((root/'contracts/traceability.json').read_text())
text = '''# Матрица requirement → implementation → test

Источник: [traceability.json](../contracts/traceability.json). Эта таблица генерируется `python3 scripts/traceability.py`; не редактировать вручную. `verified` означает реализацию с привязанными проверками в описанной области, а не автоматический допуск релиза. `partial` означает незакрытую часть требования, `planned` — отсутствие реализации. Фактические результаты и их границы записаны в [состоянии](current-state.md) и [журнале сессии](session.md); исторические измерения — в [evidence](history/development-evidence-2026-10-11.md).

| ID | Фаза | Статус | Реализация | Tests | Незакрытая часть |
|---|---|---|---|---|---|
'''
for row in rows:
    impl = ', '.join(f'[{p}](../{p})' for p in row['implementation']) or '—'
    tests = ', '.join(f"[{t['id']}](../{t['path']}) `{t['name']}`" for t in row['tests']) or 'Планируются'
    text += f"| {row['id']} | {row['phase']} | {row['status']} | {impl} | {tests} | {row['gap']} |\n"
out = root/'wiki/test-matrix.md'
if '--check' in sys.argv:
    if not out.exists() or out.read_text() != text:
        raise SystemExit('Traceability matrix is out of date; run python3 scripts/traceability.py')
else:
    out.write_text(text)
