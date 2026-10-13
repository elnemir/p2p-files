import {test,expect} from '@playwright/test';
import {spawn} from 'node:child_process';
import {createInterface} from 'node:readline';
import {inflateRawSync} from 'node:zlib';
import {randomUUID} from 'node:crypto';
import AxeBuilder from '@axe-core/playwright';
import {contractObserver} from './contracts';
import {evidencePath} from './evidence';

test('external SAML registration browser binding reauthentication and provider revocation',async({page,browser,baseURL})=>{
 test.setTimeout(180000);
 const providerName='КорпоративныйВход'.repeat(5).slice(0,80);
 const fixture=spawn('go',['run','./scripts/identity-fixture'],{env:{...process.env,GOMODCACHE:`${process.cwd()}/.cache/gomod`,GOCACHE:`${process.cwd()}/.cache/go-build`},stdio:['pipe','pipe','pipe']});
 const lines=createInterface({input:fixture.stdout})[Symbol.asyncIterator]();
 let stderr='';fixture.stderr.on('data',d=>stderr+=d.toString());
 const read=async()=>{const result=await lines.next();if(result.done)throw new Error(`Identity fixture stopped: ${stderr}`);return JSON.parse(result.value)};
 let provider:any;let csrf='';let ownerContext:Awaited<ReturnType<typeof browser.newContext>>|undefined;
 const contracts=await contractObserver();await contracts.observe(page);
 try{
  const {metadata}=await read();
  await page.goto('/');await page.getByLabel('Email',{exact:true}).fill(process.env.FILEX_E2E_ADMIN!);await page.getByLabel('Пароль',{exact:true}).fill(process.env.FILEX_E2E_PASSWORD!);await page.getByRole('button',{name:'Войти',exact:true}).click();await page.getByRole('button',{name:'Администрирование',exact:true}).click();
  const panel=page.getByRole('region',{name:'Провайдеры авторизации',exact:true});await expect(panel).toBeVisible();
  await panel.getByLabel('Название кнопки входа',{exact:true}).fill(providerName);await panel.getByLabel('Протокол',{exact:true}).selectOption('saml');await panel.getByLabel('Metadata XML провайдера (с сертификатом подписи)',{exact:true}).fill(metadata);await panel.getByLabel('Разрешить вход',{exact:true}).check();await panel.getByLabel('Создавать USER при первом входе',{exact:true}).check();await panel.getByRole('button',{name:'Сохранить провайдера',exact:true}).click();await expect(panel.getByRole('status')).toContainText('Провайдер сохранён');
  csrf=await page.evaluate(()=>localStorage.getItem('filex.csrf')!);provider=(await (await page.request.get('/api/v1/admin/identity-providers')).json()).find((p:any)=>p.config.name===providerName&&p.config.enabled);expect(provider).toBeTruthy();
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);expect((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze()).violations).toEqual([]);await panel.screenshot({path:evidencePath('identity-providers.png'),scale:'css'});
  ownerContext=await browser.newContext({baseURL,viewport:{width:390,height:844}});const owner=await ownerContext.newPage();
  const email=`external-${randomUUID()}@example.test`;const subject=randomUUID();let assertion='',state='';
  await owner.route('https://identity-fixture.example.test/login?**',async route=>{
   const url=new URL(route.request().url());state=url.searchParams.get('RelayState')!;const xml=inflateRawSync(Buffer.from(url.searchParams.get('SAMLRequest')!,'base64')).toString();const requestID=xml.match(/\bID="([^"]+)"/)?.[1];expect(requestID).toBeTruthy();
   fixture.stdin.write(JSON.stringify({RequestID:requestID,ACS:`${baseURL}/api/v1/auth/providers/${provider.id}/saml`,EntityID:`${baseURL}/api/v1/auth/providers/${provider.id}/metadata`,Email:email,Subject:subject})+'\n');assertion=(await read()).response;
   await route.fulfill({contentType:'text/html',body:'<!doctype html><html><body><h1>Isolated test identity provider</h1><button id="continue">Continue</button></body></html>'});
  });
  async function finishIdP(){await expect(owner.getByRole('heading',{name:'Isolated test identity provider'})).toBeVisible();await owner.evaluate(({acs,response,relay})=>{const form=document.createElement('form');form.method='POST';form.action=acs;for(const [name,value] of Object.entries({SAMLResponse:response,RelayState:relay})){const input=document.createElement('input');input.name=name;input.value=value;form.appendChild(input)}document.body.appendChild(form);form.submit()},{acs:`${baseURL}/api/v1/auth/providers/${provider.id}/saml`,response:assertion,relay:state});await expect(owner.getByRole('button',{name:'Новая передача',exact:true})).toBeVisible();}
  await owner.goto('/');await expect(owner.getByRole('button',{name:'Войти через '+providerName,exact:true})).toBeVisible();expect(await owner.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await owner.getByRole('button',{name:'Войти через '+providerName,exact:true}).click();await finishIdP();
  const me=await (await owner.request.get('/api/v1/auth/me')).json();expect(me).toMatchObject({email,role:'user'});expect(owner.url()).not.toContain('authFlow');
  expect((await owner.request.post(`/api/v1/auth/providers/${provider.id}/saml`,{form:{SAMLResponse:assertion,RelayState:state}})).status()).toBe(401);
  await owner.getByRole('button',{name:'Настройки',exact:true}).click();await owner.getByRole('button',{name:'Подтвердить через '+providerName,exact:true}).click();await finishIdP();expect((await (await owner.request.get('/api/v1/auth/me')).json()).id).toBe(me.id);
  expect((await page.request.put(`/api/v1/admin/identity-providers/${provider.id}`,{headers:{Origin:baseURL!,'X-CSRF-Token':csrf},data:{expectedRevision:provider.revision,config:{...provider.config,enabled:false}}})).status()).toBe(200);
  expect((await owner.request.get('/api/v1/auth/me')).status()).toBe(401);provider=undefined;
  const result=await contracts.finish();expect(result.failures).toEqual([]);
 }finally{
  if(provider&&csrf){const current=(await (await page.request.get('/api/v1/admin/identity-providers')).json()).find((p:any)=>p.id===provider.id);if(current)expect((await page.request.put(`/api/v1/admin/identity-providers/${provider.id}`,{headers:{Origin:baseURL!,'X-CSRF-Token':csrf},data:{expectedRevision:current.revision,config:{...current.config,enabled:false}}})).status()).toBe(200)}
  await ownerContext?.close();fixture.stdin.end();fixture.kill();
 }
});
