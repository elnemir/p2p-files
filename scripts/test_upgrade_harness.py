"""Protect the upgrade harness from confusing received bytes with a server commit."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location('upgrade_harness', Path(__file__).with_name('test-upgrade.py'))
upgrade = importlib.util.module_from_spec(spec)
spec.loader.exec_module(upgrade)


class UpgradeHarnessTests(unittest.TestCase):
    def test_completion_observer_runs_before_connection_close(self):
        connection = Mock()
        response = connection.getresponse.return_value
        response.status = 206
        response.read.return_value = b'bytes'
        response.getheaders.return_value = []
        observed = []

        def commit():
            connection.close.assert_not_called()
            observed.append(True)

        with patch.object(upgrade.http.client, 'HTTPConnection', return_value=connection):
            body, _ = upgrade.Client(1234).expect('GET', '/content', 206, after_read=commit)
        self.assertEqual(b'bytes', body)
        self.assertEqual([True], observed)
        connection.close.assert_called_once()

    def test_lost_commit_fails_and_still_closes_connection(self):
        connection = Mock()
        response = connection.getresponse.return_value
        response.status = 200
        response.read.return_value = b'bytes'
        response.getheaders.return_value = []

        def missing():
            raise RuntimeError('missing commit')

        with patch.object(upgrade.http.client, 'HTTPConnection', return_value=connection):
            with self.assertRaisesRegex(RuntimeError, 'missing commit'):
                upgrade.Client(1234).expect('GET', '/content', 200, after_read=missing)
        connection.close.assert_called_once()

    def test_database_barrier_waits_for_commit_and_fails_on_timeout(self):
        with patch.object(upgrade.time, 'sleep'):
            upgrade.wait_for_database(Mock(side_effect=['0', '1']), '1', 'completion')
        with patch.object(upgrade.time, 'sleep'), patch.object(upgrade.time, 'monotonic', side_effect=[0, 0, 1, 11]):
            with self.assertRaisesRegex(RuntimeError, 'Server commit was not observed'):
                upgrade.wait_for_database(lambda: '0', '1', 'completion')

    def test_unexpected_http_status_is_not_hidden_by_completion_wait(self):
        connection = Mock()
        response = connection.getresponse.return_value
        response.status = 409
        response.read.return_value = b''
        response.getheaders.return_value = []
        observer = Mock()
        with patch.object(upgrade.http.client, 'HTTPConnection', return_value=connection):
            with self.assertRaisesRegex(RuntimeError, 'expected 401, got 409'):
                upgrade.Client(1234).expect('GET', '/content', 401, after_read=observer)
        observer.assert_not_called()
