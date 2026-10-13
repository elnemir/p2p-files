import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {contractObserver} from './contracts';
import {evidencePath} from './evidence';

let originalValue: unknown;
test.afterEach(async({page,baseURL})=>{
 if(!originalValue)return;
 const latest=await (await page.request.get('/api/v1/branding')).json();
 const csrf=await page.evaluate(()=>localStorage.getItem('filex.csrf')!);
 expect((await page.request.put('/api/v1/admin/branding',{headers:{Origin:baseURL!,'X-CSRF-Token':csrf},data:{expectedRevision:latest.revision,value:originalValue}})).status()).toBe(200);
});

test('installation branding preview CAS publication SSR themes assets and reset',async({page,browser,baseURL})=>{
 test.setTimeout(120000);
 const contracts=await contractObserver();await contracts.observe(page);
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/');
 await page.getByLabel('Email',{exact:true}).fill(process.env.FILEX_E2E_ADMIN!);
 await page.getByLabel('Пароль',{exact:true}).fill(process.env.FILEX_E2E_PASSWORD!);
 await page.getByRole('button',{name:'Войти',exact:true}).click();
 await page.getByRole('button',{name:'Администрирование',exact:true}).click();await page.getByRole('navigation',{name:'Разделы администрирования'}).getByRole('button',{name:'Оформление',exact:true}).click();
 const editor=page.getByRole('region',{name:'Брендинг и оформление',exact:true});
 await expect(editor).toBeVisible();
 const original=await (await page.request.get('/api/v1/branding')).json();originalValue=original.value;
 const conditional=await page.evaluate(async()=>{
  const response=await fetch('/api/v1/branding');const etag=response.headers.get('ETag')!;
  const same=await fetch('/api/v1/branding',{headers:{'If-None-Match':etag}});
  const status=same.status,body=await same.text(),returnedTag=same.headers.get('ETag');
  const changed=await fetch('/api/v1/branding',{headers:{'If-None-Match':'"stale"'}});
  return {etag,status,body,returnedTag,changedStatus:changed.status,changed:await changed.json()};
 });
 expect(conditional.etag).toBeTruthy();expect(conditional.status).toBe(304);expect(conditional.body).toBe('');expect(conditional.returnedTag).toBe(conditional.etag);expect(conditional.changedStatus).toBe(200);expect(conditional.changed).toEqual(original);

 const observer=await browser.newContext({baseURL});const spectator=await observer.newPage();await spectator.goto('/help');
 const name='Северное пространство · Secure documents';
 try{
  await editor.getByLabel('Название продукта',{exact:true}).fill(name);
  await editor.getByLabel('Описание на русском',{exact:true}).fill('Документы под вашим брендом');
  await editor.getByLabel('Описание на английском',{exact:true}).fill('Documents under your brand');
  await editor.getByLabel('Подпись внизу страницы',{exact:true}).fill('Private workspace · 2026');
  await editor.getByRole('button',{name:'Океан',exact:true}).click();
  await editor.getByLabel('Стиль иконок',{exact:true}).selectOption('duotone');
  await editor.getByLabel('Тема по умолчанию',{exact:true}).selectOption('dark');
  const logo=await page.evaluate(()=>{const canvas=document.createElement('canvas');canvas.width=240;canvas.height=64;const ctx=canvas.getContext('2d')!;ctx.fillStyle='#2455a4';ctx.fillRect(0,0,240,64);ctx.fillStyle='#fff';ctx.font='bold 22px Arial';ctx.fillText('NORTH / SHARE',18,40);return canvas.toDataURL('image/png').split(',')[1]});
  for(const label of ['Светлый логотип','Тёмный логотип','Favicon']){await expect(editor.getByLabel(label,{exact:true})).toBeEnabled();await editor.getByLabel(label,{exact:true}).setInputFiles({name:'brand.png',mimeType:'image/png',buffer:Buffer.from(logo,'base64')});}
  // Drafts are local; the live header and a second visitor must still show the original revision.
  await expect(page.locator('header .brand')).toHaveAttribute('aria-label',original.value.productName);
  await expect(spectator.locator('header .brand')).toHaveAttribute('aria-label',original.value.productName);
  await expect(editor.getByRole('region',{name:'Светлый предпросмотр'})).toContainText(name);
  const save=editor.getByRole('button',{name:'Применить брендинг',exact:true});
  const applyBrand=async(status=200)=>{
    // Wait for the real mutation and contract capture before checking the UI update.
    const response=page.waitForResponse(r=>r.url().endsWith('/api/v1/admin/branding') && r.request().method()==='PUT');
    await save.click();expect((await response).status()).toBe(status);
  };
  const accent=editor.getByLabel('Светлая · Акцент',{exact:true});await accent.fill('#f7f8f4');await expect(save).toBeDisabled();await accent.fill('#2455a4');await expect(save).toBeEnabled();
  await applyBrand();await expect(editor.getByRole('status')).toContainText('Брендинг применён');
  await expect(page.locator('header .brand')).toHaveAttribute('aria-label',name);
  await spectator.bringToFront();await spectator.evaluate(()=>window.dispatchEvent(new Event('focus')));
  await expect(spectator.locator('header .brand')).toHaveAttribute('aria-label',name);
  await expect(spectator).toHaveTitle(/Северное пространство/);
  await expect(spectator.locator('html')).toHaveAttribute('data-theme','dark');
  await expect(page.locator('header [data-icon-style="duotone"]')).toHaveCount(2);
  expect(await page.locator('link[rel="icon"]').first().getAttribute('href')).toMatch(/^data:image\/png;base64,/);
  await editor.screenshot({path:evidencePath('branding-editor.png'),scale:'css'});
  await page.setViewportSize({width:320,height:900});
  for(const theme of ['light','dark','system']){
   await page.getByLabel('Тема',{exact:true}).selectOption(theme);
   expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
   expect((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze()).violations).toEqual([]);
  }
  await editor.screenshot({path:evidencePath('branding-editor-mobile.png'),scale:'css'});
  await page.evaluate(()=>window.scrollTo(0,0));
  await page.screenshot({path:evidencePath('branding-header-mobile.png'),scale:'css'});
  // A new document renders the configured name and logo before client JavaScript runs.
  const noJS=await browser.newContext({baseURL,javaScriptEnabled:false});
  try{const staticPage=await noJS.newPage();await staticPage.goto('/help');await expect(staticPage.locator('header .brand')).toHaveAttribute('aria-label',name);await expect(staticPage.locator('header img:visible')).toBeVisible();await expect(staticPage).toHaveTitle(/Северное пространство/)}finally{await noJS.close()}
  for(const path of ['/','/receive','/help','/status','/verify','/verify-report']){
   await spectator.goto(path);await expect(spectator.locator('header .brand')).toHaveAttribute('aria-label',name);await expect(spectator.locator('footer')).toContainText('Private workspace');
  }
  await spectator.getByLabel('Язык',{exact:true}).selectOption('en');await expect(spectator).toHaveTitle(/Documents under your brand/);
  await spectator.screenshot({path:evidencePath('branding-public.png'),fullPage:true,scale:'css'});
  // Publish in another authenticated session: stale drafts must not overwrite it.
  const applied=await (await page.request.get('/api/v1/branding')).json();
  const csrf=await page.evaluate(()=>localStorage.getItem('filex.csrf')!);
  const rival={...applied.value,productName:'Concurrent brand'};
  expect((await page.request.put('/api/v1/admin/branding',{headers:{Origin:baseURL!,'X-CSRF-Token':csrf},data:{expectedRevision:applied.revision,value:rival}})).status()).toBe(200);
  await editor.getByLabel('Название продукта',{exact:true}).fill('Stale draft');await applyBrand(409);await expect(editor.getByRole('alert')).toContainText('другой сессии');await expect(editor.getByLabel('Название продукта',{exact:true})).toHaveValue('Stale draft');
  await editor.getByRole('button',{name:'Загрузить актуальную версию',exact:true}).click();await expect(editor.getByLabel('Название продукта',{exact:true})).toHaveValue('Concurrent brand');
  const longName='ОченьДлинноеНазвание'.repeat(4);
  await editor.getByLabel('Название продукта',{exact:true}).fill(longName);
  for(const label of ['Светлый логотип','Тёмный логотип'])await editor.getByRole('button',{name:'Удалить · '+label,exact:true}).click();
  await applyBrand();await expect(page.locator('header .brand')).toHaveAttribute('aria-label',longName);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await editor.getByRole('button',{name:'Исходное оформление',exact:true}).click();await expect(page.locator('header .brand')).toHaveAttribute('aria-label',longName);
  await applyBrand();await expect(page.locator('header .brand')).toHaveAttribute('aria-label','FileX');await expect(page.locator('header .brand img')).toHaveCount(0);
  const result=await contracts.finish();expect(result.failures).toEqual([]);expect(errors).toEqual([]);
 }finally{
  await observer.close();
 }
});
