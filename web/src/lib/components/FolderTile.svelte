<script lang="ts">
	import { videoCountLabel } from '$lib/videos';

	let {
		name,
		itemCount,
		href
	}: {
		name: string;
		itemCount: number;
		href: string;
	} = $props();
</script>

<a class="folder-tile" {href}>
	<span class="folder-tile-art" aria-hidden="true">
		<svg viewBox="0 0 160 110" role="presentation" focusable="false">
			<path
				d="M14 31h47l12 14h73v47a8 8 0 0 1-8 8H22a8 8 0 0 1-8-8z"
				fill="var(--granate-oscuro)"
				opacity="0.82"
			/>
			<path d="M14 45h132v47a8 8 0 0 1-8 8H22a8 8 0 0 1-8-8z" fill="var(--granate)" opacity="0.24" />
			<path d="M28 45h104" stroke="var(--dorado)" stroke-width="2" opacity="0.72" />
			<path
				d="M68 63l26 15-26 15z"
				fill="var(--dorado)"
				opacity="0.92"
			/>
			<path d="M118 33h22M129 22v22" stroke="var(--dorado)" stroke-width="1.5" opacity="0.45" />
		</svg>
	</span>
	<span class="folder-tile-copy">
		<strong>{name}</strong>
		<span>{videoCountLabel(itemCount)}</span>
	</span>
</a>

<style>
	.folder-tile {
		display: grid;
		grid-template-columns: 5rem minmax(0, 1fr);
		align-items: center;
		gap: 0.9rem;
		min-width: 0;
		padding: 0.85rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
		box-shadow: 0 0.8rem 1.5rem -1.3rem var(--sombra);
		color: inherit;
		text-decoration: none;
		transition:
			border-color 160ms ease,
			transform 160ms ease;
	}
	.folder-tile:hover,
	.folder-tile:focus-visible {
		border-color: var(--naranja);
		transform: translateY(-2px);
	}
	.folder-tile-art {
		display: block;
		overflow: hidden;
		aspect-ratio: 16 / 11;
		border-radius: 3px;
		background:
			linear-gradient(145deg, var(--granate-oscuro), var(--granate)),
			repeating-linear-gradient(90deg, rgb(255 255 255 / 5%) 0 1px, transparent 1px 12px);
	}
	.folder-tile-art svg {
		display: block;
		width: 100%;
		height: 100%;
	}
	.folder-tile-copy {
		display: grid;
		gap: 0.2rem;
		min-width: 0;
	}
	.folder-tile-copy strong {
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.05rem;
		line-height: 1.15;
		color: var(--granate);
	}
	.folder-tile-copy span {
		color: var(--texto-apagado);
		font-size: 0.74rem;
	}
	@media (max-width: 520px) {
		.folder-tile {
			grid-template-columns: 4rem minmax(0, 1fr);
		}
	}
</style>
