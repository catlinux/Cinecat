// Package library implementa el escaneo de la biblioteca de vídeo y el
// análisis de nombres de archivo y carpeta.
package library

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Tipos de entrada, coherentes con el CHECK de media_items.kind.
const (
	KindMovie   = "movie"
	KindSeries  = "series"
	KindEpisode = "episode"
)

// Media es lo que se extrae de un nombre de archivo o de una ruta relativa.
type Media struct {
	Title        string
	SeriesFolder string
	Year         int
	Kind         string
	Season       int
	Episode      int
}

// El analizador es propio y no usa librerías de terceros: las alternativas
// conocidas (guessit, parse-torrent-title) están en Python/JS y no aportan
// nada como dependencia Go para los patrones que necesitamos.

var (
	reSeasonEpisode = regexp.MustCompile(`(?i)\b(?:season|temporada|temp)[\s._-]*(\d{1,2})[\s._-]*(?:episode|episodio|ep)[\s._-]*(\d{1,3})\b`)
	reSxxEyy        = regexp.MustCompile(`(?i)\bs(\d{1,2})[\s._-]*e(\d{1,3})(?:[\s._-]*e\d{1,3})?\b`)
	reNxM           = regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{2,3})\b`)
	reNxMGlued      = regexp.MustCompile(`(?i)(\d{1,2})x(\d{2})`)
	reThreeDigits   = regexp.MustCompile(`\d{3}`)
	reSeasonOnly    = regexp.MustCompile(`(?i)\b(?:season|temporada|temp)[\s._-]*(\d{1,2})\b`)
	reYear          = regexp.MustCompile(`\b(?:19|20)\d{2}\b`)

	reSeasonFolder      = regexp.MustCompile(`(?i)^(?:season|temporada|temp)[\s._-]*(\d{1,2})$`)
	reSeasonFolderShort = regexp.MustCompile(`(?i)^s(\d{1,2})$`)
	reLeadingEpisode    = regexp.MustCompile(`^(\d{1,3})\s*[-–—]\s*`)
	reBareEpisode       = regexp.MustCompile(`^\d{1,3}$`)
	reEpisodeLetter     = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:episode|episodio|ep|e)[\s._-]?(\d{1,3})\b`)

	reBracketGroup = regexp.MustCompile(`\[[^\]]*\]|\{[^}]*\}`)
	reEmptyParens  = regexp.MustCompile(`\(\s*\)`)
	reSpaces       = regexp.MustCompile(`\s+`)
	reResolution   = regexp.MustCompile(`^\d{3,4}[pi]$`)
	reBitDepth     = regexp.MustCompile(`^\d{1,2}bit$`)
	reGroupSuffix  = regexp.MustCompile(`^(.+)-([A-Z0-9]{2,})$`)
)

// Etiquetas de calidad, códec, audio y grupo que no forman parte del título.
var releaseTags = map[string]bool{
	"480p": true, "576p": true, "720p": true, "1080p": true, "1080i": true,
	"2048p": true, "2160p": true, "4320p": true, "4k": true, "8k": true,
	"uhd": true, "hd": true, "sd": true, "fullhd": true,
	"web": true, "webdl": true, "web-dl": true, "webrip": true, "web-rip": true,
	"bluray": true, "blu-ray": true, "brrip": true, "bdrip": true, "bdremux": true,
	"remux": true, "dvdrip": true, "dvd": true, "dvd5": true, "dvd9": true,
	"hdrip": true, "hdtv": true, "hdtvrip": true, "cam": true, "telesync": true,
	"hdcam": true, "hdts": true, "ts": true,
	"x264": true, "x265": true, "h264": true, "h265": true, "hevc": true,
	"avc": true, "xvid": true, "divx": true, "av1": true, "vp9": true,
	"aac": true, "aac2": true, "ac3": true, "eac3": true, "dts": true,
	"dtshd": true, "truehd": true, "flac": true, "mp3": true, "opus": true,
	"atmos": true, "dd": true, "ddp": true, "dd+": true,
	"hdr": true, "hdr10": true, "hdr10+": true, "sdr": true, "dv": true,
	"dolby": true, "dolbyvision": true, "10bit": true, "8bit": true,
	"proper": true, "repack": true, "extended": true, "unrated": true,
	"remastered": true, "limited": true, "internal": true, "directors": true,
	"director's": true,
	"multi":      true, "dual": true, "dubbed": true, "subbed": true, "subs": true,
	"sub": true, "vose": true, "vo": true, "spa": true, "eng": true,
	"castellano": true, "espanol": true, "español": true, "latino": true,
	"spanish": true, "english": true,
	"3d": true, "hsbs": true, "sbs": true, "sample": true,
}

