"""QA of the exact shipped Preview test.10, without rebuilding the app.

Runs only in disposable CI Windows profiles. Native renderer tests use the
same pinned source, real bootstrapped FFmpeg, and an official unmodified font.
This does not claim testing the user's PC or matching a Figma asset.
"""
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import zipfile

if os.name != 'nt':
    raise SystemExit('Native Windows required')

EXE_HASH = '7071d4f21ce52b36f2dc9c54cd8af280fa42c7934eb0c91c447c05ea10cce3ff'
SOURCE = '57fd6ae2030c522e625ddee93e5f8605584ab492'
FONT_HASH = '7e8c6c5d7bdf4b9e3f171b8e5577ab36cc780735d978db08006fae8306e6bae2'
FONT_URL = 'https://www.onestorecorp.com/assets/fonts/ONE%20Mobile%20POP.zip'
BASE = 'http://127.0.0.1:18768'
exe, source, out = [Path(x).resolve() for x in sys.argv[1:4]]
out.mkdir(parents=True, exist_ok=True)
assert hashlib.sha256(exe.read_bytes()).hexdigest() == EXE_HASH
assert subprocess.check_output(['git', '-C', str(source), 'rev-parse', 'HEAD'], text=True).strip() == SOURCE
go = shutil.which('go')
assert go

root = Path(tempfile.mkdtemp(prefix='preview-clean-windows-'))
profile = root / 'New User 한글'
env = {k: v for k, v in os.environ.items() if not k.startswith('CLIPCATCH_')}
env.update(LOCALAPPDATA=str(profile / 'local'), APPDATA=str(profile / 'roaming'),
           USERPROFILE=str(profile), CLIPCATCH_NO_BROWSER='1')
# The app sees only normal OS executables, not CI's Go/Python/Node toolchains.
env['PATH'] = os.pathsep.join([str(Path(os.environ['SystemRoot']) / 'System32'), os.environ['SystemRoot']])
for key in ['LOCALAPPDATA', 'APPDATA', 'USERPROFILE']:
    Path(env[key]).mkdir(parents=True, exist_ok=True)
data = Path(env['LOCALAPPDATA']) / 'ClipCatch-Preview-V2'
tools = data / 'tools'
assert not tools.exists()
sentinels = []
for name in ['ClipCatch', 'ClipCatch-Test2', 'ClipCatch-Preview']:
    for parent in [Path(env['LOCALAPPDATA']), profile / 'Downloads']:
        marker = parent / name / 'preserve.txt'
        marker.parent.mkdir(parents=True, exist_ok=True)
        marker.write_bytes(b'existing installation and downloads preserved')
        sentinels.append((marker, hashlib.sha256(marker.read_bytes()).hexdigest()))

def get(path):
    with urllib.request.urlopen(BASE + path, timeout=5) as r:
        return json.load(r)

def post(path, value):
    req = urllib.request.Request(BASE + path, data=json.dumps(value).encode(),
                                 headers={'Content-Type': 'application/json', 'Origin': BASE})
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.load(r)

def wait(fn, timeout):
    end = time.monotonic() + timeout
    last = None
    while time.monotonic() < end:
        try:
            value = fn()
            if value:
                return value
        except (OSError, ValueError) as error:
            last = error
        time.sleep(.5)
    raise AssertionError(f'Condition timed out: {last}')

report = {'version': '2.0.1-test.10', 'source': SOURCE, 'exe_sha256': EXE_HASH,
          'app_rebuilt': False, 'user_pc_tested': False, 'figma_asset_equivalence_tested': False}
