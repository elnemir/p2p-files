#!/usr/bin/env python3
"""Audit or apply the reviewed FileX GitHub settings through the authenticated gh CLI."""
import argparse
import datetime
import json
from pathlib import Path
import re
import subprocess
from urllib.parse import quote

ROOT = Path(__file__).resolve().parents[1]
POLICY = ROOT / '.github/repository.json'


def validate_policy(policy):
    if policy['repository'] != 'elnemir/p2p-files' or policy['branch'] != 'main':
        raise ValueError('Only the reviewed FileX repository and main branch are supported')
    allowed_settings = {'description', 'homepage', 'has_issues', 'has_wiki',
                        'allow_squash_merge', 'allow_merge_commit', 'allow_rebase_merge',
                        'allow_auto_merge', 'delete_branch_on_merge', 'allow_update_branch',
                        'squash_merge_commit_title', 'squash_merge_commit_message'}
    if set(policy['settings']) - allowed_settings:
        raise ValueError('Unreviewed repository setting; visibility/access/deploy are out of scope')
    protection = policy['protection']
    if not protection['enforce_admins'] or protection['allow_force_pushes'] or protection['allow_deletions']:
        raise ValueError('Main must remain protected, including administrators')
    required = {'checks', 'supply-chain', 'repository-policy', 'security-checks'}
    checks = protection['required_status_checks']
    if not checks['strict'] or {x['context'] for x in checks['checks']} != required:
        raise ValueError('All reviewed required checks must remain enabled and current')
    if any(x['app_id'] != 15368 for x in checks['checks']):
        raise ValueError('Required checks must originate from GitHub Actions')
    if not protection['required_linear_history'] or not protection['required_conversation_resolution']:
        raise ValueError('Linear history and resolved review conversations are required')
    if policy['workflowPermissions'] != {'default_workflow_permissions': 'read', 'can_approve_pull_request_reviews': False}:
        raise ValueError('Workflows must not receive default write/approval permissions')
    if policy['actions'] != {'enabled': True, 'allowed_actions': 'selected', 'sha_pinning_required': True}:
        raise ValueError('Actions require the reviewed allowlist and immutable pins')
    if policy['allowedActions'] != {'github_owned_allowed': True, 'verified_allowed': False, 'patterns_allowed': []}:
        raise ValueError('Third-party actions require explicit review')


def api(endpoint, method='GET', payload=None, missing=()):
    command = ['gh', 'api', '--method', method, endpoint]
    if payload is not None:
        command += ['--input', '-']
    result = subprocess.run(command, input=json.dumps(payload) if payload is not None else None,
                            text=True, capture_output=True, cwd=ROOT)
    body = json.loads(result.stdout) if result.stdout.strip() else None
    if result.returncode:
        if isinstance(body, dict) and str(body.get('status')) in {str(x) for x in missing}:
            return None
        raise RuntimeError(f'{method} {endpoint}: {result.stderr.strip()}')
    return body


def preflight(policy):
    validate_policy(policy)
    remote = subprocess.check_output(['git', 'remote', 'get-url', 'origin'], cwd=ROOT, text=True).strip()
    match = re.fullmatch(r'(?:git@github\.com:|https://github\.com/)([^/]+/[^/]+?)(?:\.git)?', remote)
    if not match or match[1] != policy['repository']:
        raise ValueError('origin does not identify the reviewed GitHub repository')
    repo = api('repos/' + policy['repository'])
    if repo['full_name'] != policy['repository'] or repo['default_branch'] != policy['branch']:
        raise ValueError('Remote repository/default branch changed; review before applying')
    if repo.get('archived') or not repo.get('permissions', {}).get('admin'):
        raise ValueError('An active repository and administrator access are required')
    return repo


def subset_differences(expected, actual, path):
    differences = []
    if isinstance(expected, dict):
        if not isinstance(actual, dict):
            return [path]
        for key, value in expected.items():
            differences += subset_differences(value, actual.get(key), path + '.' + key)
    elif isinstance(expected, bool) and isinstance(actual, dict) and 'enabled' in actual:
        if expected != actual['enabled']:
            differences.append(path)
    elif isinstance(expected, list):
        if not isinstance(actual, list) or len(expected) != len(actual):
            return [path]
        key = next((candidate for candidate in ['context', 'type', 'tool']
                    if expected and all(isinstance(x, dict) and candidate in x for x in expected)), None)
        if key:
            actual_by_context = {x.get(key): x for x in actual if isinstance(x, dict)}
            for item in expected:
                differences += subset_differences(item, actual_by_context.get(item[key]), path + '.' + item[key])
        elif sorted(expected) != sorted(actual):
            differences.append(path)
    elif expected != actual:
        differences.append(path)
    return differences


