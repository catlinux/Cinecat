<script lang="ts">
	import type { PersonMedia } from '$lib/api';
	import { crewPriority, jobLabel, personInitials } from '$lib/rows';
	import { posterColors } from '$lib/poster-color';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const person = $derived(data.person);
	const colors = $derived(person ? posterColors(person.name) : { from: '#5a1a1f', to: '#c99a2e' });
	const initials = $derived(person ? personInitials(person.name) : '');
	const biography = $derived(person?.biography.trim() ?? '');
	const longBiography = $derived(biography.length > 420);
	const biographyPreview = $derived(
		longBiography ? `${biography.slice(0, 420).replace(/\s+\S*$/, '')}…` : biography
	);

	function formatDate(value: string): string {
		const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value.trim());
		if (!match) {
			return value;
		}
		const date = new Date(
			Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
		);
		return new Intl.DateTimeFormat('es-ES', {
			day: 'numeric',
			month: 'long',
			year: 'numeric',
			timeZone: 'UTC'
		}).format(date);
	}

	function roleLabel(role: string): string {
		const value = role.trim();
		if (!value || value.toLowerCase().startsWith('como ')) {
			return value;
		}
		return crewPriority(value) < 7 ? jobLabel(value) : `como ${value}`;
	}

	function rolesLabel(roles: string[]): string {
		return roles.map(roleLabel).filter(Boolean).join(' · ');
	}

	function mediaHref(item: PersonMedia): string | null {
		if (!item.id) {
			return null;
		}
		return item.type === 'series' ? `/series/${item.id}` : `/items/${item.id}`;
	}

	const birthday = $derived(person?.birthday ? formatDate(person.birthday) : '');
	const deathday = $derived(person?.deathday ? formatDate(person.deathday) : '');
	const life = $derived(
		[
			birthday ? `Nacimiento: ${birthday}` : '',
			deathday ? `Fallecimiento: ${deathday}` : '',
			person?.place_of_birth ? person.place_of_birth : ''
		].filter(Boolean)
	);
</script>

<svelte:head>
	<title>{person?.name ?? 'Persona'} · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

