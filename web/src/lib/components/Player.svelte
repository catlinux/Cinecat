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

	onMount(() => {
		if (mode !== 'remux' || !video) {
			return;
		}

		const nativeHls = video.canPlayType('application/vnd.apple.mpegurl') !== '';
		if (chooseHlsStrategy(nativeHls, false) === 'native') {
			video.src = src;
			return;
		}

		void setupHlsJs();
	});

	async function setupHlsJs() {
		try {
			const { default: HlsPlayer } = await import('hls.js');
			if (destroyed) {
				return;
			}

			if (chooseHlsStrategy(false, HlsPlayer.isSupported()) !== 'hlsjs') {
				errorMessage = 'Este navegador no puede reproducir HLS.';
				return;
			}

			const instance = new HlsPlayer();
			hls = instance;
			instance.on(HlsPlayer.Events.ERROR, (_event, data) => {
				if (data.fatal) {
					errorMessage =
						'No se pudo reproducir el vídeo. Puede que el enlace haya caducado o el formato no sea compatible.';
				}
			});
			instance.loadSource(src);
			instance.attachMedia(video);
		} catch {
			errorMessage = 'No se pudo cargar el reproductor HLS.';
		}
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
