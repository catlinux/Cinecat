package library

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/catlinux/cinecat/internal/metadata"
)

// MatchThreshold es la puntuación mínima para dar una película por
// identificada. Por debajo queda «sin identificar»: es preferible a ponerle la
// ficha de otra película.
//
// La puntuación y sus pesos se ajustaron contra las 870 películas reales de la
// biblioteca de producción (títulos en castellano, casi nunca con año) y los
// resultados de TMDB; testdata/match_fixture.json conserva los casos difíciles.
const MatchThreshold = 0.75

// MovieSearcher es lo que necesita la identificación de TMDB.
type MovieSearcher interface {
	SearchMovies(ctx context.Context, query string, year int) ([]metadata.Candidate, error)
	MovieCollection(ctx context.Context, movieID int) ([]metadata.Candidate, error)
}

// TVSearcher es lo que necesita la identificación de series.
type TVSearcher interface {
	SearchTV(ctx context.Context, query string, year int) ([]metadata.Candidate, error)
}

var (
	reMatchCamel       = regexp.MustCompile(`([a-z])([A-Z])`)
	reMatchSA          = regexp.MustCompile(`\bS A\b`)
	reMatchInnerParen  = regexp.MustCompile(`\(([^)]*)\)?`)
	reMatchFromParen   = regexp.MustCompile(`\(.*$`)
	reMatchNoise       = regexp.MustCompile(`(?i)\b(versi[oó]n\s+extendida|extendida|extended|v\s+ext|walt\s+disney|remake|nova|cd\s?\d)\b`)
	reMatchSpaces      = regexp.MustCompile(`\s+`)
	reMatchDr          = regexp.MustCompile(`(?i)\bdr\b\.?`)
	reMatchDrWord      = regexp.MustCompile(`\bdr\b`)
	reMatchTrailingNum = regexp.MustCompile(`\s+\d{1,2}$`)
	reMatchOnlyNum     = regexp.MustCompile(`^\d{1,2}$`)
	reMatchNonAlnum    = regexp.MustCompile(`[^a-z0-9]+`)
	reMatchPieceSplit  = regexp.MustCompile(`[:(.]| - `)
	reMatchAltTitle    = regexp.MustCompile(`^\s*(.+?)\s*\((.+)\)\s*$`)
	reMatchShortNum    = regexp.MustCompile(`^\d{1,2}$`)
	reMatchLongNum     = regexp.MustCompile(`^\d{3,4}$`)
)

var (
	matchArticles = map[string]bool{"el": true, "la": true, "los": true, "las": true, "the": true, "a": true, "an": true, "un": true, "una": true}
	matchStop     = map[string]bool{
		"el": true, "la": true, "los": true, "las": true, "lo": true, "un": true, "una": true, "y": true, "e": true,
		"de": true, "del": true, "al": true, "en": true, "a": true, "o": true,
		"the": true, "of": true, "and": true, "an": true, "in": true, "on": true, "to": true, "or": true,
		"episodio": true, "episode": true, "parte": true, "part": true, "vol": true, "volumen": true, "capitulo": true, "chapter": true,
	}
	matchRomanAny   = map[string]int{"ii": 2, "iii": 3, "iv": 4, "vi": 6, "vii": 7, "viii": 8, "ix": 9}
	matchRomanAfter = map[string]int{"i": 1, "v": 5, "x": 10}
	matchNumberLead = map[string]bool{"episodio": true, "episode": true, "parte": true, "part": true, "vol": true, "volumen": true, "capitulo": true, "chapter": true}
	// matchFold quita acentos como la descomposición NFKD; «³» pasa a «3»
	// («Alien³»). Lo que no está aquí y no es [a-z0-9] acaba como separador.
	matchFold = map[rune]string{
		'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a",
		'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e",
		'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i",
		'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ō': "o",
		'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u",
		'ñ': "n", 'ç': "c", 'ý': "y", 'ÿ': "y",
		'¹': "1", '²': "2", '³': "3",
	}
)

