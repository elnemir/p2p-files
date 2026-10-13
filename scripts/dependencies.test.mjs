import assert from 'node:assert/strict';
import {test} from 'node:test';
import {fileURLToPath} from 'node:url';
import {readDependencyInputs, validateDependencyInputs} from './check_dependencies.mjs';

const baseline = readDependencyInputs(fileURLToPath(new URL('../', import.meta.url)));
const reviewedAt = new Date(`${baseline.policy.reviewedAt}T00:00:00Z`);
const differentMajor = baseline.policy.node.major + 1;

test('the reviewed dependency set is consistent', () => {
  validateDependencyInputs(baseline, reviewedAt);
});

for (const [name, mutate, error] of [
  ['PR #9: ESLint config major without matching engine', (i) => {i.manifest.devDependencies.eslint = '9.39.5';}, /upgraded together/],
  ['Node types advertise a different major than the runtime', (i) => {i.web.devDependencies['@types/node'] = `${differentMajor}.0.0`;}, /types advertise/],
  ['Docker Node upgrade without runtime review', (i) => {i.dockerWeb = i.dockerWeb.replaceAll(`node:${i.policy.node.major}-`, `node:${differentMajor}-`);}, /reviewed runtime/],
  ['different Node build/runtime versions', (i) => {i.dockerWeb = i.dockerWeb.replace(`node:${i.policy.node.major}-`, `node:${differentMajor}-`);}, /images differ/],
  ['unpinned Node images', (i) => {i.dockerWeb = i.dockerWeb.replaceAll(/@sha256:[a-f0-9]{64}/g, '');}, /unpinned/],
  ['PR #3: Redis 8 without support/license review', (i) => {i.dockerRedis = i.dockerRedis.replace('redis:7.2-', 'redis:8.10-');}, /support\/license policy/],
  ['unpinned Redis image', (i) => {i.dockerRedis = i.dockerRedis.replace(/@sha256:[a-f0-9]{64}/, '');}, /support\/license policy/],
  ['CI switches Node independently', (i) => {i.nvmrc = String(differentMajor);}, /CI Node major/],
  ['development engine switches Node independently', (i) => {i.manifest.engines.node = `${differentMajor}.x`;}, /Development Node engine/],
  ['peer checks bypassed', (i) => {i.npmrc += '\nlegacy-peer-deps=true\n';}, /bypass/],
  ['peer checks disabled', (i) => {i.npmrc = i.npmrc.replace('strict-peer-deps=true', 'strict-peer-deps=false');}, /must remain enabled/],
  ['engine checks disabled', (i) => {i.npmrc = i.npmrc.replace('engine-strict=true', 'engine-strict=false');}, /must remain enabled/],
  ['unknown Node channel', (i) => {i.policy.node.channel = 'Experimental';}, /Unknown Node release channel/],
  ['premature LTS label', (i) => {i.policy.node.channel = 'LTS';}, /channel does not match/],
  ['invalid review calendar date', (i) => {i.policy.reviewedAt = '2026-02-30';}, /invalid calendar date/],
  ['invalid LTS calendar date', (i) => {i.policy.node.ltsFrom = '2026-13-01';}, /invalid calendar date/],
  ['LTS starts at support expiry', (i) => {i.policy.node.ltsFrom = i.policy.node.securityUntil;}, /must precede/],
  ['future policy review', (i) => {i.policy.reviewedAt = '2026-10-14';}, /cannot be in the future/],
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
    changed.policy[name].securityUntil = baseline.policy.reviewedAt;
    assert.throws(() => validateDependencyInputs(changed, reviewedAt), /security support expired/);
  });
}

test('a Current review stays valid when the same line becomes LTS', () => {
  validateDependencyInputs(baseline, new Date(`${baseline.policy.node.ltsFrom}T00:00:00Z`));
});

test('a new review at the LTS boundary must use the LTS channel', () => {
  const changed = structuredClone(baseline);
  changed.policy.reviewedAt = changed.policy.node.ltsFrom;
  const boundary = new Date(`${changed.policy.reviewedAt}T00:00:00Z`);
  assert.throws(() => validateDependencyInputs(changed, boundary), /channel does not match/);
  changed.policy.node.channel = 'LTS';
  validateDependencyInputs(changed, boundary);
});

test('the original Current review cannot outlive Node security support', () => {
  assert.throws(() => validateDependencyInputs(baseline, new Date(`${baseline.policy.node.securityUntil}T00:00:00Z`)), /Node: security support expired/);
});
