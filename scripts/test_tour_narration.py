"""Fault tests for billable request deduplication and offline narration integrity."""
import base64
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent / 'tour'))
import narrate


class NarrationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.output = Path(self.temp.name) / 'audio'
        self.text = 'Привет.'
        self.source = {'chapters': [{'number': 1, 'narration': self.text}]}
        self.alignment = {'characters': list(self.text),
                          'character_start_times_seconds': [i / 10 for i in range(len(self.text))],
                          'character_end_times_seconds': [(i + 1) / 10 for i in range(len(self.text))]}
        self.calls = []

    def api(self, key, endpoint, payload=None):
        self.calls.append((endpoint, payload is not None))
        if endpoint.startswith('voices/'):
            return {'name': 'Test voice'}
        if endpoint == 'models':
            return [{'model_id': 'test_model', 'can_do_text_to_speech': True,
                     'languages': [{'language_id': 'ru'}]}]
        return {'audio_base64': base64.b64encode(b'synthetic-test-audio').decode(),
                'alignment': self.alignment}

    def generate(self, **kwargs):
        return narrate.generate(self.source, self.output, 'test-only-key', 'TestVoice',
                                'test_model', 100, **kwargs)

    def test_resume_does_not_repeat_billable_request(self):
        with patch.object(narrate, 'request', side_effect=self.api):
            self.generate()
        self.assertEqual(sum(is_post for _, is_post in self.calls), 1)
        with patch.object(narrate, 'request', side_effect=AssertionError('No network on cached resume')):
            self.generate()
        self.assertNotIn('test-only-key', (self.output / 'manifest.json').read_text())

    def test_concurrent_generator_cannot_spend_again(self):
        def concurrent(key, endpoint, payload=None):
            if endpoint.startswith('voices/'):
                with self.assertRaisesRegex(RuntimeError, 'locked'):
                    self.generate()
            return self.api(key, endpoint, payload)
        with patch.object(narrate, 'request', side_effect=concurrent):
            self.generate()
        self.assertEqual(sum(is_post for _, is_post in self.calls), 1)
        self.assertFalse((self.output / '.generation.lock').exists())

    def test_budget_and_voice_validation_precede_network(self):
        with patch.object(narrate, 'request', side_effect=AssertionError('No network before validation')):
            with self.assertRaisesRegex(ValueError, 'budget'):
                narrate.generate(self.source, self.output, 'test-only-key', 'TestVoice', 'test_model', 1)
            with self.assertRaisesRegex(ValueError, 'voice ID'):
                narrate.generate(self.source, self.output, 'test-only-key', '../other', 'test_model', 100)

    def test_unknown_request_outcome_is_not_retried(self):
        def timeout(key, endpoint, payload=None):
            if payload:
                raise RuntimeError('Unknown outcome')
            return self.api(key, endpoint, payload)
        with patch.object(narrate, 'request', side_effect=timeout):
            with self.assertRaisesRegex(RuntimeError, 'Unknown outcome'):
                self.generate()
        with patch.object(narrate, 'request', side_effect=AssertionError('Ambiguous charge cannot repeat')):
            with self.assertRaisesRegex(ValueError, 'incomplete'):
                self.generate(retry_rejected=True)

    def test_payment_rejection_requires_explicit_retry(self):
        def payment(key, endpoint, payload=None):
            if payload:
                raise narrate.ProviderError(402, 'payment_required')
            return self.api(key, endpoint, payload)
        with patch.object(narrate, 'request', side_effect=payment):
            with self.assertRaises(narrate.ProviderError):
                self.generate()
        self.assertEqual(json.loads((self.output / 'manifest.json').read_text())['chapters']['1']['status'], 'rejected')
        with patch.object(narrate, 'request', side_effect=AssertionError('Payment failure cannot auto retry')):
            with self.assertRaisesRegex(ValueError, 'incomplete'):
                self.generate()
        with patch.object(narrate, 'request', side_effect=self.api):
            self.generate(retry_rejected=True)

    def test_changed_text_and_corrupt_audio_fail_closed(self):
        with patch.object(narrate, 'request', side_effect=self.api):
            metadata = self.generate()
        changed = {'number': 1, 'narration': 'Другой текст.'}
        with self.assertRaisesRegex(ValueError, 'stale'):
            narrate.load_audio(self.output, changed, metadata)
        (self.output / '01.mp3').write_bytes(b'corrupt')
        with patch.object(narrate, 'request', side_effect=AssertionError('Corrupt cache must not spend')):
            with self.assertRaisesRegex(ValueError, 'checksum'):
                self.generate()

    def test_invalid_alignment_preserves_response_without_retry(self):
        self.alignment['character_end_times_seconds'][2] = -1
        with patch.object(narrate, 'request', side_effect=self.api):
            with self.assertRaisesRegex(ValueError, 'alignment'):
                self.generate()
        self.assertTrue((self.output / '01.mp3').exists())
        with patch.object(narrate, 'request', side_effect=AssertionError('Do not repeat completed synthesis')):
            with self.assertRaisesRegex(ValueError, 'alignment'):
                self.generate()

    def test_credential_file_must_not_be_world_readable(self):
        path = Path(self.temp.name) / 'settings.env'
        path.write_text('ELEVENLABS_API_KEY=test-only-key\n')
        path.chmod(0o644)
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(ValueError, '0600'):
                narrate.credentials(path)
            path.chmod(0o600)
            self.assertEqual(narrate.credentials(path)['ELEVENLABS_API_KEY'], 'test-only-key')


if __name__ == '__main__':
    unittest.main()
