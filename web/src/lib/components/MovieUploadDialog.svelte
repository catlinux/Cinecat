<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import type { UploadTargetRoot } from '$lib/api';
	import {
		UPLOAD_ACCEPT,
		buildUploadURL,
		formatUploadSize,
		isUploadableFileName,
		isUploadVideoName,
		uploadErrorMessage,
		uploadFileStem,
		uploadSubdirForFile,
		validUploadComponent
	} from '$lib/uploads';

	type UploadStatus = 'queued' | 'uploading' | 'done' | 'error';

	interface UploadEntry {
		file: File;
		name: string;
		size: number;
		isVideo: boolean;
		folder: string;
		status: UploadStatus;
		progress: number;
		error: string;
	}

	let {
		id = 'movie-upload',
		roots = [],
		allowed = true,
		triggerLabel = 'Subir película',
		triggerClass = 'button button-secondary movie-upload-trigger'
	}: {
		id?: string;
		roots?: UploadTargetRoot[];
		allowed?: boolean;
		triggerLabel?: string;
		triggerClass?: string;
	} = $props();

	const titleID = $derived(`${id}-title`);
	const destinationOptions = $derived(roots.filter((root) => root.library_kind === 'video'));
	const preferredRoot = $derived(
		[...destinationOptions].sort(
			(left, right) =>
				right.movie_count - left.movie_count || left.path.localeCompare(right.path)
		)[0] ?? null
	);
	const showDestinationSelect = $derived(destinationOptions.length > 1);

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
	let createFolders = $state(true);
	let selectedRootPath = $state(untrack(() => preferredRoot?.path ?? ''));
	let currentXHR: XMLHttpRequest | null = null;

	const videos = $derived(
		entries
			.filter((entry) => entry.isVideo)
			.map((entry) => ({ name: entry.name, folder: entry.folder }))
	);
	const selectedRoot = $derived(
		destinationOptions.find((root) => root.path === selectedRootPath) ?? preferredRoot
	);

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
		entries = accepted.map((file) => {
			const isVideo = isUploadVideoName(file.name);
			return {
				file,
				name: file.name,
				size: file.size,
				isVideo,
				folder: isVideo ? uploadFileStem(file.name) : '',
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

	function effectiveSubdir(entry: UploadEntry): string {
		return uploadSubdirForFile(entry.name, createFolders, videos).trim();
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
		if (!selectedRoot) {
			message = 'No hay una carpeta de biblioteca de vídeo disponible.';
			return;
		}
		for (const entry of entries) {
			if (!validUploadComponent(entry.name)) {
				message = `El nombre de «${entry.name}» no es válido.`;
				return;
			}
			if (createFolders && entry.isVideo && !validUploadComponent(entry.folder.trim())) {
				message = `La carpeta de «${entry.name}» no es válida.`;
				return;
			}
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
			entry.status = 'uploading';
			entry.progress = 0;
			try {
				await uploadEntry(entry);
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

	function uploadEntry(entry: UploadEntry): Promise<void> {
		if (!selectedRoot) {
			return Promise.reject(new Error('No hay una carpeta de biblioteca de vídeo disponible.'));
		}
		const xhr = new XMLHttpRequest();
		currentXHR = xhr;
		return new Promise((resolve, reject) => {
			xhr.open(
				'PUT',
				buildUploadURL({
					library: selectedRoot.library_id,
					dir: selectedRoot.path,
					name: entry.name,
					subdir: effectiveSubdir(entry)
				})
			);
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

<span class="movie-upload-trigger-wrap" hidden={!jsReady}>
	<button
		class={triggerClass}
		type="button"
		disabled={processing}
		aria-haspopup="dialog"
		aria-controls={id}
		bind:this={openButton}
		onclick={openDialog}>{triggerLabel}</button
	>
</span>

<dialog
	bind:this={dialogEl}
	{id}
	class="movie-upload-dialog"
	open={dialogOpen}
	aria-labelledby={titleID}
	onclick={handleClick}
	oncancel={handleCancel}
	onclose={handleClose}
>
	<div class="movie-upload-body">
		<header class="movie-upload-head">
			<div>
				<p class="kicker">Subida guiada</p>
				<h2 id={titleID}>Subir película</h2>
			</div>
			<button
				class="movie-upload-close"
				type="button"
				bind:this={closeButton}
				disabled={processing}
				onclick={closeDialog}
				aria-label="Cerrar">×</button
			>
		</header>

		<p class="movie-upload-warning">No cierres la página hasta que terminen las subidas.</p>

		{#if !allowed}
			<p class="error" role="alert">
				La modificación de archivos está desactivada. Actívala en Administración → Biblioteca.
			</p>
		{/if}

		<label>
			<span>Destino</span>
			{#if showDestinationSelect}
				<select bind:value={selectedRootPath} disabled={processing}>
					{#each destinationOptions as root (root.path)}
						<option value={root.path}>{root.library_name} · {root.path}</option>
					{/each}
				</select>
			{:else if selectedRoot}
				<span class="movie-upload-destination"
					>{selectedRoot.library_name} · {selectedRoot.path}</span
				>
			{:else}
				<span class="error">No hay raíces de biblioteca de vídeo disponibles.</span>
			{/if}
		</label>

		<label class="movie-upload-check">
			<input type="checkbox" bind:checked={createFolders} disabled={processing} />
			<span>Crear una carpeta para cada película</span>
		</label>

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
			<ul class="movie-upload-list">
				{#each entries as entry, index (`${entry.name}-${entry.status}-${index}`)}
					<li
						class="movie-upload-item"
						class:movie-upload-item--error={entry.status === 'error'}
					>
						<div class="movie-upload-file">
							<span class="movie-upload-name">{entry.name}</span>
							<span class="movie-upload-size">{formatUploadSize(entry.size)}</span>
						</div>
						{#if createFolders}
							<label class="movie-upload-folder">
								<span>Carpeta</span>
								{#if entry.isVideo}
									<input type="text" bind:value={entry.folder} disabled={processing} />
								{:else}
									<span class="movie-upload-folder-value">
										{effectiveSubdir(entry) || 'Raíz'}
									</span>
								{/if}
							</label>
						{:else}
							<p class="movie-upload-root">Se subirá a la raíz de la biblioteca.</p>
						{/if}
						<progress
							class="movie-upload-progress"
							value={entry.progress}
							max="100"
							aria-label={`Progreso de ${entry.name}`}
						></progress>
						<p class="movie-upload-status" aria-live="polite">{statusLabel(entry)}</p>
					</li>
				{/each}
			</ul>
		{/if}

		{#if message}
			<p class="error" role="alert">{message}</p>
		{/if}

		<div class="movie-upload-actions">
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
				Subida terminada; el escaneo identificará las películas en unos minutos.
			</p>
		{/if}
	</div>
</dialog>

<style>
	.movie-upload-trigger-wrap[hidden] {
		display: none;
	}

	.movie-upload-trigger {
		margin-top: 0;
		padding: 0.45rem 0.75rem;
		font-size: 0.68rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.movie-upload-dialog {
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

	.movie-upload-dialog::backdrop {
		background: rgb(20 10 11 / 65%);
	}

	.movie-upload-body {
		display: grid;
		gap: 0.8rem;
		padding: 1rem;
	}

	.movie-upload-head {
		display: flex;
		align-items: start;
		justify-content: space-between;
		gap: 1rem;
	}

	.movie-upload-head h2 {
		margin: 0.2rem 0 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.45rem;
		line-height: 1.1;
	}

	.movie-upload-close {
		padding: 0.1rem 0.4rem;
		border: 0;
		background: transparent;
		color: var(--texto-suave);
		font-size: 1.4rem;
		line-height: 1;
		cursor: pointer;
	}

	.movie-upload-warning {
		margin: 0;
		padding: 0.5rem 0.65rem;
		border-left: 3px solid var(--dorado);
		background: var(--tinte);
		color: var(--texto-suave);
		font-size: 0.82rem;
	}

	.movie-upload-body > label,
	.movie-upload-folder {
		display: grid;
		gap: 0.3rem;
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--granate);
	}

	.movie-upload-body input[type='text'],
	.movie-upload-body input[type='file'],
	.movie-upload-body select {
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

	.movie-upload-body input[type='file'] {
		padding: 0.3rem 0;
		border: 0;
		background: transparent;
	}

	.movie-upload-check {
		grid-template-columns: auto 1fr;
		align-items: center;
		text-transform: none;
		letter-spacing: normal;
		font-size: 0.85rem;
		font-weight: 600;
	}

	.movie-upload-check input {
		width: auto;
	}

	.movie-upload-destination,
	.movie-upload-folder-value {
		padding: 0.45rem 0;
		color: var(--texto-tenue);
		font-size: 0.82rem;
		font-weight: 400;
		letter-spacing: normal;
		text-transform: none;
	}

	.movie-upload-list {
		display: grid;
		gap: 0.55rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.movie-upload-item {
		padding: 0.55rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--tinte);
	}

	.movie-upload-item--error {
		border-color: var(--naranja);
	}

	.movie-upload-file {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 0.4rem;
	}

	.movie-upload-name {
		font-weight: 600;
		color: var(--granate);
		overflow-wrap: anywhere;
	}

	.movie-upload-size {
		color: var(--texto-apagado);
		font-size: 0.75rem;
	}

	.movie-upload-folder {
		margin-top: 0.45rem;
	}

	.movie-upload-root {
		margin: 0.45rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.78rem;
	}

	.movie-upload-progress {
		width: 100%;
		height: 0.5rem;
		margin-top: 0.45rem;
		accent-color: var(--naranja);
	}

	.movie-upload-status {
		margin: 0.3rem 0 0;
		color: var(--texto-tenue);
		font-size: 0.75rem;
		overflow-wrap: anywhere;
	}

	.movie-upload-item--error .movie-upload-status {
		color: var(--naranja);
	}

	.movie-upload-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.movie-upload-actions .button,
	.movie-upload-actions .button-danger {
		margin: 0;
	}
</style>