var releaseTagPrefixes = []string{
	"x264", "x265", "h264", "h265", "hevc", "web", "bluray", "brrip", "bdrip",
	"hdtv", "dvdrip", "hdrip", "remux", "aac", "ac3", "eac3", "dts", "ddp",
	"truehd", "atmos", "hdr", "amzn", "dsnp", "hmax", "atvp",
}

var junkFolders = map[string]bool{
	"extras": true, "extra": true, "special features": true, "featurettes": true,
	"sample": true, "samples": true, "subs": true, "subtitles": true,
	"artwork": true, "backdrops": true,
}

// Una temporada de 1 a 30 es plausible para una serie; si el nombre trae otra,
// nos quedamos con la temporada de la carpeta.
const maxPlausibleSeason = 30

// Números que suelen ser resolución o códec, no episodios.
var nonEpisodeNumbers = map[int]bool{
	264: true, 265: true, 480: true, 720: true, 1080: true,
}

// ParsePath analiza la ruta relativa de un archivo dentro de una biblioteca,
// usando también las carpetas (serie, temporada, año) cuando el nombre del
// archivo no basta.
func ParsePath(relPath string) Media {
	clean := filepath.Clean(relPath)
	dir, file := filepath.Split(clean)
	media := ParseFilename(file)

	dirs := splitDirs(dir)
	seriesFolder := ""
	if len(dirs) > 0 {
		first := dirs[0]
		if !junkFolders[strings.ToLower(first)] {
			if _, isSeason := seasonFromFolder(first); !isSeason {
				seriesFolder = first
			}
		}
	}
	season, seriesDir := 0, ""
	for i := len(dirs) - 1; i >= 0; i-- {
		name := dirs[i]
		if junkFolders[strings.ToLower(name)] {
			continue
		}
		if n, ok := seasonFromFolder(name); ok {
			if season == 0 {
				season = n
			}
			continue
		}
		seriesDir = name
		break
	}

	base := strings.TrimSuffix(file, filepath.Ext(file))

	if media.Episode == 0 && season > 0 {
		// "ChicagoFire720p4x17" y "Serie.x264.2x05" dentro de una temporada.
		if episodeSeason, episode, ok := firstGluedNxM(base, season); ok {
			media.Kind = KindEpisode
			media.Season = episodeSeason
			media.Episode = episode
		} else if episode, ok := firstThreeDigitEpisode(base, season); ok {
			media.Kind = KindEpisode
			media.Season = season
			media.Episode = episode
		} else {
			// "04 - Título.mkv", "04.mkv" o "Serie - E04.mkv" dentro de una temporada.
			switch {
			case reLeadingEpisode.MatchString(base):
				raw := reLeadingEpisode.FindStringSubmatch(base)[1]
				if isIgnoredThreeDigitNumber(raw) {
					break
				}
				episode, _ := strconv.Atoi(raw)
				media.Kind = KindEpisode
				media.Season = season
				media.Episode = episode
				rest := reLeadingEpisode.ReplaceAllString(base, "")
				if seriesDir != "" {
					media.Title = folderTitle(seriesDir)
				} else {
					media.Title = ParseFilename(rest).Title
				}
			case reBareEpisode.MatchString(strings.TrimSpace(base)):
				raw := strings.TrimSpace(base)
				if isIgnoredThreeDigitNumber(raw) {
					break
				}
				episode, _ := strconv.Atoi(raw)
				media.Kind = KindEpisode
				media.Season = season
				media.Episode = episode
				if seriesDir != "" {
					media.Title = folderTitle(seriesDir)
				}
			default:
				if m := reEpisodeLetter.FindStringSubmatch(base); m != nil {
					episode, _ := strconv.Atoi(m[1])
					media.Kind = KindEpisode
					media.Season = season
					media.Episode = episode
					if seriesDir != "" {
						media.Title = folderTitle(seriesDir)
					}
				}
			}
		}
	}

	if media.Kind == KindEpisode {
		if media.Season == 0 {
			media.Season = season
		}
		if seriesFolder != "" {
			media.SeriesFolder = seriesFolder
			if title := folderTitle(seriesFolder); title != "" {
				media.Title = title
			}
			if media.Year == 0 {
				media.Year = ParseFilename(seriesFolder).Year
			}
		} else if seriesDir != "" {
			if media.Title == "" {
				media.Title = folderTitle(seriesDir)
			}
			if media.Year == 0 {
				media.Year = ParseFilename(seriesDir).Year
			}
		}
	}

	// Película dentro de su propia carpeta: "El Orfanato (2007)/El Orfanato.mkv".
	if media.Kind == KindMovie && media.Year == 0 && seriesDir != "" {
		folder := ParseFilename(seriesDir)
		if folder.Year > 0 && strings.EqualFold(folder.Title, media.Title) {
			media.Year = folder.Year
		}
	}

	if media.Title == "" {
		media.Title = cleanTitle(reBracketGroup.ReplaceAllString(file, " "))
	}
	if media.Title == "" {
		media.Title = strings.TrimSpace(base)
	}
	if media.Kind == "" {
		media.Kind = KindMovie
	}
	return media
}

