<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PlayResponse } from '$lib/api';
	import { formatAudioTrack, formatSubtitleTrack } from '$lib/media';

	let { options }: { options: PlayResponse } = $props();

	const audioTracks = $derived(options.audio_tracks ?? []);
	const qualities = $derived(options.qualities ?? []);
	const subtitles = $derived(options.subtitles ?? []);

	function submitOnChange(event: Event) {
		const select = event.currentTarget as HTMLSelectElement;
		select.form?.requestSubmit();
	}
</script>

<section class="playback-options" aria-label="Opciones de reproducción">
	<h2>Reproducción</h2>
	<form method="POST" action="?/preferences" use:enhance>
		<label>
			<span>Audio</span>
			<select
				name="audio_index"
				disabled={audioTracks.length === 0}
				onchange={submitOnChange}
			>
				{#each audioTracks as track (track.index)}
					<option
						value={track.index}
						selected={track.index === (options.selected_audio ?? 0)}
					>
						{formatAudioTrack(track)}
					</option>
				{/each}
				{#if audioTracks.length === 0}<option value="">No disponible</option>{/if}
			</select>
		</label>
		<label>
			<span>Subtítulos</span>
			<select
				name="subtitle_id"
				disabled={subtitles.length === 0}
				onchange={submitOnChange}
			>
				<option value="" selected={(options.selected_subtitle ?? null) === null}>
					Sin subtítulos
				</option>
				{#each subtitles as subtitle (subtitle.id)}
					<option value={subtitle.id} selected={subtitle.id === options.selected_subtitle}>
						{formatSubtitleTrack(subtitle)}
					</option>
				{/each}
			</select>
		</label>
		<label>
			<span>Calidad</span>
			<select
				name="quality"
				disabled={qualities.length === 0}
				onchange={submitOnChange}
			>
				{#each qualities as quality (quality.value)}
					<option
						value={quality.value}
						selected={quality.value === (options.selected_quality ?? 'auto')}
					>
						{quality.label}
					</option>
				{/each}
			</select>
		</label>
		<button type="submit" class="visually-hidden">Guardar</button>
		<noscript><button type="submit" class="noscript-save">Guardar</button></noscript>
	</form>
	<form method="POST" action="?/preferences" use:enhance>
		<input type="hidden" name="reset" value="1" />
		<button type="submit" class="text-link">Restablecer</button>
	</form>
</section>

<style>
	.playback-options {
		margin: 1.2rem 0;
		padding: 0.9rem 1rem;
		border: 1px solid rgba(123, 30, 36, 0.22);
		border-radius: 8px;
		background: rgba(246, 234, 215, 0.35);
		color: #2b1b15;
	}
	.playback-options h2 {
		margin: 0 0 0.7rem;
		font-size: 1rem;
	}
	form {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.75rem;
	}
	label {
		display: grid;
		gap: 0.25rem;
		font-size: 0.82rem;
		min-width: 150px;
	}
	select {
		min-height: 40px;
		padding: 0.4rem 0.5rem;
		border: 1px solid rgba(80, 53, 38, 0.35);
		border-radius: 6px;
		background: #fffaf0;
		color: #2b1b15;
		font: inherit;
	}
	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
	.noscript-save {
		min-height: 40px;
		padding: 0.35rem 0.7rem;
		border: 1px solid rgba(80, 53, 38, 0.35);
		border-radius: 6px;
		background: #7b1e24;
		color: #fff4e1;
		font: inherit;
		cursor: pointer;
	}
	.text-link {
		border: 0;
		background: none;
		color: #7b1e24;
		text-decoration: underline;
		cursor: pointer;
		min-height: 40px;
		font: inherit;
	}
</style>
