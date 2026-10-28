<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '@sveltejs/kit';
	import type { BackupsResponse } from '$lib/api';
	import { formatBytes, formatDateTime } from '$lib/media';

	type BackupForm = {
		backupAction?: boolean;
		success?: boolean;
		message?: string;
	} | null;

	let {
		info = null,
		loadError = false,
		form = null
	}: {
		info?: BackupsResponse | null;
		loadError?: boolean;
		form?: BackupForm;
	} = $props();

	let submitting = $state(false);

	const backups = $derived(
		[...(info?.backups ?? [])].sort(
			(first, second) => Date.parse(second.created_at) - Date.parse(first.created_at)
		)
	);
	const latest = $derived(backups[0] ?? null);

	const submitBackup: SubmitFunction = () => {
		submitting = true;
		return async ({ update }) => {
			try {
				await update();
			} finally {
				submitting = false;
			}
		};
	};
</script>

<div class="backup-panel">
	{#if loadError || !info}
		<p class="error" role="alert">
			No se pudieron cargar las copias de seguridad. El resto de los ajustes sigue disponible.
		</p>
	{:else}
		<div class="backup-state" aria-live="polite">
			<p>
				<span>Copias automáticas</span>
				<strong>{info.enabled ? 'Activadas' : 'Desactivadas'}</strong>
			</p>
			<p>
				<span>Días de retención</span>
				<strong>{info.retention_days} días</strong>
			</p>
			<p class="admin-note">La programación se cambia en los ajustes de arriba.</p>
		</div>

		{#if latest}
			<p class="backup-latest">
				Última copia: <strong>{formatDateTime(latest.created_at)}</strong>
			</p>
		{:else}
			<p class="admin-note">No hay copias todavía.</p>
		{/if}

		{#if info.last_error}
			<p class="error" role="alert">Último error: {info.last_error}</p>
		{/if}

		{#if backups.length > 0}
			<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
			<div
				class="backup-table-wrap"
				role="region"
				tabindex="0"
				aria-label="Lista de copias de seguridad"
			>
				<table class="backup-table">
					<caption>Copias disponibles</caption>
					<thead>
						<tr>
							<th scope="col">Nombre</th>
							<th scope="col">Fecha</th>
							<th scope="col">Tamaño</th>
						</tr>
					</thead>
					<tbody>
						{#each backups as backup (backup.name)}
							<tr>
								<th scope="row" class="backup-name">{backup.name}</th>
								<td>{formatDateTime(backup.created_at)}</td>
								<td>{formatBytes(backup.size_bytes)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}

	{#if form?.message}
		<p
			class={form.success ? 'success' : 'error'}
			role={form.success ? 'status' : 'alert'}
			aria-live="polite"
		>
			{form.message}
		</p>
	{/if}

	<form
		class="backup-actions"
		method="POST"
		action="?/backup"
		aria-busy={submitting}
		use:enhance={submitBackup}
	>
		<button class="button" type="submit" disabled={submitting}>
			{submitting ? 'Haciendo copia…' : 'Hacer copia ahora'}
		</button>
	</form>
</div>

<style>
	.backup-panel {
		display: grid;
		gap: 0.8rem;
		margin-top: 1.2rem;
		min-width: 0;
	}

	.backup-state {
		display: grid;
		gap: 0.35rem;
	}

	.backup-state p {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		margin: 0;
		padding-bottom: 0.35rem;
		border-bottom: 1px dashed var(--borde-tenue);
		color: var(--texto-apagado);
		font-size: 0.9rem;
	}

	.backup-state span {
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.backup-state strong {
		color: var(--granate);
		text-align: right;
	}

	.backup-state .admin-note {
		display: block;
		padding: 0;
		border: 0;
		font-size: 0.85rem;
		letter-spacing: normal;
		text-transform: none;
	}

	.backup-latest {
		margin: 0;
		color: var(--texto-tenue);
		font-size: 0.9rem;
	}

	.backup-latest strong {
		color: var(--granate);
	}

	.backup-table-wrap {
		max-width: 100%;
		overflow-x: auto;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie-suave);
	}

	.backup-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.84rem;
	}

	.backup-table caption {
		padding: 0.6rem 0.75rem;
		border-bottom: 1px solid var(--borde-suave);
		color: var(--granate);
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-align: left;
		text-transform: uppercase;
	}

	.backup-table th,
	.backup-table td {
		padding: 0.55rem 0.75rem;
		border-bottom: 1px solid var(--borde-tenue);
		text-align: left;
		vertical-align: top;
	}

	.backup-table thead th {
		color: var(--texto-apagado);
		font-size: 0.64rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.backup-table tbody tr:last-child th,
	.backup-table tbody tr:last-child td {
		border-bottom: 0;
	}

	.backup-table tbody th {
		color: var(--texto-tenue);
		font-weight: 600;
	}

	.backup-name {
		overflow-wrap: anywhere;
	}

	.backup-actions {
		margin: 0;
	}

	.backup-actions .button {
		margin: 0;
	}
</style>
