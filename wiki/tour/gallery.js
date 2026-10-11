/* A static gallery works from file:// and loopback HTTP without external assets. */
'use strict';
const data = JSON.parse(document.getElementById('tour-data').textContent);
const grid = document.getElementById('grid');
const filters = document.getElementById('filters');
const search = document.getElementById('search');
const video = document.querySelector('video');
const lightbox = document.getElementById('lightbox');
const time = value => `${Math.floor(value / 60).toString().padStart(2, '0')}:${Math.floor(value % 60).toString().padStart(2, '0')}`;
let category = 'Все экраны';
let shown = data.screens;
let index = 0;
let previousFocus;
function element(tag, text, className) {
  const node = document.createElement(tag);
  if (text) node.textContent = text;
  if (className) node.className = className;
  return node;
}
document.getElementById('duration').textContent = time(Math.round(data.duration));
document.getElementById('screenCount').textContent = data.screens.length;
document.getElementById('chapterCount').textContent = data.chapters.length;
for (const ch of data.chapters) {
  const item = element('li');
  const button = element('button');
  button.append(element('time', time(ch.time)), element('span', ch.title));
  button.setAttribute('aria-label', `${time(ch.time)} ${ch.title}`);
  button.addEventListener('click', () => {
    video.currentTime = ch.time;
    video.play().catch(() => { /* The native controls remain usable if autoplay is disallowed. */ });
  });
  item.append(button);
  document.getElementById('chapters').append(item);
}
video.addEventListener('timeupdate', () => {
  document.querySelectorAll('#chapters button').forEach((node, i) => {
    const active = video.currentTime >= data.chapters[i].time && (!data.chapters[i + 1] || video.currentTime < data.chapters[i + 1].time);
    node.classList.toggle('active', active);
    if (active) node.setAttribute('aria-current', 'true');
    else node.removeAttribute('aria-current');
  });
});
const groups = ['Все экраны', ...new Set(data.screens.map(s => s.group))];
for (const name of groups) {
  const b = element('button', name);
  b.dataset.group = name;
  b.append(element('span', String(name === 'Все экраны' ? data.screens.length : data.screens.filter(s => s.group === name).length)));
  b.addEventListener('click', () => { category = name; render(); });
  filters.append(b);
}
function openImage(i) {
  index = (i + shown.length) % shown.length;
  const screen = shown[index];
  document.getElementById('large').src = screen.file;
  document.getElementById('large').alt = screen.title;
  document.getElementById('lightboxTitle').textContent = screen.title;
  document.getElementById('lightboxDescription').textContent = screen.description;
  document.getElementById('position').textContent = `${screen.id.split('-')[0]} / ${data.screens.length} · ${screen.group}`;
  document.getElementById('original').href = screen.file;
  if (!lightbox.open) { previousFocus = document.activeElement; lightbox.showModal(); }
}
function render() {
  const query = search.value.trim().toLocaleLowerCase('ru');
  shown = data.screens.filter(s => (category === 'Все экраны' || s.group === category) && `${s.title} ${s.description} ${s.group}`.toLocaleLowerCase('ru').includes(query));
  for (const b of filters.children) b.setAttribute('aria-pressed', String(b.dataset.group === category));
  grid.replaceChildren();
  shown.forEach((screen, i) => {
    const card = element('figure', null, 'card');
    const button = element('button', null, `preview${screen.viewport.width < 600 ? ' mobile' : ''}`);
    button.setAttribute('aria-label', `Открыть экран ${screen.id.split('-')[0]}: ${screen.title}`);
    const img = element('img');
    img.src = screen.file; img.alt = screen.title; img.loading = 'lazy'; img.decoding = 'async';
    button.append(img, element('span', '↗', 'zoom'));
    button.addEventListener('click', () => openImage(i));
    const caption = element('figcaption');
    const meta = element('div', null, 'meta');
    meta.append(element('b', screen.id.split('-')[0]), element('span', ` / ${screen.group}`));
    caption.append(meta, element('h3', screen.title), element('p', screen.description));
    card.append(button, caption); grid.append(card);
  });
  document.getElementById('results').textContent = `Показано ${shown.length} из ${data.screens.length}`;
  document.getElementById('empty').hidden = shown.length > 0;
}
search.addEventListener('input', render);
document.getElementById('close').addEventListener('click', () => lightbox.close());
document.getElementById('prev').addEventListener('click', () => openImage(index - 1));
document.getElementById('next').addEventListener('click', () => openImage(index + 1));
lightbox.addEventListener('keydown', event => {
  if (event.key === 'ArrowRight') { event.preventDefault(); openImage(index + 1); }
  if (event.key === 'ArrowLeft') { event.preventDefault(); openImage(index - 1); }
});
lightbox.addEventListener('click', event => { if (event.target === lightbox) lightbox.close(); });
lightbox.addEventListener('close', () => previousFocus?.focus());
render();
