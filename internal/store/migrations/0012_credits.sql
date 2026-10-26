-- Reparto y equipo con foto y personaje (plan fichas-tmdb, paso 14).
ALTER TABLE media_items ADD COLUMN credits TEXT;
ALTER TABLE series ADD COLUMN credits TEXT;