process = None
log = (out / 'standalone-app.log').open('w', encoding='utf-8')
try:
    process = subprocess.Popen([str(exe)], env=env, stdout=log, stderr=log)
    health = wait(lambda: get('/api/health'), 30)
    assert health == {'app': 'clipcatch-preview-v2', 'version': '2.0.1-test.10', 'channel': 'preview'}, health
    phases = []
    def ready():
        state = get('/api/state')
        phase = (state.get('phase'), state.get('setup_text'))
        if not phases or phases[-1] != phase:
            phases.append(phase)
        assert state.get('phase') != 'error', {k: v for k, v in state.items() if k.startswith('setup')}
        return state if state.get('phase') == 'ready' else None
    state = wait(ready, 360)
    report.update(clean_profile_cold_start=True, developer_tools_absent_from_app_path=True,
                  bootstrap_phases=phases, standalone_health=health)
    for name in ['yt-dlp.exe', 'ffmpeg.exe', 'ffprobe.exe']:
        assert (tools / name).stat().st_size > 1000, name
    ffmpeg = tools / 'ffmpeg.exe'
    report['tools_downloaded_by_app'] = True
    report['ffmpeg_version'] = subprocess.check_output([str(ffmpeg), '-version'], env=env, text=True).splitlines()[0]
    report['font_before_install'] = get('/api/segment/preset')['installed_font']
    assert report['font_before_install'] is False
    try:
        post('/api/segment', {'url': 'https://www.youtube.com/watch?v=jNQXAC9IVRw',
                             'start_seconds': 0, 'end_seconds': 2, 'kind': 'carousel', 'caption': '본문 테스트'})
        raise AssertionError('Missing font must not silently render another font')
    except urllib.error.HTTPError as error:
        assert error.code == 409
        assert 'POP' in error.read().decode('utf-8')
    req = urllib.request.Request(FONT_URL, headers={'User-Agent': 'ClipCatch-QA'})
    with urllib.request.urlopen(req, timeout=60) as response:
        font_zip = response.read(12 * 1024 * 1024)
    with zipfile.ZipFile(io.BytesIO(font_zip)) as archive:
        name = next(x for x in archive.namelist() if x.endswith('ONE Mobile POP OTF.otf') and not x.startswith('__MACOSX'))
        font_bytes = archive.read(name)
    assert hashlib.sha256(font_bytes).hexdigest() == FONT_HASH
    font = Path(env['LOCALAPPDATA']) / 'Microsoft/Windows/Fonts/ONE Mobile POP OTF.otf'
    font.parent.mkdir(parents=True, exist_ok=True)
    font.write_bytes(font_bytes)
    assert get('/api/segment/preset')['installed_font'] is True
    report.update(missing_font_explained=True, official_font_detected=True, font_sha256=FONT_HASH)
    post('/api/stop', {})
    assert process.wait(timeout=30) == 0
    process = None

    harness = Path(__file__).with_name('preview_native_windows_test.go')
    installed_test = source / 'src' / 'preview_cross_pc_qa_test.go'
    assert not installed_test.exists()
    shutil.copy2(harness, installed_test)
    native_env = os.environ.copy()
    native_env.update(LOCALAPPDATA=env['LOCALAPPDATA'],
                      CLIPCATCH_QA_FFMPEG=str(ffmpeg), CLIPCATCH_QA_OUTPUT=str(out))
    native_env.pop('CLIPCATCH_PREVIEW_FONT', None)
    try:
        result = subprocess.run([go, 'test', '-count=1', '-timeout=4m', '-run', '^TestPreviewNativeCrossPC$', '-v', '.'],
                                cwd=source / 'src', env=native_env, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, text=True, encoding='utf-8', timeout=300)
        (out / 'native-render-tests.log').write_text(result.stdout, encoding='utf-8')
        assert result.returncode == 0, result.stdout[-14000:]
    finally:
        installed_test.unlink(missing_ok=True)
    report['native_media'] = json.loads((out / 'native-media-results.json').read_text())
    for marker, digest in sentinels:
        assert hashlib.sha256(marker.read_bytes()).hexdigest() == digest
    report['stable_test2_old_preview_files_preserved'] = True
    report['passed'] = True
finally:
    if process is not None and process.poll() is None:
        try:
            post('/api/stop', {})
            process.wait(timeout=10)
        except Exception:
            process.terminate()
            process.wait(timeout=10)
    log.close()
    (out / 'clean-windows-results.json').write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
print(json.dumps(report, ensure_ascii=False))
