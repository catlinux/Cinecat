# TODO

[English version](TODO.en.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

## Antes de escribir código
- 🔴 **Elegir la licencia** (compatible con el código GPL que queramos reutilizar, y que conserve autoría y créditos). Ver [licencias y créditos](docs/es/licencias-y-creditos.md).
- 🔴 Comprobar que el nombre **Cinecat** está libre (GitHub, Google Play, marcas).
- 🔴 Desactivar la coautoría automática de Claude Code en los commits y fijar `user.name` / correo `noreply` de Catlinux.
- 🔴 Definir la pila tecnológica del servidor, web y Android (propuestas en [arquitectura](docs/es/arquitectura.md)).
- 🟠 Crear `CLAUDE.md` con las normas del proyecto.
- 🟠 Proponer 2-3 direcciones de diseño para la interfaz y elegir una.
- 🟠 Activar en GitHub los avisos de seguridad privados (private vulnerability reporting).

## Producto
- 🟠 Servidor mínimo: biblioteca, identificación de películas y series, metadatos en `es-ES`.
- 🟠 Motor de decisión de reproducción (directa / remux / transcodificación).
- 🟠 Cliente web con inicio de sesión.
- 🟠 Clientes Android y Android TV.
- 🟢 Fotos y presentaciones.
- 🟢 Contenido de internet y televisión en directo (solo si se decide).
- 🟢 Segmentos de intro/créditos, descargas sin conexión, sincronizar reproducción entre dispositivos.
- 🟢 Otras plataformas (Apple, televisores Samsung/LG).

## Documentación
- 🟠 Guía de usuario y glosario «para tontos» cuando exista algo que probar.
- 🟠 Código de conducta.
- 🟢 Capturas de pantalla y guía de despliegue con Docker.

## Mantenimiento de documentos
- Mantener sincronizadas las versiones en español e inglés de cada documento.
