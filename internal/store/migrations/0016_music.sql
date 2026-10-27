CREATE TABLE music_artists (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    name      TEXT NOT NULL,
    sort_name TEXT NOT NULL,
    UNIQUE (name COLLATE NOCASE)
) STRICT;

CREATE TABLE music_albums (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    artist_id  INTEGER NOT NULL REFERENCES music_artists (id),
    title      TEXT NOT NULL,
    year       INTEGER,
    cover_path TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (library_id, artist_id, title COLLATE NOCASE)
) STRICT;

CREATE TABLE music_tracks (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id         INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    album_id           INTEGER NOT NULL REFERENCES music_albums (id) ON DELETE CASCADE,
    artist_id          INTEGER NOT NULL REFERENCES music_artists (id),
    path               TEXT NOT NULL,
    title              TEXT NOT NULL,
    track_no           INTEGER,
    disc_no            INTEGER NOT NULL DEFAULT 1,
    year               INTEGER,
    genre              TEXT,
    duration_seconds   REAL,
    codec              TEXT,
    bitrate            INTEGER,
    sample_rate        INTEGER,
    channels           INTEGER,
    size_bytes         INTEGER NOT NULL,
    mtime_unix         INTEGER NOT NULL,
    has_embedded_cover INTEGER NOT NULL DEFAULT 0 CHECK (has_embedded_cover IN (0, 1)),
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (library_id, path)
) STRICT;

CREATE INDEX music_albums_library_id ON music_albums (library_id);
CREATE INDEX music_albums_artist_id ON music_albums (artist_id);
CREATE INDEX music_tracks_library_id ON music_tracks (library_id);
CREATE INDEX music_tracks_album_id ON music_tracks (album_id);
CREATE INDEX music_tracks_artist_id ON music_tracks (artist_id);
