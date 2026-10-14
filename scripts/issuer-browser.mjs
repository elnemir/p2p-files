import { chromium } from "@playwright/test";
import { readFile, mkdir } from "node:fs/promises";
import assert from "node:assert/strict";
import AxeBuilder from "@axe-core/playwright";
const origin = (
  await readFile(process.env.ISSUER_FIXTURE_URL_FILE, "utf8")
).trim();
const output = ".cache/licensing-issuer-browser";
await mkdir(output, { recursive: true });
const browser = process.env.FILEX_BROWSER_WS
  ? await chromium.connect(process.env.FILEX_BROWSER_WS, {
      exposeNetwork: "<loopback>",
    })
  : await chromium.launch({ headless: true });
try {
  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
    viewport: { width: 1440, height: 1000 },
  });
  const page = await context.newPage();
  await page.goto(origin + "/issuer");
  await page.getByRole("link", { name: "Войти с MFA" }).click();
  await page
    .getByRole("heading", { name: "Новая лицензия или продление" })
    .waitFor();
  await page.getByLabel("Клиент / номер договора").fill("browser-customer");
  await page
    .getByLabel("Код установки", { exact: true })
    .fill("49ca5e21-4f82-4a02-93d9-a2f939c711f5");
  await page.getByLabel("Подписка до, включительно (UTC)").fill("2027-10-14");
  await page.getByLabel("Обновления до (UTC)").fill("2027-10-14");
  await page.getByLabel("Поддержка до (UTC)").fill("2027-10-14");
  await page.getByLabel("WebDAV", { exact: true }).check();
  await page.getByRole("button", { name: "Подготовить к выпуску" }).click();
  await page
    .getByRole("heading", { name: "Проверка перед подписью" })
    .waitFor();
  await page.screenshot({
    path: output + "/preview-desktop.png",
    fullPage: true,
  });
  assert.deepEqual(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => v.id),
    [],
  );
  await page
    .getByRole("button", { name: "Подтвердить и передать на подпись" })
    .click();
  await page.getByRole("status").waitFor();
  await page.getByRole("button", { name: "Обновить", exact: true }).click();
  for (
    let i = 0;
    i < 20 &&
    (await page.getByRole("link", { name: "Скачать .jws" }).count()) === 0;
    i++
  ) {
    await new Promise((r) => setTimeout(r, 100));
    await page.getByRole("button", { name: "Обновить", exact: true }).click();
  }
  const downloadEvent = page.waitForEvent("download");
  await page.getByRole("link", { name: "Скачать .jws" }).click();
  const download = await downloadEvent;
  const file = output + "/fixture-license.jws";
  await download.saveAs(file);
  assert.equal((await readFile(file, "utf8")).split(".").length, 3);
  await page.getByRole("button", { name: "Продлить", exact: true }).click();
  assert.equal(await page.getByLabel("Версия выдачи").inputValue(), "2");
  await page.getByRole("combobox").selectOption("perpetual");
  await page.getByRole("button", { name: "Подготовить к выпуску" }).click();
  assert.match(await page.locator("pre").innerText(), /"expiresAt": null/);
  await page.setViewportSize({ width: 390, height: 844 });
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth,
    ),
    false,
  );
  await page.screenshot({
    path: output + "/renewal-mobile.png",
    fullPage: true,
  });
  assert.deepEqual(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => v.id),
    [],
  );
  console.log(
    "PASS: issuer OIDC login, preview, signing, download, perpetual renewal, desktop/mobile and axe. Synthetic local MFA provider only.",
  );
} finally {
  await browser.close();
}
