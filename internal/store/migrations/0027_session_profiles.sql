-- Ajustes por dispositivo (sesión): tope de calidad y idiomas preferidos.
-- Si no hay fila, el perfil está vacío y se usan los valores por defecto.
CREATE TABLE session_profiles (
    session_id        INTEGER PRIMARY KEY REFERENCES sessions (id) ON DELETE CASCADE,
    max_quality       TEXT    NOT NULL DEFAULT '',
    audio_language    TEXT    NOT NULL DEFAULT '',
    subtitle_language TEXT    NOT NULL DEFAULT ''
) STRICT;