def snapshot(policy):
    base = 'repos/' + policy['repository']
    repo = api(base)
    values = {'settings': {k: repo.get(k) for k in policy['settings']},
              'security': repo.get('security_and_analysis', {}),
              'topics': repo.get('topics', []),
              'visibility': repo.get('visibility'), 'defaultBranch': repo.get('default_branch')}
    for key, suffix in [('actions', 'actions/permissions'),
                        ('allowedActions', 'actions/permissions/selected-actions'),
                        ('workflowPermissions', 'actions/permissions/workflow'),
                        ('forkApproval', 'actions/permissions/fork-pr-contributor-approval'),
                        ('protection', 'branches/main/protection'),
                        ('privateReporting', 'private-vulnerability-reporting'),
                        ('securityUpdates', 'automated-security-fixes')]:
        values[key] = api(base + '/' + suffix, missing=(404, 409))
    # This endpoint returns 204 when enabled and 404 when disabled.
    alerts = subprocess.run(['gh', 'api', base + '/vulnerability-alerts'], cwd=ROOT, capture_output=True)
    values['vulnerabilityAlerts'] = alerts.returncode == 0
    values['labels'] = [{k: row.get(k) for k in ['name', 'color', 'description']}
                        for row in api(base + '/labels?per_page=100')]
    rulesets = [x for x in api(base + '/rulesets?per_page=100') if x['name'] == policy['codeScanningRuleset']['name']]
    if len(rulesets) > 1:
        raise ValueError('Duplicate managed rulesets; resolve explicitly before applying')
    values['codeScanningRuleset'] = api(base + '/rulesets/' + str(rulesets[0]['id'])) if rulesets else None
    return values


def differences(policy, state):
    result = []
    for key in ['settings', 'topics', 'actions', 'allowedActions', 'workflowPermissions', 'forkApproval', 'security', 'protection', 'codeScanningRuleset']:
        result += subset_differences(policy[key], state.get(key), key)
    result += subset_differences(True, state.get('vulnerabilityAlerts'), 'vulnerabilityAlerts')
    result += subset_differences({'enabled': True}, state.get('privateReporting'), 'privateReporting')
    result += subset_differences({'enabled': True, 'paused': False}, state.get('securityUpdates'), 'securityUpdates')
    labels = {x['name']: x for x in state['labels']}
    for label in policy['labels']:
        result += subset_differences(label, labels.get(label['name']), 'labels.' + label['name'])
    return result


def apply(policy, before):
    base = 'repos/' + policy['repository']
    api(base, 'PATCH', policy['settings'])
    api(base + '/topics', 'PUT', {'names': policy['topics']})
    api(base + '/actions/permissions', 'PUT', policy['actions'])
    api(base + '/actions/permissions/selected-actions', 'PUT', policy['allowedActions'])
    api(base + '/actions/permissions/workflow', 'PUT', policy['workflowPermissions'])
    api(base + '/actions/permissions/fork-pr-contributor-approval', 'PUT', policy['forkApproval'])
    api(base, 'PATCH', {'security_and_analysis': policy['security']})
    for suffix in ['vulnerability-alerts', 'automated-security-fixes', 'private-vulnerability-reporting']:
        api(base + '/' + suffix, 'PUT')
    existing_labels = {x['name'] for x in before['labels']}
    for label in policy['labels']:
        if label['name'] in existing_labels:
            api(base + '/labels/' + quote(label['name'], safe=''), 'PATCH',
                {'new_name': label['name'], 'color': label['color'], 'description': label['description']})
        else:
            api(base + '/labels', 'POST', label)
    # Enable the complete merge gate only after the remaining settings succeed.
    api(base + '/branches/main/protection', 'PUT', policy['protection'])
    current = before['codeScanningRuleset']
    suffix = '/rulesets/' + str(current['id']) if current else '/rulesets'
    api(base + suffix, 'PUT' if current else 'POST', policy['codeScanningRuleset'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['audit', 'apply'], nargs='?', default='audit')
    args = parser.parse_args()
    policy = json.loads(POLICY.read_text())
    preflight(policy)
    before = snapshot(policy)
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S.%fZ')
    directory = ROOT / '.cache/repository-setup' / stamp
    directory.mkdir(parents=True)
    (directory / 'before.json').write_text(json.dumps(before, ensure_ascii=False, indent=2) + '\n')
    if args.action == 'apply':
        apply(policy, before)
    after = snapshot(policy) if args.action == 'apply' else before
    drift = differences(policy, after)
    report = {'repository': policy['repository'], 'action': args.action,
              'checkedAt': stamp, 'matchesPolicy': not drift, 'differences': drift, 'state': after}
    (directory / 'result.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'matchesPolicy': not drift, 'differences': drift,
                      'evidence': str(directory.relative_to(ROOT) / 'result.json')}, ensure_ascii=False))
    if drift:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
