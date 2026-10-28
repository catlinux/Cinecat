"""H3c: prueba del modo copia contra la instancia de prueba (solo lectura).

Uso en el Debian: python3 h3c.py TOKEN CAPS ruta-parcial [indices de segmento...]
Busca el ítem en la copia de la base, pide la reproducción con esas capacidades,
lee la playlist y comprueba que cada segmento pedido empieza donde dice la playlist.
"""

import json
import os
import re
import sqlite3
import subprocess
import sys
import time
import urllib.parse
import urllib.request

BASE = 'http://127.0.0.1:18090'
DB = os.path.expanduser('~/cinecat-prueba/data/cinecat.db')
OFFSET = 10.0  # hlsTimestampOffsetSeconds
MUXDELAY = 1.4  # retardo constante del muxer MPEG-TS medido en H2


def get(url, token, raw=False):
    request = urllib.request.Request(url if url.startswith('http') else BASE + url,
                                     headers={'Authorization': f'Bearer {token}'})
    with urllib.request.urlopen(request, timeout=180) as response:
        data = response.read()
    return data if raw else json.loads(data)


def first_video_pts(path):
    out = subprocess.run(['docker', 'exec', '-i', 'cinecat-prueba-servidor-1', 'ffprobe', '-v', 'error',
                          '-select_streams', 'v:0', '-show_entries', 'packet=pts_time,flags,codec_type',
                          '-show_entries', 'stream=codec_name', '-of', 'json', '-read_intervals', '%+#1',
                          'pipe:0'], input=open(path, 'rb').read(), capture_output=True)
    data = json.loads(out.stdout or b'{}')
    packets = data.get('packets', [])
    codec = (data.get('streams') or [{}])[0].get('codec_name')
    return (float(packets[0]['pts_time']) if packets else None), codec


def main(token, caps, like, indexes):
    db = sqlite3.connect(f'file:{DB}?mode=ro', uri=True)
    row = db.execute('SELECT id, path FROM media_items WHERE path LIKE ? ORDER BY id LIMIT 1',
                     (f'%{like}%',)).fetchone()
    if not row:
        print('No encontrado:', like)
        return
    item_id, path = row
    print(f'Ítem {item_id}: {path}')
    play_query = urllib.parse.urlencode({'caps': caps}) if caps else ''
    t0 = time.monotonic()
    play = get(f'/api/v1/items/{item_id}/play?{play_query}', token)
    print(f'  play ({time.monotonic() - t0:.2f}s): modo={play.get("mode")} url={str(play.get("url"))[:90]}')
    if play.get('mode') != 'remux':
        return
    t0 = time.monotonic()
    playlist = get(play['url'], token, raw=True).decode()
    print(f'  playlist ({time.monotonic() - t0:.2f}s)')
    entries = re.findall(r'#EXTINF:([0-9.]+),\s*\n(\S+)', playlist)
    durations = [float(d) for d, _ in entries]
    starts = [sum(durations[:i]) for i in range(len(durations))]
    modes = {urllib.parse.parse_qs(urllib.parse.urlparse(u).query).get('m', ['?'])[0] for _, u in entries}
    uniform = len({round(d, 3) for d in durations[:-1]}) == 1
    print(f'  {len(entries)} segmentos, m={modes}, duraciones uniformes={uniform}, primeras={durations[:4]}')
    playlist_url = urllib.parse.urljoin(BASE + play['url'], '.')
    for index in indexes:
        if index >= len(entries):
            continue
        uri = entries[index][1]
        t0 = time.monotonic()
        data = get(urllib.parse.urljoin(BASE + play['url'], uri), token, raw=True)
        elapsed = time.monotonic() - t0
        tmp = f'/tmp/h3c-{index}.ts'
        with open(tmp, 'wb') as f:
            f.write(data)
        pts, codec = first_video_pts(tmp)
        os.remove(tmp)
        expected = starts[index]
        got = None if pts is None else pts - OFFSET - MUXDELAY
        diff = None if got is None else got - expected
        print(f'  segmento {index}: {len(data) / 1e6:.1f} MB en {elapsed:.2f}s, códec={codec}, '
              f'previsto {expected:.3f}s, real {got if got is None else round(got, 3)}s, '
              f'desfase {None if diff is None else round(diff * 1000)} ms')


if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2], sys.argv[3], [int(x) for x in sys.argv[4:]])
