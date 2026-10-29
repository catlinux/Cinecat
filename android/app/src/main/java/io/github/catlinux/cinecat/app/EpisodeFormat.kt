package io.github.catlinux.cinecat.app

fun formatEpisodeCode(season: Int?, episode: Int?): String {
    if (season == null && episode == null) {
        return ""
    }
    val seasonPart = season?.toString()?.padStart(2, '0') ?: "--"
    val episodePart = episode?.toString()?.padStart(2, '0') ?: "--"
    return "S${seasonPart}E${episodePart}"
}
