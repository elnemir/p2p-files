import {evidencePath} from "./evidence";
import { test,expect, type Page } from '@playwright/test';
import {contractObserver} from './contracts';
import AxeBuilder from '@axe-core/playwright';
const email=process.env.FILEX_E2E_EMAIL!; const password=process.env.FILEX_E2E_PASSWORD!;
async function login(page:Page){await page.goto('/');await page.getByLabel('Email',{exact:true}).fill(email);await page.getByLabel('Пароль',{exact:true}).fill(password);await page.getByRole('button',{name:'Войти',exact:true}).click();await expect(page.getByRole('button',{name:'Новая передача'})).toBeVisible();}
async function a11y(page:Page){const results=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze();expect(results.violations).toEqual([]);}
test('authenticated upload resumes, publishes, issues ZIP, renders reports and exports',async({page,browser})=>{
 const contract=await contractObserver();contract.observe(page);const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));await login(page);await a11y(page);
 await page.getByRole('button',{name:'Новая передача'}).click();await page.getByLabel('Название',{exact:true}).fill('Контракт · длинное название для проверки переноса и всех действий');await page.getByRole('button',{name:'Сохранить название'}).click();
 let interrupted=false;await page.route('**/api/v1/uploads/*',async route=>{if(route.request().method()==='PATCH'&&!interrupted){interrupted=true;await route.fetch();await route.abort('connectionreset');}else await route.continue();});
 await page.locator('input[type=file]:not([webkitdirectory])').setInputFiles([{name:'договор.txt',mimeType:'text/plain',buffer:Buffer.from('Тестовые данные договора\n')},{name:'a-very-long-file-name-for-responsive-layout-and-actions-012345678901234567890.txt',mimeType:'text/plain',buffer:Buffer.alloc(1024*64,65)}]);
 await expect(page.getByRole('button',{name:'Проверить перед публикацией'})).toBeEnabled({timeout:30000});expect(interrupted).toBe(true);await a11y(page);await page.screenshot({path:evidencePath('draft-desktop.png'),fullPage:true});
 await page.getByRole('button',{name:'Проверить перед публикацией'}).click();await page.getByRole('button',{name:'Опубликовать передачу'}).click();await expect(page.getByRole('heading',{name:'Ссылки получателей'})).toBeVisible();const link=await page.getByLabel('Ссылка получателя',{exact:true}).inputValue();await page.screenshot({path:evidencePath('published-desktop.png'),fullPage:true});
 const qrPromise=page.waitForEvent('download');await page.getByRole('button',{name:'Скачать все QR',exact:true}).click();await (await qrPromise).saveAs(evidencePath('qr-links.zip'));
 const recipient=await browser.newContext({acceptDownloads:true,viewport:{width:390,height:844}});const receive=await recipient.newPage();contract.observe(receive);await receive.goto(link);await receive.getByRole('button',{name:'Открыть файлы'}).click();await expect(receive.getByRole('button',{name:'Скачать всё ZIP'})).toBeVisible();await a11y(receive);await receive.screenshot({path:evidencePath('recipient-mobile.png'),fullPage:true});const downloadPromise=receive.waitForEvent('download');await receive.getByRole('button',{name:'Скачать всё ZIP'}).click();const download=await downloadPromise;await download.saveAs(evidencePath('transfer.zip'));expect(await download.failure()).toBeNull();await recipient.close();
 await page.getByRole('button',{name:'Создать отчёт',exact:true}).click();await expect(page.getByRole('button',{name:'Пользовательский · #1'})).toBeVisible();await expect(page.locator('.list-row').filter({hasText:'Пользовательский'})).toContainText('Готово',{timeout:30000});await page.getByRole('button',{name:'Пользовательский · #1'}).click();await expect(page.getByRole('button',{name:'PDF',exact:true})).toBeVisible();const pdfPromise=page.waitForEvent('download');await page.getByRole('button',{name:'PDF',exact:true}).click();await (await pdfPromise).saveAs(evidencePath('report.pdf'));const jsonPromise=page.waitForEvent('download');await page.getByRole('button',{name:'JSON',exact:true}).click();await (await jsonPromise).saveAs(evidencePath('report.json'));await a11y(page);
 await page.getByRole('link',{name:'Проверить',exact:true}).click();await page.getByRole('button',{name:'Проверить',exact:true}).click();await page.getByLabel('JSON отчёта для сравнения').setInputFiles(evidencePath('report.json'));await expect(page.getByText('Контрольная сумма совпадает',{exact:true})).toBeVisible();await a11y(page);
 await page.goto('/');await page.getByRole('button',{name:'Задачи',exact:true}).click();await page.getByRole('button',{name:'+ CSV',exact:true}).click();await expect(page.locator('.list-row').filter({hasText:'CSV'})).toContainText('Завершено',{timeout:30000});const csvPromise=page.waitForEvent('download');await page.locator('.list-row').filter({hasText:'CSV'}).getByRole('button',{name:'Скачать',exact:true}).click();await (await csvPromise).saveAs(evidencePath('export.csv'));await page.setViewportSize({width:390,height:844});await a11y(page);await page.screenshot({path:evidencePath('jobs-mobile.png'),fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false);
 await page.getByRole('button',{name:'Настройки',exact:true}).click();await page.getByLabel('Тема',{exact:true}).selectOption('dark');await a11y(page);await page.screenshot({path:evidencePath('settings-mobile-dark.png'),fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false);expect(errors).toEqual([]);const result=await contract.finish();expect(result.count).toBeGreaterThan(20);expect(result.failures).toEqual([]);
});
test('public pages, locale and internal route isolation',async({page,request})=>{
 // Hold hydration to exercise the real SSR state instead of relying on a fast browser.
 const reportId='00000000-0000-4000-8000-000000000001';
 let releaseScripts!:()=>void;const scriptsReady=new Promise<void>(resolve=>{releaseScripts=resolve;});
 const scriptRoute='**/_next/static/**/*.js';
 await page.route(scriptRoute,async route=>{await scriptsReady;await route.continue();});
 try{
  await page.goto(`/verify#report=${reportId}`,{waitUntil:'commit'});
  await expect(page.getByLabel('Идентификатор отчёта',{exact:true})).toBeDisabled();
  await expect(page.getByRole('button',{name:'Проверить',exact:true})).toBeDisabled();
 }finally{releaseScripts();}
 await expect(page.getByLabel('Идентификатор отчёта',{exact:true})).toHaveValue(reportId);
 await expect(page.getByRole('button',{name:'Проверить',exact:true})).toBeEnabled();
 await page.unroute(scriptRoute);
 const verified=page.waitForResponse(response=>response.url().endsWith(`/api/v1/verify/reports/${reportId}`));
 await page.getByRole('button',{name:'Проверить',exact:true}).click();
 expect((await verified).status()).toBe(404);
 await page.getByLabel('Идентификатор отчёта',{exact:true}).fill('');
 await expect(page.getByRole('button',{name:'Проверить',exact:true})).toBeDisabled();
 await page.goto('/help');await a11y(page);await page.getByLabel('Язык',{exact:true}).selectOption('en');await expect(page.getByRole('heading',{name:'Help',exact:true})).toBeVisible();await page.setViewportSize({width:360,height:780});await a11y(page);await page.screenshot({path:evidencePath('help-mobile-en.png'),fullPage:true});await page.goto('/status');await expect(page.getByText('Operational',{exact:true})).toBeVisible();for(const path of ['/health','/ready','/metrics'])expect((await request.get(path)).status()).toBe(404);expect((await request.get('/api/v1/transfers')).status()).toBe(401);});
