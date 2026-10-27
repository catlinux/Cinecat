<script lang="ts">
	import type { Collection, ItemCollection } from '$lib/api';

	let {
		itemID,
		collections,
		options,
		open = false,
		defaultName = '',
		defaultOverview = ''
	}: {
		itemID: number;
		collections: ItemCollection[];
		options: Collection[];
		open?: boolean;
		defaultName?: string;
		defaultOverview?: string;
	} = $props();

	let query = $state('');
	const currentIDs = $derived(new Set(collections.map((collection) => collection.id)));
	const available = $derived(options.filter((collection) => !currentIDs.has(collection.id)));
	const normalizedQuery = $derived(query.trim().toLocaleLowerCase('es'));
	const filtered = $derived(
		normalizedQuery
			? available.filter((collection) =>
					collection.name.toLocaleLowerCase('es').includes(normalizedQuery)
				)
			: available
	);
</script>

<details
	class="collection-manage-panel"
	id={`collection-manage-${itemID}`}
	open={open}
>
	<summary class="collection-manage-summary">Gestionar colecciones</summary>

	<div class="collection-manage-body">
		<p class="admin-note">
			Añade esta película a una colección existente, quítala de una o crea una nueva.
		</p>

		<section class="collection-manage-section">
			<h3>En sus colecciones</h3>
			{#if collections.length > 0}
				<ul class="collection-manage-current">
					{#each collections as collection (collection.id)}
						<li>
							<a href={`/collections/${collection.id}`}>{collection.name}</a>
							<details class="collection-remove">
								<summary class="collection-remove-summary">Quitar</summary>
								<form method="POST" action="?/collectionRemove">
									<input type="hidden" name="collection_id" value={collection.id} />
									<p>¿Quitar la película de «{collection.name}»?</p>
									<button class="button button-danger" type="submit">
										Quitar de la colección
									</button>
								</form>
							</details>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Esta película no está en ninguna colección.</p>
			{/if}
		</section>

		<section class="collection-manage-section">
			<h3>Añadir a…</h3>
			<form class="collection-manage-form" method="POST" action="?/collectionAdd">
				<label>
					Buscar colección
					<input
						type="search"
						bind:value={query}
						placeholder="Nombre de la colección"
						autocomplete="off"
						disabled={available.length === 0}
					/>
				</label>
				<label>
					Colección
					<select name="collection_id" required disabled={filtered.length === 0}>
						<option value="">Elige una colección</option>
						{#each filtered as collection (collection.id)}
							<option value={collection.id}>
								{collection.name} ({collection.item_count})
							</option>
						{/each}
					</select>
				</label>
				<button class="button" type="submit" disabled={filtered.length === 0}>
					Añadir a la colección
				</button>
			</form>
			{#if available.length === 0}
				<p class="admin-note">No hay otras colecciones disponibles.</p>
			{:else if normalizedQuery && filtered.length === 0}
				<p class="admin-note" role="status">Ninguna colección coincide con la búsqueda.</p>
			{/if}
		</section>

		<section class="collection-manage-section">
			<h3>Crear colección nueva con esta película</h3>
			<form class="collection-manage-form" method="POST" action="?/collectionCreate">
				<label>
					Nombre
					<input name="name" type="text" maxlength="120" value={defaultName} required />
				</label>
				<label>
					Descripción (opcional)
					<textarea name="overview" rows="3" maxlength="2000">{defaultOverview}</textarea>
				</label>
				<button class="button" type="submit">Crear y añadir</button>
			</form>
		</section>
	</div>
</details>

<style>
	.collection-manage-panel {
		margin-top: 0.7rem;
	}

	.collection-manage-summary {
		display: inline-block;
		color: var(--naranja);
		font-weight: 600;
		text-decoration: underline;
		text-decoration-color: var(--dorado);
		text-underline-offset: 0.2em;
		cursor: pointer;
		list-style: none;
	}

	.collection-manage-summary::-webkit-details-marker,
	.collection-remove-summary::-webkit-details-marker {
		display: none;
	}

	.collection-manage-summary::after {
		content: ' +';
		color: var(--dorado);
	}

	.collection-manage-panel[open] .collection-manage-summary::after {
		content: ' −';
	}

	.collection-manage-body {
		max-width: 38rem;
		margin-top: 0.8rem;
		padding: 0.9rem 1rem 1rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}

	.collection-manage-section + .collection-manage-section {
		margin-top: 1.2rem;
		padding-top: 1rem;
		border-top: 1px solid var(--borde-suave);
	}

	.collection-manage-section h3 {
		margin: 0 0 0.65rem;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.05rem;
	}

	.collection-manage-current {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.collection-manage-current li {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.5rem 1rem;
		padding: 0.35rem 0;
		border-bottom: 1px solid var(--borde-suave);
	}

	.collection-manage-current a {
		color: var(--naranja);
		font-weight: 600;
	}

	.collection-remove-summary {
		color: var(--texto-suave);
		font-size: 0.82rem;
		text-decoration: underline;
		text-underline-offset: 0.15em;
		cursor: pointer;
		list-style: none;
	}

	.collection-remove p {
		margin: 0.5rem 0 0;
		font-size: 0.85rem;
	}

	.collection-manage-form {
		display: grid;
		gap: 0.65rem;
	}

	.collection-manage-form label {
		display: grid;
		gap: 0.25rem;
		font-size: 0.82rem;
		font-weight: 600;
	}

	.collection-manage-form input,
	.collection-manage-form select,
	.collection-manage-form textarea {
		width: 100%;
		padding: 0.45rem 0.55rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--superficie);
		color: var(--tinta);
		font: inherit;
	}

	.collection-manage-form .button {
		justify-self: start;
		margin-top: 0;
	}
</style>
