-- Imágenes elegidas a mano que el escaneo no debe pisar.
ALTER TABLE media_items ADD COLUMN poster_locked INTEGER NOT NULL DEFAULT 0;
ALTER TABLE media_items ADD COLUMN backdrop_locked INTEGER NOT NULL DEFAULT 0;
