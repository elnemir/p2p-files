import assert from 'node:assert/strict';
import {test} from 'node:test';
import {fileURLToPath} from 'node:url';
import {readDependencyInputs, validateDependencyInputs} from './check_dependencies.mjs';

const baseline = readDependencyInputs(fileURLToPath(new URL('../', import.meta.url)));
const reviewedAt = new Date('2026-10-12T00:00:00Z');

test('the reviewed dependency set is consistent', () => {
  validateDependencyInputs(baseline, reviewedAt);
});

for (const [name, mutate, error] of [
  ['PR #9: ESLint config major without matching engine', (i) => {i.manifest.devDependencies.eslint = '9.39.5';}, /upgraded together/],
  ['PR #10: Node 26 types on Node 24', (i) => {i.web.devDependencies['@types/node'] = '26.6.4';}, /types advertise/],
  ['PR #5: Node 25 Docker upgrade without runtime review', (i) => {i.dockerWeb = i.dockerWeb.replaceAll('node:24-', 'node:25-');}, /reviewed LTS/],
  ['different Node build/runtime versions', (i) => {i.dockerWeb = i.dockerWeb.replace('node:24-', 'node:26-');}, /images differ/],
  ['PR #3: Redis 8 without support/license review', (i) => {i.dockerRedis = i.dockerRedis.replace('redis:7.2-', 'redis:8.10-');}, /support\/license policy/],
  ['unpinned Redis image', (i) => {i.dockerRedis = i.dockerRedis.replace(/@sha256:[a-f0-9]{64}/, '');}, /support\/license policy/],
  ['CI switches Node independently', (i) => {i.nvmrc = '26';}, /CI Node major/],
  ['peer checks bypassed', (i) => {i.npmrc += '\nlegacy-peer-deps=true\n';}, /bypass/],
  ['peer checks disabled', (i) => {i.npmrc = i.npmrc.replace('strict-peer-deps=true', 'strict-peer-deps=false');}, /must remain enabled/],
]) {
  test(name, () => {
    const changed = structuredClone(baseline);
    mutate(changed);
    assert.throws(() => validateDependencyInputs(changed, reviewedAt), error);
  });
}

for (const name of ['node', 'redis']) {
  test(`${name} support expiration fails the gate`, () => {
    const changed = structuredClone(baseline);
    changed.policy[name].securityUntil = '2026-10-12';
    assert.throws(() => validateDependencyInputs(changed, reviewedAt), /security support expired/);
  });
}