type candidateSearch func(ctx context.Context, query string, year int) ([]metadata.Candidate, error)

// identifyCandidate hace la búsqueda y la puntuación comunes a películas y
// series. El ajuste por colección solo se aplica a las películas.
func identifyCandidate(ctx context.Context, search candidateSearch, title string, year int) (metadata.Candidate, float64, []string, bool, error) {
	variants := matchVariants(title, year)
	if len(variants) == 0 {
		return metadata.Candidate{}, 0, nil, false, nil
	}

	var pool []metadata.Candidate
	seen := map[int]bool{}
	for _, query := range variants {
		results, err := search(ctx, query, 0)
		if err != nil {
			if errors.Is(err, metadata.ErrNotFound) {
				continue
			}
			return metadata.Candidate{}, 0, variants, false, err
		}
		for _, candidate := range results {
			if !seen[candidate.TMDBID] {
				seen[candidate.TMDBID] = true
				pool = append(pool, candidate)
			}
		}
	}
	if len(pool) == 0 {
		return metadata.Candidate{}, 0, variants, false, nil
	}

	weight := poolWeights(pool)
	type scored struct {
		total, score float64
		candidate    metadata.Candidate
	}
	ranked := make([]scored, 0, len(pool))
	for _, candidate := range pool {
		score := scoreCandidate(title, year, variants[0], candidate, weight)
		// Desempate: entre títulos que encajan igual gana la más votada.
		tie := math.Min(float64(candidate.VoteCount), 20000) / 20000 * 0.08
		ranked = append(ranked, scored{score + tie, score, candidate})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].total > ranked[j].total })

	// Una homónima casi desconocida no gana a la famosa que también encaja
	// («Terminator 2» italiana de 1989 frente a «Terminator 2: El juicio final»).
	if top := ranked[0].candidate; top.VoteCount < 200 {
		for i := 1; i < len(ranked); i++ {
			other := ranked[i]
			if other.score >= MatchThreshold && other.candidate.VoteCount >= 500 && other.candidate.VoteCount >= 20*top.VoteCount {
				ranked[0], ranked[i] = ranked[i], ranked[0]
				break
			}
		}
	}

	best, score := ranked[0].candidate, ranked[0].score
	return best, score, variants, true, nil
}

// IdentifyMovie busca en TMDB la película de un título analizado y devuelve el
// mejor candidato con su puntuación. ok es false si nada llega al umbral. Los
// errores de red se devuelven para que el escaneo lo reintente más adelante;
// «sin resultados» no es un error.
func IdentifyMovie(ctx context.Context, searcher MovieSearcher, title string, year int) (metadata.Candidate, float64, bool, error) {
	best, score, variants, found, err := identifyCandidate(ctx, searcher.SearchMovies, title, year)
	if err != nil || !found {
		return best, score, false, err
	}
	if fixed, ok := byCollection(ctx, searcher, variants[0], best); ok {
		best, score = fixed, math.Max(score, MatchThreshold)
	}
	return best, score, score >= MatchThreshold, nil
}

// IdentifySeries busca en TMDB la serie de un título analizado. Comparte la
// puntuación y el umbral con las películas, pero no aplica la regla de sagas.
func IdentifySeries(ctx context.Context, searcher TVSearcher, title string, year int) (metadata.Candidate, float64, bool, error) {
	best, score, _, found, err := identifyCandidate(ctx, searcher.SearchTV, title, year)
	if err != nil || !found {
		return best, score, false, err
	}
	return best, score, score >= MatchThreshold, nil
}

