<script lang="ts">
	import type { FileInfoResult } from '$lib/api';
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

	let { result }: { result: FileInfoResult | Promise<FileInfoResult> } = $props();

	const errorMessages = {
		not_found: 'No se encuentra el archivo en el servidor',
		unavailable: 'ffprobe no está disponible en el servidor',
		failed: 'No se pudo analizar el archivo'
	} as const;
</script>

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
