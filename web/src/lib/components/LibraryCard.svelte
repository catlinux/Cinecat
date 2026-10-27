<script lang="ts">
	import { enhance } from '$app/forms';
	import { untrack } from 'svelte';
	import type { Library } from '$lib/api';
	import { libraryKindLabel, type LibraryFormState } from '$lib/libraries';
	import FolderBrowser from './FolderBrowser.svelte';

	let {
		library,
		form = null,
		scanRunning = false
	}: {
		library: Library;
		form?: LibraryFormState | null;
		scanRunning?: boolean;
	} = $props();

	const paths = $derived(
		library.paths && library.paths.length > 0
			? library.paths
			: [{ id: 0, path: library.path }]
	);
	const kindLabel = $derived(libraryKindLabel(library.kind));
	const removeMessage =
		'Se quitarán de Cinecat las fichas de esta carpeta; los archivos del disco no se tocan';
	const itemCount = $derived(
		typeof library.item_count === 'number'
			? `${library.item_count} ${library.item_count === 1 ? 'ficha' : 'fichas'}`
			: null
	);

	let addPath = $state(
		untrack(() =>
			form?.action === 'addPath' && form.libraryId === library.id ? (form.path ?? '') : ''
		)
	);
	let renameName = $state(
		untrack(() =>
			form?.action === 'rename' && form.libraryId === library.id
				? (form.renameName ?? library.name)
				: library.name
		)
	);
	let browserOpen = $state(false);
	let pending = $state(false);

	$effect(() => {
		if (form?.action === 'addPath' && form.libraryId === library.id) {
			addPath = form.path ?? '';
		}
		if (form?.action === 'rename' && form.libraryId === library.id) {
			renameName = form.renameName ?? library.name;
		} else {
			renameName = library.name;
		}
	});

	function enhanceSubmit() {
		pending = true;
		return async ({ update }: { update: () => Promise<void> }) => {
			try {
				await update();
			} finally {
				pending = false;
			}
		};
	}

	function confirmRemove(event: SubmitEvent) {
		if (!confirm(removeMessage)) {
			event.preventDefault();
		}
	}

	function chooseFolder(path: string) {
		addPath = path;
		browserOpen = false;
	}
</script>

<article class="admin-card library-card">
	<header class="library-card-head">
		<div>
			<h2>{library.name}</h2>
			<p class="library-kind">{kindLabel}</p>
		</div>
		{#if itemCount}
			<p class="library-count">{itemCount}</p>
		{/if}
	</header>

	{#if form?.message && form.libraryId === library.id}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	<div class="library-paths">
		<h3>Carpetas</h3>
		{#if paths.length === 0}
			<p class="admin-note">Esta biblioteca no tiene carpetas.</p>
		{:else}
			<ul class="path-list">
				{#each paths as folder (folder.id || folder.path)}
					<li class="path-row">
						<span class="path-value">{folder.path}</span>
						<form
							method="POST"
							action="?/removePath"
							use:enhance={enhanceSubmit}
							onsubmit={confirmRemove}
						>
							<input type="hidden" name="id" value={library.id} />
							<input type="hidden" name="pathId" value={folder.id} />
							<button
								type="submit"
								class="button-danger"
								disabled={paths.length <= 1 || folder.id <= 0 || pending}
								title={paths.length <= 1
									? 'Una biblioteca debe tener al menos una carpeta'
									: removeMessage}
								aria-label={`Quitar ${folder.path}`}
							>
								Quitar
							</button>
						</form>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<form method="POST" action="?/addPath" class="library-add-path" use:enhance={enhanceSubmit}>
		<input type="hidden" name="id" value={library.id} />
		<label>
			<span>Añadir carpeta</span>
			<input
				type="text"
				name="path"
				bind:value={addPath}
				placeholder="/media/otra-carpeta"
				required
			/>
		</label>
		<div class="path-row-actions">
			<button
				type="button"
				class="button button-secondary"
				onclick={() => (browserOpen = true)}
			>
				Elegir…
			</button>
			<button type="submit" class="button" disabled={pending}>Añadir carpeta</button>
		</div>
	</form>

	<div class="library-actions">
		<details class="library-action">
			<summary class="button button-secondary">Renombrar</summary>
			<form method="POST" action="?/rename" use:enhance={enhanceSubmit}>
				<input type="hidden" name="id" value={library.id} />
				<label>
					<span>Nuevo nombre</span>
					<input type="text" name="name" bind:value={renameName} required maxlength="80" />
				</label>
				<button type="submit" class="button" disabled={pending}>Guardar nombre</button>
			</form>
		</details>

		<form method="POST" action="?/scan" class="library-action" use:enhance={enhanceSubmit}>
			<input type="hidden" name="id" value={library.id} />
			<button type="submit" class="button button-secondary" disabled={scanRunning || pending}>
				Escanear
			</button>
		</form>

		<details class="library-action library-delete">
			<summary class="button-danger">Borrar biblioteca</summary>
			<form method="POST" action="?/delete" use:enhance={enhanceSubmit}>
				<input type="hidden" name="id" value={library.id} />
				<input type="hidden" name="name" value={library.name} />
				<label>
					<span>Escribe «{library.name}» para confirmar</span>
					<input
						type="text"
						name="confirm"
						autocomplete="off"
						placeholder={library.name}
						required
					/>
				</label>
				<button type="submit" class="button-danger" disabled={pending}>
					Borrar biblioteca
				</button>
			</form>
		</details>
	</div>

	<FolderBrowser
		open={browserOpen}
		onclose={() => (browserOpen = false)}
		onselect={chooseFolder}
	/>
</article>