// matchVariants genera las búsquedas para TMDB. La primera es el título limpio
// y es contra la que se puntúa siempre: las demás solo sirven para encontrar
// candidatos (recortar «2 - La venganza de los caídos» no debe salir gratis).
func matchVariants(title string, year int) []string {
	t := strings.TrimSpace(reMatchCamel.ReplaceAllString(title, "$1 $2"))
	t = reMatchSA.ReplaceAllString(t, "S.A.")
	var inner []string
	for _, m := range reMatchInnerParen.FindAllStringSubmatch(t, -1) {
		inner = append(inner, strings.TrimSpace(m[1]))
	}
	t = reMatchFromParen.ReplaceAllString(t, "")
	t = reMatchNoise.ReplaceAllString(t, "")
	t = strings.Trim(reMatchSpaces.ReplaceAllString(t, " "), " -")

	out := []string{t}
	for _, part := range inner {
		if utf8.RuneCountInString(part) > 3 && !reMatchNoise.MatchString(part) {
			out = append(out, part)
		}
	}
	// TMDB no expande abreviaturas: «Dr Strange» no devuelve «Doctor Strange».
	if reMatchDr.MatchString(t) {
		out = append(out, reMatchDr.ReplaceAllString(t, "Doctor"))
	}
	if year > 0 {
		// El analizador toma como año números que son parte del título
		// («Blade Runner 2049»).
		out = append(out, t+" "+strconv.Itoa(year))
	}
	noTrailing := reMatchTrailingNum.ReplaceAllString(t, "")
	out = append(out, noTrailing)
	if strings.Contains(t, " - ") {
		var words []string
		for _, part := range strings.Split(t, " - ") {
			part = strings.TrimSpace(part)
			if part != "" && !reMatchOnlyNum.MatchString(part) {
				words = append(words, part)
			}
		}
		if len(words) > 0 {
			stripped := make([]string, len(words))
			for i, word := range words {
				stripped[i] = reMatchTrailingNum.ReplaceAllString(word, "")
			}
			out = append(out, strings.Join(stripped, " "), words[len(words)-1], stripped[0])
		}
	}
	// Número de saga en medio: «Ice Age 2 El deshielo» → «Ice Age El deshielo».
	out = append(out, stripMiddleNumbers(noTrailing))

	seen := map[string]bool{}
	var result []string
	for _, v := range out {
		v = strings.Trim(reMatchSpaces.ReplaceAllString(v, " "), " -")
		key := normTitle(v, false)
		if v == "" || normTitle(v, true) == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}
	return result
}

// stripMiddleNumbers quita los números de una o dos cifras rodeados de
// espacios y de caracteres que no son cifras, sin tocar el primero ni el
// último.
func stripMiddleNumbers(s string) string {
	padded := " " + s + " "
	var b strings.Builder
	i := 0
	for i < len(padded) {
		if i > 0 && isSpace(padded[i]) && !isDigit(padded[i-1]) {
			digits := 0
			for digits < 2 && i+1+digits < len(padded) && isDigit(padded[i+1+digits]) {
				digits++
			}
			end := i + 1 + digits // posición del espacio de cierre
			if digits > 0 && end+1 < len(padded) && isSpace(padded[end]) && !isDigit(padded[end+1]) {
				b.WriteByte(' ')
				i = end + 1
				continue
			}
		}
		b.WriteByte(padded[i])
		i++
	}
	return strings.TrimSpace(b.String())
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// normTitle pasa a minúsculas sin acentos ni puntuación y quita los artículos
// iniciales. expand convierte «dr» en «doctor».
func normTitle(s string, expand bool) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if folded, ok := matchFold[r]; ok {
			b.WriteString(folded)
			continue
		}
		b.WriteRune(r)
	}
	out := strings.ReplaceAll(b.String(), "&", " and ")
	out = strings.TrimSpace(reMatchNonAlnum.ReplaceAllString(out, " "))
	if expand {
		out = reMatchDrWord.ReplaceAllString(out, "doctor")
	}
	words := strings.Fields(out)
	for len(words) > 1 && matchArticles[words[0]] {
		words = words[1:]
	}
	return strings.Join(words, " ")
}

