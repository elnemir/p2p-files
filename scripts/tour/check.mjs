// Validate shipped artifacts and exercise the standalone gallery in a local browser.
import { chromium, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { createHash } from 'node:crypto';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const root = process.cwd();
const out = path.join(root, 'wiki/tour');
const evidence = path.join(root, '.cache/tour/qa');
await mkdir(evidence, { recursive: true });
const data = JSON.parse(await readFile(path.join(out, 'manifest.json')));
expect(data.screens).toHaveLength(44);
expect(data.chapters).toHaveLength(19);
expect(new Set(data.screens.map(s => s.id)).size).toBe(data.screens.length);
const listed = (await readFile(path.join(out, 'SHA256SUMS'), 'utf8')).trim().split('\n');
for (const line of listed) {
  const [, expected, filename] = line.match(/^([0-9a-f]{64})  (.+)$/);
  const actual = createHash('sha256').update(await readFile(path.join(out, filename))).digest('hex');
  expect(actual, filename).toBe(expected);
}
for (const screen of data.screens) {
  const png = await readFile(path.join(out, screen.file));
  expect(png.subarray(1, 4).toString()).toBe('PNG');
  expect(png.readUInt32BE(16)).toBe(screen.viewport.width);
  expect(png.readUInt32BE(20)).toBeGreaterThanOrEqual(screen.viewport.height);
}
const browser = await chromium.launch(process.env.FILEX_TOUR_BROWSER_CHANNEL ? { channel: process.env.FILEX_TOUR_BROWSER_CHANNEL } : {});
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: 'reduce' });
const page = await context.newPage();
const errors = [];
page.on('pageerror', error => errors.push(error.message));
try {
  await page.goto(pathToFileURL(path.join(out, 'index.html')).href);
  await expect(page.locator('.card')).toHaveCount(data.screens.length);
  await expect(page.locator('#chapters button')).toHaveCount(data.chapters.length);
  await page.screenshot({ path: path.join(evidence, 'gallery-desktop.png') });
  const search = page.getByRole('searchbox');
  await search.fill('backup');
  const found = await page.locator('.card').count();
  expect(found).toBeGreaterThan(0);
  expect(found).toBeLessThan(data.screens.length);
  await search.fill('неттакогоэкрана');
  await expect(page.locator('#empty')).toBeVisible();
  await search.fill('');
  await page.locator('#filters button[data-group="Восстановление"]').click();
  await expect(page.locator('.card')).toHaveCount(4);
  await page.locator('.preview').first().click();
  await expect(page.getByRole('dialog')).toBeVisible();
  const before = await page.locator('#lightboxTitle').innerText();
  await page.keyboard.press('ArrowRight');
  expect(await page.locator('#lightboxTitle').innerText()).not.toBe(before);
  await page.screenshot({ path: path.join(evidence, 'gallery-lightbox.png') });
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await expect(page.locator('.preview').first()).toBeFocused();
  await page.locator('#filters button[data-group="Все экраны"]').click();
  await page.evaluate(() => window.scrollTo(0, 0));
  const accessibility = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
  expect(accessibility.violations).toEqual([]);
  const video = page.locator('video');
  await expect.poll(() => video.evaluate(v => v.readyState), { timeout: 30000 }).toBeGreaterThanOrEqual(1);
  const media = await video.evaluate(v => ({ width: v.videoWidth, height: v.videoHeight, duration: v.duration, error: v.error?.message }));
  expect(media.error).toBeUndefined();
  expect(media.width).toBe(1440);
  expect(media.height).toBe(1080);
  expect(Math.abs(media.duration - data.duration)).toBeLessThan(2);
  await page.locator('#chapters button').nth(14).click();
  await expect.poll(() => video.evaluate(v => v.currentTime)).toBeGreaterThanOrEqual(data.chapters[14].time);
  await expect.poll(() => video.evaluate(v => v.paused)).toBe(false);
  await expect.poll(() => video.evaluate(v => v.currentTime), { timeout: 15000 }).toBeGreaterThan(data.chapters[14].time + 1);
  await video.evaluate(v => v.pause());
  await page.locator('#video').scrollIntoViewIfNeeded();
  await page.screenshot({ path: path.join(evidence, 'video-playing.png') });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: path.join(evidence, 'gallery-mobile.png'), fullPage: false });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.locator('#screens').scrollIntoViewIfNeeded();
  await page.screenshot({ path: path.join(evidence, 'gallery-mobile-screens.png') });
  const mobileA11y = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
  expect(mobileA11y.violations).toEqual([]);
  expect(errors).toEqual([]);
  // Exercise a tampered embedded manifest through the real renderer, without navigation.
  const html = (await readFile(path.join(out, 'index.html'), 'utf8')).replace('<script src="gallery.js"></script>', '');
  const rejectedPaths = ['javascript:alert(1)', 'data:text/html,<script>alert(1)</script>',
    'https://example.test/tracker.png', '//example.test/tracker.png', 'screenshots/../secret.png',
    'screenshots/01-login.svg', 'screenshots/01-login.png?redirect=1'];
  for (const file of rejectedPaths) {
    const unsafe = await context.newPage();
    const failures = [];
    unsafe.on('pageerror', error => failures.push(error.message));
    await unsafe.setContent(html);
    await unsafe.evaluate(file => {
      const node = document.getElementById('tour-data');
      const manifest = JSON.parse(node.textContent);
      manifest.screens[0].file = file;
      node.textContent = JSON.stringify(manifest);
    }, file);
    await unsafe.addScriptTag({ path: path.join(out, 'gallery.js') });
    await expect.poll(() => failures).toContain('Invalid screenshot path in tour manifest');
    expect(await unsafe.locator('img,a').evaluateAll((nodes, file) => nodes.some(node =>
      node.getAttribute('src') === file || node.getAttribute('href') === file), file)).toBe(false);
    await unsafe.close();
  }
  // Contact sheets are private review aids; original PNG bytes are never changed.
  for (let start = 0; start < data.screens.length; start += 6) {
    const review = await context.newPage();
    await review.setViewportSize({ width: 1600, height: 1800 });
    await review.goto(pathToFileURL(path.join(out, 'index.html')).href);
    await review.evaluate(screens => {
      document.body.replaceChildren();
      const style = document.createElement('style');
      style.textContent = 'body{margin:0;padding:16px;display:grid;grid-template-columns:1fr 1fr;gap:18px;background:#dce4d7}figure{margin:0;padding:9px;background:white;min-width:0}figcaption{font:18px Arial;margin:4px 0 9px}img{display:block;width:100%;height:506px;object-fit:contain;object-position:top;background:#eff3e9}';
      document.head.append(style);
      for (const s of screens) {
        const figure = document.createElement('figure');
        const caption = document.createElement('figcaption');
        caption.textContent = `${s.id} · ${s.title}`;
        const img = document.createElement('img'); img.src = s.file;
        figure.append(caption, img); document.body.append(figure);
      }
    }, data.screens.slice(start, start + 6));
    await review.locator('img').evaluateAll(images => Promise.all(images.map(img => img.decode())));
    await review.screenshot({ path: path.join(evidence, `screens-${String(start + 1).padStart(2, '0')}.png`), fullPage: true });
    await review.close();
  }
  await writeFile(path.join(evidence, 'result.json'), JSON.stringify({ passed: true, checkedAt: new Date().toISOString(),
    browser: browser.version(), screenshots: data.screens.length, chapters: data.chapters.length, checksums: listed.length, media,
    checks: ['PNG dimensions', 'SHA256SUMS', 'search and empty results', 'category filter', 'lightbox arrows/Escape/focus',
      'desktop/mobile axe', 'no horizontal overflow', 'video decoding/playback/seeking', 'no JavaScript errors',
      'tampered manifest URL schemes/origins/traversal rejected'] }, null, 2));
  console.log(`PASS: ${listed.length} checksums; ${data.screens.length} screens; ${data.chapters.length} chapters; gallery, a11y and video playback`);
} finally { await browser.close(); }
