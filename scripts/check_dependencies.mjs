import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {createRequire} from 'node:module';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const major = (version) => {
  assert.match(version, /^\d+\.\d+\.\d+$/, `An exact dependency version is required: ${version}`);
  return Number(version.split('.')[0]);
};

export function readDependencyInputs(directory) {
  const read = (name) => fs.readFileSync(path.join(directory, name), 'utf8');
  return {
    policy: JSON.parse(read('.github/dependency-policy.json')),
    manifest: JSON.parse(read('package.json')),
    web: JSON.parse(read('apps/web/package.json')),
    nvmrc: read('.nvmrc').trim(),
    npmrc: read('.npmrc'),
    dockerWeb: read('deploy/Dockerfile.web'),
    dockerRedis: read('deploy/Dockerfile.redis'),
  };
}

export function validateDependencyInputs(input, now = new Date()) {
  const {policy, manifest, web, nvmrc, npmrc, dockerWeb, dockerRedis} = input;
  assert.equal(policy.version, 1, 'Unknown dependency policy version');
  for (const [name, runtime] of [['Node', policy.node], ['Redis', policy.redis]]) {
    assert.match(runtime.securityUntil, /^\d{4}-\d{2}-\d{2}$/);
    const end = new Date(`${runtime.securityUntil}T00:00:00Z`);
    assert.equal(end.toISOString().slice(0, 10), runtime.securityUntil, `${name}: invalid support deadline`);
    assert(now < end, `${name}: security support expired; review and qualify a supported runtime`);
    assert.match(runtime.source, /^https:\/\//, `${name}: support policy needs an upstream source`);
  }
  assert.equal(policy.node.channel, 'LTS', 'Production Node must use a reviewed LTS line');
  assert.equal(nvmrc, String(policy.node.major), 'CI Node major differs from runtime policy');
  assert.equal(manifest.engines?.node, `${policy.node.major}.x`, 'Development Node engine differs from CI/runtime');
  assert.equal(major(web.devDependencies['@types/node']), policy.node.major, 'Node types advertise a different runtime major');
  const images = [...dockerWeb.matchAll(/^FROM (node:[^\s]+)(?: AS \w+)?$/gmi)].map((m) => m[1]);
  assert.equal(images.length, 2, 'Both web build and runtime images must be pinned');
  assert.equal(images[0], images[1], 'Web build/runtime Node images differ');
  for (const image of images) {
    assert.match(image, new RegExp(`^node:${policy.node.major}-alpine@sha256:[a-f0-9]{64}$`), 'Node image differs from reviewed LTS line or is unpinned');
  }
  const redis = [...dockerRedis.matchAll(/^FROM (redis:[^\s]+)$/gmi)].map((m) => m[1]);
  assert.equal(redis.length, 1, 'Redis must have one pinned runtime base');
  const escapedSeries = policy.redis.series.replaceAll('.', '\\.');
  assert.match(redis[0], new RegExp(`^redis:${escapedSeries}-alpine@sha256:[a-f0-9]{64}$`), 'Redis image differs from reviewed support/license policy');
  const deps = manifest.devDependencies;
  assert.equal(major(deps.eslint), major(deps['@eslint/js']), 'ESLint and @eslint/js must be upgraded together');
  for (const name of ['strict-peer-deps', 'engine-strict']) {
    assert.match(npmrc, new RegExp(`^${name}=true$`, 'm'), `${name} must remain enabled`);
  }
  assert(!/^\s*(?:force|legacy-peer-deps)\s*=\s*true\s*$/m.test(npmrc), 'Do not bypass dependency compatibility');
}

export function checkDependencies(directory = root) {
  const input = readDependencyInputs(directory);
  validateDependencyInputs(input);
  assert.equal(Number(process.versions.node.split('.')[0]), input.policy.node.major, 'Use the Node version from .nvmrc to run checks');
  // npm can warn about an optional peer without failing installation. The full
  // installed tree is an independent gate, including transitive/optional peers.
  const tree = spawnSync('npm', ['ls', '--all'], {cwd: directory, encoding: 'utf8'});
  assert.equal(tree.status, 0, `Invalid installed dependency tree:\n${tree.stderr || tree.error || tree.stdout}`);
  const local = createRequire(path.join(directory, 'package.json'));
  const parser = createRequire(local.resolve('@typescript-eslint/parser'));
  const generator = createRequire(local.resolve('openapi-typescript'));
  const app = createRequire(path.join(directory, 'apps/web/package.json'));
  const api = local('typescript/package.json').version;
  assert.equal(api, input.manifest.devDependencies.typescript, 'Root TypeScript API is not the declared version');
  for (const [name, require] of [['parser', parser], ['OpenAPI generator', generator]]) {
    assert.equal(require('typescript/package.json').version, api, `${name} must use the reviewed TypeScript API`);
  }
  const compiler = app('typescript/package.json').version;
  assert.equal(compiler, input.web.devDependencies.typescript, 'App compiler is not the declared version');
  for (const name of ['eslint', '@eslint/js']) {
    assert.equal(local(`${name}/package.json`).version, input.manifest.devDependencies[name], `${name} differs from its manifest`);
  }
  console.log(`PASS: peer tree; Node ${process.versions.node}; ESLint ${input.manifest.devDependencies.eslint}; TypeScript API ${api}, app compiler ${compiler}; supported runtime policies`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) checkDependencies();
