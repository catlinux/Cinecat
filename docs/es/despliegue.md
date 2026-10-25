# Despliegue con Docker

[English](../en/deployment.md) · [Volver al README](../../README.md)

Cinecat se despliega con Docker Compose: tres contenedores (servidor Go, cliente web, proxy Caddy) detrás de un único punto de entrada, para que las cookies de sesión y el reproductor HLS funcionen sin configurar CORS a mano.

## Requisitos

- Docker y Docker Compose (plugin `docker compose`, no el antiguo `docker-compose`).
- La carpeta de tu biblioteca de vídeos accesible en el equipo donde corre Docker.

## Primer despliegue

```sh
git clone git@github-catlinux:catlinux/Cinecat.git
cd Cinecat
cp .env.deploy.example .env
```

Edita `.env` y pon al menos `CINECAT_MEDIA_PATH` (la ruta real de tu biblioteca). Los demás valores son opcionales al principio.

```sh
docker compose up -d --build
docker compose logs servidor
```

El primer arranque crea el usuario `admin`. Si no fijaste `CINECAT_ADMIN_PASSWORD` en `.env`, la contraseña generada aparece **una sola vez** en ese log: cópiala antes de que se pierda de vista.

Abre `http://<IP del Debian>/` (o `http://localhost/` si estás en la misma máquina). Inicia sesión como `admin`, crea una biblioteca apuntando a `/media` (la ruta interna del contenedor: Docker la traduce a `CINECAT_MEDIA_PATH` por ti) y lanza un escaneo desde la propia interfaz o con:

```sh
curl -X POST http://localhost/api/v1/libraries/scan -H "Authorization: Bearer <tu token>"
```

## Aplicar una versión nueva

Cada vez que haya cambios que probar:

```sh
./scripts/actualizar.sh
```

Hace `git pull`, reconstruye las imágenes y reinicia los contenedores. La base de datos (volumen `cinecat_data`) y los certificados de Caddy (volúmenes `caddy_data`/`caddy_config`) no se tocan: tu biblioteca, usuarios y bibliotecas dadas de alta se conservan entre actualizaciones.

## Dominio propio y HTTPS

Por defecto (`CINECAT_DOMAIN=:80`) Caddy sirve en HTTP sin certificado. Si apuntas un dominio propio a la IP del Debian, pon ese dominio en `CINECAT_DOMAIN` (por ejemplo `cinecat.tudominio.com`) y Caddy obtendrá un certificado HTTPS automático de Let's Encrypt la primera vez que alguien acceda desde fuera. Asegúrate de que los puertos 80 y 443 llegan al Debian (redirección en el router) antes de activar esto.

## Qué no cubre todavía este despliegue

- **Metadatos de TMDB**: `internal/metadata` no está conectado al servidor; verás títulos extraídos del nombre de archivo, sin carteles ni sinopsis.
- **Bloqueo de login por IP**: el límite de intentos fallidos se indexa por usuario, no por IP (ver `TODO.md`). Aceptable en red local; revisar antes de exponer el dominio a internet sin más protección.
- **Copias de seguridad**: el volumen `cinecat_data` guarda la base de datos SQLite; haz copia periódica de ese volumen (`docker run --rm -v cinecat_data:/data -v $(pwd):/backup debian tar czf /backup/cinecat-db.tar.gz /data`) si te importa no perder usuarios y bibliotecas dadas de alta.