{#if data.errorMessage}
	<section class="person-error">
		<p class="kicker">Persona</p>
		<h1>No se pudo cargar la ficha</h1>
		<p class="error" role="alert">{data.errorMessage}</p>
		<a class="text-link" href="/movies">← Volver a la cartelera</a>
	</section>
{:else if person}
	<section class="person-hero">
		<div
			class="person-hero-photo"
			style={`--person-a: ${colors.from}; --person-b: ${colors.to};`}
		>
			{#if person.photo_url}
				<img src={person.photo_url} alt={person.name} loading="lazy" />
			{:else}
				<span aria-hidden="true">{initials}</span>
			{/if}
		</div>

		<div class="person-hero-info">
			<p class="kicker">Persona</p>
			<h1>{person.name}</h1>
			{#if life.length > 0}
				<p class="person-life">
					{#each life as value, index (value)}
						{#if index > 0}<span aria-hidden="true"> · </span>{/if}{value}
					{/each}
				</p>
			{/if}
			{#if person.known_for}
				<p class="person-known"><strong>Conocido por:</strong> {person.known_for}</p>
			{/if}

			{#if biography}
				<p class="person-biography">{biographyPreview}</p>
				{#if longBiography}
					<details class="person-bio-more">
						<summary>Leer más</summary>
						<p>{biography}</p>
					</details>
				{/if}
			{/if}
		</div>
	</section>

	<section class="person-section">
		<div class="person-section-head">
			<h2>En tu biblioteca</h2>
			<span class="person-count">{data.inLibrary.length}</span>
		</div>
		{#if data.inLibrary.length > 0}
			<div class="person-grid">
				{#each data.inLibrary as item, index (item.type + '-' + (item.id ?? item.tmdb_id ?? index))}
					{@const itemColors = posterColors(item.title)}
					{@const href = mediaHref(item)}
					{#if href}
						<a
							class="person-title-card"
							{href}
							style={`--poster-a: ${itemColors.from}; --poster-b: ${itemColors.to};`}
						>
							{@render titlePoster(item)}
							{@render titleCaption(item)}
						</a>
					{:else}
						<article
							class="person-title-card"
							style={`--poster-a: ${itemColors.from}; --poster-b: ${itemColors.to};`}
						>
							{@render titlePoster(item)}
							{@render titleCaption(item)}
						</article>
					{/if}
				{/each}
			</div>
		{:else}
			<p class="person-empty">No aparece en ninguna película o serie de tu biblioteca.</p>
		{/if}
	</section>

	<section class="person-section person-section-not">
		<div class="person-section-head">
			<h2>No la tienes</h2>
			<span class="person-count">{data.notInLibrary.length}</span>
		</div>
		<p class="person-note">
			Estos títulos no están en tu biblioteca; se muestran como referencia.
		</p>
		{#if data.notInLibrary.length > 0}
			<div class="person-grid">
				{#each data.notInLibrary as item, index (item.type + '-' + (item.tmdb_id ?? item.title ?? index))}
					{@const itemColors = posterColors(item.title)}
					<article
						class="person-title-card person-title-card-not"
						style={`--poster-a: ${itemColors.from}; --poster-b: ${itemColors.to};`}
					>
						{@render titlePoster(item)}
						{@render titleCaption(item)}
					</article>
				{/each}
			</div>
		{:else}
			<p class="person-empty">No hay otros títulos suyos para mostrar.</p>
		{/if}
	</section>
{/if}

{#snippet titlePoster(item: PersonMedia)}
	<div class="person-title-poster">
		{#if item.poster_url}
			<img src={item.poster_url} alt={`Cartel de ${item.title}`} loading="lazy" />
		{:else}
			<span>{item.title}</span>
		{/if}
	</div>
{/snippet}

{#snippet titleCaption(item: PersonMedia)}
	<div class="person-title-caption">
		<h3>{item.title}</h3>
		{#if item.year !== null}
			<p>{item.year}</p>
		{/if}
		{#if item.roles.length > 0}
			<p class="person-role" title={item.roles.join(' · ')}>{rolesLabel(item.roles)}</p>
		{/if}
	</div>
{/snippet}

<style>
	.person-hero {
		display: grid;
		grid-template-columns: minmax(10rem, 13rem) minmax(0, 1fr);
		gap: clamp(1.2rem, 4vw, 2.5rem);
		align-items: start;
		margin: 1rem 0 2.5rem;
	}

	.person-hero-photo {
		display: grid;
		place-items: center;
		aspect-ratio: 2 / 3;
		overflow: hidden;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--person-a);
		color: var(--crema-clara);
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(2.6rem, 8vw, 4.5rem);
		font-weight: 700;
	}

	.person-hero-photo img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.person-hero-info h1 {
		margin: 0.25rem 0 0.7rem;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(2rem, 5vw, 3.2rem);
		line-height: 1;
	}

	.person-life,
	.person-known {
		margin: 0.35rem 0 0;
		color: var(--texto-suave);
		font-size: 0.9rem;
	}

	.person-biography {
		max-width: 54rem;
		margin: 1rem 0 0;
		color: var(--texto-tenue);
		line-height: 1.6;
	}

	.person-bio-more {
		margin-top: 0.5rem;
	}

	.person-bio-more summary {
		color: var(--naranja);
		font-size: 0.75rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		cursor: pointer;
	}

	.person-bio-more p {
		max-width: 54rem;
		color: var(--texto-tenue);
		line-height: 1.6;
	}

	.person-section {
		margin-top: 2rem;
		padding-top: 1.2rem;
		border-top: 4px double var(--granate);
	}

	.person-section-head {
		display: flex;
		align-items: baseline;
		gap: 0.75rem;
	}

	.person-section-head h2 {
		margin: 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.4rem, 3vw, 2rem);
	}

	.person-count {
		color: var(--texto-apagado);
		font-size: 0.8rem;
	}

	.person-note,
	.person-empty {
		margin: 0.7rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.9rem;
	}

	.person-grid {
		display: grid;
		grid-template-columns: repeat(var(--grid-cols, 6), minmax(0, 1fr));
		gap: 1.5rem 1rem;
		margin-top: 1rem;
	}

	.person-title-card {
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		min-width: 0;
		color: inherit;
		text-decoration: none;
	}

	.person-title-card:focus-visible {
		outline: 2px solid var(--naranja);
		outline-offset: 3px;
	}

	.person-title-poster {
		display: grid;
		place-items: center;
		aspect-ratio: 2 / 3;
		overflow: hidden;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--tinte);
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		text-align: center;
	}

	.person-title-poster img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.person-title-poster span {
		padding: 0.7rem;
		font-size: 0.9rem;
		line-height: 1.15;
	}

	.person-title-caption h3 {
		margin: 0;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 0.95rem;
		line-height: 1.15;
	}

	.person-title-caption p {
		margin: 0.2rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.75rem;
		line-height: 1.3;
	}

	.person-role {
		color: var(--texto-tenue) !important;
	}

	.person-title-card-not {
		opacity: 0.62;
	}

	.person-section-not {
		margin-top: 2.5rem;
	}

	.person-error {
		margin-top: 1rem;
	}

	.person-error h1 {
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
	}

	@media (max-width: 800px) {
		.person-grid {
			--grid-cols: 4;
		}
	}

	@media (max-width: 560px) {
		.person-hero {
			grid-template-columns: minmax(7rem, 9rem) minmax(0, 1fr);
		}

		.person-grid {
			--grid-cols: 2;
		}
	}
</style>
