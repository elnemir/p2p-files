"""A partial or stale capture must never replace the delivered tour."""
import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent / 'tour'))
from validate import PLAN, validate_capture


class CaptureIntegrity(unittest.TestCase):
    def setUp(self):
        chapters = json.loads(PLAN.read_text())['chapters']
        self.source = {'passed': True, 'chapters': [dict(ch, number=i, narration='Текст.', start=0, end=2)
                                                  for i, ch in enumerate(chapters, 1)],
                       'screens': [{'id': s, 'file': f'screenshots/{s}.png'} for ch in chapters for s in ch['screens']]}

    def test_complete_plan(self):
        self.assertEqual(len(validate_capture(self.source)), 64)

    def test_partial_stale_and_misdirected_capture(self):
        cases = [lambda s: s.update(passed=False),
                 lambda s: s['chapters'].pop(),
                 lambda s: s['screens'].pop(),
                 lambda s: s['screens'].reverse(),
                 lambda s: s['chapters'][0].update(screens=[]),
                 lambda s: s['chapters'][0].update(narration=''),
                 lambda s: s['chapters'][0].update(end=float('nan')),
                 lambda s: s['chapters'][0].update(start=3),
                 lambda s: s['screens'][0].update(file='../private.png'),
                 lambda s: s.update(capturePlanSHA256='stale')]
        for change in cases:
            with self.subTest(change=change):
                source = copy.deepcopy(self.source)
                change(source)
                with self.assertRaises(ValueError):
                    validate_capture(source)


if __name__ == '__main__':
    unittest.main()
