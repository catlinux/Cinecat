<script lang="ts">
	import type { FileInfoResult, PlaybackSubtitle } from '$lib/api';
	import {
		formatAudioChannels,
		formatBitrate,
		formatBytes,
		formatDateTime,
		formatDurationSeconds,
		formatFps,
		formatLanguage,
		formatResolution,
		formatSampleRate
	} from '$lib/media';

	let {
		result,
		itemId = null,
		isAdmin = false,
		uploadedSubtitles = [],
		subtitleMessage = null,
		subtitleSuccess = false
	}: {
		result: FileInfoResult | Promise<FileInfoResult>;
		itemId?: number | null;
		isAdmin?: boolean;
		uploadedSubtitles?: PlaybackSubtitle[] | Promise<PlaybackSubtitle[]> | null;
		subtitleMessage?: string | null;
		subtitleSuccess?: boolean;
	} = $props();

	function subtitleLabel(track: PlaybackSubtitle): string {
		const parts = [formatLanguage(track.language) || 'Sin idioma'];
		if (track.title.trim()) {
			parts.push(track.title);
		}
		return parts.join(' · ');
	}

	const errorMessages = {
		not_found: 'No se encuentra el archivo en el servidor',
		unavailable: 'ffprobe no está disponible en el servidor',
		failed: 'No se pudo analizar el archivo'
	} as const;
</script>

