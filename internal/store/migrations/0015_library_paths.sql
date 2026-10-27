-- Varias carpetas y tipo por biblioteca. `libraries.path` se conserva como
-- carpeta principal para no romper las bases y los clientes antiguos.
ALTER TABLE libraries ADD COLUMN kind TEXT NOT NULL DEFAULT 'video'
	CHECK (kind IN ('video', 'clips', 'music'));

CREATE TABLE library_paths (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    path       TEXT    NOT NULL UNIQUE,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
) STRICT;

CREATE INDEX library_paths_library_id ON library_paths (library_id);

INSERT INTO library_paths (library_id, path)
SELECT id, path FROM libraries;
