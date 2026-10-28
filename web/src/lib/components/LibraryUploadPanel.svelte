<script lang="ts">
	import { untrack } from 'svelte';
	import type { Library } from '$lib/api';
	import {
		UPLOAD_ACCEPT,
		buildUploadURL,
		formatUploadSize,
		isUploadableFileName,
		uploadErrorMessage
	} from '$lib/uploads';
	import FolderBrowser from './FolderBrowser.svelte';

	type UploadStatus = 'queued' | 'uploading' | 'done' | 'error';

	interface UploadEntry {
		file: File;
		name: string;
		size: number;
		status: UploadStatus;
		progress: number;
		error: string;
	}

	let { library }: { library: Library } = $props();

	const firstPath = $derived(library.paths?.[0]?.path ?? library.path ?? '');
	let destination = $state(untrack(() => library.paths?.[0]?.path ?? library.path ?? ''));
	let subdir = $state('');
	let selectedFiles = $state<File[]>([]);
	let entries = $state<UploadEntry[]>([]);
	let processing = $state(false);
	let finished = $state(false);
	let message = $state('');
	let browserOpen = $state(false);
	let currentXHR: XMLHttpRequest | null = null;
	const browserStart = $derived(destination.trim() || firstPath);
	const browserRoots = $derived([{ name: library.name, path: browserStart }]);

	function chooseFiles(event: Event): void {
		const input = event.currentTarget as HTMLInputElement;
		const all = Array.from(input.files ?? []);
		const accepted = all.filter((file) => isUploadableFileName(file.name));
		selectedFiles = accepted;
		entries = accepted.map((file) => ({
			file,
			name: file.name,
			size: file.size,
			status: 'queued',
			progress: 0,
			error: ''
		}));
		finished = false;
		message =
			accepted.length === all.length
				? ''
				: 'Se han descartado los archivos con una extensión no admitida.';
	}

	function startUpload(): void {
		if (processing) {
			return;
		}
		if (!destination.trim()) {
			message = 'Elige una carpeta de destino.';
			return;
		}
		if (selectedFiles.length === 0) {
			message = 'Elige al menos un archivo.';
			return;
		}
		entries = selectedFiles.map((file) => ({
			file,
			name: file.name,
			size: file.size,
			status: 'queued',
			progress: 0,
			error: ''
		}));
		message = '';
		finished = false;
		processing = true;
		void runQueue();
	}

	async function runQueue(): Promise<void> {
		for (const entry of entries) {
			if (entry.status !== 'queued') {
				continue;
			}
			entry.status = 'uploading';
			entry.progress = 0;
			try {
				await uploadEntry(entry);
				entry.status = 'done';
				entry.progress = 100;
			} catch (reason) {
				entry.status = 'error';
				entry.error = reason instanceof Error ? reason.message : 'No se pudo subir el archivo.';
			}
		}
		currentXHR = null;
		processing = false;
		finished = true;
	}

	function uploadEntry(entry: UploadEntry): Promise<void> {
		const xhr = new XMLHttpRequest();
		currentXHR = xhr;
		return new Promise((resolve, reject) => {
			xhr.open(
				'PUT',
				buildUploadURL({
					library: library.id,
					dir: destination.trim(),
					name: entry.name,
					subdir: subdir.trim()
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
				reject(new Error(uploadErrorMessage(xhr.status, responsePayload(xhr.responseText))));
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

	function cancelCurrent(): void {
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

<details class="library-action library-upload">
	<summary class="button button-secondary">Subir archivos</summary>
	<div class="library-upload-body">
		<p class="admin-note">
			Las subidas grandes pueden tardar; no cierres la página hasta que terminen.
		</p>

		<label>
			<span>Carpeta de destino</span>
			<input type="text" bind:value={destination} placeholder="/media/peliculas" />
		</label>
		<button
			type="button"
			class="button button-secondary"
			onclick={() => (browserOpen = true)}
		>
			Elegir carpeta…
		</button>

		<label>
			<span>Carpeta nueva (opcional)</span>
			<input type="text" bind:value={subdir} placeholder="Temporada 1" />
		</label>

		<label>
			<span>Archivos</span>
			<input
				type="file"
				accept={UPLOAD_ACCEPT}
				multiple
				disabled={processing}
				onchange={chooseFiles}
			/>
		</label>

		{#if message}
			<p class="error" role="alert">{message}</p>
		{/if}

		<div class="upload-controls">
			<button
				type="button"
				class="button"
				disabled={processing || selectedFiles.length === 0}
				onclick={startUpload}>Subir</button
			>
			{#if processing}
				<button type="button" class="button-danger" onclick={cancelCurrent}>
					Cancelar la subida en curso
				</button>
			{/if}
		</div>

		{#if entries.length > 0}
			<ul class="upload-list">
				{#each entries as entry, index (`${entry.name}-${index}`)}
					<li class="upload-item" class:upload-item--error={entry.status === 'error'}>
						<div class="upload-item-head">
							<span class="upload-name">{entry.name}</span>
							<span class="upload-size">{formatUploadSize(entry.size)}</span>
						</div>
						<progress
							class="upload-progress"
							value={entry.progress}
							max="100"
							aria-label={`Progreso de ${entry.name}`}
						></progress>
						<p class="upload-status" aria-live="polite">{statusLabel(entry)}</p>
					</li>
				{/each}
			</ul>
		{/if}

		{#if finished && entries.some((entry) => entry.status === 'done')}
			<p class="success" role="status">
				Subida terminada; el escaneo añadirá los archivos en unos minutos.
			</p>
		{/if}
	</div>
</details>

<FolderBrowser
	open={browserOpen}
	roots={browserRoots}
	onclose={() => (browserOpen = false)}
	onselect={(path) => {
		destination = path;
		browserOpen = false;
	}}
/>

<style>
	.library-upload-body {
		display: grid;
		gap: 0.65rem;
		margin-top: 0.6rem;
		padding: 0.7rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--tinte);
	}
	.library-upload-body label {
		display: grid;
		gap: 0.3rem;
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--granate);
	}
	.library-upload-body input[type='text'] {
		width: 100%;
		padding: 0.55rem 0.65rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--crema-clara);
		color: var(--tinta);
		font-family: 'Inter', system-ui, sans-serif;
		font-size: 0.9rem;
		letter-spacing: normal;
		text-transform: none;
	}
	.library-upload-body input[type='file'] {
		font-size: 0.8rem;
		text-transform: none;
	}
	.library-upload-body .admin-note {
		margin: 0;
	}
	.upload-controls {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.upload-controls .button,
	.upload-controls .button-danger {
		margin: 0;
	}
	.upload-list {
		display: grid;
		gap: 0.55rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.upload-item {
		padding: 0.55rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--superficie);
	}
	.upload-item--error {
		border-color: var(--naranja);
	}
	.upload-item-head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 0.4rem;
	}
	.upload-name {
		font-weight: 600;
		color: var(--granate);
		overflow-wrap: anywhere;
	}
	.upload-size {
		color: var(--texto-apagado);
		font-size: 0.75rem;
	}
	.upload-progress {
		width: 100%;
		height: 0.55rem;
		margin-top: 0.35rem;
		accent-color: var(--naranja);
	}
	.upload-status {
		margin: 0.3rem 0 0;
		color: var(--texto-tenue);
		font-size: 0.75rem;
		overflow-wrap: anywhere;
	}
	.upload-item--error .upload-status {
		color: var(--naranja);
	}
</style>
