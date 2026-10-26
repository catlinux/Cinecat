<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import type Hls from 'hls.js';
	import { chooseHlsStrategy } from '$lib/hls-strategy';

	let {
		src,
		mode,
		title = 'Vídeo'
	}: {
		src: string;
		mode: 'direct' | 'remux';
		title?: string;
	} = $props();

	let video: HTMLVideoElement;
	let errorMessage = $state<string | null>(null);
	let hls: Hls | null = null;
	let destroyed = false;

	// Reintentos ante errores fatales antes de rendirse: un salto puede tardar
	// unos segundos mientras el servidor relanza FFmpeg en el punto pedido.
	const maxNetworkRecoveries = 5;
	const maxMediaRecoveries = 2;

	onMount(() => {
		if (mode !== 'remux' || !video) {
			return;
		}
		void setupHls();
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
	></video>
</div>
