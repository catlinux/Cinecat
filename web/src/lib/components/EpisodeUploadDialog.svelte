<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import type { UploadTargetRoot } from '$lib/api';
	import { detectEpisode } from '$lib/episode-detect';
	import {
		UPLOAD_ACCEPT,
		buildEpisodeUploadName,
		buildUploadURL,
		formatUploadSize,
		isUploadableFileName,
		uploadErrorMessage
	} from '$lib/uploads';

	type UploadMode = 'serie-nueva' | 'temporada-nueva' | 'capitulo';
	type UploadStatus = 'queued' | 'uploading' | 'done' | 'error';

	interface UploadEntry {
		file: File;
		name: string;
		size: number;
		season: string | number;
		episode: string | number;
		status: UploadStatus;
		progress: number;
		error: string;
	}

	let {
		id = 'episode-upload',
		mode,
		triggerLabel,
		seriesId = null,
		seriesTitle = '',
		roots = [],
		allowed = true,
		season = null,
		defaultSeason = null,
		defaultEpisode = 1,
		triggerClass = 'button button-secondary episode-upload-trigger'
	}: {
		id?: string;
		mode: UploadMode;
		triggerLabel?: string;
		seriesId?: number | null;
		seriesTitle?: string;
		roots?: UploadTargetRoot[];
		allowed?: boolean;
		season?: number | null;
		defaultSeason?: number | null;
		defaultEpisode?: number | null;
		triggerClass?: string;
	} = $props();

	const titleID = $derived(`${id}-title`);
	const videoRoots = $derived(roots.filter((root) => root.library_kind === 'video'));
	const rootsWithSeries = $derived(videoRoots.filter((root) => root.series_count > 0));
	const destinationOptions = $derived(
		rootsWithSeries.length > 0 ? rootsWithSeries : videoRoots
	);
	const preferredRoot = $derived(
		[...destinationOptions].sort(
			(left, right) =>
				right.series_count - left.series_count || left.path.localeCompare(right.path)
		)[0] ?? null
	);

	let dialogEl: HTMLDialogElement | null = $state(null);
	let fileInput: HTMLInputElement | null = $state(null);
	let openButton: HTMLButtonElement | null = $state(null);
	let closeButton: HTMLButtonElement | null = $state(null);
	let jsReady = $state(false);
	let dialogOpen = $state(false);
	let entries = $state<UploadEntry[]>([]);
	let processing = $state(false);
	let finished = $state(false);
	let cancelled = $state(false);
	let message = $state('');
	let seriesName = $state('');
	let seasonInput = $state<string | number>(
		untrack(() =>
			String(mode === 'temporada-nueva' ? (defaultSeason ?? 1) : (season ?? 1))
		)
	);
	let selectedRootPath = $state(untrack(() => preferredRoot?.path ?? ''));
	let currentXHR: XMLHttpRequest | null = null;

	const buttonLabel = $derived(
		triggerLabel ??
			(mode === 'serie-nueva'
				? 'Crear nueva serie'
				: mode === 'temporada-nueva'
					? 'Nueva temporada'
					: 'Subir capítulo')
	);
	const selectedRoot = $derived(
		destinationOptions.find((root) => root.path === selectedRootPath) ?? preferredRoot
	);
	const showDestinationSelect = $derived(destinationOptions.length > 1);

	onMount(() => {
		jsReady = true;
	});

	function openDialog(): void {
		if (processing) {
			return;
		}
		if (finished) {
			resetEntries();
		}
		selectedRootPath = destinationOptions.some((root) => root.path === selectedRootPath)
			? selectedRootPath
			: (preferredRoot?.path ?? '');
		dialogOpen = true;
		try {
			if (typeof dialogEl?.showModal === 'function') {
				dialogEl.showModal();
			}
		} catch {
			// El atributo open mantiene el diálogo visible si showModal falla.
		}
		queueMicrotask(() => closeButton?.focus());
	}

	function closeDialog(): void {
		if (!dialogEl || processing) {
			return;
		}
		dialogOpen = false;
		if (typeof dialogEl.close === 'function' && dialogEl.open) {
			dialogEl.close();
		}
		openButton?.focus();
	}

	function handleClose(): void {
		dialogOpen = false;
		openButton?.focus();
	}

	function handleCancel(event: Event): void {
		if (processing) {
			event.preventDefault();
			return;
		}
		closeDialog();
	}

	function handleClick(event: MouseEvent): void {
		if (event.target === dialogEl) {
			closeDialog();
		}
	}

	function resetEntries(): void {
		entries = [];
		if (fileInput) {
			fileInput.value = '';
		}
		finished = false;
		message = '';
	}

	function chooseFiles(event: Event): void {
		const input = event.currentTarget as HTMLInputElement;
		const all = Array.from(input.files ?? []);
		const accepted = all.filter((file) => isUploadableFileName(file.name));
		const detectSeason =
			mode === 'capitulo'
				? season
				: mode === 'temporada-nueva'
					? numberValue(seasonInput)
					: 1;
		let nextFallback = defaultEpisode ?? 1;
		entries = accepted.map((file) => {
			const guess = detectEpisode(file.name, detectSeason);
			let episode = guess.episode;
			if (episode === null) {
				episode = nextFallback;
				nextFallback += 1;
			}
			const entrySeason =
				mode === 'serie-nueva'
					? (guess.season ?? 1)
					: mode === 'temporada-nueva'
						? (numberValue(seasonInput) ?? 1)
						: (season ?? null);
			return {
				file,
				name: file.name,
				size: file.size,
				season: entrySeason === null ? '' : String(entrySeason),
				episode: String(episode),
				status: 'queued',
				progress: 0,
				error: ''
			};
		});
		finished = false;
		message =
			accepted.length === all.length
				? ''
				: 'Se han descartado los archivos con una extensión no admitida.';
	}

	function numberValue(value: string | number | null | undefined): number | null {
		const raw = String(value ?? '').trim();
		if (!raw || !/^\d+$/.test(raw)) {
			return null;
		}
		const parsed = Number(raw);
		return Number.isSafeInteger(parsed) ? parsed : null;
	}

	function entrySeason(entry: UploadEntry): number | null {
		if (mode === 'serie-nueva') {
			return numberValue(entry.season);
		}
		if (mode === 'temporada-nueva') {
			return numberValue(seasonInput);
		}
		return season;
	}

	function entryEpisode(entry: UploadEntry): number | null {
		return numberValue(entry.episode);
	}

	function validSeriesName(value: string): boolean {
		const name = value.trim();
		return (
			name !== '' &&
			!name.startsWith('.') &&
			!name.includes('/') &&
			!name.includes('\\')
		);
	}

	function startUpload(): void {
		if (processing) {
			return;
		}
		message = '';
		if (!allowed) {
			message =
				'La modificación de archivos está desactivada. Actívala en Administración → Biblioteca.';
			return;
		}
		if (entries.length === 0) {
			message = 'Elige al menos un archivo.';
			return;
		}
		if (mode === 'serie-nueva') {
			if (!validSeriesName(seriesName)) {
				message = 'Escribe un nombre de serie sin /, \\ ni punto inicial.';
				return;
			}
			if (!selectedRoot) {
				message = 'No hay una carpeta de biblioteca disponible para la serie.';
				return;
			}
		} else if (!seriesId) {
			message = 'No se ha indicado la serie.';
			return;
		}

		const seen = new Set<string>();
		for (const entry of entries) {
			const entrySeasonValue = entrySeason(entry);
			const episode = entryEpisode(entry);
			if (
				entrySeasonValue === null ||
				entrySeasonValue < 0 ||
				entrySeasonValue > 99 ||
				episode === null ||
				episode < 1 ||
				episode > 999
			) {
				message = 'Revisa las temporadas (0-99) y los episodios (1-999).';
				return;
			}
			const key = `${entrySeasonValue}-${episode}`;
			if (seen.has(key)) {
				message = 'Hay temporadas y episodios repetidos en la cola.';
				return;
			}
			seen.add(key);
		}

		for (const entry of entries) {
			entry.status = 'queued';
			entry.progress = 0;
			entry.error = '';
		}
		finished = false;
		cancelled = false;
		processing = true;
		void runQueue();
	}

	async function runQueue(): Promise<void> {
		for (const entry of entries) {
			if (cancelled) {
				break;
			}
			const entrySeasonValue = entrySeason(entry);
			const episode = entryEpisode(entry);
			if (entrySeasonValue === null || episode === null) {
				continue;
			}
			entry.status = 'uploading';
			entry.progress = 0;
			try {
				await uploadEntry(entry, entrySeasonValue, episode);
				entry.status = 'done';
				entry.progress = 100;
			} catch (reason) {
				entry.status = 'error';
				entry.error =
					reason instanceof Error ? reason.message : 'No se pudo subir el archivo.';
				if (cancelled) {
					break;
				}
			}
		}
		currentXHR = null;
		processing = false;
		finished = true;
	}

	function uploadEntry(entry: UploadEntry, entrySeason: number, episode: number): Promise<void> {
		const xhr = new XMLHttpRequest();
		currentXHR = xhr;
		return new Promise((resolve, reject) => {
			let url: string;
			if (mode === 'serie-nueva' && selectedRoot) {
				url = buildUploadURL({
					library: selectedRoot.library_id,
					dir: selectedRoot.path,
					subdir: seriesName.trim(),
					name: buildEpisodeUploadName(
						seriesName.trim(),
						entrySeason,
						episode,
						entry.name
					)
				});
			} else {
				const params = new URLSearchParams({
					season: String(entrySeason),
					episode: String(episode),
					name: entry.name
				});
				url = `/series/${seriesId}/upload?${params.toString()}`;
			}
			xhr.open('PUT', url);
			xhr.setRequestHeader('Content-Type', 'application/octet-stream');
			xhr.upload.onprogress = (event) => {
				if (event.lengthComputable && event.total > 0) {
					entry.progress = Math.round((event.loaded / event.total) * 100);
				}
			};
			xhr.onload = () => {
				currentXHR = null;
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve();
					return;
				}
				reject(
					new Error(uploadErrorMessage(xhr.status, responsePayload(xhr.responseText)))
				);
			};
			xhr.onerror = () => {
				currentXHR = null;
				reject(new Error('No se pudo conectar con el servidor.'));
			};
			xhr.onabort = () => {
				currentXHR = null;
				reject(new Error('Subida cancelada.'));
			};
			xhr.send(entry.file);
		});
	}

	function responsePayload(text: string): unknown {
		try {
			return JSON.parse(text) as unknown;
		} catch {
			return null;
		}
	}

	function cancelUpload(): void {
		cancelled = true;
		currentXHR?.abort();
	}

	function statusLabel(entry: UploadEntry): string {
		if (entry.status === 'queued') {
			return 'En cola';
		}
		if (entry.status === 'uploading') {
			return `Subiendo (${entry.progress} %)`;
		}
		if (entry.status === 'done') {
			return 'Hecho';
		}
		return entry.error || 'Error';
	}
