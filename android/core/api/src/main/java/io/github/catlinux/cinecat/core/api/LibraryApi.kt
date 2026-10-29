package io.github.catlinux.cinecat.core.api

private val TmdbImageSize = Regex("^https://image\\.tmdb\\.org/t/p/(?:w\\d+|original)/")

val DefaultPortalRows: List<PortalRow> = listOf(
    PortalRow("recent_movies", true, "Películas recién añadidas", "Estrenos"),
    PortalRow("continue_movies", true, "Continuar viendo películas", "En pausa"),
    PortalRow("watchlist_movies", true, "Películas por ver", "Para más tarde"),
    PortalRow("watched_movies", true, "Vistas hace poco", "Tu historial"),
    PortalRow("recent_series", true, "Series recién añadidas", "Estrenos"),
    PortalRow("continue_series", true, "Continuar viendo series", "En pausa"),
    PortalRow("next_up", true, "Siguiente episodio", "Series"),
    PortalRow("watchlist_series", true, "Series por ver", "Para más tarde"),
    PortalRow("favorites", true, "Tus favoritos", "Tu selección")
)

fun portalRows(settings: PublicSettings): List<PortalRow> = portalRows(settings.portalRows)

fun portalRows(raw: String?): List<PortalRow> {
    if (raw.isNullOrBlank()) {
        return DefaultPortalRows
    }
    val rows = runCatching {
        CinecatJson.decodeFromString<List<PortalRow>>(raw)
    }.getOrNull()
    return rows?.takeIf { it.isNotEmpty() } ?: DefaultPortalRows
}

fun tmdbImage(url: String?, size: String): String? {
    if (url == null) {
        return null
    }
    val match = TmdbImageSize.find(url) ?: return url
    return "https://image.tmdb.org/t/p/$size/" + url.substring(match.range.last + 1)
}
