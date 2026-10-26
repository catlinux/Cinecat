<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import type Hls from 'hls.js';
	import { chooseHlsStrategy } from '$lib/hls-strategy';
	import {
		decideProgress,
		progressEnabled,
		resumeTarget,
		type ProgressDecision,
		type ProgressSnapshot
	} from '$lib/progress';

	let {
		src,
		mode,
		title = 'Vídeo',
		itemId = undefined,
		startAt = 0
	}: {
		src: string;
		mode: 'direct' | 'remux';
		title?: string;
		itemId?: number;
		startAt?: number;
	} = $props();

	let video: HTMLVideoElement;
	let errorMessage = $state<string | null>(null);
	let hls: Hls | null = null;
	let destroyed = false;

	let resumeApplied = false;
	let lastProgressAt: number | null = null;
	let beaconSent = false;

	// Reintentos ante errores fatales antes de rendirse: un salto puede tardar
	// unos segundos mientras el servidor relanza FFmpeg en el punto pedido.
	const maxNetworkRecoveries = 5;
	const maxMediaRecoveries = 2;

	onMount(() => {
		if (mode === 'remux' && video) {
			void setupHls();
		}
		if (!progressEnabled(itemId)) {
			return;
		}

		const onVisibility = () => {
			if (document.visibilityState === 'hidden') {
				sendBeaconProgress();
			} else {
				beaconSent = false;
			}
		};
		const onPageHide = () => {
			sendBeaconProgress();
		};

		document.addEventListener('visibilitychange', onVisibility);
		window.addEventListener('pagehide', onPageHide);
		return () => {
			document.removeEventListener('visibilitychange', onVisibility);
			window.removeEventListener('pagehide', onPageHide);
		};
	});

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

		// El servidor no responde a un segmento hasta tenerlo generado; tras un
		// salto lejano eso puede pasar de los 10 s que espera hls.js por defecto.
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
	});

	function snapshot(): ProgressSnapshot {
		return {
			position: video?.currentTime ?? 0,
			duration: video?.duration ?? Number.NaN
		};
	}

	function handleLoadedMetadata() {
		if (!video || resumeApplied) {
			return;
		}
		const target = resumeTarget(startAt, video.duration, resumeApplied);
		resumeApplied = true;
		if (target !== null) {
			try {
				video.currentTime = target;
			} catch {
				// El navegador puede rechazar el salto si aún no puede buscar.
			}
		}
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
		saveSnapshot('timeupdate');
	}

	function handlePause() {
		saveSnapshot('pause');
	}

	function handleEnded() {
		saveSnapshot('ended');
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
</script>

<div class="player">
	{#if errorMessage}
		<p class="player-error" role="alert">{errorMessage}</p>
	{/if}

	<!-- svelte-ignore a11y_media_has_caption (los subtítulos llegan en la Fase 3) -->
	<video
		bind:this={video}
		class="player-video"
		controls
		playsinline
		preload="metadata"
		src={mode === 'direct' ? src : undefined}
		aria-label={title}
		onloadedmetadata={handleLoadedMetadata}
		ontimeupdate={handleTimeUpdate}
		onpause={handlePause}
		onended={handleEnded}
	></video>
</div>