</script>

<span class="episode-upload-trigger-wrap" hidden={!jsReady}>
	<button
		class={triggerClass}
		type="button"
		disabled={processing}
		aria-haspopup="dialog"
		aria-controls={id}
		bind:this={openButton}
		onclick={openDialog}>{buttonLabel}</button
	>
</span>

<dialog
	bind:this={dialogEl}
	{id}
	class="episode-upload-dialog"
	open={dialogOpen}
	aria-labelledby={titleID}
	onclick={handleClick}
	oncancel={handleCancel}
	onclose={handleClose}
>
	<div class="episode-upload-body">
		<header class="episode-upload-head">
			<div>
				<p class="kicker">Subida guiada</p>
				<h2 id={titleID}>{buttonLabel}</h2>
				{#if seriesTitle}
					<p class="episode-upload-subtitle">{seriesTitle}</p>
				{/if}
			</div>
			<button
				class="episode-upload-close"
				type="button"
				bind:this={closeButton}
				disabled={processing}
				onclick={closeDialog}
				aria-label="Cerrar">×</button
			>
		</header>

		<p class="episode-upload-warning">
			No cierres la página hasta que terminen las subidas.
		</p>

		{#if !allowed}
			<p class="error" role="alert">
				La modificación de archivos está desactivada. Actívala en Administración → Biblioteca.
			</p>
		{/if}

		{#if mode === 'serie-nueva'}
			<label>
				<span>Nombre de la serie</span>
				<input
					type="text"
					bind:value={seriesName}
					placeholder="Por ejemplo, The Big Bang Theory"
					disabled={processing}
					required
				/>
			</label>
			<label>
				<span>Serie de destino</span>
				{#if showDestinationSelect}
					<select bind:value={selectedRootPath} disabled={processing}>
						{#each destinationOptions as root (root.path)}
							<option value={root.path}>{root.library_name} · {root.path}</option>
						{/each}
					</select>
				{:else if selectedRoot}
					<span class="episode-upload-destination"
						>{selectedRoot.library_name} · {selectedRoot.path}</span
					>
				{:else}
					<span class="error">No hay raíces de biblioteca de vídeo disponibles.</span>
				{/if}
			</label>
		{:else if mode === 'temporada-nueva'}
			<label>
				<span>Número de temporada</span>
				<input
					type="number"
					min="0"
					max="99"
					bind:value={seasonInput}
					disabled={processing}
					required
				/>
			</label>
		{/if}

		<label>
			<span>Archivos de vídeo o subtítulos</span>
			<input
				bind:this={fileInput}
				type="file"
				accept={UPLOAD_ACCEPT}
				multiple
				disabled={processing}
				onchange={chooseFiles}
			/>
		</label>

		{#if entries.length > 0}
			<ul class="episode-upload-list">
				{#each entries as entry, index (`${entry.name}-${entry.status}-${index}`)}
					<li class="episode-upload-item" class:episode-upload-item--error={entry.status === 'error'}>
						<div class="episode-upload-file">
							<span class="episode-upload-name">{entry.name}</span>
							<span class="episode-upload-size">{formatUploadSize(entry.size)}</span>
						</div>
						<div class="episode-upload-fields">
							<label>
								<span>Temporada</span>
								{#if mode === 'serie-nueva'}
									<input
										type="number"
										min="0"
										max="99"
										bind:value={entry.season}
										disabled={processing}
									/>
								{:else}
									<input
										type="number"
										value={mode === 'capitulo' ? (season ?? '') : seasonInput}
										disabled
									/>
								{/if}
							</label>
							<label>
								<span>Episodio</span>
								<input
									type="number"
									min="1"
									max="999"
									bind:value={entry.episode}
									disabled={processing}
								/>
							</label>
						</div>
						<progress
							class="episode-upload-progress"
							value={entry.progress}
							max="100"
							aria-label={`Progreso de ${entry.name}`}
						></progress>
						<p class="episode-upload-status" aria-live="polite">{statusLabel(entry)}</p>
					</li>
				{/each}
			</ul>
		{/if}

		{#if message}
			<p class="error" role="alert">{message}</p>
		{/if}

		<div class="episode-upload-actions">
			<button
				class="button"
				type="button"
				disabled={processing || entries.length === 0}
				onclick={startUpload}>Subir</button
			>
			{#if processing}
				<button class="button-danger" type="button" onclick={cancelUpload}>
					Cancelar la subida
				</button>
			{/if}
		</div>

		{#if finished && entries.some((entry) => entry.status === 'done')}
			<p class="success" role="status">
				Subida terminada; el escaneo añadirá los capítulos en unos minutos.
			</p>
		{/if}
	</div>
</dialog>

<style>
	.episode-upload-trigger-wrap[hidden] {
		display: none;
	}

	.episode-upload-trigger {
		margin-top: 0;
		padding: 0.45rem 0.75rem;
		font-size: 0.68rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.episode-upload-dialog {
		width: min(46rem, calc(100vw - 2rem));
		max-width: calc(100vw - 2rem);
		max-height: calc(100vh - 2rem);
		max-height: calc(100dvh - 2rem);
		margin: auto;
		padding: 0;
		overflow-y: auto;
		border: 1px solid var(--borde-medio);
		border-radius: 4px;
		background: var(--superficie);
		color: var(--texto-tenue);
	}

	.episode-upload-dialog::backdrop {
		background: rgb(20 10 11 / 65%);
	}

	.episode-upload-body {
		display: grid;
		gap: 0.8rem;
		padding: 1rem;
	}

	.episode-upload-head {
		display: flex;
		align-items: start;
		justify-content: space-between;
		gap: 1rem;
	}

	.episode-upload-head h2 {
		margin: 0.2rem 0 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.45rem;
		line-height: 1.1;
	}

	.episode-upload-subtitle {
		margin: 0.35rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.82rem;
	}

	.episode-upload-close {
		padding: 0.1rem 0.4rem;
		border: 0;
		background: transparent;
		color: var(--texto-suave);
		font-size: 1.4rem;
		line-height: 1;
		cursor: pointer;
	}

	.episode-upload-warning {
		margin: 0;
		padding: 0.5rem 0.65rem;
		border-left: 3px solid var(--dorado);
		background: var(--tinte);
		color: var(--texto-suave);
		font-size: 0.82rem;
	}

	.episode-upload-body > label,
	.episode-upload-fields label {
		display: grid;
		gap: 0.3rem;
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--granate);
	}

	.episode-upload-body input,
	.episode-upload-body select {
		width: 100%;
		padding: 0.48rem 0.55rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--crema-clara);
		color: var(--tinta);
		font: inherit;
		font-size: 0.86rem;
		letter-spacing: normal;
		text-transform: none;
	}

	.episode-upload-body input[type='file'] {
		padding: 0.3rem 0;
		border: 0;
		background: transparent;
	}

	.episode-upload-destination {
		padding: 0.45rem 0;
		color: var(--texto-tenue);
		font-size: 0.82rem;
		font-weight: 400;
		letter-spacing: normal;
		text-transform: none;
	}

	.episode-upload-list {
		display: grid;
		gap: 0.55rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.episode-upload-item {
		padding: 0.55rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--tinte);
	}

	.episode-upload-item--error {
		border-color: var(--naranja);
	}

	.episode-upload-file {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 0.4rem;
	}

	.episode-upload-name {
		font-weight: 600;
		color: var(--granate);
		overflow-wrap: anywhere;
	}

	.episode-upload-size {
		color: var(--texto-apagado);
		font-size: 0.75rem;
	}

	.episode-upload-fields {
		display: grid;
		grid-template-columns: repeat(2, minmax(6rem, 1fr));
		gap: 0.5rem;
		margin-top: 0.5rem;
	}

	.episode-upload-progress {
		width: 100%;
		height: 0.5rem;
		margin-top: 0.45rem;
		accent-color: var(--naranja);
	}

	.episode-upload-status {
		margin: 0.3rem 0 0;
		color: var(--texto-tenue);
		font-size: 0.75rem;
		overflow-wrap: anywhere;
	}

	.episode-upload-item--error .episode-upload-status {
		color: var(--naranja);
	}

	.episode-upload-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.episode-upload-actions .button,
	.episode-upload-actions .button-danger {
		margin: 0;
	}
</style>
