// Marca que el episodio actual ha terminado de dar paso al siguiente dentro de
// la misma página: el reproductor nuevo debe arrancar solo. Vive en memoria
// porque una recarga o una visita nueva no deben reproducir nada por sí solas.
let continuePlaying = false;

export function markContinuePlaying(): void {
	continuePlaying = true;
}

export function consumeContinuePlaying(): boolean {
	const value = continuePlaying;
	continuePlaying = false;
	return value;
}
