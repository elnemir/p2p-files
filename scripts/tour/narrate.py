#!/usr/bin/env python3
"""Generate bounded, resumable ElevenLabs narration without exposing credentials."""
import argparse
import base64
from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import re
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
API = 'https://api.elevenlabs.io/v1/'


class ProviderError(RuntimeError):
    def __init__(self, status, code):
        self.status = status
        self.code = code
        super().__init__(f'ElevenLabs HTTP {status}: {code}; no automatic retry')


def digest(data):
    return hashlib.sha256(data).hexdigest()


def save_json(path, data):
    temporary = path.with_suffix('.tmp')
    temporary.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n')
    temporary.replace(path)


def credentials(path):
    values = {}
    if path.exists():
        if path.is_symlink() or not path.is_file() or path.stat().st_mode & 0o077:
            raise ValueError('Credential file must be a private regular file (mode 0600)')
        for line in path.read_text().splitlines():
            if '=' in line and not line.lstrip().startswith('#'):
                key, value = line.split('=', 1)
                if key in ('ELEVENLABS_API_KEY', 'ELEVENLABS_VOICE_ID'):
                    values[key] = value.strip()
    for key in ('ELEVENLABS_API_KEY', 'ELEVENLABS_VOICE_ID'):
        if os.environ.get(key):
            values[key] = os.environ[key]
    if not values.get('ELEVENLABS_API_KEY'):
        raise ValueError('ELEVENLABS_API_KEY is missing; use the private env file or environment')
    return values


def request(key, endpoint, payload=None):
    body = json.dumps(payload, ensure_ascii=False).encode() if payload is not None else None
    req = urllib.request.Request(API + endpoint, data=body,
                                 headers={'xi-api-key': key, 'Content-Type': 'application/json'})
    # Never retry a billable POST automatically: a timeout may follow a charged request.
    try:
        with urllib.request.urlopen(req, timeout=90) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        code = 'http_error'
        try:
            detail = json.load(error).get('detail', {})
            candidate = detail.get('status', '') if isinstance(detail, dict) else ''
            if re.fullmatch(r'[a-z_]+', candidate):
                code = candidate
        except (ValueError, AttributeError):
            pass
        raise ProviderError(error.code, code) from None
    except (urllib.error.URLError, TimeoutError):
        raise RuntimeError('ElevenLabs connection failed; request outcome may be unknown; no automatic retry') from None


def valid_alignment(alignment, text):
    if not isinstance(alignment, dict):
        return False
    chars = alignment.get('characters', [])
    starts = alignment.get('character_start_times_seconds', [])
    ends = alignment.get('character_end_times_seconds', [])
    if not all(isinstance(char, str) and len(char) == 1 for char in chars):
        return False
    if ''.join(chars) != text or not len(chars) == len(starts) == len(ends):
        return False
    last_start = last_end = 0
    for start, end in zip(starts, ends):
        if not isinstance(start, (int, float)) or not isinstance(end, (int, float)):
            return False
        if not math.isfinite(start) or not math.isfinite(end) or start < last_start or end < last_end or end < start:
            return False
        last_start, last_end = start, end
    return bool(ends) and ends[-1] > 0


def load_audio(directory, chapter, metadata):
    entry = metadata.get('chapters', {}).get(str(chapter['number']))
    if not entry or entry.get('status') != 'complete':
        raise ValueError(f'Narration chapter {chapter["number"]} is missing or incomplete')
    if entry['textSha256'] != digest(chapter['narration'].encode()):
        raise ValueError(f'Narration chapter {chapter["number"]} is stale; text changed')
    number = chapter['number']
    if not isinstance(number, int) or not 1 <= number <= 999:
        raise ValueError('Invalid chapter number')
    audio = directory / f'{number:02}.mp3'
    if audio.is_symlink() or not audio.is_file() or digest(audio.read_bytes()) != entry['audioSha256']:
        raise ValueError(f'Narration chapter {number} audio checksum mismatch')
    alignment = entry.get('alignment')
    if not valid_alignment(alignment, chapter['narration']):
        raise ValueError(f'Narration chapter {number} has invalid character alignment')
    return audio, alignment


