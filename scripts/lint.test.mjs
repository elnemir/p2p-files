import assert from 'node:assert/strict';
import fs from 'node:fs';
import {test} from 'node:test';
import {ESLint} from 'eslint';
import a11y from 'eslint-plugin-jsx-a11y-x';

const baseline = JSON.parse(fs.readFileSync(new URL('./fixtures/a11y-baseline.json', import.meta.url)));
const eslint = new ESLint();
const filePath = 'apps/web/components/AccessibilityRegression.tsx';
const config = await eslint.calculateConfigForFile(filePath);
const diagnostics = async (jsx) => {
  const [result] = await eslint.lintText(`export default function Probe() { return (${jsx}); }`, {filePath});
  assert.equal(result.fatalErrorCount, 0, JSON.stringify(result.messages));
  return result.messages;
};

// These are behavioral examples, independent of either plugin's implementation.
// The captured pre-migration config also protects severity and rule options.
const cases = {
  'alt-text': ['<img src="photo.png" />', '<img src="photo.png" alt="Portrait" />'],
  'anchor-has-content': ['<a href="/help" />', '<a href="/help">Help</a>'],
  'anchor-is-valid': ['<a href="#">Help</a>', '<a href="/help">Help</a>'],
  'aria-activedescendant-has-tabindex': ['<div aria-activedescendant="item" />', '<div aria-activedescendant="item" tabIndex={0} />'],
  'aria-props': ['<div aria-missing="true" />', '<div aria-hidden="true" />'],
  'aria-proptypes': ['<div aria-hidden="maybe" />', '<div aria-hidden="true" />'],
  'aria-role': ['<div role="nonexistent" />', '<div role="button" tabIndex={0}>Open</div>'],
  'aria-unsupported-elements': ['<meta aria-hidden="true" />', '<meta name="description" content="FileX" />'],
  'autocomplete-valid': ['<input autoComplete="invalid-token" />', '<input autoComplete="email" />'],
  'click-events-have-key-events': ['<div onClick={() => undefined}>Open</div>', '<button onClick={() => undefined}>Open</button>'],
  'heading-has-content': ['<h1 />', '<h1>Transfers</h1>'],
  'html-has-lang': ['<html />', '<html lang="ru" />'],
  'iframe-has-title': ['<iframe src="/help" />', '<iframe src="/help" title="Help" />'],
  'img-redundant-alt': ['<img src="x.png" alt="Image of a cat" />', '<img src="x.png" alt="Cat" />'],
  'interactive-supports-focus': ['<div role="button" onClick={() => undefined}>Open</div>', '<div role="button" tabIndex={0} onClick={() => undefined}>Open</div>'],
  'label-has-associated-control': ['<label>Name</label>', '<label>Name<input /></label>'],
  'media-has-caption': ['<video src="tour.mp4" />', '<video src="tour.mp4"><track kind="captions" src="tour.vtt" /></video>'],
  'mouse-events-have-key-events': ['<div onMouseOver={() => undefined} />', '<div onMouseOver={() => undefined} onFocus={() => undefined} />'],
  'no-access-key': ['<button accessKey="s">Save</button>', '<button>Save</button>'],
  'no-autofocus': ['<input autoFocus />', '<input />'],
  'no-distracting-elements': ['<marquee>News</marquee>', '<div>News</div>'],
  'no-interactive-element-to-noninteractive-role': ['<button role="article">Open</button>', '<button>Open</button>'],
  'no-noninteractive-element-interactions': ['<article onClick={() => undefined}>Text</article>', '<article>Text</article>'],
  'no-noninteractive-element-to-interactive-role': ['<h1 role="button">Open</h1>', '<h1>Open</h1>'],
  'no-noninteractive-tabindex': ['<article tabIndex={0}>Text</article>', '<article>Text</article>'],
  'no-redundant-roles': ['<button role="button">Open</button>', '<button>Open</button>'],
  'no-static-element-interactions': ['<div onClick={() => undefined}>Open</div>', '<button onClick={() => undefined}>Open</button>'],
  'role-has-required-aria-props': ['<div role="checkbox" />', '<div role="checkbox" aria-checked="false" />'],
  'role-supports-aria-props': ['<div role="article" aria-checked="false" />', '<div role="checkbox" aria-checked="false" />'],
  scope: ['<div scope="col" />', '<th scope="col">Name</th>'],
  'tabindex-no-positive': ['<button tabIndex={1}>Open</button>', '<button tabIndex={0}>Open</button>'],
};

test('all 31 previously enabled accessibility rules retain severity and options', () => {
  assert.equal(Object.keys(baseline.rules).length, 31);
  assert.deepEqual(Object.keys(cases).sort(), Object.keys(baseline.rules).map((name) => name.split('/')[1]).sort());
  for (const [name, value] of Object.entries(baseline.rules)) {
    const currentName = name.replace('jsx-a11y/', 'jsx-a11y-x/');
    assert.deepEqual(a11y.configs.recommended.rules[currentName], value, `${name}: preset options changed`);
    const normalized = Array.isArray(value) ? [2, ...value.slice(1)] : [2];
    assert.equal(config.rules[currentName][0], 2, `${name}: severity weakened`);
    // ESLint 10 expands the plugin's implicit defaults. Explicit project options
    // must still match exactly; both default and configured behavior are tested below.
    if (Array.isArray(value)) assert.deepEqual(config.rules[currentName], normalized, name);
  }
});

for (const [rule, [invalid, valid]] of Object.entries(cases)) {
  test(`accessibility: ${rule} detects a violation and accepts its correction`, async () => {
    const name = `jsx-a11y-x/${rule}`;
    const bad = await diagnostics(invalid);
    assert(bad.some((m) => m.ruleId === name && m.severity === 2), `Missing ${name}: ${JSON.stringify(bad)}`);
    const good = await diagnostics(valid);
    assert(!good.some((m) => m.ruleId === name), `False positive ${name}: ${JSON.stringify(good)}`);
  });
}

test('TypeScript and React Hooks checks survive the ESLint major upgrade', async () => {
  const [result] = await eslint.lintText('import {useState} from "react"; export default function Probe({active}: {active: boolean}) { if (active) { useState(0); } const unused: number = 1; return null; }', {filePath});
  assert.equal(result.fatalErrorCount, 0);
  for (const name of ['@typescript-eslint/no-unused-vars', 'react-hooks/rules-of-hooks']) {
    assert(result.messages.some((m) => m.ruleId === name && m.severity === 2), `Missing ${name}`);
  }
});
