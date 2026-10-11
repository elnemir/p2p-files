"""Exercise fail-closed repository targeting and remote-state comparisons without GitHub writes."""
import copy
import json
import unittest
from unittest.mock import patch
import github_repository as subject


class RepositoryPolicyTests(unittest.TestCase):
    def setUp(self):
        self.policy = json.loads(subject.POLICY.read_text())

    def test_wrong_origin_rejected_before_github_request(self):
        for remote in ['git@github.com:another/project.git',
                       'https://github.com.evil.test/elnemir/p2p-files.git',
                       'https://github.com/elnemir/p2p-files/fork']:
            with self.subTest(remote=remote), patch.object(subject.subprocess, 'check_output', return_value=remote), patch.object(subject, 'api') as api:
                with self.assertRaises(ValueError):
                    subject.preflight(self.policy)
                api.assert_not_called()

    def test_no_admin_or_changed_default_branch_rejected(self):
        remote = {'full_name': 'elnemir/p2p-files', 'default_branch': 'main', 'permissions': {'admin': True}}
        for change in [{'default_branch': 'other'}, {'permissions': {'admin': False}}, {'archived': True}]:
            with self.subTest(change=change), patch.object(subject.subprocess, 'check_output', return_value='git@github.com:elnemir/p2p-files.git'), patch.object(subject, 'api', return_value=remote | change):
                with self.assertRaises(ValueError):
                    subject.preflight(self.policy)

    def test_policy_cannot_silently_widen_scope_or_weaken_main(self):
        candidates = []
        for key, value in [('private', True), ('default_branch', 'other')]:
            policy = copy.deepcopy(self.policy)
            policy['settings'][key] = value
            candidates.append(policy)
        for key, value in [('allow_force_pushes', True), ('enforce_admins', False)]:
            policy = copy.deepcopy(self.policy)
            policy['protection'][key] = value
            candidates.append(policy)
        policy = copy.deepcopy(self.policy)
        policy['protection']['required_status_checks']['checks'].pop()
        candidates.append(policy)
        for policy in candidates:
            with self.subTest(policy=policy), self.assertRaises(ValueError):
                subject.validate_policy(policy)

    def test_api_wrapper_and_unordered_checks_are_normalized(self):
        expected = self.policy['protection']
        actual = copy.deepcopy(expected)
        actual['enforce_admins'] = {'enabled': True, 'url': 'ignored'}
        actual['required_status_checks']['checks'].reverse()
        self.assertEqual([], subject.subset_differences(expected, actual, 'protection'))
        actual['required_status_checks']['checks'][0]['app_id'] = -1
        self.assertTrue(subject.subset_differences(expected, actual, 'protection'))

    def test_malformed_or_missing_remote_state_fails_audit(self):
        self.assertEqual(['security'], subject.subset_differences(self.policy['security'], None, 'security'))
        self.assertTrue(subject.subset_differences({'enabled': True}, {'enabled': False}, 'reports'))


if __name__ == '__main__':
    unittest.main()