// titleTokens separa las palabras significativas de los números (arábigos o
// romanos) de un título.
func titleTokens(s string) ([]string, map[int]bool) {
	var words []string
	nums := map[int]bool{}
	fields := strings.Fields(normTitle(s, true))
	for i, w := range fields {
		prev := ""
		if i > 0 {
			prev = fields[i-1]
		}
		switch {
		case reMatchShortNum.MatchString(w) || reMatchLongNum.MatchString(w):
			n, _ := strconv.Atoi(w)
			nums[n] = true
		case matchRomanAny[w] > 0:
			nums[matchRomanAny[w]] = true
		case matchRomanAfter[w] > 0 && matchNumberLead[prev]:
			nums[matchRomanAfter[w]] = true
		case !matchStop[w]:
			words = append(words, w)
		}
	}
	return words, nums
}

// sameWord tolera erratas en palabras largas («Einfield» ≈ «Enfield»).
func sameWord(a, b string) bool {
	if a == b {
		return true
	}
	return len(a) >= 5 && len(b) >= 5 && seqRatio(a, b) >= 0.85
}

// seqRatio es la medida de Ratcliff/Obershelp, la misma que difflib de Python
// (con la que se ajustaron los umbrales).
func seqRatio(a, b string) float64 {
	if len(a)+len(b) == 0 {
		return 1
	}
	return 2 * float64(matchingChars(a, b)) / float64(len(a)+len(b))
}

func matchingChars(a, b string) int {
	if a == "" || b == "" {
		return 0
	}
	bi, bj, bk := 0, 0, 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			k := 0
			for i+k < len(a) && j+k < len(b) && a[i+k] == b[j+k] {
				k++
			}
			if k > bk {
				bi, bj, bk = i, j, k
			}
		}
	}
	if bk == 0 {
		return 0
	}
	return bk + matchingChars(a[:bi], b[:bj]) + matchingChars(a[bi+bk:], b[bj+bk:])
}

// poolWeights pondera cada palabra por lo poco que aparece entre los
// candidatos: «Star Wars» sale en todos y apenas distingue; «contraataca»
// solo en uno y decide.
func poolWeights(pool []metadata.Candidate) func(string) float64 {
	df := map[string]int{}
	for _, candidate := range pool {
		titleWords, _ := titleTokens(candidate.Title)
		originalWords, _ := titleTokens(candidate.OriginalTitle)
		for w := range toSet(append(titleWords, originalWords...)) {
			df[w]++
		}
	}
	n := float64(len(pool))
	cache := map[string]float64{}
	return func(w string) float64 {
		if value, ok := cache[w]; ok {
			return value
		}
		count := 0
		for k, v := range df {
			if v > count && sameWord(w, k) {
				count = v
			}
		}
		value := math.Log(1 + n/(float64(count)+0.5))
		cache[w] = value
		return value
	}
}

func overlapWeight(queryWords, nameWords []string, weight func(string) float64) float64 {
	total := 0.0
	for q := range toSet(queryWords) {
		for _, n := range nameWords {
			if sameWord(q, n) {
				total += weight(q)
				break
			}
		}
	}
	return total
}

func setWeight(words []string, weight func(string) float64) float64 {
	total := 0.0
	for w := range toSet(words) {
		total += weight(w)
	}
	return total
}

// altNames añade a los títulos del candidato la otra mitad de un título
// «Marte (The Martian)» cuando una de las dos es el título original.
func altNames(candidate metadata.Candidate) []string {
	names := []string{candidate.Title, candidate.OriginalTitle}
	if m := reMatchAltTitle.FindStringSubmatch(candidate.Title); m != nil {
		original := normTitle(candidate.OriginalTitle, true)
		switch {
		case normTitle(m[1], true) == original:
			names = append(names, m[2])
		case normTitle(m[2], true) == original:
			names = append(names, m[1])
		}
	}
	return names
}

func titleForScore(title string) string {
	return reMatchNoise.ReplaceAllString(reMatchFromParen.ReplaceAllString(title, ""), "")
}