{#snippet uploadedList(list: PlaybackSubtitle[])}
	<div class="file-uploaded-subtitles">
		<h4>Subidos</h4>
		{#if subtitleMessage}
			<p class={subtitleSuccess ? 'success' : 'error'} role={subtitleSuccess ? 'status' : 'alert'}>
				{subtitleMessage}
			</p>
		{/if}
		{#if list.length === 0}
			<p class="admin-note">No hay subtítulos subidos.</p>
		{:else}
			<ul class="file-uploaded-list">
				{#each list as track (track.id)}
					<li>
						<span>{subtitleLabel(track)}</span>
						<code>{track.id}</code>
						<form method="POST" action="?/subtitleDelete">
							<input type="hidden" name="subtitle_id" value={track.id} />
							<button type="submit" class="text-link">Eliminar</button>
						</form>
					</li>
				{/each}
			</ul>
		{/if}
		<form
			method="POST"
			action="?/subtitleUpload"
			enctype="multipart/form-data"
			class="subtitle-upload-form"
		>
			<label>
				<span>Archivo</span>
				<input type="file" name="file" accept=".srt,.vtt,.ass" required />
			</label>
			<label>
				<span>Idioma</span>
				<input name="language" maxlength="3" placeholder="es" />
			</label>
			<label>
				<span>Etiqueta</span>
				<input name="label" placeholder="Castellano" />
			</label>
			<button type="submit" class="button button-secondary">Subir subtítulo</button>
		</form>
	</div>
{/snippet}

{#snippet content(value: FileInfoResult)}
	{#if value.data}
		{@const file = value.data}
		<div class="file-panel-header">
			<span class="file-quality">{file.quality || '—'}</span>
			<dl class="file-summary">
				<div>
					<dt>Contenedor</dt>
					<dd>{file.container_long_name || file.container || '—'}</dd>
				</div>
				<div>
					<dt>Tamaño</dt>
					<dd>{formatBytes(file.size_bytes)}</dd>
				</div>
				<div>
					<dt>Duración</dt>
					<dd>{formatDurationSeconds(file.duration_seconds) || '—'}</dd>
				</div>
				<div>
					<dt>Tasa total</dt>
					<dd>{formatBitrate(file.bit_rate, true)}</dd>
				</div>
				<div>
					<dt>Modificado</dt>
					<dd>{formatDateTime(file.modified_at) || '—'}</dd>
				</div>
			</dl>
		</div>

		<p class="file-path">
			<strong>{file.file_name}</strong>
			{#if file.path}
				<code>{file.path}</code>
			{/if}
		</p>

		<section class="file-section">
			{#if file.video.length > 0}
				<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
				<div
					class="file-table-scroll"
					role="region"
					tabindex="0"
					aria-label="Tabla de vídeo, desplazable"
				>
					<table class="file-table">
						<caption>Vídeo</caption>
						<thead>
							<tr>
								<th scope="col">Códec</th>
								<th scope="col">Perfil</th>
								<th scope="col">Resolución</th>
								<th scope="col">FPS</th>
								<th scope="col">Color</th>
								<th scope="col">HDR</th>
								<th scope="col">Tasa</th>
							</tr>
						</thead>
						<tbody>
							{#each file.video as stream, index (index)}
								<tr>
									<th scope="row" title={stream.codec_long_name || undefined}>
										{stream.codec || '—'}
									</th>
									<td>{stream.profile || '—'}</td>
									<td>{formatResolution(stream.width, stream.height, stream.display_aspect_ratio)}</td>
									<td>{formatFps(stream.frame_rate)}</td>
									<td>{stream.bit_depth > 0 ? `${stream.bit_depth} bits` : '—'}</td>
									<td>{stream.hdr || '—'}</td>
									<td>{formatBitrate(stream.bit_rate)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<p class="admin-note">Sin pistas de vídeo.</p>
			{/if}
		</section>

		<section class="file-section">
			{#if file.audio.length > 0}
				<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
				<div
					class="file-table-scroll"
					role="region"
					tabindex="0"
					aria-label="Tabla de audio, desplazable"
				>
					<table class="file-table">
						<caption>Audio</caption>
						<thead>
							<tr>
								<th scope="col">Códec</th>
								<th scope="col">Canales</th>
								<th scope="col">Frecuencia</th>
								<th scope="col">Tasa</th>
								<th scope="col">Idioma</th>
								<th scope="col">Por defecto</th>
							</tr>
						</thead>
						<tbody>
							{#each file.audio as stream, index (index)}
								<tr>
									<th scope="row" title={stream.codec_long_name || undefined}>
										{stream.codec || '—'}
									</th>
									<td>{formatAudioChannels(stream.channels, stream.channel_layout) || '—'}</td>
									<td>{formatSampleRate(stream.sample_rate)}</td>
									<td>{formatBitrate(stream.bit_rate)}</td>
									<td>{formatLanguage(stream.language) || 'Sin idioma'}</td>
									<td>
										{#if stream.default}
											<span class="file-tag">Por defecto</span>
										{:else}
											—
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<p class="admin-note">Sin pistas de audio.</p>
			{/if}
		</section>

		<section class="file-section">
			<h3>Subtítulos</h3>
			{#if file.subtitles.length > 0}
				<ul class="file-subtitles">
					{#each file.subtitles as stream, index (index)}
						<li>
							<span class="file-subtitle-language">
								{formatLanguage(stream.language) || 'Sin idioma'}
							</span>
							<span class="file-subtitle-codec">{stream.codec || '—'}</span>
							{#if stream.forced}
								<span class="file-tag">forzado</span>
							{/if}
							{#if stream.default}
								<span class="file-tag">por defecto</span>
							{/if}
							{#if !stream.text}
								<span class="file-tag">imagen</span>
							{/if}
						</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Sin subtítulos incrustados</p>
			{/if}

			{#if isAdmin && itemId !== null}
				{#if uploadedSubtitles instanceof Promise}
					{#await uploadedSubtitles}
						<p class="admin-note" role="status">Cargando subtítulos subidos…</p>
					{:then list}
						{@render uploadedList(list)}
					{:catch}
						<p class="error" role="alert">No se pudieron cargar los subtítulos subidos.</p>
					{/await}
				{:else}
					{@render uploadedList(uploadedSubtitles ?? [])}
				{/if}
			{/if}
		</section>

		<p
			class="file-playback"
			class:file-playback--warning={!file.playback.supported}
			role={file.playback.supported ? undefined : 'status'}
		>
			<span>Reproducción:</span>
			{#if file.playback.hardware && file.playback.method === 'transcode'}
				Recodificación (gráfica)
			{:else}
				{file.playback.description || '—'}
			{/if}
		</p>
	{:else}
		<p class="error file-panel-error" role="alert">
			{errorMessages[value.error ?? 'failed']}
		</p>
	{/if}
{/snippet}

<details class="admin-card file-panel" open>
	<summary class="file-panel-summary">Archivo</summary>

	{#if result instanceof Promise}
		{#await result}
			<p class="admin-note" role="status">Analizando el archivo…</p>
		{:then value}
			{@render content(value)}
		{:catch}
			<p class="error file-panel-error" role="alert">No se pudo analizar el archivo</p>
		{/await}
	{:else}
		{@render content(result)}
	{/if}
</details>

<style>
	.file-uploaded-subtitles {
		margin-top: 1rem;
		padding-top: 0.8rem;
		border-top: 1px dashed rgba(123, 30, 36, 0.25);
	}

	.file-uploaded-subtitles h4 {
		margin: 0 0 0.5rem;
		font-size: 0.92rem;
	}

	.file-uploaded-list {
		display: grid;
		gap: 0.35rem;
		margin: 0 0 0.8rem;
		padding: 0;
		list-style: none;
	}

	.file-uploaded-list li {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		padding: 0.35rem 0.5rem;
		border: 1px solid rgba(123, 30, 36, 0.18);
		border-radius: 4px;
	}

	.file-uploaded-list code {
		color: var(--texto-suave, #6b5a4e);
		font-size: 0.78rem;
	}

	.subtitle-upload-form {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.6rem;
	}

	.subtitle-upload-form label {
		display: grid;
		gap: 0.2rem;
		font-size: 0.78rem;
	}

	.subtitle-upload-form input {
		min-height: 36px;
		padding: 0.3rem 0.4rem;
		border: 1px solid rgba(80, 53, 38, 0.35);
		border-radius: 4px;
		background: #fffaf0;
		color: inherit;
		font: inherit;
	}

	.subtitle-upload-form input[type='file'] {
		max-width: 15rem;
	}
</style>
