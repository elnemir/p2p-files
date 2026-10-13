// Capture real screens and recorded interactions; fixtures never leave the local stack.
import { chromium, expect } from '@playwright/test';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import path from 'node:path';

const command = promisify(execFile);
const root = process.cwd();
const raw = path.join(root, '.cache/tour/raw');
const output = path.join(root, 'wiki/tour');
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
manifest.applicationSourceScope = 'Sorted unique git ls-files --cached --others --exclude-standard apps cmd internal migrations deploy go.mod go.sum package.json package-lock.json; path+NUL+bytes+NUL';
const sessions = [];
const errors = [];
let current;
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
  await chapter(recipient, 'ZIP и завершённая выдача', 'Нажимаем «Скачать всё ZIP». Браузер действительно сохраняет архив. После завершения всей передачи прежняя ссылка больше не открывает файлы: повторный вход отклоняется даже с правильным ПИН-кодом. Завершение означает передачу всех байтов сервером, а сохранение на своём диске получатель проверяет самостоятельно.', async p => {
    await download(p, button(p, 'Скачать всё ZIP'), 'transfer.zip');
    await expect.poll(async () => (await (await owner.page.request.get(`/api/v1/transfers/${transferId}`)).json()).state,
      { timeout: 45000 }).toBe('completed');
    expect((await p.request.get('/api/v1/recipient/files')).status()).toBe(401);
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
    await shot(p, 'preferences', 'Личные настройки', 'Локаль, часовой пояс и каналы уведомлений задаются в кабинете. Тихие часы и дайджест регулируют email-уведомления после подключения SMTP.', 'Настройки');
  });
  await chapter(owner, 'Безопасность учётной записи', 'Чувствительные действия требуют свежего подтверждения пароля. В настройках видны сеансы и история входов. API-токену можно выдать только нужные права и ограниченный срок; секрет показывается один раз. В туре новый токен не создаём.', async p => {
    await focus(p, heading(p, 'Подтверждение личности'));
    await shot(p, 'account', 'Пароль и сеансы', 'Подтверждение личности действует пять минут. Смена пароля отзывает сеансы и API-токены. Активные сеансы и история входов доступны ниже.', 'Настройки');
    await focus(p, heading(p, 'API-токен'));
    await shot(p, 'tokens', 'API-токены с ограничениями', 'Форма позволяет выбрать scopes и срок от 1 до 30 дней. Секреты не создавались и не включены в тур; на экране показана конфигурация.', 'Настройки');
    await focus(p, heading(p, 'Webhook'));
    await shot(p, 'webhooks', 'Webhook-интеграция', 'Для webhook задаётся HTTPS URL. Реальный endpoint не подключён; экран показывает форму, а не подтверждение внешней доставки.', 'Настройки');
  });
  await chapter(owner, 'Тёмная тема и мобильный интерфейс', 'Интерфейс поддерживает светлую, тёмную и системную темы. Посмотрим настройки на узком экране. Это та же рабочая сессия и те же данные; проверка размера выполнена в браузере, без физического устройства.', async p => {
    await p.getByLabel('Тема', { exact: true }).selectOption('dark');
    await top(p);
    await shot(p, 'dark', 'Тёмная тема', 'Тема переключена штатным селектором. Текст и формы сняты с реального UI без ретуши.', 'Настройки');
    await p.setViewportSize({ width: 390, height: 844 });
    await shot(p, 'settings-mobile', 'Настройки на телефоне', 'Мобильное представление верхней части настроек в тёмной теме, 390 × 844. Полная страница остаётся доступна прокруткой.', 'Настройки');
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
    const exports = p.getByRole('region', { name: 'Экспорты пользователей', exact: true });
    await button(exports, 'Загрузить экспорты').click();
    await exports.getByLabel('Причина доступа к экспорту').fill('Учебная проверка доступа к синтетическому экспорту');
    await expect(button(exports, 'Скачать с записью в аудит').first()).toBeEnabled();
    await focus(p, exports);
    await shot(p, 'admin-exports', 'Аудируемый доступ к экспортам', 'Экспорты владельца доступны через отдельную форму с причиной действия. Административный доступ не превращается в неявный пользовательский доступ.', 'Администрирование');
    await download(p, button(exports, 'Скачать с записью в аудит').first(), 'admin-export.bin');
  });
  await chapter(admin, 'Хранилища, перенос и ключи', 'Для payload, артефактов и резервных копий используются отдельные назначения хранилищ. В нашем стенде это три Local-провайдера. Интерфейс также поддерживает S3, перенос между провайдерами и ротацию классов ключей. Эти экраны показываем без подключения внешнего S3.', async p => {
    await focus(p, heading(p, 'Хранилища'));
    await shot(p, 'storage', 'Провайдеры хранилищ', 'Три реально созданных Local-провайдера: payload, artifacts и backups. Можно выводить провайдера из новых загрузок; S3 требует отдельной конфигурации и внешней приёмки.', 'Администрирование');
    await focus(p, heading(p, 'Перенос хранилища'));
    await shot(p, 'migration', 'Перенос между хранилищами', 'Перенос выбирает исходный и целевой провайдеры и создаёт фоновую задачу. На этом экране задача не запущена: в каждом назначении стенда только один Local-провайдер.', 'Администрирование');
    await focus(p, heading(p, 'Ротация ключей'));
    await shot(p, 'keys', 'Ключи и удержание', 'Ротация разделена по классам ключей. Ниже расположено удержание объектов с обязательной причиной. Тур не выполняет ненужную ротацию или реальное расследование.', 'Администрирование');
  });
  await chapter(admin, 'Расследование, SMTP и системные настройки', 'Для расследования предусмотрены отдельные действия и причина; экспертная выдача не равна обычному скачиванию получателя. Ниже находятся SMTP и типизированные настройки приложения. Здесь показаны формы: реальные почтовые реквизиты и внешние адресаты отсутствуют.', async p => {
    await focus(p, heading(p, 'Экспертная проверка файлов'));
    await shot(p, 'forensic', 'Экспертные операции', 'Forensic flow имеет собственные условия и аудит; ZIP требует удержания. В туре снята форма без получения пользовательских payload через административный обход.', 'Администрирование');
    await focus(p, heading(p, 'SMTP'));
    await shot(p, 'smtp', 'Настройка SMTP', 'Поля SMTP оставлены без рабочих credentials. Пустая форма не подтверждает проверку соединения или доставку письма.', 'Администрирование');
    await focus(p, heading(p, 'Настройки приложения'));
    await shot(p, 'app-settings', 'Типизированные настройки', 'Форма позволяет загрузить текущие настройки, сохранить черновик JSON и проверить ревизию. В кадре активна ревизия №1, редактор ещё пуст; изменения в production не выполнялись.', 'Администрирование');
    await focus(p, heading(p, 'Журнал безопасности'));
    await shot(p, 'audit', 'Журнал безопасности', 'Блок запускает проверку целостности журнала безопасности; сами записи здесь не перечислены. Административные действия фиксируются сервером, а контрольные точки скачиваются в разделе эксплуатации.', 'Администрирование');
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
