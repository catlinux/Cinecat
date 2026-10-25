# Despliegue con Docker

[English](../en/deployment.md) · [Volver al README](../../README.md)

Cinecat se despliega con Docker Compose: dos contenedores (servidor Go, cliente web), cada uno publicado solo en `127.0.0.1` del Debian. El Apache ya instalado en esa máquina hace de proxy inverso hacia ambos y gestiona HTTPS, igual que con sus otros vhosts.

## Requisitos

- Docker y Docker Compose (plugin `docker compose`, no el antiguo `docker-compose`).
- Apache con `mod_proxy`, `mod_proxy_http` y `mod_headers` activados (`a2enmod proxy proxy_http headers`), ya en marcha con otros vhosts.
- La carpeta de tu biblioteca de vídeos accesible en el equipo donde corre Docker.

## 1. Levantar los contenedores

```sh
git clone git@github-catlinux:catlinux/Cinecat.git
cd Cinecat
cp .env.deploy.example .env
```

Edita `.env` y pon al menos `CINECAT_MEDIA_PATH` (la ruta real de tu biblioteca). Los puertos `CINECAT_API_PORT`/`CINECAT_WEB_PORT` (por defecto 8080/8081) solo hace falta cambiarlos si ya los usas para otra cosa en ese Debian.

```sh
docker compose up -d --build
docker compose logs servidor
```

El primer arranque crea el usuario `admin`. Si no fijaste `CINECAT_ADMIN_PASSWORD` en `.env`, la contraseña generada aparece **una sola vez** en ese log: cópiala antes de que se pierda de vista.

En este punto el servidor ya responde en `http://127.0.0.1:8080` y el cliente web en `http://127.0.0.1:8081`, pero solo desde la propia máquina: el siguiente paso es el vhost de Apache que los hace accesibles desde fuera.

## 2. Vhost de Apache

El archivo `deploy/video.warcrafted.com.conf` del repositorio ya trae la configuración para `video.warcrafted.com`. Cópialo a `sites-available` y actívalo:

```sh
sudo cp deploy/video.warcrafted.com.conf /etc/apache2/sites-available/
sudo a2ensite video.warcrafted.com.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
```

Para HTTPS, usa certbot igual que con tus otros vhosts (añade el bloque `:443` y el redirect automáticamente):

```sh
sudo certbot --apache -d video.warcrafted.com
```

Certbot copia las directivas `ProxyPass` al nuevo bloque `:443`, pero no actualiza la cabecera `X-Forwarded-Proto`. Edita `/etc/apache2/sites-available/video.warcrafted.com-le-ssl.conf` (o el archivo que certbot haya creado) y cambia, dentro del bloque `<VirtualHost *:443>`:

```apache
RequestHeader set X-Forwarded-Proto "https"
```

y recarga Apache (`sudo systemctl reload apache2`). Si no lo haces, la cookie de sesión no se marcará `Secure` en HTTPS.

Abre `https://video.warcrafted.com/`. Inicia sesión como `admin`, crea una biblioteca apuntando a `/media` (la ruta interna del contenedor: Docker la traduce a `CINECAT_MEDIA_PATH` por ti) y lanza un escaneo desde la propia interfaz o con:

```sh
curl -X POST https://video.warcrafted.com/api/v1/libraries/scan -H "Authorization: Bearer <tu token>"
```

## Aplicar una versión nueva

Cada vez que haya cambios que probar:

```sh
./scripts/actualizar.sh
```

Hace `git pull`, reconstruye las imágenes y reinicia los contenedores. La base de datos (volumen `cinecat_data`) no se toca: tu biblioteca, usuarios y bibliotecas dadas de alta se conservan entre actualizaciones. El vhost de Apache no cambia salvo que el archivo `deploy/video.warcrafted.com.conf` se modifique en un commit (entonces hay que volver a copiarlo y recargar Apache a mano).

## Qué no cubre todavía este despliegue

- **Metadatos de TMDB**: `internal/metadata` no está conectado al servidor; verás títulos extraídos del nombre de archivo, sin carteles ni sinopsis.
- **Bloqueo de login por IP**: el límite de intentos fallidos se indexa por usuario, no por IP (ver `TODO.md`). Revisar antes de depender solo de esto para seguridad con el dominio ya expuesto a internet.
- **Copias de seguridad**: el volumen `cinecat_data` guarda la base de datos SQLite; haz copia periódica de ese volumen (`docker run --rm -v cinecat_data:/data -v $(pwd):/backup debian tar czf /backup/cinecat-db.tar.gz /data`) si te importa no perder usuarios y bibliotecas dadas de alta.
