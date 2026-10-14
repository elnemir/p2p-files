"""Reject partial or mismatched tours before publishing generated media."""
import hashlib
import json
import math
from pathlib import Path

PLAN = Path(__file__).with_name('plan.json')


def validate_capture(source):
    plan = json.loads(PLAN.read_text())
    chapters = source.get('chapters', [])
    expected = plan['chapters']
    if source.get('passed') is not True or len(chapters) != len(expected):
        raise ValueError('Capture is incomplete; fix the scenario before rendering')
    ids = []
    for number, (chapter, item) in enumerate(zip(chapters, expected), 1):
        if (chapter.get('number') != number or chapter.get('title') != item['title']
                or chapter.get('screens') != item['screens']
                or not isinstance(chapter.get('narration'), str)
                or not chapter['narration'].strip()):
            raise ValueError(f'Capture chapter {number} does not match the reviewed plan')
        start, end = chapter.get('start'), chapter.get('end')
        if (any(type(v) not in (int, float) or not math.isfinite(v) for v in (start, end))
                or not 0 <= start < end):
            raise ValueError(f'Capture chapter {number} has invalid recording bounds')
        ids.extend(item['screens'])
    screens = source.get('screens', [])
    if len(ids) != len(set(ids)) or [s.get('id') for s in screens] != ids:
        raise ValueError('Capture screenshot set does not match the reviewed plan')
    for screen in screens:
        if screen.get('file') != f'screenshots/{screen["id"]}.png':
            raise ValueError('Invalid capture screenshot path')
    fingerprint = hashlib.sha256(PLAN.read_bytes()).hexdigest()
    if source.get('capturePlanSHA256', fingerprint) != fingerprint:
        raise ValueError('Capture belongs to a different reviewed plan')
    return fingerprint
