<script lang="ts">
	import { onMount } from 'svelte';
	import type { PhotoItem } from '$lib/api';
	import {
		nextPhotoIndex,
		photoDateLabel,
		photoOriginalURL,
		previousPhotoIndex
	} from '$lib/photos';

	let {
		photos,
		startIndex,
		onclose
	}: {
		photos: PhotoItem[];
		startIndex: number;
		onclose: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	let index = $state(Math.min(Math.max(0, startIndex), Math.max(0, photos.length - 1)));
	let playing = $state(false);
	let closeButton = $state<HTMLButtonElement | null>(null);
	let timer: ReturnType<typeof setInterval> | null = null;

	const current = $derived(photos[index] ?? null);
	const counter = $derived(photos.length > 0 ? `${index + 1} de ${photos.length}` : '');
	const originalHref = $derived(current ? photoOriginalURL(current.id) : '');

	function stopSlideshow() {
		playing = false;
		if (timer !== null) {
			clearInterval(timer);
			timer = null;
		}
	}

	function startSlideshow() {
		stopSlideshow();
		if (photos.length < 2) {
			return;
		}
		playing = true;
		timer = setInterval(() => {
			index = nextPhotoIndex(index, photos.length);
		}, 5000);
	}

	function toggleSlideshow() {
		if (playing) {
			stopSlideshow();
		} else {
			startSlideshow();
		}
	}

	function previous() {
		stopSlideshow();
		index = previousPhotoIndex(index, photos.length);
	}

	function next() {
		stopSlideshow();
		index = nextPhotoIndex(index, photos.length);
	}

	function close() {
		stopSlideshow();
		onclose();
	}

	function handleKeydown(event: KeyboardEvent) {
		stopSlideshow();
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
			return;
		}
		if (event.key === 'ArrowLeft') {
			event.preventDefault();
			index = previousPhotoIndex(index, photos.length);
			return;
		}
		if (event.key === 'ArrowRight') {
			event.preventDefault();
			index = nextPhotoIndex(index, photos.length);
		}
	}

	onMount(() => {
		closeButton?.focus();
		const previousOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			stopSlideshow();
			document.body.style.overflow = previousOverflow;
		};
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="photo-viewer" role="dialog" aria-modal="true" aria-label="Visor de fotos">
	<div class="photo-viewer-top">
		<p class="photo-viewer-counter" aria-live="polite">{counter}</p>
		<div class="photo-viewer-actions">
			<button
				type="button"
				class="photo-viewer-button"
				bind:this={closeButton}
				onclick={close}
			>
				Cerrar
			</button>
		</div>
	</div>

	{#if current}
		<div class="photo-viewer-stage">
			<img src={photoOriginalURL(current.id)} alt={current.name} />
		</div>
		<div class="photo-viewer-caption">
			<strong>{current.name}</strong>
			{#if photoDateLabel(current.taken_at)}
				<span>{photoDateLabel(current.taken_at)}</span>
			{/if}
		</div>
	{:else}
		<p class="photo-viewer-empty">No hay fotos que mostrar.</p>
	{/if}

	<div class="photo-viewer-controls">
		<button
			type="button"
			class="photo-viewer-button"
			onclick={previous}
			disabled={photos.length < 2}
			aria-label="Foto anterior"
		>
			← Anterior
		</button>
		<button
			type="button"
			class="photo-viewer-button photo-viewer-play"
			onclick={toggleSlideshow}
			disabled={photos.length < 2}
			aria-pressed={playing}
		>
			{playing ? 'Detener presentación' : 'Presentación automática'}
		</button>
		<button
			type="button"
			class="photo-viewer-button"
			onclick={next}
			disabled={photos.length < 2}
			aria-label="Foto siguiente"
		>
			Siguiente →
		</button>
	</div>

	{#if current}
		<a
			class="photo-viewer-original"
			href={originalHref}
			target="_blank"
			rel="noopener noreferrer"
			onclick={stopSlideshow}
		>
			Abrir original en otra pestaña
		</a>
	{/if}
</div>

<style>
	.photo-viewer {
		position: fixed;
		z-index: 1000;
		inset: 0;
		display: flex;
		flex-direction: column;
		gap: 0.8rem;
		padding: clamp(0.8rem, 3vw, 2rem);
		background: #140a0b;
		color: var(--texto-sobre-oscuro);
	}
	.photo-viewer-top,
	.photo-viewer-controls {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.6rem;
	}
	.photo-viewer-actions {
		display: flex;
		gap: 0.5rem;
	}
	.photo-viewer-counter,
	.photo-viewer-caption {
		margin: 0;
		font-size: 0.78rem;
		letter-spacing: 0.08em;
	}
	.photo-viewer-stage {
		display: grid;
		flex: 1;
		min-height: 0;
		place-items: center;
	}
	.photo-viewer-stage img {
		display: block;
		max-width: 100%;
		max-height: 100%;
		object-fit: contain;
	}
	.photo-viewer-caption {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.5rem 1rem;
		text-align: center;
	}
	.photo-viewer-caption strong {
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1rem, 2vw, 1.4rem);
		font-weight: 600;
	}
	.photo-viewer-caption span {
		color: var(--texto-sobre-oscuro);
		opacity: 0.75;
	}
	.photo-viewer-controls {
		justify-content: center;
	}
	.photo-viewer-button,
	.photo-viewer-original {
		padding: 0.55rem 0.8rem;
		border: 1px solid var(--texto-sobre-oscuro);
		border-radius: 3px;
		background: transparent;
		color: var(--texto-sobre-oscuro);
		font: inherit;
		font-size: 0.76rem;
		font-weight: 600;
		text-decoration: none;
		cursor: pointer;
	}
	.photo-viewer-button:hover:not(:disabled),
	.photo-viewer-button:focus-visible,
	.photo-viewer-original:hover,
	.photo-viewer-original:focus-visible {
		border-color: var(--dorado);
		color: var(--dorado);
		outline: 2px solid var(--foco);
		outline-offset: 2px;
	}
	.photo-viewer-button:disabled {
		cursor: default;
		opacity: 0.45;
	}
	.photo-viewer-play[aria-pressed='true'] {
		border-color: var(--dorado);
		color: var(--dorado);
	}
	.photo-viewer-original {
		align-self: center;
	}
	.photo-viewer-empty {
		margin: auto;
	}
	@media (max-width: 620px) {
		.photo-viewer-controls {
			justify-content: stretch;
		}
		.photo-viewer-controls .photo-viewer-button {
			flex: 1 1 8rem;
		}
	}
</style>
