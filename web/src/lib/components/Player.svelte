<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import type Hls from 'hls.js';
	import type { PlayResponse } from '$lib/api';
	import { chooseHlsStrategy } from '$lib/hls-strategy';
	import {
		absoluteSubtitleURL,
		clampSeekTime,
		countdownLabel,
		controlsShouldHide,
		formatSpeed,
		PLAYER_SPEEDS,
		shortcutAction
	} from '$lib/player';
	import {
		decideProgress,
		progressEnabled,
		resumeTarget,
		type ProgressDecision,
		type ProgressSnapshot
	} from '$lib/progress';
	import PlayerControls from './PlayerControls.svelte';

	let {
		src,
		mode,
		title = 'Vídeo',
		itemId = undefined,
		startAt = 0,
		options = null,
		nextEpisodeUrl = null
	}: {
		src: string;
		mode: 'direct' | 'remux';
		title?: string;
		itemId?: number;
		startAt?: number;
		options?: PlayResponse | null;
		nextEpisodeUrl?: string | null;
	} = $props();

	const volumeStorageKey = 'cinecat.player.volume';
	const mutedStorageKey = 'cinecat.player.muted';
	const speedStorageKey = 'cinecat.player.speed';

	let video: HTMLVideoElement;
	let container: HTMLElement;
	let errorMessage = $state<string | null>(null);
	let hls: Hls | null = null;
	let destroyed = false;
	let nativeControls = $state(true);

	let playing = $state(false);
	let currentTime = $state(0);
	let duration = $state(0);
	let buffered = $state(0);
	let volume = $state(1);
	let muted = $state(false);
	let speed = $state(1);
	let buffering = $state(false);
	let controlsHidden = $state(false);
	let lastActivity = Date.now();
	let menuOpen = false;
	let controlFocused = false;
	let activityTimer: ReturnType<typeof setInterval> | null = null;

	// La selección inicial viene de la carga del servidor; los cambios
	// posteriores navegan y reconstruyen el componente.
	// svelte-ignore state_referenced_locally
	let selectedSubtitle = $state<string | null>(options?.selected_subtitle ?? null);
	// svelte-ignore state_referenced_locally
	let selectedAudio = $state(options?.selected_audio ?? 0);
	// svelte-ignore state_referenced_locally
	let selectedQuality = $state(options?.selected_quality ?? 'auto');
	let notice = $state<string | null>(null);
	let noticeTimer: ReturnType<typeof setTimeout> | null = null;
	let countdown = $state<number | null>(null);
	let countdownTimer: ReturnType<typeof setInterval> | null = null;
	let clickTimer: ReturnType<typeof setTimeout> | null = null;
	let pictureInPictureEnabled = $state(false);

	let resumeApplied = false;
	let lastProgressAt: number | null = null;
	let beaconSent = false;

	const audioTracks = $derived(options?.audio_tracks ?? []);
	const qualities = $derived(options?.qualities ?? []);
	const subtitles = $derived(options?.subtitles ?? []);
	const canPictureInPicture = $derived(pictureInPictureEnabled);
	const selectedSubtitleLabel = $derived(
		selectedSubtitle
			? subtitles.find((subtitle) => subtitle.id === selectedSubtitle)?.language_name ?? ''
			: ''
	);

	// Reintentos ante errores fatales antes de rendirse: un salto puede tardar
	// unos segundos mientras el servidor relanza FFmpeg en el punto pedido.
	const maxNetworkRecoveries = 5;
	const maxMediaRecoveries = 2;

	onMount(() => {
		nativeControls = false;
		pictureInPictureEnabled = Boolean(document.pictureInPictureEnabled);
		if (video) {
			video.controls = false;
			loadLocalPreferences();
			if (mode === 'remux') {
				void setupHls();
			}
			void tick().then(() => applySubtitle(selectedSubtitle, false));
		}
		activityTimer = setInterval(() => {
			controlsHidden = controlsShouldHide(
				playing,
				menuOpen,
				controlFocused,
				lastActivity,
				Date.now()
			);
			if (!controlsHidden) {
				controlsHidden = false;
			}
		}, 250);

		const onVisibility = () => {
			if (document.visibilityState === 'hidden') {
				sendBeaconProgress();
			} else {
				beaconSent = false;
			}
		};
		const onPageHide = () => sendBeaconProgress();
		document.addEventListener('visibilitychange', onVisibility);
		window.addEventListener('pagehide', onPageHide);
		return () => {
			if (activityTimer !== null) {
				clearInterval(activityTimer);
			}
			if (noticeTimer !== null) {
				clearTimeout(noticeTimer);
			}
			if (countdownTimer !== null) {
				clearInterval(countdownTimer);
			}
			if (clickTimer !== null) {
				clearTimeout(clickTimer);
			}
			document.removeEventListener('visibilitychange', onVisibility);
			window.removeEventListener('pagehide', onPageHide);
		};
	});

	function loadLocalPreferences() {
		try {
			const storedVolume = localStorage.getItem(volumeStorageKey);
			const storedMuted = localStorage.getItem(mutedStorageKey);
			const storedSpeed = localStorage.getItem(speedStorageKey);
			if (storedVolume !== null && Number.isFinite(Number(storedVolume))) {
				volume = Math.min(1, Math.max(0, Number(storedVolume)));
			}
			if (storedMuted !== null) {
				muted = storedMuted === '1';
			}
			const storedRate = Number(storedSpeed);
			if (storedSpeed !== null && PLAYER_SPEEDS.includes(storedRate as (typeof PLAYER_SPEEDS)[number])) {
				speed = storedRate;
			}
			if (video) {
				video.volume = volume;
				video.muted = muted;
				video.playbackRate = speed;
			}
		} catch {
			// localStorage puede estar deshabilitado; se usan los valores por defecto.
		}
	}

	function remember(key: string, value: string) {
		try {
			localStorage.setItem(key, value);
		} catch {
			// Sin persistencia local no pasa nada.
		}
	}

	async function setupHls() {
		const nativeHls = video.canPlayType('application/vnd.apple.mpegurl') !== '';
		let HlsPlayer: typeof Hls | null = null;
		try {
			HlsPlayer = (await import('hls.js')).default;
		} catch {
			HlsPlayer = null;
		}
		if (destroyed) {
			return;
		}

		const strategy = chooseHlsStrategy(nativeHls, HlsPlayer?.isSupported() ?? false);
		if (strategy === 'native') {
			video.src = src;
			return;
		}
		if (strategy === 'unsupported' || !HlsPlayer) {
			errorMessage = 'Este navegador no puede reproducir HLS.';
			return;
		}

		const instance = new HlsPlayer({
			fragLoadPolicy: {
				default: {
					maxTimeToFirstByteMs: 60_000,
					maxLoadTimeMs: 120_000,
					timeoutRetry: { maxNumRetry: 4, retryDelayMs: 0, maxRetryDelayMs: 0 },
					errorRetry: { maxNumRetry: 6, retryDelayMs: 1000, maxRetryDelayMs: 8000 }
				}
			}
		});
		hls = instance;
		let networkRecoveries = 0;
		let mediaRecoveries = 0;
		instance.on(HlsPlayer.Events.FRAG_LOADED, () => {
			networkRecoveries = 0;
		});
		instance.on(HlsPlayer.Events.ERROR, (_event, data) => {
			if (!data.fatal || !HlsPlayer) {
				return;
			}
			if (
				data.type === HlsPlayer.ErrorTypes.NETWORK_ERROR &&
				networkRecoveries < maxNetworkRecoveries
			) {
				networkRecoveries++;
				instance.startLoad();
				return;
			}
			if (data.type === HlsPlayer.ErrorTypes.MEDIA_ERROR && mediaRecoveries < maxMediaRecoveries) {
				mediaRecoveries++;
				instance.recoverMediaError();
				return;
			}
			errorMessage =
				'No se pudo reproducir el vídeo. Puede que el enlace haya caducado o el formato no sea compatible.';
		});
		instance.loadSource(src);
		instance.attachMedia(video);
	}

	onDestroy(() => {
		destroyed = true;
		hls?.destroy();
		hls = null;
		if (countdownTimer !== null) {
			clearInterval(countdownTimer);
		}
	});

	function snapshot(): ProgressSnapshot {
		return {
			position: video?.currentTime ?? 0,
			duration: video?.duration ?? Number.NaN
		};
	}

	function handleLoadedMetadata() {
		if (!video) {
			return;
		}
		duration = Number.isFinite(video.duration) ? video.duration : 0;
		if (!resumeApplied) {
			const target = resumeTarget(startAt, video.duration, resumeApplied);
			resumeApplied = true;
			if (target !== null) {
				try {
					video.currentTime = target;
					currentTime = target;
				} catch {
					// El navegador puede rechazar el salto si aún no puede buscar.
				}
			}
		}
		void tick().then(() => applySubtitle(selectedSubtitle, false));
	}

	function sendDecision(decision: ProgressDecision, now: number) {
		if (!progressEnabled(itemId) || !decision.send) {
			return;
		}
		lastProgressAt = now;
		const path = `/items/${itemId}/progress`;
		const body = JSON.stringify({
			position_seconds: decision.position,
			duration_seconds: decision.duration
		});
		void fetch(path, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body,
			keepalive: true
		}).catch(() => {
			// Un fallo de progreso no debe afectar a la reproducción.
		});
	}

	function saveSnapshot(event: 'timeupdate' | 'pause' | 'ended') {
		if (!progressEnabled(itemId) || !video) {
			return;
		}
		const now = Date.now();
		sendDecision(decideProgress(event, snapshot(), lastProgressAt, now), now);
	}

	function handleTimeUpdate() {
		if (!video) {
			return;
		}
		currentTime = video.currentTime;
		duration = Number.isFinite(video.duration) ? video.duration : duration;
		if (video.buffered.length > 0) {
			buffered = video.buffered.end(video.buffered.length - 1);
		}
		saveSnapshot('timeupdate');
	}

	function handlePause() {
		playing = false;
		controlsHidden = false;
		saveSnapshot('pause');
	}

	function handlePlay() {
		playing = true;
		buffering = false;
		activity();
		clearCountdown();
	}

	function handleEnded() {
		playing = false;
		controlsHidden = false;
		saveSnapshot('ended');
		if (nextEpisodeUrl) {
			startCountdown();
		}
	}

	function handleWaiting() {
		buffering = true;
	}
	function handlePlaying() {
		buffering = false;
	}

	function handleVolumeChange() {
		if (!video) {
			return;
		}
		volume = video.volume;
		muted = video.muted;
		remember(volumeStorageKey, String(volume));
		remember(mutedStorageKey, muted ? '1' : '0');
	}

	function handleRateChange() {
		if (video) {
			speed = video.playbackRate;
			remember(speedStorageKey, String(speed));
		}
	}

	function togglePlay() {
		if (!video) {
			return;
		}
		activity();
		if (video.paused) {
			void video.play().catch(() => {
				errorMessage = 'El navegador no permite reproducir el vídeo.';
			});
		} else {
			video.pause();
		}
	}

	function seek(value: number) {
		if (!video) {
			return;
		}
		const target = clampSeekTime(value, 0, duration || video.duration);
		video.currentTime = target;
		currentTime = target;
		activity();
	}

	function seekBy(delta: number) {
		seek((video?.currentTime ?? 0) + delta);
		showNotice(`${delta > 0 ? '+' : ''}${delta} s`);
	}

	function setVolume(value: number) {
		if (!video) {
			return;
		}
		video.volume = Math.min(1, Math.max(0, value));
		if (video.volume > 0) {
			video.muted = false;
		}
		volume = video.volume;
		muted = video.muted;
		remember(volumeStorageKey, String(volume));
	}

	function toggleMute() {
		if (!video) {
			return;
		}
		video.muted = !video.muted;
		muted = video.muted;
		remember(mutedStorageKey, muted ? '1' : '0');
		showNotice(muted ? 'Silencio' : `Volumen ${Math.round(volume * 100)} %`);
	}

	function setSpeed(value: number) {
		if (!video) {
			return;
		}
		video.playbackRate = value;
		speed = value;
		remember(speedStorageKey, String(value));
		showNotice(`Velocidad ${formatSpeed(value)}`);
	}

	function activity() {
		lastActivity = Date.now();
		controlsHidden = false;
	}

	function handleMenuChange(open: boolean) {
		menuOpen = open;
		if (open) {
			controlsHidden = false;
		}
		activity();
	}

	function handleFocusChange(focused: boolean) {
		controlFocused = focused;
		if (focused) {
			controlsHidden = false;
		}
	}

	function showNotice(message: string) {
		notice = message;
		if (noticeTimer !== null) {
			clearTimeout(noticeTimer);
		}
		noticeTimer = setTimeout(() => {
			notice = null;
			noticeTimer = null;
		}, 1400);
	}

	async function savePreference(body: Record<string, unknown>) {
		if (!itemId) {
			return;
		}
		try {
			await fetch(`/items/${itemId}/preferences`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
		} catch {
			// La preferencia es secundaria; la reproducción sigue.
		}
	}

	function chooseAudio(index: number) {
		if (!itemId || index === selectedAudio) {
			return;
		}
		selectedAudio = index;
		void savePreference({ audio_index: index }).then(() => {
			const params = new URLSearchParams();
			params.set('audio', String(index));
			params.set('t', String(Math.floor(video?.currentTime ?? currentTime)));
			window.location.href = `/items/${itemId}/play?${params}`;
		});
	}

	function chooseQuality(value: string) {
		if (!itemId || value === selectedQuality) {
			return;
		}
		selectedQuality = value;
		void savePreference({ quality: value }).then(() => {
			const params = new URLSearchParams();
			params.set('quality', value);
			params.set('t', String(Math.floor(video?.currentTime ?? currentTime)));
			window.location.href = `/items/${itemId}/play?${params}`;
		});
	}

	function applySubtitle(id: string | null, persist = true) {
		selectedSubtitle = id;
		if (!video) {
			return;
		}
		const tracks = video.querySelectorAll<HTMLTrackElement>('track[data-subtitle-id]');
		tracks.forEach((track) => {
			track.track.mode = id !== null && track.dataset.subtitleId === id ? 'showing' : 'disabled';
		});
		if (persist) {
			void savePreference({ subtitle_id: id ?? '' });
			showNotice(id === null ? 'Sin subtítulos' : `Subtítulos ${selectedSubtitleLabel || id}`);
		}
	}

	function chooseSubtitle(value: string) {
		applySubtitle(value === '' ? null : value);
	}

	function toggleSubtitles() {
		if (subtitles.length === 0) {
			return;
		}
		if (selectedSubtitle !== null) {
			chooseSubtitle('');
		} else {
			chooseSubtitle(subtitles[0].id);
		}
	}

	async function togglePictureInPicture() {
		if (!video || typeof document === 'undefined' || !document.pictureInPictureEnabled) {
			return;
		}
		try {
			if (document.pictureInPictureElement) {
				await document.exitPictureInPicture();
			} else {
				await video.requestPictureInPicture();
			}
		} catch {
			// El navegador puede rechazar PiP sin gesto de usuario.
		}
	}

	async function toggleFullscreen() {
		if (!container) {
			return;
		}
		const element = container as HTMLElement & {
			webkitRequestFullscreen?: () => Promise<void>;
		};
		const doc = document as Document & {
			webkitFullscreenElement?: Element | null;
			webkitExitFullscreen?: () => Promise<void>;
		};
		const current = document.fullscreenElement ?? doc.webkitFullscreenElement ?? null;
		try {
			if (current) {
				if (document.exitFullscreen) {
					await document.exitFullscreen();
				} else if (doc.webkitExitFullscreen) {
					await doc.webkitExitFullscreen();
				}
			} else if (element.requestFullscreen) {
				await element.requestFullscreen();
			} else if (element.webkitRequestFullscreen) {
				await element.webkitRequestFullscreen();
			} else {
				const webkitVideo = video as HTMLVideoElement & {
					webkitEnterFullscreen?: () => void;
				};
				webkitVideo.webkitEnterFullscreen?.();
			}
		} catch {
			// La pantalla completa puede no estar permitida.
		}
	}

	function startCountdown() {
		if (countdownTimer !== null) {
			clearInterval(countdownTimer);
		}
		countdown = 8;
		countdownTimer = setInterval(() => {
			if (countdown === null) {
				return;
			}
			countdown -= 1;
			if (countdown <= 0) {
				clearCountdown();
				if (nextEpisodeUrl) {
					window.location.href = nextEpisodeUrl;
				}
			}
		}, 1000);
	}

	function clearCountdown() {
		if (countdownTimer !== null) {
			clearInterval(countdownTimer);
			countdownTimer = null;
		}
		countdown = null;
	}

	function handleKeydown(event: KeyboardEvent) {
		const target = event.target as HTMLElement | null;
		if (
			target &&
			(target.tagName === 'INPUT' ||
				target.tagName === 'TEXTAREA' ||
				target.tagName === 'SELECT' ||
				target.tagName === 'BUTTON' ||
				target.tagName === 'A' ||
				target.isContentEditable)
		) {
			return;
		}
		const action = shortcutAction({
			key: event.key,
			ctrlKey: event.ctrlKey,
			metaKey: event.metaKey,
			altKey: event.altKey,
			shiftKey: event.shiftKey,
			paused: video?.paused ?? true,
			duration: duration || video?.duration || 0,
			currentTime: video?.currentTime ?? 0,
			volume
		});
		if (!action) {
			return;
		}
		event.preventDefault();
		activity();
		switch (action.type) {
			case 'toggle':
				togglePlay();
				break;
			case 'seek': {
				const previous = video?.currentTime ?? currentTime;
				seek(action.value);
				const delta = Math.round(action.value - previous);
				showNotice(`${delta > 0 ? '+' : ''}${delta} s`);
				break;
			}
			case 'volume':
				setVolume(action.value);
				showNotice(`Volumen ${Math.round(action.value * 100)} %`);
				break;
			case 'mute':
				toggleMute();
				break;
			case 'fullscreen':
				void toggleFullscreen();
				break;
			case 'subtitles':
				toggleSubtitles();
				break;
			case 'frame':
				seek((video?.currentTime ?? 0) + action.delta);
				showNotice(action.delta < 0 ? 'Fotograma anterior' : 'Fotograma siguiente');
				break;
			case 'percent':
				seek(action.value);
				showNotice(`${Math.round((action.value / Math.max(duration, 1)) * 100)} %`);
				break;
			case 'escape':
				clearCountdown();
				break;
		}
	}

	function handleVideoClick(event: MouseEvent) {
		if (event.detail === 2) {
			if (clickTimer !== null) {
				clearTimeout(clickTimer);
				clickTimer = null;
			}
			void toggleFullscreen();
			return;
		}
		if (clickTimer !== null) {
			clearTimeout(clickTimer);
		}
		clickTimer = setTimeout(() => {
			clickTimer = null;
			togglePlay();
		}, 220);
	}

	function sendBeaconProgress() {
		if (!progressEnabled(itemId) || !video || beaconSent) {
			return;
		}
		const now = Date.now();
		const decision = decideProgress('pagehide', snapshot(), lastProgressAt, now);
		if (!decision.send) {
			return;
		}
		beaconSent = true;
		lastProgressAt = now;
		const path = `/items/${itemId}/progress`;
		const body = JSON.stringify({
			position_seconds: decision.position,
			duration_seconds: decision.duration
		});
		try {
			if (typeof navigator.sendBeacon === 'function') {
				const blob = new Blob([body], { type: 'application/json' });
				if (navigator.sendBeacon(path, blob)) {
					return;
				}
			}
		} catch {
			// Si sendBeacon falla, se intenta el fetch con keepalive.
		}
		void fetch(path, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body,
			keepalive: true
		}).catch(() => {
			// La página se está cerrando; no hay nada más que hacer.
		});
	}

	function handleContainerMove() {
		activity();
	}

	// Los subtítulos del servidor son rutas relativas a la API; el <video> usa
	// la misma base que la URL firmada del stream.
	const subtitleTracks = $derived(
		subtitles.map((subtitle) => ({
			...subtitle,
			absoluteURL: absoluteSubtitleURL(src, subtitle.url)
		}))
	);
