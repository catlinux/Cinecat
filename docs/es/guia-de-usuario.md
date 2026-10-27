# Guía de usuario

[English](../en/user-guide.md) · [Volver al README](../../README.md) · [Estado del proyecto](estado.md)

Esta guía explica cómo usar la web de Cinecat. Está pensada para una persona que solo quiere ver su biblioteca; la administración se explica en [administración](administracion.md).

## Entrar y salir

1. Abre la dirección de tu Cinecat (por ejemplo, `https://video.warcrafted.com`).
2. En la pantalla «Entrar», escribe tu **Usuario** y **Contraseña** y pulsa «Entrar en la sala».
3. Si los datos son correctos, llegarás al portal. Si fallas varias veces, el servidor puede bloquear temporalmente el intento: espera unos minutos y vuelve a probar.

Para cerrar sesión, entra en `/login` y pulsa «Cerrar sesión». La sesión se guarda en una cookie `httpOnly`; el navegador no guarda el token en `localStorage`.

## El portal

El portal es la página de inicio. Arriba verás el nombre del servidor, el eslogan y el texto de bienvenida que haya configurado el administrador. Después aparecen los accesos a **Películas**, **Colecciones**, **Series** y, si existe una biblioteca de Vídeos con contenido, **Vídeos**.

Debajo se muestran, solo si tienen contenido:

- **Continuar viendo**: lo que dejaste a medias.
- **Siguiente episodio**: el próximo capítulo de tus series.
- **Añadido recientemente**: las películas más nuevas de la biblioteca.

Las filas se pueden desplazar con las flechas `‹` y `›`; con el dedo o el trackpad también puedes deslizar.

## Navegar por la biblioteca

El menú puede estar a la izquierda (**lateral**) o arriba (**superior**), según lo configure el administrador. Contiene **Inicio**, **Películas**, **Colecciones**, **Series**, **Vídeos** cuando hay una biblioteca de ese tipo con contenido, **Mi cuenta** y, si eres administrador, **Administración**.

### Películas

La pantalla «Películas» muestra la cartelera paginada. Puedes:

- buscar por **Todo**, **Título**, **Reparto** o **Dirección**;
- filtrar por **Género**;
- elegir el **Orden**: Título, Año o Recientes;
- filtrar por **Visionado**: Todas, Sin ver o Vistas;
- pulsar una letra (`#` o `A`–`Z`) para filtrar por inicial;
- quitar un filtro con su `×` o con «Limpiar filtros».

El número de resultados por página depende de **columnas × filas** configurado en administración (por defecto 6 × 8 = 48). La paginación aparece cuando hay más de una página.

Si eres administrador verás además «Sin identificar (N)», que muestra las películas que el escaneo no ha podido identificar.

### Colecciones

«Colecciones» agrupa sagas y ciclos. Puedes buscar por nombre y ordenar por **Nombre**, **Añadidas** o **Tamaño**. Al abrir una colección verás su descripción, sus películas y un botón para reproducir la primera sin ver (o la primera, si ya están todas vistas).

Si eres administrador, dentro de una colección aparece «Gestionar colección» para cambiar nombre, descripción, cartel o fondo, añadir o quitar películas, reordenarlas y borrar la colección.

### Series

«Series» muestra las series paginadas. Puedes buscar, filtrar por género, elegir orden y filtrar por letra. Al abrir una serie verás su ficha y el selector de temporadas; cada temporada muestra sus episodios con título, sinopsis, imagen y duración.

Los botones de visionado permiten marcar un episodio, una temporada o la serie completa como vistos. El botón principal de la serie cambia según tu progreso: «Empezar», «Continuar» o «Volver a ver». Si eres administrador, al final de la ficha aparecen **Cambiar imagen** y **Corregir ficha**.

### Vídeos

La sección **Vídeos** aparece en el menú cuando existe una biblioteca de tipo Vídeos con contenido. Para crearla, entra en **Administración → Bibliotecas**, elige el tipo «Vídeos (videoclips, conciertos…)», añade sus carpetas y lanza el escaneo.

Este tipo no identifica los archivos ni descarga carátulas de TMDB u OMDb: el título sale del nombre del archivo. Al abrir la sección verás las carpetas de la biblioteca, con el número de vídeos que contienen (incluidas sus subcarpetas). Dentro de una carpeta, las subcarpetas aparecen primero y después los vídeos en tarjetas apaisadas con una portada de respaldo generada; las migas de pan permiten volver a cualquier nivel. Puedes reproducir o continuar un vídeo, marcarlo como visto y cambiar audio, subtítulos o calidad. Todavía no hay miniaturas de fotograma.

