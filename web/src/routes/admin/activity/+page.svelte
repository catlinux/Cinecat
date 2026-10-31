<script lang="ts">
	import { formatDateTime, formatDurationSeconds, kindLabel } from '$lib/media';
	import type { ActivityEntry, ManagedUser } from '$lib/api';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const entries = $derived((data.entries ?? []) as ActivityEntry[]);
	const users = $derived((data.users ?? []) as ManagedUser[]);
	const selectedUserId = $derived(data.selectedUserId ?? null);

	function secondsLabel(seconds: number): string {
		return formatDurationSeconds(seconds) || '0 s';
	}

	function progressLabel(entry: ActivityEntry): string {
		if (!Number.isFinite(entry.duration_seconds) || entry.duration_seconds <= 0) {
			return secondsLabel(entry.position_seconds);
		}
		return `${secondsLabel(entry.position_seconds)} de ${secondsLabel(entry.duration_seconds)}`;
	}

	function episodeLabel(entry: ActivityEntry): string {
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
</script>

<svelte:head>
	<title>Actividad · Administración · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="admin activity-page">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>Actividad</h1>
		<p class="admin-note">
			Últimas reproducciones de todos los usuarios, con la fecha más reciente del progreso o
			de la marca de visto.
		</p>
	</div>

	<form class="activity-filter" method="GET" action="/admin/activity">
		<label>
			Usuario
			<select name="user_id">
				<option value="" selected={selectedUserId === null}>Todos los usuarios</option>
				{#each users as user (user.id)}
					<option value={user.id} selected={selectedUserId === user.id}>
						{user.username}
					</option>
				{/each}
			</select>
		</label>
		<button class="button" type="submit">Filtrar</button>
		{#if selectedUserId !== null}
			<a class="clear-filters" href="/admin/activity">Quitar filtro</a>
		{/if}
	</form>

	{#if entries.length === 0}
		<div class="admin-card activity-empty">
			<h2>Sin actividad</h2>
			<p class="admin-note">
				{#if selectedUserId !== null}
					Este usuario todavía no tiene reproducciones ni elementos vistos.
				{:else}
					Todavía no hay reproducciones ni elementos vistos.
				{/if}
			</p>
		</div>
	{:else}
		<p class="admin-note activity-count">
			{entries.length}
			{entries.length === 1 ? 'entrada' : 'entradas'} · últimas {data.limit ?? 100}
		</p>
		<div class="activity-table-wrap">
			<table class="activity-table">
				<caption class="activity-caption">Últimas reproducciones de los usuarios</caption>
				<thead>
					<tr>
						<th scope="col">Fecha</th>
						<th scope="col">Usuario</th>
						<th scope="col">Elemento</th>
						<th scope="col">Progreso</th>
						<th scope="col">Estado</th>
					</tr>
				</thead>
				<tbody>
					{#each entries as entry (`${entry.user_id}-${entry.item_id}`)}
						<tr>
							<td class="activity-date">{formatDateTime(entry.updated_at)}</td>
							<td class="activity-user">{entry.username}</td>
							<td class="activity-item">
								<a href={`/items/${entry.item_id}`}>{entry.title}</a>
								{#if episodeLabel(entry)}
									<span class="activity-episode">{episodeLabel(entry)}</span>
								{/if}
								<span class="activity-kind">{kindLabel(entry.kind)}</span>
							</td>
							<td class="activity-progress">{progressLabel(entry)}</td>
							<td class="activity-state">
								{#if entry.watched}
									<span class="activity-watched">Visto</span>
								{:else}
									<span class="activity-unwatched">En curso</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>

<style>
	.activity-filter {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.7rem;
		margin: 1.2rem 0 1.4rem;
		padding: 0.9rem 0;
		border-top: 1px solid var(--borde-suave);
		border-bottom: 1px solid var(--borde-suave);
	}

	.activity-filter label {
		display: grid;
		flex: 1 1 15rem;
		gap: 0.25rem;
		color: var(--texto-apagado);
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.activity-filter select {
		padding: 0.62rem 0.72rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--superficie);
		color: var(--texto-suave);
		font: inherit;
		font-size: 0.9rem;
		letter-spacing: normal;
		text-transform: none;
	}

	.activity-filter .button {
		margin: 0;
	}

	.activity-count {
		margin: 0 0 0.7rem;
	}

	.activity-empty {
		margin-top: 1rem;
	}

	.activity-empty h2 {
		margin-top: 0;
	}

	.activity-table-wrap {
		overflow-x: auto;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
	}

	.activity-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.88rem;
	}

	.activity-caption {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	.activity-table th {
		padding: 0.8rem 0.9rem;
		border-bottom: 1px solid var(--borde-medio);
		background: var(--tinte);
		color: var(--granate);
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-align: left;
		text-transform: uppercase;
		white-space: nowrap;
	}

	.activity-table td {
		padding: 0.85rem 0.9rem;
		border-bottom: 1px solid var(--borde-suave);
		vertical-align: top;
	}

	.activity-table tr:last-child td {
		border-bottom: 0;
	}

	.activity-table tbody tr:hover {
		background: var(--tinte);
	}

	.activity-date {
		white-space: nowrap;
		color: var(--texto-apagado);
		font-size: 0.8rem;
	}

	.activity-user {
		font-weight: 600;
		color: var(--granate);
	}

	.activity-item a {
		color: var(--naranja);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1rem;
		font-weight: 600;
		text-decoration-color: var(--dorado);
		text-underline-offset: 0.2em;
	}

	.activity-episode,
	.activity-kind {
		display: block;
		margin-top: 0.2rem;
		color: var(--texto-apagado);
		font-size: 0.78rem;
	}

	.activity-kind {
		margin-top: 0.35rem;
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.activity-progress {
		white-space: nowrap;
		color: var(--texto-suave);
	}

	.activity-watched,
	.activity-unwatched {
		display: inline-block;
		padding: 0.2rem 0.45rem;
		border-radius: 999px;
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		white-space: nowrap;
	}

	.activity-watched {
		background: var(--dorado);
		color: var(--granate);
	}

	.activity-unwatched {
		border: 1px solid var(--borde-medio);
		color: var(--texto-apagado);
	}

	@media (max-width: 700px) {
		.activity-filter {
			align-items: stretch;
		}

		.activity-filter .button {
			width: 100%;
		}
	}
</style>
