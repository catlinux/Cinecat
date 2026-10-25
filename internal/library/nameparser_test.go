package library

import "testing"

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name string
		want Media
	}{
		{
			name: "El laberinto del fauno (2006).mkv",
			want: Media{Title: "El laberinto del fauno", Year: 2006, Kind: KindMovie},
		},
		{
			name: "Amélie (2001) 1080p BluRay x264-GRUPO.mkv",
			want: Media{Title: "Amélie", Year: 2001, Kind: KindMovie},
		},
		{
			name: "Spider-Man.Homecoming.2017.2160p.WEB-DL.HDR.x265-TERMiNAL.mkv",
			want: Media{Title: "Spider-Man Homecoming", Year: 2017, Kind: KindMovie},
		},
		{
			name: "1917.2019.1080p.BluRay.x264.mkv",
			want: Media{Title: "1917", Year: 2019, Kind: KindMovie},
		},
		{
			name: "2001 A Space Odyssey (1968) [1080p].mkv",
			want: Media{Title: "2001 A Space Odyssey", Year: 1968, Kind: KindMovie},
		},
		{
			name: "Blade Runner 2049.2017.1080p.BluRay.REMUX.mkv",
			want: Media{Title: "Blade Runner 2049", Year: 2017, Kind: KindMovie},
		},
		{
			name: "1917.mkv",
			want: Media{Title: "1917", Kind: KindMovie},
		},
		{
			name: "The Matrix 1999 1080p.mkv",
			want: Media{Title: "The Matrix", Year: 1999, Kind: KindMovie},
		},
		{
			name: "No_Country_for_Old_Men_2007_DVDRip.avi",
			want: Media{Title: "No Country for Old Men", Year: 2007, Kind: KindMovie},
		},
		{
			name: "Le.Fabuleux.Destin.d'Amélie.Poulain.2001.mkv",
			want: Media{Title: "Le Fabuleux Destin d'Amélie Poulain", Year: 2001, Kind: KindMovie},
		},
		{
			name: " - Leading Dash Movie (2015) [1080p].mkv",
			want: Media{Title: "Leading Dash Movie", Year: 2015, Kind: KindMovie},
		},
		{
			name: "Movie.2018.SPANISH.720p.BluRay.x264-GRUPO.mkv",
			want: Media{Title: "Movie", Year: 2018, Kind: KindMovie},
		},
		{
			name: "Film (2016) [Director's Cut].mkv",
			want: Media{Title: "Film", Year: 2016, Kind: KindMovie},
		},
		{
			name: "Breaking Bad S05E14 Ozymandias.mkv",
			want: Media{Title: "Breaking Bad", Kind: KindEpisode, Season: 5, Episode: 14},
		},
		{
			name: "breaking.bad.s01e02.720p.hdtv.x264.mkv",
			want: Media{Title: "breaking bad", Kind: KindEpisode, Season: 1, Episode: 2},
		},
		{
			name: "The.Office.US.S03E12.1080p.WEB.x264.mkv",
			want: Media{Title: "The Office US", Kind: KindEpisode, Season: 3, Episode: 12},
		},
		{
			name: "Doctor Who 1x05 World War Three.avi",
			want: Media{Title: "Doctor Who", Kind: KindEpisode, Season: 1, Episode: 5},
		},
		{
			name: "Serie Española - 2x08 - El final.mkv",
			want: Media{Title: "Serie Española", Kind: KindEpisode, Season: 2, Episode: 8},
		},
		{
			name: "Show.Name.S02E03E04.1080p.mkv",
			want: Media{Title: "Show Name", Kind: KindEpisode, Season: 2, Episode: 3},
		},
		{
			name: "El.Internado.4x02.1080p.mkv",
			want: Media{Title: "El Internado", Kind: KindEpisode, Season: 4, Episode: 2},
		},
		{
			name: "Anime.Show.S01E03.1080p.[GRUPO].mkv",
			want: Media{Title: "Anime Show", Kind: KindEpisode, Season: 1, Episode: 3},
		},
		{
			name: "Cosmos A Spacetime Odyssey Season 1 Episode 3.mkv",
			want: Media{Title: "Cosmos A Spacetime Odyssey", Kind: KindEpisode, Season: 1, Episode: 3},
		},
		{
			name: "The.100.S01E01.1080p.mkv",
			want: Media{Title: "The 100", Kind: KindEpisode, Season: 1, Episode: 1},
		},
		{
			name: "Show Season 2.mkv",
			want: Media{Title: "Show", Kind: KindSeries, Season: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFilename(tt.name)
			if got != tt.want {
				t.Fatalf("ParseFilename(%q) = %+v, se esperaba %+v", tt.name, got, tt.want)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want Media
	}{
		{
			name: "título de serie desde la carpeta cuando el archivo solo trae SxxEyy",
			path: "Series/Season 1/S01E03.mkv",
			want: Media{Title: "Series", Kind: KindEpisode, Season: 1, Episode: 3},
		},
		{
			name: "serie con título y temporada en la ruta",
			path: "Breaking Bad/Season 2/Breaking Bad S02E05.mkv",
			want: Media{Title: "Breaking Bad", Kind: KindEpisode, Season: 2, Episode: 5},
		},
		{
			name: "número de episodio al principio del archivo dentro de una temporada",
			path: "Dark/Season 1/04 - Secretos.mkv",
			want: Media{Title: "Dark", Kind: KindEpisode, Season: 1, Episode: 4},
		},
		{
			name: "episodio con E05 dentro de una temporada",
			path: "Show/Season 1/Show - E05 - Un título.mkv",
			want: Media{Title: "Show", Kind: KindEpisode, Season: 1, Episode: 5},
		},
		{
			name: "año de la carpeta de la serie",
			path: "Series (2008)/Season 1/S01E01.mkv",
			want: Media{Title: "Series", Year: 2008, Kind: KindEpisode, Season: 1, Episode: 1},
		},
		{
			name: "película dentro de una carpeta con año",
			path: "Peliculas/El Orfanato (2007)/El Orfanato.mkv",
			want: Media{Title: "El Orfanato", Year: 2007, Kind: KindMovie},
		},
		{
			name: "temporada abreviada S1",
			path: "Serie/S1/Serie - 1x02.mkv",
			want: Media{Title: "Serie", Kind: KindEpisode, Season: 1, Episode: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePath(tt.path)
			if got != tt.want {
				t.Fatalf("ParsePath(%q) = %+v, se esperaba %+v", tt.path, got, tt.want)
			}
		})
	}
}