// scoreCandidate es un F1 ponderado (palabras de la búsqueda frente a las del
// candidato) más reglas para contención, palabras pegadas, números de secuela
// y año.
func scoreCandidate(title string, year int, query string, candidate metadata.Candidate, weight func(string) float64) float64 {
	names := altNames(candidate)
	queryWords, queryNums := titleTokens(query)
	type nameTokens struct {
		words []string
		nums  map[int]bool
	}
	tokens := make([]nameTokens, len(names))
	var union []string
	for i, name := range names {
		words, nums := titleTokens(name)
		tokens[i] = nameTokens{words, nums}
		union = append(union, words...)
	}

	queryWeight := setWeight(queryWords, weight)
	if queryWeight == 0 {
		queryWeight = 1
	}
	recall := overlapWeight(queryWords, union, weight) / queryWeight
	precision := 0.0
	for _, tk := range tokens {
		if len(tk.words) > 0 {
			precision = math.Max(precision, overlapWeight(tk.words, queryWords, weight)/setWeight(tk.words, weight))
		}
	}
	best := 0.0
	if recall+precision > 0 {
		best = 2 * recall * precision / (recall + precision)
	}
	f1 := best

	// Un título contenido entero en el otro («Grindhouse Death Proof» / «Death
	// Proof»), también por partes del candidato («Training Day (Día de
	// entrenamiento)»). Vale menos que coincidir entero, para que «Misión
	// imposible» no empate con «Misión imposible: Protocolo fantasma».
	querySet := toSet(queryWords)
	queryShortNums := map[int]bool{}
	for n := range queryNums {
		if n > 1 && n < 100 {
			queryShortNums[n] = true
		}
	}
	for _, name := range names {
		for _, piece := range append([]string{name}, reMatchPieceSplit.Split(name, -1)...) {
			pieceWords, pieceNums := titleTokens(piece)
			pieceSet := toSet(pieceWords)
			// «Terminator 2: El juicio final»: la parte «Terminator 2» coincide
			// en palabra y número con el archivo; el número la hace distintiva.
			if len(queryShortNums) > 0 && setsEqual(pieceSet, querySet) && subset(queryShortNums, pieceNums) {
				best = math.Max(best, 0.95)
				f1 = math.Max(f1, 0.95)
				continue
			}
			if len(pieceSet) < 2 || joinedLen(pieceSet) < 8 || len(querySet) < 2 {
				continue
			}
			if subset(pieceSet, querySet) || subset(querySet, pieceSet) {
				best = math.Max(best, 0.85)
			}
		}
	}

	// Palabras pegadas o separadas de más: «MoneyBall», «HarryPotter».
	squashed := strings.ReplaceAll(normTitle(titleForScore(title), true), " ", "")
	for _, name := range names {
		if squashed == strings.ReplaceAll(normTitle(name, true), " ", "") {
			best = math.Max(best, 1)
		}
	}

	candidateNums := map[int]bool{}
	for _, tk := range tokens {
		for n := range tk.nums {
			candidateNums[n] = true
		}
	}
	if year > 0 {
		delete(candidateNums, year)
	}
	// Un año dentro del título («Los viajes de Gulliver 2010») no distingue;
	// «2049» sí.
	maxYear := time.Now().Year() + 1
	for n := range candidateNums {
		if n >= 100 && (n < 1900 || n > maxYear) && !queryNums[n] {
			best -= 0.1
			break
		}
	}
	// El 1 marca la primera película, no una parte: no suma ni resta.
	candidateShort := map[int]bool{}
	for n := range candidateNums {
		if n > 1 && n < 100 {
			candidateShort[n] = true
		}
	}
	switch {
	case len(queryShortNums) > 0 && intersects(queryShortNums, candidateShort):
		if f1 >= 0.75 {
			best += 0.15
		}
	case len(queryShortNums) > 0 && len(candidateShort) > 0:
		best -= 0.15
	case len(queryShortNums) > 0:
		if recall >= 0.99 {
			best -= 0.1
		} else {
			best -= 0.2
		}
	case len(candidateShort) > 0:
		best -= 0.1
	}

	if year > 0 {
		yearText := strconv.Itoa(year)
		inName := false
		for _, name := range names {
			for _, w := range strings.Fields(normTitle(name, true)) {
				if w == yearText {
					inName = true
				}
			}
		}
		switch {
		case inName:
			best += 0.1
		case candidate.Year-year <= 1 && year-candidate.Year <= 1:
			best += 0.15
		default:
			best -= 0.3
		}
	}
	return best
}

