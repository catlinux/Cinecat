<script lang="ts">
	import { enhance } from '$app/forms';
	import type { CollectionCandidate, CollectionDetail, MediaItem } from '$lib/api';
	import {
		collectionIdentificationLabel,
		tmdbCollectionURL
	} from '$lib/collections';
	import { truncateSynopsis } from '$lib/media';

	let {
		collection,
		searchQuery,
		searchResults,
		searchError,
		candidates = [],
		matchQuery = '',
		matchSearched = false,
		open
	}: {
		collection: CollectionDetail;
		searchQuery: string;
		searchResults: MediaItem[];
		searchError: string | null;
		candidates?: CollectionCandidate[];
		matchQuery?: string;
		matchSearched?: boolean;
		open: boolean;
	} = $props();

	const collectionIDs = $derived(new Set(collection.items.map((item) => item.id)));
	const posterChoices = $derived(collection.items.filter((item) => item.poster_url));
	const backdropChoices = $derived(collection.items.filter((item) => item.backdrop_url));
	const identification = $derived(collectionIdentificationLabel(collection));
	const tmdbURL = $derived(tmdbCollectionURL(collection.tmdb_collection_id));
</script>

<details class="admin-card collection-admin" id="gestionar" open={open}>
	<summary class="collection-admin-summary">Gestionar colección</summary>

	<div class="collection-admin-body">
		<p class="admin-note">
			{#if collection.source === 'tmdb'}
				Colección automática: se creó a partir de la saga de TMDB; tus cambios se respetan.
			{:else}
				Colección manual.
			{/if}
		</p>

		<section class="collection-admin-section">
			<h2>Identificación en TMDB</h2>
			<p class="collection-tmdb-status">
				<span>{identification}.</span>
				{#if tmdbURL}
					<a class="text-link" href={tmdbURL} target="_blank" rel="noopener noreferrer">
						Ver ficha en TMDB
					</a>
				{/if}
			</p>

			<form
				class="match-search collection-tmdb-search"
				method="POST"
				action="?/candidates"
				use:enhance
			>
				<label>
					Buscar colección en TMDB
					<input
						name="q"
						type="search"
						value={matchQuery}
						placeholder={collection.name}
						autocomplete="off"
					/>
				</label>
				<button class="button" type="submit">Buscar</button>
			</form>

			{#if matchSearched && candidates.length === 0}
				<p class="admin-note" role="status">
					Sin candidatos: prueba con otro nombre o deja el campo vacío para buscar por el nombre
					de la colección.
				</p>
			{/if}

			{#if candidates.length > 0}
				<ul class="match-candidates" aria-label="Colecciones de TMDB">
					{#each candidates as candidate (candidate.tmdb_id)}
						<li class="match-candidate">
							{#if candidate.poster_url}
								<img
									class="match-poster"
									src={candidate.poster_url}
									alt={`Cartel de ${candidate.name}`}
									loading="lazy"
								/>
							{:else}
								<span class="match-poster match-poster--empty" aria-hidden="true">
									{candidate.name.slice(0, 1)}
								</span>
							{/if}
							<div class="match-candidate-info">
								<p class="match-title">{candidate.name}</p>
								<p class="match-meta">TMDB · {candidate.tmdb_id}</p>
								{#if candidate.overview}
									<p class="match-overview">{truncateSynopsis(candidate.overview)}</p>
								{/if}
							</div>
							<form method="POST" action="?/match" use:enhance>
								<input
									type="hidden"
									name="tmdb_collection_id"
									value={candidate.tmdb_id}
								/>
								<button
									class="button"
									type="submit"
									aria-label={`Usar la colección ${candidate.name}`}
								>
									Usar esta
								</button>
							</form>
						</li>
					{/each}
				</ul>
			{/if}

			{#if collection.tmdb_collection_id}
				{#if collection.source === 'manual'}
					<details class="collection-delete collection-unlink">
						<summary>Desvincular de TMDB</summary>
						<p>
							La colección seguirá existiendo, pero dejará de estar identificada con TMDB.
						</p>
						<form method="POST" action="?/unmatch" use:enhance>
							<button class="button-danger" type="submit">Sí, desvincular</button>
						</form>
					</details>
				{:else}
					<p class="admin-note">
						Las colecciones automáticas de TMDB no se pueden desvincular.
					</p>
				{/if}
			{/if}
		</section>

		<section class="collection-admin-section">
			<h2>Datos</h2>
			<form class="collection-edit-form" method="POST" action="?/update" use:enhance>
				<label>
					Nombre
					<input name="name" type="text" value={collection.name} maxlength="120" required />
				</label>
				<label>
					Descripción
					<textarea name="overview" rows="3" maxlength="2000">{collection.overview ?? ''}</textarea>
				</label>
				<button class="button" type="submit">Guardar datos</button>
			</form>
		</section>

		<section class="collection-admin-section">
			<h2>Cartel</h2>
			<form class="collection-url-form" method="POST" action="?/poster" use:enhance>
				<input type="hidden" name="kind" value="poster" />
				<label>
					URL del cartel (vacía para quitarlo)
					<input name="url" type="url" value={collection.poster_url ?? ''} />
				</label>
				<button class="button" type="submit">Guardar cartel</button>
			</form>
			{#if posterChoices.length > 0}
				<ul class="collection-choice-grid" aria-label="Carteles de las películas">
					{#each posterChoices as item, index (item.id)}
						<li>
							<form method="POST" action="?/poster" use:enhance>
								<input type="hidden" name="kind" value="poster" />
								<input type="hidden" name="url" value={item.poster_url} />
								<button
									class="collection-choice"
									type="submit"
									aria-label={`Usar el cartel de ${item.title}`}
								>
									<img src={item.poster_url ?? ''} alt={`Cartel de ${item.title}`} loading="lazy" />
									<span>Usar este cartel</span>
								</button>
							</form>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Ninguna película de la colección tiene cartel.</p>
			{/if}
		</section>

		<section class="collection-admin-section">
			<h2>Fondo</h2>
			<form class="collection-url-form" method="POST" action="?/poster" use:enhance>
				<input type="hidden" name="kind" value="backdrop" />
				<label>
					URL del fondo (vacía para quitarlo)
					<input name="url" type="url" value={collection.backdrop_url ?? ''} />
				</label>
				<button class="button" type="submit">Guardar fondo</button>
			</form>
			{#if backdropChoices.length > 0}
				<ul class="collection-choice-grid collection-choice-grid--backdrop" aria-label="Fondos de las películas">
					{#each backdropChoices as item, index (item.id)}
						<li>
							<form method="POST" action="?/poster" use:enhance>
								<input type="hidden" name="kind" value="backdrop" />
								<input type="hidden" name="url" value={item.backdrop_url} />
								<button
									class="collection-choice"
									type="submit"
									aria-label={`Usar el fondo de ${item.title}`}
								>
									<img src={item.backdrop_url ?? ''} alt={`Fondo de ${item.title}`} loading="lazy" />
									<span>Usar este fondo</span>
								</button>
							</form>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Ninguna película de la colección tiene fondo.</p>
			{/if}
		</section>

		<section class="collection-admin-section">
			<h2>Añadir película</h2>
			<form class="collection-search" method="GET" action={`/collections/${collection.id}`}>
				<input type="hidden" name="gestionar" value="1" />
				<label>
					Buscar película
					<input name="buscar" type="search" value={searchQuery} />
				</label>
				<button class="button" type="submit">Buscar</button>
			</form>
			{#if searchError}
				<p class="error" role="alert">{searchError}</p>
			{/if}
			{#if searchResults.length > 0}
				<ul class="collection-search-results">
					{#each searchResults as item, index (item.id)}
						<li>
							{#if item.poster_url}
								<img src={item.poster_url} alt={`Cartel de ${item.title}`} loading="lazy" />
							{:else}
								<span class="collection-search-empty" aria-hidden="true">🎞</span>
							{/if}
							<div>
								<p>{item.title}</p>
								<p>{item.year ?? ''}</p>
							</div>
							{#if collectionIDs.has(item.id)}
								<span class="file-tag">Ya está</span>
							{:else}
								<form method="POST" action="?/addItem" use:enhance>
									<input type="hidden" name="media_item_id" value={item.id} />
									<button class="button" type="submit">Añadir</button>
								</form>
							{/if}
						</li>
					{/each}
				</ul>
			{:else if searchQuery && !searchError}
				<p class="admin-note">Sin resultados para esa búsqueda.</p>
			{/if}
		</section>

		<section class="collection-admin-section">
			<h2>Películas y orden</h2>
			{#if collection.items.length > 0}
				<ol class="collection-manage-list">
					{#each collection.items as item, index (item.id)}
						<li>
							<span class="collection-manage-index">{index + 1}</span>
							{#if item.poster_url}
								<img src={item.poster_url} alt={`Cartel de ${item.title}`} loading="lazy" />
							{:else}
								<span class="collection-search-empty" aria-hidden="true">🎞</span>
							{/if}
							<span class="collection-manage-title">{item.title}</span>
							<form method="POST" action="?/move" use:enhance>
								<input type="hidden" name="media_item_id" value={item.id} />
								<input type="hidden" name="direction" value="up" />
								<button class="button button-secondary" type="submit" disabled={index === 0}>
									Subir
								</button>
							</form>
							<form method="POST" action="?/move" use:enhance>
								<input type="hidden" name="media_item_id" value={item.id} />
								<input type="hidden" name="direction" value="down" />
								<button
									class="button button-secondary"
									type="submit"
									disabled={index === collection.items.length - 1}
								>
									Bajar
								</button>
							</form>
							<form method="POST" action="?/removeItem" use:enhance>
								<input type="hidden" name="media_item_id" value={item.id} />
								<button class="button-danger" type="submit">Quitar</button>
							</form>
						</li>
					{/each}
				</ol>
			{:else}
				<p class="admin-note">La colección todavía no tiene películas.</p>
			{/if}
		</section>

		<details class="collection-delete">
			<summary>Borrar colección</summary>
			<form method="POST" action="?/delete" use:enhance>
				<button class="button-danger" type="submit">Sí, borrar esta colección</button>
			</form>
		</details>
	</div>
</details>

<style>
	.collection-tmdb-status {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 0.35rem 0.8rem;
		margin: 0;
		color: var(--texto-suave);
		font-size: 0.88rem;
	}

	.collection-tmdb-search {
		grid-template-columns: minmax(0, 1fr) auto;
	}

	.collection-unlink {
		margin-top: 1rem;
	}

	.collection-unlink p {
		margin: 0.65rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.82rem;
		line-height: 1.5;
	}

	@media (max-width: 700px) {
		.collection-tmdb-search {
			grid-template-columns: 1fr;
		}
	}
</style>
