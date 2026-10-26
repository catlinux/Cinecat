<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import { rowScrollState, type RowScrollState } from '$lib/rows';

	let {
		title,
		kicker = '',
		href = null,
		hrefLabel = 'Ver todas',
		children
	}: {
		title: string;
		kicker?: string;
		href?: string | null;
		hrefLabel?: string;
		children: Snippet;
	} = $props();

	let viewport: HTMLElement | null = $state(null);
	let scroll: RowScrollState = $state({
		canPrev: false,
		canNext: false,
		prevOffset: 0,
		nextOffset: 0,
		overflow: false
	});
	let frame: number | null = null;
	let observer: ResizeObserver | null = null;

	function measure() {
		if (!viewport) {
			return;
		}
		scroll = rowScrollState(viewport.scrollLeft, viewport.clientWidth, viewport.scrollWidth);
	}

	function scheduleMeasure() {
		if (frame !== null) {
			return;
		}
		frame = requestAnimationFrame(() => {
			frame = null;
			measure();
		});
	}

	function move(offset: number) {
		viewport?.scrollBy({ left: offset - (viewport?.scrollLeft ?? 0), behavior: 'smooth' });
	}

	onMount(() => {
		measure();
		observer = new ResizeObserver(scheduleMeasure);
		if (viewport) {
			observer.observe(viewport);
			for (const child of Array.from(viewport.children) as Element[]) {
				observer.observe(child);
			}
		}
		window.addEventListener('resize', scheduleMeasure);
		return () => {
			observer?.disconnect();
			window.removeEventListener('resize', scheduleMeasure);
			if (frame !== null) {
				cancelAnimationFrame(frame);
			}
		};
	});
</script>

<section class="row-scroller">
	<div class="row-head">
		<div class="row-title">
			{#if kicker}<p class="kicker">{kicker}</p>{/if}
			<h2>{title}</h2>
		</div>
		<div class="row-actions">
			{#if href}
				<a class="text-link row-link" href={href}>{hrefLabel} →</a>
			{/if}
			<!-- Sin JavaScript las flechas quedan ocultas; el contenedor sigue siendo desplazable. -->
			<div class="row-arrows" hidden={!scroll.overflow}>
				<button
					type="button"
					class="row-arrow"
					aria-label="Anterior"
					disabled={!scroll.canPrev}
					onclick={() => move(scroll.prevOffset)}
				>‹</button>
				<button
					type="button"
					class="row-arrow"
					aria-label="Siguiente"
					disabled={!scroll.canNext}
					onclick={() => move(scroll.nextOffset)}
				>›</button>
			</div>
		</div>
	</div>
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div
		bind:this={viewport}
		class="row-viewport"
		role="region"
		aria-label={title}
		tabindex="0"
		onscroll={scheduleMeasure}
	>
		<ul class="row-track">
			{@render children()}
		</ul>
	</div>
</section>

<style>
	.row-scroller {
		min-width: 0;
		max-width: 100%;
		margin: 1.5rem 0;
	}
	.row-head {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.35rem 0.75rem;
		margin-bottom: 0.55rem;
	}
	.row-title {
		min-width: 0;
	}
	.row-head h2 {
		margin: 0;
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.25rem, 2vw, 1.7rem);
	}
	.row-actions {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		flex-shrink: 0;
		margin-left: auto;
	}
	.row-link {
		white-space: nowrap;
		font-size: 0.8rem;
	}
	.row-arrows {
		display: flex;
		gap: 0.2rem;
	}
	.row-arrows[hidden] {
		display: none;
	}
	.row-arrow {
		width: 40px;
		height: 40px;
		border: 1px solid var(--borde-medio);
		border-radius: 50%;
		background: var(--superficie-suave);
		color: var(--granate);
		font: inherit;
		font-size: 1.25rem;
		line-height: 1;
		cursor: pointer;
	}
	.row-arrow:hover:not(:disabled),
	.row-arrow:focus-visible:not(:disabled) {
		background: var(--tinte);
	}
	.row-arrow:disabled {
		opacity: 0.35;
		cursor: default;
	}
	.row-viewport {
		width: 100%;
		max-width: 100%;
		overflow-x: auto;
		overflow-y: hidden;
		scroll-snap-type: x mandatory;
		scrollbar-width: none;
		-ms-overflow-style: none;
		padding-bottom: 0.25rem;
	}
	.row-viewport::-webkit-scrollbar {
		display: none;
	}
	.row-track {
		display: flex;
		gap: 0.75rem;
		align-items: stretch;
		margin: 0;
		padding: 0;
		list-style: none;
		width: max-content;
	}
	.row-track > :global(li) {
		flex: 0 0 auto;
		scroll-snap-align: start;
	}
</style>
