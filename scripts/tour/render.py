#!/usr/bin/env python3
"""Assemble recorded UI clips, Russian narration, captions and a portable gallery."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import re
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[2]
RAW = ROOT / '.cache/tour/raw'
OUT = ROOT / 'wiki/tour'
BUILD = ROOT / '.cache/tour/render'


def run(args):
    result = subprocess.run([str(x) for x in args], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if result.returncode:
        raise RuntimeError(result.stderr[-4000:])
    return result


def timestamp(value, vtt=True):
    milliseconds = round(value * 1000)
    hours, milliseconds = divmod(milliseconds, 3600000)
    minutes, milliseconds = divmod(milliseconds, 60000)
    seconds, milliseconds = divmod(milliseconds, 1000)
    if vtt:
        return f'{hours:02}:{minutes:02}:{seconds:02}.{milliseconds:03}'
    return f'{hours}:{minutes:02}:{seconds:02}.{milliseconds // 10:02}'


def chunks(text, limit=118):
    result = []
    for sentence in re.split(r'(?<=[.!?])\s+', text):
        remaining = sentence.strip()
        parts = math.ceil(len(remaining) / limit)
        # Balance long sentences to avoid a final one-word flash caption.
        while parts > 1:
            spaces = [m.start() for m in re.finditer(' ', remaining)]
            split = min(spaces, key=lambda pos: abs(pos - len(remaining) / parts))
            result.append(remaining[:split])
            remaining = remaining[split + 1:]
            parts -= 1
        if remaining:
            result.append(remaining)
    return result


def caption_lines(text, limit=83):
    words, lines, current = text.split(), [], ''
    for word in words:
        if len(current) + len(word) + 1 > limit:
            lines.append(current)
            current = ''
        current = f'{current} {word}'.strip()
    if current:
        lines.append(current)
    return r'\N'.join(lines)


def gallery(data):
    template = (ROOT / 'scripts/tour/gallery.html').read_text()
    serialized = json.dumps(data, ensure_ascii=False).replace('<', r'\u003c')
    (OUT / 'index.html').write_text(template.replace('/* MANIFEST */', serialized))
    shutil.copyfile(ROOT / 'scripts/tour/gallery.css', OUT / 'gallery.css')
    shutil.copyfile(ROOT / 'scripts/tour/gallery.js', OUT / 'gallery.js')


def markdown(data):
    lines = ['# FileX: скрин-тур и видеотур', '',
             '[Открыть интерактивный тур](tour/index.html) · [Видео с русской озвучкой](tour/filex-tour.mp4) · [Субтитры](tour/captions.ru.vtt)', '',
             '## Контекст и границы', '',
             f'Съёмка: {data["recordedAt"][:10]}. Исходные скриншоты: {len(data["screens"])}; главы видео: {len(data["chapters"])}. '
             'Это настоящий интерфейс FileX в отдельном локальном Compose: вымышленные пользователи, документы и ссылки. '
             'Цель — дать новому пользователю и инженеру полный маршрут по основным разделам без запуска приложения.', '',
             'Видеоряд записан во время действий браузера; ожидания сокращены, темп отдельных фрагментов подстроен под русскую синтезированную озвучку Milena. '
             'Подписи добавлены за пределами интерфейса. PNG сохранены без ретуши. Мобильные кадры — эмуляция viewport Chromium, не физические устройства.', '',
             'ZIP, QR, PDF, JSON, CSV, backup/restore, сверка и открытие доступа выполнены на стенде. '
             'S3/SMTP/webhook/миграция провайдеров/ротация/forensic показаны как формы, без внешней операции. '
             'Временные ресурсы после записи удалены; ссылки из кадров больше не действуют. '
             'Тур не закрывает [внешнюю приёмку](environment-acceptance.md), 5 partial MUST или 14 pending decisions.', '',
             '## Как смотреть', '',
             'Интерактивная галерея работает без сборки и внешних сервисов: откройте HTML или отдайте каталог обычным HTTP-сервером. '
             'В ней есть поиск, разделы, увеличение изображения, стрелки/клавиша Escape и переход к главам видео. '
             'MP4 содержит русскую озвучку и вшитые субтитры; отдельный VTT доступен для использования в другом плеере.', '',
             '## Главы видео', '', '| Время | Глава |', '|---|---|']
    for ch in data['chapters']:
        seconds = int(ch['time'])
        lines.append(f'| {seconds // 60:02}:{seconds % 60:02} | {ch["title"]} |')
    lines.extend(['', '## Скрин-тур', ''])
    last = None
    for screen in data['screens']:
        if screen['group'] != last:
            lines += [f'### {screen["group"]}', '']
            last = screen['group']
        lines += [f'**{screen["id"].split("-")[0]}. {screen["title"]}**', '',
                  screen['description'], '', f'![{screen["title"]}](tour/{screen["file"]})', '']
    lines += ['## Как продолжить', '', '[Воспроизведение тура и проверка артефактов](tour-maintenance.md). '
              '[Архитектура](architecture.md), [разработка](development.md), [эксплуатация](operations-recovery.md), '
              '[текущий статус](current-state.md), [индекс wiki](index.md).', '']
    (ROOT / 'wiki/screen-tour.md').write_text('\n'.join(lines))


def checksums():
    files = [p for p in OUT.rglob('*') if p.is_file() and p.name != 'SHA256SUMS'
             and not any(part.startswith('.') for part in p.relative_to(OUT).parts)]
    (OUT / 'SHA256SUMS').write_text(''.join(f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.relative_to(OUT)}\n' for p in sorted(files)))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--ffmpeg', default=shutil.which('ffmpeg'))
    parser.add_argument('--voice', default='Milena')
    parser.add_argument('--font', default='/System/Library/Fonts/Supplemental/Arial.ttf')
    parser.add_argument('--gallery-only', action='store_true')
    args = parser.parse_args()
    if args.gallery_only:
        data = json.loads((OUT / 'manifest.json').read_text())
        gallery(data)
        markdown(data)
        checksums()
        return
    if not args.ffmpeg or not Path(args.ffmpeg).is_file() or not shutil.which('say'):
        raise SystemExit('Rendering requires ffmpeg with libx264/libass and macOS say (Milena), or reuse the shipped media with --gallery-only')
    source = json.loads((RAW / 'capture.json').read_text())
    if not source.get('passed') or len(source['chapters']) != 19 or len(source['screens']) != 44:
        raise SystemExit('Capture is incomplete; fix the scenario before rendering')
    BUILD.mkdir(parents=True, exist_ok=True)
    subtitles = ['WEBVTT', '']
    metadata = [';FFMETADATA1', 'title=FileX — экскурсия по системе', 'comment=Local synthetic demo; Russian synthesized narration; edited pacing']
    offset, clips, chapters = 0.0, [], []
    for ch in source['chapters']:
        number = ch['number']
        text = BUILD / f'{number:02}.txt'
        audio = BUILD / f'{number:02}.aiff'
        title = BUILD / f'{number:02}.title.txt'
        ass = BUILD / f'{number:02}.ass'
        clip = BUILD / f'{number:02}.mp4'
        if not text.exists() or text.read_text() != ch['narration']:
            text.write_text(ch['narration'])
        title.write_text(f'FileX   /   {number:02}   {ch["title"]}')
        if not audio.exists() or audio.stat().st_mtime < text.stat().st_mtime:
            run(['say', '-v', args.voice, '-r', '176', '-f', text, '-o', audio])
        probe = subprocess.run([args.ffmpeg, '-hide_banner', '-i', str(audio)], capture_output=True, text=True)
        match = re.search(r'Duration: (\d+):(\d+):(\d+\.\d+)', probe.stderr)
        if not match:
            raise RuntimeError('Cannot inspect narration duration')
        hours, minutes, seconds = map(float, match.groups())
        speech = hours * 3600 + minutes * 60 + seconds
        original = ch['end'] - ch['start']
        duration = math.ceil(max(speech + 1.0, original / 2.25) * 25) / 25
        captions = chunks(ch['narration'])
        weight = sum(len(x) for x in captions)
        cursor = 0.2
        ass_lines = ['[Script Info]', 'ScriptType: v4.00+', 'PlayResX: 1440', 'PlayResY: 1080',
                     '[V4+ Styles]', 'Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding',
                     'Style: Default,Arial,22,&H00FFFFFF,&H00FFFFFF,&H00312F13,&H00312F13,0,0,0,0,100,100,0,0,1,0,0,2,32,32,9,1',
                     '[Events]', 'Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text']
        for caption in captions:
            end = cursor + speech * len(caption) / weight
            ass_lines.append(f'Dialogue: 0,{timestamp(cursor, False)},{timestamp(end, False)},Default,,0,0,0,,{caption_lines(caption)}')
            subtitles += [f'{timestamp(offset + cursor)} --> {timestamp(offset + end)}', caption, '']
            cursor = end
        ass.write_text('\n'.join(ass_lines))
        filters = (f'setpts={(duration / original):.8f}*(PTS-STARTPTS),fps=25,'
                   f'pad=1440:1080:0:56:color=0x132f31,'
                   f'drawtext=fontfile={args.font}:textfile={title}:fontcolor=white:fontsize=25:x=28:y=14,'
                   f'ass={ass},tpad=stop_mode=clone:stop_duration=2')
        run([args.ffmpeg, '-hide_banner', '-loglevel', 'error', '-y', '-ss', f'{ch["start"]:.3f}',
             '-t', f'{original:.3f}', '-i', RAW / ch['video'], '-i', audio,
             '-vf', filters, '-af', 'apad', '-t', f'{duration:.3f}', '-c:v', 'libx264',
             '-preset', 'fast', '-crf', '24', '-pix_fmt', 'yuv420p', '-c:a', 'aac', '-b:a', '96k',
             '-ar', '48000', '-ac', '1', '-movflags', '+faststart', clip])
        clips.append(clip)
        chapters.append({ 'number': number, 'title': ch['title'], 'time': round(offset, 2),
                         'duration': duration, 'screens': ch['screens'], 'narration': ch['narration'] })
        metadata += ['[CHAPTER]', 'TIMEBASE=1/1000', f'START={round(offset * 1000)}',
                     f'END={round((offset + duration) * 1000)}', f'title={ch["title"]}']
        offset += duration
        print(f'Rendered {number:02}: {duration:.1f}s', flush=True)
    listing = BUILD / 'concat.txt'
    listing.write_text('\n'.join(f"file '{p}'" for p in clips))
    meta = BUILD / 'chapters.txt'
    meta.write_text('\n'.join(metadata))
    run([args.ffmpeg, '-hide_banner', '-loglevel', 'error', '-y', '-f', 'concat', '-safe', '0', '-i', listing,
         '-i', meta, '-map', '0:v', '-map', '0:a', '-map_metadata', '1', '-map_chapters', '1',
         '-c', 'copy', '-movflags', '+faststart', OUT / 'filex-tour.mp4'])
    (OUT / 'captions.ru.vtt').write_text('\n'.join(subtitles))
    public = {k: v for k, v in source.items() if k != 'chapters'}
    public.update(chapters=chapters, duration=round(offset, 2), narration='macOS Milena, synthesized Russian',
                  recording='Real Chromium interactions, edited pace; no screenshot retouching', video='filex-tour.mp4')
    (OUT / 'manifest.json').write_text(json.dumps(public, ensure_ascii=False, indent=2) + '\n')
    gallery(public)
    markdown(public)
    checksums()
    print(f'PASS: {len(public["screens"])} screenshots, {offset:.2f}s video with audio/subtitles/chapters')


if __name__ == '__main__':
    main()