</script>

<svelte:window onkeydown={handleKeydown} />

<div
	bind:this={container}
	class="player"
	class:cursor-hidden={controlsHidden}
	role="region"
	aria-label={`Reproductor de ${title}`}
	onmousemove={handleContainerMove}
	ontouchstart={handleContainerMove}
>
	{#if errorMessage}
		<p class="player-error" role="alert">{errorMessage}</p>
	{/if}

	{#if buffering}
		<div class="buffering" role="status" aria-label="Cargando vídeo"><span></span></div>
	{/if}

	{#if notice}
		<p class="shortcut-notice" role="status">{notice}</p>
	{/if}

	{#if countdown !== null && nextEpisodeUrl}
		<div class="up-next" role="dialog" aria-label="Siguiente episodio">
			<p>{countdownLabel(countdown)}</p>
			<div>
				<button type="button" class="button button-secondary" onclick={clearCountdown}>Cancelar</button>
				<button
					type="button"
					class="button"
					onclick={() => {
						if (nextEpisodeUrl) {
							window.location.href = nextEpisodeUrl;
						}
					}}>Reproducir ya</button
				>
			</div>
		</div>
	{/if}

	<!-- svelte-ignore a11y_media_has_caption -->
	<video
		bind:this={video}
		class="player-video"
		controls={nativeControls}
		playsinline
		preload="metadata"
		src={mode === 'direct' ? src : undefined}
		aria-label={title}
		onclick={handleVideoClick}
		onloadedmetadata={handleLoadedMetadata}
		ontimeupdate={handleTimeUpdate}
		onpause={handlePause}
		onplay={handlePlay}
		onended={handleEnded}
		onwaiting={handleWaiting}
		onplaying={handlePlaying}
		onvolumechange={handleVolumeChange}
		onratechange={handleRateChange}
	>
		{#each subtitleTracks as subtitle (subtitle.id)}
			<track
				kind="subtitles"
				srclang={subtitle.language || 'und'}
				label={subtitle.title || subtitle.language_name || 'Subtítulos'}
				src={subtitle.absoluteURL}
				data-subtitle-id={subtitle.id}
			/>
		{/each}
	</video>

	<PlayerControls
		{playing}
		{currentTime}
		{duration}
		{buffered}
		{volume}
		{muted}
		{speed}
		hidden={controlsHidden}
		{audioTracks}
		{selectedAudio}
		{qualities}
		{selectedQuality}
		subtitles={subtitleTracks}
		{selectedSubtitle}
		{nextEpisodeUrl}
		{canPictureInPicture}
		onToggle={togglePlay}
		onSeek={seek}
		onSeekBy={seekBy}
		onVolume={setVolume}
		onMute={toggleMute}
		onSpeed={setSpeed}
		onAudio={chooseAudio}
		onSubtitle={chooseSubtitle}
		onQuality={chooseQuality}
		onPictureInPicture={() => void togglePictureInPicture()}
		onFullscreen={() => void toggleFullscreen()}
		onActivity={activity}
		onMenuChange={handleMenuChange}
		onFocusChange={handleFocusChange}
	/>
</div>

<style>
	.player {
		position: relative;
		width: 100%;
		max-width: 100%;
		overflow: hidden;
		background: #080505;
		border-radius: 8px;
		aspect-ratio: 16 / 9;
	}
	.player.cursor-hidden {
		cursor: none;
	}
	.player-video {
		display: block;
		width: 100%;
		height: 100%;
		max-height: none;
		object-fit: contain;
		background: #080505;
	}
	.player-error {
		position: absolute;
		inset: 1rem 1rem auto;
		z-index: 5;
		margin: 0;
		padding: 0.75rem 1rem;
		border-radius: 6px;
		background: rgba(123, 30, 36, 0.94);
		color: #fff4e1;
	}
	.buffering {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		pointer-events: none;
		z-index: 2;
	}
	.buffering span {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		border: 3px solid rgba(246, 234, 215, 0.25);
		border-top-color: #d9a441;
		animation: spin 0.8s linear infinite;
	}
	.shortcut-notice {
		position: absolute;
		left: 50%;
		top: 12%;
		transform: translateX(-50%);
		margin: 0;
		padding: 0.35rem 0.8rem;
		border-radius: 999px;
		background: rgba(12, 7, 6, 0.88);
		color: #f6ead7;
		font-size: 0.9rem;
		z-index: 4;
	}
	.up-next {
		position: absolute;
		right: 1rem;
		top: 1rem;
		z-index: 5;
		padding: 0.8rem;
		border-radius: 8px;
		background: rgba(12, 7, 6, 0.94);
		border: 1px solid rgba(246, 234, 215, 0.3);
		color: #f6ead7;
	}
	.up-next p {
		margin: 0 0 0.6rem;
	}
	.up-next div {
		display: flex;
		gap: 0.4rem;
	}
	:global(video::cue) {
		color: #fff4e1;
		background: transparent;
		text-shadow:
			0 1px 2px #000,
			0 0 4px #000,
			0 0 8px #000;
		font-size: 1.05rem;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
