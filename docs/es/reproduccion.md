# Reproducción y códecs

[English](../en/playback.md) · [Volver al README](../../README.md)

> Estado: **diseño**. Las cifras son orientativas y deben medirse en la práctica.

## Idea central

Cada dispositivo **declara qué sabe hacer** y el servidor **elige la forma más barata** de reproducir cada vídeo sin perder calidad ni fluidez.

## Perfil de capacidades del cliente

Al conectarse, el cliente envía un perfil con, entre otros:

- códecs de vídeo que descodifica (H.264, H.265, VP9, AV1…), perfiles y niveles, 10 bits, HDR;
- códecs de audio (AAC, AC3, EAC3, Opus, DTS…) y canales;
- contenedores y formatos de subtítulos que soporta;
- resolución máxima y ancho de banda disponible;
- **preferencia de descodificación:** por hardware del dispositivo, por software del dispositivo o delegada en el servidor.

Cómo obtenerlo: en Android, consultando los códecs del sistema; en el navegador, con las APIs de capacidades de medios y comprobaciones de formato.

## Cadena de decisión

1. **Reproducción directa:** el cliente entiende vídeo, audio, contenedor y subtítulos. El servidor solo envía el archivo (con soporte de saltos en la línea de tiempo). Coste de servidor casi nulo.
2. **Cambio de contenedor (remux):** el vídeo es compatible pero el contenedor no (por ejemplo MKV en un navegador). El servidor reempaqueta al vuelo (HLS con fMP4) sin recodificar vídeo.
3. **Transcodificación parcial:** el vídeo se copia y solo se convierte el audio (por ejemplo DTS a AAC/AC3).
4. **Transcodificación completa:** el servidor recodifica el vídeo (con aceleración por hardware si existe) y adapta resolución y tasa de bits.

Antes de remuxear, el servidor comprueba con `ffprobe` el intervalo entre keyframes del primer flujo de vídeo. Si supera **15 s** (habitual en AVI/DivX antiguos), no intenta el remux sin recodificar: el primer segmento HLS podría ser enorme y tardar mucho en empezar a reproducirse. En su lugar devuelve un error claro. La transcodificación real de esos archivos queda para el Paso 2 de la Fase 3.

El servidor **prefiere H.265** cuando el cliente lo soporta (menos espacio y ancho de banda) y **recurre a H.264** cuando no.

## Quién descodifica

| Situación | Decisión |
|---|---|
| Móvil o tableta potente con descodificador H.265 por hardware | Reproducción directa; descodifica el propio aparato. |
| Televisor flojo o sin soporte del códec | El servidor transcodifica (H.264) y el televisor solo muestra. |
| Cliente capaz pero con descodificación por hardware fallida | El usuario puede forzar software en el cliente (por ejemplo libmpv) o delegar en el servidor. |
| Ancho de banda limitado (fuera de casa) | Transcodificación con tasa de bits adaptada. |

La preferencia puede fijarse por dispositivo y por usuario.

## Navegadores y H.265

No se puede dar por hecho. A fecha 2026-10-24, según las fuentes consultadas:

- Chrome admite H.265 con **descodificación por hardware**; en Linux depende de la GPU y de la configuración.
- Firefox lo admite con hardware en Windows (desde 134), macOS (136) y Linux (137), con limitaciones iniciales de contenedor.
- Los resultados reales varían entre equipos.

Por eso el cliente web **comprueba** si puede reproducir el vídeo y el servidor **cae a H.264** cuando no. Los MKV se sirven por HLS con segmentos fMP4, que sí soportan H.265.

## Casos que obligan a transcodificar

- Vídeo de 10 bits en dispositivos que solo aceptan 8 bits.
- HDR / Dolby Vision en pantallas SDR (conversión a SDR con *tone mapping* si FFmpeg lo permite).
- Subtítulos de imagen (PGS/VobSub) en clientes que no los muestran (se «queman» en el vídeo).
- Audio no soportado (DTS, TrueHD).

## Subtítulos, audio y capítulos

- Subtítulos de texto: se envían como texto y los pinta el cliente (con soporte de estilos cuando sea posible).
- Subtítulos de imagen: se muestran si el cliente puede, o se queman en el vídeo.
- Selección de pista de audio y subtítulos por idioma, con recuerdo de la elección.
- Capítulos y marcas en la línea de tiempo.

## Sesiones

- Control de **transcodificaciones simultáneas** para no saturar el servidor.
- Limpieza de archivos temporales.
- Registro de por qué se eligió cada modo (útil para depurar y para mostrarlo al usuario).

## Relación con Reduvid

Convertir antes los vídeos con [Reduvid](https://github.com/catlinux/reduvid) a un formato compatible reduce la necesidad de transcodificar en el servidor.
