<script lang="ts">
	import { enhance } from '$app/forms';
	import { kindLabel } from '$lib/media';
	import type { HistoryEntry } from '$lib/api';
	import type { PageData } from './$types';

	interface HistoryForm {
		success?: boolean;
		message?: string;
	}

	let { data, form = null }: { data: PageData; form?: HistoryForm | null } = $props();

	const entries = $derived((data.entries ?? []) as HistoryEntry[]);

	function relativeTime(value: string | null): string {
		if (!value) {
			return 'sin fecha';
		}
		const timestamp = Date.parse(value);
		if (Number.isNaN(timestamp)) {
			return value;
		}
		const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
		if (seconds < 60) {
			return `hace ${seconds} s`;
		}
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) {
			return `hace ${minutes} min`;
		}
		const hours = Math.floor(minutes / 60);
		if (hours < 24) {
			return `hace ${hours} h`;
		}
		return `hace ${Math.floor(hours / 24)} días`;
	}

	function episodeLabel(entry: HistoryEntry): string {
		if (entry.kind !== 'episode') {
			return '';
		}
		const code =
			entry.season !== null && entry.episode !== null
				? `T${entry.season} E${entry.episode}`
				: entry.episode !== null
					? `Episodio ${entry.episode}`
					: '';
		return [entry.series_title, code].filter(Boolean).join(' · ');
	}

	function progressPercent(entry: HistoryEntry): number {
		if (!Number.isFinite(entry.duration_seconds) || entry.duration_seconds <= 0) {
			return 0;
		}
		return Math.max(
			0,
			Math.min(100, Math.round((entry.position_seconds / entry.duration_seconds) * 100))
		);
	}
</script>

<svelte:head>
	<title>Historial · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="history-page">
	<div class="history-head">
		<p class="kicker">Tu sesión</p>
		<h1>Historial</h1>
		<p class="history-lede">
			Lo último que has reproducido o marcado como visto, con la fecha más reciente.
		</p>
	</div>

	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	{#if entries.length === 0}
		<div class="admin-card history-empty" role="status">
			<h2>Todavía no has visto nada</h2>
			<p class="admin-note">Cuando reproduzcas algo, aparecerá aquí.</p>
		</div>
	{:else}
		<ul class="history-list">
			{#each entries as entry (`${entry.item_id}`)}
				<li class="history-item">
					<a
						class="history-cover"
						href={`/items/${entry.item_id}`}
						aria-hidden="true"
						tabindex="-1"
					>
						{#if entry.poster_url}
							<img src={entry.poster_url} alt="" loading="lazy" />
						{:else}
							<span>{entry.title.slice(0, 1)}</span>
						{/if}
					</a>
					<div class="history-copy">
						<p class="history-kind">{kindLabel(entry.kind)}</p>
						<a class="history-title" href={`/items/${entry.item_id}`}>{entry.title}</a>
						{#if episodeLabel(entry)}
							<p class="history-episode">{episodeLabel(entry)}</p>
						{/if}
						<p class="history-date" title={entry.updated_at}>{relativeTime(entry.updated_at)}</p>
					</div>
					<div class="history-state">
						{#if entry.watched}
							<span class="history-watched">Visto</span>
						{:else}
							<div
								class="history-progress"
								role="progressbar"
								aria-valuemin="0"
								aria-valuemax="100"
								aria-valuenow={progressPercent(entry)}
								aria-label={`Progreso ${progressPercent(entry)} %`}
							>
								<span style={`width: ${progressPercent(entry)}%`}></span>
							</div>
						{/if}
						<form method="POST" action="?/delete" use:enhance>
							<input type="hidden" name="item_id" value={entry.item_id} />
							<button class="button button-secondary history-remove" type="submit">
								Quitar del historial
							</button>
						</form>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>

<style>
	.history-page {
		max-width: 62rem;
		margin: 0 auto;
		padding: clamp(1.2rem, 4vw, 3rem) clamp(1rem, 4vw, 3rem) 4rem;
	}
	.history-head {
		margin-bottom: 1.4rem;
	}
	.history-head h1 {
		margin: 0.2rem 0 0.5rem;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(2rem, 5vw, 3.4rem);
		font-weight: 600;
		line-height: 1;
	}
	.history-lede {
		max-width: 42rem;
		margin: 0;
		color: var(--texto-apagado);
		line-height: 1.5;
	}
	.history-empty {
		margin-top: 1.2rem;
	}
	.history-empty h2 {
		margin-top: 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
	}
	.history-list {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		margin: 1.2rem 0 0;
		padding: 0;
		list-style: none;
	}
	.history-item {
		display: grid;
		grid-template-columns: 3.4rem minmax(0, 1fr) auto;
		gap: 0.9rem;
		align-items: center;
		padding: 0.65rem;
		border: 1px solid var(--borde-suave);
		border-radius: 5px;
		background: var(--superficie);
	}
	.history-cover {
		display: block;
		width: 3.4rem;
		aspect-ratio: 2 / 3;
		overflow: hidden;
		border-radius: 4px;
		background: linear-gradient(155deg, var(--granate), var(--naranja));
		color: var(--texto-sobre-oscuro);
		text-decoration: none;
	}
	.history-cover img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.history-cover span {
		display: flex;
		width: 100%;
		height: 100%;
		align-items: center;
		justify-content: center;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.3rem;
		font-weight: 700;
	}
	.history-copy {
		min-width: 0;
	}
	.history-kind {
		margin: 0 0 0.15rem;
		color: var(--texto-apagado);
		font-size: 0.62rem;
		font-weight: 700;
		letter-spacing: 0.09em;
		text-transform: uppercase;
	}
	.history-title {
		display: block;
		overflow: hidden;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.06rem;
		font-weight: 600;
		line-height: 1.2;
		text-decoration: none;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.history-episode,
	.history-date {
		margin: 0.2rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.78rem;
	}
	.history-date {
		font-size: 0.72rem;
	}
	.history-state {
		display: flex;
		min-width: 9rem;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.45rem;
	}
	.history-progress {
		width: 100%;
		height: 0.35rem;
		overflow: hidden;
		border-radius: 999px;
		background: var(--tinte);
	}
	.history-progress span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: var(--naranja);
	}
	.history-watched {
		padding: 0.18rem 0.45rem;
		border-radius: 999px;
		background: var(--tinte);
		color: var(--granate);
		font-size: 0.66rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}
	.history-remove {
		padding: 0.35rem 0.55rem;
		font-size: 0.7rem;
	}
	@media (max-width: 640px) {
		.history-item {
			grid-template-columns: 3.2rem minmax(0, 1fr);
		}
		.history-state {
			grid-column: 1 / -1;
			min-width: 0;
			align-items: stretch;
		}
	}
</style>
