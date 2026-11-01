-- Filas del portal que cada usuario decide ocultar en su inicio.
CREATE TABLE user_portal_rows (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    row_id  TEXT    NOT NULL,
    hidden  INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, row_id)
) STRICT;