// byCollection corrige una saga numerada sin subtítulo («Jungla de Cristal 4»,
// «A Todo Gas 3»), cuyo título en castellano no lleva el número: elige la
// parte N de la colección de TMDB.
func byCollection(ctx context.Context, searcher MovieSearcher, query string, winner metadata.Candidate) (metadata.Candidate, bool) {
	queryWords, queryNums := titleTokens(query)
	n := 0
	for value := range queryNums {
		if value >= 2 && value < 30 {
			if n != 0 {
				return metadata.Candidate{}, false
			}
			n = value
		}
	}
	if n == 0 {
		return metadata.Candidate{}, false
	}

	winnerTitle, _ := titleTokens(winner.Title)
	winnerOriginal, _ := titleTokens(winner.OriginalTitle)
	if !containsAll(queryWords, append(winnerTitle, winnerOriginal...)) {
		return metadata.Candidate{}, false
	}

	parts, err := searcher.MovieCollection(ctx, winner.TMDBID)
	if err != nil || len(parts) == 0 {
		return metadata.Candidate{}, false
	}
	hasAll := func(text string) bool {
		words, _ := titleTokens(text)
		return containsAll(queryWords, words)
	}

	// Solo si todas las palabras buscadas son de la saga (salen en 2+ partes):
	// «X-Men 7 - Días del futuro pasado» lleva subtítulo y no se renumera.
	if len(queryWords) > 0 {
		inParts := 0
		for _, part := range parts {
			if hasAll(part.Title + " " + part.OriginalTitle) {
				inParts++
			}
		}
		if inParts < 2 {
			return metadata.Candidate{}, false
		}
	}

	var filtered []metadata.Candidate
	for _, part := range parts {
		if hasAll(part.Title) {
			filtered = append(filtered, part)
		}
	}
	// La numeración del usuario suele ser la posición en la saga («El
	// caballero oscuro 2» tras «Batman Begins 1»); si esa parte no lleva el
	// nombre buscado, se cuenta solo entre las que sí lo llevan («Jurassic
	// World 2»).
	var chosen *metadata.Candidate
	switch {
	case len(parts) >= n && hasAll(parts[n-1].Title):
		chosen = &parts[n-1]
	case len(filtered) >= n:
		chosen = &filtered[n-1]
	case len(parts) >= n:
		chosen = &parts[n-1]
	}
	if chosen == nil || chosen.TMDBID == winner.TMDBID {
		return metadata.Candidate{}, false
	}
	return *chosen, true
}

func containsAll(queryWords, words []string) bool {
	for _, q := range queryWords {
		found := false
		for _, w := range words {
			if sameWord(q, w) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func toSet(words []string) map[string]bool {
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}
	return set
}

func setsEqual(a, b map[string]bool) bool {
	return len(a) == len(b) && subset(a, b)
}

func subset[K comparable](a, b map[K]bool) bool {
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func intersects(a, b map[int]bool) bool {
	for k := range a {
		if b[k] {
			return true
		}
	}
	return false
}

func joinedLen(set map[string]bool) int {
	total := 0
	for w := range set {
		total += len(w)
	}
	if len(set) > 1 {
		total += len(set) - 1
	}
	return total
}
