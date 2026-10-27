<script lang="ts">
	import { enhance } from '$app/forms';
	import ConfirmActionDialog from '$lib/components/ConfirmActionDialog.svelte';
	import { formatDateTime } from '$lib/media';
	import { trashExpiryLabel, trashKindLabel } from '$lib/trash';
	import type { TrashItem } from '$lib/api';
	import type { PageData } from './$types';

	interface TrashForm {
		success?: boolean;
		message?: string;
	}

	let { data, form = null }: { data: PageData; form?: TrashForm | null } = $props();

	const items = $derived((data.items ?? []) as TrashItem[]);

	function deletedBy(item: TrashItem): string {
		return item.deleted_by || 'Sistema';
	}
</script>

<svelte:head>
	<title>Papelera · Administración · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="admin trash-page">
	<div class="admin-head">
		<p class="kicker">Administración</p>
		<h1>Papelera</h1>
		<p class="admin-note">
			Los archivos se conservan aquí antes de eliminarse definitivamente del disco.
		</p>
	</div>

	{#if data.moved}
		<p class="success" role="status">Elemento movido a la papelera.</p>
	{/if}
	{#if form?.message}
		<p class={form.success ? 'success' : 'error'} role={form.success ? 'status' : 'alert'}>
			{form.message}
		</p>
	{/if}

	{#if items.length === 0}
		<p class="admin-note trash-empty">La papelera está vacía.</p>
	{:else}
		<div class="trash-toolbar">
			<p class="admin-note">
				{items.length} {items.length === 1 ? 'elemento' : 'elementos'} en la papelera
			</p>
			<form method="POST" action="?/purge" use:enhance>
				<button class="button button-secondary" type="submit">Vaciar lo caducado</button>
			</form>
		</div>

		<ul class="admin-list trash-list">
			{#each items as item (item.id)}
				<li class="admin-item trash-item">
					<div class="trash-copy">
						<p class="admin-item-name">
							{item.title}{#if item.year !== null} ({item.year}){/if}
						</p>
						<p class="trash-meta">
							{trashKindLabel(item)} · Eliminado el {formatDateTime(item.deleted_at)} por
							{deletedBy(item)}
						</p>
						<p class="admin-item-path">{item.original_path}</p>
						<p class="trash-expiry">{trashExpiryLabel(item.expires_at)}</p>
					</div>
					<div class="trash-actions">
						<form method="POST" action="?/restore" use:enhance>
							<input type="hidden" name="id" value={item.id} />
							<button class="button button-secondary" type="submit">Restaurar</button>
						</form>
						<ConfirmActionDialog
							id={`trash-delete-${item.id}`}
							triggerLabel="Eliminar definitivamente"
							title="Eliminar de la papelera"
							description="Se borrará del disco para siempre."
							confirmLabel="Eliminar definitivamente"
							action="?/delete"
							fields={{ id: item.id }}
							triggerClass="button-danger"
							danger
						/>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>
