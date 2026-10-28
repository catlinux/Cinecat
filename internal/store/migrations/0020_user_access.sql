-- Acceso por biblioteca y estado de las cuentas de usuario.
ALTER TABLE users ADD COLUMN disabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN all_libraries INTEGER NOT NULL DEFAULT 1;

CREATE TABLE user_libraries (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    library_id INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, library_id)
) STRICT;
