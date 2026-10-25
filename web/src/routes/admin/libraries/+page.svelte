<script lang="ts">
	import type { Library, ScanResult } from '$lib/api';

	type FormState = {
		message?: string;
		success?: boolean;
		name?: string;
		path?: string;
		scan?: ScanResult;
	} | null;

	let {
		data,
		form = null
	}: {
		data: { libraries: Library[] };
		form?: FormState;
	} = $props();

	function confirmDelete(event: SubmitEvent, name: string) {
		if (!confirm(`¿Borrar la biblioteca «${name}»? Esta acción no se puede deshacer.`)) {
			event.preventDefault();
		}
	}
</script>

<svelte:head>
	<title>Bibliotecas · Cinecat</title>
</svelte:head>

<a class="back" href="/">← Volver a la cartelera</a>

<section class="admin">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>Bibliotecas</h1>
		<p class="library-count">
			{data.libraries.length}
			{data.libraries.length === 1 ? 'biblioteca dada de alta' : 'bibliotecas dadas de alta'}
		</p>
	</div>

	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	{#if form?.scan}
		<div class="scan-result" role="status">
			<p>
				<strong>{form.scan.libraries}</strong> bibliotecas · <strong>{form.scan.files}</strong>
				archivos encontrados · <strong>{form.scan.upserted}</strong> entradas actualizadas
			</p>
			{#if form.scan.warnings.length > 0}
				<ul>
					{#each form.scan.warnings as warning}
						<li>{warning}</li>
					{/each}
				</ul>
			{:else}
				<p class="admin-note">Sin avisos.</p>
			{/if}
		</div>
	{/if}

	<div class="admin-grid">
		<div class="admin-card">
			<h2>Bibliotecas</h2>

			{#if data.libraries.length === 0}
				<p class="admin-note">
					Todavía no hay ninguna biblioteca. Crea una con el formulario de al lado.
				</p>
			{:else}
				<ul class="admin-list">
					{#each data.libraries as library (library.id)}
						<li class="admin-item">
							<div>
								<p class="admin-item-name">{library.name}</p>
								<p class="admin-item-path">{library.path}</p>
							</div>
							<form
								method="POST"
								action="?/delete"
								onsubmit={(event) => confirmDelete(event, library.name)}
							>
								<input type="hidden" name="id" value={library.id} />
								<button type="submit" class="button-danger">Borrar</button>
							</form>
						</li>
					{/each}
				</ul>
			{/if}

			<form method="POST" action="?/scan" class="admin-toolbar">
				<button type="submit" class="button">Escanear todas las bibliotecas</button>
			</form>
		</div>

		<div class="admin-card">
			<h2>Nueva biblioteca</h2>
			<form method="POST" action="?/create" class="login-form">
				<label>
					<span>Nombre</span>
					<input type="text" name="name" value={form?.name ?? ''} required />
				</label>

				<label>
					<span>Ruta en el servidor</span>
					<input
						type="text"
						name="path"
						placeholder="/media/peliculas"
						value={form?.path ?? ''}
						required
					/>
				</label>

				<button type="submit">Crear biblioteca</button>
			</form>
			<p class="admin-note">La ruta debe existir y ser un directorio en el servidor.</p>
		</div>
	</div>
</section>