// ParseFilename analiza el nombre de un archivo de vídeo (con o sin extensión).
func ParseFilename(name string) Media {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	text := normalizeSeparators(base)

	episode, hasEpisode := findEpisode(text)
	seasonOnly, seasonStart, hasSeason := findSeason(text)
	year, yearStart := findYear(text)

	cut := len(text)
	if hasEpisode && episode.start < cut {
		cut = episode.start
	}
	if yearStart >= 0 && yearStart < cut {
		cut = yearStart
	}
	if hasSeason && seasonStart < cut {
		cut = seasonStart
	}

	media := Media{Title: cleanTitle(text[:cut]), Year: year}
	switch {
	case hasEpisode:
		media.Kind = KindEpisode
		media.Season = episode.season
		media.Episode = episode.episode
	case hasSeason:
		media.Kind = KindSeries
		media.Season = seasonOnly
	default:
		media.Kind = KindMovie
	}
	return media
}

type episodeMatch struct {
	season  int
	episode int
	start   int
}

func findEpisode(text string) (episodeMatch, bool) {
	var best episodeMatch
	found := false

	consider := func(re *regexp.Regexp) {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if found && m[0] >= best.start {
				continue
			}
			season, err1 := strconv.Atoi(text[m[2]:m[3]])
			episode, err2 := strconv.Atoi(text[m[4]:m[5]])
			if err1 != nil || err2 != nil {
				continue
			}
			best = episodeMatch{season: season, episode: episode, start: m[0]}
			found = true
		}
	}

	consider(reSeasonEpisode)
	consider(reSxxEyy)
	consider(reNxM)
	return best, found
}

// firstGluedNxM busca un NxMM pegado a letras o números. Si la temporada del
// nombre no coincide con la carpeta, manda la del nombre cuando es creíble.
func firstGluedNxM(text string, folderSeason int) (season, episode int, ok bool) {
	for _, m := range reNxMGlued.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[0], m[1]
		if start > 0 && isASCIIDigit(text[start-1]) {
			continue
		}
		if end < len(text) && isASCIIDigit(text[end]) {
			continue
		}
		nameSeason, err1 := strconv.Atoi(text[m[2]:m[3]])
		nameEpisode, err2 := strconv.Atoi(text[m[4]:m[5]])
		if err1 != nil || err2 != nil || nameSeason <= 0 || nameEpisode <= 0 {
			continue
		}
		if nameSeason == folderSeason || nameSeason <= maxPlausibleSeason {
			return nameSeason, nameEpisode, true
		}
		return folderSeason, nameEpisode, true
	}
	return 0, 0, false
}

