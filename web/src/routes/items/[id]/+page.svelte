<script lang="ts">
	import { enhance } from '$app/forms';
	import type {
		Candidate,
		FileInfoResult,
		ImageKind,
		ImageOptions
	} from '$lib/api';
	import ItemFacts from '$lib/components/ItemFacts.svelte';
	import FilePanel from '$lib/components/FilePanel.svelte';
	import ImagePanel from '$lib/components/ImagePanel.svelte';
	import MatchDialog from '$lib/components/MatchDialog.svelte';
	import {
		formatMovieMeta,
		itemSubtitle,
		kindLabel,
		matchStatusLabel,
		overviewInSpanish
	} from '$lib/media';
	import { posterColors } from '$lib/poster-color';
	import { formatProgressTime, progressPercent } from '$lib/progress';
	import type { PageData } from './$types';

	interface MatchSearch {
		q: string;
		year: number | null;
	}

	interface ActionForm {
		success?: boolean;
		panel?: boolean;
		imagePanel?: boolean;
		imageKind?: ImageKind;
		imageOptions?: ImageOptions;
		message?: string;
		candidates?: Candidate[];
		search?: MatchSearch;
	}

	let { data, form }: { data: PageData; form?: ActionForm | null } = $props();

	const item = $derived(data.item);
	const progress = $derived(item.progress);
	const hasProgress = $derived(progress !== null && progress.position_seconds > 0);
	const progressLabel = $derived(
		progress ? formatProgressTime(progress.position_seconds) : ''
	);
	const progressValue = $derived(
		progress ? progressPercent(progress.position_seconds, progress.duration_seconds) : 0
	);
	const isMovie = $derived(item.kind === 'movie');
	const colors = $derived(posterColors(item.title));
	const meta = $derived(isMovie ? formatMovieMeta(item) : itemSubtitle(item));
	const originalTitle = $derived(
		isMovie &&
			item.original_title &&
			item.original_title.trim() !== '' &&
			item.original_title !== item.title
			? item.original_title
			: ''
	);
	const showStatus = $derived(
		isMovie && (item.match_status === null || item.match_status === 'none')
	);
	const showLanguageWarning = $derived(
		isMovie && Boolean(item.overview) && !overviewInSpanish(item.overview_language)
	);
	const actionForm = $derived((form ?? null) as ActionForm | null);
	const candidates = $derived(actionForm?.candidates ?? []);
	const search = $derived(actionForm?.search ?? null);
	const panelOpen = $derived(Boolean(data.corregir || actionForm?.panel));
	const imageOptions = $derived(actionForm?.imageOptions ?? null);
	const imageListKind = $derived(actionForm?.imageKind ?? null);
	const imagePanelOpen = $derived(Boolean(data.imagenes || actionForm?.imagePanel));
	const fileInfo = $derived(
		(data.fileInfo ?? { error: 'failed' }) as FileInfoResult | Promise<FileInfoResult>
	);
</script>

<svelte:head>
	<title>{item.title} · Cinecat</title>
</svelte:head>

<a class="back" href="/movies">← Volver a la cartelera</a>

<section class="detail" class:detail-movie={isMovie}>
	{#if isMovie && item.backdrop_url}
		<div class="detail-backdrop" aria-hidden="true">
			<img src={item.backdrop_url} alt="" loading="lazy" />
		</div>
	{/if}

	<div
		class="detail-poster"
		class:detail-poster--image={Boolean(item.poster_url)}
		style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
		aria-hidden="true"
	>
		<span class="poster-kind">{kindLabel(item.kind)}</span>
		{#if item.poster_url}
			<img
				class="detail-poster-img"
				src={item.poster_url}
				alt={`Cartel de ${item.title}`}
				loading="lazy"
			/>
		{:else}
			<span class="detail-poster-title">{item.title}</span>
		{/if}
	</div>

	<div class="detail-info">
		<p class="kicker">{kindLabel(item.kind)}</p>
		<h1>{item.title}</h1>

		{#if originalTitle}
			<p class="detail-original-title"><em>{originalTitle}</em></p>
		{/if}

		{#if meta}
			<p class="detail-meta">{meta}</p>
		{/if}

		{#if item.kind === 'episode' && item.season !== null && item.episode !== null}
			<p class="detail-extra">Temporada {item.season}, episodio {item.episode}</p>
		{/if}

		{#if showStatus}
			<p class="detail-status">{matchStatusLabel(item)}</p>
		{/if}

		{#if isMovie && item.tagline}
			<blockquote class="detail-tagline">
				<p>«{item.tagline}»</p>
			</blockquote>
		{/if}

		{#if item.overview}
			<p class="detail-overview">{item.overview}</p>
		{/if}

		{#if showLanguageWarning}
			<p class="detail-note">Sinopsis no disponible en castellano.</p>
		{/if}

		<div class="detail-actions">
			{#if hasProgress}
				<a class="button" href={`/items/${item.id}/play`}>Continuar · {progressLabel}</a>
				<a class="button button-secondary" href={`/items/${item.id}/play?t=0`}>
					Desde el principio
				</a>
			{:else}
				<a class="button" href={`/items/${item.id}/play`}>Reproducir</a>
			{/if}
			<form method="POST" action="?/watched" use:enhance>
				<input type="hidden" name="watched" value={item.watched ? '0' : '1'} />
				<button class="button button-secondary" type="submit" aria-pressed={item.watched}>
					{item.watched ? 'Marcada como vista · Desmarcar' : 'Marcar como visto'}
				</button>
			</form>
		</div>

		{#if hasProgress}
			<div
				class="detail-progress"
				role="progressbar"
				aria-label={`Progreso de ${item.title}`}
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={progressValue}
			>
				<span style={`width: ${progressValue}%`}></span>
			</div>
		{/if}

		{#if actionForm?.message}
			<p
				class={actionForm.success ? 'success' : 'error'}
				role={actionForm.success ? 'status' : 'alert'}
			>
				{actionForm.message}
			</p>
		{/if}

		{#if isMovie}
			<ItemFacts {item} />
		{/if}

		{#if item.collections.length > 0}
			<section class="item-collections">
				<p class="kicker">Forma parte de</p>
				<ul>
					{#each item.collections as collection (collection.id)}
						<li><a href={`/collections/${collection.id}`}>{collection.name}</a></li>
					{/each}
				</ul>
			</section>
		{/if}

		<FilePanel result={fileInfo} />

		{#if isMovie && data.isAdmin}
			<ImagePanel
				{item}
				open={imagePanelOpen}
				{imageOptions}
				{imageListKind}
			/>
			<MatchDialog {item} {candidates} {search} open={panelOpen} />
		{/if}
	</div>
</section>
