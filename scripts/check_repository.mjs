import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import * as yaml from 'js-yaml';
import {checkDependencies} from './check_dependencies.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const read = (name) => fs.readFileSync(path.join(root, name), 'utf8');
const policy = JSON.parse(read('.github/repository.json'));
const required = new Set(policy.protection.required_status_checks.checks.map((x) => x.context));
const jobs = new Set();
const directory = path.join(root, '.github/workflows');
for (const name of fs.readdirSync(directory).filter((x) => /\.ya?ml$/.test(x))) {
  const workflow = yaml.load(fs.readFileSync(path.join(directory, name), 'utf8'));
  assert(workflow.on && workflow.jobs, `${name}: missing events/jobs`);
  assert(!('pull_request_target' in workflow.on), `${name}: privileged PR event needs separate review`);
  assert.equal(workflow.permissions?.contents, 'read', `${name}: read-only default token required`);
  assert(workflow.concurrency?.['cancel-in-progress'], `${name}: cancel superseded runs`);
  for (const [id, job] of Object.entries(workflow.jobs)) {
    jobs.add(job.name || id);
    assert(Number.isInteger(job['timeout-minutes']) && job['timeout-minutes'] > 0, `${name}/${id}: bounded job required`);
    assert(!job['continue-on-error'], `${name}/${id}: do not hide failed checks`);
    assert(job.permissions !== 'write-all', `${name}/${id}: explicit permissions only`);
    for (const step of job.steps ?? []) {
      if (!step.uses) continue;
      assert.match(step.uses, /^(?:actions|github)\/[^@]+@[0-9a-f]{40}$/, `${name}/${id}: official SHA-pinned action required`);
      if (step.uses.startsWith('actions/checkout@')) {
        assert.equal(step.with?.['persist-credentials'], false, `${name}/${id}: no persistent checkout credential`);
      }
    }
  }
}
for (const name of required) assert(jobs.has(name), `Required check has no workflow job: ${name}`);
const updates = yaml.load(read('.github/dependabot.yml'));
assert.equal(updates.version, 2);
assert.deepEqual(new Set(updates.updates.map((x) => x['package-ecosystem'])), new Set(['npm', 'gomod', 'docker', 'github-actions']));
const labels = new Set(['bug', 'enhancement', ...policy.labels.map((x) => x.name)]);
for (const update of updates.updates) {
  assert(fs.statSync(path.join(root, update.directory)).isDirectory(), `Missing Dependabot directory ${update.directory}`);
  for (const label of update.labels ?? []) assert(labels.has(label), `Missing label ${label}`);
}
for (const name of ['bug_report.yml', 'feature_request.yml']) {
  const form = yaml.load(read(`.github/ISSUE_TEMPLATE/${name}`));
  assert(form.name && form.description && form.body?.length, `Invalid issue form ${name}`);
  const ids = form.body.filter((x) => x.type !== 'markdown').map((x) => x.id);
  assert.equal(new Set(ids).size, ids.length, `Duplicate issue field ${name}`);
  assert(ids.every(Boolean), `Missing issue field ID ${name}`);
  for (const label of form.labels ?? []) assert(labels.has(label), `Missing form label ${label}`);
}
assert.equal(yaml.load(read('.github/ISSUE_TEMPLATE/config.yml')).blank_issues_enabled, false);
assert.match(read('.github/CODEOWNERS'), /^\* @elnemir$/m);
checkDependencies(root);
for (const name of ['SECURITY.md', 'CONTRIBUTING.md', '.editorconfig', '.gitattributes', '.github/pull_request_template.md']) assert(read(name).trim(), `Empty ${name}`);
console.log(`PASS: repository policy, ${jobs.size} workflow jobs, four dependency ecosystems and issue forms`);