// firstThreeDigitEpisode acepta un número de tres cifras aislado por
// separadores: la primera cifra es la temporada y las dos últimas el episodio.
func firstThreeDigitEpisode(text string, folderSeason int) (episode int, ok bool) {
	for _, loc := range reThreeDigits.FindAllStringIndex(text, -1) {
		start, end := loc[0], loc[1]
		if start > 0 && isASCIIDigitOrLetter(text[start-1]) {
			continue
		}
		if end < len(text) && isASCIIDigitOrLetter(text[end]) {
			continue
		}
		value, err := strconv.Atoi(text[start:end])
		if err != nil || nonEpisodeNumbers[value] || value/100 != folderSeason {
			continue
		}
		episode = value % 100
		if episode > 0 {
			return episode, true
		}
	}
	return 0, false
}

func isIgnoredThreeDigitNumber(raw string) bool {
	if len(raw) != 3 {
		return false
	}
	value, err := strconv.Atoi(raw)
	return err == nil && nonEpisodeNumbers[value]
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isASCIIDigitOrLetter(b byte) bool {
	return isASCIIDigit(b) || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func findSeason(text string) (season, start int, ok bool) {
	if m := reSeasonOnly.FindStringSubmatchIndex(text); m != nil {
		if n, err := strconv.Atoi(text[m[2]:m[3]]); err == nil && n > 0 {
			return n, m[0], true
		}
	}
	return 0, 0, false
}

// findYear devuelve el último año plausible del texto. Se descarta un año al
// principio del nombre ("1917.mkv", "2012.mkv"), que suele ser parte del
// título, y las resoluciones tipo 2010p.
func findYear(text string) (year, start int) {
	year, start = 0, -1
	for _, m := range reYear.FindAllStringIndex(text, -1) {
		if m[0] == 0 {
			continue
		}
		after := text[m[1]:]
		if len(after) > 0 && (after[0] == 'p' || after[0] == 'i' || after[0] == 'P' || after[0] == 'I') {
			continue
		}
		if len(after) > 1 && (after[0] == 'x' || after[0] == 'X') && after[1] >= '0' && after[1] <= '9' {
			continue
		}
		if m[0] > 0 && (text[m[0]-1] == 'x' || text[m[0]-1] == 'X') {
			continue
		}
		v, err := strconv.Atoi(text[m[0]:m[1]])
		if err != nil {
			continue
		}
		year, start = v, m[0]
	}
	return year, start
}

func normalizeSeparators(s string) string {
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return strings.TrimSpace(reSpaces.ReplaceAllString(s, " "))
}

func cleanTitle(s string) string {
	s = reBracketGroup.ReplaceAllString(s, " ")
	s = reEmptyParens.ReplaceAllString(s, " ")

	var kept []string
	for _, token := range strings.Fields(s) {
		if isReleaseTag(token) {
			break
		}
		kept = append(kept, token)
	}

	title := strings.Join(kept, " ")
	title = reSpaces.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "-–—_.:;,()[]{}")
	title = strings.TrimSpace(title)

	// Grupo de release pegado con guion: "Movie-GROUP" -> "Movie".
	if m := reGroupSuffix.FindStringSubmatch(title); m != nil {
		title = strings.TrimSpace(m[1])
	}
	return title
}

func isReleaseTag(token string) bool {
	t := strings.ToLower(strings.Trim(token, "[](){},;:!\"'"))
	if t == "" {
		return false
	}
	if releaseTags[t] {
		return true
	}
	if reResolution.MatchString(t) || reBitDepth.MatchString(t) {
		return true
	}
	for _, prefix := range releaseTagPrefixes {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

func splitDirs(dir string) []string {
	var dirs []string
	for _, part := range strings.Split(filepath.ToSlash(dir), "/") {
		if part == "" || part == "." || part == ".." {
			continue
		}
		dirs = append(dirs, part)
	}
	return dirs
}

func seasonFromFolder(name string) (int, bool) {
	if m := reSeasonFolder.FindStringSubmatch(name); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n, true
		}
	}
	if m := reSeasonFolderShort.FindStringSubmatch(name); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func folderTitle(folder string) string {
	title := ParseFilename(folder).Title
	if title == "" {
		title = cleanTitle(folder)
	}
	if title == "" {
		title = strings.TrimSpace(folder)
	}
	return title
}
