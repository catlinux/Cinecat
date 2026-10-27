"""Prototipo H2: lee los fotogramas clave de vídeo del índice Cues de un MKV.

Uso: python3 cues.py archivo.mkv [archivo2.mkv ...]
Imprime por archivo: segundos tardados, nº de claves, intervalo medio y máximo.
Con -v imprime además los tiempos.
"""

import struct
import sys
import time

SEGMENT, SEEKHEAD, SEEK, SEEKID, SEEKPOS = 0x18538067, 0x114D9B74, 0x4DBB, 0x53AB, 0x53AC
INFO, TIMESCALE, TRACKS, TRACKENTRY, TRACKNUM, TRACKTYPE = 0x1549A966, 0x2AD7B1, 0x1654AE6B, 0xAE, 0xD7, 0x83
CUES, CUEPOINT, CUETIME, CUETRACKPOS, CUETRACK = 0x1C53BB6B, 0xBB, 0xB3, 0xB7, 0xF7
CLUSTER = 0x1F43B675


def read_id(f):
    b = f.read(1)
    if not b:
        return None
    first = b[0]
    length = 1
    mask = 0x80
    while length <= 4 and not first & mask:
        mask >>= 1
        length += 1
    rest = f.read(length - 1)
    return int.from_bytes(b + rest, 'big')


def read_size(f):
    b = f.read(1)
    if not b:
        return None
    first = b[0]
    length = 1
    mask = 0x80
    while length <= 8 and not first & mask:
        mask >>= 1
        length += 1
    value = first & (mask - 1)
    for byte in f.read(length - 1):
        value = (value << 8) | byte
    if value == (1 << (7 * length)) - 1:
        return -1  # tamaño desconocido
    return value


def children(f, end):
    while f.tell() < end:
        eid = read_id(f)
        size = read_size(f)
        if eid is None or size is None:
            return
        start = f.tell()
        yield eid, start, size
        if size < 0:
            return
        f.seek(start + size)


def uint(f, start, size):
    f.seek(start)
    return int.from_bytes(f.read(size), 'big')


def keyframes(path):
    with open(path, 'rb') as f:
        f.seek(0, 2)
        file_end = f.tell()
        f.seek(0)
        segment_start = segment_end = None
        for eid, start, size in children(f, file_end):
            if eid == SEGMENT:
                segment_start = start
                segment_end = file_end if size < 0 else start + size
                break
        if segment_start is None:
            raise ValueError('sin Segment')
        positions = {}
        f.seek(segment_start)
        # SeekHead al principio del Segment: posiciones de Info, Tracks y Cues.
        for eid, start, size in children(f, segment_end):
            if eid == SEEKHEAD:
                for sid, sstart, ssize in children(f, start + size):
                    if sid != SEEK:
                        continue
                    target = pos = None
                    for cid, cstart, csize in children(f, sstart + ssize):
                        if cid == SEEKID:
                            f.seek(cstart)
                            target = int.from_bytes(f.read(csize), 'big')
                        elif cid == SEEKPOS:
                            pos = uint(f, cstart, csize)
                    if target is not None and pos is not None:
                        positions.setdefault(target, segment_start + pos)
                    f.seek(sstart + ssize)
                f.seek(start + size)
            elif eid in (INFO, TRACKS):
                positions.setdefault(eid, start - 0)  # ya estamos dentro
                positions[eid] = ('inline', start, size)
            elif eid == CLUSTER:
                break

        def element_at(key):
            value = positions.get(key)
            if value is None:
                return None
            if isinstance(value, tuple):
                return value[1], value[2]
            f.seek(value)
            eid = read_id(f)
            size = read_size(f)
            if eid != key:
                return None
            return f.tell(), size

        scale = 1_000_000
        info = element_at(INFO)
        if info:
            for eid, start, size in children(f, info[0] + info[1]) if f.seek(info[0]) is not None or True else []:
                if eid == TIMESCALE:
                    scale = uint(f, start, size)
                    f.seek(start + size)
        video_track = None
        tracks = element_at(TRACKS)
        if tracks:
            f.seek(tracks[0])
            for eid, start, size in children(f, tracks[0] + tracks[1]):
                if eid != TRACKENTRY:
                    continue
                number = kind = None
                for cid, cstart, csize in children(f, start + size):
                    if cid == TRACKNUM:
                        number = uint(f, cstart, csize)
                    elif cid == TRACKTYPE:
                        kind = uint(f, cstart, csize)
                    f.seek(cstart + csize)
                if kind == 1 and video_track is None:
                    video_track = number
                f.seek(start + size)
        cues = element_at(CUES)
        if not cues:
            raise ValueError('sin Cues')
        times = []
        f.seek(cues[0])
        for eid, start, size in children(f, cues[0] + cues[1]):
            if eid != CUEPOINT:
                continue
            cue_time = None
            tracks_here = []
            for cid, cstart, csize in children(f, start + size):
                if cid == CUETIME:
                    cue_time = uint(f, cstart, csize)
                elif cid == CUETRACKPOS:
                    for tid, tstart, tsize in children(f, cstart + csize):
                        if tid == CUETRACK:
                            tracks_here.append(uint(f, tstart, tsize))
                        f.seek(tstart + tsize)
                f.seek(cstart + csize)
            if cue_time is not None and (video_track is None or video_track in tracks_here):
                times.append(cue_time * scale / 1e9)
            f.seek(start + size)
        return sorted(set(times))


if __name__ == '__main__':
    verbose = '-v' in sys.argv
    for path in [a for a in sys.argv[1:] if a != '-v']:
        t0 = time.monotonic()
        try:
            kf = keyframes(path)
        except Exception as exc:  # prototipo: informar y seguir
            print(f'{path}|ERROR {exc}')
            continue
        dt = time.monotonic() - t0
        gaps = [b - a for a, b in zip(kf, kf[1:])]
        mean = sum(gaps) / len(gaps) if gaps else 0
        print(f'{path}|{dt:.3f}s|{len(kf)} claves|medio {mean:.2f}s|max {max(gaps) if gaps else 0:.2f}s')
        if verbose:
            print(' '.join(f'{t:.3f}' for t in kf))
