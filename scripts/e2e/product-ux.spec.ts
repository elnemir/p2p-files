import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { evidencePath } from './evidence';
import { createDraft } from './transfers';
import { contractObserver } from './contracts';

async function login(page: Page, admin = false) {
  await page.goto('/');
  await page.getByLabel('Email', { exact: true }).fill(process.env[admin ? 'FILEX_E2E_ADMIN' : 'FILEX_E2E_EMAIL']!);
  await page.getByLabel('Пароль', { exact: true }).fill(process.env.FILEX_E2E_PASSWORD!);
  await page.getByRole('button', { name: 'Войти', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Новая передача', exact: true })).toBeVisible();
}
async function a11y(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  expect((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze()).violations).toEqual([]);
}
async function headers(page: Page, origin: string) {
  return { Origin: origin, 'X-CSRF-Token': await page.evaluate(() => localStorage.getItem('filex.csrf')!) };
}

test('guided publication preserves metadata, validates access, confirms deletion and clears completed recipient controls', async ({page, browser, baseURL}) => {
  test.setTimeout(120000);
  const contracts = await contractObserver(); await contracts.observe(page);
  await login(page);
  const id = await createDraft(page, 'UX · подготовка передачи');
  const requestHeaders = await headers(page, baseURL!);
  expect((await page.request.patch(`/api/v1/transfers/${id}`, {headers: requestHeaders, data: {title: 'UX · подготовка передачи', message: 'Сохранить сообщение из API'}})).status()).toBe(204);
  await page.getByRole('button', {name:'Все передачи',exact:true}).click();
  await page.getByRole('button', {name:/UX · подготовка передачи/}).click();
  await page.getByLabel('Название', {exact:true}).fill('UX · название сохраняется перед проверкой');
  await page.locator('input[type=file]:not([webkitdirectory])').setInputFiles({name:'Документ с длинным именем для удобного интерфейса.txt',mimeType:'text/plain',buffer:Buffer.from('UX fixture\n')});
  const review = page.getByRole('button', {name:'Проверить перед публикацией',exact:true});
  await expect(review).toBeEnabled();
  await expect(page.getByLabel(/Новый путь /)).not.toBeVisible();
  await page.getByText('Действия с файлом',{exact:true}).click();
  await page.getByRole('button',{name:'Удалить',exact:true}).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('Удалить файл?');
  await page.keyboard.press('Escape');
  await expect(dialog).not.toBeVisible();
  expect((await (await page.request.get(`/api/v1/transfers/${id}`)).json()).files).toHaveLength(1);
  await page.getByRole('button',{name:'3 дня',exact:true}).click();
  await expect(page.getByLabel('Срок доступа, часов',{exact:true})).toHaveValue('72');
  const slider = page.getByRole('slider');await slider.focus();await slider.press('ArrowRight');
  await expect(page.getByLabel('Срок доступа, часов',{exact:true})).toHaveValue('73');
  await page.getByLabel('PIN или пароль для ссылок (необязательно)',{exact:true}).fill('123');
  await expect(review).toBeDisabled();
  await expect(page.getByRole('status').filter({hasText:'Используйте PIN'})).toBeVisible();
  await page.getByLabel('PIN или пароль для ссылок (необязательно)',{exact:true}).fill('');
  await page.getByLabel('Создать общую ссылку',{exact:true}).uncheck();await expect(review).toBeDisabled();
  await page.getByLabel('Создать общую ссылку',{exact:true}).check();
  await review.click();
  await expect(page.getByRole('region',{name:'Проверка публикации'})).toBeFocused();
  expect(await (await page.request.get(`/api/v1/transfers/${id}/editor`)).json()).toMatchObject({title:'UX · название сохраняется перед проверкой',message:'Сохранить сообщение из API'});
  const uploaded=(await (await page.request.get(`/api/v1/transfers/${id}`)).json()).files[0];
  expect((await page.request.patch(`/api/v1/files/${uploaded.id}`,{headers:requestHeaders,data:{path:'Документ с новым проверенным путём.txt'}})).status()).toBe(204);
  await expect(page.getByRole('region',{name:'Проверка публикации'})).toHaveCount(0,{timeout:10000});
  await review.click();await expect(page.getByRole('region',{name:'Проверка публикации'})).toBeFocused();
  await page.screenshot({path:evidencePath('ux-draft-review.png'),fullPage:true});await a11y(page);
  await page.getByRole('button',{name:'Опубликовать передачу',exact:true}).click();
  const link = await page.getByLabel('Ссылка получателя',{exact:true}).inputValue();
  const context = await browser.newContext({baseURL,acceptDownloads:true,viewport:{width:390,height:844}});
  let releaseOld = () => {};
  try {
    const receive = await context.newPage();await contracts.observe(receive);await receive.goto(link);
    await receive.getByRole('button',{name:'Открыть файлы',exact:true}).click();
    await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toBeVisible();
    // An explicit new capability must never silently display the previous link's files.
    await receive.goto('/receive#token=invalid-ux-capability');
    await expect(receive.getByRole('button',{name:'Открыть файлы',exact:true})).toBeVisible();
    await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toHaveCount(0);
    await receive.goto(link);await receive.getByRole('button',{name:'Открыть файлы',exact:true}).click();
    await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toBeVisible();
    // A transient transport failure must leave valid controls and session available.
    await receive.route('**/api/v1/recipient/files', route => route.abort('connectionreset'));
    await receive.getByRole('button',{name:'Обновить список',exact:true}).click();
    await expect(receive.getByRole('status').filter({hasText:'Не удалось обновить список'})).toBeVisible();
    await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toBeVisible();
    await receive.unroute('**/api/v1/recipient/files');
    let oldCaptured=false;let holdOne=true;
    const oldResponse=new Promise<void>(resolve=>{releaseOld=resolve;});
    await receive.route('**/api/v1/recipient/files',async route=>{
      if(holdOne){holdOne=false;const response=await route.fetch();oldCaptured=true;await oldResponse;await route.fulfill({response});}else await route.fallback();
    });
    await expect.poll(()=>oldCaptured,{timeout:10000}).toBe(true);
    // Native browser downloads bypass Playwright routing. Cancel the initial client
    // navigation before content GET, then resume the real server reservation via UI.
    await receive.evaluate(()=>{
      const cancel=(event:MouseEvent)=>{
        const link=event.target instanceof Element?event.target.closest('a[download]'):null;
        if(link instanceof HTMLAnchorElement && link.pathname.startsWith('/api/v1/recipient/downloads/')) {
          event.preventDefault();document.removeEventListener('click',cancel,true);
          document.documentElement.dataset.testDownloadCancelled='true';
        }
      };
      document.addEventListener('click',cancel,true);
    });
    await receive.getByRole('button',{name:'Скачать всё ZIP',exact:true}).click();
    await expect(receive.locator('html')).toHaveAttribute('data-test-download-cancelled','true');
    const resume=receive.getByRole('link',{name:'Продолжить начатое скачивание',exact:true});
    await expect(resume).toBeVisible();
    await receive.getByRole('button',{name:'Обновить список',exact:true}).click();await expect(resume).toBeVisible();
    const remaining=await receive.request.get('/api/v1/recipient/files');expect(remaining.status()).toBe(200);
    expect((await remaining.json()).every((file:any)=>file.state==='leased')).toBe(true);
    const download = receive.waitForEvent('download');await resume.click();
    await (await download).saveAs(evidencePath('ux-completed.zip'));
    await receive.getByRole('button',{name:'Обновить список',exact:true}).click();
    await expect(receive.getByRole('heading',{name:'Доступ к файлам завершён',exact:true})).toBeVisible({timeout:15000});
    releaseOld();await receive.unrouteAll({behavior:'wait'});
    await expect(receive.getByRole('heading',{name:'Доступ к файлам завершён',exact:true})).toBeVisible();
    await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toHaveCount(0);
    await expect(receive.getByRole('link',{name:'Продолжить начатое скачивание',exact:true})).toHaveCount(0);
    await a11y(receive);await receive.screenshot({path:evidencePath('ux-recipient-completed.png'),fullPage:true});
    // Revoke and natural expiry have the same privacy-safe terminal presentation.
    for (const closure of ['revoke','expiry']) {
      await page.getByRole('button',{name:'Все передачи',exact:true}).click();
      await createDraft(page,`UX-${closure}`);
      await page.locator('input[type=file]:not([webkitdirectory])').setInputFiles({name:'closure.txt',mimeType:'text/plain',buffer:Buffer.from('closure fixture\n')});
      await expect(review).toBeEnabled();
      if(closure==='expiry')await page.getByLabel('Срок доступа, часов',{exact:true}).fill(String(12/3600));
      await review.click();await page.getByRole('button',{name:'Опубликовать передачу',exact:true}).click();
      await expect(page.getByLabel('Ссылка получателя',{exact:true})).toBeVisible();
      await receive.goto(await page.getByLabel('Ссылка получателя',{exact:true}).inputValue());
      await receive.getByRole('button',{name:'Открыть файлы',exact:true}).click();
      await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toBeVisible();
      if(closure==='revoke') {
        await page.getByRole('button',{name:'Отозвать всю передачу',exact:true}).click();
        await page.getByRole('dialog').getByRole('button',{name:'Отозвать передачу',exact:true}).click();
      }
      await expect(receive.getByRole('heading',{name:'Доступ к файлам завершён',exact:true})).toBeVisible({timeout:20000});
      await expect(receive.getByRole('button',{name:'Скачать всё ZIP',exact:true})).toHaveCount(0);
    }
  } finally { releaseOld();await context.close(); }
  expect((await contracts.finish()).failures).toEqual([]);
});

test('search ignores late results and settings offer accessible timezone and quiet hours', async ({page}) => {
  await login(page);
  await createDraft(page,'UX-search-alpha');await page.getByRole('button',{name:'Все передачи',exact:true}).click();
  await createDraft(page,'UX-search-beta');await page.getByRole('button',{name:'Все передачи',exact:true}).click();
  let held!: () => void;
  const release = new Promise<void>(resolve => {held=resolve;});
  let captured!: () => void;
  const capturedPromise=new Promise<void>(resolve=>{captured=resolve;});
  await page.route('**/api/v1/transfers/search',async route=>{
    if(route.request().postDataJSON()?.query==='UX-search-alpha') {const response=await route.fetch();captured();await release;await route.fulfill({response});} else await route.fallback();
  });
  const search=page.getByLabel('Поиск по названию',{exact:true});
  await search.fill('UX-search-alpha');await capturedPromise;
  await search.fill('UX-search-beta');
  await expect(page.locator('.rows .row-link')).toHaveCount(1);await expect(page.locator('.rows .row-link')).toContainText('UX-search-beta');
  held();await expect(page.locator('.rows .row-link')).toContainText('UX-search-beta');
  await page.unroute('**/api/v1/transfers/search');
  await search.fill('UX-no-such-transfer');await expect(page.getByRole('heading',{name:'Ничего не найдено',exact:true})).toBeVisible();
  await page.getByRole('button',{name:'Сбросить фильтры',exact:true}).click();await expect(search).toHaveValue('');
  expect(page.url()).not.toContain('UX-search');
  await page.getByRole('button',{name:'Настройки',exact:true}).click();
  await page.getByLabel('Часовой пояс IANA',{exact:true}).selectOption('Europe/Moscow');
  await page.getByLabel('Включить тихие часы',{exact:true}).check();
  await page.getByLabel('Тихие часы: с',{exact:true}).selectOption('23');
  await page.getByLabel('Тихие часы: до',{exact:true}).selectOption('7');
  await page.getByRole('button',{name:'Сохранить',exact:true}).click();await expect(page.getByRole('status')).toContainText('Настройки сохранены');
  expect(await (await page.request.get('/api/v1/notification-preferences')).json()).toMatchObject({timezone:'Europe/Moscow',quietStart:23,quietEnd:7});
  await page.setViewportSize({width:320,height:900});await a11y(page);await page.screenshot({path:evidencePath('ux-settings-mobile.png'),fullPage:true});
  await page.getByRole('button',{name:'Аккаунт и безопасность',exact:true}).click();await expect(page.getByRole('heading',{name:'Сеансы',exact:true})).toBeVisible();
  const reauth=page.locator('section').filter({has:page.getByRole('heading',{name:'Подтверждение личности',exact:true})});
  await reauth.getByLabel('Текущий пароль',{exact:true}).fill('wrong-ux-password');
  const rejected=page.waitForResponse(r=>r.url().endsWith('/auth/reauth') && r.request().method()==='POST');
  await reauth.getByRole('button',{name:'Подтвердить на 5 минут',exact:true}).click();expect((await rejected).status()).toBe(401);
  await expect(page.getByRole('main').getByRole('alert')).toContainText('проверьте ссылку и пароль');
  await expect(page.getByRole('heading',{name:'Сеансы',exact:true})).toBeVisible();expect((await page.request.get('/api/v1/auth/me')).status()).toBe(200);
  await page.getByRole('button',{name:'Уведомления и язык',exact:true}).click();await expect(page.getByLabel('Тихие часы: с',{exact:true})).toHaveValue('23');
  await page.getByLabel('Включить тихие часы',{exact:true}).uncheck();await page.getByRole('button',{name:'Сохранить',exact:true}).click();
  expect(await (await page.request.get('/api/v1/notification-preferences')).json()).toMatchObject({quietStart:null,quietEnd:null});
  const me=await (await page.request.get('/api/v1/auth/me')).json();
  const h=await headers(page,new URL(page.url()).origin);
  expect((await page.request.delete(`/api/v1/auth/sessions/${me.sessionId}`,{headers:h})).status()).toBe(204);
  await expect(page.getByRole('button',{name:'Войти',exact:true})).toBeVisible({timeout:15000});
  await expect(page.getByRole('button',{name:'UX-search-beta'})).toHaveCount(0);

  // Do not navigate or reload here: that would hide stale in-memory state.
  const signInAgain=async(admin=false)=>{
    await page.getByLabel('Email',{exact:true}).fill(process.env[admin?'FILEX_E2E_ADMIN':'FILEX_E2E_EMAIL']!);
    await page.getByLabel('Пароль',{exact:true}).fill(process.env.FILEX_E2E_PASSWORD!);
    await page.getByRole('button',{name:'Войти',exact:true}).click();
    await expect(page.getByRole('button',{name:'Новая передача',exact:true})).toBeVisible();
  };
  await signInAgain();
  let releaseEditor!:()=>void;let editorCaptured!:()=>void;let editorDelivered!:()=>void;let hasEditor=false;
  const heldEditor=new Promise<void>(resolve=>{releaseEditor=resolve;});
  const capturedEditor=new Promise<void>(resolve=>{editorCaptured=resolve;});
  const deliveredEditor=new Promise<void>(resolve=>{editorDelivered=resolve;});
  await page.route('**/api/v1/transfers/*/editor',async route=>{
    hasEditor=true;
    try {const response=await route.fetch();editorCaptured();await heldEditor;await route.fulfill({response});}
    finally {editorDelivered();}
  });
  try {
    await page.getByRole('button',{name:'UX-search-beta'}).click();await capturedEditor;
    const oldSession=await (await page.request.get('/api/v1/auth/me')).json();
    expect((await page.request.delete(`/api/v1/auth/sessions/${oldSession.sessionId}`,{headers:await headers(page,new URL(page.url()).origin)})).status()).toBe(204);
    await expect(page.getByRole('heading',{name:'Войти в FileX',exact:true})).toBeVisible({timeout:15000});
    await expect(page.getByRole('button',{name:'Войти',exact:true})).toBeEnabled();
    await signInAgain(true);
    await expect(page.getByRole('button',{name:'Администрирование',exact:true})).toBeVisible();
    const refreshed=page.waitForResponse(response=>response.url().endsWith('/auth/me') && response.status()===200);
    releaseEditor();await deliveredEditor;await refreshed;
    await expect(page.getByRole('button',{name:'Новая передача',exact:true})).toBeVisible();
    await expect(page.getByRole('heading',{name:'UX-search-beta',exact:true})).toHaveCount(0);
  } finally {releaseEditor();if(hasEditor)await deliveredEditor;await page.unroute('**/api/v1/transfers/*/editor');}

});

test('admin typed settings validate before apply, preserve network values, detect CAS conflict and retain form input',async({page,baseURL})=>{
  test.setTimeout(120000);
  await login(page,true);const contracts=await contractObserver();await contracts.observe(page);
  const h=await headers(page,baseURL!);
  const getActive=async()=> (await (await page.request.get('/api/v1/admin/settings',{timeout:10000})).json()).find((r:any)=>r.active);
  const original=await getActive();
  async function applyValue(value:unknown) {
    const current=await getActive();
    const response=await page.request.post('/api/v1/admin/settings',{headers:h,data:value});expect(response.status()).toBe(201);
    const {revision}=await response.json();expect((await page.request.post(`/api/v1/admin/settings/${revision}/validate`,{headers:h})).status()).toBe(204);
    expect((await page.request.post(`/api/v1/admin/settings/${revision}/apply`,{headers:h,data:{expectedRevision:current.revision}})).status()).toBe(204);
  }
  try {
    await page.getByRole('button',{name:'Администрирование',exact:true}).click();
    const nav=page.getByRole('navigation',{name:'Разделы администрирования'});
    await nav.getByRole('button',{name:'Параметры системы',exact:true}).click();
    const panel=page.getByRole('region',{name:'Настройки приложения',exact:true});
    await expect(panel.getByLabel('Максимум файлов в передаче',{exact:true})).toHaveValue(String(original.value.maxFiles));
    await panel.getByLabel('Максимум файлов в передаче',{exact:true}).fill('9000');
    await panel.getByLabel('Режим проверки типов',{exact:true}).selectOption('deny');
    await panel.getByLabel('MIME-типы, по одному на строку',{exact:true}).fill('application/x-ux-test\n');
    await nav.getByRole('button',{name:'Почта',exact:true}).click();
    await page.getByLabel('TLS',{exact:true}).selectOption('smtps');await expect(page.getByLabel('Порт',{exact:true})).toHaveValue('465');
    await nav.getByRole('button',{name:'Параметры системы',exact:true}).click();await expect(panel.getByLabel('Максимум файлов в передаче',{exact:true})).toHaveValue('9000');
    await panel.getByRole('button',{name:'Сохранить и проверить',exact:true}).click();
    await expect(panel.getByRole('status')).toContainText('Изменения ещё не применены');expect((await getActive()).revision).toBe(original.revision);
    // Concurrent apply must not be silently overwritten by this older editor.
    await applyValue({...original.value,maxFiles:9500});
    await panel.getByRole('button',{name:/^Применить настройки/}).click();await page.getByRole('dialog').getByRole('button',{name:'Применить',exact:true}).click();
    await expect(panel.getByRole('alert')).toContainText('другой сессией');await expect(panel.getByLabel('Максимум файлов в передаче',{exact:true})).toHaveValue('9000');
    expect((await getActive()).value.maxFiles).toBe(9500);
    await panel.getByRole('button',{name:'Загрузить текущие',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'Загрузить',exact:true}).click();
    await expect(panel.getByLabel('Максимум файлов в передаче',{exact:true})).toHaveValue('9500');
    await panel.getByLabel('Максимум файлов в передаче',{exact:true}).fill('9001');await panel.getByRole('button',{name:'Сохранить и проверить',exact:true}).click();
    await panel.getByRole('button',{name:/^Применить настройки/}).click();await expect(page.getByRole('dialog')).not.toContainText('Доступ будет закрыт');await page.getByRole('dialog').getByRole('button',{name:'Применить',exact:true}).click();
    await expect(panel.getByRole('status')).toContainText('Настройки применены');
    expect((await getActive()).value).toMatchObject({...original.value,maxFiles:9001});
    await panel.getByRole('button',{name:'Экспертный JSON',exact:true}).click();await panel.getByLabel('Settings JSON',{exact:true}).fill('{invalid');
    await panel.getByRole('button',{name:'Обычная форма',exact:true}).click();await expect(panel.getByRole('alert')).toContainText('Исправьте JSON');await expect(panel.getByLabel('Settings JSON',{exact:true})).toHaveValue('{invalid');
    await panel.getByLabel('Settings JSON',{exact:true}).fill(JSON.stringify((await getActive()).value));await panel.getByRole('button',{name:'Обычная форма',exact:true}).click();
    for(const width of [1280,390,320]) {await page.setViewportSize({width,height:900});await a11y(page);}
    await page.screenshot({path:evidencePath('ux-admin-settings-mobile.png'),fullPage:true});
    // Publication controls use the active installation policy, including sub-hour limits.
    await applyValue({...original.value,transferMaxTtlSeconds:90});
    await page.getByRole('button',{name:'Передачи',exact:true}).click();
    const limited = await createDraft(page,'UX-policy-limit');
    expect((await (await page.request.get(`/api/v1/transfers/${limited}/editor`)).json()).maxTtlSeconds).toBe(90);
    await expect(page.getByLabel('Срок доступа, секунд',{exact:true})).toHaveValue('90');
    await expect(page.getByRole('slider')).toHaveAttribute('max','90');
    await expect(page.getByRole('button',{name:'1 день',exact:true})).toHaveCount(0);
    await page.locator('input[type=file]:not([webkitdirectory])').setInputFiles({name:'policy.txt',mimeType:'text/plain',buffer:Buffer.from('policy fixture\n')});
    const review = page.getByRole('button',{name:'Проверить перед публикацией',exact:true});
    await expect(review).toBeEnabled();
    await page.getByLabel('Срок доступа, секунд',{exact:true}).fill('91');await expect(review).toBeDisabled();
    await page.getByRole('button',{name:/^Максимум/}).click();await expect(review).toBeEnabled();
    await review.click();
    const published=page.waitForResponse(r=>r.url().endsWith(`/transfers/${limited}/publish`) && r.request().method()==='POST');
    await page.getByRole('button',{name:'Опубликовать передачу',exact:true}).click();
    const response=await published;expect(response.status()).toBe(200);expect(response.request().postDataJSON().ttlSeconds).toBe(90);
    expect((await contracts.finish()).failures).toEqual([]);
  } finally {await applyValue(original.value);}
});
