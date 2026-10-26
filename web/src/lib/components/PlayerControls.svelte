<script lang="ts">
	import type { PlaybackAudioTrack, PlaybackQuality, PlaybackSubtitle } from '$lib/api';
	import { formatPlayerTime, formatSpeed, pointerPosition, PLAYER_SPEEDS } from '$lib/player';
	import { formatAudioCodec } from '$lib/media';
	import PlayerMenu from './PlayerMenu.svelte';

	let {
		playing,
		currentTime,
		duration,
		buffered,
		volume,
		muted,
		speed,
		hidden,
		audioTracks = [],
		selectedAudio = 0,
		qualities = [],
		selectedQuality = 'auto',
		subtitles = [],
		selectedSubtitle = null,
		nextEpisodeUrl = null,
		canPictureInPicture = false,
		canFullscreen = true,
		onToggle,
		onSeek,
		onSeekBy,
		onVolume,
		onMute,
		onSpeed,
		onAudio,
		onSubtitle,
		onQuality,
		onPictureInPicture,
		onFullscreen,
		onActivity,
		onMenuChange,
		onFocusChange
	}: {
		playing: boolean;
		currentTime: number;
		duration: number;
		buffered: number;
		volume: number;
		muted: boolean;
		speed: number;
		hidden: boolean;
		audioTracks?: PlaybackAudioTrack[];
		selectedAudio?: number;
		qualities?: PlaybackQuality[];
		selectedQuality?: string;
		subtitles?: PlaybackSubtitle[];
		selectedSubtitle?: string | null;
		nextEpisodeUrl?: string | null;
		canPictureInPicture?: boolean;
		canFullscreen?: boolean;
		onToggle: () => void;
		onSeek: (value: number) => void;
		onSeekBy: (delta: number) => void;
		onVolume: (value: number) => void;
		onMute: () => void;
		onSpeed: (value: number) => void;
		onAudio: (index: number) => void;
		onSubtitle: (id: string) => void;
		onQuality: (value: string) => void;
		onPictureInPicture: () => void;
		onFullscreen: () => void;
		onActivity: () => void;
		onMenuChange: (open: boolean) => void;
		onFocusChange: (focused: boolean) => void;
	} = $props();

	let openMenu = $state<string | null>(null);
	let shortcutsOpen = $state(false);
	let hoverTime = $state<number | null>(null);

	const progressPercent = $derived(duration > 0 ? (currentTime / duration) * 100 : 0);
	const bufferedPercent = $derived(duration > 0 ? Math.min(100, (buffered / duration) * 100) : 0);
	const audioLabel = $derived(
		audioTracks.find((track) => track.index === selectedAudio)?.language_name ?? 'Audio'
	);
	const subtitleLabel = $derived(
		selectedSubtitle
			? subtitles.find((track) => track.id === selectedSubtitle)?.language_name ?? 'Subtítulos'
			: 'Subtítulos'
	);
	const qualityLabel = $derived(
		qualities.find((quality) => quality.value === selectedQuality)?.label ?? 'Calidad'
	);

	function updateMenu(name: string) {
		openMenu = openMenu === name ? null : name;
		shortcutsOpen = false;
		onMenuChange(openMenu !== null || shortcutsOpen);
	}

	function closeMenus() {
		openMenu = null;
		shortcutsOpen = false;
		onMenuChange(false);
	}

	function chooseAudio(value: string) {
		closeMenus();
		onAudio(Number(value));
	}

	function chooseQuality(value: string) {
		closeMenus();
		onQuality(value);
	}

	function chooseSubtitle(value: string) {
		closeMenus();
		onSubtitle(value);
	}

	function chooseSpeed(value: string) {
		closeMenus();
		onSpeed(Number(value));
	}

	function audioLabelFor(track: PlaybackAudioTrack): string {
		const parts = [track.language_name || 'Desconocido', formatAudioCodec(track.codec)];
		if (track.channel_layout) {
			parts.push(track.channel_layout);
		} else if (track.channels > 0) {
			parts.push(`${track.channels} canales`);
		}
		if (track.title) {
			parts.push(track.title);
		}
		return parts.join(' · ');
	}

	function subtitleLabelFor(track: PlaybackSubtitle): string {
		const parts = [track.language_name || 'Desconocido'];
		if (track.title) {
			parts.push(track.title);
		}
		if (track.forced) {
			parts.push('Forzados');
		}
		return parts.join(' · ');
	}

	function progressInput(event: Event) {
		onActivity();
		const input = event.currentTarget as HTMLInputElement;
		onSeek(Number(input.value));
	}

	function progressHover(event: MouseEvent) {
		onActivity();
		const target = event.currentTarget as HTMLElement;
		const rect = target.getBoundingClientRect();
		hoverTime = pointerPosition(event.clientX, rect.left, rect.width, duration);
	}

	function activeSubtitleCount(): number {
		return selectedSubtitle ? 1 : 0;
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === 'Escape') {
			closeMenus();
		}
	}}
