<script lang="ts">
	import { enhance } from '$app/forms';
	import type {
		Candidate,
		FileInfoResult,
		ImageKind,
		ImageOptions,
		MediaItem,
		PlayResponse
	} from '$lib/api';
	import CastRow from '$lib/components/CastRow.svelte';
	import EpisodeNav from '$lib/components/EpisodeNav.svelte';
	import ItemFacts from '$lib/components/ItemFacts.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';
	import RowScroller from '$lib/components/RowScroller.svelte';
	import FilePanel from '$lib/components/FilePanel.svelte';
	import ImagePanel from '$lib/components/ImagePanel.svelte';
	import MatchDialog from '$lib/components/MatchDialog.svelte';
	import PlaybackOptions from '$lib/components/PlaybackOptions.svelte';
	import {
		episodeImageURL,
		formatDurationSeconds,
		formatMovieMeta,
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
	const isClip = $derived(item.library_kind === 'clips');
	const isMovie = $derived(item.kind === 'movie' && !isClip);
	const isEpisode = $derived(item.kind === 'episode');
	const colors = $derived(posterColors(item.title));
	const meta = $derived(formatMovieMeta(item));
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
	const playbackOptions = $derived(
		data.playbackOptions as
			| Promise<PlayResponse | null>
			| PlayResponse
			| null
			| undefined
	);
	const similarItems = $derived(
		data.similarItems as Promise<MediaItem[] | null> | MediaItem[] | null | undefined
	);
	const episodeNavigation = $derived(data.episodeNavigation ?? null);
	const seriesHref = $derived(item.series_id !== null ? `/series/${item.series_id}` : null);
	const seasonHref = $derived(
		item.series_id !== null && item.season !== null
			? `/series/${item.series_id}?season=${item.season}`
			: null
	);
	const episodeHeading = $derived(
		item.episode !== null ? `E${item.episode} · ${item.title}` : item.title
	);
	const episodeArtwork = $derived(isEpisode ? episodeImageURL(item) : null);
	const episodeDuration = $derived(
		Promise.resolve(fileInfo)
			.then((result) => (result.data ? formatDurationSeconds(result.data.duration_seconds) : ''))
			.catch(() => '')
	);
</script>

<svelte:head>
	<title>{item.title} · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

{#if isEpisode}
	<nav class="breadcrumbs" aria-label="Ruta">
		{#if seriesHref}
			<a href="/series">Series</a>
			<span class="breadcrumb-separator" aria-hidden="true">›</span>
			<a href={seriesHref}>{item.series_title || 'Serie'}</a>
			{#if seasonHref && item.season !== null}
				<span class="breadcrumb-separator" aria-hidden="true">›</span>
				<a href={seasonHref}>Temporada {item.season}</a>
			{/if}
			<span class="breadcrumb-separator" aria-hidden="true">›</span>
			<span aria-current="page">
				{item.episode !== null ? `Episodio ${item.episode}` : 'Episodio'}
			</span>
		{:else}
			<span>Series</span>
			{#if item.season !== null}
				<span class="breadcrumb-separator" aria-hidden="true">›</span>
				<span>Temporada {item.season}</span>
			{/if}
			<span class="breadcrumb-separator" aria-hidden="true">›</span>
			<span aria-current="page">
				{item.episode !== null ? `Episodio ${item.episode}` : 'Episodio'}
			</span>
		{/if}
	</nav>

	<section class="episode-detail">
		{#if episodeArtwork}
			<div class="episode-hero" aria-hidden="true">
				<img src={episodeArtwork} alt="" loading="lazy" />
			</div>
		{/if}

		<div class="episode-detail-body">
			<p class="kicker">{item.series_title || 'Episodio'}</p>
			<h1>{episodeHeading}</h1>

			<p class="detail-meta">
				{#if item.season !== null && item.episode !== null}
					<span>T{item.season}E{item.episode}</span>
				{/if}
				{#await episodeDuration}
					<span class="episode-duration">…</span>
				{:then duration}
					{#if duration}
						<span class="episode-duration"> · {duration}</span>
					{/if}
				{/await}
			</p>

			{#if item.season !== null && item.episode !== null}
				<p class="detail-extra">Temporada {item.season}, episodio {item.episode}</p>
			{/if}

			<p class="detail-overview">{item.overview ?? 'Sin sinopsis'}</p>

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

			{#if playbackOptions}
				{#if playbackOptions instanceof Promise}
					{#await playbackOptions then playback}
						{#if playback}<PlaybackOptions options={playback} />{/if}
					{/await}
				{:else}
					<PlaybackOptions options={playbackOptions} />
				{/if}
			{/if}

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

			<CastRow credits={item.credits} cast={item.cast} directors={item.directors} />

			<FilePanel result={fileInfo} />

			<EpisodeNav
				previous={episodeNavigation?.previous ?? null}
				next={episodeNavigation?.next ?? null}
			/>

			{#if data.isAdmin && seriesHref}
				<p class="episode-admin">
					<a class="text-link" href={`${seriesHref}?corregir=1`}>Corregir la serie</a>
				</p>
			{/if}
		</div>
	</section>
{:else if isClip}
	<a class="back" href="/videos">← Volver a Vídeos</a>

	<section class="detail clip-detail">
		<div
			class="clip-detail-art"
			style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
			aria-hidden="true"
		>
			<span class="clip-detail-play">▶</span>
			<span class="clip-detail-title">{item.title}</span>
		</div>

		<div class="detail-info">
			<p class="kicker">Vídeo</p>
			<h1>{item.title}</h1>

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

			{#if playbackOptions}
				{#if playbackOptions instanceof Promise}
					{#await playbackOptions then playback}
						{#if playback}<PlaybackOptions options={playback} />{/if}
					{/await}
				{:else}
					<PlaybackOptions options={playbackOptions} />
				{/if}
			{/if}

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

			<FilePanel result={fileInfo} />
		</div>
	</section>
{:else}
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

			{#if playbackOptions}
				{#if playbackOptions instanceof Promise}
					{#await playbackOptions then playback}
						{#if playback}<PlaybackOptions options={playback} />{/if}
					{/await}
				{:else}
					<PlaybackOptions options={playbackOptions} />
				{/if}
			{/if}

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

			<CastRow credits={item.credits} cast={item.cast} directors={item.directors} />

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

			{#if isMovie && similarItems}
				{#if Array.isArray(similarItems)}
					{@const similar = similarItems as MediaItem[]}
					{#if similar.length > 0}
						<RowScroller title="Más como este">
							{#snippet children()}
								{#each similar as similarItem (similarItem.id)}
									<li>
										<PosterCard item={similarItem} posterUrl={similarItem.poster_url} />
									</li>
								{/each}
							{/snippet}
						</RowScroller>
					{/if}
				{:else}
					{#await similarItems then similar}
						{#if similar && similar.length > 0}
							<RowScroller title="Más como este">
								{#snippet children()}
									{#each similar as similarItem (similarItem.id)}
										<li>
											<PosterCard item={similarItem} posterUrl={similarItem.poster_url} />
										</li>
									{/each}
								{/snippet}
							</RowScroller>
						{/if}
					{/await}
				{/if}
			{/if}

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
{/if}
