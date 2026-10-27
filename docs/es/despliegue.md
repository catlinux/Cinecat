# Despliegue con Docker

[English](../en/deployment.md) · [Volver al README](../../README.md) · [Actualizaciones](actualizaciones.md)

Cinecat se despliega con Docker Compose: un contenedor para el servidor Go y otro para el cliente web, publicados solo en `127.0.0.1`. El Apache del Debian hace de proxy inverso, gestiona HTTPS y enruta `/api/` y `/healthz` al servidor.

## Requisitos

- Docker con el plugin `docker compose`.
- Git y acceso al repositorio.
- Apache con `mod_proxy`, `mod_proxy_http` y `mod_headers`.
- Para el actualizador automático: `sqlite3`, `curl`, `python3`, `flock`, usuario en el grupo `docker` y linger activo.

## Servidor nuevo

```sh
sudo mkdir -p /opt/cinecat
sudo chown "$USER":"$USER" /opt/cinecat
git clone https://github.com/catlinux/Cinecat.git /opt/cinecat
cd /opt/cinecat
cp .env.deploy.example .env
```

Edita `.env`:

- `CINECAT_MEDIA_PATH`: ruta del host que se monta completa y en solo lectura en `/media`.
- `CINECAT_MEDIA_ROOTS`: raíces visibles dentro del contenedor para elegir carpetas de biblioteca; por defecto `/media`.
- `CINECAT_TMDB_API_KEY`: clave de TMDB para fichas completas; si falta, se puede usar OMDb solo para sinopsis y cartel de películas.
- `CINECAT_OMDB_API_KEY`: opcional, respaldo sin TMDB.
- `CINECAT_SCAN_INTERVAL`: intervalo automático; `0` lo desactiva.
- `CINECAT_ADMIN_PASSWORD`: contraseña del `admin` del primer arranque.
- `CINECAT_API_PORT` y `CINECAT_WEB_PORT`: solo si 8080/8081 están ocupados.

```sh
mkdir -p run
docker compose up -d --build
docker compose logs servidor
```

El primer arranque crea el usuario `admin`. Si no fijaste contraseña, aparece una sola vez en el log: cópiala antes de que se pierda. El servidor queda en `http://127.0.0.1:8080` y la web en `http://127.0.0.1:8081`, accesibles solo desde el propio Debian.

## Apache y HTTPS

```sh
sudo cp deploy/video.warcrafted.com.conf /etc/apache2/sites-available/
sudo a2ensite video.warcrafted.com.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
sudo certbot --apache -d video.warcrafted.com
```

Certbot copia los `ProxyPass` al bloque `:443`, pero no cambia `X-Forwarded-Proto`. Dentro del bloque HTTPS, deja:

```apache
RequestHeader set X-Forwarded-Proto "https"
```

Recarga Apache. Sin esa cabecera la cookie de sesión no se marca `Secure`.

## Bibliotecas y primer escaneo

Abre la web, inicia sesión y ve a **Administración → Bibliotecas**. Crea una biblioteca por cada carpeta real dentro del montaje:

- `/media/Multimedia/Pelis` para películas;
- `/media/Multimedia/Series` para series.

Pulsa **Escanear todas las bibliotecas**. El escaneo sigue en segundo plano y el panel muestra el progreso; puedes cerrar la página. El contenedor no puede borrar ni subir archivos porque `/media` se monta `:ro`.

## Actualizar

Manual:

```sh
cd /opt/cinecat
./scripts/actualizar.sh
```

Hace `git pull --ff-only`, reconstruye las imágenes y levanta; la base de datos y las bibliotecas se conservan. La primera vez que quieras actualizar desde Administración → Sistema, instala el servicio del host:

```sh
cd /opt/cinecat
./scripts/instalar-actualizador.sh
```

El instalador comprueba requisitos, crea las unidades de systemd de usuario y avisa si falta linger o el acceso a Docker. Los detalles, copias y recuperación están en [actualizaciones](actualizaciones.md).

## Copias de seguridad

La base SQLite vive en el volumen `cinecat_data`. Una copia en caliente:

```sh
mkdir -p "$HOME/cinecat-backups"
docker cp cinecat-servidor-1:/data/cinecat.db "$HOME/cinecat-backups/cinecat.db"
docker cp cinecat-servidor-1:/data/cinecat.db-wal "$HOME/cinecat-backups/cinecat.db-wal" 2>/dev/null || true
docker cp cinecat-servidor-1:/data/cinecat.db-shm "$HOME/cinecat-backups/cinecat.db-shm" 2>/dev/null || true
sqlite3 "$HOME/cinecat-backups/cinecat.db" "VACUUM INTO '$HOME/cinecat-backups/cinecat-copia.db'"
```

El actualizador automático ya hace una copia antes de cada actualización y conserva las 10 últimas en `~/cinecat-backups/`.

## Cambiar de dominio o de servidor

1. Copia el repositorio y el `.env`; conserva el volumen `cinecat_data` o restaura una copia de la base.
2. Copia el vhost, cambia `ServerName` y ejecuta certbot para el dominio nuevo.
3. Ajusta `CINECAT_MEDIA_PATH` a la ruta real del nuevo host.
4. Levanta los contenedores y comprueba `/healthz`, `/api/v1/system/version` y la web.

## Límites del despliegue

- El servidor Go escucha HTTP plano en localhost; TLS lo pone Apache.
- No hay CORS: la web y la API deben compartir origen a través del proxy.
- El actualizador automático necesita el servicio del host; sin él, la web avisa de que no está instalado.
- La cuota de TMDB/OMDb y el TTL de caché de TMDB deben revisarse antes de un uso intensivo.