/>

<div
	class="controls"
	class:hidden
	role="group"
	aria-label="Controles del reproductor"
	onmousemove={onActivity}
	onfocusin={() => onFocusChange(true)}
	onfocusout={(event) => {
		if (!(event.currentTarget as HTMLElement).contains(event.relatedTarget as Node | null)) {
			onFocusChange(false);
		}
	}}
>
	<button
		type="button"
		class="control-button primary"
		aria-label={playing ? 'Pausar' : 'Reproducir'}
		onclick={() => onToggle()}
	>
		{#if playing}
			<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 5h4v14H7zM13 5h4v14h-4z" /></svg>
		{:else}
			<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 5v14l11-7z" /></svg>
		{/if}
	</button>
	<button type="button" class="control-button" aria-label="Retroceder 10 segundos" onclick={() => onSeekBy(-10)}>
		<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M11 5 4 12l7 7v-4.1a5 5 0 1 1 4 1.7v2a7 7 0 1 0-4-12.6z" /><text x="11" y="15" font-size="6" text-anchor="middle">10</text></svg>
	</button>
	<button type="button" class="control-button" aria-label="Avanzar 10 segundos" onclick={() => onSeekBy(10)}>
		<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m13 5 7 7-7 7v-4.1a5 5 0 1 0-4 1.7v2a7 7 0 1 1 4-12.6z" /><text x="13" y="15" font-size="6" text-anchor="middle">10</text></svg>
	</button>

	<span class="time" aria-live="off">
		{formatPlayerTime(currentTime)} / {formatPlayerTime(duration)}
	</span>

	<div
		class="progress-wrap"
		role="group"
		aria-label="Progreso de reproducción"
		onmousemove={progressHover}
		onmouseleave={() => (hoverTime = null)}
	>
		<div class="buffered" style={`width: ${bufferedPercent}%`}></div>
		<div class="played" style={`width: ${progressPercent}%`}></div>
		<input
			class="progress"
			type="range"
			min="0"
			max={duration > 0 ? duration : 0}
			step="0.1"
			value={currentTime}
			aria-label="Posición de reproducción"
			aria-valuetext={`${formatPlayerTime(currentTime)} de ${formatPlayerTime(duration)}`}
			oninput={progressInput}
			onchange={progressInput}
		/>
		{#if hoverTime !== null && duration > 0}
			<span
				class="hover-time"
				style={`left: ${Math.min(100, Math.max(0, (hoverTime / duration) * 100))}%`}
			>
				{formatPlayerTime(hoverTime)}
			</span>
		{/if}
	</div>

	<div class="volume-control">
		<button type="button" class="control-button" aria-label={muted ? 'Activar sonido' : 'Silenciar'} onclick={() => onMute()}>
			{#if muted || volume === 0}
				<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9v6h4l5 4V5L8 9H4zm12.5 3 2.5 2.5-1 1L15.5 13 13 15.5l-1-1 2.5-2.5L12 9.5l1-1 2.5 2.5L18 8.5l1 1z" /></svg>
			{:else}
				<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9v6h4l5 4V5L8 9H4zm12 3a4 4 0 0 0-2-3.5v7A4 4 0 0 0 16 12zm-2-7v2a6 6 0 0 1 0 12v2a8 8 0 0 0 0-16z" /></svg>
			{/if}
		</button>
		<input
			class="volume"
			type="range"
			min="0"
			max="1"
			step="0.05"
			value={muted ? 0 : volume}
			aria-label="Volumen"
			oninput={(event) => onVolume(Number((event.currentTarget as HTMLInputElement).value))}
		/>
	</div>

	<PlayerMenu
		id="speed-menu"
		label="Velocidad de reproducción"
		triggerLabel={`${formatSpeed(speed)}`}
		open={openMenu === 'speed'}
		items={PLAYER_SPEEDS.map((value) => ({
			value: String(value),
			label: formatSpeed(value),
			checked: value === speed
		}))}
		onToggle={() => updateMenu('speed')}
		onSelect={chooseSpeed}
		onClose={closeMenus}
	/>

	{#if audioTracks.length > 0}
		<PlayerMenu
			id="audio-menu"
			label="Pista de audio"
			triggerLabel={`Audio · ${audioLabel}`}
			open={openMenu === 'audio'}
			items={audioTracks.map((track) => ({
				value: String(track.index),
				label: audioLabelFor(track),
				checked: track.index === selectedAudio
			}))}
			onToggle={() => updateMenu('audio')}
			onSelect={chooseAudio}
			onClose={closeMenus}
		/>
	{/if}

	{#if subtitles.length > 0}
		<PlayerMenu
			id="subtitle-menu"
			label="Subtítulos"
			triggerLabel={`Subtítulos${activeSubtitleCount() ? ' · activos' : ''}`}
			open={openMenu === 'subtitles'}
			items={[
				{ value: '', label: 'Sin subtítulos', checked: selectedSubtitle === null },
				...subtitles.map((track) => ({
					value: track.id,
					label: subtitleLabelFor(track),
					checked: track.id === selectedSubtitle
				}))
			]}
			onToggle={() => updateMenu('subtitles')}
			onSelect={chooseSubtitle}
			onClose={closeMenus}
		/>
	{/if}

	{#if qualities.length > 0}
		<PlayerMenu
			id="quality-menu"
			label="Calidad de reproducción"
			triggerLabel={qualityLabel}
			open={openMenu === 'quality'}
			items={qualities.map((quality) => ({
				value: quality.value,
				label: quality.label,
				checked: quality.value === selectedQuality,
				hint: quality.value === 'original' ? undefined : 'Recodificado'
			}))}
			onToggle={() => updateMenu('quality')}
			onSelect={chooseQuality}
			onClose={closeMenus}
		/>
	{/if}

	{#if canPictureInPicture}
		<button type="button" class="control-button" aria-label="Imagen en imagen" onclick={() => onPictureInPicture()}>
			<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 5h18v14H3V5zm2 2v10h14V7H5zm7 4h5v4h-5v-4z" /></svg>
		</button>
	{/if}

	{#if canFullscreen}
		<button type="button" class="control-button" aria-label="Pantalla completa" onclick={() => onFullscreen()}>
			<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4h6v2H6v4H4V4zm10 0h6v6h-2V6h-4V4zM4 14h2v4h4v2H4v-6zm14 0h2v6h-6v-2h4v-4z" /></svg>
		</button>
	{/if}

	{#if nextEpisodeUrl}
		<button
			type="button"
			class="next-episode"
			onclick={() => {
				if (nextEpisodeUrl) {
					window.location.href = nextEpisodeUrl;
				}
			}}>Siguiente episodio</button
		>
	{/if}

	<button
		type="button"
		class="control-button shortcuts-trigger"
		aria-label="Atajos de teclado"
		aria-expanded={shortcutsOpen}
		onclick={() => {
			shortcutsOpen = !shortcutsOpen;
			openMenu = null;
			onMenuChange(shortcutsOpen);
		}}>?</button
	>
	{#if shortcutsOpen}
		<div class="shortcuts-panel" role="dialog" aria-label="Atajos de teclado">
			<strong>Atajos</strong>
			<p>Espacio / K: reproducir o pausar</p>
			<p>← / J y → / L: ±10 s (Mayús: ±30 s)</p>
			<p>↑ / ↓: volumen · M: silencio · F: pantalla completa</p>
			<p>C: subtítulos · , y .: fotograma · 0-9: salto</p>
		</div>
	{/if}
</div>

<style>
	.controls {
		position: absolute;
		inset: auto 0 0;
		display: flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.55rem 0.7rem;
		background: linear-gradient(transparent, rgba(12, 7, 6, 0.88));
		color: #f6ead7;
		z-index: 3;
		transition: opacity 160ms ease;
		flex-wrap: wrap;
	}
	.controls.hidden {
		opacity: 0;
		pointer-events: none;
	}
	.control-button,
	.next-episode {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 40px;
		min-height: 40px;
		padding: 0.35rem;
		border: 1px solid transparent;
		border-radius: 6px;
		background: transparent;
		color: inherit;
		font: inherit;
		cursor: pointer;
		text-decoration: none;
	}
	.control-button:hover,
	.control-button:focus-visible,
	.next-episode:hover,
	.next-episode:focus-visible {
		background: rgba(246, 234, 215, 0.16);
		border-color: rgba(246, 234, 215, 0.42);
		outline: none;
	}
	.control-button svg {
		width: 22px;
		height: 22px;
		fill: currentColor;
	}
	.primary {
		background: #7b1e24;
	}
	.time {
		font-variant-numeric: tabular-nums;
		font-size: 0.82rem;
		white-space: nowrap;
	}
	.progress-wrap {
		position: relative;
		flex: 1 1 180px;
		height: 28px;
		display: flex;
		align-items: center;
		min-width: 120px;
	}
	.progress-wrap::before {
		content: '';
		position: absolute;
		left: 0;
		right: 0;
		height: 4px;
		background: rgba(246, 234, 215, 0.25);
		border-radius: 4px;
	}
	.buffered,
	.played {
		position: absolute;
		left: 0;
		height: 4px;
		border-radius: 4px;
	}
	.buffered {
		background: rgba(246, 234, 215, 0.45);
	}
	.played {
		background: #d9a441;
	}
	.progress {
		position: relative;
		width: 100%;
		height: 24px;
		margin: 0;
		opacity: 0;
		cursor: pointer;
	}
	.progress-wrap:focus-within {
		outline: 2px solid #d9a441;
		outline-offset: 2px;
		border-radius: 4px;
	}
	.hover-time {
		position: absolute;
		bottom: 26px;
		transform: translateX(-50%);
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		background: rgba(12, 7, 6, 0.92);
		font-size: 0.75rem;
		pointer-events: none;
		white-space: nowrap;
	}
	.volume-control {
		display: flex;
		align-items: center;
		gap: 0.2rem;
	}
	.volume {
		width: 76px;
		accent-color: #d9a441;
	}
	.next-episode {
		margin-left: auto;
		border-color: rgba(246, 234, 215, 0.35);
		font-size: 0.82rem;
	}
	.shortcuts-trigger {
		font-weight: 700;
	}
	.shortcuts-panel {
		position: absolute;
		right: 0.6rem;
		bottom: 3.4rem;
		max-width: 300px;
		padding: 0.75rem 0.9rem;
		border-radius: 8px;
		background: rgba(12, 7, 6, 0.96);
		border: 1px solid rgba(246, 234, 215, 0.3);
		font-size: 0.78rem;
	}
	.shortcuts-panel p {
		margin: 0.25rem 0;
	}
	:global(.player-menu) {
		position: relative;
	}
	:global(.menu-panel) {
		position: absolute;
		right: 0;
		bottom: 3rem;
		z-index: 10;
		min-width: 210px;
		max-width: min(320px, 86vw);
		padding: 0.35rem;
		border-radius: 8px;
		background: rgba(12, 7, 6, 0.97);
		border: 1px solid rgba(246, 234, 215, 0.3);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
	}
	:global(.menu-panel button) {
		display: flex;
		width: 100%;
		justify-content: space-between;
		gap: 0.7rem;
		padding: 0.5rem 0.55rem;
		border: 0;
		border-radius: 5px;
		background: transparent;
		color: #f6ead7;
		font: inherit;
		font-size: 0.82rem;
		text-align: left;
		cursor: pointer;
	}
	:global(.menu-panel button:hover),
	:global(.menu-panel button:focus-visible),
	:global(.menu-panel button.active) {
		background: rgba(246, 234, 215, 0.14);
		outline: none;
	}
	:global(.menu-panel button.active) {
		color: #f0c878;
	}
	:global(.menu-panel small) {
		color: #d9a441;
	}
	@media (max-width: 700px) {
		.volume {
			display: none;
		}
		.time {
			order: 5;
			width: 100%;
			text-align: center;
		}
		.progress-wrap {
			order: 4;
			flex-basis: 100%;
		}
		.controls {
			gap: 0.15rem;
		}
	}
</style>
