"""Host capability generation must not expose or overwrite secrets."""
import importlib.util
from pathlib import Path
import hashlib
import tempfile
import unittest
spec = importlib.util.spec_from_file_location('setup_access', Path(__file__).with_name('setup-access.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)

class SetupAccessTests(unittest.TestCase):
    def test_private_fresh_capability_and_no_overwrite(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'issuer'
            helper.create(path, 'issuer')
            token = (path / 'token.txt').read_text().strip()
            env = (path / 'bootstrap.env').read_text()
            self.assertNotIn(token, env)
            self.assertIn(hashlib.sha256(token.encode()).hexdigest(), env)
            self.assertIn('ISSUER_CONFIG_KEY=', env)
            self.assertEqual(path.stat().st_mode & 0o777, 0o700)
            self.assertEqual((path / 'token.txt').stat().st_mode & 0o777, 0o600)
            self.assertEqual((path / 'bootstrap.env').stat().st_mode & 0o777, 0o600)
            with self.assertRaises(FileExistsError):
                helper.create(path, 'issuer')
            self.assertEqual((path / 'bootstrap.env').read_text(), env)

if __name__ == '__main__':
    unittest.main()
