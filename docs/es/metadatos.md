# Metadatos

[English](../en/metadata.md) · [Volver al README](../../README.md)

> Estado: **diseño**. Las condiciones de cada fuente se verificaron a 2026-10-24 y **deben revisarse** antes de implementar, porque pueden cambiar.

## Principio

No creamos una base de datos propia: usamos las que ya existen, en **español de España (`es-ES`)**.

## Fuentes

| Fuente | Uso | Condiciones relevantes |
|---|---|---|
| **TMDB** | Fuente principal de películas y series: títulos, sinopsis, reparto, carteles, fondos, certificaciones por país. | Gratuita para uso **no comercial**, con atribución (logotipo y aviso) y con límite de tiempo para conservar los datos en caché. Uso comercial requiere licencia. |
| **TheTVDB** | Complemento para series (episodios, traducciones). | Gratuita para proyectos pequeños con atribución, o mediante suscripción por usuario. Se reserva el derecho de cambiar condiciones. |
| **Fanart.tv** | Arte adicional (logotipos, fondos). | Requiere clave de proyecto; la clave personal es opcional. |
| **TVmaze** | Opcional para calendarios de series. | Datos con licencia CC BY-SA 4.0 (compartir igual). |
| **OMDb** | Opcional para valoraciones. | Límite diario en el plan gratuito. |

**No** se usa FilmAffinity: no tiene API oficial y solo se podría obtener rascando la web.

## Identificación de archivos

1. Intentar leer **metadatos locales** si existen (por ejemplo archivos `.nfo` y carátulas junto al vídeo).
2. Analizar el **nombre del archivo y de las carpetas** para extraer título, año, temporada y episodio (con una biblioteca de análisis de nombres, a elegir en la Fase 0 según licencia y calidad).
3. Buscar en TMDB con ese título y año, en `es-ES`.
4. Si hay duda, ofrecer al usuario una **corrección manual** (elegir el resultado correcto) y recordarla.

## Estrategia de idioma e imágenes

Un problema conocido en otros servidores es que, si falta una imagen en el idioma principal, se queda sin cartel o con uno sin texto. En Cinecat el servidor elige **él mismo** la prioridad:

1. Cartel en `es-ES`.
2. Cartel en español genérico.
3. Cartel en inglés.
4. Cartel sin idioma.

Se pide a la fuente imágenes en varios idiomas a la vez y se aplica este orden, que el usuario podrá cambiar.

## Caché

- Se guardan localmente los datos e imágenes necesarios para funcionar sin conexión.
- Se **refrescan** periódicamente respetando el límite de conservación de cada fuente (TMDB: revisar el plazo vigente antes de implementar).

## Atribución

- Pantalla «Acerca de» con el logotipo oficial de TMDB sin alterar y el aviso: *This product uses the TMDB API but is not endorsed or certified by TMDB.*
- El logotipo de TMDB debe ser menos prominente que el de Cinecat.
- Créditos de las demás fuentes en [CREDITS.md](../../CREDITS.md).

## Certificaciones por edades

Se usarán las certificaciones de **España** cuando la fuente las ofrezca, para filtrar por perfil (por ejemplo, perfil infantil).
