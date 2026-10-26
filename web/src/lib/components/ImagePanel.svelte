<script lang="ts">
	import { enhance } from '$app/forms';
	import type { ImageKind, ImageOptions, MediaItem } from '$lib/api';
	import { formatLanguage, formatRating, formatResolution } from '$lib/media';

	let {
		item,
		open,
		imageOptions,
		imageListKind
	}: {
		item: MediaItem;
		open: boolean;
		imageOptions: ImageOptions | null;
		imageListKind: ImageKind | null;
	} = $props();
</script>

{#snippet imageSection(
	kind: ImageKind,
	label: string,
	current: string | null,
	custom: boolean
)}
	{@const options =
		imageListKind === kind && imageOptions
			? kind === 'poster'
				? imageOptions.posters
				: imageOptions.backdrops
			: []}

	<section class="image-section">
		<h3>{label}</h3>

		<div class="image-current" class:image-current--backdrop={kind === 'backdrop'}>
			{#if current}
				<img src={current} alt={`${label} actual de ${item.title}`} loading="lazy" />
			{:else}
				<span class="image-current-empty">Sin imagen</span>
			{/if}
			<div class="image-current-info">
				{#if custom}
					<span class="file-tag">personalizada</span>
					<form method="POST" action="?/imageReset" use:enhance>
						<input type="hidden" name="kind" value={kind} />
						<button class="button-danger" type="submit">Restaurar la de TMDB</button>
					</form>
				{/if}
			</div>
		</div>

		<form class="image-url-form" method="POST" action="?/imageSet" use:enhance>
			<input type="hidden" name="kind" value={kind} />
			<label for={`${kind}-url`}>Usar esta URL</label>
			<input
				id={`${kind}-url`}
				name="url"
				type="url"
				inputmode="url"
				placeholder="https://…"
				pattern="https?://.+"
				title="Debe empezar por http:// o https://"
				required
			/>
			<button class="button" type="submit">Usar esta URL</button>
		</form>

		<form method="POST" action="?/imageList" use:enhance>
			<input type="hidden" name="kind" value={kind} />
			<button class="button button-secondary" type="submit">Ver las que ofrece TMDB</button>
		</form>

		{#if imageListKind === kind}
			{#if options.length > 0}
				<ul class="image-grid" class:image-grid--backdrop={kind === 'backdrop'}>
					{#each options as option, index (index)}
						<li class="image-option">
							<img
								src={option.url}
								alt={`${label} de TMDB en ${formatLanguage(option.language) || 'sin idioma'}`}
								loading="lazy"
							/>
							<p class="image-option-language">
								{formatLanguage(option.language) || 'Sin idioma'}
							</p>
							<p class="image-option-meta">
								{formatResolution(option.width, option.height, '')} · {formatRating(option.vote_average)}
							</p>
							<form method="POST" action="?/imageSet" use:enhance>
								<input type="hidden" name="kind" value={kind} />
								<input type="hidden" name="url" value={option.full_url} />
								<button class="button" type="submit">Usar esta</button>
							</form>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Sin imágenes disponibles.</p>
			{/if}
		{/if}
	</section>
{/snippet}

<details class="admin-card image-panel" id="imagenes" open={open}>
	<summary class="image-panel-summary">Cambiar imagen</summary>

	<div class="image-panel-body">
		{@render imageSection('poster', 'Cartel', item.poster_url, item.poster_custom)}
		{@render imageSection('backdrop', 'Fondo', item.backdrop_url, item.backdrop_custom)}
	</div>
</details>