def load_web_narration(directory, source):
    """Import explicitly downloaded UI audio without claiming provider alignment."""
    journal = json.loads((directory / 'progress.json').read_text())
    if journal.get('provider') != 'ElevenLabs' or journal.get('channel') != 'Chrome web UI':
        raise ValueError('Unsupported web narration journal')
    chapters = journal.get('chapters', [])
    numbers = [ch['number'] for ch in chapters]
    expected = [ch['number'] for ch in source['chapters']]
    if (any(type(n) is not int or not 1 <= n <= 999 for n in numbers)
            or len(numbers) != len(set(numbers)) or set(numbers) != set(expected)):
        raise ValueError('Web narration chapter set does not match the capture')
    if not re.fullmatch(r'[A-Za-z0-9]+', journal.get('voiceId', '')):
        raise ValueError('Invalid web narration voice ID')
    for field in ('voiceName', 'modelId'):
        if not isinstance(journal.get(field), str) or not journal[field].strip():
            raise ValueError(f'Missing web narration {field}')
    if not isinstance(journal.get('voiceSettings'), dict):
        raise ValueError('Missing web narration voice settings')
    entries = {ch['number']: ch for ch in chapters}
    audio_sources = {}
    for chapter in source['chapters']:
        number, text = chapter['number'], chapter['narration']
        entry = entries[number]
        if entry.get('status') != 'downloaded':
            raise ValueError(f'Web narration chapter {number} is incomplete')
        if entry.get('text') != text or entry.get('textSha256') != digest(text.encode()):
            raise ValueError(f'Web narration chapter {number} is stale; text changed')
        if entry.get('file') != f'{number:02}.mp3':
            raise ValueError(f'Web narration chapter {number} has an invalid audio path')
        audio = directory / entry['file']
        if (audio.is_symlink() or not audio.is_file()
                or audio.stat().st_size == 0
                or digest(audio.read_bytes()) != entry.get('audioSha256')):
            raise ValueError(f'Web narration chapter {number} audio checksum mismatch')
        # UI MP3 downloads contain no verified character timestamps; keep that distinction explicit.
        audio_sources[number] = (audio, None)
    metadata = {key: journal[key] for key in ('provider', 'voiceId', 'voiceName', 'modelId', 'voiceSettings')}
    plan = {**metadata, 'acquisition': 'Chrome web UI download',
            'texts': [{'number': ch['number'], 'text': ch['narration']} for ch in source['chapters']]}
    metadata.update(planSha256=digest(json.dumps(plan, sort_keys=True, ensure_ascii=False).encode()),
                    acquisition=plan['acquisition'],
                    characters=sum(len(ch['narration']) for ch in source['chapters']))
    return metadata, audio_sources


def generate(source, output, key, voice_id, model, limit, only=None, retry_rejected=False):
    output.mkdir(parents=True, exist_ok=True)
    lock = output / '.generation.lock'
    try:
        handle = lock.open('x')
    except FileExistsError:
        raise RuntimeError('Narration cache is locked; verify no generator is running before removing a stale lock') from None
    try:
        with handle:
            handle.write(str(os.getpid()))
        return generate_locked(source, output, key, voice_id, model, limit, only, retry_rejected)
    finally:
        lock.unlink()


