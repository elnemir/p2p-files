// Capture real screens and recorded interactions; fixtures never leave the local stack.
import { chromium, expect } from '@playwright/test';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { execFile, spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { inflateRawSync } from 'node:zlib';
import { promisify } from 'node:util';
import path from 'node:path';

const command = promisify(execFile);
const root = process.cwd();
const raw = path.join(root, '.cache/tour/raw');
const output = raw; // Publish only a complete capture during rendering.
const baseURL = 'http://localhost:8090';
const password = process.env.FILEX_TOUR_PASSWORD;
const project = process.env.FILEX_TOUR_PROJECT;
if (!/^filex-tour-[a-f0-9]{10}$/.test(project || '') || !password || !process.env.FILEX_BROWSER_WS) {
  throw new Error('Only the disposable tour harness may run this scenario');
}
await mkdir(path.join(output, 'screenshots'), { recursive: true });
const browser = await chromium.connect(process.env.FILEX_BROWSER_WS, { exposeNetwork: '<loopback>' });
const manifest = { title: 'FileX — экскурсия по системе', recordedAt: new Date().toISOString(),
  environment: 'Изолированный локальный Compose; синтетические данные; Chromium Linux',
  baseCommit: (await command('git', ['rev-parse', 'HEAD'])).stdout.trim(),
  viewport: { width: 1440, height: 960 }, screens: [], chapters: [], assertions: [] };
const applicationFiles = (await command('git', ['ls-files', '--cached', '--others', '--exclude-standard', 'apps', 'cmd', 'internal', 'migrations', 'deploy', 'go.mod', 'go.sum', 'package.json', 'package-lock.json'])).stdout.trim().split('\n');
const applicationHash = createHash('sha256');
for (const name of [...new Set(applicationFiles)].sort()) applicationHash.update(name + '\0').update(await readFile(name)).update('\0');
manifest.applicationSourceSHA256 = applicationHash.digest('hex');
manifest.capturePlanSHA256 = createHash('sha256').update(await readFile('scripts/tour/plan.json')).digest('hex');
manifest.applicationSourceScope = 'Sorted unique git ls-files --cached --others --exclude-standard apps cmd internal migrations deploy go.mod go.sum package.json package-lock.json; path+NUL+bytes+NUL';
const sessions = [];
const errors = [];
let current;
let identityFixture;
let fixtureRead;
let provider;
const button = (p, name) => p.getByRole('button', { name, exact: true });
const heading = (p, name) => p.getByRole('heading', { name, exact: true });
const pause = (p, ms = 650) => p.waitForTimeout(ms);
async function focus(p, locator) {
  await locator.evaluate(el => window.scrollTo({ top: window.scrollY + el.getBoundingClientRect().top - 30, behavior: 'instant' }));
  await pause(p);
}
async function top(p) { await p.evaluate(() => window.scrollTo(0, 0)); await pause(p); }
async function session(name, viewport = manifest.viewport) {
  const context = await browser.newContext({ baseURL, acceptDownloads: true, viewport,
    locale: 'ru-RU', timezoneId: 'Europe/Moscow', colorScheme: 'light', reducedMotion: 'reduce',
    recordVideo: { dir: raw, size: manifest.viewport } });
  const page = await context.newPage();
  page.setDefaultTimeout(25000);
  page.on('pageerror', e => errors.push(`${name}: ${e.message}`));
  const value = { name, context, page, zero: Date.now() };
  sessions.push(value);
  return value;
}
async function chapter(s, title, narration, fn) {
  current = s;
  console.log(`CHAPTER ${manifest.chapters.length + 1}: ${title}`);
  const start = (Date.now() - s.zero) / 1000;
  const first = manifest.screens.length;
  await fn(s.page);
  await pause(s.page, 2200);
  manifest.chapters.push({ number: manifest.chapters.length + 1, title, narration,
    video: `${s.name}.webm`, start, end: (Date.now() - s.zero) / 1000,
    screens: manifest.screens.slice(first).map(v => v.id) });
  await writeFile(path.join(raw, 'capture.json'), JSON.stringify(manifest, null, 2));
}
async function shot(p, slug, title, description, group, options = {}) {
  await p.evaluate(() => document.fonts.ready);
  await pause(p, 800);
  const id = String(manifest.screens.length + 1).padStart(2, '0') + '-' + slug;
  const file = `screenshots/${id}.png`;
  await p.screenshot({ path: path.join(output, file), fullPage: options.fullPage || false, animations: 'disabled' });
  expect(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  manifest.screens.push({ id, title, description, group, file, viewport: p.viewportSize(), fullPage: options.fullPage || false });
  console.log(`SCREEN ${id}`);
}
async function login(p, email) {
  await p.goto('/');
  await p.getByLabel('Email', { exact: true }).fill(email);
  await p.getByLabel('Пароль', { exact: true }).fill(password);
  await button(p, 'Войти').click();
  await expect(button(p, email === process.env.FILEX_TOUR_ADMIN ? 'Администрирование' : 'Новая передача')).toBeVisible();
}
async function tab(p, name) { await button(p, name).click(); await top(p); }
async function draft(p, title) {
  await tab(p, 'Передачи');
  if (await button(p, 'Все передачи').isVisible()) await button(p, 'Все передачи').click();
  await button(p, 'Новая передача').click();
  await p.getByLabel('Название', { exact: true }).fill(title);
  await button(p, 'Сохранить название').click();
  await expect(heading(p, title)).toBeVisible();
}
async function upload(p, files) {
  await p.locator('input[type=file]:not([webkitdirectory])').setInputFiles(files);
  await expect(button(p, 'Проверить перед публикацией')).toBeEnabled({ timeout: 45000 });
}
async function download(p, locator, filename) {
  const ready = p.waitForEvent('download');
  await locator.click();
  const file = await ready;
  await file.saveAs(path.join(raw, filename));
  expect(await file.failure()).toBeNull();
  manifest.assertions.push(`Файл ${filename} получен без ошибки браузера`);
}
const textFile = (name, text) => ({ name, mimeType: 'text/plain', buffer: Buffer.from(text) });

try {
  const owner = await session('owner');
  let recipientLink;
  let verificationLink;
  let transferId;
  await chapter(owner, 'Вход и личный кабинет', 'Это FileX: приватная передача файлов с одноразовой серверной выдачей. Перед нами настоящий локальный стенд с вымышленными данными. У владельца есть передачи, отчёты, фоновые задачи, уведомления и настройки. Начнём с создания пакета для клиента.', async p => {
    await p.goto('/');
    await shot(p, 'login', 'Вход в FileX', 'Пользователь входит по email и паролю. Публичные помощь и статус доступны из верхнего меню. Для тура созданы отдельные демонстрационные учётные записи.', 'Начало');
    await login(p, process.env.FILEX_TOUR_OWNER);
    await shot(p, 'empty', 'Первый вход', 'Пустой кабинет объясняет следующий шаг: создать черновик и добавить файлы или папку. Основные разделы всегда доступны в навигации.', 'Начало');
  });
  await chapter(owner, 'Черновик и загрузка', 'Создаём передачу «Айдентика Север: материалы для клиента». Пока это черновик, можно менять название, состав и пути файлов. Загружаем три демонстрационных документа. При обрыве повторный выбор тех же файлов продолжит загрузку.', async p => {
    await draft(p, 'Айдентика «Север» · материалы для клиента');
    await shot(p, 'draft', 'Черновик передачи', 'В черновике редактируется название, работает выбор файлов и каталогов, drag-and-drop и возобновление загрузки. После публикации состав и пути фиксируются.', 'Владелец');
    await p.getByText('Загрузить в определённую папку', { exact: true }).click();
    await p.getByLabel('Папка назначения (необязательно)').fill('Север/Документы');
    await upload(p, [
      textFile('Начните здесь.txt', 'Демонстрационные материалы студии «Север».\nВсе имена и содержимое вымышлены.\n1. Прочитайте концепцию.\n2. Согласуйте этапы.\n'),
      textFile('Концепция бренда.md', '# Север\n\nЧёткая типографика, спокойная палитра и внимание к деталям.\n\nЭто синтетический пример для локального тура FileX.\n'),
      { name: 'План проекта.csv', mimeType: 'text/csv', buffer: Buffer.from('Этап;Статус\nКонцепция;Согласована\nМатериалы;Переданы\nОбратная связь;Ожидается\n') },
    ]);
    await focus(p, p.getByLabel('Поиск файлов', { exact: true }));
    await shot(p, 'files', 'Файлы и структура папок', 'Пути, размеры и статус «Готово» видны до публикации. Поиск, сортировка и дерево папок помогают проверить состав. Для каждого файла доступна контрольная сумма SHA-256.', 'Владелец');
    await p.getByRole('checkbox', { name: 'Выбрать Север/Документы/План проекта.csv', exact: true }).check();
    await p.getByLabel('Переместить в папку', { exact: true }).fill('Север/Планирование');
    await shot(p, 'organize', 'Управление составом', 'Выбранные файлы можно переместить вместе. Удаление имеет отдельный предпросмотр. Назначение папки явно задаётся полным относительным путём.', 'Владелец');
    await button(p, 'Переместить').click();
    await expect(p.getByText('Север/Планирование/План проекта.csv', { exact: true }).first()).toBeVisible();
    const transfers = await (await p.request.get('/api/v1/transfers')).json();
    transferId = transfers.items.find(v => v.title.startsWith('Айдентика')).id;
  });
  await chapter(owner, 'WebDAV: папка на компьютере', 'Для загрузки из файлового менеджера откройте раздел WebDAV. Здесь есть инструкции для Finder, Проводника Windows и GNOME Files. Подтвердите пароль аккаунта и получите отдельный доступ к черновику. До публикации файлы можно читать, заменять и переименовывать. Неудачная замена сохраняет прежний файл. После копирования обновите список в FileX. Перед публикацией закройте файлы и отключите сетевую папку. Проверка на настоящем Windows-стенде ещё ожидается.', async p => {
    const panel = p.locator('details').filter({ has: p.getByText('Загрузка через WebDAV', { exact: true }) });
    await p.getByText('Загрузка через WebDAV', { exact: true }).click();
    await focus(p, panel);
    for (const [os, title] of [['macos', 'macOS · Finder'], ['windows', 'Windows · Проводник'], ['linux', 'Linux · GNOME Files']]) {
      await p.getByLabel('Ваш файловый менеджер', { exact: true }).selectOption(os);
      await shot(p, `webdav-${os}`, title, 'Инструкция подключения одного черновика отдельным паролем. Windows ожидает реального стенда; эта страница показывает настройку, а не изображает проверенный Windows-клиент.', 'WebDAV');
    }
    await p.getByLabel('Пароль аккаунта для выдачи доступа', { exact: true }).fill(password);
    await button(p, 'Создать доступ WebDAV').click();
    const url = await p.getByLabel('Адрес WebDAV', { exact: true }).inputValue();
    const secret = await p.getByLabel('Пароль подключения WebDAV', { exact: true }).inputValue();
    const headers = { Authorization: `Basic ${Buffer.from(`filex:${secret}`).toString('base64')}` };
    const target = url + encodeURIComponent('Из файлового менеджера.txt');
    const response = await p.request.put(target, { headers, data: Buffer.from('Синтетический пример WebDAV для тура FileX.\n') });
    expect(response.status()).toBe(201);
    await button(p, 'Обновить файлы после WebDAV').click();
    await expect(p.getByText('Из файлового менеджера.txt', { exact: true }).first()).toBeVisible();
    await focus(p, p.getByLabel('Поиск файлов', { exact: true }));
    await shot(p, 'webdav-upload', 'Файл из WebDAV в черновике', 'Настоящий PUT прошёл шифрование и серверную проверку; файл появился в общем дереве. Для основного сценария три исходных документа сохранены, дополнительный пример удалён перед публикацией.', 'WebDAV');
    expect((await p.request.delete(target, { headers })).status()).toBe(204);
    await focus(p, panel);
    await button(p, 'Отозвать доступ WebDAV').click();
    await button(p, 'Обновить файлы после WebDAV').click();
    await p.getByText('Загрузка через WebDAV', { exact: true }).click();
    manifest.assertions.push('WebDAV: scoped credential, real PUT, ready list, DELETE and revocation; password masked throughout recording');
  });
  await chapter(owner, 'Проверка и публикация', 'Перед публикацией выбираем срок доступа и защиту ссылки. В этом примере действует ПИН-код и срок сорок восемь часов. Экран проверки предупреждает: набор станет неизменяемым, а успешная серверная выдача расходует доступ. Публикация сама по себе не подтверждает доставку письма.', async p => {
    await focus(p, heading(p, 'Доступ к передаче'));
    await p.getByLabel('PIN или пароль для ссылок (необязательно)').fill('482619');
    await button(p, '3 дня').click();
    await p.getByRole('slider').focus();
    await p.getByRole('slider').press('ArrowRight');
    await expect(p.getByLabel('Срок доступа, часов')).toHaveValue('73');
    await p.getByLabel('Срок доступа, часов').fill('48');
    await shot(p, 'access', 'Настройки доступа', 'Общая ссылка включена; доступ ограничен 48 часами и PIN. Персональные email-ссылки поддерживаются, но в этом локальном сценарии адресаты не добавляются и SMTP не используется.', 'Владелец');
    await button(p, 'Проверить перед публикацией').click();
    await focus(p, p.getByRole('region', { name: 'Проверка публикации' }));
    await shot(p, 'publish-review', 'Последняя проверка', 'Перед подтверждением видны число файлов, общий объём, срок и необратимые последствия публикации. Эта проверка выполнена на реальном черновике.', 'Владелец');
    await button(p, 'Опубликовать передачу').click();
    await expect(heading(p, 'Ссылки получателей')).toBeVisible();
    recipientLink = await p.getByLabel('Ссылка получателя', { exact: true }).inputValue();
    await focus(p, heading(p, 'Ссылки получателей'));
    await shot(p, 'published', 'Опубликованная передача', 'Владелец получает ссылку и варианты QR PNG/SVG, может скачать все QR, отозвать ссылку или сменить её пароль. Токен в кадре относится к удаляемому демонстрационному стенду.', 'Владелец');
    await download(p, button(p, 'Скачать все QR'), 'qr-links.zip');
  });
  const recipient = await session('recipient');
  await chapter(recipient, 'Получатель: защищённый доступ', 'Получателю не нужна учётная запись владельца. Он открывает ссылку и вводит ПИН-код. Просмотр списка не расходует доступ. Файлы можно получить отдельно или одним архивом; перед скачиванием видны пути, размеры и состояния.', async p => {
    await p.goto(recipientLink);
    await shot(p, 'recipient-pin', 'Вход получателя', 'Ссылка передаёт ключ в fragment браузера; поле скрыто. Если владелец установил PIN или пароль, получатель вводит его отдельно.', 'Получатель');
    await p.getByLabel('PIN или пароль (если задан)').fill('482619');
    await button(p, 'Открыть файлы').click();
    await expect(button(p, 'Скачать всё ZIP')).toBeVisible();
    await shot(p, 'recipient-files', 'Доступные файлы', 'Реальные три файла из опубликованной передачи доступны получателю. Открытие страницы и просмотр списка не равны скачиванию.', 'Получатель');
    await p.setViewportSize({ width: 390, height: 844 });
    await shot(p, 'recipient-mobile', 'Получение с телефона', 'Тот же сценарий в viewport 390 × 844: вертикальный список и действия скачивания. Это эмуляция размера в Chromium, не испытание на физическом телефоне.', 'Получатель', { fullPage: true });
    await p.setViewportSize(manifest.viewport);
  });
  await chapter(recipient, 'ZIP и завершённая выдача', 'Нажимаем «Скачать всё ZIP». Браузер сохраняет архив, а интерфейс сообщает о завершении доступа и убирает кнопки скачивания. Повторный вход по прежней ссылке отклоняется даже с правильным ПИН-кодом. Сервер подтверждает передачу всех байтов; сохранение файла на своём диске получатель проверяет самостоятельно.', async p => {
    await download(p, button(p, 'Скачать всё ZIP'), 'transfer.zip');
    await expect.poll(async () => (await (await owner.page.request.get(`/api/v1/transfers/${transferId}`)).json()).state,
      { timeout: 45000 }).toBe('completed');
    expect((await p.request.get('/api/v1/recipient/files')).status()).toBe(401);
    if (await button(p, 'Обновить список').isVisible()) await button(p, 'Обновить список').click();
    await expect(heading(p, 'Доступ к файлам завершён')).toBeVisible();
    await expect(button(p, 'Скачать всё ZIP')).toHaveCount(0);
    await shot(p, 'recipient-completed', 'Выдача завершена', 'После настоящего ZIP интерфейс очищает файлы и кнопки скачивания. Повторная выдача запрещена сервером; причины других отказов не раскрываются.', 'Получатель');
    // Adding a fragment to the current URL is a same-document navigation.
    await p.goto('about:blank');
    await p.goto(recipientLink);
    await p.getByLabel('PIN или пароль (если задан)').fill('482619');
    await button(p, 'Открыть файлы').click();
    await expect(p.locator('.error[role="alert"]')).toContainText('Войдите снова или проверьте ссылку и пароль.');
    await shot(p, 'downloaded', 'Доступ после завершения передачи', 'После выдачи ZIP передача перешла в completed. Повторное открытие той же ссылки с верным PIN отклоняется. Общая формулировка ошибки не раскрывает причину отказа; это ожидаемый результат одноразовой выдачи.', 'Получатель');
    manifest.assertions.push('После ZIP transfer=completed, recipient/files=401; повторный вход по той же ссылке отклонён');
  });
  await chapter(owner, 'Журнал и отчёт', 'Владелец видит общий результат скачивания без идентификации получателей. Отчёт создаёт отдельную ревизию состояния, которую worker формирует в PDF, JSON и CSV. Контрольная сумма связывает ревизию с её содержимым.', async p => {
    await button(p, 'Создать отчёт').click();
    await expect(p.locator('.list-row').filter({ hasText: 'Пользовательский' })).toContainText('Готово', { timeout: 45000 });
    await p.getByRole('button', { name: /Пользовательский.*#1/ }).click();
    await expect(button(p, 'PDF')).toBeVisible();
    await top(p);
    await shot(p, 'reports', 'Ревизия отчёта', 'Отчёт фиксирует состояние на момент создания. Здесь доступны готовые PDF/JSON/CSV и SHA-256; создание новой ревизии не переписывает прежнюю.', 'Отчёты и задачи');
    await download(p, button(p, 'PDF'), 'report.pdf');
    await download(p, button(p, 'JSON'), 'report.json');
    verificationLink = await p.getByRole('link', { name: 'Проверить', exact: true }).getAttribute('href');
    await tab(p, 'Передачи');
    await expect(heading(p, 'Журнал скачиваний')).toBeVisible();
    await focus(p, heading(p, 'Журнал скачиваний'));
    await shot(p, 'journal', 'Журнал владельца', 'Общий результат: время, число файлов, объём и состояние выдачи. Интерфейс прямо указывает, что получатели не идентифицируются в пользовательском журнале.', 'Отчёты и задачи');
  });
  await chapter(owner, 'История отчётов и корзина', 'Отчёт можно временно убрать в корзину и восстановить. Его история сохраняет оба действия, а контрольная сумма остаётся прежней. Удаление содержимого навсегда — отдельная операция. В нашем примере выполняем только обратимый цикл и проверяем восстановленный отчёт.', async p => {
    await tab(p, 'Отчёты');
    const row = p.locator('.list-row').filter({ hasText: 'Пользовательский' });
    await button(row, 'В корзину').click();
    await expect(button(row, 'Восстановить')).toBeVisible();
    await top(p);
    await shot(p, 'report-trash', 'Корзина отчётов', 'Настоящая ревизия перемещена в корзину. Доступно восстановление; необратимое удаление отдельно раскрывается и в туре не выполняется.', 'Отчёты и задачи');
    await button(row, 'Восстановить').click();
    await expect(button(row, 'В корзину')).toBeVisible();
    await row.getByRole('button', { name: /Пользовательский.*#1/ }).click();
    await focus(p, heading(p, 'Ревизия отчёта'));
    await shot(p, 'report-history', 'История восстановления', 'История ревизии содержит перемещение в корзину и восстановление. Следующий шаг сравнит исходный JSON с публичной контрольной суммой.', 'Отчёты и задачи');
    manifest.assertions.push('Report trashed/restored through UI; original JSON verified after restoration');
  });
  const publicPage = await session('public');
  await chapter(publicPage, 'Публичная проверка отчёта', 'Проверку отчёта можно открыть без входа. Сначала получаем публичные данные по идентификатору, затем выбираем скачанный JSON. Браузер вычисляет его контрольную сумму. В нашем примере она совпадает; содержимое не требуется отправлять для локального сравнения.', async p => {
    await p.goto(verificationLink);
    await button(p, 'Проверить').click();
    await p.getByLabel('JSON отчёта для сравнения').setInputFiles(path.join(raw, 'report.json'));
    await expect(p.getByText('Контрольная сумма совпадает', { exact: true })).toBeVisible();
    await shot(p, 'verify', 'Проверка подлинности', 'JSON скачан из настоящей ревизии. Локальный SHA-256 совпал с публичной контрольной суммой. Это проверка данного артефакта, не самостоятельный внешний аудит системы.', 'Отчёты и задачи');
    manifest.assertions.push('SHA-256 скачанного JSON совпал в публичном UI');
  });
  await chapter(owner, 'Поиск и сохранённые виды', 'Создадим ещё один черновик: так видны разные состояния в рабочем списке. Передачи можно искать по названию и фильтровать по состоянию. Сохранённый вид возвращает к выбранным условиям; массовые действия начинаются с явного выбора строк.', async p => {
    await draft(p, 'Каталог осень · черновик на согласование');
    await button(p, 'Все передачи').click();
    await p.getByLabel('Поиск по названию', { exact: true }).fill('Север');
    await button(p, 'Найти').click();
    await button(p, 'Сохранить вид').click();
    await shot(p, 'search', 'Поиск и сохранённый вид', 'Условия поиска по названию можно сохранить и открыть повторно. Статусы отличают черновики от опубликованных и завершённых передач.', 'Владелец');
    await p.getByLabel('Поиск по названию', { exact: true }).fill('');
    await button(p, 'Найти').click();
    await p.getByRole('checkbox', { name: 'Выбрать передачу Айдентика «Север» · материалы для клиента', exact: true }).check();
    await shot(p, 'selection', 'Список и массовые действия', 'Выбор строк открывает предпросмотр отзыва. Реальный отзыв в этом кадре не выполняется: перед запуском нужно проверить состав операции.', 'Владелец');
  });
  await chapter(owner, 'Экспорт и фоновые задачи', 'Экспорт запускается в фоне и фиксирует данные на начало задачи. Создадим CSV и JSON. После завершения файлы доступны для скачивания, а счётчики позволяют проверить объём обработки. Длинные операции не блокируют работу кабинета.', async p => {
    await tab(p, 'Задачи');
    await button(p, '+ CSV').click();
    await expect(p.locator('.list-row').filter({ hasText: 'CSV' })).toContainText('Завершено', { timeout: 45000 });
    await button(p, '+ JSON').click();
    await expect(p.locator('.list-row').filter({ hasText: 'JSON' })).toContainText('Завершено', { timeout: 45000 });
    await download(p, p.locator('.list-row').filter({ hasText: 'CSV' }).getByRole('button', { name: 'Скачать', exact: true }), 'export.csv');
    await top(p);
    await shot(p, 'exports', 'Готовые экспорты', 'CSV и JSON сформированы worker, CSV действительно скачан. Экспорты отражают snapshot на начало задачи; это отдельные артефакты со сроком хранения.', 'Отчёты и задачи');
    await p.setViewportSize({ width: 390, height: 844 });
    await shot(p, 'exports-mobile', 'Задачи на мобильном экране', 'Статусы и счётчики доступны в узком viewport без горизонтального переполнения.', 'Отчёты и задачи', { fullPage: true });
    await p.setViewportSize(manifest.viewport);
  });
  await chapter(owner, 'Уведомления и предпочтения', 'События собраны отдельно от файлов. Пользователь выбирает язык, часовой пояс, письма и тихие часы. Внешняя доставка требует настроенного почтового сервиса: данный стенд показывает настройки, но не доказывает доставку email.', async p => {
    await tab(p, 'Уведомления');
    await shot(p, 'notifications', 'Лента событий', 'Уведомления относятся к состояниям передач и задач. Внешние письма не отправляются: в демонстрационном окружении SMTP не настроен.', 'Настройки');
    await tab(p, 'Настройки');
    await p.getByLabel('Часовой пояс IANA', { exact: true }).selectOption('Europe/Moscow');
    await p.getByLabel('Включить тихие часы', { exact: true }).check();
    await p.getByLabel('Тихие часы: с', { exact: true }).selectOption('23');
    await p.getByLabel('Тихие часы: до', { exact: true }).selectOption('7');
    const saved = p.waitForResponse(r => r.url().endsWith('/notification-preferences') && r.request().method() === 'PUT');
    await button(p, 'Сохранить').click();
    expect((await saved).status()).toBe(204);
    await expect(p.getByRole('status')).toContainText('Настройки сохранены');
    expect(await (await p.request.get('/api/v1/notification-preferences')).json()).toMatchObject({ timezone: 'Europe/Moscow', quietStart: 23, quietEnd: 7 });
    await top(p);
    await shot(p, 'preferences', 'Личные настройки', 'Локаль, часовой пояс и каналы уведомлений задаются в кабинете. Тихие часы и дайджест регулируют email-уведомления после подключения SMTP.', 'Настройки');
  });
  await chapter(owner, 'Безопасность учётной записи', 'Чувствительные действия требуют свежего подтверждения пароля. В настройках видны сеансы и история входов. API-токену можно выдать только нужные права и ограниченный срок; секрет показывается один раз. В туре новый токен не создаём.', async p => {
    await button(p, 'Аккаунт и безопасность').click();
    await focus(p, heading(p, 'Подтверждение личности'));
    await shot(p, 'account', 'Пароль и сеансы', 'Подтверждение личности действует пять минут. Смена пароля отзывает сеансы и API-токены. Активные сеансы и история входов доступны ниже.', 'Настройки');
    await focus(p, heading(p, 'API-токен'));
    await shot(p, 'tokens', 'API-токены с ограничениями', 'Форма позволяет выбрать scopes и срок от 1 до 30 дней. Секреты не создавались и не включены в тур; на экране показана конфигурация.', 'Настройки');
    await button(p, 'Интеграции').click();
    await focus(p, heading(p, 'Webhook'));
    await shot(p, 'webhooks', 'Webhook-интеграция', 'Для webhook задаётся HTTPS URL. Реальный endpoint не подключён; экран показывает форму, а не подтверждение внешней доставки.', 'Настройки');
  });
  await chapter(owner, 'Тёмная тема и мобильный интерфейс', 'Интерфейс поддерживает светлую, тёмную и системную темы. Посмотрим настройки на узком экране. Это та же рабочая сессия и те же данные; проверка размера выполнена в браузере, без физического устройства.', async p => {
    await button(p, 'Уведомления и язык').click();
    await p.getByLabel('Тема', { exact: true }).selectOption('dark');
    await top(p);
    await shot(p, 'dark', 'Тёмная тема', 'Тема переключена штатным селектором. Текст и формы сняты с реального UI без ретуши.', 'Настройки');
    await p.setViewportSize({ width: 390, height: 844 });
    await shot(p, 'settings-mobile', 'Настройки на телефоне', 'Полная страница личных настроек в тёмной теме, 390 × 844: часовой пояс и тихие часы сохранены на сервере.', 'Настройки', { fullPage: true });
    await p.setViewportSize(manifest.viewport);
    await p.getByLabel('Тема', { exact: true }).selectOption('light');
  });
  const admin = await session('admin');
  const ops = admin.page.getByRole('region', { name: 'Эксплуатация и восстановление', exact: true });
  await chapter(admin, 'Панель администратора', 'У администратора появляется отдельный раздел эксплуатации. Здесь закрывают доступ для обслуживания, создают резервные копии и запускают проверки. Копия включает базу и зашифрованные настройки; пользовательские файлы и артефакты требуют отдельного хранения.', async p => {
    await login(p, process.env.FILEX_TOUR_ADMIN);
    await tab(p, 'Администрирование');
    await expect(heading(p, 'Эксплуатация и восстановление')).toBeVisible();
    await shot(p, 'admin', 'Эксплуатация системы', 'Доступ открыт. Действия эксплуатации требуют причины и свежей reauthentication. Backup базы не является копией пользовательских payload и артефактов.', 'Администрирование');
    await ops.getByLabel('Причина эксплуатационного действия').fill('Демонстрационный backup перед учебным восстановлением');
    await button(ops, 'Создать резервную копию').click();
    await expect(ops.getByLabel('Копия для восстановления').locator('option')).toHaveCount(2, { timeout: 45000 });
    await ops.getByLabel('Копия для восстановления').selectOption({ index: 1 });
    await focus(p, heading(p, 'Эксплуатация и восстановление'));
    await shot(p, 'backup', 'Готовая резервная копия', 'Настоящее задание backup завершено на отдельном Local BackupProvider. Копия доступна для выбора; реальные внешние backup storage и escrow этим не проверены.', 'Администрирование');
  });
  await chapter(admin, 'Восстановление: проверка последствий', 'Выбираем готовую копию и открываем проверку восстановления. Приложение предупреждает о замене базы, отзыве сеансов и возврате ссылок к состоянию копии. Перед восстановлением создаётся проверенная предварительная копия. Всё выполняем только внутри одноразового стенда.', async p => {
    await button(ops, 'Подготовить восстановление').click();
    await focus(p, ops.getByRole('region', { name: 'Проверка восстановления' }));
    await shot(p, 'restore-review', 'Предпросмотр восстановления', 'Показаны выбранный backup, SHA-256 и последствия. В следующем шаге сценарий действительно запускает pre-restore backup и восстановление демонстрационной БД.', 'Восстановление');
    await button(ops, 'Создать предварительную копию и восстановить').click();
    await expect.poll(async () => {
      const r = await p.request.get('/api/v1/auth/me');
      return (await r.json()).code;
    }, { timeout: 45000 }).toBe('restart_required');
    await pause(p, 4500);
    await focus(p, heading(p, 'Эксплуатация и восстановление'));
    await shot(p, 'restart', 'Восстановление требует перезапуска', 'После восстановления API/worker требуют перезапуска на хосте. Доступ не открывается автоматически; старый сеанс становится недействительным.', 'Восстановление');
  });
  // Host-side control is limited to the exact disposable project; never expose Docker to the browser.
  await command('docker', ['compose', '--env-file', '/dev/null', '--project-name', project,
    '-f', 'deploy/compose.yaml', 'restart', 'api', 'worker'], { timeout: 60000 });
  await expect.poll(async () => { try { return (await admin.page.request.get('/api/v1/auth/me')).status(); } catch { return 0; } }, { timeout: 45000 }).toBe(401);
  await chapter(admin, 'Сверка и явное открытие доступа', 'После перезапуска входим снова и выполняем сверку восстановленных данных. Пока администратор не проверил результат и явно не подтвердил открытие, выдача остаётся закрытой. У нас сверка завершена, затем доступ успешно открыт.', async p => {
    await login(p, process.env.FILEX_TOUR_ADMIN);
    await tab(p, 'Администрирование');
    await expect(ops.getByText('Ожидает проверки и открытия', { exact: true })).toBeVisible();
    await ops.getByLabel('Причина эксплуатационного действия').fill('Проверка ссылок, токенов и объектов демонстрационного стенда');
    await button(ops, 'Сверить восстановленные данные').click();
    await expect(heading(ops, 'Сверка завершена')).toBeVisible({ timeout: 45000 });
    await expect(button(ops, 'Открыть доступ после проверки')).toBeDisabled();
    await focus(p, heading(ops, 'Сверка завершена'));
    await shot(p, 'reconcile', 'Результат сверки', 'Сверка посчитала ссылки, файлы и объекты. Открытие всё ещё заблокировано, пока оператор не подтвердит проверку восстановленных ссылок, токенов и несоответствий.', 'Восстановление');
    await ops.getByRole('checkbox', { name: 'Я проверил восстановленные ссылки и токены, недоступные файлы и объекты без записей в БД.' }).check();
    await button(ops, 'Открыть доступ после проверки').click();
    await expect(ops.getByText('Доступ открыт', { exact: true })).toBeVisible();
    await focus(p, heading(p, 'Эксплуатация и восстановление'));
    await shot(p, 'reopened', 'Доступ снова открыт', 'Полный цикл backup → restore → restart → login → reconcile → reopen выполнен в локальном Compose. Это демонстрация recovery flow, не целевая DR-приёмка.', 'Восстановление');
    manifest.assertions.push('Backup/restore/restart/reconcile/reopen завершены; до подтверждения открытие заблокировано');
  });
  await chapter(admin, 'Аудит и доступ к экспортам', 'Контрольную точку аудита можно сохранить для независимого хранения. Администратор получает экспорт пользователя только с причиной и записью в аудит. В туре оба файла действительно скачаны; внешний контур хранения контрольных точек ещё не подключён.', async p => {
    await ops.getByText('Контрольные точки аудита', { exact: true }).click();
    await button(ops, 'Зафиксировать контрольную точку').click();
    await expect(button(ops, 'Сохранить новую контрольную точку')).toBeVisible();
    await focus(p, ops.getByText('Контрольные точки аудита', { exact: true }));
    await shot(p, 'checkpoint', 'Контрольная точка аудита', 'Подписанная контрольная точка создаётся внутри приложения и скачивается. Для независимого доказательства её нужно вынести за пределы VPS; здесь показана локальная часть.', 'Администрирование');
    await download(p, button(ops, 'Сохранить новую контрольную точку'), 'audit-checkpoint.json');
    await button(p, 'Безопасность и аудит').click();
    const exports = p.getByRole('region', { name: 'Экспорты пользователей', exact: true });
    await button(exports, 'Загрузить экспорты').click();
    await exports.getByLabel('Причина доступа к экспорту').fill('Учебная проверка доступа к синтетическому экспорту');
    await expect(button(exports, 'Скачать с записью в аудит').first()).toBeEnabled();
    await focus(p, exports);
    await shot(p, 'admin-exports', 'Аудируемый доступ к экспортам', 'Экспорты владельца доступны через отдельную форму с причиной действия. Административный доступ не превращается в неявный пользовательский доступ.', 'Администрирование');
    await download(p, button(exports, 'Скачать с записью в аудит').first(), 'admin-export.bin');
  });
  await chapter(admin, 'Хранилища, перенос и ключи', 'Для payload, артефактов и резервных копий используются отдельные назначения хранилищ. В нашем стенде это три Local-провайдера. Интерфейс также поддерживает S3, перенос между провайдерами и ротацию классов ключей. Эти экраны показываем без подключения внешнего S3.', async p => {
    await button(p, 'Хранение файлов').click();
    await focus(p, heading(p, 'Хранилища'));
    await shot(p, 'storage', 'Провайдеры хранилищ', 'Три реально созданных Local-провайдера: payload, artifacts и backups. Можно выводить провайдера из новых загрузок; S3 требует отдельной конфигурации и внешней приёмки.', 'Администрирование');
    await focus(p, heading(p, 'Перенос хранилища'));
    await shot(p, 'migration', 'Перенос между хранилищами', 'Перенос выбирает исходный и целевой провайдеры и создаёт фоновую задачу. На этом экране задача не запущена: в каждом назначении стенда только один Local-провайдер.', 'Администрирование');
    await button(p, 'Безопасность и аудит').click();
    await focus(p, heading(p, 'Ротация ключей'));
    await shot(p, 'keys', 'Ключи и удержание', 'Ротация разделена по классам ключей. Ниже расположено удержание объектов с обязательной причиной. Тур не выполняет ненужную ротацию или реальное расследование.', 'Администрирование');
  });
  await chapter(admin, 'Безопасность и почта', 'Управление пользователями, удержание файлов и экспертные операции собраны в разделе безопасности. Чувствительные действия требуют причины и подтверждения личности. В настройках почты выбор защищённого соединения подставляет подходящий порт. Реальные почтовые реквизиты здесь не используются.', async p => {
    await focus(p, heading(p, 'Массовые действия администратора'));
    await shot(p, 'admin-users', 'Управление пользователями', 'Администратор управляет локальными аккаунтами и массовыми действиями с обязательной причиной. В туре действующие демонстрационные аккаунты не блокируются.', 'Администрирование');
    await focus(p, heading(p, 'Экспертная проверка файлов'));
    await shot(p, 'forensic', 'Экспертные операции', 'Forensic flow имеет собственные условия и аудит; ZIP требует удержания. В туре снята форма без получения пользовательских payload через административный обход.', 'Администрирование');
    await button(p, 'Почта').click();
    await p.getByLabel('TLS', { exact: true }).selectOption('smtps');
    await expect(p.getByLabel('Порт', { exact: true })).toHaveValue('465');
    await focus(p, heading(p, 'SMTP'));
    await shot(p, 'smtp', 'Настройка SMTP', 'Поля SMTP оставлены без рабочих credentials. Пустая форма не подтверждает проверку соединения или доставку письма.', 'Администрирование');
    await button(p, 'Безопасность и аудит').click();
    await focus(p, heading(p, 'Журнал безопасности'));
    await shot(p, 'audit', 'Журнал безопасности', 'Блок запускает проверку целостности журнала безопасности; сами записи здесь не перечислены. Административные действия фиксируются сервером, а контрольные точки скачиваются в разделе эксплуатации.', 'Администрирование');
  });
  await chapter(admin, 'Системные настройки: проверить и применить', 'Лимиты, сроки и правила файлов задаются обычной формой. Для точных значений есть поля и списки, а для специалистов — режим JSON. Сначала сохраняем и проверяем черновик. Он ещё не действует. Отдельное подтверждение применяет ревизию; история позволяет вернуться к прежним настройкам.', async p => {
    await button(p, 'Параметры системы').click();
    const panel = p.getByRole('region', { name: 'Настройки приложения', exact: true });
    const original = (await (await p.request.get('/api/v1/admin/settings')).json()).find(v => v.active);
    await panel.getByLabel('Максимум файлов в передаче', { exact: true }).fill('500');
    await panel.getByLabel('Режим проверки типов', { exact: true }).selectOption('deny');
    await panel.getByLabel('MIME-типы, по одному на строку', { exact: true }).fill('application/x-msdownload');
    await focus(p, panel);
    await shot(p, 'app-settings', 'Лимиты и правила файлов', 'Форма показывает активную ревизию. Меняем лимит на 500 файлов и запрещаем один MIME-тип в учебном окружении. Остальные параметры остаются прежними.', 'Системные настройки');
    await button(panel, 'Сохранить и проверить').click();
    await expect(panel.getByRole('status')).toContainText('Изменения ещё не применены');
    expect((await (await p.request.get('/api/v1/admin/settings')).json()).find(v => v.active).revision).toBe(original.revision);
    await focus(p, panel);
    await shot(p, 'settings-validated', 'Проверенный черновик', 'Черновик прошёл серверную проверку, но активная конфигурация ещё не изменилась. Применение выполняется отдельным действием с проверкой базовой ревизии.', 'Системные настройки');
    await panel.getByRole('button', { name: /^Применить настройки/ }).click();
    await shot(p, 'settings-confirm', 'Подтверждение применения', 'Диалог подтверждает изменение настроек. Сетевую политику этот пример не меняет, поэтому эксплуатационный gate не закрывается.', 'Системные настройки');
    await button(p.getByRole('dialog'), 'Применить').click();
    await expect(panel.getByRole('status')).toContainText('Настройки применены');
    expect((await (await p.request.get('/api/v1/admin/settings')).json()).find(v => v.active).value.maxFiles).toBe(500);
    await panel.getByText('История ревизий и откат', { exact: true }).click();
    await focus(p, panel.getByText('История ревизий и откат', { exact: true }));
    await shot(p, 'settings-history', 'История системных настроек', 'Новая ревизия активна, прежняя остаётся в истории. Сервер подтвердил изменение maxFiles; без явного применения черновик не действует.', 'Системные настройки');
    await button(panel, 'Экспертный JSON').click();
    await focus(p, panel.getByLabel('Settings JSON', { exact: true }));
    await shot(p, 'settings-json', 'Экспертный редактор', 'JSON представляет ту же типизированную конфигурацию. В примере он только прочитан; параллельного источника настроек нет.', 'Системные настройки');
    await button(panel, 'Обычная форма').click();
    await p.setViewportSize({ width: 390, height: 844 });
    await focus(p, panel);
    await shot(p, 'settings-form-mobile', 'Системные параметры на телефоне', 'Обычная форма доступна на узком экране; числовые поля и селекты сохраняют подписи. Это эмуляция viewport.', 'Системные настройки');
    await p.setViewportSize(manifest.viewport);
    manifest.assertions.push('Settings UI draft validated without activation, explicit CAS apply committed maxFiles=500; network settings preserved');
  });
  await chapter(admin, 'Корпоративный вход: OIDC, SAML и LDAP', 'Администратор подключает корпоративные провайдеры. Для OpenID Connect задаются издатель и клиент, для SAML — метаданные и сертификат подписи, для LDAP — каталог и постоянный идентификатор. Первый вход может создавать только пользователя. Роли остаются локальными; совпавший email не связывает аккаунты автоматически.', async p => {
    await button(p, 'Корпоративный вход').click();
    const panel = p.getByRole('region', { name: 'Провайдеры авторизации', exact: true });
    await panel.getByLabel('Название кнопки входа', { exact: true }).fill('Компания · пример конфигурации');
    await panel.getByLabel('Issuer URL (https://)', { exact: true }).fill('https://sso.example.test/realms/company');
    await panel.getByLabel('Client ID', { exact: true }).fill('filex');
    await focus(p, panel);
    await shot(p, 'identity-oidc', 'OpenID Connect', 'Форма подключения Keycloak, Microsoft Entra ID, Google Workspace и других OIDC-провайдеров. Адрес example.test — иллюстрация; соединение и сохранение этого OIDC-примера не выполняются.', 'Корпоративный вход', { fullPage: true });
    await focus(p, panel.getByLabel('Issuer URL (https://)', { exact: true }));
    await shot(p, 'identity-oidc-fields', 'Параметры OIDC крупным планом', 'Issuer, Client ID, email claim и Google Workspace hd domain показаны отдельно. Client secret пуст; значения example.test не являются подключённым IdP.', 'Корпоративный вход');
    await panel.getByLabel('Протокол', { exact: true }).selectOption('ldap');
    await panel.getByLabel('LDAP URL (ldaps:// / ldap:// + StartTLS)', { exact: true }).fill('ldaps://directory.example.test:636');
    await panel.getByLabel('Base DN', { exact: true }).fill('dc=example,dc=test');
    await panel.getByLabel('Атрибут логина (uid / sAMAccountName)', { exact: true }).fill('sAMAccountName');
    await panel.getByLabel('Постоянный идентификатор', { exact: true }).selectOption('objectGUID');
    await focus(p, panel.getByLabel('LDAP URL (ldaps:// / ldap:// + StartTLS)', { exact: true }));
    await shot(p, 'identity-ldap', 'LDAP и Active Directory', 'Показаны LDAPS, Base DN, атрибут логина и стабильный objectGUID. Пароль service bind не введён; доступ к настоящему AD/DC не выполняется.', 'Корпоративный вход');
    identityFixture = spawn('go', ['run', './scripts/identity-fixture'], { env: process.env, stdio: ['pipe', 'pipe', 'pipe'] });
    const lines = createInterface({ input: identityFixture.stdout })[Symbol.asyncIterator]();
    let fixtureError = '';
    identityFixture.stderr.on('data', d => { fixtureError += d.toString(); });
    fixtureRead = async () => { const next = await lines.next(); if (next.done) throw new Error(`Local identity fixture stopped: ${fixtureError}`); return JSON.parse(next.value); };
    const { metadata } = await fixtureRead();
    await panel.getByLabel('Протокол', { exact: true }).selectOption('saml');
    await panel.getByLabel('Название кнопки входа', { exact: true }).fill('Учебный SAML');
    await panel.getByLabel('Metadata XML провайдера (с сертификатом подписи)', { exact: true }).fill(metadata);
    await panel.getByLabel('Разрешить вход', { exact: true }).check();
    await panel.getByLabel('Создавать USER при первом входе', { exact: true }).check();
    await focus(p, panel);
    await shot(p, 'identity-saml', 'SAML 2.0 и регистрация USER', 'Настраивается изолированный signer из browser tests. Metadata содержит только одноразовый публичный сертификат. Включены вход и регистрация USER; настоящего корпоративного tenant здесь нет.', 'Корпоративный вход');
    await button(panel, 'Сохранить провайдера').click();
    await expect(panel.getByRole('status')).toContainText('Провайдер сохранён');
    provider = (await (await p.request.get('/api/v1/admin/identity-providers')).json()).find(v => v.config.name === 'Учебный SAML' && v.config.enabled);
    expect(provider).toBeTruthy();
    await focus(p, panel.getByRole('status'));
    await shot(p, 'identity-saved', 'Провайдер сохранён', 'Сервер сохранил SAML-конфигурацию. Callback/metadata URL доступен для настройки IdP. Изменение провайдера отзывает прежние внешние сессии этого провайдера.', 'Корпоративный вход');
  });
  const corporate = await session('corporate');
  await chapter(corporate, 'Первый вход через корпоративный аккаунт', 'На странице входа появилась корпоративная кнопка. Демонстрационный SAML-провайдер возвращает подписанное подтверждение, и FileX создаёт пользователя с обычными правами. В настройках видна связь с корпоративным аккаунтом. Это реальный локальный протокольный сценарий, а подключение вашего провайдера проверяется отдельно.', async p => {
    let assertion = '', relay = '';
    await p.route('https://identity-fixture.example.test/login?**', async route => {
      const url = new URL(route.request().url());
      relay = url.searchParams.get('RelayState');
      const request = inflateRawSync(Buffer.from(url.searchParams.get('SAMLRequest'), 'base64')).toString();
      const requestID = request.match(/\bID="([^"]+)"/)?.[1];
      expect(requestID).toBeTruthy();
      identityFixture.stdin.write(JSON.stringify({ RequestID: requestID, ACS: `${baseURL}/api/v1/auth/providers/${provider.id}/saml`, EntityID: `${baseURL}/api/v1/auth/providers/${provider.id}/metadata`, Email: 'colleague@example.test', Subject: 'tour-colleague' }) + '\n');
      assertion = (await fixtureRead()).response;
      await route.fulfill({ contentType: 'text/html', body: '<!doctype html><html lang="ru"><head><meta charset="utf-8"><title>Локальный учебный IdP</title></head><body><h1>Локальный учебный IdP</h1><p>Изолированный SAML signer для демонстрации. Это не интерфейс Keycloak, Entra ID или Google.</p></body></html>' });
    });
    await p.goto('/');
    await expect(button(p, 'Войти через Учебный SAML')).toBeVisible();
    await shot(p, 'corporate-login', 'Выбор корпоративного входа', 'Публичная кнопка появилась после настройки ADMIN. Локальный вход по email и паролю остаётся доступным.', 'Корпоративный вход');
    await button(p, 'Войти через Учебный SAML').click();
    await expect(heading(p, 'Локальный учебный IdP')).toBeVisible();
    // Submit a real signed test assertion to the real callback, exactly as the browser fixture does.
    await p.evaluate(({ acs, response, state }) => {
      const form = document.createElement('form'); form.method = 'POST'; form.action = acs;
      for (const [name, value] of Object.entries({ SAMLResponse: response, RelayState: state })) {
        const input = document.createElement('input'); input.type = 'hidden'; input.name = name; input.value = value; form.appendChild(input);
      }
      document.body.appendChild(form); form.submit();
    }, { acs: `${baseURL}/api/v1/auth/providers/${provider.id}/saml`, response: assertion, state: relay });
    await expect(button(p, 'Новая передача')).toBeVisible();
    expect(await (await p.request.get('/api/v1/auth/me')).json()).toMatchObject({ email: 'colleague@example.test', role: 'user' });
    await expect(button(p, 'Администрирование')).toHaveCount(0);
    await shot(p, 'corporate-account', 'Новый корпоративный пользователь', 'FileX проверил подпись и browser-bound flow, создал локальный USER и открыл пустой кабинет. Права ADMIN не выдаются из внешних claims.', 'Корпоративный вход');
    await tab(p, 'Настройки');
    await button(p, 'Аккаунт и безопасность').click();
    await focus(p, heading(p, 'Корпоративные аккаунты'));
    await shot(p, 'corporate-linked', 'Связь с корпоративным аккаунтом', 'Сохранённая identity связана со стабильным subject провайдера. Здесь доступно повторное подтверждение через IdP; автоматического связывания по email нет.', 'Корпоративный вход');
    manifest.assertions.push('Local SAML signed response verified by real API; JIT colleague@example.test role=user, no ADMIN controls, linked identity visible');
  });
  await chapter(admin, 'White labeling: свой бренд без пересборки', 'Оформление всей установки меняется в панели администратора. Зададим название «Север», загрузим логотип и значок вкладки, выберем палитру и двухтоновые иконки. Светлый и тёмный предпросмотры показывают черновик до применения. После сохранения новый бренд появляется в кабинете и на публичных страницах. Исходное оформление можно вернуть отдельным применением.', async p => {
    await p.getByText('Подтвердить личность', { exact: true }).click();
    const reauth = p.locator('.reauth-panel');
    await reauth.getByLabel('Текущий пароль', { exact: true }).fill(password);
    const confirmed = p.waitForResponse(r => r.url().endsWith('/auth/reauth') && r.request().method() === 'POST');
    await button(reauth, 'Подтвердить на 5 минут').click();
    expect((await confirmed).status()).toBe(204);
    await reauth.getByText('Подтвердить личность', { exact: true }).click();
    await button(p, 'Оформление').click();
    const panel = p.getByRole('region', { name: 'Брендинг и оформление', exact: true });
    await panel.getByLabel('Название продукта', { exact: true }).fill('Север');
    await panel.getByLabel('Описание на русском', { exact: true }).fill('Документы вашей команды');
    await panel.getByLabel('Описание на английском', { exact: true }).fill('Documents for your team');
    await panel.getByLabel('Подпись внизу страницы', { exact: true }).fill('Демонстрационный бренд · локальный тур');
    await button(panel, 'Океан').click();
    await panel.getByLabel('Стиль иконок', { exact: true }).selectOption('duotone');
    const images = await p.evaluate(() => {
      const logo = document.createElement('canvas'); logo.width = 240; logo.height = 64;
      const ctx = logo.getContext('2d'); ctx.fillStyle = '#2455a4'; ctx.fillRect(0, 0, 240, 64);
      ctx.fillStyle = '#fff'; ctx.font = 'bold 29px Arial'; ctx.fillText('СЕВЕР', 52, 43);
      ctx.beginPath(); ctx.moveTo(18, 42); ctx.lineTo(30, 19); ctx.lineTo(42, 42); ctx.closePath(); ctx.fill();
      const icon = document.createElement('canvas'); icon.width = icon.height = 64;
      const c = icon.getContext('2d'); c.fillStyle = '#2455a4'; c.fillRect(0, 0, 64, 64);
      c.fillStyle = '#fff'; c.beginPath(); c.moveTo(12, 49); c.lineTo(32, 12); c.lineTo(52, 49); c.closePath(); c.fill();
      return { logo: logo.toDataURL('image/png').split(',')[1], icon: icon.toDataURL('image/png').split(',')[1] };
    });
    for (const label of ['Светлый логотип', 'Тёмный логотип', 'Favicon']) {
      await panel.getByLabel(label, { exact: true }).setInputFiles({ name: 'sever-demo.png', mimeType: 'image/png', buffer: Buffer.from(label === 'Favicon' ? images.icon : images.logo, 'base64') });
    }
    await focus(p, panel);
    await shot(p, 'branding-editor', 'Название, логотип и иконки', 'Учебный бренд «Север»: название, описания, светлый/тёмный логотип, favicon и двухтоновые иконки. Изменения пока видны только в редакторе.', 'Оформление');
    await focus(p, panel.getByText('Цветовые палитры', { exact: true }));
    await shot(p, 'branding-palettes', 'Палитры и точные цвета', 'Готовые палитры и девять цветов каждого режима доступны в одном редакторе. В примере выбрана палитра «Океан»; проверка контраста не отключена.', 'Оформление');
    await focus(p, panel.getByRole('region', { name: 'Светлый предпросмотр', exact: true }));
    await shot(p, 'branding-preview', 'Два предпросмотра', 'Светлая и тёмная палитры проверяются до применения. Цвета проходят серверные ограничения контраста; произвольный CSS или JavaScript не принимаются.', 'Оформление');
    const applied = p.waitForResponse(r => r.url().endsWith('/admin/branding') && r.request().method() === 'PUT');
    await button(panel, 'Применить брендинг').click();
    expect((await applied).status()).toBe(200);
    await expect(panel.getByRole('status')).toContainText('Брендинг применён');
    await expect(p.locator('header .brand')).toHaveAttribute('aria-label', 'Север');
    await top(p);
    await shot(p, 'branding-applied', 'Бренд применён', 'Реальная новая ревизия действует для всей установки. Шапка показывает загруженный логотип; интерфейс использует палитру и выбранный стиль иконок.', 'Оформление');
  });
  await chapter(publicPage, 'Бренд на публичных страницах', 'Получатели и гости видят тот же фирменный стиль. Переключение темы сохраняет читаемость, а мобильная версия показывает тот же продукт. Название также используется в новых письмах и PDF; ранее созданные отчёты сохраняют собственный снимок оформления.', async p => {
    await p.goto('/help');
    await expect(p.locator('header .brand')).toHaveAttribute('aria-label', 'Север');
    await shot(p, 'branding-public', 'Публичная помощь под своим брендом', 'Новый гость получил применённое название и логотип через публичный branding DTO/SSR. Это отдельная браузерная сессия.', 'Оформление');
    await p.getByLabel('Тема', { exact: true }).selectOption('dark');
    await p.setViewportSize({ width: 390, height: 844 });
    await shot(p, 'branding-mobile', 'Фирменная тёмная тема на телефоне', 'Загруженный тёмный логотип и палитра работают в мобильном viewport. Физическое устройство этим кадром не аттестуется.', 'Оформление');
    await p.setViewportSize(manifest.viewport);
    await p.getByLabel('Тема', { exact: true }).selectOption('light');
    const panel = admin.page.getByRole('region', { name: 'Брендинг и оформление', exact: true });
    await button(panel, 'Исходное оформление').click();
    await expect(admin.page.locator('header .brand')).toHaveAttribute('aria-label', 'Север');
    await button(panel, 'Применить брендинг').click();
    await expect(admin.page.locator('header .brand')).toHaveAttribute('aria-label', 'FileX');
    manifest.assertions.push('Branding: real PNG logos/favicon, palette+duotone draft/apply, independent public session light/dark/mobile, explicit reset committed');
  });
  await chapter(publicPage, 'Помощь, языки и статус', 'Завершим публичными страницами. Встроенная помощь объясняет передачу, одноразовое скачивание, проверку и восстановление после ошибок. Есть русский и английский интерфейс. Страница статуса показывает доступность сервиса; production-приёмка остаётся отдельной задачей.', async p => {
    await p.goto('/help');
    await shot(p, 'help', 'Встроенная помощь', 'Публичная помощь объясняет основной сценарий и ограничения одноразовой выдачи. Она доступна без входа.', 'Помощь');
    await p.getByLabel('Язык', { exact: true }).selectOption('en');
    await p.setViewportSize({ width: 390, height: 844 });
    await shot(p, 'english-mobile', 'English на мобильном экране', 'Штатный переключатель локали переводит страницу помощи. Это тот же продукт и responsive UI, снятый в Chromium.', 'Помощь', { fullPage: true });
    await p.setViewportSize(manifest.viewport);
    await p.getByLabel('Language', { exact: true }).selectOption('ru');
    await p.goto('/status');
    await shot(p, 'status', 'Публичный статус', 'Состояние настоящего локального сервиса после восстановления и открытия доступа. Внешние SMTP/S3, устройства и production этим туром не аттестуются.', 'Помощь');
  });
  expect(errors).toEqual([]);
  manifest.assertions.push('Нет JavaScript pageerror; на каждом screenshot отсутствует горизонтальное переполнение');
} catch (error) {
  if (current) {
    await current.page.screenshot({ path: path.join(raw, 'failure.png'), fullPage: true }).catch(() => {});
    await writeFile(path.join(raw, 'failure.txt'), String(error.stack));
  }
  throw error;
} finally {
  if (identityFixture) { identityFixture.stdin.end(); identityFixture.kill(); }
  for (const s of sessions) {
    await s.context.close();
    await s.page.video().saveAs(path.join(raw, `${s.name}.webm`));
  }
  await writeFile(path.join(raw, 'capture.json'), JSON.stringify(manifest, null, 2));
  await browser.close();
}
manifest.passed = true;
await writeFile(path.join(raw, 'capture.json'), JSON.stringify(manifest, null, 2));
console.log(`PASS: ${manifest.screens.length} screenshots, ${manifest.chapters.length} recorded chapters`);
