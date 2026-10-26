<script lang="ts">
	import { enhance } from '$app/forms';
	import { onMount, untrack } from 'svelte';
	import type { Library, ScanStatus } from '$lib/api';
	import ScanProgress from '$lib/components/ScanProgress.svelte';

	type FormState = {
		message?: string;
		success?: boolean;
		name?: string;
		path?: string;
		started?: boolean;
	} | null;

	let {
		data,
		form = null
	}: {
		data: { libraries: Library[] };
		form?: FormState;
	} = $props();

	const POLL_INTERVAL = 3000;

	let status = $state<ScanStatus | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let requesting = false;

	const currentStatus = $derived(status ?? (form?.started ? runningStatus() : null));

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

	function confirmDelete(event: SubmitEvent, name: string) {
		if (!confirm(`¿Borrar la biblioteca «${name}»? Esta acción no se puede deshacer.`)) {
			event.preventDefault();
		}
	}
</script>

<svelte:head>
	<title>Bibliotecas · Cinecat</title>
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

	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	{#if currentStatus}
		<ScanProgress status={currentStatus} />
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

			<form method="POST" action="?/scan" class="admin-toolbar" use:enhance>
				<button type="submit" class="button" disabled={currentStatus?.running ?? false}>
					Escanear todas las bibliotecas
				</button>
			</form>
			<p class="admin-note">
				El escaneo continúa en segundo plano; puedes cerrar esta página mientras termina.
			</p>
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

<style>
	.admin-toolbar .button:disabled {
		opacity: 0.55;
		cursor: wait;
	}
</style>
