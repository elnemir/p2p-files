import {test} from 'node:test';
import {chromium,expect} from '@playwright/test';
import {readFile} from 'node:fs/promises';
import AxeBuilder from '@axe-core/playwright';
test('Both setup wizards', async()=>{
const origin=(await readFile(process.env.ISSUER_FIXTURE_URL_FILE,'utf8')).trim();
const fixture=JSON.parse(await readFile(process.env.ISSUER_FIXTURE_URL_FILE+'.setup.json','utf8'));
const browser=process.env.FILEX_BROWSER_WS ? await chromium.connect(process.env.FILEX_BROWSER_WS,{exposeNetwork:'<loopback>'}) : await chromium.launch({headless:true});
try {
 const context=await browser.newContext({ignoreHTTPSErrors:true,viewport:{width:390,height:844}});
 const page=await context.newPage();
 for(const issuer of [false,true]) {
  const path=issuer?'/issuer/setup':'/setup';const endpoint=issuer?'/issuer/api/setup':'/api/v1/setup';
  await page.goto(origin+path);
  await page.getByLabel('Токен настройки').fill(fixture.token);
  await page.getByRole('button',{name:'Продолжить',exact:true}).click();
  if(issuer){
   for(const [label,value] of [['Адрес OIDC-провайдера',fixture.issuer],['Client ID','fixture-client'],['Client secret','fixture-secret'],['MFA ACR','fixture-mfa'],['Subject (sub) первого оператора','browser-operator']])await page.getByLabel(label,{exact:true}).fill(value);
  }else{
   await page.getByLabel('Email администратора').fill('setup@example.test');
   await page.getByLabel('Пароль администратора',{exact:true}).fill('setup-browser-password-123');
   await page.getByLabel('Повторите пароль').fill('different-password');
   await page.getByRole('button',{name:'Продолжить',exact:true}).click();
   await expect(page.locator('main').getByRole('alert')).toContainText('Пароли не совпадают');
   await page.getByLabel('Повторите пароль').fill('setup-browser-password-123');
  }
  expect((await new AxeBuilder({page}).analyze()).violations).toEqual([]);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.screenshot({path:'.cache/licensing-issuer-browser/setup-'+(issuer?'issuer':'product')+'-mobile.png',fullPage:true});
  await page.getByRole('button',{name:'Продолжить',exact:true}).click();
  await expect(page.getByText(issuer?'fixture-secret':'setup-browser-password-123',{exact:true})).toHaveCount(0);
  const saved=page.waitForResponse(r=>r.url().endsWith(endpoint)&&r.request().method()==='POST');
  await page.getByRole('button',{name:'Завершить настройку',exact:true}).click();
  expect((await saved).status()).toBe(200);
  await expect(page.getByRole('heading',{name:'Настройка завершена'})).toBeVisible();
  await page.reload();await expect(page.getByRole('heading',{name:'Настройка завершена'})).toBeVisible();
  await expect(page.getByLabel('Токен настройки')).toHaveCount(0);
  await page.getByRole('link',{name:'Перейти ко входу'}).click();
  if(issuer)await expect(page.getByRole('heading',{name:'Новая лицензия или продление'})).toBeVisible();
  else {
   await page.getByLabel('Email',{exact:true}).fill('setup@example.test');await page.getByLabel('Пароль',{exact:true}).fill('setup-browser-password-123');
   await page.getByRole('button',{name:'Войти',exact:true}).click();
   await expect(page.getByRole('button',{name:'Администрирование',exact:true})).toBeVisible();
  }
 }
 await context.close();console.log('Both setup wizards: real API/PG, mobile axe, completion/reload, ADMIN and OIDC MFA login PASS');
} finally {await browser.close();}

});
