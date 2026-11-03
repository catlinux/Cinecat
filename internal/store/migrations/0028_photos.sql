-- SQLite no permite ampliar un CHECK con ALTER TABLE. Se actualiza la
-- definición guardada de `libraries` para admitir el tipo photos sin
-- reconstruir la tabla, que rompería las claves ajenas que la referencian.
PRAGMA writable_schema = ON;
UPDATE sqlite_master
   SET sql = replace(
       sql,
       'CHECK (kind IN (''video'', ''clips'', ''music''))',
       'CHECK (kind IN (''video'', ''clips'', ''music'', ''photos''))'
   )
 WHERE type = 'table' AND name = 'libraries';
PRAGMA writable_schema = OFF;
PRAGMA writable_schema = RESET;

CREATE TABLE photos (
    id         INTEGER PRIMARY KEY,
    library_id INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    path       TEXT    NOT NULL UNIQUE,
    folder     TEXT    NOT NULL,
    name       TEXT    NOT NULL,
    size       INTEGER NOT NULL,
    mtime      INTEGER NOT NULL,
    width      INTEGER NOT NULL,
    height     INTEGER NOT NULL,
    taken_at   INTEGER
) STRICT;

CREATE INDEX photos_library_folder ON photos (library_id, folder);
