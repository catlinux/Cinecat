"""Prototipo H2, experimento 2: segmentación con copia de vídeo en claves elegidas.

Uso (en el Debian): python3 h2_segmentos.py archivo.mkv INICIO N
  INICIO: índice del segmento por el que se empieza (simula un salto).
  N: cuántos segmentos generar.
Calcula los límites con las claves del índice Cues (regla: el siguiente límite es
la primera clave >= límite anterior + 6 s), lanza FFmpeg en el contenedor con el
muxer segment y comprueba con ffprobe que cada segmento empieza donde debe.
"""

import importlib.util
import json
import os
import subprocess
import sys

HLS_SECONDS = 6.0
OFFSET = 10  # igual que hlsTimestampOffsetSeconds: marcas de tiempo absolutas + 10 s
CONTAINER = 'cinecat-servidor-1'
WORK = '/tmp/h2'
# FFmpeg resta ~0,13 s al buscar en vídeos con fotogramas B; se busca un poco
# después de la clave para no caer en la anterior.
SEEK_MARGIN = float(os.environ.get('SEEK_MARGIN', '0.2'))

spec = importlib.util.spec_from_file_location('cues', os.path.expanduser('~/.cache/cinecat-h2-cues.py'))
cues = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cues)


def boundaries(keyframes):
    result = [keyframes[0]]
    for t in keyframes[1:]:
        if t >= result[-1] + HLS_SECONDS - 1e-6:
            result.append(t)
    return result


def run(cmd):
    return subprocess.run(['docker', 'exec', CONTAINER] + cmd, capture_output=True, text=True)


def keyframes(path):
    try:
        return cues.keyframes(path)
    except Exception:
        # Respaldo para lo que no es MKV: paquetes con ffprobe (lee el archivo entero).
        proc = run(['ffprobe', '-v', 'error', '-select_streams', 'v:0', '-show_entries',
                    'packet=pts_time,flags', '-of', 'csv=p=0', path])
        return sorted({float(line.split(',')[0]) for line in proc.stdout.splitlines()
                       if 'K' in line and line.split(',')[0] not in ('', 'N/A')})


def main(path, start_index, count):
    kf = keyframes(path)
    bounds = boundaries(kf)
    start = bounds[start_index]
    wanted = bounds[start_index:start_index + count + 1]
    # Con -copyts las marcas de tiempo son las del archivo, empiece donde empiece:
    # los cortes se dan en tiempo absoluto. -ss unos ms después de la clave para
    # que la búsqueda no caiga en la clave anterior.
    # El muxer segment ve las marcas ya desplazadas por -output_ts_offset.
    cut_times = ','.join(f'{t + OFFSET:.6f}' for t in wanted[1:-1])
    seek = start + SEEK_MARGIN
    run(['sh', '-c', f'rm -rf {WORK} && mkdir -p {WORK}'])
    cmd = [
        'nice', '-n', '19', 'ffmpeg', '-hide_banner', '-loglevel', 'error', '-nostdin', '-y',
        '-copyts', '-ss', f'{seek:.6f}', '-t', f'{wanted[-1] - start:.6f}', '-i', path,
        '-map', '0:V:0', '-map', '0:a:0?',
        '-c:v', 'copy', '-c:a', 'aac', '-ac', '2', '-b:a', '160k',
        '-output_ts_offset', f'{OFFSET}',
        '-f', 'segment', '-segment_format', 'mpegts',
        '-segment_times', cut_times, '-segment_time_delta', '0.05',
        '-segment_start_number', str(start_index),
        '-segment_list', f'{WORK}/lista.csv', '-segment_list_type', 'csv',
        f'{WORK}/segment_%05d.ts',
    ]
    proc = run(cmd)
    if proc.returncode != 0:
        print('FFmpeg falló:', proc.stderr[-2000:])
        return
    print(f'Claves: {len(kf)}; segmentos totales: {len(bounds)}; empiezo en el {start_index} ({start:.3f} s)')
    worst = 0.0
    for i in range(count):
        index = start_index + i
        name = f'{WORK}/segment_{index:05d}.ts'
        probe = run(['ffprobe', '-v', 'error', '-select_streams', 'v:0', '-show_entries',
                     'packet=pts_time,flags', '-of', 'json', '-read_intervals', '%+#1', name])
        if probe.returncode != 0:
            print(f'  {index}: no existe ({probe.stderr.strip()[:80]})')
            continue
        packets = json.loads(probe.stdout).get('packets', [])
        first = packets[0] if packets else {}
        got = float(first.get('pts_time', 'nan')) - OFFSET
        expected = wanted[i]
        diff = abs(got - expected)
        worst = max(worst, diff)
        print(f'  {index}: esperado {expected:.3f} s, obtenido {got:.3f} s, clave={"K" in first.get("flags", "")}, '
              f'duración prevista {wanted[i + 1] - wanted[i]:.2f} s')
    print(f'Desviación máxima: {worst * 1000:.1f} ms')


if __name__ == '__main__':
    main(sys.argv[1], int(sys.argv[2]), int(sys.argv[3]))