def generate_locked(source, output, key, voice_id, model, limit, only, retry_rejected):
    chapters = source['chapters']
    numbers = [ch['number'] for ch in chapters]
    if not numbers or len(numbers) != len(set(numbers)) or any(type(n) is not int or not 1 <= n <= 999 for n in numbers):
        raise ValueError('Invalid or duplicate chapter numbers')
    characters = sum(len(ch['narration']) for ch in chapters)
    if any(not ch['narration'].strip() for ch in chapters):
        raise ValueError('Narration text cannot be empty')
    if characters > limit or limit <= 0:
        raise ValueError(f'Tour has {characters} characters, exceeding the {limit} character request budget')
    if only is not None and only not in numbers:
        raise ValueError('Requested chapter is absent')
    if not re.fullmatch(r'[A-Za-z0-9]+', voice_id):
        raise ValueError('Invalid voice ID')
    settings = {'stability': 0.5, 'similarity_boost': 0.75, 'style': 0.0,
                'use_speaker_boost': True, 'speed': 1.0}
    plan = {'provider': 'ElevenLabs', 'voiceId': voice_id, 'modelId': model,
            'voiceSettings': settings, 'format': 'mp3_44100_128',
            'texts': [{'number': ch['number'], 'text': ch['narration']} for ch in chapters]}
    fingerprint = digest(json.dumps(plan, sort_keys=True, ensure_ascii=False).encode())
    output.mkdir(parents=True, exist_ok=True)
    path = output / 'manifest.json'
    metadata = json.loads(path.read_text()) if path.exists() else {
        'schemaVersion': 1, 'planSha256': fingerprint, 'provider': 'ElevenLabs',
        'voiceId': voice_id, 'modelId': model, 'voiceSettings': settings,
        'format': 'mp3_44100_128', 'characters': characters, 'chapters': {}}
    if metadata.get('planSha256') != fingerprint:
        raise ValueError('Narration cache belongs to another voice/model/text; choose a new output directory')
    selected = [ch for ch in chapters if only is None or ch['number'] == only]
    pending = []
    for ch in selected:
        entry = metadata['chapters'].get(str(ch['number']))
        if entry:
            if entry.get('status') == 'rejected' and entry.get('httpStatus') == 402 and retry_rejected:
                pending.append(ch['number'])
            else:
                load_audio(output, ch, metadata)
        else:
            pending.append(ch['number'])
    if not pending:
        print(f'PASS: {len(selected)} cached chapters verified; no API request')
        return metadata
    voice = request(key, 'voices/' + voice_id)
    models = request(key, 'models')
    if not any(m['model_id'] == model and m.get('can_do_text_to_speech')
               and any(l.get('language_id') == 'ru' for l in m.get('languages', [])) for m in models):
        raise ValueError('Selected model does not advertise Russian TTS support')
    metadata['voiceName'] = voice['name']
    for index, ch in enumerate(chapters):
        number = ch['number']
        if number not in pending:
            continue
        payload = {'text': ch['narration'], 'model_id': model, 'voice_settings': settings,
                   'seed': 101100 + number}
        if index:
            payload['previous_text'] = chapters[index - 1]['narration']
        if index + 1 < len(chapters):
            payload['next_text'] = chapters[index + 1]['narration']
        entry = {'status': 'requested', 'textSha256': digest(ch['narration'].encode()),
                 'characters': len(ch['narration']), 'requestedAt': datetime.now(timezone.utc).isoformat()}
        metadata['chapters'][str(number)] = entry
        save_json(path, metadata)
        try:
            response = request(key, f'text-to-speech/{voice_id}/with-timestamps?output_format=mp3_44100_128', payload)
        except ProviderError as error:
            if error.status == 402:
                entry.update(status='rejected', httpStatus=error.status, providerCode=error.code)
                save_json(path, metadata)
            raise
        audio = base64.b64decode(response['audio_base64'], validate=True)
        if not audio:
            raise ValueError('ElevenLabs returned empty audio')
        # Persist a completed response before validation so a rejected alignment never triggers another charge.
        audio_path = output / f'{number:02}.mp3'
        audio_path.write_bytes(audio)
        entry.update(status='complete', audioSha256=digest(audio), alignment=response.get('alignment'))
        save_json(path, metadata)
        load_audio(output, ch, metadata)
        print(f'Generated chapter {number:02}: {len(ch["narration"])} characters', flush=True)
    print(f'PASS: {len(selected)} selected chapters ready; total tour {characters} characters')
    return metadata


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--env-file', type=Path, default=Path.home() / '.config/filex/elevenlabs.env')
    parser.add_argument('--source', type=Path, default=ROOT / '.cache/tour/raw/capture.json')
    parser.add_argument('--output', type=Path, default=ROOT / '.cache/tour/elevenlabs')
    parser.add_argument('--voice-id')
    parser.add_argument('--model', default='eleven_multilingual_v2')
    parser.add_argument('--max-characters', type=int, default=6000)
    parser.add_argument('--chapter', type=int)
    parser.add_argument('--retry-rejected', action='store_true', help='Retry a known HTTP 402 rejection after the API balance is fixed')
    args = parser.parse_args()
    values = credentials(args.env_file)
    voice_id = args.voice_id or values.get('ELEVENLABS_VOICE_ID', '')
    source = json.loads(args.source.read_text())
    generate(source, args.output, values['ELEVENLABS_API_KEY'], voice_id,
             args.model, args.max_characters, args.chapter, args.retry_rejected)


if __name__ == '__main__':
    try:
        main()
    except (ValueError, RuntimeError, OSError, KeyError) as error:
        raise SystemExit(str(error)) from None
