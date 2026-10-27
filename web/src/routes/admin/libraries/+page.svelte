<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount, untrack } from 'svelte';
	import type { Library, ScanStatus } from '$lib/api';
	import { LIBRARY_KIND_OPTIONS, type LibraryFormState } from '$lib/libraries';
	import FolderBrowser from '$lib/components/FolderBrowser.svelte';
	import LibraryCard from '$lib/components/LibraryCard.svelte';
	import ScanProgress from '$lib/components/ScanProgress.svelte';

	let {
		data,
		form = null
	}: {
		data: { libraries: Library[]; ui?: { server_name: string } };
		form?: LibraryFormState | null;
	} = $props();

	const POLL_INTERVAL = 3000;

	let status = $state<ScanStatus | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let requesting = false;
	let submitting = $state(false);

	let name = $state(
		untrack(() => (form?.action === 'create' ? (form.name ?? '') : ''))
	);
	let kind = $state(
		untrack(() => (form?.action === 'create' && form.kind ? form.kind : 'video'))
	);
	let paths = $state<string[]>(
		untrack(() =>
			form?.action === 'create' && form.paths && form.paths.length > 0
				? [...form.paths]
				: ['']
		)
	);
	let browserOpen = $state(false);
	let browserIndex = $state(0);

	const currentStatus = $derived(status ?? (form?.started ? runningStatus() : null));
	const createMessage = $derived(form?.action === 'create' ? form : null);
	const messageInCard = $derived(
		form?.libraryId !== undefined &&
			data.libraries.some((library) => library.id === form?.libraryId)
	);
	const otherMessage = $derived(
		form && form.action !== 'create' && !messageInCard ? form : null
	);

	$effect(() => {
		if (!form || form.action !== 'create') {
			return;
		}
		if (form.success) {
			name = '';
			kind = 'video';
			paths = [''];
			return;
		}
		name = form.name ?? '';
		kind = form.kind ?? 'video';
		paths = form.paths && form.paths.length > 0 ? [...form.paths] : [''];
	});

	function runningStatus(): ScanStatus {
		return {
			running: true,
			started_at: new Date().toISOString(),
			finished_at: null,
			libraries: 0,
			files: 0,
			upserted: 0,
			identified: 0,
			unidentified: 0,
			warnings: [],
			error: ''
		};
	}

	async function fetchStatus() {
		if (requesting) {
			return;
		}
		requesting = true;
		try {
			const response = await fetch('/admin/libraries/status', {
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!response.ok) {
				return;
			}
			const next = (await response.json()) as ScanStatus;
			status = next;
			if (!next.running) {
				stopPolling();
			}
		} catch {
			// Se reintenta en el siguiente ciclo; un fallo no rompe la página.
		} finally {
			requesting = false;
		}
	}

	function startPolling() {
		if (pollTimer !== null) {
			return;
		}
		pollTimer = setInterval(() => {
			void fetchStatus();
		}, POLL_INTERVAL);
	}

	function stopPolling() {
		if (pollTimer !== null) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	function markRunning() {
		if (!status?.running) {
			status = runningStatus();
			void fetchStatus();
		}
		startPolling();
	}

	onMount(() => {
		void fetchStatus().then(() => {
			if (status?.running) {
				startPolling();
			}
		});
		return stopPolling;
	});

	$effect(() => {
		if (form?.started) {
			untrack(() => markRunning());
		}
	});

	function addPathRow() {
		paths = [...paths, ''];
	}

	function removePathRow(index: number) {
		if (paths.length <= 1) {
			paths = [''];
			return;
		}
		paths = paths.filter((_, current) => current !== index);
	}

	function openBrowser(index: number) {
		browserIndex = index;
		browserOpen = true;
	}

	function chooseNewPath(path: string) {
		paths = paths.map((current, index) => (index === browserIndex ? path : current));
		browserOpen = false;
	}

	function enhanceSubmit() {
		submitting = true;
		return async ({ update }: { update: () => Promise<void> }) => {
			try {
				await update();
			} finally {
				submitting = false;
			}
		};
	}
</script>

<svelte:head>
	<title>Bibliotecas · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="admin">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>Bibliotecas</h1>
		<p class="library-count">
			{data.libraries.length}
			{data.libraries.length === 1 ? 'biblioteca dada de alta' : 'bibliotecas dadas de alta'}
		</p>
	</div>

	{#if otherMessage?.message}
		<p
			class={otherMessage.success ? 'success' : 'error'}
			role={otherMessage.success ? 'status' : 'alert'}
		>
			{otherMessage.message}
		</p>
	{/if}

	{#if currentStatus}
		<ScanProgress status={currentStatus} />
	{/if}

	{#if data.libraries.length === 0}
		<div class="admin-card empty-libraries">
			<h2>Bibliotecas</h2>
			<p class="admin-note">
				Todavía no hay ninguna biblioteca. Crea una con el formulario de abajo.
			</p>
		</div>
	{:else}
		<div class="admin-grid library-grid">
			{#each data.libraries as library (library.id)}
				<LibraryCard
					{library}
					{form}
					scanRunning={currentStatus?.running ?? false}
				/>
			{/each}
		</div>
	{/if}

	<form method="POST" action="?/scan" class="admin-toolbar" use:enhance={enhanceSubmit}>
		<button
			type="submit"
			class="button"
			disabled={(currentStatus?.running ?? false) || submitting}
		>
			Escanear todas las bibliotecas
		</button>
	</form>
	<p class="admin-note">
		El escaneo continúa en segundo plano; puedes cerrar esta página mientras termina.
	</p>

	<div class="admin-card new-library-card">
		<h2>Nueva biblioteca</h2>

		{#if createMessage?.message}
			<p class={createMessage.success ? 'success' : 'error'} role={createMessage.success ? 'status' : 'alert'}>
				{createMessage.message}
			</p>
		{/if}

		<form method="POST" action="?/create" class="login-form" use:enhance={enhanceSubmit}>
			<label>
				<span>Nombre</span>
				<input type="text" name="name" bind:value={name} required maxlength="80" />
			</label>

			<label>
				<span>Tipo</span>
				<select
					name="kind"
					onchange={(event) => (kind = (event.currentTarget as HTMLSelectElement).value)}
				>
					{#each LIBRARY_KIND_OPTIONS as option (option.value)}
						<option
							value={option.value}
							disabled={option.disabled}
							selected={kind === option.value}
						>
							{option.label}{option.disabled ? ' · Próximamente' : ''}
						</option>
					{/each}
				</select>
				<small class="admin-note">Los tipos Vídeos y Música aún no se escanean.</small>
			</label>

			<fieldset class="new-path-list">
				<legend>Carpetas</legend>
				{#each paths as path, index (index)}
					<div class="new-path-row">
						<label>
							<span>Carpeta {index + 1}</span>
							<input
								type="text"
								name="paths"
								bind:value={paths[index]}
								placeholder="/media/peliculas"
								required
							/>
						</label>
						<div class="path-row-actions">
							<button
								type="button"
								class="button button-secondary"
								onclick={() => openBrowser(index)}
							>
								Elegir…
							</button>
							{#if paths.length > 1}
								<button
									type="button"
									class="button-danger"
									onclick={() => removePathRow(index)}
								>
									Quitar
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</fieldset>

			<button type="button" class="button button-secondary" onclick={addPathRow}>
				Añadir otra carpeta
			</button>

			<button type="submit" class="button" disabled={submitting}>Crear biblioteca</button>
		</form>

		<p class="admin-note">
			Las rutas deben existir y estar dentro de las raíces permitidas del servidor. Puedes
			escribirlas a mano o elegirlas con el navegador.
		</p>
	</div>
</section>

<FolderBrowser
	open={browserOpen}
	onclose={() => (browserOpen = false)}
	onselect={chooseNewPath}
/>
