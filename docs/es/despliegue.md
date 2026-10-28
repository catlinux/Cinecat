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
- `CINECAT_TRUSTED_PROXIES`: redes de confianza para `X-Forwarded-For`; por defecto loopback y redes privadas.
- `CINECAT_API_PORT` y `CINECAT_WEB_PORT`: solo si 8080/8081 están ocupados.

```sh
mkdir -p run
docker compose up -d --build
docker compose logs servidor
```

El primer arranque crea el usuario `admin`. Si no fijaste contraseña, aparece una sola vez en el log: cópiala antes de que se pierda. El servidor queda en `http://127.0.0.1:8080` y la web en `http://127.0.0.1:8081`, accesibles solo desde el propio Debian.

## IP dinámica con OVH DynHost

Si tu operador cambia la IP de casa, el dominio tiene que seguirla. Con un dominio en OVH se hace con **DynHost** ([guía oficial](https://docs.ovhcloud.com/en/guides/web-cloud/domains/dns-dynhost)). Es una pieza opcional y ajena a Cinecat: si tu router ya sabe actualizar DynHost (muchos admiten un servicio DynDNS personalizado), puedes usarlo y saltarte los pasos 2 y 3.

1. En el panel de OVH, **Web Cloud → DNS zones →** tu dominio **→ pestaña DynHost**. Crea primero el acceso (**Manage access → Create a username**: sufijo, subdominio —por ejemplo `media`— y contraseña) y después el registro (**Add a DynHost**: el mismo subdominio y tu IP pública actual). El subdominio no puede existir ya como registro A o AAAA de la zona.
2. En el Debian, guarda el identificador y la contraseña de DynHost (no los de tu cuenta de OVH) en un archivo que solo pueda leer tu usuario:

   ```sh
   mkdir -p ~/.config/systemd/user ~/.config/cinecat
   install -m 600 deploy/dynhost/dynhost.env.example ~/.config/cinecat/dynhost.env
   nano ~/.config/cinecat/dynhost.env
   ```

3. Instala el temporizador de usuario, que cada 5 minutos comprueba la IP y avisa a OVH solo si ha cambiado (o si hace más de 24 horas que no lo confirma). Necesita `linger` activado para el usuario, igual que el actualizador:

   ```sh
   sed "s|@REPO@|$PWD|" deploy/dynhost/cinecat-dynhost.service > ~/.config/systemd/user/cinecat-dynhost.service
   cp deploy/dynhost/cinecat-dynhost.timer ~/.config/systemd/user/
   systemctl --user daemon-reload
   systemctl --user enable --now cinecat-dynhost.timer
   ```

4. Comprueba que funciona: `systemctl --user start cinecat-dynhost.service` y `journalctl --user -u cinecat-dynhost.service -n 5` deben mostrar «OVH confirmó media.techmarc.es → tu IP». El cambio puede tardar unos minutos en propagarse; `getent hosts media.techmarc.es` debe devolver tu IP antes de pedir el certificado. Con `DYNHOST_DRY_RUN=1 deploy/dynhost/ovh-dynhost.sh` ves lo que haría sin llamar a OVH.

La contraseña solo vive en `~/.config/cinecat/dynhost.env` (el script se niega a arrancar si otros usuarios pueden leerlo) y no aparece en la lista de procesos. Para saber tu IP, el script consulta `api.ipify.org`, `ifconfig.me` e `icanhazip.com`, por este orden (cámbialos con `DYNHOST_IP_SERVICES`); es la única conexión saliente y la hace el script, no Cinecat.

## Apache y HTTPS

```sh
sudo cp deploy/media.techmarc.es.conf /etc/apache2/sites-available/
sudo a2ensite media.techmarc.es.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
sudo certbot --apache -d media.techmarc.es
```

Certbot copia los `ProxyPass` al bloque `:443`, pero no cambia `X-Forwarded-Proto`. Dentro del bloque HTTPS, deja:

```apache
RequestHeader set X-Forwarded-Proto "https"
```

Recarga Apache. Sin esa cabecera la cookie de sesión no se marca `Secure`.

### Cabeceras y firma del servidor

Las cabeceras de seguridad (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, la CSP mínima y HSTS en HTTPS) las añaden la web y la API. Apache no debe duplicarlas; el vhost las deja comentadas solo como referencia.

Para reducir la firma del servidor, edita `/etc/apache2/conf-available/security.conf` del host (no el vhost) y deja:

```apache
ServerTokens Prod
ServerSignature Off
```

Después recarga Apache. Estos ajustes no afectan a la reproducción ni a los proxies.

### IP real del cliente y bloqueo de login

El bloqueo de login cuenta los fallos por pareja usuario+IP, así que la API necesita la IP real y no la del contenedor web. `mod_proxy_http` de Apache añade `X-Forwarded-For` por sí solo (directiva `ProxyAddHeaders`, activa por defecto), así que normalmente no hace falta configurar nada en el vhost. No la desactives ni la sobrescribas con `RequestHeader set`.

Si hay un CDN u otro proxy delante de Apache, no sobrescribas la cadena: deja que cada proxy la añada y ajusta `XFF_DEPTH` al número de proxies de confianza que hay delante de la web. El contenedor web ya usa `ADDRESS_HEADER=X-Forwarded-For` y `XFF_DEPTH=1` en `docker-compose.yml`; con solo Apache ese valor es correcto, y con un proxy adicional debe ser `2`.

El servidor solo acepta `X-Forwarded-For` si la conexión directa viene de una red de `CINECAT_TRUSTED_PROXIES` (por defecto loopback y redes privadas, que incluye la red de Docker). Si cambias la red del contenedor web o pones la API detrás de otra red, añade su CIDR a esa variable. Puedes comprobar que Apache reenvía la cabecera mirando una petición en los logs de Apache o de la API; no des por hecho que tu configuración concreta ya lo hace sin comprobarlo.

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
5. La cookie de sesión pertenece al dominio, así que al cambiar de dominio todos los usuarios tendrán que volver a iniciar sesión. Mantén el vhost del dominio antiguo hasta comprobar el nuevo y después retíralo (`sudo a2dissite <vhost>`) o redirígelo al nuevo.

## Límites del despliegue

- El servidor Go escucha HTTP plano en localhost; TLS lo pone Apache.
- No hay CORS: la web y la API deben compartir origen a través del proxy.
- El actualizador automático necesita el servicio del host; sin él, la web avisa de que no está instalado.
- La cuota de TMDB/OMDb y el TTL de caché de TMDB deben revisarse antes de un uso intensivo.
