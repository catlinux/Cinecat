-- Metadatos de OMDb para películas (sinopsis y URL del cartel).
ALTER TABLE media_items ADD COLUMN overview TEXT;
ALTER TABLE media_items ADD COLUMN poster_url TEXT;
