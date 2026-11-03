# Complementos (diseño)

[English version](../en/plugins.md)

> **Estado: propuesta, sin implementar.** Este documento fija qué podría extender un complemento y con qué límites. No se implementa nada hasta que el autor lo apruebe.

## Objetivo

Permitir que un administrador añada funciones que no deben vivir en el núcleo (otras fuentes de metadatos, avisos, tareas tras un escaneo) **sin cargar código ajeno dentro del servidor** y sin abrir conexiones que el administrador no haya autorizado.

## Qué no es

- No es una tienda ni un repositorio de complementos: Cinecat no descarga nada por su cuenta (regla del proyecto: ninguna conexión saliente salvo las fuentes de metadatos configuradas).
- No son plugins de Go (`plugin.Open`): obligan a compilar con la misma versión exacta, no funcionan en todas las plataformas y comparten memoria y permisos con el servidor.
- No son scripts dentro de la web ni de las apps: los clientes no ejecutan código de complementos.

## Modelo: proceso externo con un protocolo JSON

Un complemento es **un ejecutable** que el administrador copia en `/data/plugins/<nombre>/` junto a un manifiesto `plugin.json`. El servidor lo lanza como proceso hijo y habla con él por **entrada y salida estándar**, una petición JSON por línea y una respuesta JSON por línea (como un servidor de lenguaje, pero mínimo).

```json
{
  "name": "avisos-telegram",
  "version": "1.0.0",
  "protocol": 1,
  "command": ["./avisos-telegram"],
  "hooks": ["scan.finished", "item.added"],
  "network": ["api.telegram.org:443"],
  "settings": [{ "key": "token", "label": "Token del bot", "secret": true }]
}
```

- `hooks`: eventos a los que se suscribe. El servidor solo le envía esos.
- `network`: destinos que el complemento declara necesitar. El administrador los ve antes de activarlo. En Linux, el servidor puede aplicarlos con el espacio de red del proceso (ver «Aislamiento»); en otros sistemas, la declaración es informativa y se avisa de ello.
- `settings`: ajustes que se editan en Administración → Complementos. Los marcados como `secret` se guardan cifrados como los demás secretos y nunca se devuelven a la web.

### Ciclo de vida

1. El administrador copia la carpeta y pulsa «Buscar complementos». El servidor lee los manifiestos; **ninguno se activa solo**.
2. Al activarlo, la web muestra qué eventos recibe, a qué destinos de red accede y qué datos verá. Se guarda la huella SHA-256 del ejecutable: si el archivo cambia, el complemento se desactiva hasta que se vuelva a aprobar.
3. El servidor lanza el proceso al arrancar (o al activarlo), le envía `init` con los ajustes y lo reinicia con espera creciente si se cae. Tras varios fallos seguidos lo desactiva y lo deja en el registro de actividad.

### Eventos (versión 1 del protocolo)

| Evento | Datos | Respuesta |
|---|---|---|
| `init` | ajustes, versión de Cinecat | `ok` o error |
| `scan.finished` | biblioteca, ítems nuevos, cambiados y borrados (solo id, tipo y título) | ninguna |
| `item.added` | id, tipo, título, año | ninguna |
| `playback.started` / `playback.stopped` | usuario (solo nombre), ítem, dispositivo | ninguna |
| `metadata.lookup` | título, año, tipo, ids externos conocidos | campos de metadatos propuestos |

- Los eventos son **asíncronos y con tiempo límite** (5 s por defecto, 30 s para `metadata.lookup`). Un complemento lento o caído nunca bloquea un escaneo ni una reproducción.
- `metadata.lookup` solo **propone** datos. Se aplican con la misma precedencia que hoy (TMDB > OMDb), como una fuente más que el administrador ordena, y nunca pisan una corrección manual.
- **Nunca** se envían rutas del disco, tokens, contraseñas ni direcciones IP a un complemento.

## Aislamiento

- El proceso corre con el mismo usuario del servidor pero sin heredar variables de entorno (solo `PATH`, `HOME` apuntando a su carpeta y `LANG`), con su carpeta de trabajo como único lugar de escritura previsto y límites de CPU y memoria (`setrlimit` en Linux).
- En el contenedor Docker oficial, los complementos van en el mismo contenedor; quien quiera más aislamiento puede ejecutarlos en otro contenedor y usar el modo «socket» (versión 2 del protocolo, no incluida en la primera entrega).
- Riesgo aceptado y escrito en la interfaz: **un complemento es código que el administrador decide ejecutar en su servidor**. Cinecat limita lo que le envía y lo que puede tumbar, pero no lo convierte en código de confianza.

## API

- `GET /api/v1/plugins` (admin): lista de complementos detectados con estado, huella, eventos, red y último error.
- `POST /api/v1/plugins/{name}/enable` y `/disable` (admin).
- `PUT /api/v1/plugins/{name}/settings` (admin): los secretos se envían pero nunca se devuelven.
- Los complementos **no** añaden rutas propias a la API ni páginas a la web en la versión 1.

## Entrega propuesta

1. Servidor: detección de manifiestos, activación con huella, proceso hijo con protocolo v1, eventos `init`, `scan.finished` e `item.added`, registro de actividad. Un complemento de ejemplo en `examples/plugins/registro/` que escribe los eventos en un archivo.
2. Web: Administración → Complementos (lista, activar o desactivar, ajustes, último error).
3. `metadata.lookup` como fuente de metadatos ordenable.
4. Eventos de reproducción.

## Preguntas para el autor antes de implementar

- ¿Hay un caso concreto que quieras cubrir primero (avisos, otra fuente de metadatos, otra cosa)? Eso decide el orden de la entrega.
- ¿Te vale que los complementos solo se instalen copiando la carpeta a mano, sin instalador en la web?
