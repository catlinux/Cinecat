-- Instancia de prueba: sin escaneo, sin cambios en disco y sin buscar actualizaciones.
INSERT OR REPLACE INTO settings (key, value) VALUES
  ('scan.interval', '"0"'),
  ('scan.on_startup', 'false'),
  ('library.allow_file_changes', 'false'),
  ('system.check_updates', 'false');
SELECT key, value FROM settings;