## La ficha de una película

La ficha muestra el cartel, el título (y el original si es distinto), año, duración, géneros, nota, lema, sinopsis, reparto y equipo, enlaces a IMDb/TMDB y las colecciones a las que pertenece. Al final está **Más como este**.

- **Reproducir** empieza desde el principio; **Continuar** retoma donde lo dejaste. **Desde el principio** ignora el progreso guardado.
- **Marcar como vista** / **Desmarcar** cambia el estado de visionado y borra el progreso.
- El panel **Reproducción** permite elegir **Audio**, **Subtítulos** y **Calidad**. La elección se guarda para esa película y ese usuario.
- El panel **Archivo** muestra los datos técnicos y cómo se reproducirá.
- Si eres administrador, al final aparecen **Cambiar imagen** y **Corregir ficha**.

## La ficha de un episodio

La ficha del episodio muestra la miga de pan `Series › Serie › Temporada N › Episodio N`, el título, la sinopsis, la duración, el reparto y los datos del archivo. Puedes marcar el episodio como visto, continuar donde lo dejaste y pasar al **Episodio anterior** o **Siguiente episodio**.

## La ficha de un vídeo

La ficha de un vídeo de la sección Vídeos es mínima: el título, **Volver a Vídeos**, reproducir o continuar, marcar como visto, las opciones de reproducción y el panel del archivo. No muestra sinopsis, reparto, colecciones, enlaces a IMDb/TMDB, «Más como este», «Corregir ficha» ni «Cambiar imagen», porque estos vídeos no se identifican.

## Reproducir

El reproductor tiene controles propios. Estos son los atajos de teclado:

| Tecla | Acción |
|---|---|
| `Espacio` o `K` | Reproducir o pausar |
| `←` o `J` | Retroceder 10 s (`Mayús`: 30 s) |
| `→` o `L` | Avanzar 10 s (`Mayús`: 30 s) |
| `↑` / `↓` | Subir o bajar el volumen |
| `M` | Silenciar o activar sonido |
| `F` | Pantalla completa |
| `C` | Activar o desactivar subtítulos |
| `,` / `.` | Fotograma anterior / siguiente, solo en pausa |
| `0`–`9` | Saltar al 0 %, 10 %, …, 90 % |
| `Esc` | Cerrar menús o cancelar la cuenta atrás |

Los botones visibles permiten reproducir/pausar, retroceder o avanzar 10 s, cambiar el volumen, elegir velocidad (0,5× a 2×), audio, subtítulos y calidad, activar imagen en imagen, pantalla completa y pasar al siguiente episodio. El clic en el vídeo reproduce o pausa; el doble clic alterna pantalla completa.

El progreso se guarda cada 15 s y al salir. Si quedan 15 s o menos, no se reanuda: se considera terminado. Al acabar un episodio con siguiente, aparece una cuenta atrás de 8 s para reproducirlo.

## Temas, menú y cuenta

El selector de tema tiene **Auto**, **Claro** y **Oscuro**. Auto sigue el tema del sistema. La preferencia se guarda en ese navegador; si no hay ninguna, se usa el tema por defecto del servidor.

En **Mi cuenta** puedes cambiar tu contraseña: pide la actual, la nueva y la repetición; la nueva debe tener al menos 8 caracteres. La sesión abierta en ese dispositivo sigue funcionando.

## Glosario

- **Biblioteca:** una carpeta del servidor que Cinecat recorre para encontrar vídeos.
- **Escaneo:** el proceso que busca archivos, analiza sus nombres y rellena fichas.
- **Ficha:** la página con los datos de una película, serie o episodio.
- **Identificar:** encontrar la ficha correcta en TMDB (o OMDb).
- **Transcodificación:** recodificar el vídeo o el audio para que el reproductor pueda entenderlo.
- **HLS:** formato de reproducción por segmentos que usa el reproductor web cuando el archivo original no se puede servir tal cual.
- **Reproducción directa:** enviar el archivo original sin recodificar.
- **WebVTT:** formato de subtítulos que el navegador puede mostrar.

## Si algo no funciona

- Si la página vuelve a «Entrar», la sesión ha caducado: inicia sesión otra vez.
- Si una película no aparece, comprueba con un administrador que la biblioteca existe y que el escaneo ha terminado.
- Si el vídeo no empieza, prueba otra calidad o avisa al administrador: el servidor puede no tener FFmpeg o el archivo puede usar un formato no admitido.
- Si ves el aviso de versiones distintas, recarga la página; si persiste, el administrador debe actualizar la web o el servidor.
